package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/MindHunter86/asmas/internal/auth"
	"github.com/MindHunter86/asmas/internal/system"
	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

var (
	gCli  *cli.Context
	gLog  *zerolog.Logger
	gALog *zerolog.Logger

	gCtx   context.Context
	gAbort context.CancelFunc
)

type Service struct {
	fb *fiber.App

	syslogWriter io.Writer

	pprofPrefix string
	pprofSecret []byte
}

func NewService(c *cli.Context, l *zerolog.Logger, s io.Writer) *Service {
	gCli, gLog, gALog = c, l, nil

	if zerolog.GlobalLevel() > zerolog.DebugLevel && zerolog.GlobalLevel() < zerolog.NoLevel {
		alogger := gLog.With().Logger().Output(s)
		gALog = &alogger
	}

	service := new(Service)
	service.syslogWriter = s

	appname := fmt.Sprintf("%s/%s", c.App.Name, c.App.Version)

	service.fb = fiber.New(fiber.Config{
		EnableTrustedProxyCheck: len(gCli.String("http-trusted-proxies")) > 0,
		TrustedProxies:          strings.Split(gCli.String("http-trusted-proxies"), ","),
		ProxyHeader:             gCli.String("http-realip-header"),

		AppName:               appname,
		ServerHeader:          appname,
		DisableStartupMessage: true,

		StrictRouting:      true,
		DisableDefaultDate: true,
		DisableKeepalive:   false,

		DisablePreParseMultipartForm: true,

		IdleTimeout:  gCli.Duration("http-timeout-idle"),
		ReadTimeout:  gCli.Duration("http-timeout-read"),
		WriteTimeout: gCli.Duration("http-timeout-write"),

		DisableDefaultContentType: true,

		GETOnly: true,
		RequestMethods: []string{
			fiber.MethodHead,
			fiber.MethodGet,
		},

		// JSONEncoder: easyjson.Marshal,
		// JSONDecoder: easyjson.Unmarshal,

		ErrorHandler: service.fiberDefaultErrorHandler,
	})

	return service
}

func (m *Service) Bootstrap() (e error) {
	var wg sync.WaitGroup
	var echan = make(chan error, 32)

	// goroutine helper
	gofunc := func(w *sync.WaitGroup, p func()) {
		w.Add(1)

		go func(done, payload func()) {
			payload()
			done()
		}(w.Done, p)
	}

	gCtx, gAbort = context.WithCancel(context.Background())
	gCtx = context.WithValue(gCtx, utils.CKeyLogger, gLog)
	gCtx = context.WithValue(gCtx, utils.CKeyCliCtx, gCli)
	gCtx = context.WithValue(gCtx, utils.CKeyAbortFunc, gAbort)
	gCtx = context.WithValue(gCtx, utils.CKeyErrorChan, echan)

	// defer m.checkErrorsBeforeClosing(echan)
	// defer wg.Wait() // !!
	defer gLog.Debug().Msg("waiting for opened goroutines")
	defer gAbort()

	// BOOTSTRAP SECTION:
	// ? write any subservice initialization block above the fiber server

	// System Maintain Service
	sysservice := system.NewSystem(gCtx, gCli)
	gCtx = context.WithValue(gCtx, utils.CKeySystem, sysservice)
	gofunc(&wg, sysservice.Bootstrap)

	// Authentification Authorization Service
	aservice := auth.NewAuthService(gCtx, gCli)
	gCtx = context.WithValue(gCtx, utils.CKeyAuthService, aservice)
	gofunc(&wg, aservice.Boostrap)

	// fiber (http) server configuration && launch
	// * shall be at the end of bootstrap section
	m.fiberMiddlewareInitialization()
	m.fiberRouterInitialization()

	gofunc(&wg, func() {
		gLog.Debug().Msg("starting fiber http server...")
		defer gLog.Debug().Msg("fiber http server has been stopped")

		if e = m.fb.Listen(gCli.String("http-listen-addr")); errors.Is(e, context.Canceled) {
			return
		} else if e != nil {
			gLog.Error().Err(e).Msg("fiber internal error")
			echan <- e
		}
	})

	// service event loop
	// * last step in launch process
	return m.loop(echan, &wg)
}

func (m *Service) loop(errs chan error, wg *sync.WaitGroup) (e error) {
	defer wg.Wait()

	kernSignal := make(chan os.Signal, 1)
	signal.Notify(kernSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTERM, syscall.SIGQUIT)

	gLog.Debug().Msg("initiate main event loop...")
	defer gLog.Debug().Msg("main event loop has been closed")

	gLog.Info().Msg("ready...")

LOOP:
	for {
		select {
		case <-kernSignal:
			gLog.Info().Msg("kernel signal has been caught; initiate application closing...")
			gAbort()
		case e = <-errs:
			gLog.Info().Err(e).Msg("application error has been caught; initiate application closing...")
			gLog.Trace().Msg("calling abort()...")
			gAbort()
		case <-gCtx.Done():
			gLog.Info().Msg("internal abort() has been caught; initiate application closing...")
			break LOOP
		}
	}

	// http destruct (wtf fiber?)
	// ShutdownWithContext() may be called only after fiber.Listen is running (O_o)
	if err := m.fb.ShutdownWithContext(gCtx); err != nil && !errors.Is(err, context.Canceled) {
		gLog.Error().Msgf("BUG! fiber server Shutdown() error - %s", err.Error())
	}

	return
}

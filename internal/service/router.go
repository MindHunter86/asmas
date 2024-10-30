package service

import (
	"bytes"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/rs/zerolog"
)

var loggerPool = sync.Pool{
	New: func() interface{} {
		if gALog != nil {
			l := gALog.With().Logger()
			return &l
		} else {
			l := gLog.With().Logger()
			return &l
		}
	},
}

func (m *Service) fiberMiddlewareInitialization() {
	// pprof profiler
	// manual:
	// 	curl -o profile.out https://host/debug/pprof -H 'X-Authorization: $TOKEN'
	// 	go tool pprof profile.out
	if gCli.Bool("http-pprof-enable") {
		m.pprofPrefix = gCli.String("http-pprof-prefix")
		m.pprofSecret = []byte(gCli.String("http-pprof-secret"))

		var pprofNext func(*fiber.Ctx) bool
		if len(m.pprofSecret) != 0 {
			pprofNext = func(c *fiber.Ctx) (_ bool) {
				isecret := c.Context().Request.Header.Peek("x-pprof-secret")

				if len(isecret) == 0 {
					return
				}

				return !bytes.Equal(m.pprofSecret, isecret)
			}
		}

		m.fb.Use(pprof.New(pprof.Config{
			Next:   pprofNext,
			Prefix: gCli.String("http-pprof-prefix"),
		}))
	}

	// request id 3.0
	m.fb.Use(func(c *fiber.Ctx) error {
		c.Set("X-Request-Id", strconv.FormatUint(c.Context().ID(), 10))
		return c.Next()
	})

	// application context injection
	m.fb.Use(func(c *fiber.Ctx) error {
		c.SetUserContext(gCtx)
		return c.Next()
	})

	// prefixed logger initialization
	// - we send logs in syslog and stdout by default,
	// - but if access-log-stdout is 0 we use syslog output only
	m.fb.Use(func(c *fiber.Ctx) (e error) {
		logger := loggerPool.Get().(*zerolog.Logger)

		logger.UpdateContext(func(zc zerolog.Context) zerolog.Context {
			return zc.Uint64("id", c.Context().ID())
		})

		c.Locals("logger", logger)
		e = c.Next()

		loggerPool.Put(logger)
		return
	})

	// panic recover for all handlers
	m.fb.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			rlog(c).Error().Str("request", c.Request().String()).Bytes("stack", debug.Stack()).
				Msg("panic has been caught")
			_, _ = os.Stderr.WriteString(fmt.Sprintf("panic: %v\n%s\n", e, debug.Stack())) //nolint:errcheck // This will never fail

			c.Status(fiber.StatusInternalServerError)
		},
	}))

	// !!!
	// !!!
	// !!!
	// !!!
	// !!!
	// !!!

	// time collector + logger
	// m.fb.Use(func(c *fiber.Ctx) (e error) {
	// 	started, e := time.Now(), c.Next()
	// 	elapsed := time.Since(started).Round(time.Microsecond)

	// 	status, lvl := c.Response().StatusCode(), utils.HTTPAccessLogLevel

	// 	// ? not profitable
	// 	// TODO too much allocations here:
	// 	err := AcquireFErr()
	// 	defer ReleaseFErr(err)

	// 	var cause string
	// 	if errors.As(e, &err) || status >= fiber.StatusInternalServerError {
	// 		status, lvl, cause = err.Code, zerolog.WarnLevel, err.Error()
	// 	}

	// 	rlog(c).WithLevel(lvl).
	// 		Int("status", status).
	// 		Str("method", c.Method()).
	// 		Str("path", c.Path()).
	// 		Str("ip", c.IP()).
	// 		Dur("latency", elapsed).
	// 		Str("user-agent", c.Get(fiber.HeaderUserAgent)).Msg(cause)

	// 	return
	// })
}

func (m *Service) fiberRouterInitialization() {
	//
	// ASMAS internal cache api
	m.fb.Get("/healthz", nil)

	//
	// ASMAS public v1 api
	v1 := m.fb.Group("/v1", middlewareAuthentification)

	certs := v1.Group("/certificates/:name", middlewareAuthorization)
	certs.Get("/public", handleGetCertificate)

	// v1.Get("/certificates/public/:name", auth.HandleGetCertificate)
	// v1.Get("/certificates/private/:name", )
}

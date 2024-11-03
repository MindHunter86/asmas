package client

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/MindHunter86/asmas/internal/system"
	"github.com/MindHunter86/asmas/internal/utils"
	futils "github.com/gofiber/fiber/v2/utils"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

type Client struct {
	asmas *HttpServiceClient

	cc  *cli.Context
	log *zerolog.Logger

	ctx   context.Context
	done  func() <-chan struct{}
	abort context.CancelFunc
}

func NewClient(c *cli.Context, l *zerolog.Logger) *Client {
	return &Client{
		cc:  c,
		log: l,
	}
}

func (m *Client) Bootstrap() (e error) {
	var wg sync.WaitGroup

	// goroutine helper
	gofunc := func(w *sync.WaitGroup, p func()) {
		w.Add(1)

		go func(done, payload func()) {
			payload()
			done()
		}(w.Done, p)
	}

	m.ctx, m.abort = context.WithCancel(context.Background())
	m.ctx = context.WithValue(m.ctx, utils.CKeyLogger, m.log)
	m.ctx = context.WithValue(m.ctx, utils.CKeyAbortFunc, m.abort)

	// * BOOTSTRAP SECTION *
	// > feel free to initialize the whole world <

	// ASMAS client
	m.asmas = newHttpServiceClient(m.ctx, m.cc)
	gofunc(&wg, m.asmas.bootstrap)

	// Test block
	// !! Call custrom recover() for removing all pipe files (avoid of nginx stucking)
	var file *system.PipeFile
	if file, e = system.NewPipeFile(m.ctx, "test.pipe"); e != nil || file == nil {
		return
	}
	gofunc(&wg, file.StreamFilePayload)

	// System Maintain Service
	// sysservice := system.NewSystem(m.ctx, m.cc)
	// m.ctx = context.WithValue(m.ctx, utils.CKeySystem, sysservice)
	// gofunc(&wg, sysservice.Bootstrap)

	// client event loop
	return m.loop(&wg)
}

//
//
//

func (m *Client) loop(wg *sync.WaitGroup) (_ error) {
	defer wg.Wait()
	defer m.log.Debug().Msg("waiting for opened goroutines")

	kernSignal := make(chan os.Signal, 1)
	signal.Notify(kernSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	m.log.Debug().Msg("initiate main event loop...")
	defer m.log.Debug().Msg("main event loop has been closed")

	m.log.Info().Msg("ready...")

LOOP:
	for {
		select {
		case <-kernSignal:
			m.log.Info().Msg("kernel signal has been caught; initiate application closing...")
			m.abort()
		case <-m.ctx.Done():
			m.log.Info().Msg("internal abort() has been caught; initiate application closing...")
			break LOOP
		}
	}

	return
}

func (m *Client) getAvailableCertificates() (_ []string, e error) {
	var response []byte
	if response, e = m.asmas.doApiRequest(ApiMethodCertificates); e != nil {
		return
	}

	return strings.Split(futils.UnsafeString(response), "|"), e
}

func (m *Client) getCertificatePublic(name string) (_ []byte, e error) {
	var response []byte
	if response, e = m.asmas.doApiRequest(ApiMethodCertificatePublic, name); e != nil {
		return
	}

	return response, e
}

func (m *Client) getCertificatePrivate(name string) (_ []byte, e error) {
	var response []byte
	if response, e = m.asmas.doApiRequest(ApiMethodCertificatePrivate, name); e != nil {
		return
	}

	return response, e
}

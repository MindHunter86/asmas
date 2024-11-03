package system

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"

	"github.com/containerd/fifo"
)

type PipeFile struct {
	payload []byte
	path    string
	c       context.Context

	mu   sync.RWMutex
	fd   *os.File
	fifo io.ReadWriteCloser

	log  *zerolog.Logger
	done func() <-chan struct{}
}

func NewPipeFile(c context.Context, path string) (pfile *PipeFile, e error) {
	pfile = &PipeFile{
		path: path,
		c:    c,

		log:  c.Value(utils.CKeyLogger).(*zerolog.Logger),
		done: c.Done,
	}

	fmt.Println(1)

	// var fd *os.File
	// if fd, e = os.OpenFile(path, os.O_RDONLY, 0); e == nil {
	// 	defer fd.Close()

	// 	var isfifo bool
	// 	if isfifo, e = fifo.IsFifo(path); e != nil {
	// 		return
	// 	} else if !isfifo {
	// 		return nil, fmt.Errorf("could not create pipe file, file %s is exists and it's not fifo", path)
	// 	}
	// } else if !errors.Is(e, os.ErrNotExist) {
	// 	return nil, fmt.Errorf("unexpected open() error, pipe file won't be created, %+v", e)
	// }

	fmt.Println(1)

	if pfile.fifo, e = fifo.OpenFifo(c, path, os.O_WRONLY|os.O_CREATE|unix.O_NONBLOCK, 0644); e != nil {
		return
	}

	fmt.Println(1)

	// if e = unix.Mkfifo(path, 0644); e != nil {
	// 	return nil, fmt.Errorf("an error occurred while calling mkfifo(), pipe file wont't be created, %s",
	// 		e.Error())
	// }

	// if fd, e = os.OpenFile(path, os.O_WRONLY, os.ModeNamedPipe); e != nil {
	// 	return nil, fmt.Errorf("an error occurred while creating pipe file, %s", e.Error())
	// }

	// unix.SetNonblock(int(fd.Fd()), false)

	return
}

func (m *PipeFile) StreamFilePayload() {
	// m.log.Trace().Msgf("starting pipe (%s) file streaming...", m.fd.Name())
	// defer m.log.Trace().Msgf("pipe file (%s) streaming has been stopped", m.fd.Name())
	fmt.Println(2)

LOOP:
	for {
		select {
		case <-m.done():
			m.log.Info().Msg("internal abort() has been caught; initiate application closing...")
			break LOOP
		default:
			fmt.Println(3)
			m.log.Trace().Msg("fifo has been called")
			// if written, e = m.fd.WriteString(fmt.Sprintf("test write:%s\n", time.Now().String())); e != nil {
			// 	m.log.Error().Msgf("an error occurred while writing to pipe file %s, %s", m.fd.Name(), e.Error())
			// }

			m.mu.Lock()
			fmt.Println(3)

			var buf *bytes.Buffer
			buf = bytes.NewBuffer(nil)
			buf.WriteString(fmt.Sprintf("test write:%s\n", time.Now().String()))
			_, e := m.fifo.Write(buf.Bytes())
			fmt.Println(e)
			// m.fd.Close()
			fmt.Println(3)

			m.mu.Unlock()

			m.fifo.Close()
			if m.fifo, e = fifo.OpenFifo(m.c, m.path, os.O_WRONLY|os.O_CREATE|unix.O_NONBLOCK, 0644); e != nil {
				fmt.Println(e)
			}

			// m.reopenfifo()
			time.Sleep(1250 * time.Millisecond)
			fmt.Println(3)
		}
	}

	// !! ERRORR!! payload []byte
	// m.closeAndRemove()

	// !!!!!
}

//
//
//

func (m *PipeFile) reopenfifo() (e error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if e = unix.Unlink(m.fd.Name()); e != nil {
		m.log.Error().Msgf("an error occurred while removing pipe file %s, %s", m.fd.Name(), e.Error())
		return
	}

	if e = unix.Mkfifo(m.fd.Name(), 0666); e != nil {
		return fmt.Errorf("an error occurred while calling mkfifo(), pipe file wont't be created, %s", e.Error())
	}

	if m.fd, e = os.OpenFile(m.fd.Name(), os.O_WRONLY, os.ModeNamedPipe); e != nil {
		return fmt.Errorf("an error occurred while creating pipe file, %s", e.Error())
	}

	unix.SetNonblock(int(m.fd.Fd()), false)
	return
}

func (m *PipeFile) closeAndRemove() (e error) {
	if e = m.fd.Close(); e != nil {
		m.log.Error().Msgf("an error occurred while closing pipe file %s, %s", m.fd.Name(), e.Error())
	}

	if e = unix.Unlink(m.fd.Name()); e != nil {
		m.log.Error().Msgf("an error occurred while removing pipe file %s, %s", m.fd.Name(), e.Error())
	}

	return
}

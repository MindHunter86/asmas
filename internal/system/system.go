package system

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

type System struct {
	certpath string

	pemstorage *PemStorage

	pemsizelimit           int64
	pembuffpool            *sync.Pool
	pempubname, pemkeyname string

	log   *zerolog.Logger
	done  func() <-chan struct{}
	abort context.CancelFunc
}

// !!!!
// !!!!
// !!!!
var ErrCertNotFound = errors.New("")
var ErrBufIsUndefined = errors.New("")

func NewSystem(c context.Context, cc *cli.Context) *System {
	return &System{
		certpath:   cc.String("system-cert-path"),
		pempubname: cc.String("system-pem-pubname"),
		pemkeyname: cc.String("system-pem-keyname"),

		pemsizelimit: cc.Int64("system-pem-size-limit"),
		pembuffpool: &sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, kbyteSize*cc.Int64("system-pem-size-limit")))
			},
		},

		pemstorage: NewPemStorage(c.Value(utils.CKeyLogger).(*zerolog.Logger)),

		log:   c.Value(utils.CKeyLogger).(*zerolog.Logger),
		done:  c.Done,
		abort: c.Value(utils.CKeyAbortFunc).(context.CancelFunc),
	}
}

// todo
// - ionotify
// 		- https://github.com/fsnotify/fsnotify
//
// - check open files limit

func (m *System) Bootstrap() {
	m.log.Debug().Msg("initiate system maintaining process")
	defer m.log.Debug().Msg("system maintaining process has been finished")

	if e := m.prepareCertificatePath(m.certpath); e != nil {
		m.log.Error().Msgf("an error occurred while preparing certificate path %s, %s", m.certpath, e.Error())
		m.abort()
		return
	}

	<-m.done()
	m.closeMaintainedFiles()
}

func (m *System) AcquireBuffer() *bytes.Buffer {
	return m.pembuffpool.Get().(*bytes.Buffer)
}

func (m *System) ReleaseBuffer(bb *bytes.Buffer) {
	bb.Reset()
	m.pembuffpool.Put(bb)
}

func (m *System) WritePemTo(domain string, ftype PemType, w io.Writer) (_ int, e error) {
	if w == nil {
		e = errors.New("BUG! undefined io.writer received")
		return
	}

	if m.pemstorage.st == nil {
		e = errors.New("pem storage is not ready yet")
		return
	}

	if m.pemstorage.st[domain] == nil {
		e = errors.New("given domain is not found in pem storage")
		return
	}

	var pfile *PemFile
	if pfile = m.pemstorage.st[domain][ftype]; pfile == nil {
		e = fmt.Errorf("BUG! there is no such pemtype (%d) for domain %s",
			int(ftype), domain)
		return
	}

	bb := m.AcquireBuffer()
	defer m.ReleaseBuffer(bb)

	bb.Reset()
	bb.Grow(int(pfile.Size))
	for i := 0; i < int(pfile.Size); i++ {
		bb.WriteByte(0)
	}

	if _, e = pfile.fd.ReadAt(bb.Bytes(), 0); e != nil {
		return
	}

	m.encodePayload(bb)
	return w.Write(bb.Bytes())
}

//
//
//

func (m *System) closeMaintainedFiles() {
	var e error

	m.pemstorage.VisitAll(func(domain string, pfiles []*PemFile) {
		for _, dfile := range pfiles {
			if dfile == nil {
				continue
			}

			if e = dfile.fd.Close(); e != nil {
				m.log.Error().Msg("an error occurred while closing opened PEM file, " + e.Error())
				continue
			}

			m.log.Trace().Msgf("file %s of domain %s has been closed", dfile.Name, dfile.Domain)
		}
	})
}

func (m *System) prepareCertificatePath(path string) (e error) {
	if e = m.peekPemsFromCertPath(path); e != nil {
		return
	}

	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		m.pemstorage.VisitAll(func(domain string, pfiles []*PemFile) {
			for _, dfile := range pfiles {
				if dfile == nil {
					continue
				}

				m.log.Trace().Msgf("found domain's (%s) file %s)", domain, dfile.fd.Name())
			}
		})
	}

	return
}

// return errors.New("given certificate path is file, not a directory")
// 	m.log.Trace().Msgf("file %s is not a directory, skipping...", fd.Name())
// 	m.log.Warn().Msgf("file %s is not accessable, skipping (%s)", fd.Name(), e.Error())

// Path fromat
// live/
// ├── third.example.com/
// │   ├── cert.pem -> ../../archive/third.example.com/cert6.pem
// │   ├── chain.pem -> ../../archive/third.example.com/chain6.pem
// │   ├── fullchain.pem -> ../../archive/third.example.com/fullchain6.pem
// │   ├── privkey.pem -> ../../archive/third.example.com/privkey6.pem
// │   └── README
func (m *System) peekPemsFromCertPath(certpath string) (e error) {
	var entries []os.DirEntry
	if entries, e = m.dirEntriesFromPath(certpath); e != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if e = m.peekPemsFromCertPath(filepath.Join(certpath, entry.Name())); e != nil {
				m.log.Error().Msg("an error occurred while peeking pem files from directory " + entry.Name())
				continue
			}

		}

		var pfile *PemFile
		if pfile, e = NewPemFile(filepath.Join(certpath, entry.Name()),
			WithPemSizeLimit(m.pemsizelimit),
			WithPemFileNamings(m.pempubname, m.pemkeyname)); e != nil {

			m.log.Debug().Msgf("an error occurred while preparing pem file %s, %s ", entry.Name(), e.Error())
			continue
		}

		m.pemstorage.Put(pfile)
	}

	return nil
}

func (*System) dirEntriesFromPath(path string) (entries []os.DirEntry, e error) {
	var dirfd *os.File
	if dirfd, e = os.Open(path); e != nil {
		return
	}

	var dirinfo os.FileInfo
	if dirinfo, e = dirfd.Stat(); e != nil {
		dirfd.Close()
		return
	}

	if !dirinfo.IsDir() {
		dirfd.Close()
		return nil, errors.New("given certificate directory is not a directory, please check the arguments")
	}

	if entries, e = dirfd.ReadDir(0); e != nil {
		dirfd.Close()
		return
	}

	return
}

func (m *System) encodePayload(bb *bytes.Buffer) {
	databuf := m.AcquireBuffer()
	defer m.ReleaseBuffer(databuf)

	databuf.Write(bb.Bytes())

	elen := base64.StdEncoding.EncodedLen(bb.Len())

	bb.Reset()
	bb.Grow(elen)
	for i := 0; i < int(elen); i++ {
		bb.WriteByte(0)
	}

	base64.StdEncoding.Encode(bb.Bytes(), databuf.Bytes())
}

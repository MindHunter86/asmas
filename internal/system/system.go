package system

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

type System struct {
	certpath string

	// !!!!!!!!!
	// ! MUTEX !
	mntdomains map[string]*os.File

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

type FileType uint8

const (
	PEM_CERTIFICATE FileType = iota
	PEM_PRIVATEKEY
	PEM_CHAIN
)

const kbyteSize int64 = 1024

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

		mntdomains: make(map[string]*os.File),

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
		// log m.certpath debug
		m.log.Error().Msg("an error occurred while preparing certificate path, " + e.Error())
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

func (m *System) PeekFile(ftype FileType, name string, bb *bytes.Buffer) (e error) {
	if bb == nil {
		return ErrBufIsUndefined
	}

	var path string
	switch ftype {
	case PEM_CERTIFICATE:
		path = filepath.Join(name, m.pempubname)
	case PEM_PRIVATEKEY:
		path = filepath.Join(name, m.pemkeyname)
	default:
		// !!!
		// !!!
		// !!!
		// !!!
		panic("under construction")
	}

	var ok bool
	var fd *os.File
	if fd, ok = m.mntdomains[path]; !ok {
		return ErrCertNotFound
	}

	if _, e = fd.Read(bb.Bytes()); e != nil {
		return
	}

	m.encodePayload(bb)
	return
}

//
//
//

func (m *System) closeMaintainedFiles() {
	var e error

	for _, file := range m.mntdomains {
		if e = file.Close(); e != nil {
			// log
		}
	}
}

func (m *System) prepareCertificatePath(path string) (e error) {
	var files []*os.File
	if files, e = m.getFilesFromDirectory(path); e != nil {
		return
	}

	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		for _, file := range files {
			m.log.Trace().Msgf("found file - %s", file.Name())
		}
	}

	// !!!
	panic(1)

	for _, file := range files {
		filename := filepath.Base(file.Name())
		if filename == "" || filename == "." {
			m.log.Warn().Msgf("file %s was skipped because of unexpected result of filename.Base()", file.Name())
		}

		// check file naming
		if filename != m.pempubname && filename != m.pemkeyname {
			if strings.HasSuffix(filename, ".pem") {
				m.log.Warn().Msg("found file with extension .pem but no listed in (pub,key) filename arguments")
			}

			m.log.Debug().Msgf("file %s was excluded from whitelist list because of arguments", filename)
			continue
		}

		var fileinfo os.FileInfo
		if fileinfo, _ = file.Stat(); e != nil {
			m.log.Warn().Msgf("file %s was skipped because of Stat() error, %s", filename, e.Error())
			continue
		}

		if fileinfo.Size() >= kbyteSize*m.pemsizelimit {
			m.log.Warn().Msgf("file %s was skipped because of filesize limit, file size %d, limit %d",
				filename, fileinfo.Size(), kbyteSize*m.pemsizelimit)
			continue
		}

		//? BUG
		// how to extract private\public pems?
		// ├── third.example.com
		// │   ├── cert.pem -> ../../archive/third.example.com/cert6.pem
		// │   ├── chain.pem -> ../../archive/third.example.com/chain6.pem
		// │   ├── fullchain.pem -> ../../archive/third.example.com/fullchain6.pem
		// │   ├── privkey.pem -> ../../archive/third.example.com/privkey6.pem
		// │   └── README

		paths := strings.Split(filepath.Join(path, filename), "/")
		newpath := strings.Join(paths[len(paths)-2:], "/")
		m.mntdomains[newpath] = file

		//

		var domain string
		//
		m.log.Info().Msgf("file %s (%s) was added in maintaining domain list", filename, domain)

		// ! NEED DEBUG!
	}

	return
}

// return errors.New("given certificate path is file, not a directory")
// 	m.log.Trace().Msgf("file %s is not a directory, skipping...", fd.Name())
// 	m.log.Warn().Msgf("file %s is not accessable, skipping (%s)", fd.Name(), e.Error())

func (m *System) getFilesFromDirectory(path string) (_ []*os.File, e error) {
	// open given directory for file scanning
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
		// log
		return
	}

	// scan directory for files
	var dirfiles []os.DirEntry
	if dirfiles, e = dirfd.ReadDir(0); e != nil {
		dirfd.Close()
		return
	}

	var fds []*os.File
	fds = append(fds, dirfd)

	for _, file := range dirfiles {
		m.log.Trace().Msgf("watching %s ...", file.Name())

		if file.IsDir() {
			// ! BUG
			// ? 2:41PM TRC system.go:156 > found file - testdata/live/super.domain
			// 2:41PM TRC system.go:156 > found file - testdata/live/super.domain/README
			// ? 2:41PM TRC system.go:156 > found file - testdata/live/test.example.com
			// 2:41PM TRC system.go:156 > found file - testdata/live/test.example.com/README

			var files []*os.File
			if files, e = m.getFilesFromDirectory(filepath.Join(path, file.Name())); e != nil {
				m.log.Trace().Msgf("an error occurred while watching dir %s, %s", file.Name(), e.Error())
				continue
			}

			m.log.Trace().Msgf("appending files from dir %s", file.Name())
			fds = append(fds, files...)
			continue
		}

		// check and resolve symlink
		if linkedfd, e := m.resolveSymlink(filepath.Join(path, file.Name())); e != nil {
			m.log.Trace().Msgf("an error occurred while resolving symlink %s, %s", file.Name(), e.Error())
			continue
		} else if linkedfd != nil {
			m.log.Trace().Msgf("appending files from symlink %s (%s)", file.Name(), linkedfd.Name())
			fds = append(fds, linkedfd)
			continue
		}

		// access as to regular file
		var filefd *os.File
		if filefd, e = os.Open(filepath.Join(path, file.Name())); e != nil {
			m.log.Trace().Msgf("an error occurred while opening file %s, %s", file.Name(), e.Error())
			continue
		}

		// log
		m.log.Trace().Msgf("appending file %s", file.Name())
		fds = append(fds, filefd)
	}

	return fds, e
}

func (m *System) resolveSymlink(path string) (_ *os.File, e error) {
	var fdinfo os.FileInfo
	if fdinfo, e = os.Lstat(path); e != nil {
		return
	}

	if fdinfo.Mode()&os.ModeSymlink == 0 {
		m.log.Trace().Msgf("given file has no symlink perm (%s)", fdinfo.Mode().String())
		return
	}

	var linkpath string
	if linkpath, e = os.Readlink(path); e != nil {
		return
	}

	var abspath string
	if abspath, e = filepath.Abs(linkpath); e != nil {
		return
	}

	return os.Open(abspath)
}

func (m *System) encodePayload(bb *bytes.Buffer) {
	databuf := m.AcquireBuffer()
	defer m.ReleaseBuffer(databuf)

	databuf.Write(bb.Bytes())

	elen := base64.StdEncoding.EncodedLen(bb.Len())
	bb.Reset()
	bb.Grow(elen)

	base64.StdEncoding.Encode(bb.Bytes(), databuf.Bytes())
}

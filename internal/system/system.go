package system

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

type System struct {
	certpath string

	// !!!!!!!!!
	// ! MUTEX !
	pemstorage *PemStorage

	pemlinks map[string]*os.File

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

		pemstorage: NewPemStorage(),

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
		e = errors.New("BUG! unexpected file type recevied in function for pem file")
		return
	}

	fmt.Println(path)

	// !!!!
	// !!!!
	// !!!!
	// var ok bool
	var fd *os.File
	// if fd, ok = m.mntdomains[path]; !ok {
	// 	return ErrCertNotFound
	// }

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
	// var e error

	// for _, file := range m.mntdomains {
	// 	if e = file.Close(); e != nil {
	// 		m.log.Error().Msg("an error occurred while closing opened PEM file, " + e.Error())
	// 	}
	// }
}

func (m *System) prepareCertificatePath(path string) (e error) {
	if e = m.peekPemsFromCertPath(path); e != nil {
		return
	}

	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		for name, file := range m.pemstorage.st {
			m.log.Trace().Msgf("found file - %s (%s)", name, file.fd.Name())
		}
	}

	// for _, file := range m.mntdomains {
	// 	filepaths := strings.Split(filepath.Clean(file.Name()), "/")
	// 	fnlen := len(filepaths)

	// 	filename, parent := filepaths[fnlen-1:][0], filepaths[fnlen-2 : fnlen-1][0]
	// 	if filename == "" || filename == "." {
	// 		m.log.Warn().Msgf("file %s was skipped because of unexpected result of filename.Base()", file.Name())
	// 	}

	// 	var fileinfo os.FileInfo
	// 	if fileinfo, _ = file.Stat(); e != nil {
	// 		m.log.Warn().Msgf("file %s was skipped because of Stat() error, %s", filename, e.Error())
	// 		continue
	// 	}

	// 	if fileinfo.Size() >= kbyteSize*m.pemsizelimit {
	// 		m.log.Warn().Msgf("file %s was skipped because of filesize limit, file size %d, limit %d",
	// 			filename, fileinfo.Size(), kbyteSize*m.pemsizelimit)
	// 		continue
	// 	}

	// 	newpath := filepath.Join(parent, filename)
	// 	m.mntdomains[newpath] = file

	// 	m.log.Info().Msgf("file %s (%s) was added in maintaining domain list", filename, newpath)
	// }

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
		if pfile, e = NewPemFile(filepath.Join(certpath, entry.Name())); e != nil {
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

	base64.StdEncoding.Encode(bb.Bytes(), databuf.Bytes())
}

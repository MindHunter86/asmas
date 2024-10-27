package system

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/MindHunter86/asmas/internal/utils"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

type System struct {
	certpath string

	// !!!!!!!!!
	// ! MUTEX !
	mntdomains map[string]*os.File

	log   *zerolog.Logger
	done  func() <-chan struct{}
	abort context.CancelFunc
}

func NewSystem(c context.Context, cc *cli.Context) *System {
	return &System{
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
		m.log.Error().Msg("an error occurred while preparing certificate path - " + e.Error())
		m.abort()
		return
	}

	<-m.done()
	m.closeMaintainedFiles()
}

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

	for _, file := range files {
		// check file naming
		if !strings.HasSuffix(file.Name(), ".pem") {
			// log
			continue
		}

		m.mntdomains[filepath.Join(path, file.Name())] = file

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
		if file.IsDir() {
			var files []*os.File
			if files, e = m.getFilesFromDirectory(filepath.Join(path, file.Name())); e != nil {
				// log
				continue
			}

			// log
			fds = append(fds, files...)
			continue
		}

		// check and resolve symlink
		if linkedfd, e := m.resolveSymlink(filepath.Join(path, file.Name())); e != nil {
			// log
			continue
		} else if linkedfd != nil {
			// log
			fds = append(fds, linkedfd)
			continue
		}

		// access as to regular file
		var filefd *os.File
		if filefd, e = os.Open(filepath.Join(path, file.Name())); e != nil {
			// log
			continue
		}

		// log
		fds = append(fds, filefd)
	}

	return fds, e
}

func (*System) resolveSymlink(path string) (_ *os.File, e error) {
	var fdinfo os.FileInfo
	if fdinfo, e = os.Stat(path); e != nil {
		return
	}

	if fdinfo.Mode().Perm()&os.ModeSymlink == 0 {
		return
	}

	var linkpath string
	if linkpath, e = os.Readlink(path); e != nil {
		return
	}

	return os.Open(linkpath)
}

// ├── third.example.com
// │   ├── cert.pem -> ../../archive/third.example.com/cert6.pem
// │   ├── chain.pem -> ../../archive/third.example.com/chain6.pem
// │   ├── fullchain.pem -> ../../archive/third.example.com/fullchain6.pem
// │   ├── privkey.pem -> ../../archive/third.example.com/privkey6.pem
// │   └── README

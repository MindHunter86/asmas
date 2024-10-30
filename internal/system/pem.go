package system

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FileType uint8

const (
	PEM_CERTIFICATE FileType = iota
	PEM_PRIVATEKEY
	PEM_CHAIN
)

type PemFile struct {
	Type FileType

	Name   string
	Domain string

	mu sync.RWMutex
	fd *os.File
}

type PemStorage struct {
	mu sync.RWMutex
	st map[string]*PemFile
}

func NewPemFile(path string) (pfile *PemFile, e error) {
	pfile = &PemFile{}

	if pfile.fd, e = openPemLink(path); e != nil {
		return
	}

	return pfile, pfile.prepareForMaintaining(path)
}

func openPemLink(path string) (_ *os.File, e error) {
	var fdinfo os.FileInfo
	if fdinfo, e = os.Lstat(path); e != nil {
		return
	}

	if fdinfo.Mode()&os.ModeSymlink == 0 {
		e = errors.New("given file has no symlink perm - " + fdinfo.Mode().String())
		return
	}

	var linkpath string
	if linkpath, e = filepath.EvalSymlinks(path); e != nil {
		return
	}

	return os.Open(linkpath)
}

func (m *PemFile) prepareForMaintaining(origpath string) (e error) {
	paths := strings.Split(filepath.Clean(origpath), "/")
	pathlen := len(paths)

	if pathlen <= 1 {
		e = errors.New("unexpected paths len, certificate maintaining will be skipped")
		return
	}

	m.Name = filepath.Base(origpath)
	m.Domain = paths[pathlen-2 : pathlen-1][0]

	switch m.Name {
	case "fullchain.pem":
		m.Type = PEM_CERTIFICATE
	case "privkey.pem":
		m.Type = PEM_PRIVATEKEY
	default:
		fmt.Println(m.Name)
		e = errors.New("unexpected filename, type is undefined, certificate maintaining will be skipped")
		return
	}

	return
}

func NewPemStorage() *PemStorage {
	return &PemStorage{
		st: make(map[string]*PemFile),
	}
}

func (m *PemStorage) Put(pemfile *PemFile) {
	actionWithLock(&m.mu, func() {
		fmt.Println(filepath.Join(pemfile.Domain, pemfile.Name))
		m.st[filepath.Join(pemfile.Domain, pemfile.Name)] = pemfile
	})
}

func (m *PemStorage) Get(domain string, pemtype FileType) (*PemFile, bool) {
	return actionReturbableWithRLock(&m.mu, func() (*PemFile, bool) {
		var basename string
		switch pemtype {
		case PEM_CERTIFICATE:
			basename = "fullchain.pem"
		case PEM_PRIVATEKEY:
			basename = "privkey.pem"
		}

		pfile, ok := m.st[filepath.Join(domain, basename)]
		return pfile, ok
	})
}

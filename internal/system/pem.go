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

const kbyteSize int64 = 1024

type (
	PemFile struct {
		Type FileType

		Name   string
		Domain string

		options *pemFileOptions

		mu sync.RWMutex
		fd *os.File
	}
	pemFileOptions struct {
		certname      string
		privatename   string
		filesizelimit int64
	}
	PFOption func(*pemFileOptions)
)

func NewPemFile(path string, options ...PFOption) (pfile *PemFile, e error) {
	pfile = &PemFile{}

	if pfile.fd, e = openPemLink(path); e != nil {
		return
	}

	pfile.options = withDefaultPemOptions()
	for _, option := range options {
		option(pfile.options)
	}

	return pfile, pfile.prepareForMaintaining(path)
}

func WithPemFileNamings(cname, pname string) PFOption {
	return func(pfo *pemFileOptions) {
		pfo.certname = cname
		pfo.privatename = pname
	}
}

func WithPemSizeLimit(size int64) PFOption {
	return func(pfo *pemFileOptions) {
		pfo.filesizelimit = size
	}
}

//
//
//

func withDefaultPemOptions() *pemFileOptions {
	return &pemFileOptions{
		certname:    "fullchain.pem",
		privatename: "privkey.pem",
	}
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
	case m.options.certname:
		m.Type = PEM_CERTIFICATE
	case m.options.privatename:
		m.Type = PEM_PRIVATEKEY
	default:
		fmt.Println(m.Name)
		e = errors.New("unexpected filename, type is undefined, certificate maintaining will be skipped")
		return
	}

	if m.options.filesizelimit != 0 {
		var fdinfo os.FileInfo
		if fdinfo, e = m.fd.Stat(); e != nil {
			return e
		}

		if fdinfo.Size() > kbyteSize*m.options.filesizelimit {
			return fmt.Errorf("could not maintain fiven file because of size limits, %d bytes (limit %d kbytes)",
				fdinfo.Size(), kbyteSize*m.options.filesizelimit)
		}
	}

	return
}

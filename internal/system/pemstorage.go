package system

import (
	"path/filepath"
	"sync"
)

type PemStorage struct {
	mu sync.RWMutex
	st map[string]*PemFile
}

func NewPemStorage() *PemStorage {
	return &PemStorage{
		st: make(map[string]*PemFile),
	}
}

func (m *PemStorage) Put(pemfile *PemFile) {
	actionWithLock(&m.mu, func() {
		m.st[filepath.Join(pemfile.Domain, pemfile.Name)] = pemfile
	})
}

func (m *PemStorage) Delete(path string) {
	actionWithLock(&m.mu, func() {
		delete(m.st, path)
	})
}

func (m *PemStorage) Get(domain string, pemtype FileType) (*PemFile, bool) {
	return actionReturbableWithRLock(&m.mu, func() (*PemFile, bool) {
		// !!!!
		// !!!!
		// !!!!
		// !!!!
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

func (m *PemStorage) VisitAll(visit func(_ string, _ *PemFile)) {
	actionWithRLock(&m.mu, func() {
		for k, v := range m.st {
			visit(k, v)
		}
	})
}

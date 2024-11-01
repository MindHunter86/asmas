package system

import (
	"sync"

	"github.com/rs/zerolog"
)

type PemStorage struct {
	mu sync.RWMutex
	st map[string][]*PemFile

	log *zerolog.Logger
}

func NewPemStorage(l *zerolog.Logger) *PemStorage {
	return &PemStorage{
		st:  make(map[string][]*PemFile),
		log: l,
	}
}

func (m *PemStorage) Put(pemfile *PemFile) {
	actionWithLock(&m.mu, func() {
		if m.st[pemfile.Domain] == nil {
			m.st[pemfile.Domain] = make([]*PemFile, _PEM_MAX_SIZE-1)
		}

		m.st[pemfile.Domain][pemfile.Type] = pemfile
	})
}

func (m *PemStorage) Delete(domain string) {
	actionWithLock(&m.mu, func() {
		delete(m.st, domain)
	})
}

func (m *PemStorage) Get(domain string, pemtype PemType) (*PemFile, bool) {
	return actionReturbableWithRLock(&m.mu, func() (*PemFile, bool) {
		pfile, ok := m.st[domain]

		if len(pfile) > int(pemtype) {
			m.log.Error().Msgf("BUG! there is no such pemtype (%d) for domain %s",
				int(pemtype), domain)
			return nil, false
		}

		return pfile[pemtype], ok
	})
}

func (m *PemStorage) VisitAll(visit func(_ string, _ []*PemFile)) {
	actionWithRLock(&m.mu, func() {
		for k, v := range m.st {
			visit(k, v)
		}
	})
}

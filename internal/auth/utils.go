package auth

import "sync"

type LocalKey uint8

const (
	LKeyName LocalKey = iota
	LKeyHostname
)

func IsEmpty(b []byte) bool {
	return len(b) == 0
}

func actionWithLock(mu *sync.RWMutex, action func()) {
	mu.Lock()
	defer mu.Unlock()

	action()
}

func actionReturbableWithRLock[V bool](mu *sync.RWMutex, action func() V) V {
	mu.RLock()
	defer mu.RUnlock()

	return action()
}

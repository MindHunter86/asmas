package auth

import "sync"

type LocalKey uint8

const (
	LKeyName LocalKey = iota
	LKeyHostname
)

func actionWithLock(mu *sync.RWMutex, action func()) {
	mu.Lock()
	defer mu.Unlock()

	action()
}

func actionReturbableWithRLock[V bool | []string](mu *sync.RWMutex, action func() V) V {
	mu.RLock()
	defer mu.RUnlock()

	return action()
}

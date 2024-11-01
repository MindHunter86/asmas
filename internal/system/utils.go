package system

import "sync"

func actionWithLock(mu *sync.RWMutex, action func()) {
	mu.Lock()
	defer mu.Unlock()

	action()
}

func actionWithRLock(mu *sync.RWMutex, action func()) {
	mu.RLock()
	defer mu.RUnlock()

	action()
}

func actionReturbableWithRLock[V *PemFile](mu *sync.RWMutex, action func() (V, bool)) (V, bool) {
	mu.RLock()
	defer mu.RUnlock()

	return action()
}

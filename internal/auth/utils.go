package auth

import "sync"

func IsEmpty(b []byte) bool {
	return len(b) == 0
}

func actionWithLock(mu *sync.RWMutex, action func()) {
	mu.Lock()
	defer mu.Unlock()

	action()
}

// func actionPayloadedWithRLock[V *YamlConfig](mu *sync.RWMutex, action func() V) V {
// 	mu.RLock()
// 	defer mu.RUnlock()

// 	return action()
// }

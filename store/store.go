package store

import "sync"

var kv = struct {
	sync.RWMutex
	data map[string]string
}{
	data: make(map[string]string),
}

func Set(key, value string) {
	kv.Lock()
	defer kv.Unlock()
	kv.data[key] = value
}

func Get(key string) (string, bool) {
	kv.RLock()
	defer kv.RUnlock()
	val, ok := kv.data[key]
	return val, ok
}

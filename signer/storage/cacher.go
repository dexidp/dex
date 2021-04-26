package storage

import (
	"sync/atomic"
	"time"

	"github.com/dexidp/dex/storage"
)

// newKeyCacher returns a storage which caches keys so long as the next
func newKeyCacher(s storage.KeyStorage, now func() time.Time) storage.KeyStorage {
	if now == nil {
		now = time.Now
	}
	return &keyCacher{KeyStorage: s, now: now}
}

type keyCacher struct {
	storage.KeyStorage

	now  func() time.Time
	keys atomic.Value // Always holds nil or type *storage.Keys.
}

func (k *keyCacher) GetKeys() (storage.Keys, error) {
	keys, ok := k.keys.Load().(*storage.Keys)
	if ok && keys != nil && k.now().Before(keys.NextRotation) {
		return *keys, nil
	}

	storageKeys, err := k.KeyStorage.GetKeys()
	if err != nil {
		return storageKeys, err
	}

	if k.now().Before(storageKeys.NextRotation) {
		k.keys.Store(&storageKeys)
	}
	return storageKeys, nil
}

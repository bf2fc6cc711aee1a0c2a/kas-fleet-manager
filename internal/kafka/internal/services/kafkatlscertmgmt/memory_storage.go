package kafkatlscertmgmt

import (
	"context"
	"io/fs"
	"strings"
	"sync"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/caddyserver/certmagic"
)

var _ certmagic.Storage = &inMemoryStorage{}

type inMemoryStorageItem struct {
	value []byte
	mu    *sync.Mutex
	sync.Locker
	lastModified time.Time
}

func (item inMemoryStorageItem) Lock() {
	item.mu.Lock()
}

func (item inMemoryStorageItem) Unlock() {
	item.mu.Unlock()
}

type inMemoryStorage struct {
	store map[string]inMemoryStorageItem
}

func newInMemoryStorage() *inMemoryStorage {
	return &inMemoryStorage{
		store: map[string]inMemoryStorageItem{},
	}
}

func (storage *inMemoryStorage) Lock(ctx context.Context, key string) error {
	mu, ok := storage.store[key]
	if !ok {
		mu = inMemoryStorageItem{
			lastModified: time.Now(),
			mu:           &sync.Mutex{},
		}
		storage.store[key] = mu
	}

	mu.Lock()
	return nil
}

func (storage *inMemoryStorage) Unlock(ctx context.Context, key string) error {
	mu, ok := storage.store[key]
	if !ok {
		return fs.ErrNotExist
	}

	mu.Unlock()
	return nil
}

func (storage *inMemoryStorage) Store(ctx context.Context, key string, value []byte) error {
	mu, ok := storage.store[key]
	if !ok {
		mu = inMemoryStorageItem{
			mu: &sync.Mutex{},
		}
	}
	mu.value = value
	mu.lastModified = time.Now()
	if strings.HasPrefix(key, "acme/") {
		logger.Logger.Infof("storing key '%s' with value %v", key, string(value))
	}
	storage.store[key] = mu
	return nil
}

func (storage *inMemoryStorage) Load(ctx context.Context, key string) ([]byte, error) {
	obj, ok := storage.store[key]

	if !ok {
		return nil, fs.ErrNotExist
	}

	return obj.value, nil
}

func (storage *inMemoryStorage) Delete(ctx context.Context, key string) error {
	delete(storage.store, key)
	return nil
}

func (storage *inMemoryStorage) Exists(ctx context.Context, key string) bool {
	_, ok := storage.store[key]
	return ok
}

func (storage *inMemoryStorage) List(ctx context.Context, prefix string, recursive bool) ([]string, error) {
	keys := make([]string, 0, len(storage.store))
	for k := range storage.store {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}

	return keys, nil
}

func (storage *inMemoryStorage) Stat(ctx context.Context, key string) (certmagic.KeyInfo, error) {
	obj, ok := storage.store[key]

	if !ok {
		return certmagic.KeyInfo{}, nil
	}

	return certmagic.KeyInfo{
		Key:        key,
		Modified:   obj.lastModified,
		Size:       int64(len(obj.value)),
		IsTerminal: strings.HasSuffix(key, "/"),
	}, nil
}

func (storage *inMemoryStorage) String() string {
	return "InMemoryStorage"
}

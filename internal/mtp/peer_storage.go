package mtp

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/gotd/contrib/storage"
)

// MemStorage is the default peer storage for MTP. It uses a map to store all
// peers, hence, it's not a persistent store.
type MemStorage struct {
	s map[string]storage.Peer

	mu        sync.RWMutex
	iterating bool

	// iterator
	keys   []string
	keyIdx int

	iterErr error
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		s: make(map[string]storage.Peer, 0),
	}
}

func (ms *MemStorage) Add(_ context.Context, value storage.Peer) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	key := storage.KeyFromPeer(value).String()
	ms.s[key] = value
	return nil
}

func (ms *MemStorage) Find(ctx context.Context, key storage.PeerKey) (storage.Peer, error) {
	return ms.Resolve(ctx, key.String())
}

func (ms *MemStorage) Assign(_ context.Context, key string, value storage.Peer) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.s[key] = value

	return nil
}

func (ms *MemStorage) Resolve(_ context.Context, key string) (storage.Peer, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	peer, ok := ms.s[key]
	if !ok {
		return storage.Peer{}, storage.ErrPeerNotFound
	}
	return peer, nil
}

func (ms *MemStorage) Iterate(ctx context.Context) (storage.PeerIterator, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if ms.IsIterating() {
		return nil, errors.New("already iterating")
	}

	// preparing the iterator
	ms.mu.Lock()
	ms.keys = make([]string, 0, len(ms.s))
	for k := range ms.s {
		ms.keys = append(ms.keys, k)
	}
	sort.Strings(ms.keys)
	ms.keyIdx = -1 // set the passphrase start value

	ms.iterating = true
	ms.iterErr = nil
	ms.mu.Unlock()

	// locking for iteration
	ms.mu.RLock()
	return ms, nil
}

func (ms *MemStorage) Next(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		ms.iterErr = ctx.Err()
		return false
	default:
	}
	ms.keyIdx++
	return ms.keyIdx < len(ms.keys)
}

func (ms *MemStorage) Err() error {
	return ms.iterErr
}

func (ms *MemStorage) Value() storage.Peer {
	if !ms.IsIterating() {
		return storage.Peer{}
	}
	return ms.s[ms.keys[ms.keyIdx]]
}

func (ms *MemStorage) Close() error {
	if !ms.IsIterating() {
		return nil
	}
	ms.mu.RUnlock()
	ms.mu.Lock()
	ms.iterating = false
	ms.mu.Unlock()
	return nil
}

func (ms *MemStorage) IsIterating() bool {
	return ms.iterating
}

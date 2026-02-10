package decisioning

import (
	"fmt"
	"sync"
)

// GrabLock provides per-media-item grab locking to prevent race conditions
// between RSS sync and auto-search attempting to grab the same item.
type GrabLock struct {
	mu    sync.Mutex
	locks map[string]struct{}
}

// NewGrabLock creates a new GrabLock.
func NewGrabLock() *GrabLock {
	return &GrabLock{
		locks: make(map[string]struct{}),
	}
}

// Key returns the lock key for a media item.
func Key(mediaType MediaType, mediaID int64) string {
	return fmt.Sprintf("%s:%d", mediaType, mediaID)
}

// TryAcquire attempts to acquire a lock for the given key.
// Returns true if the lock was acquired, false if already held.
func (g *GrabLock) TryAcquire(key string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, held := g.locks[key]; held {
		return false
	}
	g.locks[key] = struct{}{}
	return true
}

// Release releases the lock for the given key.
func (g *GrabLock) Release(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.locks, key)
}

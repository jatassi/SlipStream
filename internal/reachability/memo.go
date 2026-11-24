package reachability

import (
	"sync"
	"time"
)

// memo keeps an answer for a while, so the suggestions an open page asks
// for don't send a lookup on every read.
type memo[K comparable, V any] struct {
	mu   sync.Mutex
	kept map[K]memoized[V]
}

type memoized[V any] struct {
	v  V
	at time.Time
}

func (m *memo[K, V]) get(k K, now time.Time, life time.Duration) (V, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.kept[k]
	if !ok || now.Sub(e.at) >= life {
		var zero V
		return zero, false
	}
	return e.v, true
}

func (m *memo[K, V]) put(k K, v V, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.kept == nil {
		m.kept = map[K]memoized[V]{}
	}
	m.kept[k] = memoized[V]{v: v, at: now}
}

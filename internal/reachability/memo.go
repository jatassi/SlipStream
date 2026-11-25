package reachability

import (
	"sync"
	"time"
)

// memo keeps the last answer to a lookup and when it came.
type memo[K comparable, V any] struct {
	mu   sync.Mutex
	kept map[K]memoized[V]
}

type memoized[V any] struct {
	v  V
	at time.Time
}

// get is the last answer, whether there is one, and whether it is younger
// than life.
func (m *memo[K, V]) get(k K, now time.Time, life time.Duration) (v V, known, fresh bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.kept[k]
	return e.v, ok, ok && now.Sub(e.at) < life
}

// expire keeps every answer but makes it old, so the next lookup asks
// again.
func (m *memo[K, V]) expire() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, e := range m.kept {
		e.at = time.Time{}
		m.kept[k] = e
	}
}

func (m *memo[K, V]) put(k K, v V, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.kept == nil {
		m.kept = map[K]memoized[V]{}
	}
	m.kept[k] = memoized[V]{v: v, at: now}
}

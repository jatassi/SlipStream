package fake

import "sync"

// Shared is what the fakes of one world share, such as fake-tracker's
// swarm, which its HTTPS and UDP Specs both answer from. NewEnv gives each
// Env one of its own, and a world hands every fake of a run the same one.
type Shared struct {
	mu     sync.Mutex
	values map[string]any
}

func NewShared() *Shared { return &Shared{values: map[string]any{}} }

// Put keeps v as key's value.
func (s *Shared) Put(key string, v any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = v
}

// Get is key's value, false when none was put.
func (s *Shared) Get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.values[key]
	return v, ok
}

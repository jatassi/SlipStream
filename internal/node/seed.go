package node

import (
	"crypto/sha256"
	"math/rand/v2"
	"sync"
)

// Streams hands each subsystem its own random stream derived from the Node's
// seed, so a draw in one subsystem never shifts another's.
type Streams struct {
	seed string

	mu    sync.Mutex
	taken map[string]bool
}

func NewStreams(seed string) *Streams {
	return &Streams{seed: seed, taken: map[string]bool{}}
}

// Stream is ChaCha8 keyed with SHA-256 of the seed, a zero byte and name. It
// is safe for concurrent draws. Asking for one name twice panics, so two
// subsystems cannot share a stream by accident.
func (s *Streams) Stream(name string) *rand.Rand {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.taken[name] {
		panic("node: random stream " + name + " requested twice")
	}
	s.taken[name] = true
	key := sha256.Sum256([]byte(s.seed + "\x00" + name))
	return rand.New(&lockedSource{src: rand.NewChaCha8(key)}) //nolint:gosec // seeded on purpose so Scenarios replay
}

type lockedSource struct {
	mu  sync.Mutex
	src *rand.ChaCha8
}

func (l *lockedSource) Uint64() uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.src.Uint64()
}

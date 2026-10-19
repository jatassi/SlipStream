package metadata

import (
	"slices"
	"time"
)

// Tallied attaches a tally to a, as a holder does, and reads it back.
func Tallied(a Ask) (tallied Ask, read func() []Outcome) {
	t := &tally{}
	a.tally = t
	return a, func() []Outcome {
		t.mu.Lock()
		defer t.mu.Unlock()
		return slices.Clone(t.outcomes)
	}
}

// Cached sets the cache policy the Controller fills in.
func Cached(a Ask, keepFor time.Duration, generation uint64) Ask {
	a.cacheFor, a.generation = keepFor, generation
	return a
}

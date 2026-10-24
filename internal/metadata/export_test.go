package metadata

import (
	"testing"
	"time"
)

// Tallied attaches a tally to a, as a holder does, and reads it back.
func Tallied(a Ask) (ask Ask, read func() []Outcome) { return tallied(a) }

// Cached sets the cache policy the Controller fills in.
func Cached(a Ask, keepFor time.Duration, generation uint64) Ask {
	a.cacheFor, a.generation = keepFor, generation
	return a
}

// Bundle has this build carry keys, as "<provider>:<key>" pairs, for t.
func Bundle(t *testing.T, keys string) {
	was := bundled
	bundled = keys
	t.Cleanup(func() { bundled = was })
}

package metadata

import "time"

// Tallied attaches a tally to a, as a holder does, and reads it back.
func Tallied(a Ask) (ask Ask, read func() []Outcome) { return tallied(a) }

// Cached sets the cache policy the Controller fills in.
func Cached(a Ask, keepFor time.Duration, generation uint64) Ask {
	a.cacheFor, a.generation = keepFor, generation
	return a
}

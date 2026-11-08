package seeding

import (
	"errors"
	"time"
)

// Goal is how long the Admin seeds a torrent past its rule, and what then.
// It is Forever or Finite, so a goal that never ends can't carry an action.
type Goal interface{ goal() }

type Forever struct{}

// Finite counts a ratio, a seeding time or both. Only NewFinite builds
// one, so every Finite counts at least one of them.
type Finite struct {
	combine Combine
	ratio   Ratio
	seed    time.Duration
	then    Action
}

func (Forever) goal() {}
func (Finite) goal()  {}

var ErrGoal = errors.New("a goal counts a ratio, a seeding time or both")

// NewFinite builds a goal from its parts, zero for a part it doesn't
// count. It refuses a goal that counts neither.
func NewFinite(c Combine, ratio Ratio, seed time.Duration, then Action) (Finite, error) {
	if ratio == 0 && seed == 0 {
		return Finite{}, ErrGoal
	}
	return Finite{combine: c, ratio: ratio, seed: seed, then: then}, nil
}

// MustFinite is NewFinite for a goal known to count a part.
func MustFinite(c Combine, ratio Ratio, seed time.Duration, then Action) Finite {
	f, err := NewFinite(c, ratio, seed, then)
	if err != nil {
		panic(err)
	}
	return f
}

func (f Finite) Combine() Combine { return f.combine }
func (f Finite) Then() Action     { return f.then }

// Parts are its ratio and seeding time, zero for one it doesn't count.
func (f Finite) Parts() (Ratio, time.Duration) { return f.ratio, f.seed }

// Combine is how a Finite goal's ratio and time combine, as the API and
// trackers_goal spell it. With one part, both mean that part is met.
type Combine string

const (
	// First is "Whichever first".
	First Combine = "first"
	Both  Combine = "both"
)

// Action is what a met goal does to a Torrent.
type Action string

const (
	Pause  Action = "pause"
	Remove Action = "remove"
	// RemoveDelete removes the Torrent and deletes its download copy.
	RemoveDelete Action = "remove-delete"
)

// reached is whether t meets g, on seeded time. A part g doesn't count is
// never reached, and Both doesn't wait for it.
func reached(g Goal, t Totals, seeded time.Duration) bool {
	f, ok := g.(Finite)
	if !ok {
		return false
	}
	ratio, seed := f.ratio > 0 && RatioOf(t) >= f.ratio, f.seed > 0 && seeded >= f.seed
	if f.combine == Both {
		return (ratio || f.ratio == 0) && (seed || f.seed == 0)
	}
	return ratio || seed
}

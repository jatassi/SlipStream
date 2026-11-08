package seeding

import "time"

// Goal is how long the Admin seeds a torrent past its rule, and what then.
// It is Forever or Finite, so a goal that never ends can't carry an action.
type Goal interface{ goal() }

type Forever struct{}

type Finite struct {
	Combine Combine
	Ratio   Ratio
	Seed    time.Duration
	Then    Action
}

func (Forever) goal() {}
func (Finite) goal()  {}

// Combine is how a Finite goal's ratio and time combine, as the API and
// trackers_goal spell it.
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

// reached is whether t meets g, on seeded time.
func reached(g Goal, t Totals, seeded time.Duration) bool {
	f, ok := g.(Finite)
	if !ok {
		return false
	}
	ratio, seed := RatioOf(t) >= f.Ratio, seeded >= f.Seed
	if f.Combine == Both {
		return ratio && seed
	}
	return ratio || seed
}

package seeding

// Seen is whether a Torrent announces as a seeder, as far as its Node's
// reports show.
type Seen string

const (
	Announcing Seen = "announcing"
	Quiet      Seen = "quiet"
	// Unseen is a Torrent whose Node is out of step, so its last report
	// can't say.
	Unseen Seen = "unseen"
)

// SeenOf is the at-risk table over S28's state word and hold, as
// downloadclients spells them. libtorrent queueing is off and a complete
// torrent never takes a download slot or the disk guard, so the only hold
// that keeps announcing is a window or start-spread pause, which lasts at
// most one announce interval. Checking announces nothing, so it is quiet
// at once and Health still waits its delay.
func SeenOf(state, hold string, inStep bool) Seen {
	switch {
	case !inStep:
		return Unseen
	case state == "seeding", state == "downloading", state == "stalled", state == "paused" && hold == "window":
		return Announcing
	}
	return Quiet
}

// Risk is whether an owing window is at risk of a hit and run.
type Risk string

const (
	Safe   Risk = "no"
	AtRisk Risk = "yes"
	// RiskUnknown is a window whose only copies are on Nodes out of step.
	// Nodes raises its own Health item for them.
	RiskUnknown Risk = "unknown"
)

// riskOf is the hash's risk: one copy announcing is enough.
func riskOf(copies []Copy) Risk {
	r := AtRisk
	for _, c := range copies {
		switch c.Seen {
		case Announcing:
			return Safe
		case Unseen:
			r = RiskUnknown
		case Quiet:
		}
	}
	return r
}

package plan

import (
	"slices"

	"github.com/jatassi/slipstream/internal/encode"
)

// Qualifies is the first clause n fails for j, or Fits. An away Node that
// hasn't lapsed qualifies.
func Qualifies(n Node, j Job) Why {
	switch {
	case !n.Holds:
		return NoRole
	case n.Lapsed:
		return Lapsed
	case j.RunOn != nil && *j.RunOn != n.ID:
		return RunsElsewhere
	case j.Pin != nil && *j.Pin != n.ID:
		return PinnedElsewhere
	case j.Exclude[n.ID]:
		return FailedHere
	}
	p, ok := n.Proofs[j.Key]
	switch {
	case !ok:
		return Untested
	case p.Failed:
		return TestFailed
	case p.Lost && j.Pin != nil:
		return Retesting
	case p.Lost:
		return ProofLost
	case p.Pipeline.Path == encode.Processor && !n.Processor:
		return ProcessorOff
	}
	return Fits
}

// base is the key every holder proves first: SDR H.264 from an H.264 clip.
var base = encode.Key{Shape: encode.Defaults().Shape(), Class: encode.ClassH264x8}

// owed is the keys n owes a test run of for the due jobs: base while it has
// no proof row of it, then each key a job waits on n to prove.
func owed(n Node, jobs []Job) []encode.Key {
	if !n.Holds || n.Lapsed {
		return nil
	}
	var keys []encode.Key
	if _, ok := n.Proofs[base]; !ok {
		keys = append(keys, base)
	}
	for _, j := range jobs {
		if provable(Qualifies(n, j)) && !slices.Contains(keys, j.Key) {
			keys = append(keys, j.Key)
		}
	}
	return keys
}

func provable(w Why) bool { return w == Untested || w == Retesting }

// causeRank orders the Whys of Nodes that don't qualify by how close each
// is to qualifying. A Why not listed ranks last.
var causeRank = map[Why]int{
	ProcessorOff:  0,
	TestFailed:    1,
	ProofLost:     1,
	RunsElsewhere: 2,
	Lapsed:        3,
	NoRole:        4,
}

func rank(w Why) int {
	if r, ok := causeRank[w]; ok {
		return r
	}
	return len(causeRank)
}

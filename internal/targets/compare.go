package targets

import (
	"cmp"
	"fmt"
	"strconv"
)

// Step is one step of a Target's order, which fills and ranking share.
type Step uint8

const (
	ByTier Step = iota + 1
	ByRevision
	ByScore
)

// Steps is t's order: tier, then revision unless DontPrefer, then score.
// Revision compares across the Qualities of a tier here, as Sonarr's
// ranking does, while an upgrade compares it within one Quality.
func (t Target) Steps() []Step {
	if t.Propers == DontPrefer {
		return []Step{ByTier, ByScore}
	}
	return []Step{ByTier, ByRevision, ByScore}
}

// CompareBy compares a and b at one step: +1 when a is better, -1 when b
// is. A quality t doesn't accept ranks below every tier.
func (t Target) CompareBy(s Step, a, b Scored) int {
	switch s {
	case ByTier:
		return cmp.Compare(t.rankOf(b), t.rankOf(a))
	case ByRevision:
		return a.Facts.Revision.Compare(b.Facts.Revision)
	case ByScore:
		return cmp.Compare(t.Score(a), t.Score(b))
	}
	panic(fmt.Sprintf("targets: no step %d", s))
}

// Compare is the first of Steps that a and b differ on, and which is
// better: +1 for a, -1 for b. Equal at every step is 0, 0.
func (t Target) Compare(a, b Scored) (step Step, better int) {
	for _, s := range t.Steps() {
		if c := t.CompareBy(s, a, b); c != 0 {
			return s, c
		}
	}
	return 0, 0
}

// Value is a's value at s in words: its quality's name, its revision as
// "v2", "REAL v1" or "v2 REPACK", or its score.
func (t Target) Value(s Step, a Scored) string {
	switch s {
	case ByTier:
		q, _ := a.Facts.Quality.Get()
		return q.Info().Name
	case ByRevision:
		return a.Facts.Revision.String()
	case ByScore:
		return strconv.Itoa(t.Score(a))
	}
	panic(fmt.Sprintf("targets: no step %d", s))
}

func (t Target) rankOf(s Scored) Rank {
	if q, known := s.Facts.Quality.Get(); known {
		if r, ok := t.RankOf(q); ok {
			return r
		}
	}
	return Rank(len(t.Tiers))
}

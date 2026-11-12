package grabs

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/targets"
)

const (
	StepWant     Step = "want"
	StepTier     Step = "tier"
	StepRevision Step = "revision"
	StepScore    Step = "score"
	// StepChosenByAnother and StepFewerUnwanted are the pack tie-breaks:
	// the Release another Want already chose, then the one covering fewer
	// Items that aren't wanted.
	StepChosenByAnother Step = "chosen_by_another"
	StepFewerUnwanted   Step = "fewer_unwanted"
	StepPriority        Step = "priority"
	StepFreeleech       Step = "freeleech"
	StepSeeders         Step = "seeders"
	StepSize            Step = "size"
	StepTitle           Step = "title"
)

// Steps is every Step, in the ranking's order.
var Steps = []Step{
	StepWant, StepTier, StepRevision, StepScore, StepChosenByAnother, StepFewerUnwanted, StepPriority, StepFreeleech, StepSeeders,
	StepSize, StepTitle,
}

// rung is one step of the chain: compare is +1 when a ranks first, and
// value is a's value at the step. at, when set, is the step of the
// Target of a's first cell it compares at.
type rung struct {
	step    Step
	at      targets.Step
	compare func(c cells, a, b judged) int
	value   func(c cells, a judged) string
}

// chain is the ranking order, which both sorts and says where a Release
// lost. Tier, revision and score compare at the Target of each Release's
// own first cell, so only two Releases with the same first cell reach
// them.
var chain = [...]rung{
	{StepWant, 0, byFirstCell, firstCell},
	atTarget(StepTier, targets.ByTier),
	atTarget(StepRevision, targets.ByRevision),
	atTarget(StepScore, targets.ByScore),
	onRelease(StepPriority, lowerFirst, func(r indexers.Release) int { return r.Priority }, strconv.Itoa),
	onRelease(StepFreeleech, lowerFirst, func(r indexers.Release) float64 { return r.Factors.Download }, func(f float64) string {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}),
	onRelease(StepSeeders, higherFirst, func(r indexers.Release) int { return r.Seeders }, strconv.Itoa),
	onRelease(StepSize, higherFirst, func(r indexers.Release) int64 { return r.Size }, func(n int64) string { return strconv.FormatInt(n, 10) }),
	onRelease(StepTitle, lowerFirst, func(r indexers.Release) string { return r.Title }, func(t string) string { return t }),
}

func byFirstCell(_ cells, a, b judged) int { return cmp.Compare(b.first, a.first) }

// firstCell is the cell r is first grabbable at in words: its Target's
// name, and the Edition's cut unless it takes any.
func firstCell(c cells, r judged) string {
	cell := c.at(r.first)
	if cell.Edition == targets.AnyEdition {
		return cell.Target.Name
	}
	return cell.Target.Name + " (" + string(cell.Edition) + ")"
}

// atTarget compares at s of the Target of a's first cell, which is b's
// too.
func atTarget(step Step, s targets.Step) rung {
	return rung{
		step: step, at: s,
		compare: func(c cells, a, b judged) int {
			return c.at(a.first).Target.CompareBy(s, &a.scored, &b.scored)
		},
		value: func(c cells, r judged) string { return c.at(r.first).Target.Value(s, &r.scored) },
	}
}

type order int

const (
	lowerFirst  order = -1
	higherFirst order = 1
)

// onRelease compares a value of the Release itself, which ranks first in
// order o.
func onRelease[V cmp.Ordered](step Step, o order, of func(indexers.Release) V, words func(V) string) rung {
	return rung{
		step:    step,
		compare: func(_ cells, a, b judged) int { return int(o) * cmp.Compare(of(a.Release), of(b.Release)) },
		value:   func(_ cells, r judged) string { return words(of(r.Release)) },
	}
}

// differ is the first rung a and b differ at and +1 when a ranks first,
// false when they tie at every step.
func differ(c cells, a, b judged) (rung, int, bool) {
	for _, l := range chain {
		if d := l.compare(c, a, b); d != 0 {
			return l, d, true
		}
	}
	return rung{}, 0, false
}

// rank sorts rs, each grabbable at some cell, by the chain.
func rank(c cells, rs []judged) {
	slices.SortStableFunc(rs, func(a, b judged) int {
		_, d, _ := differ(c, a, b)
		return -d
	})
}

// Loss is the first step of the chain where the reference Release ranked
// above a Release, and each one's value there. Target is the Target the
// tier, revision and score steps compare at.
type Loss struct {
	Step   Step
	Target targets.TargetID
	Own    string
	Ref    string
}

// lostAt is where ref ranked above r, nil when it didn't: when they tie
// at every step, or r ranks first, as a copy of ref offered after it can.
func lostAt(c cells, ref, r judged) *Loss {
	l, d, ok := differ(c, ref, r)
	if !ok || d < 0 {
		return nil
	}
	loss := &Loss{Step: l.step, Own: l.value(c, r), Ref: l.value(c, ref)}
	if l.at != 0 {
		loss.Target = c.at(r.first).Target.ID
	}
	return loss
}

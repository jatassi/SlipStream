package imports

import (
	"errors"
	"slices"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
)

// Fill is a cell a file fills, decided and then as it landed, with Forced
// when the Admin forced it.
type Fill struct {
	Cell   files.CellKey
	Forced bool
}

// outcome is what a Verdict makes of a file.
type outcome struct {
	To     Outcome
	Reason *Reason
	Fills  []Fill
}

// outcomeContext is what outcomes reads beside the Verdicts. othersFill
// reports whether a file of the Import outside the batch is sending,
// placing or placed. sends reports whether a file that fills something is
// sent to another Node first. cut is each candidate's cut, confirmed
// whether a folder file is past its confirm, and forced each one's forced
// cells.
type outcomeContext struct {
	othersFill bool
	source     Source
	sends      bool
	cut        func(i int) release.CutKey
	confirmed  func(i int) bool
	forced     func(i int) []files.CellKey
}

// outcomes turns the Verdicts of one batch into outcomes, at the decision
// and at landing alike: the spec's decision table, first match wins. A
// folder's files aren't one Release, so each that fills nothing waits
// with its reason for the Admin. Pure.
func outcomes(vs []files.Verdict, c outcomeContext) []outcome {
	anyFills := c.othersFill || slices.ContainsFunc(vs, func(v files.Verdict) bool { return len(v.Fills) > 0 })
	out := make([]outcome, len(vs))
	for i, v := range vs {
		out[i] = outcomeOf(i, v, vs, anyFills, c)
	}
	return out
}

func outcomeOf(i int, v files.Verdict, vs []files.Verdict, anyFills bool, c outcomeContext) outcome {
	waits := func(r Reason) outcome { return outcome{To: Waiting, Reason: &r} }
	if taken, ok := errors.AsType[*files.NameTakenError](v.Refusal); ok {
		return waits(Reason{Code: NameTaken, Name: taken.Path})
	}
	if long, ok := errors.AsType[*naming.TooLongError](v.Refusal); ok {
		return waits(Reason{Code: NameTooLong, Name: long.Segment, Length: long.Bytes, Limit: long.Limit})
	}
	switch {
	case v.EditionGone:
		return waits(Reason{Code: EditionNotWanted, Edition: c.cut(i)})
	case len(v.Meets) == 0 && (c.source == FromFolder || len(vs) == 1 && !c.othersFill):
		return waits(Reason{Code: MeetsNothing, Targets: targetsOf(reasonTargets(v))})
	case len(v.Meets) == 0:
		return outcome{To: LeftOut, Reason: &Reason{Code: MeetsNothing, Targets: targetsOf(reasonTargets(v))}}
	case len(v.Fills) > 0 && c.source == FromFolder && !c.confirmed(i):
		return outcome{To: Waiting, Reason: &Reason{Code: Confirm, Targets: targetsOf(v.Fills)}, Fills: fillsOf(v.Fills, c.forced(i))}
	case len(v.Fills) > 0 && c.sends:
		return outcome{To: Sending, Fills: fillsOf(v.Fills, c.forced(i))}
	case len(v.Fills) > 0:
		return outcome{To: Placing, Fills: fillsOf(v.Fills, c.forced(i))}
	case lostToSiblings(v):
		return outcome{To: LeftOut, Reason: &Reason{Code: SiblingBetter}}
	case anyFills && c.source != FromFolder:
		return outcome{To: NotBetter, Reason: &Reason{Code: KeptBetter}}
	}
	return waits(Reason{Code: NotAnUpgrade, Targets: targetsOf(v.Meets)})
}

// lostToSiblings reports whether another file of the batch holds every
// cell v meets and doesn't fill.
func lostToSiblings(v files.Verdict) bool {
	if len(v.Lost) == 0 {
		return false
	}
	for _, h := range v.Lost {
		if h != files.HeldBySibling {
			return false
		}
	}
	return true
}

func reasonTargets(v files.Verdict) []files.CellKey {
	out := make([]files.CellKey, 0, len(v.Reasons))
	for id := range v.Reasons {
		out = append(out, files.CellKey{Target: id})
	}
	slices.SortFunc(out, func(a, b files.CellKey) int { return a.Target.Compare(b.Target.UUID) })
	return out
}

// targetsOf is the Targets of cells, each once, in the cells' order.
func targetsOf(cells []files.CellKey) []targets.TargetID {
	var out []targets.TargetID
	for _, c := range cells {
		if !slices.Contains(out, c.Target) {
			out = append(out, c.Target)
		}
	}
	return out
}

func fillsOf(cells, forced []files.CellKey) []Fill {
	out := make([]Fill, len(cells))
	for i, c := range cells {
		out[i] = Fill{Cell: c, Forced: slices.Contains(forced, c)}
	}
	return out
}

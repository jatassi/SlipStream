package conversions

import (
	"path"
	"time"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Pair is a source file and the Target a Conversion makes a file for from
// it: at most one unfinished Conversion holds it.
type Pair struct {
	Source files.ID
	Target targets.TargetID
}

// Want is a Conversion a made-from cell wants.
type Want struct {
	Pair
	Work     works.WorkID
	Items    []works.ItemID
	Edition  targets.Edition
	Sourcing targets.Sourcing
	Grace    time.Duration
	Version  int
	// Upgrade is that the cell holds a file the made one would replace.
	Upgrade    bool
	SourceName string
}

// desired is the Conversions cells want. goals are the library's Targets
// by id. declined is the pairs an Admin took back, which an explicit ask
// asks for again. A remake makes a file even where the made one judges no
// better than the one there.
func desired(cells []files.Sourced, goals map[targets.TargetID]targets.Target, declined map[Pair]bool, a ask) []Want {
	var out []Want
	seen := map[Pair]bool{}
	for _, c := range cells {
		if !c.Counted || !c.Released || c.FilledByDownload() {
			continue
		}
		src, ok := c.Source()
		if !ok {
			continue
		}
		t, ok := goals[c.Target]
		if !ok || t.Conversion == nil {
			continue
		}
		pair := Pair{Source: src.ID, Target: c.Target}
		if seen[pair] || declined[pair] && !a.explicit() {
			continue
		}
		conv := *t.Conversion
		if c.File != nil && !replaces(t, src, *c.File, a) {
			continue
		}
		seen[pair] = true
		out = append(out, Want{
			Pair: pair, Work: c.Work, Items: c.Items, Edition: c.Edition, Sourcing: c.Sourcing, Grace: c.Grace,
			Version: conv.Version, Upgrade: c.File != nil, SourceName: path.Base(src.Path),
		})
	}
	return out
}

// replaces is whether a file t makes from src would replace held, the
// file filling the cell: never one src made with t's settings, and else
// when a remake asks for it or the made file upgrades held.
func replaces(t targets.Target, src, held files.File, a ask) bool {
	if d := held.Derived; d != nil && d.From == src.ID && d.Version == t.Conversion.Version {
		return false
	}
	return a.Why == files.ArrivalRemake || upgrades(t, encode.Predict(src.Title, src.Media, t.Conversion.Settings), held)
}

// upgrades is whether made, a predicted file, upgrades held at t.
func upgrades(t targets.Target, made release.Read, held files.File) bool {
	was := targets.Scored{Facts: held.Facts()}
	_, ok := targets.Judge(t, targets.Scored{Facts: made.Facts}, &targets.Held{Facts: was.Facts, Score: t.Score(&was)}).Upgrade()
	return ok
}

// cancel is an unfinished Conversion no cell wants, and why.
type cancel struct {
	ID  ID
	Why CancelWhy
}

// diff is the wants no unfinished Conversion of have holds, and the
// unfinished ones, failed included, no want holds.
func diff(have []Row, want []Want) (inserts []Want, cancels []cancel) {
	held := map[Pair]bool{}
	for _, r := range have {
		if p, ok := r.pair(); ok {
			held[p] = true
		}
	}
	wanted := map[Pair]bool{}
	for _, w := range want {
		wanted[w.Pair] = true
		if !held[w.Pair] {
			inserts = append(inserts, w)
		}
	}
	for _, r := range have {
		p, ok := r.pair()
		switch {
		case r.SourceFile == nil:
			cancels = append(cancels, cancel{ID: r.ID, Why: SourceGone})
		case !ok || !wanted[p]:
			cancels = append(cancels, cancel{ID: r.ID, Why: NotWanted})
		}
	}
	return inserts, cancels
}

// pair is the row's pair, false once its source file or Target is gone.
func (r Row) pair() (Pair, bool) {
	if r.SourceFile == nil || r.Target == nil {
		return Pair{}, false
	}
	return Pair{Source: *r.SourceFile, Target: *r.Target}, true
}

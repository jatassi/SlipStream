package grabs

import (
	"context"
	"maps"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// judging is what assessing a run's Releases needs, read before the
// Indexers are asked: the searched Item, what its Work expects of a name,
// its Content Type's Names and cuts, the formats its Work's Targets score
// and each of its Items' runtime.
type judging struct {
	searched works.ItemID
	expect   indexers.Expect
	names    release.Names
	cuts     []release.Cut
	scoring  *formats.Catalog
	runtime  map[works.ItemID]release.Fact[time.Duration]
}

// judgingOf reads, in r, the judging of run's Releases, run's Work being
// of Content Type typ and expecting expect.
func (gr *Grabs) judgingOf(ctx context.Context, r storage.Reader[file], typ works.TypeID, run Search, expect indexers.Expect) (judging, error) {
	known, err := gr.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return judging{}, err
	}
	timings, err := gr.cfg.Targets.TimingsIn(ctx, r, typ, works.Only([]works.WorkID{run.WorkID}))
	if err != nil {
		return judging{}, err
	}
	goals, err := targets.GoalsIn(ctx, r, run.WorkID)
	if err != nil {
		return judging{}, err
	}
	all, err := gr.cfg.Targets.FormatsIn(ctx, r)
	if err != nil {
		return judging{}, err
	}
	runtimes := make(map[works.ItemID]release.Fact[time.Duration], len(timings))
	for id, t := range timings {
		runtimes[id] = t.Runtime
	}
	return judging{
		searched: run.ItemID, expect: expect, names: gr.cfg.Names[typ], cuts: known.For(typ),
		scoring: all.Only(targets.ScoredIn(goals)), runtime: runtimes,
	}, nil
}

// assessed is one listed Release read against its Work before any cell
// judges it: its reading, the Items of the Work it covers, the formats it
// matches of those the Work's Targets score, and its own refusals. A
// Release with a hard refusal isn't matched against any format.
type assessed struct {
	listed
	read     release.Read
	covers   []works.ItemID
	formats  formats.Result
	refusals []Reason
}

// assess reads l against j. It is pure and deterministic.
func assess(j judging, l listed) assessed {
	r := l.Release
	a := assessed{listed: l, read: release.Title(r.Title, j.names, j.cuts)}
	a.covers = j.expect.Covers(a.read.Name)
	f := &a.read.Facts
	if r.Size > 0 {
		f.Size = release.Known(r.Size)
	}
	f.Flags = r.Flags
	f.Coverage = release.Coverage{Runtime: j.runtimeOf(a.covers), Original: j.expect.Original}
	a.refusals = j.refusals(a)
	if !a.hard() {
		a.formats = j.scoring.Match(a.read)
	}
	return a
}

// assessAll is assess over ls across GOMAXPROCS goroutines, in ls' order.
func assessAll(j judging, ls []listed) []assessed {
	out := make([]assessed, len(ls))
	var next atomic.Int64
	var wg sync.WaitGroup
	for range min(runtime.GOMAXPROCS(0), len(ls)) {
		wg.Go(func() {
			for i := int(next.Add(1) - 1); i < len(ls); i = int(next.Add(1) - 1) {
				out[i] = assess(j, ls[i])
			}
		})
	}
	wg.Wait()
	return out
}

// runtimeOf is the runtime of items summed, unknown when there are none
// or any one's is.
func (j judging) runtimeOf(items []works.ItemID) release.Fact[time.Duration] {
	if len(items) == 0 {
		return release.Fact[time.Duration]{}
	}
	var sum time.Duration
	for _, it := range items {
		d, ok := j.runtime[it].Get()
		if !ok {
			return release.Fact[time.Duration]{}
		}
		sum += d
	}
	return release.Known(sum)
}

// refusals is a's own refusals: an id that names another Work alone, else
// a title or year that doesn't match the Work, then a name that covers
// none of the searched Items or more than it. Its ids are the Indexer's,
// over the id tags its name carries.
func (j judging) refusals(a assessed) []Reason {
	ids := map[string]string{}
	maps.Copy(ids, a.read.Facts.IDs)
	maps.Copy(ids, a.Release.IDs)
	m, given := release.Match(a.read.Name, ids, j.expect.Claim, j.names)
	if m == release.WrongWork {
		return []Reason{own(targets.WrongWork, "")}
	}
	var out []Reason
	if m == release.WrongTitle {
		out = append(out, own(targets.WrongTitle, given))
	}
	switch {
	case !slices.Contains(a.covers, j.searched):
		out = append(out, own(targets.WrongItem, marks(a.read.Name)))
	case len(a.covers) > 1:
		out = append(out, own(targets.CoversMore, marks(a.read.Name)))
	}
	return out
}

func own(code targets.ReasonCode, value string) Reason {
	return Reason{Reason: targets.Reason{Code: code, Value: value}}
}

// hard reports a hard refusal of a's own, which stops judging.
func (a assessed) hard() bool {
	return slices.ContainsFunc(a.refusals, func(r Reason) bool { return r.Code.Hard() })
}

// namesTheWork reports that no id a's name or its Indexer gives names
// another Work and its title and year match the Work.
func (a assessed) namesTheWork() bool {
	return !slices.ContainsFunc(a.refusals, func(r Reason) bool { return r.Code == targets.WrongWork || r.Code == targets.WrongTitle })
}

func (a assessed) scored() targets.Scored {
	return targets.Scored{Facts: a.read.Facts, Formats: a.formats.Matched}
}

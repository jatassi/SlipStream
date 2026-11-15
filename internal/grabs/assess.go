package grabs

import (
	"cmp"
	"context"
	"maps"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// judging is what assessing a run's Releases needs, read before the
// Indexers are asked: scope, the Items a Release must cover one of, nil
// for any of the Work's, as the tester and Manual search's sheet judge;
// the code of each of the Work's Items that hasn't aired by its own date,
// before any delay, or has no date; the Releases failed Grabs of the Work
// tried; what its Work expects of a name, its Content Type's Names and
// cuts, the formats its Work's Targets score and each of its Items'
// runtime.
type judging struct {
	scope   map[works.ItemID]bool
	unaired map[works.ItemID]string
	tried   []triedRelease
	expect  indexers.Expect
	names   release.Names
	cuts    []release.Cut
	scoring *formats.Catalog
	runtime map[works.ItemID]release.Fact[time.Duration]
}

// scopeOf is items as a judging scope.
func scopeOf(items []works.ItemID) map[works.ItemID]bool {
	out := make(map[works.ItemID]bool, len(items))
	for _, it := range items {
		out[it] = true
	}
	return out
}

// judgingOf reads, in r, the judging of Releases of tree's Work searched
// for scope and expecting expect.
func (gr *Grabs) judgingOf(ctx context.Context, r storage.Reader[file], tree works.Tree, scope map[works.ItemID]bool, expect indexers.Expect) (judging, error) {
	typ, work := tree.ContentType, tree.ID
	known, err := gr.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return judging{}, err
	}
	timings, err := gr.cfg.Targets.TimingsIn(ctx, r, typ, works.Only([]works.WorkID{work}))
	if err != nil {
		return judging{}, err
	}
	goals, err := targets.GoalsIn(ctx, r, work)
	if err != nil {
		return judging{}, err
	}
	all, err := gr.cfg.Targets.FormatsIn(ctx, r)
	if err != nil {
		return judging{}, err
	}
	tried, err := triedOf(ctx, r, work)
	if err != nil {
		return judging{}, err
	}
	runtimes := make(map[works.ItemID]release.Fact[time.Duration], len(timings))
	for id, t := range timings {
		runtimes[id] = t.Runtime
	}
	nouns, _ := gr.cfg.Works.Describe(typ)
	today, unaired := time.Now(), map[works.ItemID]string{}
	for _, it := range tree.Items {
		if !timings[it.ID].ReleasedBy(today) {
			unaired[it.ID] = cmp.Or(itemCode(nouns, it), it.Label)
		}
	}
	return judging{
		scope: scope, unaired: unaired, tried: tried, expect: expect, names: gr.cfg.Names[typ], cuts: known.For(typ),
		scoring: all.Only(targets.ScoredIn(goals)), runtime: runtimes,
	}, nil
}

// itemCode is the code the Work page leads it with, "S02E10", "" when its
// type, described by d, places none.
func itemCode(d works.Description, it works.Item) string {
	i := slices.IndexFunc(d.Fields, func(f works.FieldSpec) bool { return slices.Contains(f.In, works.PlaceCode) })
	if i < 0 {
		return ""
	}
	for _, v := range it.Fields {
		if v.Key == d.Fields[i].Key && v.Kind == works.KindText {
			return v.Text
		}
	}
	return ""
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
	scored   targets.Scored
}

// assess reads l against j. It is pure and deterministic.
func assess(j judging, l listed) assessed {
	return assessRead(j, l, release.Title(l.Release.Title, j.names, j.cuts))
}

// assessRead is assess of l read as rd.
func assessRead(j judging, l listed, rd release.Read) assessed {
	return assessMatched(j, l, rd, j.scoring.Match)
}

// assessStored is assess of a Release a Manual run stored, read as rd
// and matching matched when it was heard, so reading it back never parses
// or matches again.
func assessStored(j judging, l listed, rd release.Read, matched formats.Set) assessed {
	return assessMatched(j, l, rd, func(release.Read) formats.Result { return formats.Result{Matched: matched} })
}

func assessMatched(j judging, l listed, rd release.Read, match func(release.Read) formats.Result) assessed {
	r := l.Release
	a := assessed{listed: l, read: rd}
	a.covers = j.expect.Covers(a.read.Name)
	f := &a.read.Facts
	if r.Size > 0 {
		f.Size = release.Known(r.Size)
	}
	f.Flags = r.Flags
	f.Coverage = release.Coverage{Runtime: j.runtimeOf(a.covers), Original: j.expect.Original}
	a.refusals = j.refusals(a)
	if !a.hard() {
		a.formats = match(a.read)
	}
	a.scored = targets.Scored{Facts: a.read.Facts, Formats: a.formats.Matched}
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
// none of the Work's Items or none of the scope, a pack listing an Item
// that hasn't aired, and a Release a failed Grab of the Work tried. Its
// ids are the Indexer's, over the id tags its name carries.
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
	if len(a.covers) == 0 || j.scope != nil && !slices.ContainsFunc(a.covers, func(it works.ItemID) bool { return j.scope[it] }) {
		out = append(out, own(targets.WrongItem, marks(a.read.Name)))
	}
	if len(a.covers) > 1 {
		if i := slices.IndexFunc(a.covers, func(it works.ItemID) bool { _, ok := j.unaired[it]; return ok }); i >= 0 {
			out = append(out, own(targets.NotAired, j.unaired[a.covers[i]]))
		}
	}
	if slices.ContainsFunc(j.tried, a.tried) {
		out = append(out, own(targets.Tried, ""))
	}
	return out
}

// tried reports whether t is a's Release or one of its copies.
func (a assessed) tried(t triedRelease) bool {
	return t.is(a.Release) || slices.ContainsFunc(a.copies, func(c copied) bool { return t.is(c.Release) })
}

func own(code targets.ReasonCode, value string) Reason {
	return Reason{Reason: targets.Reason{Code: code, Value: value}}
}

// coversItem reports whether a's name covers cell's Item.
func (a assessed) coversItem(cell files.Cell) bool { return slices.Contains(a.covers, cell.Item.ID) }

// hard reports a hard refusal of a's own, which stops judging.
func (a assessed) hard() bool {
	return slices.ContainsFunc(a.refusals, func(r Reason) bool { return r.Code.Hard() })
}

// namesTheWork reports that no id a's name or its Indexer gives names
// another Work and its title and year match the Work.
func (a assessed) namesTheWork() bool {
	return !slices.ContainsFunc(a.refusals, func(r Reason) bool { return r.Code == targets.WrongWork || r.Code == targets.WrongTitle })
}

// blocked reports blocklisted among a's own refusals.
func (a assessed) blocked() bool {
	return slices.ContainsFunc(a.refusals, func(r Reason) bool { return r.Code == targets.Blocklisted })
}

// hashes is the info hash of a's Release and of each of its copies, of
// those listed with one.
func (a assessed) hashes() []bencode.InfoHash {
	var out []bencode.InfoHash
	if h := a.Release.InfoHash; h != nil {
		out = append(out, *h)
	}
	for _, c := range a.copies {
		if h := c.Release.InfoHash; h != nil {
			out = append(out, *h)
		}
	}
	return out
}

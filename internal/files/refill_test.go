package files

import (
	"encoding/json"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// marks are what a random file's release name may carry, each the one
// title pattern of a format in marked.
var marks = []string{"PROPER", "HDR", "Atmos", "GRP"}

var marked, markedIDs = markedFormats()

func markedFormats() (*formats.Catalog, []formats.ID) {
	fs := make([]formats.Format, len(marks))
	ids := make([]formats.ID, len(marks))
	for i, m := range marks {
		value, err := json.Marshal(map[string]string{"value": `\b` + m + `\b`})
		if err != nil {
			panic(err)
		}
		ids[i] = formats.ID(storage.NewUUID().Bytes())
		fs[i] = formats.Format{ID: ids[i], Name: m, Dialect: formats.SlipStream, Conditions: []formats.Condition{{Name: m, Kind: formats.ReleaseTitle, Fields: value}}}
	}
	// No match limit: a starved runner can hold one match past any wall
	// clock limit, and a timed-out match scores the file without that
	// format, so refill and checkBest would disagree on the same file.
	return formats.Compile(fs, formats.Timings{}), ids
}

// randomWorld is a Work of up to three Items wanting from up to four
// random Targets, holding up to six files of one or two Items, random
// qualities and release names that fill cells at random, mostly their own
// Items' and whether or not they meet them, some of the fills forced. Some
// files arrive, fresh from an Import or found, the others landed by that
// Import or none, and a fresh file may force a cell.
func randomWorld(t *testing.T, r *rand.Rand) world {
	t.Helper()
	w := world{fills: map[CellKey]Fill{}, fresh: map[ID]bool{}, found: map[ID]bool{}, imports: map[ID]storage.UUID{}, force: map[CellKey]ID{}, forcedAt: storage.Now()}
	for range 1 + r.IntN(3) {
		w.tree.Items = append(w.tree.Items, works.Item{ID: works.ItemID{UUID: storage.NewUUID()}})
	}
	for i := range 1 + r.IntN(4) {
		w.goals.Targets = append(w.goals.Targets, randomTarget(t, r, string(rune('A'+i))))
	}
	imports := []storage.UUID{{}, storage.NewUUID()}
	w.importing = imports[r.IntN(len(imports))]
	for range r.IntN(7) {
		f := randomFile(r, w.tree.Items)
		w.files = append(w.files, f)
		switch r.IntN(4) {
		case 0:
			w.fresh[f.ID] = true
		case 1:
			w.found[f.ID] = true
		default:
			w.imports[f.ID] = imports[r.IntN(len(imports))]
		}
	}
	for _, it := range w.tree.Items {
		for _, target := range w.goals.Targets {
			randomFill(r, w, CellKey{Item: it.ID, Edition: targets.AnyEdition, Target: target.ID})
		}
	}
	w.scoring = newScoring(marked, w.goals, release.Fact[release.Language]{})
	return w
}

// randomFile is a file of one of items, or of it and the next, of a
// random quality and release name.
func randomFile(r *rand.Rand, items []works.Item) File {
	all := release.Qualities()
	name := marking(r, "Scarf.1080p", marks)
	first := r.IntN(len(items))
	of := []works.ItemID{items[first].ID}
	if first+1 < len(items) && r.IntN(3) == 0 {
		of = append(of, items[first+1].ID)
	}
	return File{
		ID: ID{UUID: storage.NewUUID()}, Items: of, Cut: release.Standard,
		Quality: all[r.IntN(len(all))], ReleaseName: name, Title: release.Title(name, release.TitleOnly{}, nil),
	}
}

// randomFill fills key in w at random, mostly from a file holding its
// Item and half the time forced, and may have a fresh file force it.
func randomFill(r *rand.Rand, w world, key CellKey) {
	holding := slices.DeleteFunc(slices.Clone(w.files), func(f File) bool { return !slices.Contains(f.Items, key.Item) })
	if len(holding) == 0 || r.IntN(4) == 0 {
		holding = w.files
	}
	if len(holding) > 0 && r.IntN(2) == 0 {
		fill := Fill{File: holding[r.IntN(len(holding))].ID}
		if r.IntN(2) == 0 {
			fill.Forced = storage.Now()
		}
		w.fills[key] = fill
	}
	if len(w.files) > 0 && r.IntN(5) == 0 {
		if f := w.files[r.IntN(len(w.files))]; w.fresh[f.ID] {
			w.force[key] = f.ID
		}
	}
}

// randomTarget is a Target of random tiers that scores the marked formats
// at random and takes propers by a random setting.
func randomTarget(t *testing.T, r *rand.Rand, name string) targets.Target {
	t.Helper()
	picked := slices.Clone(release.Qualities())
	r.Shuffle(len(picked), func(i, j int) { picked[i], picked[j] = picked[j], picked[i] })
	picked = picked[:1+r.IntN(6)]
	var tiers [][]release.Quality
	for len(picked) > 0 {
		n := 1 + r.IntN(len(picked))
		tiers, picked = append(tiers, picked[:n]), picked[n:]
	}
	scores := targets.Scores{Formats: map[formats.ID]int{}, UpgradeUntil: r.IntN(10), Increment: 1}
	for _, id := range markedIDs {
		if v := r.IntN(21) - 10; v != 0 && r.IntN(2) == 0 {
			scores.Formats[id] = v
		}
	}
	propers := []targets.Propers{targets.PreferAndUpgrade, targets.PreferNoUpgrade, targets.DontPrefer}
	target, err := targets.NewTarget(targets.Spec{Scores: scores, Propers: propers[r.IntN(3)], Name: name, Tiers: tiers, Cutoff: tiers[r.IntN(len(tiers))][0]})
	if err != nil {
		t.Fatal(err)
	}
	target.ID = targets.TargetID{UUID: storage.NewUUID()}
	return target
}

// marking is name with each of words added at random.
func marking(r *rand.Rand, name string, words []string) string {
	parts := []string{name}
	for _, w := range words {
		if r.IntN(2) == 0 {
			parts = append(parts, w)
		}
	}
	return strings.Join(parts, ".")
}

func TestRefillingARefilledWorkChangesNothing(t *testing.T) {
	r := rand.New(rand.NewPCG(17, 6)) //nolint:gosec // seeded on purpose so failures replay
	for i := range 2000 {
		w := randomWorld(t, r)
		was := w.fills
		once := refill(w)
		w.fills = once
		if twice := refill(w); !maps.Equal(once, twice) {
			t.Fatalf("world %d: refill = %v, refill of that = %v", i, once, twice)
		}
		checkBest(t, w, was, once)
	}
}

// checkBest checks, of a refill from was to fills, every fill meets its
// Target and holds its file's score there, and that a cell some file meets
// is filled. A cell a fresh file forces goes to it. A fill that was forced
// keeps its file unless the cell went to a better file arriving from
// another Import. Any other fill has no better file of its Item meeting
// its Target, by the Target's order.
func checkBest(t *testing.T, w world, was, fills map[CellKey]Fill) {
	t.Helper()
	for _, it := range w.tree.Items {
		for _, target := range w.goals.Targets {
			key := CellKey{Item: it.ID, Edition: targets.AnyEdition, Target: target.ID}
			meeting := meetingAt(w, it.ID, target)
			fill, filled := fills[key]
			if filled != (len(meeting) > 0) {
				t.Fatalf("cell %s filled %t, want %t", target.Name, filled, len(meeting) > 0)
			}
			if filled {
				checkCell(t, w, target, meeting, was[key], fill, w.force[key])
			}
		}
	}
}

// checkCell checks fill, of a cell of target that meeting meet, against
// prior, its fill before, and forcing, the file forcing it.
func checkCell(t *testing.T, w world, target targets.Target, meeting []File, prior, fill Fill, forcing ID) {
	t.Helper()
	meets := func(id ID) bool { return slices.ContainsFunc(meeting, func(m File) bool { return m.ID == id }) }
	switch by := w.byID()[fill.File]; {
	case meets(forcing):
		if fill.File != forcing || fill.Forced != w.forcedAt {
			t.Fatalf("cell %s holds %+v, want the file forcing it", target.Name, fill)
		}
		checkScore(t, target, by, fill.Score)
	case !prior.Forced.IsZero() && meets(prior.File) && fill.File == prior.File:
		if fill.Forced != prior.Forced {
			t.Fatalf("cell %s keeps its forced file as %+v, want it still forced at %v", target.Name, fill, prior.Forced)
		}
		if top, ok := bestOf(target, meeting); ok && challenger(w, top, prior.File) && better(target, top, *by) {
			t.Fatalf("cell %s keeps its forced %s, but %s arriving from another Import beats every file", target.Name, by.ReleaseName, top.ReleaseName)
		}
		checkScore(t, target, by, fill.Score)
	case !prior.Forced.IsZero() && meets(prior.File):
		if !challenger(w, *by, prior.File) {
			t.Fatalf("cell %s forced to %s went to %s, which isn't arriving from another Import", target.Name, prior.File, by.ReleaseName)
		}
		checkFill(t, target, meeting, by, fill.Score)
	default:
		checkFill(t, target, meeting, by, fill.Score)
	}
}

// challenger reports whether f arrives in w from another Import than
// held's.
func challenger(w world, f File, held ID) bool {
	return (w.fresh[f.ID] || w.found[f.ID]) && (w.importOf(f.ID).IsZero() || w.importOf(f.ID) != w.importOf(held))
}

// bestOf is the one file of meeting no other matches or beats at target,
// false when two tie at the top.
func bestOf(target targets.Target, meeting []File) (File, bool) {
	var tops []File
	for _, f := range meeting {
		if !slices.ContainsFunc(meeting, func(o File) bool { return o.ID != f.ID && !better(target, f, o) }) {
			tops = append(tops, f)
		}
	}
	if len(tops) != 1 {
		return File{}, false
	}
	return tops[0], true
}

// better reports whether a is better than b at target.
func better(target targets.Target, a, b File) bool {
	sa, sb := scoredOf(a), scoredOf(b)
	_, c := target.Compare(&sa, &sb)
	return c > 0
}

// checkScore checks f fills a cell of target at its score there.
func checkScore(t *testing.T, target targets.Target, f *File, score int) {
	t.Helper()
	held := scoredOf(*f)
	if want := target.Score(&held); score != want {
		t.Fatalf("cell %s holds %s at score %d, want %d", target.Name, f.ReleaseName, score, want)
	}
}

// checkFill checks f, filling a cell of target at score, is among the
// files meeting it, that none of them is better, and that score is f's.
func checkFill(t *testing.T, target targets.Target, meeting []File, f *File, score int) {
	t.Helper()
	if !slices.ContainsFunc(meeting, func(m File) bool { return m.ID == f.ID }) {
		t.Fatalf("cell %s holds %s, which doesn't meet it", target.Name, f.ReleaseName)
	}
	held := scoredOf(*f)
	for _, m := range meeting {
		other := scoredOf(m)
		if step, better := target.Compare(&other, &held); better > 0 {
			t.Fatalf("cell %s holds %s (%s), but %s (%s) is better by step %d", target.Name, f.ReleaseName, f.Quality, m.ReleaseName, m.Quality, step)
		}
	}
	if want := target.Score(&held); score != want {
		t.Fatalf("cell %s holds %s at score %d, want %d", target.Name, f.ReleaseName, score, want)
	}
}

// scoredOf is f with every marked format its release name matches.
func scoredOf(f File) targets.Scored {
	return targets.Scored{Facts: f.Facts(), Formats: marked.Match(asNamed(f.Title, f.ReleaseName, release.Fact[release.Language]{})).Matched}
}

// meetingAt is item's files that meet target.
func meetingAt(w world, item works.ItemID, target targets.Target) []File {
	var out []File
	for _, f := range w.files {
		if slices.Contains(f.Items, item) && targets.Judge(target, targets.Scored{Facts: f.Facts()}, nil).Meets() {
			out = append(out, f)
		}
	}
	return out
}

func TestATieKeepsTheIncumbentElseTheOldestFile(t *testing.T) {
	item := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}}
	hd, err := targets.NewTarget(targets.Spec{
		Scores: targets.Scores{Increment: 1},
		Name:   "HD", Tiers: [][]release.Quality{{release.Bluray1080p, release.Remux1080p}, {release.WEBDL1080p, release.WEBRip1080p}}, Cutoff: release.Bluray1080p,
	})
	if err != nil {
		t.Fatal(err)
	}
	hd.ID = targets.TargetID{UUID: storage.NewUUID()}
	key := CellKey{Item: item.ID, Edition: targets.AnyEdition, Target: hd.ID}
	file := func(q release.Quality) File {
		return File{ID: ID{UUID: storage.NewUUID()}, Items: []works.ItemID{item.ID}, Cut: release.Standard, Quality: q}
	}
	older, newer, better := file(release.WEBDL1080p), file(release.WEBRip1080p), file(release.Remux1080p)
	at := func(fills map[CellKey]Fill, fs ...File) ID {
		goals := targets.Goals{Targets: []targets.Target{hd}}
		w := world{tree: works.Bare{Items: []works.Item{item}}, goals: goals, files: fs, fills: fills, scoring: newScoring(marked, goals, release.Fact[release.Language]{})}
		return refill(w)[key].File
	}
	if got := at(map[CellKey]Fill{}, older, newer); got != older.ID {
		t.Errorf("an empty cell went to %v, want the older of two tied files %v", got, older.ID)
	}
	if got := at(map[CellKey]Fill{key: {File: newer.ID}}, older, newer); got != newer.ID {
		t.Errorf("a cell newer fills went to %v, want newer keeping it on a tie", got)
	}
	if got := at(map[CellKey]Fill{key: {File: newer.ID}}, older, newer, better); got != better.ID {
		t.Errorf("a cell newer fills went to %v, want the better-ranked Remux %v", got, better.ID)
	}
	gone := file(release.Bluray1080p)
	if got := at(map[CellKey]Fill{key: {File: gone.ID}}, older, newer); got != older.ID {
		t.Errorf("a cell whose incumbent left went to %v, want the oldest of the rest", got)
	}
}

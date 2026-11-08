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
	return formats.Compile(fs, formats.DefaultTimings()), ids
}

// randomWorld is a Work of up to three Items wanting from up to four
// random Targets, holding up to six files of random qualities and release
// names that fill cells at random, whether or not they meet them.
func randomWorld(t *testing.T, r *rand.Rand) world {
	t.Helper()
	w := world{fills: map[CellKey]Fill{}}
	for range 1 + r.IntN(3) {
		w.tree.Items = append(w.tree.Items, works.Item{ID: works.ItemID{UUID: storage.NewUUID()}})
	}
	for i := range 1 + r.IntN(4) {
		w.goals.Targets = append(w.goals.Targets, randomTarget(t, r, string(rune('A'+i))))
	}
	all := release.Qualities()
	for range r.IntN(7) {
		name := marking(r, "Scarf.1080p", marks)
		w.files = append(w.files, File{
			ID: ID{UUID: storage.NewUUID()}, ItemID: w.tree.Items[r.IntN(len(w.tree.Items))].ID, Cut: release.Standard,
			Quality: all[r.IntN(len(all))], ReleaseName: name, Title: release.Title(name, release.TitleOnly{}, nil),
		})
	}
	for _, it := range w.tree.Items {
		for _, target := range w.goals.Targets {
			if len(w.files) > 0 && r.IntN(2) == 0 {
				w.fills[CellKey{Item: it.ID, Edition: targets.AnyEdition, Target: target.ID}] = Fill{File: w.files[r.IntN(len(w.files))].ID}
			}
		}
	}
	w.scoring = newScoring(marked, w.goals, release.Fact[release.Language]{})
	return w
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
		once := refill(w)
		w.fills = once
		if twice := refill(w); !maps.Equal(once, twice) {
			t.Fatalf("world %d: refill = %v, refill of that = %v", i, once, twice)
		}
		checkBest(t, w, once)
	}
}

// checkBest checks every fill meets its Target, that no file of its Item
// that meets it is better by the Target's order, and that it holds its
// file's score there; and that a cell some file meets is filled.
func checkBest(t *testing.T, w world, fills map[CellKey]Fill) {
	t.Helper()
	for _, it := range w.tree.Items {
		for _, target := range w.goals.Targets {
			meeting := meetingAt(w, it.ID, target)
			fill, filled := fills[CellKey{Item: it.ID, Edition: targets.AnyEdition, Target: target.ID}]
			if filled != (len(meeting) > 0) {
				t.Fatalf("cell %s filled %t, want %t", target.Name, filled, len(meeting) > 0)
			}
			if filled {
				checkFill(t, target, meeting, w.byID()[fill.File], fill.Score)
			}
		}
	}
}

// checkFill checks f, filling a cell of target at score, is among the
// files meeting it, that none of them is better, and that score is f's.
func checkFill(t *testing.T, target targets.Target, meeting []File, f *File, score int) {
	t.Helper()
	if !slices.ContainsFunc(meeting, func(m File) bool { return m.ID == f.ID }) {
		t.Fatalf("cell %s holds %s, which doesn't meet it", target.Name, f.ReleaseName)
	}
	for _, m := range meeting {
		if step, better := target.Compare(scoredOf(m), scoredOf(*f)); better > 0 {
			t.Fatalf("cell %s holds %s (%s), but %s (%s) is better by step %d", target.Name, f.ReleaseName, f.Quality, m.ReleaseName, m.Quality, step)
		}
	}
	if want := target.Score(scoredOf(*f)); score != want {
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
		if f.ItemID == item && targets.Judge(target, targets.Scored{Facts: f.Facts()}, nil).Meets() {
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
		return File{ID: ID{UUID: storage.NewUUID()}, ItemID: item.ID, Cut: release.Standard, Quality: q}
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

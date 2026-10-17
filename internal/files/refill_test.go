package files

import (
	"maps"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// randomWorld is a Work of up to three Items wanting from up to four
// random Targets, holding up to six files of random qualities that fill
// cells at random, whether or not they meet them.
func randomWorld(t *testing.T, r *rand.Rand) world {
	t.Helper()
	w := world{fills: map[CellKey]ID{}}
	for range 1 + r.IntN(3) {
		w.tree.Items = append(w.tree.Items, works.Item{ID: works.ItemID{UUID: storage.NewUUID()}})
	}
	all := release.Qualities()
	for i := range 1 + r.IntN(4) {
		picked := slices.Clone(all)
		r.Shuffle(len(picked), func(i, j int) { picked[i], picked[j] = picked[j], picked[i] })
		picked = picked[:1+r.IntN(6)]
		var tiers [][]release.Quality
		for len(picked) > 0 {
			n := 1 + r.IntN(len(picked))
			tiers, picked = append(tiers, picked[:n]), picked[n:]
		}
		target, err := targets.NewTarget(targets.Spec{Name: string(rune('A' + i)), Tiers: tiers, Cutoff: tiers[r.IntN(len(tiers))][0]})
		if err != nil {
			t.Fatal(err)
		}
		target.ID = targets.TargetID{UUID: storage.NewUUID()}
		w.goals.Targets = append(w.goals.Targets, target)
	}
	for range r.IntN(7) {
		w.files = append(w.files, File{
			ID: ID{UUID: storage.NewUUID()}, ItemID: w.tree.Items[r.IntN(len(w.tree.Items))].ID, Edition: Standard,
			Quality: all[r.IntN(len(all))],
		})
	}
	for _, it := range w.tree.Items {
		for _, target := range w.goals.Targets {
			if len(w.files) > 0 && r.IntN(2) == 0 {
				w.fills[CellKey{Item: it.ID, Edition: Standard, Target: target.ID}] = w.files[r.IntN(len(w.files))].ID
			}
		}
	}
	return w
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

// checkBest checks every fill meets its Target and ranks best there among
// its Item's files, and that a cell some file meets is filled.
func checkBest(t *testing.T, w world, fills map[CellKey]ID) {
	t.Helper()
	files := w.byID()
	for _, it := range w.tree.Items {
		for _, target := range w.goals.Targets {
			best, met := bestRank(w, it.ID, target)
			id, filled := fills[CellKey{Item: it.ID, Edition: Standard, Target: target.ID}]
			if filled != met {
				t.Fatalf("cell %s filled %t, want %t", target.Name, filled, met)
			}
			if !filled {
				continue
			}
			f := files[id]
			if r, ok := target.RankOf(f.Quality); f.ItemID != it.ID || !ok || r != best {
				t.Fatalf("cell %s holds a %s file of rank %d, want one of rank %d of the Item", target.Name, f.Quality, r, best)
			}
		}
	}
}

// bestRank is the best rank at target of item's files that meet it, and
// whether any does.
func bestRank(w world, item works.ItemID, target targets.Target) (best targets.Rank, met bool) {
	for _, f := range w.files {
		if v := targets.Judge(target, f.Facts(), nil); f.ItemID == item && v.Meets() {
			r, _ := v.Rank()
			if !met || r < best {
				best, met = r, true
			}
		}
	}
	return best, met
}

func TestATieKeepsTheIncumbentElseTheOldestFile(t *testing.T) {
	item := works.Item{ID: works.ItemID{UUID: storage.NewUUID()}}
	hd, err := targets.NewTarget(targets.Spec{
		Name: "HD", Tiers: [][]release.Quality{{release.Bluray1080p, release.Remux1080p}, {release.WEBDL1080p, release.WEBRip1080p}}, Cutoff: release.Bluray1080p,
	})
	if err != nil {
		t.Fatal(err)
	}
	hd.ID = targets.TargetID{UUID: storage.NewUUID()}
	key := CellKey{Item: item.ID, Edition: Standard, Target: hd.ID}
	file := func(q release.Quality) File {
		return File{ID: ID{UUID: storage.NewUUID()}, ItemID: item.ID, Edition: Standard, Quality: q}
	}
	older, newer, better := file(release.WEBDL1080p), file(release.WEBRip1080p), file(release.Remux1080p)
	at := func(fills map[CellKey]ID, fs ...File) ID {
		w := world{tree: works.Tree{Items: []works.Item{item}}, goals: targets.Goals{Targets: []targets.Target{hd}}, files: fs, fills: fills}
		return refill(w)[key]
	}
	if got := at(map[CellKey]ID{}, older, newer); got != older.ID {
		t.Errorf("an empty cell went to %v, want the older of two tied files %v", got, older.ID)
	}
	if got := at(map[CellKey]ID{key: newer.ID}, older, newer); got != newer.ID {
		t.Errorf("a cell newer fills went to %v, want newer keeping it on a tie", got)
	}
	if got := at(map[CellKey]ID{key: newer.ID}, older, newer, better); got != better.ID {
		t.Errorf("a cell newer fills went to %v, want the better-ranked Remux %v", got, better.ID)
	}
	gone := file(release.Bluray1080p)
	if got := at(map[CellKey]ID{key: gone.ID}, older, newer); got != older.ID {
		t.Errorf("a cell whose incumbent left went to %v, want the oldest of the rest", got)
	}
}

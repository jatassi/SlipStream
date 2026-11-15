package grabs_test

import (
	"context"
	"maps"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// TestAGroupRunGrabsAPackAndTheSingleItLeavesThenALaterRunJoinsThePack
// decides a run of a ten-episode season whose fifth episode isn't
// monitored, over a WEB-DL pack of E01 to E09, a BluRay of E01 and a
// WEB-DL of E10: the pack fills the eight wanted episodes it covers over
// the better single, and E10's single fills the last. Once E05 is
// monitored, a run of E05 alone that hears the pack again joins its
// Grab rather than making another.
func TestAGroupRunGrabsAPackAndTheSingleItLeavesThenALaterRunJoinsThePack(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()
		w := grabstest.New(t, grabstest.Options{
			Sites: []string{"Quick Site"}, Season: 10,
			Grabs: grabs.DefaultTimings(), Indexers: indexers.DefaultTimings(), Formats: formats.DefaultTimings(),
		})
		pack := heard(w, grabstest.Pack(1, 9), "1080p.WEB-DL", 'p')
		monitor(t, w, false)
		first, err := w.Grabs.DecideRun(ctx, w.Work, w.Items, []indexers.Release{pack, heard(w, grabstest.Pack(1, 1), "1080p.BluRay", '1'), heard(w, grabstest.Pack(10, 10), "1080p.WEB-DL", 'z')})
		if err != nil {
			t.Fatal(err)
		}
		got, fills := grabbed(t, w)
		want := map[string][]int{pack.Title: {1, 2, 3, 4, 6, 7, 8, 9}, "The.Attic.2020.E10.1080p.WEB-DL-GRP": {10}}
		if !maps.EqualFunc(got, want, slices.Equal[[]int]) {
			t.Fatalf("the run grabbed %v, want the pack for the eight wanted episodes it covers and E10's single", got)
		}
		if fills[pack.Title] != 8 || fills["The.Attic.2020.E10.1080p.WEB-DL-GRP"] != 0 {
			t.Errorf("the Grabs were chosen to fill %v, want the pack 8 by the fewer-Releases step and the single none", fills)
		}
		monitor(t, w, true)
		second, err := w.Grabs.DecideRun(ctx, w.Work, w.Items[4:5], []indexers.Release{pack})
		if err != nil {
			t.Fatal(err)
		}
		want[pack.Title] = []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
		if got, _ := grabbed(t, w); !maps.EqualFunc(got, want, slices.Equal[[]int]) {
			t.Fatalf("after the second run the Grabs hold %v, want the same two, the pack's holding E05 too", got)
		}
		runs, err := w.Grabs.Searches(ctx, identity.Admin{}, w.Work)
		if err != nil {
			t.Fatal(err)
		}
		took := map[grabs.SearchID]int{}
		for _, r := range runs {
			took[r.ID] = len(r.Grabs)
		}
		if took[first] != 2 || took[second] != 1 {
			t.Errorf("the runs took %d and %d Grabs, want 2, then the pack's", took[first], took[second])
		}
		if made := madeOf(t, w, second); !slices.Equal(made, []bool{false}) {
			t.Errorf("the second run's links to its Grabs are made %v, want one join", made)
		}
	})
}

// monitor turns monitoring of the World's fifth episode on or off.
func monitor(t *testing.T, w *grabstest.World, on bool) {
	t.Helper()
	if _, err := w.Config.Works.SetMonitored(t.Context(), identity.Admin{}, w.Work, works.Within{Items: w.Items[4:5]}, on); err != nil {
		t.Fatal(err)
	}
}

// heard is a Release of The Attic that Quick Site lists: the episodes
// mark names, at quality, with an info hash of hash.
func heard(w *grabstest.World, mark, quality string, hash byte) indexers.Release {
	h := bencode.InfoHash([]byte(strings.Repeat(string(hash), 20)))
	return indexers.Release{Indexer: w.Sites["Quick Site"], Release: searching.Release{
		Title: "The.Attic.2020." + mark + "." + quality + "-GRP", Size: 4 << 30, Seeders: 10, InfoHash: &h,
		Link: searching.LinkOf("https://site.test/dl/" + mark),
	}}
}

// episodes is the episodes, from 1, whose cells g holds.
func episodes(w *grabstest.World, g grabs.Grab) []int {
	var out []int
	for _, c := range g.Targets {
		out = append(out, slices.Index(w.Items, c.Key.Item)+1)
	}
	slices.Sort(out)
	return out
}

func madeOf(t *testing.T, w *grabstest.World, run grabs.SearchID) []bool {
	t.Helper()
	var made []bool
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := db.New(r).ListSearchGrabs(ctx, []db.SearchID{run})
		for _, row := range rows {
			made = append(made, row.Made)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

// grabbed is each Grab's Release title, with the episodes it holds and
// the Items it was chosen to fill.
func grabbed(t *testing.T, w *grabstest.World) (held map[string][]int, fills map[string]int) {
	t.Helper()
	views, err := w.Grabs.List(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	held, fills = map[string][]int{}, map[string]int{}
	for _, v := range views {
		held[v.Release.Title], fills[v.Release.Title] = episodes(w, v.Grab), v.ChosenFills
	}
	return held, fills
}

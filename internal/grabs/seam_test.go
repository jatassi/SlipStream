package grabs_test

import (
	"context"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestReleasedTwiceQueuesOneRowPerGroupInBacklogWithTheUnionScope(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Levels: []grabstest.Level{
			{Label: "Winter", Items: []string{"Scarf", "Mitten"}}, {Label: "Spring", Items: []string{"Kite"}},
		}})
		scarf, mitten, kite := w.Items[0], w.Items[1], w.Items[2]
		w.Reconciling(true)
		for _, items := range [][]works.ItemID{{scarf}, {mitten, kite}, {scarf}} {
			if err := w.Grabs.Released(t.Context(), items); err != nil {
				t.Fatal(err)
			}
		}
		w.Run()
		want := map[indexers.GroupKey][]works.ItemID{
			indexers.LevelGroup(w.Levels[0]): sortedItems([]works.ItemID{scarf, mitten}), indexers.LevelGroup(w.Levels[1]): {kite},
		}
		q := w.Queue(t)
		if len(q.Nudged) != 0 || len(q.Due) != len(want) {
			t.Fatalf("the queue holds %+v, want one row per group", q)
		}
		for _, d := range q.Due {
			if d.Lane != pace.Backlog || d.Trigger != searches.ReleaseDate || !slices.Equal(sortedItems(d.Items), want[d.Group]) {
				t.Errorf("group %s is queued %+v, want release_date in backlog with the Items released in it, %v", d.Group, d, want[d.Group])
			}
		}
	})
}

func TestAProfileChangeOfOneWorkQueuesNowAndOfSeveralBacklog(t *testing.T) {
	for _, tc := range []struct {
		name  string
		works int
		lane  pace.Lane
	}{
		{"one Work", 1, pace.Now},
		{"several Works", 2, pace.Backlog},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
				ws := []works.WorkID{w.Work}
				if tc.works > 1 {
					ws = append(ws, w.AddWork())
				}
				w.Reconciling(true)
				change(t, w, targets.SetProfiles{Works: ws, Profile: otherProfile(t, w)})
				w.Run()
				q := w.Queue(t)
				if len(q.Due) != len(ws) {
					t.Fatalf("the queue holds %+v, want one row per Work changed", q)
				}
				for _, d := range q.Due {
					if d.Lane != tc.lane || d.Trigger != searches.Profile {
						t.Errorf("Work %s is queued %+v, want profile in lane %d", d.Work, d, tc.lane)
					}
				}
			})
		})
	}
}

func TestATargetEditWithImpactResetsTheGapAndQueuesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, BacklogEvery: 2 * time.Hour})
		steady(t, w)
		group := indexers.LevelGroup(w.Level)
		w.Run()
		enqueue(t, w, searches.Backlog)
		if gap := gapOf(t, w, group); gap != 4*time.Hour {
			t.Fatalf("after a run that found nothing the next search is %v later, want 4 hours", gap)
		}
		listed, err := w.Config.Targets.ListTargets(t.Context(), identity.Admin{})
		if err != nil {
			t.Fatal(err)
		}
		hd := listed[slices.IndexFunc(listed, func(g targets.Target) bool { return g.Name == "HD" })]
		spec := grabstest.HD[0]
		spec.Tiers = append(spec.Tiers, []rel.Quality{rel.WEBRip1080p})
		change(t, w, targets.UpdateTarget{ID: hd.ID, Spec: spec})
		if gap := gapOf(t, w, group); gap != 2*time.Hour {
			t.Fatalf("after the Target edit the next search is %v after the last, want Backlog search's 2 hours", gap)
		}
		if q := w.Queue(t); len(q.Nudged) != 0 || len(q.Runs) != 0 || len(q.Due) != 1 || q.Due[0].Lane != 0 {
			t.Fatalf("after the Target edit the queue holds %+v, want its row with nothing queued", q)
		}
	})
}

func TestAnAddQueuedAfterAPressCantPutAHoldOnIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		press(t, w, grabs.Scope{})
		err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return w.Grabs.Queue(ctx, tx, searches.Add, searches.Ask{Work: w.Work})
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Reconciling(true)
		w.Run()
		q := w.Queue(t)
		if len(q.Nudged) != 0 || len(q.Due) != 1 || q.Due[0].Trigger != searches.Search || q.Due[0].Lane != pace.Now || !q.Due[0].Gone {
			t.Fatalf("after the add's drain the queue holds %+v, want the press's row, whose trigger sets no Hold", q)
		}
	})
}

func change(t *testing.T, w *grabstest.World, c targets.Change) {
	t.Helper()
	if _, err := w.Config.Files.Change(t.Context(), identity.Admin{}, c, files.Mode{}); err != nil {
		t.Fatal(err)
	}
}

// otherProfile is a Profile beside the World's, with a Target of its own.
func otherProfile(t *testing.T, w *grabstest.World) targets.ProfileID {
	t.Helper()
	ctx := t.Context()
	spec := grabstest.HD[0]
	spec.Name = "Other HD"
	made, err := w.Config.Targets.CreateTarget(ctx, identity.Admin{}, spec)
	if err != nil {
		t.Fatal(err)
	}
	p, err := w.Config.Targets.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{
		Name: "Other", ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: made.ID, Sourcing: targets.Download}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

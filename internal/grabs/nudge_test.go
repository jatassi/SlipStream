package grabs_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func nudge(t *testing.T, w *grabstest.World, work works.WorkID, trigger searches.Trigger, items ...works.ItemID) {
	t.Helper()
	if err := w.Grabs.Nudge(t.Context(), work, trigger, items); err != nil {
		t.Fatal(err)
	}
}

func sortedItems(items []works.ItemID) []works.ItemID {
	out := slices.Clone(items)
	slices.SortFunc(out, func(a, b works.ItemID) int { return a.Compare(b.UUID) })
	return out
}

func TestTwoNudgesForOneWorkDrainIntoOneRowPerGroup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Levels: []grabstest.Level{
			{Label: "Winter", Items: []string{"Scarf", "Mitten"}}, {Label: "Spring", Items: []string{"Kite"}},
		}})
		scarf, mitten, kite := w.Items[0], w.Items[1], w.Items[2]
		w.Reconciling(true)
		nudge(t, w, w.Work, searches.ReleaseDate, scarf)
		nudge(t, w, w.Work, searches.ReleaseDate, mitten, kite)
		if q := w.Queue(t); !slices.Equal(q.Nudged, []works.WorkID{w.Work, w.Work}) || len(q.Due) != 0 {
			t.Fatalf("before a drain the queue holds %+v, want the two nudges and no row", q)
		}
		w.Run()
		q := w.Queue(t)
		want := map[indexers.GroupKey][]works.ItemID{
			indexers.LevelGroup(w.Levels[0]): sortedItems([]works.ItemID{scarf, mitten}), indexers.LevelGroup(w.Levels[1]): {kite},
		}
		if len(q.Nudged) != 0 || len(q.Due) != len(want) {
			t.Fatalf("after the drain the queue holds %+v, want no nudge and one row per group", q)
		}
		for _, d := range q.Due {
			if d.Lane != pace.Backlog || d.Trigger != searches.ReleaseDate || !slices.Equal(sortedItems(d.Items), want[d.Group]) {
				t.Errorf("group %s is queued %+v, want release_date in backlog with the Items both nudges asked of it, %v", d.Group, d, want[d.Group])
			}
		}
	})
}

func TestANudgeForADeletedWorkCascades(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		other := w.AddWork()
		w.Reconciling(true)
		nudge(t, w, other, searches.Refresh)
		nudge(t, w, w.Work, searches.Refresh)
		present := func(_ context.Context, _ *storage.Tx[storage.Library], ws []works.WorkID) ([]works.WorkID, error) {
			return ws, nil
		}
		if _, err := w.Config.Works.Delete(t.Context(), identity.Admin{}, []works.WorkID{other}, present); err != nil {
			t.Fatal(err)
		}
		if q := w.Queue(t); !slices.Equal(q.Nudged, []works.WorkID{w.Work}) {
			t.Fatalf("after the Work's delete the nudges are of %v, want only the World's Work's", q.Nudged)
		}
		w.Run()
		if q := w.Queue(t); len(q.Nudged) != 0 || len(q.Due) != 1 || q.Due[0].Work != w.Work {
			t.Fatalf("after the drain the queue holds %+v, want the World's Work's one row", q)
		}
	})
}

// landings is the Works of each grabs.queue-changed, in order.
type landings struct {
	mu    sync.Mutex
	works [][]string
}

func (p *landings) Publish(_ context.Context, c grabs.QueueChanged) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.works = append(p.works, c.WorkIDs)
}

func TestADrainTakesAtMostDrainBatchWorksPerWrite(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
		ws := []works.WorkID{w.Work, w.AddWork(), w.AddWork()}
		seen := &landings{}
		w.Config.QueueChanged, w.Config.Timings.DrainBatch = seen, 2
		w.Restart()
		w.Reconciling(true)
		for _, work := range ws {
			nudge(t, w, work, searches.Refresh)
		}
		w.Run()
		synctest.Wait()
		var sizes []int
		var drained []string
		for _, p := range seen.works {
			sizes = append(sizes, len(p))
			drained = append(drained, p...)
		}
		first := []string{ws[0].String(), ws[1].String()}
		if !slices.Equal(sizes, []int{2, 1}) || !slices.Equal(sortedStrings(drained[:2]), sortedStrings(first)) || drained[2] != ws[2].String() {
			t.Fatalf("the drains landed %v, want the first 2 Works nudged in one Write, then the third", seen.works)
		}
		if q := w.Queue(t); len(q.Nudged) != 0 || len(q.Due) != len(ws) {
			t.Fatalf("after the drains the queue holds %+v, want a row for each Work", q)
		}
	})
}

func sortedStrings(s []string) []string {
	out := slices.Clone(s)
	slices.Sort(out)
	return out
}

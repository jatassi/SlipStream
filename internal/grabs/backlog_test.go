package grabs_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/media"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

const backlogEvery = 8 * time.Hour

func backlogWorld(t *testing.T) *grabstest.World {
	t.Helper()
	return queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, BacklogEvery: backlogEvery})
}

func planBacklog(t *testing.T, w *grabstest.World) grabs.BacklogPlan {
	t.Helper()
	plan, on, err := w.Grabs.PlanBacklog(t.Context())
	if err != nil || !on {
		t.Fatalf("PlanBacklog = %v, %v, want a plan of Backlog search on", on, err)
	}
	return plan
}

func applyBacklog(t *testing.T, w *grabstest.World, plan grabs.BacklogPlan) string {
	t.Helper()
	note, err := w.Grabs.ApplyBacklog(t.Context(), plan)
	if err != nil {
		t.Fatal(err)
	}
	return note
}

// backlogRun runs Backlog search's unit once and answers its note.
func backlogRun(t *testing.T, w *grabstest.World) string {
	t.Helper()
	return applyBacklog(t, w, planBacklog(t, w))
}

// later lets d pass on the bubble's clock.
func later(d time.Duration) {
	time.Sleep(d)
	synctest.Wait()
}

func monitorAll(t *testing.T, w *grabstest.World, on bool) {
	t.Helper()
	if _, err := w.Config.Works.SetMonitored(t.Context(), identity.Admin{}, w.Work, works.Within{Items: w.Items}, on); err != nil {
		t.Fatal(err)
	}
}

func idle(d grabstest.Due) bool { return d.Lane == 0 }

func byBacklog(d grabstest.Due) bool {
	return d.Lane == pace.Backlog && d.Trigger == searches.Backlog && d.Items == nil && !d.Gone
}

func not(f func(grabstest.Due) bool) func(grabstest.Due) bool {
	return func(d grabstest.Due) bool { return !f(d) }
}

func TestBacklogSearchRunTwiceGivesTheSameRowsAndQueuesOnlyWhatIsDue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := backlogWorld(t)
		w.AddWork()
		w.AddWork()
		if note := backlogRun(t, w); note != "Queued 0 searches." {
			t.Fatalf("the first run's note is %q, want nothing queued before the first due times", note)
		}
		first := w.Queue(t)
		if len(first.Due) != 3 || slices.ContainsFunc(first.Due, not(idle)) {
			t.Fatalf("after the first run the queue holds %+v, want a row per Work's group, each idle until it is due", first)
		}
		if note := backlogRun(t, w); note != "Queued 0 searches." || !reflect.DeepEqual(w.Queue(t), first) {
			t.Fatalf("Run now noted %q and left %+v, want nothing queued and the same rows %+v", note, w.Queue(t), first)
		}
		later(backlogEvery)
		if note := backlogRun(t, w); note != "Queued 3 searches." {
			t.Fatalf("a run one interval later noted %q, want every group queued", note)
		}
		queued := w.Queue(t)
		if slices.ContainsFunc(queued.Due, not(byBacklog)) {
			t.Fatalf("the queue holds %+v, want every whole group in backlog by Backlog search", queued.Due)
		}
		if note := backlogRun(t, w); note != "Queued 0 searches." || !reflect.DeepEqual(w.Queue(t), queued) {
			t.Fatalf("a second run noted %q and left %+v, want the same rows %+v", note, w.Queue(t), queued)
		}
	})
}

func TestFirstDueTimesSpreadOverOneInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := backlogWorld(t)
		ws := []works.WorkID{w.Work}
		for range 7 {
			ws = append(ws, w.AddWork())
		}
		seen := time.Now()
		backlogRun(t, w)
		dues := map[time.Time]bool{}
		for _, d := range w.Queue(t).Due {
			at := w.DueOf(t, d.Work, d.Group)
			if at.Before(seen) || !at.Before(seen.Add(backlogEvery)) {
				t.Errorf("%s is first due at %v, want within one interval of %v", d.Group, at, seen)
			}
			dues[at] = true
		}
		if len(dues) < len(ws)/2 {
			t.Fatalf("%d Works share %d first due times, want them spread", len(ws), len(dues))
		}
	})
}

func TestARowNothingWantsGoesUnlessATriggerTouchedItAfterTheSnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := backlogWorld(t)
		backlogRun(t, w)
		later(time.Minute)
		monitorAll(t, w, false)
		plan := planBacklog(t, w)
		later(time.Minute)
		monitorAll(t, w, true)
		enqueue(t, w, searches.Retarget)
		applyBacklog(t, w, plan)
		if q := w.Queue(t); len(q.Due) != 1 {
			t.Fatalf("the queue holds %+v, want the row a trigger touched after the snapshot", q)
		}
		monitorAll(t, w, false)
		later(time.Minute)
		backlogRun(t, w)
		if q := w.Queue(t); len(q.Due) != 0 {
			t.Fatalf("the queue holds %+v, want the row nothing wants gone", q)
		}
	})
}

func TestARunStartedBetweenTheSnapshotAndItsWriteKeepsItsRow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := backlogWorld(t)
		w.Answer("Quick Site", time.Hour, searching.Answer{})
		backlogRun(t, w)
		later(time.Minute)
		monitorAll(t, w, false)
		plan := planBacklog(t, w)
		monitorAll(t, w, true)
		press(t, w, grabs.Scope{})
		w.Run()
		if q := w.Queue(t); len(q.Runs) != 1 {
			t.Fatalf("the queue holds %+v, want the press's open run", q)
		}
		applyBacklog(t, w, plan)
		if q := w.Queue(t); len(q.Due) != 1 || !q.Due[0].Searching {
			t.Fatalf("the queue holds %+v, want the searching group's row", q)
		}
	})
}

// stopped stops the World's first Item's cell at its first failed Grab.
func stopped(t *testing.T, w *grabstest.World) {
	t.Helper()
	settings(t, w, map[string]string{grabs.StopCount.Key: "1"})
	failWith(t, w, grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", 'a')), grabs.ReleaseFault)
	if got := reads(cellOf(t, w)); got != "Failed, stopped" {
		t.Fatalf("the cell reads %q, want it stopped", got)
	}
}

func TestAStoppedCellKeepsItsRowAndOnlyBacklogSearchOpensIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := backlogWorld(t)
		w.Answer("Quick Site", time.Hour, searching.Answer{})
		stopped(t, w)
		w.Run()
		enqueue(t, w, searches.Add)
		synctest.Wait()
		if q := w.Queue(t); len(q.Runs) != 0 {
			t.Fatalf("an add opened %+v, want no run of a stopped cell", q.Runs)
		}
		backlogRun(t, w)
		later(backlogEvery)
		if note := backlogRun(t, w); note != "Queued 1 search." {
			t.Fatalf("Backlog search noted %q, want the stopped cell's group queued", note)
		}
		synctest.Wait()
		q := w.Queue(t)
		if len(q.Runs) != 1 || q.Runs[0].Trigger != searches.Backlog || !reflect.DeepEqual(q.Runs[0].Items, []works.ItemID{w.Item}) {
			t.Fatalf("the queue holds %+v, want a Backlog search run of the stopped cell's Item", q)
		}
	})
}

// filed gives work's first Item a file of quality, filling its Targets.
func filed(t *testing.T, w *grabstest.World, work works.WorkID, quality rel.Quality) {
	t.Helper()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		tree, err := w.Config.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return err
		}
		root, err := roots.OfWork(ctx, tx, work)
		if err != nil {
			return err
		}
		_, err = files.Add(ctx, tx, files.File{
			WorkID: work, Items: []works.ItemID{tree.Items[0].ID}, Cut: rel.Standard, RootID: root.ID, Path: work.String() + "/Scarf.mkv",
			Size: 1 << 30, Media: media.Record{Video: media.Video{Width: 1920, Height: 1080}}, Quality: quality,
		}, w.Goals)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWithUpgradesOffBacklogSearchQueuesOnlyMissingRows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := backlogWorld(t)
		missing := w.AddWork()
		filed(t, w, w.Work, rel.WEBDL1080p)
		settings(t, w, map[string]string{grabs.BacklogUpgrades.Key: "false"})
		backlogRun(t, w)
		later(backlogEvery)
		if note := backlogRun(t, w); note != "Queued 1 search." {
			t.Fatalf("Backlog search noted %q, want only the missing Work's group queued", note)
		}
		for _, d := range w.Queue(t).Due {
			if (d.Lane == pace.Backlog) != (d.Work == missing) {
				t.Errorf("row %+v, want only the missing Work's row queued", d)
			}
		}
		settings(t, w, map[string]string{grabs.BacklogUpgrades.Key: "true"})
		if note := backlogRun(t, w); note != "Queued 1 search." {
			t.Fatalf("with upgrades on Backlog search noted %q, want the upgradable Work's group queued too", note)
		}
	})
}

// chunks counts grabs.queue-changed publishes.
type chunks struct{ n int }

func (c *chunks) Publish(context.Context, grabs.QueueChanged) { c.n++ }

func TestNoBacklogWriteCoversMoreThanBacklogChunkRows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := backlogWorld(t)
		for range 4 {
			w.AddWork()
		}
		seen := &chunks{}
		w.Config.QueueChanged, w.Config.Timings.BacklogChunk = seen, 2
		w.Restart()
		backlogRun(t, w)
		if seen.n != 3 || len(w.Queue(t).Due) != 5 {
			t.Fatalf("5 groups reconciled in %d Writes into %d rows, want 3 Writes of at most 2 and 5 rows", seen.n, len(w.Queue(t).Due))
		}
	})
}

func TestACellNoLongerWantedDropsItsFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := backlogWorld(t)
		bad := heard(w, "E01", "1080p.WEB-DL", 'a')
		failWith(t, w, grabbedOver(t, w, bad), grabs.ReleaseFault)
		if !refused(t, w, bad) {
			t.Fatal("the failed Release isn't refused")
		}
		later(time.Minute)
		backlogRun(t, w)
		if !refused(t, w, bad) {
			t.Fatal("Backlog search cleared the failures of a cell still wanted")
		}
		monitorAll(t, w, false)
		backlogRun(t, w)
		if refused(t, w, bad) {
			t.Fatal("the cell is no longer wanted and its failure still refuses the failed Release")
		}
	})
}

func TestBacklogSearchOffSkipsOwned(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := backlogWorld(t)
		settings(t, w, map[string]string{grabs.BacklogOn.Key: "false"})
		task := w.Grabs.Tasks()[0]
		err := task.Run(t.Context(), tasks.Unit{Key: "due"}, &tasks.Progress{})
		skip, ok := errors.AsType[*tasks.SkipError](err)
		if !ok || !skip.Owned || skip.Cause != "Backlog search is off in Settings." {
			t.Fatalf("Backlog search off ran %v, want an owned skip", err)
		}
		if q := w.Queue(t); len(q.Due) != 0 {
			t.Fatalf("Backlog search off left %+v, want nothing reconciled", q)
		}
	})
}

func TestACellAGrabHoldsKeepsItsFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := backlogWorld(t)
		bad := heard(w, "E01", "1080p.WEB-DL", 'a')
		failWith(t, w, grabbedOver(t, w, bad), grabs.ReleaseFault)
		grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", 'b'))
		later(time.Minute)
		backlogRun(t, w)
		if !refused(t, w, bad) {
			t.Fatal("Backlog search cleared the failures of a cell a Grab still holds")
		}
	})
}

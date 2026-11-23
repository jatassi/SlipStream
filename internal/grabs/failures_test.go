package grabs_test

import (
	"context"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestEveryCodeHasAFailureRule(t *testing.T) {
	for _, c := range grabs.Codes {
		if !grabs.HasFailureRule(c) {
			t.Errorf("the failure table has no row for %s", c)
		}
	}
}

func failureWorld(t *testing.T) *grabstest.World {
	t.Helper()
	return queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}})
}

// grabbedOver decides a run of the World's Item over r and answers the Grab it
// made.
func grabbedOver(t *testing.T, w *grabstest.World, r indexers.Release) grabs.GrabID {
	t.Helper()
	if _, err := w.Grabs.DecideRun(t.Context(), w.Work, w.Items, []indexers.Release{r}); err != nil {
		t.Fatal(err)
	}
	views, err := w.Grabs.List(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range views {
		if v.Release.Title == r.Title && v.Stage == grabs.Fetching {
			return v.ID
		}
	}
	t.Fatalf("the run over %s made no Grab of it", r.Title)
	return grabs.GrabID{}
}

func failWith(t *testing.T, w *grabstest.World, id grabs.GrabID, code grabs.Code) {
	t.Helper()
	if err := w.Grabs.FailWith(t.Context(), id, code); err != nil {
		t.Fatal(err)
	}
}

// cellOf is the World's first Item's cell.
func cellOf(t *testing.T, w *grabstest.World) files.Cell {
	t.Helper()
	var cells []files.Cell
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		cells, err = w.Config.Files.Cells(ctx, r, w.Work)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(cells, func(c files.Cell) bool { return c.Item.ID == w.Item })
	if i < 0 {
		t.Fatal("the World's Item has no cell")
	}
	return cells[i]
}

func refused(t *testing.T, w *grabstest.World, r indexers.Release) bool {
	t.Helper()
	got, err := w.Grabs.Tried(t.Context(), w.Work, r)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func reads(c files.Cell) string {
	words := map[files.State]string{files.Missing: "Missing", files.Failed: "Failed"}[c.State]
	if c.Search.Stopped {
		words += ", stopped"
	}
	return words
}

func TestAFaultLoopStopsACellAtTheThirdCountedFailureAndTheStopOutlivesTheWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := failureWorld(t)
		for i, hash := range []byte{'a', 'b', 'c'} {
			failWith(t, w, grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", hash)), grabs.ReleaseFault)
			want, nudged := "Missing", i+1
			if i == 2 {
				want, nudged = "Failed, stopped", 2
			}
			if got := reads(cellOf(t, w)); got != want {
				t.Fatalf("after failure %d the cell reads %q, want %q", i+1, got, want)
			}
			if got := len(w.Queue(t).Nudged); got != nudged {
				t.Fatalf("after failure %d the queue holds %d nudges, want %d: the stopped cell isn't searched again", i+1, got, nudged)
			}
		}
		time.Sleep(8 * 24 * time.Hour)
		synctest.Wait()
		if got := reads(cellOf(t, w)); got != "Failed, stopped" {
			t.Fatalf("past the 7-day window the cell reads %q, want it still stopped", got)
		}
	})
}

func TestAReleaseFaultSearchesItsCellsAgainAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := failureWorld(t)
		w.Answer("Quick Site", time.Hour, searching.Answer{})
		failWith(t, w, grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", 'a')), grabs.ReleaseFault)
		w.Run()
		q := w.Queue(t)
		if len(q.Runs) != 1 || q.Runs[0].Trigger != searches.FailedGrab || q.Runs[0].Lane != pace.Now || !slices.Equal(q.Runs[0].Items, []works.ItemID{w.Item}) {
			t.Fatalf("the queue holds %+v, want one run of the failed cell's Item after a failed Grab, in the now lane", q)
		}
	})
}

func TestOnlyACodeThatSearchesAgainNudgesAndOnlyASetupFaultReadsFailed(t *testing.T) {
	for _, tc := range []struct {
		code   grabs.Code
		reads  string
		nudges int
	}{
		{grabs.NoVideoFile, "Missing", 1},
		{grabs.IndexerUnavailable, "Missing", 0},
		{grabs.ClientError, "Missing", 0},
		{grabs.NameTooLong, "Failed", 0},
		{grabs.Canceled, "Missing", 0},
	} {
		t.Run(string(tc.code), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := failureWorld(t)
				failWith(t, w, grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", 'a')), tc.code)
				if got := reads(cellOf(t, w)); got != tc.reads {
					t.Errorf("the cell reads %q, want %q", got, tc.reads)
				}
				if got := len(w.Queue(t).Nudged); got != tc.nudges {
					t.Errorf("the queue holds %d nudges, want %d", got, tc.nudges)
				}
			})
		})
	}
}

func TestTheAdminsSearchClearsTheCountAndTheStopAndKeepsTheRefusal(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := failureWorld(t)
		settings(t, w, map[string]string{grabs.StopCount.Key: "1"})
		bad := heard(w, "E01", "1080p.WEB-DL", 'a')
		failWith(t, w, grabbedOver(t, w, bad), grabs.ReleaseFault)
		if got := reads(cellOf(t, w)); got != "Failed, stopped" {
			t.Fatalf("with a stop count of 1 the first failure leaves the cell %q, want it stopped", got)
		}
		press(t, w, grabs.Scope{})
		if got := reads(cellOf(t, w)); got != "Missing" {
			t.Fatalf("after the Admin's Search the cell reads %q, want Missing", got)
		}
		if !refused(t, w, bad) {
			t.Fatal("the Admin's Search lifted the refusal of the failed Release")
		}
	})
}

func TestTheSameGrabFailingTwiceWritesOneRowPerCell(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := failureWorld(t)
		settings(t, w, map[string]string{grabs.StopCount.Key: "2"})
		id := grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", 'a'))
		failWith(t, w, id, grabs.ReleaseFault)
		if err := w.Grabs.RecordFailure(t.Context(), id, grabs.ReleaseFault); err != nil {
			t.Fatal(err)
		}
		if got := reads(cellOf(t, w)); got != "Missing" {
			t.Fatalf("one Grab failing twice leaves the cell %q, want Missing with one counted failure of the two it stops at", got)
		}
	})
}

func TestAPrunedGrabsFailureStillRefusesItsRelease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := failureWorld(t)
		bad := heard(w, "E01", "1080p.WEB-DL", 'a')
		id := grabbedOver(t, w, bad)
		failWith(t, w, id, grabs.ReleaseFault)
		run(t, w.Lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `DELETE FROM grabs_grab WHERE id = ?`, id)
			return err
		})
		if !refused(t, w, bad) {
			t.Fatal("pruning the failed Grab lifted the refusal of its Release")
		}
		other := heard(w, "E01", "1080p.WEB-DL", 'b')
		if refused(t, w, other) {
			t.Fatal("the refusal reaches a Release of another info hash")
		}
	})
}

func TestARefusalMatchesByIndexerAndGUIDWhenNoHashIsKnown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := failureWorld(t)
		bare := heard(w, "E01", "1080p.WEB-DL", 'a')
		bare.InfoHash = nil
		id := grabbedOver(t, w, bare)
		failWith(t, w, id, grabs.ReleaseFault)
		if v, err := w.Grabs.Get(t.Context(), identity.Admin{}, id); err != nil || v.Stage != grabs.Failed {
			t.Fatalf("the Grab is %s (%v), want failed", v.Stage, err)
		}
		if refused(t, w, bare) {
			t.Fatal("a failure with no hash and no GUID refuses its Release")
		}

		bad := heard(w, "E01", "1080p.WEB-DL", 'b')
		bad.InfoHash, bad.GUID = nil, "https://site.test/t/1"
		failWith(t, w, grabbedOver(t, w, bad), grabs.ReleaseFault)
		if !refused(t, w, bad) {
			t.Fatal("the failure doesn't refuse its Release by its Indexer and GUID")
		}
		again := bad
		again.GUID = "https://site.test/t/2"
		if refused(t, w, again) {
			t.Fatal("the refusal reaches a Release of another GUID")
		}
	})
}

func TestAManualPicksReleaseFaultReadsFailedAndNeverSearchesAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := failureWorld(t)
		id := grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", 'a'))
		run(t, w.Lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `UPDATE grabs_grab SET source = 'manual', grabbed_by_name = 'Admin' WHERE id = ?`, id)
			return err
		})
		failWith(t, w, id, grabs.ReleaseFault)
		if got := reads(cellOf(t, w)); got != "Failed" {
			t.Fatalf("the Manual pick's cell reads %q, want Failed with its card", got)
		}
		if got := w.Queue(t).Nudged; len(got) != 0 {
			t.Fatalf("the queue holds nudges %v, want none: a Manual pick never searches again by itself", got)
		}
	})
}

func TestACellThatFillsDeletesItsFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := failureWorld(t)
		bad := heard(w, "E01", "1080p.WEB-DL", 'a')
		failWith(t, w, grabbedOver(t, w, bad), grabs.ReleaseFault)
		imported(t, w, grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", 'b')))
		if refused(t, w, bad) {
			t.Fatal("the cell filled and its failure still refuses the failed Release")
		}
	})
}

// imported moves Grab id through importing to imported, its Import having
// placed every cell it held.
func imported(t *testing.T, w *grabstest.World, id grabs.GrabID) {
	t.Helper()
	followed(t, w, id, grabs.SettledPlaced, grabs.Report{Placed: true})
	if v, err := w.Grabs.Get(t.Context(), identity.Admin{}, id); err != nil || v.Stage != grabs.Imported {
		t.Fatalf("the Grab is %v (%v), want imported", v.Stage, err)
	}
}

// followed moves Grab id to importing and has its Import report r, with
// every cell settled as s, in one Write.
func followed(t *testing.T, w *grabstest.World, id grabs.GrabID, s grabs.Settled, r grabs.Report) {
	t.Helper()
	rel := newRelease(t)
	raw, err := rel.link.Torrent(rel.info)
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	v, err := w.Grabs.Get(t.Context(), identity.Admin{}, id)
	if err != nil {
		t.Fatal(err)
	}
	r.Settled = map[files.CellKey]grabs.Settled{}
	for _, c := range v.Targets {
		r.Settled[c.Key] = s
	}
	run(t, w.Lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		torrent, _, err := downloadclients.Attach(ctx, tx, downloadclients.Want{
			Target: downloadclients.Target{Node: w.Home}, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID,
			WrittenOff: func(role.NodeID) bool { return false },
		}, storage.Now())
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE grabs_grab SET stage = 'importing', torrent_id = ? WHERE id = ?`, torrent, id); err != nil {
			return err
		}
		_, err = w.Grabs.Follow(ctx, tx, id, r)
		return err
	})
}

// importEnded has Grab id's Import end with code, placing nothing and
// settling every cell none in the Write that fails the Grab, as an Import
// that found no file for them does.
func importEnded(t *testing.T, w *grabstest.World, id grabs.GrabID, code grabs.Code) {
	t.Helper()
	followed(t, w, id, grabs.SettledNone, grabs.Report{Ended: &grabs.Failure{Stage: grabs.Importing, Code: code}})
}

func TestAnImportThatPlacesNothingCountsLikeAnyFailedGrab(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := failureWorld(t)
		settings(t, w, map[string]string{grabs.StopCount.Key: "2"})
		first := grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", 'a'))
		importEnded(t, w, first, grabs.NoVideoFile)
		if got := reads(cellOf(t, w)); got != "Missing" {
			t.Fatalf("after the first failed Import the cell reads %q, want Missing", got)
		}
		if v, err := w.Grabs.Get(t.Context(), identity.Admin{}, first); err != nil || !v.SearchingAgain {
			t.Fatalf("the first failed Grab reads searching again %v, %v: want true, so Needs you doesn't list it while its cell reads Missing", v.SearchingAgain, err)
		}
		if got := len(w.Queue(t).Nudged); got != 1 {
			t.Fatalf("after the first failed Import the queue holds %d nudges, want 1: it searches again at once", got)
		}
		second := grabbedOver(t, w, heard(w, "E01", "1080p.WEB-DL", 'b'))
		importEnded(t, w, second, grabs.NoVideoFile)
		if got := reads(cellOf(t, w)); got != "Failed, stopped" {
			t.Fatalf("after the second failed Import the cell reads %q, want it stopped at the stop count of 2", got)
		}
		if got := len(w.Queue(t).Nudged); got != 1 {
			t.Fatalf("after the stop the queue holds %d nudges, want still 1", got)
		}
		v, err := w.Grabs.Get(t.Context(), identity.Admin{}, second)
		if err != nil {
			t.Fatal(err)
		}
		if !v.Current || !v.NeedsYou() || v.SearchingAgain {
			t.Fatalf("the stopping Grab reads current %v, needs you %v, searching again %v: want current and needs you only, so Needs you lists it", v.Current, v.NeedsYou(), v.SearchingAgain)
		}
	})
}

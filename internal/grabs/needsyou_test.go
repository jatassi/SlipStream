package grabs_test

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func needsYouCount(t *testing.T, w *grabstest.World) int {
	t.Helper()
	var n int
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		n, err = grabs.NeedsYou(ctx, r)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// needsYouRows is what Activity lists under Needs you: each failure that
// needs the Admin, isn't searched again, and no later Grab of its cell
// replaced.
func needsYouRows(t *testing.T, w *grabstest.World) int {
	t.Helper()
	views, err := w.Grabs.List(t.Context(), identity.Admin{})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, v := range views {
		if v.NeedsYou() && !v.SearchingAgain && v.Current {
			n++
		}
	}
	return n
}

// grabbedFor decides a run of item over r and answers the Grab it made.
func grabbedFor(t *testing.T, w *grabstest.World, item works.ItemID, r indexers.Release) grabs.GrabID {
	t.Helper()
	if _, err := w.Grabs.DecideRun(t.Context(), w.Work, []works.ItemID{item}, []indexers.Release{r}); err != nil {
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

func TestTheNeedsYouCountIsActivitysNeedsYouRows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Season: 5})
		failed := grabbedFor(t, w, w.Items[0], heard(w, "Scarf", "1080p.WEB-DL", 'a'))
		failWith(t, w, failed, grabs.NameTooLong)
		failWith(t, w, grabbedFor(t, w, w.Items[1], heard(w, "E02", "1080p.WEB-DL", 'b')), grabs.ReleaseFault)
		canceled := grabbedFor(t, w, w.Items[2], heard(w, "E03", "1080p.WEB-DL", 'c'))
		if err := w.Grabs.Cancel(t.Context(), identity.Admin{}, canceled, false); err != nil {
			t.Fatal(err)
		}
		failWith(t, w, grabbedFor(t, w, w.Items[3], heard(w, "E04", "1080p.WEB-DL", 'd')), grabs.NameTooLong)
		grabbedFor(t, w, w.Items[3], heard(w, "E04", "1080p.WEB-DL", 'e'))
		if got, rows := needsYouCount(t, w), needsYouRows(t, w); got != 1 || rows != 1 {
			t.Fatalf("the count is %d and Activity lists %d under Needs you, want both 1: the failure no later Grab replaced, not searched again or canceled", got, rows)
		}
	})
}

// copies seeds a failed Grab like id for each of items, one cell each,
// so the count reads many failures without a run each.
func copies(t *testing.T, w *grabstest.World, id grabs.GrabID, items []works.ItemID) {
	t.Helper()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, item := range items {
			cp := storage.NewUUID()
			if _, err := tx.ExecContext(ctx, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size,
				release_magnet, stage, failed_stage, failure_code, failure_message, created_at, updated_at)
				SELECT ?, work_id, ?, edition, source, node_id, release_title, release_size, release_magnet, stage, failed_stage,
				failure_code, failure_message, created_at, updated_at FROM grabs_grab WHERE id = ?`, cp, item, id); err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO grabs_target (grab_id, position, item_id, target_id)
				SELECT ?, 0, ?, target_id FROM grabs_target WHERE grab_id = ?`, cp, item, id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTheNeedsYouCountStopsAt100(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := queueWorld(t, grabstest.Options{Sites: []string{"Quick Site"}, Season: 101})
		first := grabbedFor(t, w, w.Items[0], heard(w, "Scarf", "1080p.WEB-DL", 'a'))
		failWith(t, w, first, grabs.NameTooLong)
		copies(t, w, first, w.Items[1:])
		if rows := needsYouRows(t, w); rows != 101 {
			t.Fatalf("Activity lists %d under Needs you, want 101", rows)
		}
		if got := needsYouCount(t, w); got != 100 {
			t.Fatalf("the count is %d, want it to stop at 100", got)
		}
	})
}

package reconciling_test

import (
	"context"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// newMachine begins Reconciling for a restore onto a new machine, with
// one root on node.
func (w *world) newMachine(node role.NodeID) roots.ID {
	w.t.Helper()
	root := w.rootOn(node)
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return reconciling.Begin(ctx, tx, reconciling.BeginArgs{ID: "restore-1", Cause: reconciling.Restore, Backup: "nightly", TakenAt: time.Now(), NewMachine: true})
	})
	return root
}

func (w *world) rootOn(node role.NodeID) roots.ID {
	w.t.Helper()
	root := roots.ID{UUID: storage.NewUUID()}
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO roots_root (id, content_type, node_id, path, is_default) VALUES (?, ?, ?, '/media/fixture', 1)", root, contenttypetest.ID, node)
		return err
	})
	return root
}

// reports has root's Node report it Ready or not, and publishes
// roots.changed as the roots recorder does.
func (w *world) reports(root roots.ID, ready bool) {
	w.t.Helper()
	w.mu.Lock()
	w.ready[root] = ready
	w.mu.Unlock()
	roots.ChangedEvent.Bind(w.hub).Publish(w.t.Context(), roots.Changed{RootIDs: []string{root.String()}})
	synctest.Wait()
}

func (w *world) scansOf(root roots.ID) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, ids := range w.scans {
		if slices.Contains(ids, root) {
			n++
		}
	}
	return n
}

// scanned finishes root's scan as imports' Write does.
func (w *world) scanned(root roots.ID) {
	w.t.Helper()
	w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return w.keeper.Scanned(ctx, tx, root)
	})
	w.keeper.ScanEnded()
	synctest.Wait()
}

// TestTheEndRunsQueuedWork saves a root in the re-home step, which starts
// its scan once, and ends Reconciling once the scan finishes and the Node
// is continued without: the end's Write decides queued work, and the work
// that waited runs once it commits.
func TestTheEndRunsQueuedWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.node("vps-relay")
		root := w.newMachine(relay)
		saved, err := w.keeper.Rehome(t.Context(), identity.Admin{}, []reconciling.Save{{Root: root}}, true)
		if err != nil || len(saved) != 1 || saved[0].Err != nil {
			t.Fatalf("Rehome = %+v, %v, want the root saved", saved, err)
		}
		synctest.Wait()
		w.reports(root, true)
		if !slices.Equal(w.rehomed, []roots.ID{root}) || len(w.scans) != 1 || !slices.Equal(w.scans[0], []roots.ID{root}) {
			t.Fatalf("the re-home step rehomed %v and started scans %v, want the root through Rehome and one scan of it", w.rehomed, w.scans)
		}
		w.scanned(root)
		if w.phase() != reconciling.Reconciling || w.ending != 0 || w.ended != 0 {
			t.Fatalf("with vps-relay still waited on Reconciling is %v after %d and %d end hooks, want it running and none", w.phase(), w.ending, w.ended)
		}
		if err := w.keeper.ContinueNode(t.Context(), relay); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if w.phase() != reconciling.None || w.ending != 1 || w.ended != 1 {
			t.Errorf("after the last wait Reconciling is %v after %d in-Write and %d after-commit hooks, want ended with one of each", w.phase(), w.ending, w.ended)
		}
	})
}

// notHeld records n torrents on node, restores, and has node report none
// of them, so each goes on the review list as not_held.
func (w *world) notHeld(node role.NodeID, trs ...torrent) []dc.ID {
	w.t.Helper()
	ids := make([]dc.ID, len(trs))
	for i, tr := range trs {
		ids[i] = w.record(node, tr)
	}
	w.restore()
	w.reportAll(node)
	return ids
}

func TestReAddWhileReconcilingRunsAtTheEnd(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay, pc := w.paired("vps-relay"), w.paired("media-pc")
		orchard := newTorrent(t, "orchard")
		ids := w.notHeld(relay, orchard)
		if err := w.keeper.Readd(t.Context(), ids, &pc); err != nil {
			t.Fatal(err)
		}
		if got := w.only(orchard); !got.withheld || w.review(ids[0]) != "not_held" || w.lists(pc, orchard) {
			t.Fatalf("while Reconciling orchard is recorded as %+v and on the review list as %q, want withheld and still listed for review", got, w.review(ids[0]))
		}
		if err := w.keeper.Finish(t.Context()); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if got := w.only(orchard); got.withheld || got.node != pc || w.review(ids[0]) != "" || !w.lists(pc, orchard) {
			t.Errorf("once Reconciling ends orchard is recorded as %+v and on the review list as %q, want released onto media-pc", got, w.review(ids[0]))
		}
	})
}

func TestForgetFailsTheGrabRestored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.paired("vps-relay")
		ids := w.notHeld(relay, newTorrent(t, "orchard"))
		if err := w.keeper.Forget(t.Context(), ids); err != nil {
			t.Fatal(err)
		}
		var out map[dc.ID]dc.Attached
		err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			out, err = dc.Outcomes(ctx, r, ids)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if gone, ok := out[ids[0]].Outcome.(dc.Gone); !ok || gone.Reason != dc.Forgotten || w.review(ids[0]) != "" || w.forgot != 1 {
			t.Errorf("after Forget orchard's outcome is %+v, on the review list as %q, after %d Forgot calls; want gone forgotten, off the list and one call",
				out[ids[0]].Outcome, w.review(ids[0]), w.forgot)
		}
	})
}

// TestTheReviewListPagesByAStableCursor pages five not held torrents two
// at a time while a listed one leaves the list: each of the five is listed
// once.
func TestTheReviewListPagesByAStableCursor(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.paired("vps-relay")
		ids := w.notHeld(relay, torrents(t, "t", 5)...)
		seen := map[dc.ID]int{}
		first := true
		w.pages(2, func(rows []reconciling.Reviewed) {
			for _, r := range rows {
				seen[r.Torrent]++
			}
			if first {
				first = false
				if err := w.keeper.Forget(t.Context(), []dc.ID{rows[0].Torrent}); err != nil {
					t.Fatal(err)
				}
			}
		})
		for _, id := range ids {
			if seen[id] != 1 {
				t.Errorf("torrent %s was listed %d times, want once: %v", id, seen[id], seen)
			}
		}
	})
}

// pages reads the whole review list n rows a page, handing each page to
// each before it reads the next.
func (w *world) pages(n int, each func([]reconciling.Reviewed)) {
	w.t.Helper()
	var c *reconciling.Cursor
	for {
		rows, next, err := w.keeper.ReviewPage(w.t.Context(), reconciling.Filter{}, c, n)
		if err != nil {
			w.t.Fatal(err)
		}
		each(rows)
		if next == nil {
			return
		}
		c = next
	}
}

// TestTheReviewListSaysWhichFilesItsNodeStillHas puts two recorded
// torrents on the review list: their Node still has orchard's files in
// its save folder, and none of attic's.
func TestTheReviewListSaysWhichFilesItsNodeStillHas(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.paired("vps-relay")
		orchard, attic := newTorrent(t, "orchard"), newTorrent(t, "attic")
		w.onDisk["/downloads/orchard"] = true
		for _, tr := range []torrent{orchard, attic} {
			id := w.record(relay, tr)
			w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				_, err := tx.ExecContext(ctx, "UPDATE downloadclients_torrent SET save_folder = ? WHERE id = ?", "/downloads/"+tr.name, id)
				return err
			})
		}
		w.restore()
		w.reportAll(relay)
		found := map[string]reconciling.Found{}
		w.pages(10, func(rows []reconciling.Reviewed) {
			for _, r := range rows {
				found[r.Name] = r.Found
			}
		})
		if found["orchard"] != reconciling.Files || found["attic"] != reconciling.NoFiles {
			t.Errorf("the review list found %v, want orchard's files and none of attic's", found)
		}
	})
}

// TestARestoredRootScansOnceItsNodeReportsItReady restores in place
// while the root's Node hasn't checked its folder since it started: a
// scan then would end Skipped and record nothing, so Reconciling starts
// it only once the Node reports the root Ready, and again each time the
// root turns Ready after its scan ended without one.
func TestARestoredRootScansOnceItsNodeReportsItReady(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.node("vps-relay")
		root := w.rootOn(relay)
		w.restore()
		w.keeper.Wake()
		synctest.Wait()
		if n := w.scansOf(root); n != 0 {
			t.Fatalf("before vps-relay checked the root Reconciling started %d scans of it, want none", n)
		}
		w.reports(root, true)
		if n := w.scansOf(root); n != 1 {
			t.Fatalf("once vps-relay reports the root Ready Reconciling started %d scans of it, want 1", n)
		}
		w.reports(root, true)
		if n := w.scansOf(root); n != 1 {
			t.Fatalf("with the root still Ready Reconciling started %d scans of it, want still 1", n)
		}
		w.reports(root, false)
		w.reports(root, true)
		if n := w.scansOf(root); n != 2 {
			t.Fatalf("after the root turned Ready again with its scan unrecorded Reconciling started %d scans of it, want 2", n)
		}
		w.scanned(root)
		w.reports(root, false)
		w.reports(root, true)
		if n := w.scansOf(root); n != 2 {
			t.Fatalf("after the root's scan was recorded Reconciling started %d scans of it, want still 2", n)
		}
	})
}

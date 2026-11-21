package backups_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/identity"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/transfers"
)

const vpsFolder = "/srv/backups"

// add adds vpsFolder on vps as a Backup destination and waits for its first
// queue.
func (w *world) add() backups.Destination {
	w.t.Helper()
	a, err := w.backups.Add(w.t.Context(), identity.Admin{}, w.vps, vpsFolder)
	if err != nil {
		w.t.Fatal(err)
	}
	w.wg.Wait()
	return a.Destination
}

// queue runs d's unit of the Backup task.
func (w *world) queue(d backups.Destination) {
	w.t.Helper()
	if err := w.backups.Task().Run(w.t.Context(), tasks.Unit{Key: backups.DestinationUnit(d.ID)}, &tasks.Progress{}); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) settle() {
	w.t.Helper()
	if err := w.backups.Settle(w.t.Context()); err != nil {
		w.t.Fatal(err)
	}
}

// records is every Backup Transfer the library holds.
func (w *world) records() []transfers.Transfer {
	w.t.Helper()
	var out []transfers.Transfer
	if err := w.lib.Read(w.t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		out, err = w.tr.OfPurpose(ctx, rd, transfers.Backup)
		return err
	}); err != nil {
		w.t.Fatal(err)
	}
	return out
}

func (w *world) only() transfers.Transfer {
	w.t.Helper()
	rs := w.records()
	if len(rs) != 1 {
		w.t.Fatalf("the library holds %d Backup Transfers, want 1", len(rs))
	}
	return rs[0]
}

// result has node report r.
func (w *world) result(node role.NodeID, r *tv1.Result) {
	w.t.Helper()
	if err := w.lib.Write(w.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return w.tr.Apply(ctx, tx, node, r)
	}); err != nil {
		w.t.Fatal(err)
	}
}

func (w *world) arrive(node role.NodeID, tr transfers.Transfer) {
	w.result(node, &tv1.Result{Result: &tv1.Result_Verified{Verified: &tv1.FileVerified{TransferId: tr.ID.Bytes(), Position: 0, Place: 1}}})
}

// report sets what node last reported of its Backup folders.
func (w *world) report(node role.NodeID, folders ...*tv1.BackupFolderFacts) {
	w.factsMu.Lock()
	defer w.factsMu.Unlock()
	w.facts[node] = &tv1.Facts{BackupFolders: folders}
}

func (w *world) entry(id archive.ID) backups.Entry {
	w.t.Helper()
	l := w.eng.Index().Read()
	e, ok := l.Entry(id)
	if !ok {
		w.t.Fatalf("the index lists no Backup %x", id[:4])
	}
	return e
}

func (w *world) copyOf(id archive.ID, d backups.Destination) backups.Copy {
	w.t.Helper()
	e := w.entry(id)
	c, ok := e.Copy(d.ID)
	if !ok {
		w.t.Fatalf("Backup %s has no copy on %s", e.Name, d.Folder)
	}
	return c
}

func (w *world) change(id archive.ID, fn func(*backups.Entry)) {
	w.t.Helper()
	if err := w.eng.Index().Update(func(l *backups.Ledger) error {
		e, _ := l.Entry(id)
		fn(&e)
		for i := range l.Backups {
			if l.Backups[i].ID == id {
				l.Backups[i] = e
			}
		}
		return nil
	}); err != nil {
		w.t.Fatal(err)
	}
}

// copied is a world with one Backup copied to vps, which lists it.
func copied(t *testing.T) (*world, backups.Entry, backups.Destination) {
	t.Helper()
	w := newWorld(t)
	w.opened("2.1.0")
	e := w.taken(archive.Scheduled)
	d := w.add()
	w.arrive(w.vps, w.only())
	w.report(w.vps, &tv1.BackupFolderFacts{Folder: vpsFolder, Names: []string{e.Name.String()}})
	w.settle()
	if c := w.copyOf(e.ID, d); c.State != backups.Copied {
		t.Fatalf("the copy is %s, want copied", c.State)
	}
	if rs := w.records(); len(rs) != 0 {
		t.Fatalf("a verified copy left %d Transfers", len(rs))
	}
	return w, e, d
}

// check is the check vps's folder asks for, failing when none is asked.
func (w *world) check() transfers.BackupFolder {
	w.t.Helper()
	for _, f := range w.backups.Folders(w.vps) {
		if f.Folder == vpsFolder && len(f.CheckKey) > 0 {
			return f
		}
	}
	w.t.Fatal("vps is asked no check")
	return transfers.BackupFolder{}
}

// TestACheckMismatchFailsTheCopyAndQueuesItAgain: once Check copies has
// passed, vps is asked a keyed BLAKE3 of its copy under a fresh key. A
// matching answer passes; a sum that isn't the Backup's fails the copy
// and queues it again at once.
func TestACheckMismatchFailsTheCopyAndQueuesItAgain(t *testing.T) {
	w, e, d := copied(t)
	w.queue(d)
	if f := w.backups.Folders(w.vps); len(f) != 1 || len(f[0].CheckKey) > 0 {
		t.Fatalf("vps is asked %+v before Check copies passed", f)
	}

	w.ahead.Store(int64(8 * 24 * time.Hour))
	w.queue(d)
	asked := w.check()
	if asked.Check != e.Name.String() {
		t.Fatalf("vps is asked to check %q", asked.Check)
	}
	want := keyed(t, w.backups.Path(e), asked.CheckKey)
	w.report(w.vps, &tv1.BackupFolderFacts{Folder: vpsFolder, Names: []string{e.Name.String()}, Check: &tv1.BackupCheck{Name: asked.Check, Key: asked.CheckKey, Sum: want}})
	w.settle()
	if c := w.copyOf(e.ID, d); c.State != backups.Copied || c.Check.Result != backups.CheckOK {
		t.Fatalf("a matching check left the copy %s, checked %s", c.State, c.Check.Result)
	}

	w.ahead.Store(int64(16 * 24 * time.Hour))
	w.queue(d)
	asked = w.check()
	w.report(w.vps, &tv1.BackupFolderFacts{Folder: vpsFolder, Names: []string{e.Name.String()}, Check: &tv1.BackupCheck{Name: asked.Check, Key: asked.CheckKey, Sum: want}})
	w.settle()
	c := w.copyOf(e.ID, d)
	if c.Check.Result != backups.CheckMismatch {
		t.Fatalf("a stale sum under a new key checked %s", c.Check.Result)
	}
	if c.State != backups.Copying || w.only().Job == "" {
		t.Fatalf("a mismatched copy is %s, want queued again", c.State)
	}
}

// pullFailed is whether e's pull failed, and false once it arrived.
func pullFailed(e backups.Entry) bool { return e.Pull != nil && e.Pull.State == backups.CopyFailed }

func keyed(t *testing.T, path string, key []byte) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	h, err := blake3.NewKeyed(key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(h, f); err != nil {
		t.Fatal(err)
	}
	return h.Sum(nil)
}

// TestACopyWithNoLocalFileCantBeChecked: a Backup this machine no longer
// holds has nothing to compare a copy with, so its check says so and asks
// vps nothing.
func TestACopyWithNoLocalFileCantBeChecked(t *testing.T) {
	w, e, d := copied(t)
	w.change(e.ID, func(e *backups.Entry) { e.Local = false })
	w.ahead.Store(int64(8 * 24 * time.Hour))
	w.queue(d)
	if c := w.copyOf(e.ID, d); c.Check.Result != backups.CheckNoLocalFile || c.State != backups.Copied {
		t.Fatalf("the copy is %s, checked %q", c.State, c.Check.Result)
	}
	if f := w.backups.Folders(w.vps); len(f) != 1 || len(f[0].CheckKey) > 0 {
		t.Fatalf("vps is asked %+v", f)
	}
}

// TestDeletesTravelByNameAndRemovingADestinationDeletesNothing: a pruned
// Backup is deleted on vps by its name, and leaves the index once vps no
// longer lists it. Removing a destination asks its Node to delete
// nothing and leaves its copies there.
func TestDeletesTravelByNameAndRemovingADestinationDeletesNothing(t *testing.T) {
	w, e, _ := copied(t)
	w.change(e.ID, func(e *backups.Entry) { e.State = backups.Deleting })
	if err := w.eng.Recover(); err != nil {
		t.Fatal(err)
	}
	if f := w.backups.Folders(w.vps); len(f) != 1 || !slices.Equal(f[0].Delete, []string{e.Name.String()}) {
		t.Fatalf("vps is asked %+v, want the Backup deleted by its name", f)
	}
	if got := w.entry(e.ID); got.Local {
		t.Fatal("the pruned Backup's file is still Local")
	}
	w.report(w.vps, &tv1.BackupFolderFacts{Folder: vpsFolder})
	w.settle()
	if l := w.eng.Index().Read(); len(l.Backups) != 0 {
		t.Fatalf("the index still lists %d Backups once vps deleted its copy", len(l.Backups))
	}

	w, e, d := copied(t)
	if err := w.backups.Remove(t.Context(), d.ID); err != nil {
		t.Fatal(err)
	}
	if f := w.backups.Folders(w.vps); len(f) != 1 || len(f[0].Delete) > 0 {
		t.Fatalf("a removed destination asks %+v", f)
	}
	w.settle()
	if f := w.backups.Folders(w.vps); len(f) != 0 {
		t.Fatalf("vps still keeps %+v once nothing runs into it", f)
	}
	if got := w.entry(e.ID); len(got.Copies) != 0 || !got.Local {
		t.Fatalf("the Backup reads %+v", got)
	}
}

// TestACopyFinishingAfterItsDestinationWasRemovedChangesNothing: a copy
// in flight when its destination is removed still lands in its folder,
// and its outcome lists no copy.
func TestACopyFinishingAfterItsDestinationWasRemovedChangesNothing(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	e := w.taken(archive.Scheduled)
	d := w.add()
	tr := w.only()
	if err := w.backups.Remove(t.Context(), d.ID); err != nil {
		t.Fatal(err)
	}
	w.settle()
	if f := w.backups.Folders(w.vps); len(f) != 1 || f[0].Folder != vpsFolder {
		t.Fatalf("vps keeps %+v while the copy runs, want its folder", f)
	}
	w.arrive(w.vps, tr)
	w.settle()
	if got := w.entry(e.ID); len(got.Copies) != 0 {
		t.Fatalf("the Backup lists %+v", got.Copies)
	}
	if l := w.eng.Index().Read(); len(l.Destinations) != 0 || len(w.records()) != 0 {
		t.Fatalf("the index lists %d destinations and the library %d Transfers", len(l.Destinations), len(w.records()))
	}
}

// TestAnOfflineDestinationCatchesUp: a destination's unit is tied to its
// Node, so it runs once the Node returns, and that run queues every
// Backup taken while it was away, Low.
func TestAnOfflineDestinationCatchesUp(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	d := w.add()
	var units []tasks.Unit
	if err := w.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		units, err = w.backups.Task().Units(ctx, rd)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(units) != 2 || units[1].Place != tasks.TiedTo(w.vps) || units[1].Label != vpsFolder+" on vps" {
		t.Fatalf("the Backup task's units are %+v", units)
	}

	first := w.taken(archive.Scheduled)
	second := w.taken(archive.Manual)
	w.wg.Wait()
	w.queue(d)
	rs := w.records()
	if len(rs) != 2 {
		t.Fatalf("vps's return queued %d copies, want 2", len(rs))
	}
	for _, tr := range rs {
		if tr.Priority != transfers.Low || tr.Destination != (transfers.BackupDestination{ID: d.ID, Node: w.vps, Folder: vpsFolder}) {
			t.Errorf("a copy is %s to %+v", tr.Priority, tr.Destination)
		}
	}
	for _, id := range []archive.ID{first.ID, second.ID} {
		if c := w.copyOf(id, d); c.State != backups.Copying {
			t.Errorf("a copy is %s", c.State)
		}
	}
}

// TestAPulledCopyIsCheckedAgainstTheIndex: a Backup pulled back from vps
// is Local only when its BLAKE3 is the index's. One that isn't is deleted
// and the pull fails.
func TestAPulledCopyIsCheckedAgainstTheIndex(t *testing.T) {
	w, e, d := copied(t)
	path := w.backups.Path(e)
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		arrive []byte
		local  bool
	}{
		{"a damaged copy", bytes.Repeat([]byte{1}, len(good)), false},
		{"the copy", good, true},
	} {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			t.Fatal(err)
		}
		w.change(e.ID, func(e *backups.Entry) { e.Local, e.Pull = false, nil })
		if err := w.backups.Pull(t.Context(), e.ID, d.ID); err != nil {
			t.Fatal(err)
		}
		tr := w.only()
		if want := (transfers.BackupsFolder{Node: w.home, Folder: w.folder}); tr.Source != w.vps || tr.Destination != want {
			t.Fatalf("%s: the pull runs from %v to %+v", tc.name, tr.Source, tr.Destination)
		}
		if err := os.WriteFile(path, tc.arrive, 0o600); err != nil {
			t.Fatal(err)
		}
		w.arrive(w.home, tr)
		w.settle()
		got := w.entry(e.ID)
		_, statErr := os.Stat(path)
		if got.Local != tc.local || (statErr == nil) != tc.local || pullFailed(got) == tc.local {
			t.Fatalf("%s: Local %v with the file there %v, the pull %+v", tc.name, got.Local, statErr == nil, got.Pull)
		}
	}
}

// TestACopyWhoseTransferTheLibraryDoesntKnowWaits: a copying copy whose
// Transfer the library doesn't hold, as after restoring an older library,
// waits, and its destination's next run queues it again.
func TestACopyWhoseTransferTheLibraryDoesntKnowWaits(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	e := w.taken(archive.Scheduled)
	d := w.add()
	tr := w.only()
	if err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := w.tr.Want(ctx, tx, transfers.Request{Purpose: transfers.Backup, Job: tr.Job})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	w.settle()
	if c := w.copyOf(e.ID, d); c.State != backups.CopyWaiting {
		t.Fatalf("the copy is %s, want waiting", c.State)
	}
	w.queue(d)
	if c := w.copyOf(e.ID, d); c.State != backups.Copying || len(w.records()) != 1 {
		t.Fatalf("the copy is %s with %d Transfers", c.State, len(w.records()))
	}
}

// TestADestinationWarnsWhenItsCopiesFailOrGoStale: a destination's item
// is quiet while its newest copy is fresh, warns once a check finds a
// mismatch or the newest copy is older than the threshold, and is unsure
// while its Node is offline.
func TestADestinationWarnsWhenItsCopiesFailOrGoStale(t *testing.T) {
	w, e, d := copied(t)
	names := map[role.NodeID]string{w.home: "home", w.vps: "vps"}
	up := func(role.NodeID) bool { return true }
	stale := 3 * 24 * time.Hour
	if a := w.backups.DestinationHealth(w.now(), stale, names, up); len(a.Items) != 0 || a.Next.IsZero() {
		t.Fatalf("a fresh copy reads %+v", a)
	}
	a := w.backups.DestinationHealth(w.now().Add(4*24*time.Hour), stale, names, up)
	if len(a.Items) != 1 || a.Items[0].Subject != "Backup copies in "+vpsFolder+" on vps" || a.Items[0].Text != "The newest copy is 4 days old." {
		t.Fatalf("a stale copy reads %+v", a.Items)
	}
	w.change(e.ID, func(e *backups.Entry) {
		c, _ := e.Copy(d.ID)
		c.Check.Result = backups.CheckMismatch
		e.Copies = []backups.Copy{c}
	})
	if a := w.backups.DestinationHealth(w.now(), stale, names, up); len(a.Items) != 1 || a.Items[0].Key != backups.DestinationKey(d.ID) {
		t.Fatalf("a mismatched copy reads %+v", a.Items)
	}
	if a := w.backups.DestinationHealth(w.now(), stale, names, func(role.NodeID) bool { return false }); len(a.Items) != 0 || len(a.Unsure) != 1 || a.Unsure[0].Text != "vps is offline." {
		t.Fatalf("an offline destination reads %+v", a)
	}
}

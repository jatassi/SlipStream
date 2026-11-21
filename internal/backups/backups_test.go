package backups_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

func at(day int) time.Time { return time.Date(2026, 11, day, 4, 0, 0, 0, time.UTC) }

func entryOf(n byte, trigger archive.Trigger, taken time.Time, state backups.EntryState) backups.Entry {
	return backups.Entry{ID: archive.ID{n}, Trigger: trigger, TakenAt: taken, State: state}
}

func pruned(l backups.Ledger, r backups.Retention) []byte {
	var out []byte
	for _, id := range backups.Prune(l, r) {
		out = append(out, id[0])
	}
	slices.Sort(out)
	return out
}

func TestRetentionPerKind(t *testing.T) {
	l := backups.Ledger{Backups: []backups.Entry{
		entryOf(1, archive.Scheduled, at(20), backups.Verified),
		entryOf(2, archive.Scheduled, at(13), backups.Verified),
		entryOf(3, archive.Scheduled, at(12), backups.Verified),
		entryOf(4, archive.Manual, at(1), backups.Verified),
		entryOf(5, archive.Update, at(19), backups.Verified),
		entryOf(6, archive.Update, at(18), backups.Verified),
		entryOf(7, archive.Update, at(17), backups.Verified),
		entryOf(8, archive.BeforeRestore, at(16), backups.Verified),
		entryOf(9, archive.BeforeRestore, at(15), backups.Verified),
	}}
	got := pruned(l, backups.Retention{KeepScheduled: 7 * 24 * time.Hour, KeepEach: 1})
	if want := []byte{3, 6, 7, 9}; !slices.Equal(got, want) {
		t.Fatalf("pruned %v, want %v: scheduled older than 7 days before the newest scheduled, and each other kind beyond its newest; never a manual one", got, want)
	}
	newest := backups.Ledger{Backups: []backups.Entry{entryOf(1, archive.Update, at(20), backups.Verified), entryOf(2, archive.Scheduled, at(1), backups.Verified)}}
	if got := pruned(newest, backups.Retention{KeepScheduled: 24 * time.Hour, KeepEach: 1}); len(got) != 0 {
		t.Fatalf("pruned %v of the newest of each kind", got)
	}
}

func TestAFailedBackupCountsForNothing(t *testing.T) {
	l := backups.Ledger{Backups: []backups.Entry{
		entryOf(1, archive.Scheduled, at(30), backups.Failed),
		entryOf(2, archive.Update, at(29), backups.Failed),
		entryOf(3, archive.Update, at(28), backups.Taking),
		entryOf(4, archive.Update, at(20), backups.Verified),
		entryOf(5, archive.Scheduled, at(20), backups.Verified),
		entryOf(6, archive.Scheduled, at(14), backups.Verified),
	}}
	if got := pruned(l, backups.Retention{KeepScheduled: 7 * 24 * time.Hour, KeepEach: 1}); len(got) != 0 {
		t.Fatalf("pruned %v: a failed or unchecked Backup moved the horizon or took a place", got)
	}

	w := newWorld(t)
	w.opened("2.1.0")
	w.free.Store(0)
	if _, err := w.take(backups.Request{Trigger: archive.Manual}); err == nil {
		t.Fatal("a Backup with no room passed")
	}
	w.free.Store(1 << 40)
	w.taken(archive.Manual)
	if got := w.ledger().Backups; len(got) != 1 || got[0].State != backups.Verified {
		t.Fatalf("after a failed run and a good one the index lists %v, want only the good one", w.listed())
	}
}

func TestRetentionAfterACrashFinishesDeleting(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	gone := w.taken(archive.Scheduled)
	kept := w.taken(archive.Scheduled)
	w.rewrite(gone.ID, func(e *backups.Entry) { e.State = backups.Deleting })
	w.opened("2.1.0")
	if got := w.ledger().Backups; len(got) != 1 || got[0].ID != kept.ID {
		t.Fatalf("after the start the index lists %v, want only the Backup kept", w.listed())
	}
	if got := w.files(); !slices.Equal(got, []string{kept.Name.String()}) {
		t.Fatalf("the folder holds %v, want only %s", got, kept.Name)
	}
}

func TestRecoverChecksATakingEntryWithItsFile(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	good := w.taken(archive.Scheduled)
	damaged := w.taken(archive.Scheduled)
	for _, id := range []archive.ID{good.ID, damaged.ID} {
		w.rewrite(id, func(e *backups.Entry) {
			*e = backups.Entry{ID: e.ID, Name: e.Name, TakenAt: e.TakenAt, Trigger: e.Trigger, Version: e.Version, Generation: e.Generation, State: backups.Taking}
		})
	}
	path := filepath.Join(w.folder, damaged.Name.String())
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b[:len(b)/2], 0o600); err != nil {
		t.Fatal(err)
	}
	w.opened("2.1.0")
	got, _ := w.ledger().Entry(good.ID)
	if got.State != backups.Verified || got.Sum != good.Sum || got.Size != good.Size || !got.Local {
		t.Fatalf("the Backup whose file is whole is %+v, want verified with its sum %x", got, good.Sum)
	}
	if bad, _ := w.ledger().Entry(damaged.ID); bad.State != backups.Failed || bad.Cause == "" {
		t.Fatalf("the cut Backup is %+v, want failed with its cause", bad)
	}
	w.taken(archive.Manual)
	if _, listed := w.ledger().Entry(damaged.ID); listed || slices.Contains(w.files(), damaged.Name.String()) {
		t.Fatalf("the failed Backup is still listed or on disk after the next run: %v, %v", w.listed(), w.files())
	}
}

func TestRecoverDropsATakingEntryWithOnlyAPart(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	e := w.taken(archive.Scheduled)
	final := filepath.Join(w.folder, e.Name.String())
	if err := os.Rename(final, final+".part"); err != nil {
		t.Fatal(err)
	}
	w.rewrite(e.ID, func(e *backups.Entry) { e.State = backups.Taking })
	if err := os.WriteFile(filepath.Join(w.dir, backups.SnapshotFile), []byte("left"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.opened("2.1.0")
	if got := w.listed(); len(got) != 0 {
		t.Fatalf("the index lists %v, want the Backup that never finished dropped", got)
	}
	if got := w.files(); len(got) != 0 {
		t.Fatalf("the folder holds %v, want the .part deleted", got)
	}
	if _, err := os.Stat(filepath.Join(w.dir, backups.SnapshotFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the snapshot is left: %v", err)
	}
}

func TestAPlantedFileIsNeverListed(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	taken := w.taken(archive.Scheduled)
	b, err := os.ReadFile(filepath.Join(w.folder, taken.Name.String()))
	if err != nil {
		t.Fatal(err)
	}
	planted := "slipstream-20261101T040000Z-2.0.0-scheduled-0000.tar.zst.age"
	if err := os.WriteFile(filepath.Join(w.folder, planted), b, 0o600); err != nil {
		t.Fatal(err)
	}
	w.opened("2.1.0")
	w.taken(archive.Scheduled)
	for _, e := range w.ledger().Backups {
		if e.Name.String() == planted {
			t.Fatalf("the planted file is listed: %v", w.listed())
		}
	}
	if !slices.Contains(w.files(), planted) {
		t.Fatal("the planted file was deleted")
	}
}

func TestOneBackupRunsAtATime(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	w.parallel = true
	first, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	w.freeHook = func() {
		if calls.Add(1) == 1 {
			close(first)
			<-release
		}
	}
	runA, err := w.backups.Take(t.Context(), backups.Request{Trigger: archive.Manual})
	if err != nil {
		t.Fatal(err)
	}
	<-first
	runB, err := w.backups.Take(t.Context(), backups.Request{Trigger: archive.Manual})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if n := calls.Load(); n != 1 {
		t.Fatalf("a second Backup read the free space %d times while the first ran", n-1)
	}
	close(release)
	for _, run := range []*backups.Run{runA, runB} {
		if _, err := run.Wait(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(w.ledger().Backups); got != 2 {
		t.Fatalf("listed %d Backups, want 2", got)
	}
}

func TestBackUpNowCarriesWhoPressedIt(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	admin := storage.NewUUID()
	e, err := w.take(backups.Request{Trigger: archive.Manual, By: admin})
	if err != nil {
		t.Fatal(err)
	}
	listed, _ := w.ledger().Entry(e.ID)
	if listed.By != admin.String() || listed.Trigger != archive.Manual {
		t.Fatalf("the Backup lists by %q and trigger %s, want %s and manual", listed.By, listed.Trigger, admin)
	}
	scheduled := w.taken(archive.Scheduled)
	if listed, _ := w.ledger().Entry(scheduled.ID); listed.By != "" {
		t.Fatalf("a Backup nobody pressed lists by %q", listed.By)
	}
}

func TestTakeIsATaskRunThatWaits(t *testing.T) {
	w := newWorld(t)
	w.opened("2.1.0")
	e, err := w.take(backups.Request{Trigger: archive.BeforeImport})
	if err != nil {
		t.Fatal(err)
	}
	want := tasks.Start{Task: backups.TaskID, Units: []tasks.UnitKey{"backup"}, Trigger: tasks.ByEvent}
	if len(w.starts) != 1 || !slices.Equal(w.starts[0].Units, want.Units) || w.starts[0].Task != want.Task || w.starts[0].Trigger != want.Trigger {
		t.Fatalf("Take started %+v, want %+v", w.starts, want)
	}
	if e.State != backups.Verified || e.Trigger != archive.BeforeImport {
		t.Fatalf("Wait answered %+v, want a verified before-import Backup", e)
	}
	if _, err := os.Stat(filepath.Join(w.folder, e.Name.String())); err != nil {
		t.Fatalf("the Backup's file: %v", err)
	}
	if got := entrytest.Of(w.history, backups.FailedRow); len(got) != 0 {
		t.Fatalf("a good Backup recorded %v", got)
	}

	w.free.Store(1 << 10)
	_, err = w.take(backups.Request{Trigger: archive.BeforeImport})
	var space *backups.SpaceError
	if !errors.As(err, &space) || space.Free != 1<<10 || space.Need <= space.Free {
		t.Fatalf("Wait on a full disk = %v, want the folder's sizes", err)
	}
	if got := entrytest.Of(w.history, backups.FailedRow); len(got) != 1 || got[0].Trigger != archive.BeforeImport {
		t.Fatalf("the failed run recorded %v, want one Backup failed", got)
	}
}

func TestANewerVersionWithNoMigrationBacksUpOnce(t *testing.T) {
	w := newWorld(t)
	w.opened("2.0.0")
	if got := w.listed(); len(got) != 0 {
		t.Fatalf("the first start backed up: %v", got)
	}
	w.opened("2.1.0")
	w.opened("2.1.0")
	ledger := w.ledger()
	if len(ledger.Backups) != 1 || ledger.Backups[0].Trigger != archive.Update || ledger.Backups[0].Version != "2.1.0" {
		t.Fatalf("after two starts of 2.1.0 the index lists %v, want one update Backup", w.listed())
	}
	if ledger.StartedVersion != "2.1.0" {
		t.Fatalf("started version %q", ledger.StartedVersion)
	}
	w.opened("2.0.5")
	if got := len(w.ledger().Backups); got != 1 {
		t.Fatalf("an older version backed up: %v", w.listed())
	}
}

func TestDevNeverTakesAnUpdateBackup(t *testing.T) {
	w := newWorld(t)
	w.opened("2.0.0")
	w.opened("dev")
	w.opened("2.1.0")
	if got := w.listed(); len(got) != 0 {
		t.Fatalf("the index lists %v, want no update Backup across a dev start", got)
	}
}

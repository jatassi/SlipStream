package node

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

const markerV1 = "-- +goose Up\nCREATE TABLE marker (id INTEGER PRIMARY KEY, value TEXT NOT NULL) STRICT;\n"

// markerAt is the marker set with a second migration, which fails when
// fails is set.
func markerAt(fails bool) storage.Schema[storage.Library] {
	second := "-- +goose Up\nCREATE TABLE marker_two (id INTEGER PRIMARY KEY) STRICT;\n"
	if fails {
		second = "-- +goose Up\nCREATE TABLE marker (id INTEGER PRIMARY KEY) STRICT;\n"
	}
	return storage.NewSchema[storage.Library]("marker", fstest.MapFS{
		"schema/00001_marker.sql": {Data: []byte(markerV1)},
		"schema/00002_marker.sql": {Data: []byte(second)},
	})
}

// oldLibrary is a data dir whose library.db was set up before the marker
// set's second migration, as a Node of the previous version left it.
func oldLibrary(t *testing.T) Env {
	t.Helper()
	env := markedEnv(t)
	if err := openMarked(t, env, env.DataDir, "old").close(t.Context()); err != nil {
		t.Fatal(err)
	}
	return env
}

// freeDisk is the OS disk with every volume reading free bytes free, and
// each Volume call first waiting on gate when it is set.
type freeDisk struct {
	storing.Disk
	free int64
	gate chan struct{}
}

func (d freeDisk) Volume(ctx context.Context, path string) (storing.Volume, error) {
	if d.gate != nil {
		<-d.gate
	}
	v, err := d.Disk.Volume(ctx, path)
	v.Free = d.free
	return v, err
}

func withDisk(env *Env, free int64, gate chan struct{}) {
	env.Disk = func(data string) (storing.Disk, error) {
		d, err := OSDisk(data)
		return freeDisk{Disk: d, free: free, gate: gate}, err
	}
}

func ledgerOf(t *testing.T, env Env) backups.Ledger {
	t.Helper()
	x, err := backups.OpenIndex(filepath.Join(env.DataDir.Path(), backups.IndexFile), 0)
	if err != nil {
		t.Fatal(err)
	}
	return x.Read()
}

func markerVersion(t *testing.T, env Env) int64 {
	t.Helper()
	got, err := storage.Inspect(t.Context(), env.DataDir.LibraryDB(), append(LibrarySchemas(env.Types), markerAt(false)))
	if err != nil {
		t.Fatal(err)
	}
	return got.Versions["marker"]
}

func TestTheHooksBackupIsListedAfterACrashPastTheMigration(t *testing.T) {
	env := oldLibrary(t)
	env.Test.Library = []storage.Schema[storage.Library]{markerAt(true)}
	if _, err := Start(t.Context(), env); err == nil || ExitCode(err) != ExitFailed {
		t.Fatalf("start with a failing migration = %v, want it to fail with 1", err)
	}
	l := ledgerOf(t, env)
	if len(l.Backups) != 1 || l.Backups[0].State != backups.Verified || l.Backups[0].Trigger != archive.Update || l.Backups[0].Schemas["marker"] != 1 {
		t.Fatalf("after the stopped migration the index lists %+v, want one verified update Backup of marker 1", l.Backups)
	}
	if _, err := os.Stat(filepath.Join(env.DataDir.Resolve("backups"), l.Backups[0].Name.String())); err != nil {
		t.Fatalf("the Backup's file: %v", err)
	}
}

func TestAFullDiskExitsSevenAndMigratesNothing(t *testing.T) {
	env := oldLibrary(t)
	env.Test.Library = []storage.Schema[storage.Library]{markerAt(false)}
	withDisk(&env, 1024, nil)
	_, err := Start(t.Context(), env)
	if got := ExitCode(err); got != ExitBackupFailed {
		t.Fatalf("start on a full disk = %v, exit %d; want %d", err, got, ExitBackupFailed)
	}
	if !strings.Contains(err.Error(), "back up library.db: "+env.DataDir.Path()+" has 1.0 KB free, and the Backup needs about ") {
		t.Fatalf("the error is %q, want the folder and both sizes", err)
	}
	if got := markerVersion(t, env); got != 1 {
		t.Fatalf("marker is at %d after the refusal, want 1", got)
	}
	if got := ledgerOf(t, env).Backups; len(got) != 0 {
		t.Fatalf("the refusal listed %+v", got)
	}
}

func TestTheCLIAddsTheSpaceHint(t *testing.T) {
	env, stderr := testEnv(t)
	env.DataDir = oldLibrary(t).DataDir
	env.Test.Library = []storage.Schema[storage.Library]{markerAt(false)}
	withDisk(&env, 1024, nil)
	if _, err := Start(t.Context(), env); err == nil {
		t.Fatal("start on a full disk passed")
	}
	if want := "Free space in " + env.DataDir.Path() + ", or start the previous version of SlipStream.\n"; !strings.Contains(stderr.String(), want) {
		t.Fatalf("stderr is %q, want %q", stderr, want)
	}

	env, stderr = testEnv(t)
	env.DataDir = oldLibrary(t).DataDir
	env.Test.Library = []storage.Schema[storage.Library]{markerAt(false)}
	writeFile(t, env.DataDir.Resolve("backups"), "a file where the Backups folder goes")
	_, err := Start(t.Context(), env)
	if got := ExitCode(err); got != ExitBackupFailed {
		t.Fatalf("start with no Backups folder = %v, exit %d; want %d", err, got, ExitBackupFailed)
	}
	if want := "Start the previous version of SlipStream, then check the log for the cause.\n"; !strings.Contains(stderr.String(), want) || strings.Contains(stderr.String(), "Free space") {
		t.Fatalf("stderr is %q, want only %q", stderr, want)
	}
}

func TestALibraryWithNoMigrationTakesNoHookBackup(t *testing.T) {
	env := oldLibrary(t)
	startTest(t, env).Stop()
	if got := ledgerOf(t, env).Backups; len(got) != 0 {
		t.Fatalf("a start with nothing to migrate listed %+v", got)
	}
}

func TestAForeignLibraryKeyRefusesWithFive(t *testing.T) {
	env := oldLibrary(t)
	other := scratchDir(t)
	if err := openMarked(t, env, other, "other").close(t.Context()); err != nil {
		t.Fatal(err)
	}
	copyFile(t, other.LibraryKey(), env.DataDir.LibraryKey())
	env.Test.Library = []storage.Schema[storage.Library]{markerAt(false)}
	_, err := Start(t.Context(), env)
	if got := ExitCode(err); got != ExitLibraryRefused {
		t.Fatalf("start with a foreign library.key = %v, exit %d; want %d", err, got, ExitLibraryRefused)
	}
	if got := markerVersion(t, env); got != 1 {
		t.Fatalf("marker is at %d after the refusal, want 1", got)
	}
	if _, err := os.Stat(filepath.Join(env.DataDir.Path(), backups.IndexFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the refusal wrote backups.json: %v", err)
	}
}

func TestStatusAnswersBackingUpDuringTheHook(t *testing.T) {
	env := oldLibrary(t)
	env.Test.Library = []storage.Schema[storage.Library]{markerAt(false)}
	gate := make(chan struct{})
	withDisk(&env, 1<<40, gate)
	started := make(chan error, 1)
	var n *Node
	go func() {
		var err error
		n, err = Start(t.Context(), env)
		started <- err
	}()
	deadline := time.Now().Add(30 * time.Second)
	for {
		reply, err := Call(t.Context(), env.Net, env.DataDir, StatusOp, StatusRequest{})
		if err == nil && reply.Phase == PhaseBackingUp {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("status never answered backing up: %+v, %v", reply, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	close(gate)
	if err := <-started; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		n.Stop()
		<-n.Done()
	})
	if reply, err := Call(t.Context(), env.Net, env.DataDir, StatusOp, StatusRequest{}); err != nil || reply.Phase != PhaseReady {
		t.Fatalf("status after the start = %+v, %v; want ready", reply, err)
	}
	if got := markerVersion(t, env); got != 2 {
		t.Fatalf("marker is at %d once the Backup passed, want 2", got)
	}
}

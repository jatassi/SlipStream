package lifecycle_test

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/lifecycle"
)

// names is every file the swap can find or make.
func names() []string {
	var all []string
	for _, f := range lifecycle.SwapFiles {
		all = append(all, f, f+lifecycle.Restored, f+lifecycle.Aside)
	}
	for _, s := range []string{"-wal", "-shm"} {
		all = append(all, "library.db"+s, "library.db"+lifecycle.Aside+s)
	}
	return all
}

func apply(t *testing.T, files map[string]bool, s lifecycle.Step) {
	t.Helper()
	if !files[s.From] {
		t.Fatalf("step %v renames a file that isn't there: %v", s, files)
	}
	delete(files, s.From)
	files[s.To] = true
}

// starts is every set of live, restored and sidecar files a swap can
// begin from.
func starts() []map[string]bool {
	var start []string
	for _, f := range lifecycle.SwapFiles {
		start = append(start, f, f+lifecycle.Restored)
	}
	start = append(start, "library.db-wal", "library.db-shm")
	var all []map[string]bool
	for mask := range 1 << len(start) {
		files := map[string]bool{}
		for i, name := range start {
			if mask&(1<<i) != 0 {
				files[name] = true
			}
		}
		all = append(all, files)
	}
	return all
}

func TestPlanDerivesTheRestFromAnyPrefix(t *testing.T) {
	for _, files := range starts() {
		plan := func() []lifecycle.Step { return lifecycle.Plan(func(name string) bool { return files[name] }) }
		full := plan()
		for k, s := range full {
			if rest := plan(); !slices.Equal(rest, full[k:]) {
				t.Fatalf("after %d of %v, Plan = %v; want %v", k, full, rest, full[k:])
			}
			if s.From == "library.db"+lifecycle.Restored && (files["library.db-wal"] || files["library.db-shm"] || files["library.db"]) {
				t.Fatalf("library.db.restore moves in beside %v", files)
			}
			apply(t, files, s)
		}
		if rest := plan(); len(rest) != 0 {
			t.Fatalf("after the whole plan, Plan = %v", rest)
		}
		for _, f := range lifecycle.SwapFiles {
			if files[f+lifecycle.Restored] {
				t.Fatalf("%s.restore is still there after the swap: %v", f, files)
			}
		}
	}
}

func TestPlanMovesTheLiveFilesAsideAndTheRestoredIn(t *testing.T) {
	files := map[string]bool{"library.key": true, "library.key.restore": true, "library.db": true, "library.db-wal": true, "library.db-shm": true, "library.db.restore": true, "backups.json": true}
	got := lifecycle.Plan(func(name string) bool { return files[name] })
	want := []lifecycle.Step{
		{From: "library.key", To: "library.key.pre-restore"},
		{From: "library.key.restore", To: "library.key"},
		{From: "library.db-wal", To: "library.db.pre-restore-wal"},
		{From: "library.db-shm", To: "library.db.pre-restore-shm"},
		{From: "library.db", To: "library.db.pre-restore"},
		{From: "library.db.restore", To: "library.db"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Plan = %v, want %v", got, want)
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// contents is every file in dir with its body.
func contents(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		got[e.Name()] = string(b)
	}
	return got
}

var errCrash = errors.New("crash")

func TestASwapStoppedBeforeAnyStepFinishesNextTime(t *testing.T) {
	stage := func(dir string) {
		for _, f := range []string{"library.key", "backup.key", "library.db", "library.db-wal"} {
			write(t, dir, f, "old "+f)
		}
		for _, f := range lifecycle.SwapFiles {
			write(t, dir, f+lifecycle.Restored, "restored "+f)
		}
	}
	want := map[string]string{
		"library.key": "restored library.key", "library.key.pre-restore": "old library.key",
		"backup.key": "restored backup.key", "backup.key.pre-restore": "old backup.key",
		"backups.json": "restored backups.json",
		"library.db":   "restored library.db", "library.db.pre-restore": "old library.db",
		"library.db.pre-restore-wal": "old library.db-wal",
	}
	whole := t.TempDir()
	stage(whole)
	var steps int
	if err := (lifecycle.Dir{Path: whole}).Swap(func(lifecycle.Step) error { steps++; return nil }); err != nil {
		t.Fatal(err)
	}
	if got := contents(t, whole); !maps.Equal(got, want) {
		t.Fatalf("after the swap, dir = %v; want %v", got, want)
	}
	for stop := range steps {
		t.Run(strconv.Itoa(stop), func(t *testing.T) {
			d := lifecycle.Dir{Path: t.TempDir()}
			stage(d.Path)
			var i int
			err := d.Swap(func(lifecycle.Step) error {
				if i == stop {
					return errCrash
				}
				i++
				return nil
			})
			if !errors.Is(err, errCrash) {
				t.Fatalf("swap = %v, want it stopped", err)
			}
			if err := d.Swap(func(lifecycle.Step) error { return nil }); err != nil {
				t.Fatal(err)
			}
			if got := contents(t, d.Path); !maps.Equal(got, want) {
				t.Fatalf("after a swap stopped before step %d, dir = %v; want %v", stop, got, want)
			}
		})
	}
}

func TestAsideAndLeftoversDeleteOnlyTheirFiles(t *testing.T) {
	d := lifecycle.Dir{Path: t.TempDir()}
	for _, name := range names() {
		write(t, d.Path, name, name)
	}
	write(t, d.Path, "library.db.restore-wal", "")
	if err := d.Aside(); err != nil {
		t.Fatal(err)
	}
	if err := d.Leftovers(); err != nil {
		t.Fatal(err)
	}
	got := slices.Sorted(maps.Keys(contents(t, d.Path)))
	want := []string{"backup.key", "backups.json", "library.db", "library.db-shm", "library.db-wal", "library.key"}
	if !slices.Equal(got, want) {
		t.Fatalf("dir = %v, want %v", got, want)
	}
}

func restore() *lifecycle.Restore {
	return &lifecycle.Restore{
		ID:          "r1",
		Stage:       lifecycle.Staged,
		Cause:       lifecycle.ByRestore,
		Backup:      lifecycle.Backup{Name: "slipstream-20261114T040012Z-2.1.0-scheduled-7k3m.tar.zst.age", TakenAt: time.Date(2026, 11, 14, 4, 0, 12, 0, time.UTC), Version: "2.1.0", ControllerKey: []byte{1, 2}},
		NewMachine:  true,
		Remove:      []string{"node-1"},
		RemovedKeys: [][]byte{{3}},
		Typed:       true,
		Superseded:  []lifecycle.Proof{{OldKey: []byte{1, 2}, Proof: []byte{4}}},
		Adopt:       "restore.part",
	}
}

func TestARecordRoundTripsUntilFinished(t *testing.T) {
	d := lifecycle.Dir{Path: t.TempDir()}
	if _, ok, err := d.Read(); ok || err != nil {
		t.Fatalf("read with no record = %v, %v", ok, err)
	}
	r := restore()
	if err := d.Begin(r); err != nil {
		t.Fatal(err)
	}
	got, ok, err := d.Read()
	if err != nil || !ok || !reflect.DeepEqual(got, r) {
		t.Fatalf("read = %+v, %v, %v; want %+v", got, ok, err, r)
	}
	r.Stage = lifecycle.Swapped
	if err := d.Advance(r); err != nil {
		t.Fatal(err)
	}
	if got, _, err := d.Read(); err != nil || !reflect.DeepEqual(got, r) {
		t.Fatalf("read after Advance = %+v, %v; want %+v", got, err, r)
	}
	if err := d.Finish(); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := d.Read(); ok || err != nil {
		t.Fatalf("read after Finish = %v, %v; want no record", ok, err)
	}
	w := &lifecycle.Wipe{Stage: lifecycle.Deleting, JoinCode: "ABCD-EFGH-JKLM"}
	if err := d.Begin(w); err != nil {
		t.Fatal(err)
	}
	if got, _, err := d.Read(); err != nil || !reflect.DeepEqual(got, w) {
		t.Fatalf("read the wipe = %+v, %v; want %+v", got, err, w)
	}
	if info, err := os.Stat(filepath.Join(d.Path, lifecycle.FileName)); err != nil || (os.PathSeparator == '/' && info.Mode().Perm() != 0o600) {
		t.Fatalf("record stat = %v, %v; want mode 0600", info, err)
	}
}

func TestABegunRecordRefusesASecondOp(t *testing.T) {
	d := lifecycle.Dir{Path: t.TempDir()}
	if err := d.Begin(restore()); err != nil {
		t.Fatal(err)
	}
	if err := d.Begin(&lifecycle.Wipe{Stage: lifecycle.Deleting}); !errors.Is(err, lifecycle.ErrBusy) {
		t.Fatalf("second Begin = %v, want ErrBusy", err)
	}
	if got, _, err := d.Read(); err != nil || !reflect.DeepEqual(got, restore()) {
		t.Fatalf("record after a refused Begin = %+v, %v; want the first", got, err)
	}
}

func TestARecordThisBinaryCantResumeIsRefused(t *testing.T) {
	for name, body := range map[string]string{
		"newer format":      `{"format": 2, "op": "update", "update": {}}`,
		"unknown op":        `{"format": 1, "op": "handover", "handover": {}}`,
		"unknown field":     `{"format": 1, "op": "wipe", "wipe": {"stage": "deleting", "joinCode": "", "address": "x"}}`,
		"unknown stage":     `{"format": 1, "op": "wipe", "wipe": {"stage": "backed-up", "joinCode": ""}}`,
		"another op's body": `{"format": 1, "op": "wipe", "restore": {}}`,
		"not json":          `{"format": 1,`,
		"no format":         `{"op": "wipe", "wipe": {"stage": "deleting", "joinCode": ""}}`,
		"empty":             ``,
	} {
		t.Run(name, func(t *testing.T) {
			d := lifecycle.Dir{Path: t.TempDir()}
			write(t, d.Path, lifecycle.FileName, body)
			if _, _, err := d.Read(); !errors.As(err, new(*lifecycle.RefusedError)) {
				t.Fatalf("read %s = %v, want a *lifecycle.RefusedError", body, err)
			}
		})
	}
}

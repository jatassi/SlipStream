package files_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/housekeeping"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// keepDefault is Keep history for at its default.
type keepDefault struct{}

func (keepDefault) Fields(context.Context) ([]setting.Field, error) { return nil, nil }
func (keepDefault) Apply(context.Context, setting.Patch) error      { return nil }
func (keepDefault) Keep(context.Context) (time.Duration, error)     { return history.KeepFor.Default, nil }

// pastKeep moves the clock past Keep history for and runs svc's Import
// records step one record a batch.
func pastKeep(t *testing.T, svc *files.Files) {
	t.Helper()
	time.Sleep(history.KeepFor.Default + 24*time.Hour)
	task := housekeeping.Task(housekeeping.Config{Reconciling: func(context.Context) (bool, error) { return false, nil }, Keep: keepDefault{}, Batch: 1, Steps: []housekeeping.Step{svc.Retention()}})
	if err := task.Run(t.Context(), tasks.Unit{Key: "import-records"}, &tasks.Progress{}); err != nil {
		t.Fatal(err)
	}
}

func (l *library) records(t *testing.T) int {
	t.Helper()
	var n int
	if err := l.lib.QueryRowContext(t.Context(), `SELECT count(*) FROM files_imported`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestAFileKeepsHowItWasPlacedAfterItsImportRecordGoes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := open(t)
		hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
		p := l.profile(t, "HD", hd)
		tree := l.work(t, "w1", true, &p)
		svc := l.files(files.NoGrabs{})
		for item := range 2 {
			c := holdingOf(tree, item, files.HoldingOf(storage.NewUUID(), ".mkv"), 5, read1080, release.WEBDL1080p)
			if _, err := l.importOne(t, svc, files.Batch{Work: tree.ID, Root: l.root.ID, Import: storage.NewUUID()}, c); err != nil {
				t.Fatal(err)
			}
		}
		recorded := l.records(t)
		pastKeep(t, svc)

		if n := l.records(t); recorded != 2 || n != 0 {
			t.Fatalf("%d import records are left of %d, want every one past Keep history for gone, across batches", n, recorded)
		}
		listing, err := svc.WorkFiles(t.Context(), admin, tree.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(listing.Files) != 2 {
			t.Fatalf("the Work lists %d files, want its 2", len(listing.Files))
		}
		for _, f := range listing.Files {
			if f.PlacedBy != "copy" {
				t.Errorf("%s lists placed by %q after its import record went, want copy", f.Path, f.PlacedBy)
			}
		}
	})
}

// cellAt is the cell of tree's first Item at target.
func (l *library) cellAt(t *testing.T, svc *files.Files, tree works.Tree, target targets.TargetID) files.Cell {
	t.Helper()
	for _, c := range l.cellsOf(t, svc, tree.ID) {
		if c.Item.ID == tree.Items[0].ID && c.Target.ID == target {
			return c
		}
	}
	t.Fatal("the Work has no cell of its first Item at the Target")
	return files.Cell{}
}

func TestAForcedFillIsntTakenByItsOwnImportAfterItsImportRecordGoes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := open(t)
		hd := l.target(t, "HD", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
		p := l.profile(t, "HD", hd)
		tree := l.work(t, "w1", true, &p)
		svc := l.files(files.NoGrabs{})
		cell := func() files.Cell { return l.cellAt(t, svc, tree, hd.ID) }
		c := cell()
		key := files.CellKey{Item: c.Item.ID, Edition: c.Edition, Target: c.Target.ID}
		forcing := files.Batch{Work: tree.ID, Root: l.root.ID, Import: storage.NewUUID()}
		web := holdingOf(tree, 0, files.HoldingOf(storage.NewUUID(), ".mkv"), 5, read1080, release.WEBDL1080p)
		web.Force = []files.CellKey{key}
		forced, err := l.importOne(t, svc, forcing, web)
		if err != nil {
			t.Fatal(err)
		}
		pastKeep(t, svc)
		if n := l.records(t); n != 0 {
			t.Fatalf("%d import records are left, want none", n)
		}

		bluray := func() files.Candidate {
			return holdingOf(tree, 0, files.HoldingOf(storage.NewUUID(), ".mkv"), 5, read1080, release.Bluray1080p)
		}
		if _, err := l.importOne(t, svc, forcing, bluray()); err == nil {
			t.Fatal("a better file from the forcing Import landed, want it refused for filling nothing")
		}
		if got := cell().File; got == nil || got.ID != forced.File.ID {
			t.Fatalf("the forced cell holds %+v after a better file of its own Import, want the forced file", got)
		}
		other, err := l.importOne(t, svc, files.Batch{Work: tree.ID, Root: l.root.ID, Import: storage.NewUUID()}, bluray())
		if err != nil {
			t.Fatal(err)
		}
		if got := cell().File; got == nil || got.ID != other.File.ID {
			t.Fatalf("the forced cell holds %+v after a better file of another Import, want that file", got)
		}
	})
}

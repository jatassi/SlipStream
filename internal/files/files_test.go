package files_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var admin identity.Admin

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type grabbing map[files.CellKey]files.GrabState

func (g grabbing) GrabCells(context.Context, storage.Reader[storage.Library], works.WorkID) (map[files.CellKey]files.GrabState, error) {
	return g, nil
}

type library struct {
	lib     *storage.DB[storage.Library]
	fx      *contenttypetest.Fixture
	works   *works.Works
	targets *targets.Targets
	root    roots.Root
}

func open(t *testing.T) *library {
	t.Helper()
	lib := storagetest.Library(t, works.Schema(), targets.Schema(), roots.Schema(), files.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(t.Context(), works.Config{
		Library:      lib,
		Catalogs:     map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Changed:      discard[works.WorkChanged]{},
		TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	types := []works.TypeID{contenttypetest.ID}
	root, err := roots.New(roots.Config{Library: lib, Types: types, Self: role.NodeID{UUID: storage.NewUUID()}, Changed: discard[roots.Changed]{}}).
		Create(t.Context(), admin, contenttypetest.ID, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return &library{lib: lib, fx: fx, works: w, targets: targets.New(targets.Config{Library: lib, Types: types, Changed: discard[targets.Changed]{}}), root: root}
}

func (l *library) work(t *testing.T, key works.Key, monitor bool) works.Tree {
	t.Helper()
	l.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: "Shelf " + string(key), Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}, {Key: "c2", Label: "Mittens"}}},
	}}})
	tree, _, err := l.works.Add(t.Context(), admin, contenttypetest.ID, key, monitor)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func (l *library) target(t *testing.T, name string, rs ...targets.Resolution) targets.Target {
	t.Helper()
	target, err := l.targets.CreateTarget(t.Context(), admin, name, rs)
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func (l *library) place(t *testing.T, item works.ItemID, work works.WorkID, path string, fills ...targets.TargetID) {
	t.Helper()
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := files.Add(ctx, tx, files.File{
			WorkID: work, ItemID: item, Edition: files.Standard, RootID: l.root.ID, Path: path, Size: 1024, Probe: files.Probe{Width: 1920, Height: 1080},
		}, fills)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func lines(cells []files.Cell) string {
	var b strings.Builder
	for _, c := range cells {
		fmt.Fprintf(&b, "%s %s %s %s wanted=%t", c.Item.Label, c.Edition, c.Target.Name, c.State, c.Wanted())
		if c.File != nil {
			fmt.Fprintf(&b, " %s %dx%d", c.File.Path, c.File.Probe.Width, c.File.Probe.Height)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func TestCellsAreComputedFromFillsGrabsAndTheDefaultProfile(t *testing.T) {
	l := open(t)
	hd := l.target(t, "1080p", targets.R1080p)
	uhd := l.target(t, "2160p", targets.R2160p)
	if _, err := l.targets.CreateProfile(t.Context(), admin, "Standard", contenttypetest.ID, []targets.TargetID{uhd.ID, hd.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := l.targets.CreateProfile(t.Context(), admin, "Small", contenttypetest.ID, []targets.TargetID{hd.ID}); err != nil {
		t.Fatal(err)
	}
	watched := l.work(t, "w1", true)
	scarf, mittens := watched.Items[0].ID, watched.Items[1].ID
	l.place(t, scarf, watched.ID, "Shelf w1/Scarf - 1080p.mkv", hd.ID)
	svc := files.New(files.Config{Library: l.lib, Works: l.works, Grabs: grabbing{
		{Item: mittens, Edition: files.Standard, Target: uhd.ID}: files.GrabDownloading,
		{Item: mittens, Edition: files.Standard, Target: hd.ID}:  files.GrabFailed,
		{Item: scarf, Edition: files.Standard, Target: hd.ID}:    files.GrabDownloading,
	}})
	cells, err := svc.WorkCells(t.Context(), admin, watched.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "Scarf standard 2160p missing wanted=true\n" +
		"Scarf standard 1080p met wanted=false Shelf w1/Scarf - 1080p.mkv 1920x1080\n" +
		"Mittens standard 2160p downloading wanted=false\n" +
		"Mittens standard 1080p failed wanted=true\n"
	if got := lines(cells); got != want {
		t.Fatalf("cells =\n%swant\n%s", got, want)
	}

	unwatched := l.work(t, "w2", false)
	cells, err = svc.WorkCells(t.Context(), admin, unwatched.ID)
	if err != nil {
		t.Fatal(err)
	}
	want = "Scarf standard 2160p missing wanted=false\n" +
		"Scarf standard 1080p missing wanted=false\n" +
		"Mittens standard 2160p missing wanted=false\n" +
		"Mittens standard 1080p missing wanted=false\n"
	if got := lines(cells); got != want {
		t.Fatalf("unmonitored cells =\n%swant\n%s", got, want)
	}
}

func TestAWorkWhoseTypeHasNoProfileHasNoCells(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true)
	svc := files.New(files.Config{Library: l.lib, Works: l.works, Grabs: files.NoGrabs{}})
	cells, err := svc.WorkCells(t.Context(), admin, tree.ID)
	if err != nil || len(cells) != 0 {
		t.Fatalf("cells = %v, %v; want none", cells, err)
	}
	if _, err := svc.WorkCells(t.Context(), admin, works.WorkID{UUID: storage.NewUUID()}); !errors.Is(err, works.ErrNoWork) {
		t.Fatalf("cells of no Work: err = %v, want works.ErrNoWork", err)
	}
}

func TestACellIsFilledOnce(t *testing.T) {
	l := open(t)
	hd := l.target(t, "1080p", targets.R1080p)
	tree := l.work(t, "w1", true)
	l.place(t, tree.Items[0].ID, tree.ID, "a.mkv", hd.ID)
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := files.Add(ctx, tx, files.File{WorkID: tree.ID, ItemID: tree.Items[0].ID, Edition: files.Standard, RootID: l.root.ID, Path: "b.mkv"}, []targets.TargetID{hd.ID})
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "UNIQUE constraint failed: files_fill.item_id, files_fill.edition, files_fill.target_id") {
		t.Fatalf("a second fill of one cell: err = %v", err)
	}
}

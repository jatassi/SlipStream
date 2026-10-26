package files_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var admin identity.Admin

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

type grabbing map[files.CellKey]files.GrabState

func (g grabbing) GrabCells(context.Context, storage.Reader[storage.Library], []works.WorkID) (map[files.CellKey]files.GrabState, error) {
	return g, nil
}

func (grabbing) Unfinished(context.Context, storage.Reader[storage.Library]) ([]storage.UUID, error) {
	return nil, nil
}

func (grabbing) Settle(context.Context, *storage.Tx[storage.Library], works.WorkID, map[files.CellKey]bool, files.Unwanted) (int, error) {
	return 0, nil
}

type library struct {
	lib     *storage.DB[storage.Library]
	fx      *contenttypetest.Fixture
	works   *works.Works
	targets *targets.Targets
	root    roots.Root
	held    *roots.Roots
}

func open(t testing.TB) *library {
	t.Helper()
	lib := storagetest.Library(t, nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), files.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(t.Context(), works.Config{
		Library:      lib,
		Catalogs:     map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Source:       fx.Source(),
		Artwork:      fx,
		Changed:      discard[works.Changed]{},
		TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	node := role.NodeID{UUID: storage.NewUUID()}
	root := rootstest.Root(t, lib, node, contenttypetest.ID, t.TempDir())
	goals, err := targets.Open(t.Context(), targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &library{lib: lib, fx: fx, works: w, targets: goals, root: root, held: rootstest.Held(t, lib, node, nil, contenttypetest.ID)}
}

// files is the files owner over l, whose today is the fixture's release
// date, with grabs in flight.
func (l *library) files(cells files.Inflight) *files.Files {
	return l.on(cells, l.held)
}

// on is the files owner over l whose disk queue works through disk.
func (l *library) on(cells files.Inflight, disk files.Disk) *files.Files {
	return files.New(files.Config{
		Library: l.lib, Works: l.works, Targets: l.targets, Grabs: cells,
		Naming: map[works.TypeID]naming.Facet{contenttypetest.ID: l.fx}, Today: func() time.Time { return today },
		Disk: disk, Timings: files.Timings{RetryMin: time.Millisecond, RetryMax: 4 * time.Millisecond},
		Rand: rand.New(rand.NewPCG(1, 2)), Log: slog.New(slog.DiscardHandler), //nolint:gosec // seeded on purpose so tests replay
	})
}

var today = time.Date(contenttypetest.Released.Year, time.Month(contenttypetest.Released.Month), contenttypetest.Released.Day, 9, 0, 0, 0, time.UTC)

// work adds a Work of two Items wanting from profile, the type's default
// when nil.
func (l *library) work(t *testing.T, key works.Key, monitor bool, profile *targets.ProfileID) works.Tree {
	t.Helper()
	l.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: "Shelf " + string(key), Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}, {Key: "c2", Label: "Mittens"}}},
	}}})
	tree, _, err := l.works.Add(t.Context(), admin, contenttypetest.ID, key, works.Adding{
		Monitor: monitor,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, profile); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func (l *library) profile(t testing.TB, name string, of ...targets.Target) targets.ProfileID {
	t.Helper()
	s := targets.ProfileSpec{Name: name, ContentType: contenttypetest.ID}
	for _, target := range of {
		s.Entries = append(s.Entries, targets.Entry{Target: target.ID, Sourcing: targets.Download})
	}
	p, err := l.targets.CreateProfile(t.Context(), admin, s)
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

// target creates a Target whose cutoff is its top tier.
func (l *library) target(t testing.TB, name string, tiers ...[]release.Quality) targets.Target {
	t.Helper()
	target, err := l.targets.CreateTarget(t.Context(), admin, targets.Spec{Name: name, Tiers: tiers, Cutoff: tiers[0][0]})
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func (l *library) place(t testing.TB, item works.ItemID, work works.WorkID, path string, q release.Quality, fills ...targets.TargetID) {
	t.Helper()
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := files.Add(ctx, tx, files.File{
			WorkID: work, ItemID: item, Cut: release.Standard, RootID: l.root.ID, Path: path, Size: 1024, Probe: files.Probe{Width: 1920, Height: 1080}, Quality: q,
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
			fmt.Fprintf(&b, " %s %s %dx%d", c.File.Path, c.File.Quality, c.File.Probe.Width, c.File.Probe.Height)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func TestCellsAreComputedFromFillsGrabsAndTheWorksOwnProfile(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p, release.WEBRip1080p})
	uhd := l.target(t, "UHD", []release.Quality{release.Remux2160p}, []release.Quality{release.WEBDL2160p})
	standard := l.profile(t, "Standard", uhd, hd)
	small := l.profile(t, "Small", hd)
	watched := l.work(t, "w1", true, &standard)
	scarf, mittens := watched.Items[0].ID, watched.Items[1].ID
	l.place(t, scarf, watched.ID, "Shelf w1/Scarf - HD.mkv", release.WEBRip1080p, hd.ID)
	l.place(t, scarf, watched.ID, "Shelf w1/Scarf - UHD.mkv", release.WEBDL2160p, uhd.ID)
	svc := l.files(grabbing{
		{Item: mittens, Edition: targets.AnyEdition, Target: uhd.ID}: files.GrabDownloading,
		{Item: mittens, Edition: targets.AnyEdition, Target: hd.ID}:  files.GrabFailed,
		{Item: scarf, Edition: targets.AnyEdition, Target: hd.ID}:    files.GrabDownloading,
	})
	_, cells, err := svc.WorkCells(t.Context(), admin, watched.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "Scarf any UHD upgradable wanted=true Shelf w1/Scarf - UHD.mkv webdl-2160p 1920x1080\n" +
		"Scarf any HD met wanted=false Shelf w1/Scarf - HD.mkv webrip-1080p 1920x1080\n" +
		"Mittens any UHD downloading wanted=false\n" +
		"Mittens any HD failed wanted=true\n"
	if got := lines(cells); got != want {
		t.Fatalf("cells =\n%swant\n%s", got, want)
	}

	unwatched := l.work(t, "w2", false, &small)
	_, cells, err = svc.WorkCells(t.Context(), admin, unwatched.ID)
	if err != nil {
		t.Fatal(err)
	}
	want = "Scarf any HD missing wanted=false\n" +
		"Mittens any HD missing wanted=false\n"
	if got := lines(cells); got != want {
		t.Fatalf("cells of a Work wanting from Small =\n%swant\n%s", got, want)
	}

	defaulted := l.work(t, "w3", true, nil)
	_, cells, err = svc.WorkCells(t.Context(), admin, defaulted.ID)
	if err != nil {
		t.Fatal(err)
	}
	want = "Scarf any 1080p missing wanted=true\n" +
		"Mittens any 1080p missing wanted=true\n"
	if got := lines(cells); got != want {
		t.Fatalf("cells of a Work added with no Profile =\n%swant the type's default, holding the starter\n%s", got, want)
	}
}

func TestAnItemNotOutYetIsUnreleasedAndUnwanted(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	scarf, mittens := tree.Items[0].ID, tree.Items[1].ID
	tomorrow := works.Date{Year: today.Year(), Month: int(today.Month()), Day: today.Day() + 1}
	l.fx.SetTiming(scarf, targets.Timing{Released: release.Known(tomorrow)})
	l.fx.SetTiming(mittens, targets.Timing{})
	starter := must[[]targets.Target](t)(l.targets.ListTargets(t.Context(), admin))[0]
	items, cells, err := l.files(grabbing{{Item: mittens, Edition: targets.AnyEdition, Target: starter.ID}: files.GrabFailed}).WorkCells(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "Scarf any 1080p unreleased wanted=false\n" +
		"Mittens any 1080p failed wanted=false\n"
	if got := lines(cells); got != want {
		t.Fatalf("cells =\n%swant\n%s", got, want)
	}
	if len(items) != 2 {
		t.Fatalf("WorkCells answered %d Items, want 2", len(items))
	}
}

func TestAnItemsStatusIsItsCellsFirstStateInGlossaryOrder(t *testing.T) {
	cells := func(states ...files.State) []files.Cell {
		var out []files.Cell
		for _, s := range states {
			out = append(out, files.Cell{State: s})
		}
		return out
	}
	for _, c := range []struct {
		cells []files.Cell
		want  files.State
	}{
		{cells(files.Met, files.Upgradable), files.Upgradable},
		{cells(files.Met, files.Failed, files.Downloading), files.Downloading},
		{cells(files.Met, files.Missing, files.Unreleased), files.Unreleased},
		{cells(files.Met), files.Met},
	} {
		if got, ok := files.Status(c.cells); !ok || got != c.want {
			t.Errorf("Status of %v = %s, %t; want %s", c.cells, got, ok, c.want)
		}
	}
	if s, ok := files.Status(nil); ok {
		t.Errorf("an Item with no cell has status %s", s)
	}
}

func TestAWorkWhoseProfileHoldsNoTargetHasNoCells(t *testing.T) {
	l := open(t)
	empty := l.profile(t, "Empty")
	tree := l.work(t, "w1", true, &empty)
	svc := l.files(files.NoGrabs{})
	items, cells, err := svc.WorkCells(t.Context(), admin, tree.ID)
	if err != nil || len(cells) != 0 || len(items) != 2 {
		t.Fatalf("WorkCells = %v, %v, %v; want two Items and no cell", items, cells, err)
	}
	if _, _, err := svc.WorkCells(t.Context(), admin, works.WorkID{UUID: storage.NewUUID()}); !errors.Is(err, works.ErrNoWork) {
		t.Fatalf("cells of no Work: err = %v, want works.ErrNoWork", err)
	}
}

func TestACellIsFilledOnce(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	tree := l.work(t, "w1", true, nil)
	l.place(t, tree.Items[0].ID, tree.ID, "a.mkv", release.WEBDL1080p, hd.ID)
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := files.Add(ctx, tx, files.File{WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: l.root.ID, Path: "b.mkv", Quality: release.WEBDL1080p}, []targets.TargetID{hd.ID})
		return err
	})
	if err == nil || !strings.Contains(err.Error(), "UNIQUE constraint failed: files_fill.item_id, files_fill.edition, files_fill.target_id") {
		t.Fatalf("a second fill of one cell: err = %v", err)
	}
}

func TestATargetsEstimateIsItsFilesRateOverTheProfilesRuntime(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	uhd := l.target(t, "UHD", []release.Quality{release.WEBDL2160p})
	standard := l.profile(t, "Standard", hd, uhd)
	small := l.profile(t, "Small", hd)
	minutes := func(item works.ItemID, n int) {
		l.fx.SetTiming(item, targets.Timing{Released: release.Known(contenttypetest.Released), Runtime: release.Known(time.Duration(n) * time.Minute)})
	}
	sized := func(item works.ItemID, work works.WorkID, path string, mib int64, fills ...targets.TargetID) {
		t.Helper()
		err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := files.Add(ctx, tx, files.File{
				WorkID: work, ItemID: item, Cut: release.Standard, RootID: l.root.ID, Path: path, Size: mib << 20, Quality: release.WEBDL1080p,
			}, fills)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	first := l.work(t, "w1", true, &standard)
	minutes(first.Items[0].ID, 100)
	minutes(first.Items[1].ID, 60)
	sized(first.Items[0].ID, first.ID, "a.mkv", 1000, hd.ID)
	other := l.work(t, "w2", true, &small)
	minutes(other.Items[0].ID, 50)
	sized(other.Items[0].ID, other.ID, "b.mkv", 1000, hd.ID)
	sized(other.Items[1].ID, other.ID, "c.mkv", 9000, hd.ID)
	l.work(t, "w3", true, &standard)

	p := must[targets.Profile](t)(l.targets.GetProfile(t.Context(), admin, standard))
	got := must[map[targets.TargetID]files.Estimate](t)(l.files(files.NoGrabs{}).Estimates(t.Context(), admin, p))
	rate := 2000.0 / 150
	want := map[targets.TargetID]files.Estimate{hd.ID: {MiBPerMinute: rate, Bytes: int64(rate * 160 * (1 << 20))}}
	if len(got) != 1 || got[hd.ID] != want[hd.ID] {
		t.Fatalf("Estimates = %+v, want %+v: HD's files of known runtime over Standard's 160 known minutes, and no estimate for UHD", got, want)
	}
}

func TestANamedListWantsStandardAndEachOtherCutOnceSeen(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	listed := func(name string, cuts ...release.CutKey) targets.ProfileID {
		list := must[targets.Editions](t)(targets.NamedCuts(cuts...))
		p := must[targets.Profile](t)(l.targets.CreateProfile(t.Context(), admin, targets.ProfileSpec{
			Name: name, ContentType: contenttypetest.ID, Entries: []targets.Entry{{Target: hd.ID, Sourcing: targets.Download}}, Editions: list,
		}))
		return p.ID
	}
	both, outright := listed("Both", release.Standard, "extended"), listed("Outright", "extended", "imax")
	tree := l.work(t, "w1", true, &both)
	scarf, mittens := tree.Items[0].ID, tree.Items[1].ID
	svc := l.files(files.NoGrabs{})
	cells := func(w works.WorkID) string {
		t.Helper()
		_, cells, err := svc.WorkCells(t.Context(), admin, w)
		if err != nil {
			t.Fatal(err)
		}
		return lines(cells)
	}
	if got, want := cells(tree.ID), "Scarf standard HD missing wanted=true\nMittens standard HD missing wanted=true\n"; got != want {
		t.Fatalf("cells before any sighting =\n%swant Standard alone\n%s", got, want)
	}
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, key := range []string{"a", "b"} {
			if err := targets.See(ctx, tx, scarf, "extended", key); err != nil {
				return err
			}
		}
		if err := targets.See(ctx, tx, scarf, "imax", "c"); err != nil {
			return err
		}
		_, err := files.Add(ctx, tx, files.File{
			WorkID: tree.ID, ItemID: mittens, Cut: "extended", RootID: l.root.ID, Path: "Shelf w1/Mittens.mkv", Size: 1024, Quality: release.WEBDL1080p,
		}, []targets.TargetID{hd.ID})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "Scarf standard HD missing wanted=true\nScarf extended HD missing wanted=true\n" +
		"Mittens standard HD missing wanted=true\nMittens extended HD met wanted=false Shelf w1/Mittens.mkv webdl-1080p 0x0\n"
	if got := cells(tree.ID); got != want {
		t.Fatalf("cells once Scarf's Extended is seen and Mittens holds an Extended file =\n%swant\n%s", got, want)
	}
	other := l.work(t, "w2", true, &outright)
	want = "Scarf extended HD missing wanted=true\nScarf imax HD missing wanted=true\nMittens extended HD missing wanted=true\nMittens imax HD missing wanted=true\n"
	if got := cells(other.ID); got != want {
		t.Fatalf("cells of a list without Standard =\n%swant every cut it names outright\n%s", got, want)
	}
}

package imports

import (
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

// downloads is w over a Store that holds w's root and the download folder
// it answers, whose files at seeded a torrent holds, on disk when set.
func (w *world) downloads(t *testing.T, disk storing.Disk, seeded ...string) string {
	t.Helper()
	dir := t.TempDir()
	torrents := make([]string, len(seeded))
	for i, p := range seeded {
		torrents[i] = filepath.Join(dir, filepath.FromSlash(p))
	}
	w.cfg.Roots = rootstest.HeldOn(t, w.lib, w.node, rootstest.On{Disk: disk, Downloads: []string{dir}, Torrents: torrents, Tune: func(cfg *roots.Config) {
		cfg.Names, cfg.Describe = w.names, w.works.Describe
	}}, contenttypetest.ID)
	w.disk.Disk = w.cfg.Roots
	w.im = w.newImports(w.cfg)
	return dir
}

func sameIDs(a, b []ImportID) bool {
	return slices.EqualFunc(sortedIDs(a), sortedIDs(b), func(x, y ImportID) bool { return x == y })
}

func sortedIDs(ids []ImportID) []ImportID {
	return slices.SortedFunc(slices.Values(ids), func(x, y ImportID) int { return x.Compare(y.UUID) })
}

func write(t *testing.T, dir string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		at := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(at, []byte(p), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Truncate(at, 1<<30); err != nil {
			t.Fatal(err)
		}
	}
}

// opened is the folder Imports of dir by their Works, opening it.
func (w *world) opened(t *testing.T, dir string) (o Opened, byWork map[works.WorkID]*Import) {
	t.Helper()
	o, err := w.im.ImportFolder(t.Context(), identity.Admin{}, w.node, dir)
	if err != nil {
		t.Fatal(err)
	}
	byWork = map[works.WorkID]*Import{}
	for _, id := range o.IDs {
		imp := w.stored(t, id)
		byWork[imp.Work] = imp
	}
	return o, byWork
}

func (w *world) act(t *testing.T, imp *Import, position int, a Action) Import {
	t.Helper()
	got, err := w.im.Act(t.Context(), identity.Admin{}, FileRef{Import: imp.ID, Position: position}, a)
	if err != nil {
		t.Fatalf("%s on file %d of %s: %v", a.Kind(), position, imp.Folder, err)
	}
	return got
}

func positionOf(imp *Import, base string) int {
	i := slices.IndexFunc(imp.Files, func(f File) bool { return filepath.Base(f.Path) == base })
	return imp.Files[i].Position
}

func waitsConfirm(imp *Import) bool {
	return !slices.ContainsFunc(imp.Files, func(f File) bool { return f.Outcome != Waiting || f.Reason.Code != Confirm })
}

func placed(imp *Import) bool {
	return !slices.ContainsFunc(imp.Files, func(f File) bool { return f.Outcome != Placed })
}

func TestAFolderImportsIntoTwoWorksWithEditedItems(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	one, two := w.add(t, "w1", w.root.ID), w.add(t, "w2", w.root.ID)
	dir := w.downloads(t, nil, "Shelf w2/Mystery.mkv")
	write(t, dir, "Shelf w1/Clip 1.mkv", "Shelf w1/Clip 2.mkv", "Shelf w2/Mystery.mkv", "Unknown.mkv")
	w.reconcile(t)

	o, byWork := w.opened(t, dir)
	if !o.Created || len(byWork) != 3 || byWork[one.ID] == nil || byWork[two.ID] == nil || byWork[works.WorkID{}] == nil {
		t.Fatalf("opening the folder made %d Imports, created %t, want one for each Work and one for the file no name matches", len(byWork), o.Created)
	}
	if again, _ := w.opened(t, dir); again.Created || !sameIDs(again.IDs, o.IDs) {
		t.Fatalf("reopening the folder answers %v, created %t, want the same Imports %v", again.IDs, again.Created, o.IDs)
	}
	folder := FromFolder
	if l, err := w.im.List(t.Context(), identity.Admin{}, Filter{Source: &folder}); err != nil || len(l.Imports) != 3 {
		t.Fatalf("Pending imports lists %d folder Imports (%v), want the three opened", len(l.Imports), err)
	}
	if !errors.Is(w.im.SetMoveFiles(t.Context(), identity.Admin{}, o.IDs, true), ErrMoveSeeding) {
		t.Fatal("Move files on a folder with a seeding file was taken, want ErrMoveSeeding")
	}

	unknown := byWork[works.WorkID{}]
	if f := unknown.Files[0]; f.Outcome != Waiting || f.Reason.Code != NoMatch {
		t.Fatalf("Unknown.mkv is %s, want waiting no_match", f.Outcome)
	}
	bluray := func(name string) string { return name + ".1080p.BluRay-GRP" }
	moved := w.act(t, unknown, 0, MatchFile{Work: two.ID, Items: []works.ItemID{two.Items[0].ID}, ReleaseName: bluray("Unknown")})
	if moved.ID != byWork[two.ID].ID {
		t.Fatalf("Change the match to %s left the file in Import %s, want it in that Work's Import %s", two.Label, moved.ID, byWork[two.ID].ID)
	}
	mystery := byWork[two.ID]
	w.act(t, mystery, positionOf(mystery, "Mystery.mkv"), MatchFile{Work: two.ID, Items: []works.ItemID{two.Items[2].ID, two.Items[1].ID}, ReleaseName: bluray("Mystery")})
	ones := w.until(t, "Shelf w1's clips are read", byWork[one.ID].ID, func(imp *Import) bool {
		return !slices.ContainsFunc(imp.Files, func(f File) bool { return f.Outcome == Pending })
	})
	for _, f := range ones.Files {
		w.act(t, ones, f.Position, MatchFile{Work: one.ID, Items: f.Items, ReleaseName: bluray(f.ReleaseName)})
	}

	for _, imp := range []*Import{byWork[one.ID], mystery} {
		w.until(t, imp.Folder+" waits confirm", imp.ID, waitsConfirm)
	}
	if _, err := w.im.ConfirmAll(t.Context(), identity.Admin{}, []ImportID{byWork[one.ID].ID, mystery.ID}); err != nil {
		t.Fatal(err)
	}
	w.until(t, "Shelf w1 is placed", byWork[one.ID].ID, placed)
	got := w.until(t, "Shelf w2 is placed", mystery.ID, placed)
	if i := slices.IndexFunc(got.Files, func(f File) bool { return filepath.Base(f.Path) == "Mystery.mkv" }); !slices.Equal(got.Files[i].Items, []works.ItemID{two.Items[1].ID, two.Items[2].ID}) {
		t.Errorf("Mystery.mkv holds %v, want the two Items the Admin chose in tree order", got.Files[i].Items)
	}
	if n := len(w.filesOf(t, one.ID)); n != 2 {
		t.Errorf("%s has %d files, want its two clips", one.Label, n)
	}
	if n := len(w.filesOf(t, two.ID)); n != 2 {
		t.Errorf("%s has %d files, want the matched file and the two-Item one", two.Label, n)
	}
	if _, err := os.Stat(filepath.Join(dir, "Unknown.mkv")); err != nil {
		t.Errorf("Unknown.mkv's source with Move files off: %v, want it kept", err)
	}
}

func TestAFolderOutsideTheNodesRootsAndDownloadFoldersIsRefused(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.downloads(t, nil)
	if _, err := w.im.ImportFolder(t.Context(), identity.Admin{}, w.node, t.TempDir()); !errors.Is(err, ErrOutside) {
		t.Fatalf("opening a folder outside the fence = %v, want ErrOutside", err)
	}
}

func TestMoveFilesPlacesByAMoveAndRemovesTheSource(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	one := w.add(t, "w1", w.root.ID)
	dir := w.downloads(t, nil)
	write(t, dir, "Shelf w1/Clip 1.mkv")
	w.reconcile(t)
	o, byWork := w.opened(t, dir)
	if err := w.im.SetMoveFiles(t.Context(), identity.Admin{}, o.IDs, true); err != nil {
		t.Fatal(err)
	}
	imp := w.until(t, "the clip is read", byWork[one.ID].ID, func(imp *Import) bool { return imp.Files[0].Outcome != Pending })
	w.act(t, imp, 0, MatchFile{Work: one.ID, Items: imp.Files[0].Items, ReleaseName: "Clip 1.1080p.BluRay-GRP"})
	w.until(t, "the clip waits confirm", imp.ID, waitsConfirm)
	w.act(t, imp, 0, ConfirmFile{})
	got := w.until(t, "the Import finishes", imp.ID, func(imp *Import) bool { return !imp.Finished.IsZero() })
	if f := got.Files[0]; f.Outcome != Placed || f.PlacedBy != ByMove || f.SourceRemoved.IsZero() {
		t.Fatalf("the clip is %s placed by %q, source removed at %v, want placed by a move with its source gone", f.Outcome, f.PlacedBy, f.SourceRemoved)
	}
	if _, err := os.Stat(filepath.Join(dir, "Shelf w1", "Clip 1.mkv")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the moved clip's source: %v, want it gone", err)
	}
}

// crossDisk is a disk whose volumes link and clone nothing, so a place
// copies.
type crossDisk struct{ storing.Disk }

func (crossDisk) Link(context.Context, string, string) error  { return storing.ErrCrossDevice }
func (crossDisk) Clone(context.Context, string, string) error { return errors.ErrUnsupported }

func TestACopyWaitsForItsGrantThenPlaces(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	one := w.add(t, "w1", w.root.ID)
	disk, err := storing.NewOSDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := w.downloads(t, crossDisk{disk})
	write(t, dir, "Shelf w1/Clip 1.mkv")
	w.reconcile(t)
	_, byWork := w.opened(t, dir)
	imp := w.until(t, "the clip is read", byWork[one.ID].ID, func(imp *Import) bool { return imp.Files[0].Outcome != Pending })
	w.act(t, imp, 0, MatchFile{Work: one.ID, Items: imp.Files[0].Items, ReleaseName: "Clip 1.1080p.BluRay-GRP"})
	w.until(t, "the clip waits confirm", imp.ID, waitsConfirm)

	w.act(t, imp, 0, ConfirmFile{})
	got := w.until(t, "the clip is placed", imp.ID, placed)
	if f := got.Files[0]; f.PlacedBy != ByCopy {
		t.Errorf("the clip was placed by %q, want a copy, which the Store makes only once the Controller grants it", f.PlacedBy)
	}
}

// seasonFacet is shelfFacet with a release reader that reads a name's
// title as the words before its season mark, as in Shelf.w1.S01.1080p.
type seasonFacet struct{ shelfFacet }

func (seasonFacet) ReadName(p release.Prepared) release.Name {
	title, _, ok := strings.Cut(p.Original, ".S01.")
	if !ok {
		return release.Name{}
	}
	return release.Name{Titles: []string{strings.ReplaceAll(title, ".", " ")}}
}

// TestAFolderNamedAsAReleaseMatchesTheWorkItsTitleNames opens a folder a
// download client named for its release, whose files' names don't match
// on their own.
func TestAFolderNamedAsAReleaseMatchesTheWorkItsTitleNames(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.cfg.Facets = map[works.TypeID]scanning.Facet{contenttypetest.ID: seasonFacet{}}
	one := w.add(t, "w1", w.root.ID)
	dir := w.downloads(t, nil)
	write(t, dir, "Shelf.w1.S01.1080p.WEB-DL-GRP/Clip 1.mkv", "Shelf.w1.S01.1080p.WEB-DL-GRP/Clip 2.mkv")

	_, byWork := w.opened(t, dir)
	if len(byWork) != 1 || byWork[one.ID] == nil || len(byWork[one.ID].Files) != 2 {
		t.Fatalf("opening the folder made Imports for %v, want both files under Shelf w1", slices.Collect(maps.Keys(byWork)))
	}
}

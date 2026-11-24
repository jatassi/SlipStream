package imports

import (
	"context"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// pack adds Work key, homed in w's root, with one box of n clips, opens a
// folder of a file per clip, and matches each to a release name: the last
// to one no Target takes, so it waits while the rest wait confirm.
func (w *world) pack(t *testing.T, key works.Key, n int) (tree works.Tree, imp *Import, dir string) {
	t.Helper()
	var clips []works.Entry
	for i := range n {
		clips = append(clips, works.Entry{Key: works.Key("c" + strconv.Itoa(i+1)), Label: "Clip " + strconv.Itoa(i+1)})
	}
	w.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: "Shelf " + string(key), Children: []works.Entry{{Key: "b1", Label: "Box", Children: clips}}}})
	added, err := w.library.Add(t.Context(), identity.Admin{}, library.Adding{Type: contenttypetest.ID, Key: key, Root: &w.root.ID})
	if err != nil {
		t.Fatal(err)
	}
	dir = w.downloads(t, nil)
	var paths []string
	for i := range n {
		paths = append(paths, "Shelf "+string(key)+"/Clip "+strconv.Itoa(i+1)+".mkv")
	}
	write(t, dir, paths...)
	w.reconcile(t)
	_, byWork := w.opened(t, dir)
	imp = w.until(t, "the clips are read", byWork[added.Tree.ID].ID, func(imp *Import) bool {
		return !slices.ContainsFunc(imp.Files, func(f File) bool { return f.Outcome == Pending })
	})
	for _, f := range imp.Files {
		name := "Shelf.1080p.BluRay-GRP"
		if filepath.Base(f.Path) == lastOf(n) {
			name = "Shelf.480p.SDTV-GRP"
		}
		w.act(t, imp, f.Position, MatchFile{Work: imp.Work, Items: f.Items, ReleaseName: name})
	}
	return added.Tree, w.stored(t, imp.ID), dir
}

func lastOf(n int) string { return "Clip " + strconv.Itoa(n) + ".mkv" }

func (w *world) match(t *testing.T, imp *Import, f File) {
	t.Helper()
	w.act(t, imp, f.Position, MatchFile{Work: imp.Work, Items: f.Items, ReleaseName: "Shelf.1080p.BluRay-GRP"})
}

// confirmAll confirms every file of imp that waits confirm and waits until
// imp rests again.
func (w *world) confirmAll(t *testing.T, imp *Import) *Import {
	t.Helper()
	if _, err := w.im.ConfirmAll(t.Context(), identity.Admin{}, []ImportID{imp.ID}); err != nil {
		t.Fatal(err)
	}
	return w.until(t, "the confirmed files are placed", imp.ID, func(got *Import) bool {
		return got.atRest() && !slices.ContainsFunc(got.Files, func(f File) bool { return f.Confirmed && f.Outcome != Placed })
	})
}

func importedFiles(fs []ImportedFacts) []int {
	out := make([]int, len(fs))
	for i, f := range fs {
		out[i] = len(f.Files)
	}
	return out
}

func TestAPackRecordsOneImportedAndASecondOnceTheWaitingFileIsPlaced(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	tree, imp, _ := w.pack(t, "w1", 10)
	imp = w.confirmAll(t, imp)
	got := entrytest.Of(w.notices, importedRow)
	if !slices.Equal(importedFiles(got), []int{9}) || got[0].Waiting != 1 || got[0].Work.ID != tree.ID || !strings.HasPrefix(got[0].Root, "Shelves library root") {
		t.Fatalf("after nine of ten are placed Imported is %+v, want one of nine files with one waiting, into the root", got)
	}
	if about := entrytest.Abouts(w.notices, importedRow)[0]; about.Import != imp.ID.UUID || about.Grab != imp.Grab.UUID || !slices.Equal(about.Works, []works.WorkID{tree.ID}) {
		t.Errorf("Imported is about %+v, want Import %s of Work %s with its Grab", about, imp.ID, tree.ID)
	}
	if files := importedRow.Render(got[0], entrytest.Context).Detail.Files; len(files) != 9 || files[0].Quality != "Bluray-1080p" || len(files[0].Items) != 1 {
		t.Errorf("Imported's detail lists %+v, want each of the nine files with its Item and quality", files)
	}
	last := fileAt(imp, positionOf(imp, lastOf(10)))
	w.match(t, imp, *last)
	w.confirmAll(t, imp)
	got = entrytest.Of(w.notices, importedRow)
	if !slices.Equal(importedFiles(got), []int{9, 1}) || got[1].Waiting != 0 || got[1].Files[0].Items[0].Number != 10 {
		t.Fatalf("after the tenth is placed Imported is %+v, want a second for the tenth alone", got)
	}
	if up := entrytest.Of(w.notices, upgradedRow); len(up) != 0 {
		t.Errorf("a pack that replaced nothing recorded Upgraded %+v", up)
	}
}

func TestImportsMessagesReadAsWritten(t *testing.T) {
	book := entry.Nouns{Work: works.Noun{One: "Book", Many: "Books"}, Item: works.Noun{One: "Book", Many: "Books"}, Single: true}
	zine := entry.Nouns{Work: works.Noun{One: "Zine", Many: "Zines"}, Levels: []works.Noun{{One: "Volume", Many: "Volumes"}}, Item: works.Noun{One: "Issue", Many: "Issues"}}
	id, err := storage.ParseUUID("019a8f5e-0000-7000-8000-000000000002")
	if err != nil {
		t.Fatal(err)
	}
	poster := artwork.Ref{Source: "tmdb", Shape: artwork.ShapePoster, Path: "/cover.jpg"}
	dune := entry.Work{ID: works.WorkID{UUID: id}, Label: "Dune (1965)", Type: "book", Nouns: book, Poster: poster}
	ark := entry.Work{ID: works.WorkID{UUID: id}, Label: "Ark (2019)", Type: "zine", Nouns: zine, Poster: poster}
	const gb = 1 << 30
	issues := func(volume, from, to int, q release.Quality, replaced ...release.Quality) []PlacedFile {
		var out []PlacedFile
		for n := from; n <= to; n++ {
			out = append(out, PlacedFile{Items: []ItemAt{{Level: "Volume " + strconv.Itoa(volume), Number: n}}, Quality: q, Size: gb / 2, PlacedBy: ByLink, Replaced: replaced})
		}
		return out
	}
	page := "/works/" + id.String()
	stored := func(e entry.Entry, facts any, name string, names ...string) {
		entrytest.Stored(t, e, facts, "testdata/notices", name, names...)
	}
	stored(importedRow, ImportedFacts{Work: dune, Root: "Books library root on Den", Files: []PlacedFile{{Quality: release.WEBDL1080p, Size: 6 * gb, PlacedBy: ByLink}}}, "imported-one",
		"Dune (1965)", "WEBDL-1080p", "6.0 GB", "Books library root on Den", page)
	stored(importedRow, ImportedFacts{Work: ark, Root: "Zines library root on Den", Files: issues(2, 1, 9, release.WEBDL1080p), Failed: 1, Waiting: 2}, "imported-many",
		"Ark (2019)", "9 issues", "WEBDL-1080p", "4.5 GB", "1 issue", "2 issues", page)
	stored(importedRow, ImportedFacts{Work: ark, Root: "Zines library root on Den", Files: append(issues(1, 9, 10, release.HDTV720p), issues(2, 1, 1, release.HDTV720p)...)}, "imported-two-levels",
		"Ark (2019)", "3 issues", "HDTV-720p", "1.5 GB", "volume 1", "volume 2", page)
	stored(upgradedRow, ImportedFacts{Work: dune, Root: "Books library root on Den", Files: []PlacedFile{{Quality: release.Bluray1080p, Size: 9 * gb, PlacedBy: ByCopy, Replaced: []release.Quality{release.WEBDL720p}}}}, "upgraded-one",
		"Dune (1965)", "Bluray-1080p", "WEBDL-720p", page)
	stored(upgradedRow, ImportedFacts{Work: ark, Root: "Zines library root on Den", Files: issues(1, 1, 5, release.Bluray1080p, release.HDTV720p)}, "upgraded-many",
		"Ark (2019)", "5 issues", "HDTV-720p", "Bluray-1080p", "2 more", page)
	stored(upgradedRow, nil, "upgraded-webdl", "WEBDL-1080p")
	stored(scanAddedRow, ScanAddedFacts{Root: "Books library root on Den", Nouns: book, Count: 140, First: []string{"Dune (1965)", "Emma (1815)", "Ulysses (1922)"}}, "works-added-by-scan",
		"Books library root on Den", "140 books", "Dune (1965)", "Emma (1815)", "Ulysses (1922)", "137 more", "/library")
	stored(scanAddedRow, ScanAddedFacts{Root: "Books library root on Den", Nouns: book, Count: 1, First: []string{"Dune (1965)"}}, "works-added-by-scan-one",
		"Books library root on Den", "1 book", "Dune (1965)", "/library")
}

// adopt adds Work key homed in root as a Library scan adopts one.
func (w *world) adopt(t *testing.T, key works.Key, root roots.ID) {
	t.Helper()
	w.stock(key)
	if _, err := w.library.Add(t.Context(), identity.Admin{}, library.Adding{Type: contenttypetest.ID, Key: key, Root: &root, By: works.AddedByScan}); err != nil {
		t.Fatal(err)
	}
}

func (w *world) recordScan(t *testing.T, root roots.Root) []ScanAddedFacts {
	t.Helper()
	if err := w.im.recordScan(t.Context(), root, Comparison{Place: root.Place}); err != nil {
		t.Fatal(err)
	}
	return entrytest.Of(w.notices, scanAddedRow)
}

func TestAScanRecordsEveryWorkAdoptedInItsRootSinceTheScanBefore(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	other := w.newRoot(t)
	for i := range 500 {
		w.adopt(t, works.Key("a"+strconv.Itoa(i+1)), w.root.ID)
	}
	for i := range 10 {
		w.adopt(t, works.Key("b"+strconv.Itoa(i+1)), other.ID)
	}
	w.add(t, "manual", w.root.ID)
	got := w.recordScan(t, w.root)
	if len(got) != 1 || got[0].Count != 500 || len(got[0].First) != 3 || slices.ContainsFunc(got[0].First, func(l string) bool { return !strings.HasPrefix(l, "Shelf a") }) || !strings.HasPrefix(got[0].Root, "Shelves library root") {
		t.Fatalf("the first scan of the root recorded %+v, want one event counting its 500 adopted Works with 3 of their titles", got)
	}
	if about := entrytest.Abouts(w.notices, scanAddedRow)[0]; about.Type != contenttypetest.ID || len(about.Works) != 500 {
		t.Errorf("the scan's entry is about %d Works of %s, want each of the 500 it added", len(about.Works), about.Type)
	}
	if got = w.recordScan(t, w.root); len(got) != 1 {
		t.Fatalf("a scan that adopted nothing recorded %+v", got[1:])
	}
	w.adopt(t, "a501", w.root.ID)
	w.adopt(t, "a502", w.root.ID)
	if got = w.recordScan(t, w.root); len(got) != 2 || got[1].Count != 2 {
		t.Fatalf("after a scan that adopted a501 and failed, the next recorded %+v, want both a501 and a502", got[1:])
	}
	if got = w.recordScan(t, other); len(got) != 3 || got[2].Count != 10 || !strings.HasPrefix(got[2].First[0], "Shelf b") {
		t.Fatalf("a scan of the other root recorded %+v, want its 10 Works alone", got[2:])
	}
	if n := w.shelved.Len(); n != 0 {
		t.Errorf("adopting and adding without the add flow recorded %d library events", n)
	}
	if imported := entrytest.Of(w.notices, importedRow); len(imported) != 0 {
		t.Errorf("a scan recorded Imported %+v", imported)
	}
}

func TestAFileThatReplacesAPlacedOneRecordsUpgradedAndNotImported(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	tree, imp, dir := w.pack(t, "w1", 2)
	w.confirmAll(t, imp)
	write(t, dir, "Again/Shelf w1/Clip 1.mkv")
	_, byWork := w.opened(t, filepath.Join(dir, "Again"))
	again := w.until(t, "the clip is read", byWork[tree.ID].ID, func(imp *Import) bool {
		return !slices.ContainsFunc(imp.Files, func(f File) bool { return f.Outcome == Pending })
	})
	f := again.Files[0]
	w.act(t, again, f.Position, MatchFile{Work: again.Work, Items: f.Items, ReleaseName: "Shelf.1080p.BluRay-GRP"})
	waiting := w.stored(t, again.ID)
	said, err := w.im.Words(t.Context(), identity.Admin{}, *waiting)
	if err != nil {
		t.Fatal(err)
	}
	if len(said[f.Position].Forces) != 1 {
		t.Fatalf("the clip offers Force %+v, want one Target", said[f.Position].Forces)
	}
	w.act(t, waiting, f.Position, ForceFile{Cell: said[f.Position].Forces[0].Cell})
	if _, err := w.im.ConfirmAll(t.Context(), identity.Admin{}, []ImportID{again.ID}); err != nil {
		t.Fatal(err)
	}
	w.until(t, "the clip is forced in", again.ID, func(imp *Import) bool { return imp.atRest() && imp.Files[0].Outcome == Placed })
	up := entrytest.Of(w.notices, upgradedRow)
	if len(up) != 1 || len(up[0].Files) != 1 || !slices.Equal(up[0].Files[0].Replaced, []release.Quality{release.Bluray1080p}) || up[0].Files[0].Items[0].Number != 1 {
		t.Fatalf("Upgraded is %+v, want one for clip 1 replacing a Bluray-1080p file", up)
	}
	m := upgradedRow.Render(up[0], entrytest.Context)
	if len(m.Detail.Files) != 1 || m.Detail.Files[0].Replaced != "Bluray-1080p" || m.Detail.Files[0].ReplacedLabel != "HD" {
		t.Errorf("Upgraded's detail is %+v, want the clip with the quality and label of the file it replaced", m.Detail.Files)
	}
	if got := entrytest.Of(w.notices, importedRow); len(got) != 1 {
		t.Errorf("Imported is %+v, want only the pack's", got)
	}
}

func TestFilesARootImportPlacesRecordNothing(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	target := w.add(t, "w1", w.root.ID)
	w.pend(t, pend{Folder: "Odd", Files: []File{found("Odd/a.mkv")}})
	odd := w.at(t, "Odd")
	left, err := w.im.Into(t.Context(), identity.Admin{}, odd.ID, target.ID, []Chosen{{Position: position(odd, "Odd/a.mkv"), Items: []works.ItemID{target.Items[0].ID}}})
	if err != nil || left.Stage() != StageDone {
		t.Fatalf("Into answers %+v, %v; want the Import done", left, err)
	}
	if n := w.notices.Len(); n != 0 {
		t.Errorf("a root Import that placed its file recorded %d events", n)
	}
}

func TestAnImportItsWorksDeleteEndsRecordsNoImported(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	tree, imp, _ := w.pack(t, "w1", 3)
	second := positionOf(imp, "Clip 2.mkv")
	held, resume := make(chan struct{}), make(chan struct{})
	hold := func(_ context.Context, s Step) {
		if s.Import == imp.ID && s.Position == second && s.At == AtPlace {
			w.stop.Store(nil)
			close(held)
			<-resume
		}
	}
	w.stop.Store(&hold)
	if _, err := w.im.ConfirmAll(t.Context(), identity.Admin{}, []ImportID{imp.ID}); err != nil {
		t.Fatal(err)
	}
	<-held
	first := positionOf(imp, "Clip 1.mkv")
	w.until(t, "clip 1 is placed while clip 2 is held", imp.ID, func(imp *Import) bool { return fileAt(imp, first).Outcome == Placed })
	if _, _, err := w.library.Delete(t.Context(), identity.Admin{}, []works.WorkID{tree.ID}, library.Deleting{Files: library.DeleteFiles}); err != nil {
		t.Fatal(err)
	}
	close(resume)
	w.until(t, "the delete ends the Import", imp.ID, func(imp *Import) bool { return !imp.Finished.IsZero() })
	if got := entrytest.Of(w.notices, importedRow); len(got) != 0 {
		t.Errorf("an Import its Work's delete ended recorded Imported %+v", got)
	}
}

func TestASingleItemWorksUpgradedListsItsFileWithNoItemAndLeavesWhatItReplacedToItsLine(t *testing.T) {
	t.Parallel()
	f := ImportedFacts{
		Work: entry.Work{Label: "Heat (1995)", Nouns: entry.Nouns{Single: true}},
		Files: []PlacedFile{{
			Items: []ItemAt{{Number: 1}}, Quality: release.Bluray1080p, PlacedBy: "hardlink",
			Replaced: []release.Quality{release.WEBDL1080p}, ReplacedLabels: []string{"1080p"},
		}},
	}
	m := upgradedRow.Render(f, entrytest.Context)
	want := []entry.DetailFile{{Quality: "Bluray-1080p", PlacedBy: "hardlink"}}
	if !slices.Equal(m.Lines, []string{"It replaced WEBDL-1080p."}) || !slices.EqualFunc(m.Detail.Files, want, func(a, b entry.DetailFile) bool {
		return slices.Equal(a.Items, b.Items) && a.Quality == b.Quality && a.PlacedBy == b.PlacedBy && a.Replaced == b.Replaced && a.ReplacedLabel == b.ReplacedLabel
	}) {
		t.Errorf("a single-Item Work's Upgraded reads %q with files %+v, want its line to say what it replaced and its file to name only its quality", m.Lines, m.Detail.Files)
	}
}

package imports

import (
	"context"
	"flag"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/notice/noticetest"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var update = flag.Bool("update", false, "rewrite the rendered messages under testdata/notices")

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
	got := noticetest.Of(w.notices, importedRow)
	if !slices.Equal(importedFiles(got), []int{9}) || got[0].Waiting != 1 || got[0].Work.ID != tree.ID || !strings.HasPrefix(got[0].Root, "Shelves library root") {
		t.Fatalf("after nine of ten are placed Imported is %+v, want one of nine files with one waiting, into the root", got)
	}
	last := fileAt(imp, positionOf(imp, lastOf(10)))
	w.match(t, imp, *last)
	w.confirmAll(t, imp)
	got = noticetest.Of(w.notices, importedRow)
	if !slices.Equal(importedFiles(got), []int{9, 1}) || got[1].Waiting != 0 || got[1].Files[0].Items[0].Number != 10 {
		t.Fatalf("after the tenth is placed Imported is %+v, want a second for the tenth alone", got)
	}
	if up := noticetest.Of(w.notices, upgradedRow); len(up) != 0 {
		t.Errorf("a pack that replaced nothing recorded Upgraded %+v", up)
	}
}

func TestImportsMessagesReadAsWritten(t *testing.T) {
	book := notice.Nouns{Work: works.Noun{One: "Book", Many: "Books"}, Item: works.Noun{One: "Book", Many: "Books"}, Single: true}
	zine := notice.Nouns{Work: works.Noun{One: "Zine", Many: "Zines"}, Levels: []works.Noun{{One: "Volume", Many: "Volumes"}}, Item: works.Noun{One: "Issue", Many: "Issues"}}
	id, err := storage.ParseUUID("019a8f5e-0000-7000-8000-000000000002")
	if err != nil {
		t.Fatal(err)
	}
	poster := artwork.Ref{Source: "tmdb", Shape: artwork.ShapePoster, Path: "/cover.jpg"}
	dune := notice.Work{ID: works.WorkID{UUID: id}, Label: "Dune (1965)", Type: "book", Nouns: book, Poster: poster}
	ark := notice.Work{ID: works.WorkID{UUID: id}, Label: "Ark (2019)", Type: "zine", Nouns: zine, Poster: poster}
	const gb = 1 << 30
	issues := func(volume, from, to int, q release.Quality, replaced ...release.Quality) []PlacedFile {
		var out []PlacedFile
		for n := from; n <= to; n++ {
			out = append(out, PlacedFile{Items: []ItemAt{{Level: "Volume " + strconv.Itoa(volume), Number: n}}, Quality: q, Size: gb / 2, PlacedBy: ByLink, Replaced: replaced})
		}
		return out
	}
	golden := func(e notice.Entry, facts any, name string) {
		noticetest.Golden(t, e, facts, "testdata/notices", name, *update)
	}
	golden(importedRow, ImportedFacts{Work: dune, Root: "Books library root on Den", Files: []PlacedFile{{Quality: release.WEBDL1080p, Size: 6 * gb, PlacedBy: ByLink}}}, "imported-one")
	golden(importedRow, ImportedFacts{Work: ark, Root: "Zines library root on Den", Files: issues(2, 1, 9, release.WEBDL1080p), Failed: 1, Waiting: 2}, "imported-many")
	golden(importedRow, ImportedFacts{Work: ark, Root: "Zines library root on Den", Files: append(issues(1, 9, 10, release.HDTV720p), issues(2, 1, 1, release.HDTV720p)...)}, "imported-two-levels")
	golden(upgradedRow, ImportedFacts{Work: dune, Root: "Books library root on Den", Files: []PlacedFile{{Quality: release.Bluray1080p, Size: 9 * gb, PlacedBy: ByCopy, Replaced: []release.Quality{release.WEBDL720p}}}}, "upgraded-one")
	golden(upgradedRow, ImportedFacts{Work: ark, Root: "Zines library root on Den", Files: issues(1, 1, 5, release.Bluray1080p, release.HDTV720p)}, "upgraded-many")
	golden(scanAddedRow, ScanAddedFacts{Root: "Books library root on Den", Nouns: book, Count: 140, First: []string{"Dune (1965)", "Emma (1815)", "Ulysses (1922)"}}, "works-added-by-scan")
	golden(scanAddedRow, ScanAddedFacts{Root: "Books library root on Den", Nouns: book, Count: 1, First: []string{"Dune (1965)"}}, "works-added-by-scan-one")
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
	return noticetest.Of(w.notices, scanAddedRow)
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
	if imported := noticetest.Of(w.notices, importedRow); len(imported) != 0 {
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
	up := noticetest.Of(w.notices, upgradedRow)
	if len(up) != 1 || len(up[0].Files) != 1 || !slices.Equal(up[0].Files[0].Replaced, []release.Quality{release.Bluray1080p}) || up[0].Files[0].Items[0].Number != 1 {
		t.Fatalf("Upgraded is %+v, want one for clip 1 replacing a Bluray-1080p file", up)
	}
	if got := noticetest.Of(w.notices, importedRow); len(got) != 1 {
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
	hold := func(ctx context.Context, s Step) {
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
	if got := noticetest.Of(w.notices, importedRow); len(got) != 0 {
		t.Errorf("an Import its Work's delete ended recorded Imported %+v", got)
	}
}

package storing

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/mediatest"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

type walking struct {
	disk   *testDisk
	store  *Store
	holder *holder
	remote Remote
	at     At
	folder string
}

func newWalking(t *testing.T, prober media.Prober) walking {
	t.Helper()
	d := newTestDisk(t)
	id := storage.NewUUID()
	w := walking{disk: d, at: At{Node: node, Root: id}, folder: marked(t, id)}
	w.store = NewStore(Config{Disk: d, Timings: Timings{Stat: 10 * time.Second}, Prober: prober, ProbeTimeout: time.Minute})
	w.holder = running(t, w.store, intentOf(rootAt(id, w.folder, false)))
	w.holder.next(t)
	w.remote = remoteOf(w.store)
	return w
}

func (w walking) file(t *testing.T, rel, body string) string {
	t.Helper()
	return write(t, filepath.Join(w.folder, filepath.FromSlash(rel)), body)
}

// pages walks the whole root, limit entries a page.
func (w walking) pages(t *testing.T, limit int) []Page {
	t.Helper()
	var pages []Page
	var after Cursor
	for {
		p, err := w.remote.Walk(t.Context(), w.at, after, limit)
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, p)
		if p.Done {
			return pages
		}
		after = p.Next
	}
}

func paths(pages ...Page) []string {
	var out []string
	for _, p := range pages {
		for _, e := range p.Entries {
			out = append(out, e.Path.String())
		}
	}
	return out
}

func TestAWalkOrdersEachFolderByItsNames(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		for _, rel := range []string{"A-1/y.mkv", "A B/x.mkv", "A/z.mkv", "A/Season 1/e.mkv", "Heat (1995).mkv"} {
			w.file(t, rel, "film")
		}
		want := []string{"A/Season 1/e.mkv", "A/z.mkv", "A B/x.mkv", "A-1/y.mkv", "Heat (1995).mkv"}
		if got := paths(w.pages(t, 1000)...); !slices.Equal(got, want) {
			t.Errorf("one page lists %q, want %q", got, want)
		}
		pages := w.pages(t, 1)
		if got := paths(pages...); !slices.Equal(got, want) {
			t.Errorf("a page per file lists %q, want %q", got, want)
		}
		if len(pages) != len(want)+1 || len(pages[len(want)].Entries) != 0 {
			t.Errorf("a page per file took %d pages, want %d and a last empty one", len(pages), len(want)+1)
		}
	})
}

func TestAWalkResumesPastAnEntryGoneSinceItsPage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		for _, rel := range []string{"A/Season 1/e1.mkv", "A/Season 1/e2.mkv", "A/z.mkv", "B/b.mkv"} {
			w.file(t, rel, "film")
		}
		first, err := w.remote.Walk(t.Context(), w.at, Cursor{}, 1)
		if err != nil || !slices.Equal(paths(first), []string{"A/Season 1/e1.mkv"}) {
			t.Fatalf("first page = %q, %v", paths(first), err)
		}
		if err := os.RemoveAll(filepath.Join(w.folder, "A", "Season 1")); err != nil {
			t.Fatal(err)
		}
		rest, err := w.remote.Walk(t.Context(), w.at, first.Next, 1000)
		if err != nil || !rest.Done || !slices.Equal(paths(rest), []string{"A/z.mkv", "B/b.mkv"}) {
			t.Errorf("the page after a removed folder = %q, done %v, %v; want the files after it", paths(rest), rest.Done, err)
		}
	})
}

func TestAWalkReadsWhatAScanCompares(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		heat := w.file(t, "Heat (1995)/Heat (1995).mkv", "a film")
		modified := time.Date(2025, 11, 3, 21, 4, 5, 678_901_234, time.UTC)
		if err := os.Chtimes(heat, modified, modified); err != nil {
			t.Fatal(err)
		}
		p := w.pages(t, 1000)[0]
		if len(p.Entries) != 1 {
			t.Fatalf("entries = %+v, want one", p.Entries)
		}
		e := p.Entries[0]
		if e.Size != 6 || !e.Seen.Modified.Equal(modified.Truncate(time.Millisecond)) || e.Seen.ID.IsZero() || e.Seen.ID.Volume == "" {
			t.Errorf("the file reads %+v, want 6 bytes modified %v with an id", e, modified.Truncate(time.Millisecond))
		}
		if want := hostFolds(t, w.folder); p.CaseFold != want {
			t.Errorf("the root reads CaseFold %v, want the host's %v", p.CaseFold, want)
		}
		if err := os.Rename(heat, filepath.Join(w.folder, "Heat (1995)", "Heat.mkv")); err != nil {
			t.Fatal(err)
		}
		if after := w.pages(t, 1000)[0].Entries[0]; after.Path.String() != "Heat (1995)/Heat.mkv" || !after.Seen.Same(e.Seen) {
			t.Errorf("renamed, the file reads %+v, want %+v at its new path", after, e.Seen)
		}
	})
}

func TestAWalkReadsAFoldingRootAndAVolumeWithoutIDs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "Heat.mkv", "film")
		w.disk.foldCase, w.disk.noIDs = true, true
		p := w.pages(t, 1000)[0]
		if !p.CaseFold {
			t.Error("a root that finds its marker in upper case reads case-sensitive")
		}
		if len(p.Entries) != 1 || p.Entries[0].Seen.ID != (FileID{}) || p.Entries[0].Seen.Modified.IsZero() {
			t.Errorf("entries = %+v, want one with no id and a modification time", p.Entries)
		}
	})
}

func TestAWalkSkipsWhatALibraryDoesntHold(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		for _, rel := range []string{
			HoldingDir + "/0190.mkv", "@eaDir/Heat.mkv", "Heat/Sample/Heat.mkv", "Heat/Featurettes/Making of.mkv",
			"Heat/Heat-trailer.mkv", "Heat/notes.nfo", "Heat/.Heat.mkv", "Heat/Heat.mkv",
		} {
			w.file(t, rel, "film")
		}
		elsewhere := t.TempDir()
		write(t, filepath.Join(elsewhere, "Ronin.mkv"), "film")
		outside := write(t, filepath.Join(t.TempDir(), "Heat.mkv"), "a longer film")
		want := []string{"Heat/Heat.mkv"}
		if err := os.Symlink(elsewhere, filepath.Join(w.folder, "Linked")); err == nil {
			if err := os.Symlink(outside, filepath.Join(w.folder, "Heat", "Heat link.mkv")); err != nil {
				t.Fatal(err)
			}
			want = []string{"Heat/Heat link.mkv", "Heat/Heat.mkv"}
		} else {
			t.Logf("no symlink on this runner: %v", err)
		}
		p := w.pages(t, 1000)[0]
		if got := paths(p); !slices.Equal(got, want) {
			t.Errorf("walk lists %q, want %q", got, want)
		}
		if len(want) == 2 && p.Entries[0].Size != int64(len("a longer film")) {
			t.Errorf("a linked file reads %d bytes, want its target's %d", p.Entries[0].Size, len("a longer film"))
		}
	})
}

func TestAWalkSendsNamesItCantCarryToUnread(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("only Linux keeps names that aren't UTF-8 or hold a backslash")
	}
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		for _, rel := range []string{"Bad\xff/Heat.mkv", "A: Movie/Heat.mkv", `Back\slash.mkv`, "Heat/Heat\xfe.mkv", "Heat/notes\xff.nfo", "Heat/Heat.mkv", "Locked/Heat.mkv"} {
			w.file(t, rel, "film")
		}
		w.disk.flag(w.disk.unreadable, filepath.Join(w.folder, "Locked"))
		p := w.pages(t, 1000)[0]
		want := []Unread{
			{"A: Movie", UnreadRefused},
			{"Back\\slash.mkv", UnreadRefused},
			{"Bad�", UnreadNotUTF8},
			{"Heat/Heat�.mkv", UnreadNotUTF8},
			{"Locked", UnreadDenied},
		}
		if !slices.Equal(p.Unread, want) {
			t.Errorf("unread = %q, want %q", p.Unread, want)
		}
		if got := paths(p); !slices.Equal(got, []string{"Heat/Heat.mkv"}) {
			t.Errorf("walk lists %q, want only Heat/Heat.mkv", got)
		}
	})
}

func TestAWalkPagesASlowShare(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		for _, rel := range []string{"A/a.mkv", "B/b.mkv", "C/c.mkv"} {
			w.file(t, rel, "film")
		}
		w.disk.slow[filepath.Join(w.folder, "A")] = 6 * time.Second
		w.disk.slow[filepath.Join(w.folder, "B")] = 6 * time.Second
		start := time.Now()
		first, err := w.remote.Walk(t.Context(), w.at, Cursor{}, 1000)
		if err != nil || first.Done || !slices.Equal(paths(first), []string{"A/a.mkv", "B/b.mkv"}) || time.Since(start) != 12*time.Second {
			t.Fatalf("a page past the stat deadline = %q, done %v, %v after %v; want it to end after B at 12s", paths(first), first.Done, err, time.Since(start))
		}
		rest, err := w.remote.Walk(t.Context(), w.at, first.Next, 1000)
		if err != nil || !rest.Done || !slices.Equal(paths(rest), []string{"C/c.mkv"}) {
			t.Errorf("the next page = %q, done %v, %v; want C", paths(rest), rest.Done, err)
		}
	})
}

func TestAHungFolderFailsTheWalkFast(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "A/a.mkv", "film")
		w.file(t, "Hung/b.mkv", "film")
		answer := w.disk.hang(filepath.Join(w.folder, "Hung"))
		defer answer()
		start := time.Now()
		_, err := w.remote.Walk(t.Context(), w.at, Cursor{}, 1000)
		if fe := folderErr(t, err); fe.Kind != FolderUnresponsive || time.Since(start) != 10*time.Second {
			t.Errorf("a walk into a hung folder = %+v after %v, want unresponsive after 10s", fe, time.Since(start))
		}
		start = time.Now()
		_, err = w.remote.Walk(t.Context(), w.at, Cursor{}, 1000)
		if nr := notReady(t, err); nr.State != Unresponsive || time.Since(start) != 0 {
			t.Errorf("the next walk = %+v after %v, want unresponsive at once", nr, time.Since(start))
		}
	})
}

func TestAWalkReResolvesItsRootEachPage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "A/a.mkv", "film")
		w.file(t, "B/b.mkv", "film")
		first, err := w.remote.Walk(t.Context(), w.at, Cursor{}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filepath.Join(w.folder, MarkerName)); err != nil {
			t.Fatal(err)
		}
		_, err = w.remote.Walk(t.Context(), w.at, first.Next, 1)
		if nr := notReady(t, err); nr.State != Unmounted {
			t.Errorf("a page after the marker went = %+v, want unmounted", nr)
		}
		_, err = w.remote.Walk(t.Context(), At{Node: node, Root: storage.NewUUID()}, Cursor{}, 1)
		if nr := notReady(t, err); !nr.Unknown {
			t.Errorf("a walk of a root not in the intent = %+v, want unknown", nr)
		}
		res, err := w.store.Walk(t.Context(), &storagev1.WalkRequest{RootId: w.at.Root.Bytes(), Limit: 1001})
		if rpc.CodeOf(err) != rpc.CodeInvalidArgument {
			t.Errorf("a page of 1001 = %v, %v; want invalid argument", res, err)
		}
	})
}

func TestVerifyInARootReadsItsMarkerFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "Heat/Heat.mkv", "film")
		files := []Recorded{{Sits: mustRel(t, "Heat/Heat.mkv"), Size: 4}, {Sits: mustRel(t, "Ronin/Ronin.mkv"), Size: 4}}
		missing, err := w.remote.VerifyIn(t.Context(), w.at, files, func(int) {})
		if err != nil || !slices.Equal(missing, []MissingFile{{1, FileMissing}}) {
			t.Errorf("VerifyIn = %v, %v; want Ronin missing", missing, err)
		}
		if err := os.Remove(filepath.Join(w.folder, MarkerName)); err != nil {
			t.Fatal(err)
		}
		_, err = w.remote.VerifyIn(t.Context(), w.at, files, func(int) {})
		if nr := notReady(t, err); nr.State != Unmounted {
			t.Errorf("VerifyIn on an unmounted root = %+v, want unmounted", nr)
		}
	})
}

func TestProbeReadsARootFile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		heat := media.Record{
			Version: media.Version, Container: "matroska,webm", Duration: 170 * time.Minute,
			Video: media.Video{Codec: "hevc", Width: 3840, Height: 2160, BitDepth: 10, HDR: []release.HDR{"HDR10"}},
			Audio: []media.Audio{{Codec: "truehd", Channels: 8, Language: "en"}},
		}
		w := newWalking(t, mediatest.Prober{
			Of:   map[string]media.Record{"Heat/Heat.mkv": heat},
			Fail: map[string]string{"Ronin/Ronin.mkv": "Invalid data found when processing input"},
		})
		w.file(t, "Heat/Heat.mkv", "film")
		w.file(t, "Ronin/Ronin.mkv", "film")
		got, err := w.remote.Probe(t.Context(), w.at, mustRel(t, "Heat/Heat.mkv"))
		if err != nil || !reflect.DeepEqual(got, heat) {
			t.Errorf("Probe = %+v, %v; want %+v", got, err, heat)
		}
		_, err = w.remote.Probe(t.Context(), w.at, mustRel(t, "Ronin/Ronin.mkv"))
		if u, ok := errors.AsType[*media.UnreadableError](err); !ok || u.Line != "Invalid data found when processing input" {
			t.Errorf("Probe of a file ffprobe can't read = %v, want its line", err)
		}
		_, err = w.remote.Probe(t.Context(), w.at, mustRel(t, "Gone/Gone.mkv"))
		if fe := folderErr(t, err); fe.Kind != FolderNotFound {
			t.Errorf("Probe of a file gone since the walk = %+v, want not found", fe)
		}
	})
}

func TestAPlacedFileReadsAsTheNextWalkReadsIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPlacing(t, false)
		holding := mustRel(t, HoldingDir+"/0190.mkv")
		if _, err := p.remote.Place(t.Context(), p.at, p.payload(), holding, true); err != nil {
			t.Fatal(err)
		}
		moved, err := p.remote.Move(t.Context(), p.at, holding, mustRel(t, "Heat (1995)/Heat (1995).mkv"), false)
		if err != nil {
			t.Fatal(err)
		}
		page, err := p.remote.Walk(t.Context(), p.at, Cursor{}, 1000)
		if err != nil || len(page.Entries) != 1 || !page.Entries[0].Seen.Same(moved.Seen) || page.Entries[0].Size != moved.Size {
			t.Errorf("the walk reads %+v, %v; want the moved file as %+v", page.Entries, err, moved)
		}
	})
}

// hostFolds reports whether folder's own file system finds its marker
// asked for in upper case, as APFS and NTFS do by default.
func hostFolds(t *testing.T, folder string) bool {
	t.Helper()
	marker, err := os.Stat(filepath.Join(folder, MarkerName))
	if err != nil {
		t.Fatal(err)
	}
	upper, err := os.Stat(filepath.Join(folder, strings.ToUpper(MarkerName)))
	return err == nil && os.SameFile(marker, upper)
}

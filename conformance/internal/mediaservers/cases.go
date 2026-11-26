package mediaservers

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/storing"
)

// Poll is the wait between looks while a case waits for the server.
const Poll = 250 * time.Millisecond

// Files are the files one case placed.
type Files struct {
	t     *testing.T
	tg    Target
	disk  Disk
	lib   string
	root  link.ServerPath
	rels  []storing.Rel
	moved map[string]storing.Rel
}

// Case is a new case's files, in the target's library.
func (tg Target) Case(t *testing.T) *Files {
	t.Helper()
	f := &Files{t: t, tg: tg, disk: tg.Disk, lib: tg.Library, root: tg.Folder, moved: map[string]storing.Rel{}}
	return f
}

// In is f's files in l, such as the target's fresh or second library.
func (f *Files) In(l Library) *Files {
	f.disk, f.lib, f.root = l.Disk, l.Library, l.Folder
	return f
}

// Library is the library f's files are in.
func (f *Files) Library() Library { return Library{Library: f.lib, Folder: f.root, Disk: f.disk} }

// Rel is the video, or with ext a Sidecar, of the movie title in a
// folder of its own.
func Rel(t *testing.T, title, ext string) storing.Rel {
	t.Helper()
	name := title + " (2001)"
	r, err := storing.ParseRel("SlipStream conformance/" + name + "/" + name + ext)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// Elsewhere is r in another top folder, the same file name in a
// different folder.
func Elsewhere(t *testing.T, r storing.Rel) storing.Rel {
	t.Helper()
	moved, err := storing.ParseRel("SlipStream conformance moved/" + strings.TrimPrefix(r.String(), "SlipStream conformance/"))
	if err != nil {
		t.Fatal(err)
	}
	return moved
}

func (f *Files) At(r storing.Rel) link.ServerPath { return f.root.Join(r) }

func (f *Files) Place(rels ...storing.Rel) {
	f.t.Helper()
	for _, r := range rels {
		if err := f.disk.Place(r); err != nil {
			f.t.Fatal(err)
		}
		f.rels = append(f.rels, r)
	}
}

func (f *Files) Move(from, to storing.Rel) {
	f.t.Helper()
	if err := f.disk.Move(from, to); err != nil {
		f.t.Fatal(err)
	}
	f.moved[from.String()] = to
}

func (f *Files) Remove(r storing.Rel) {
	f.t.Helper()
	if err := f.disk.Remove(r); err != nil {
		f.t.Fatal(err)
	}
	f.rels = slices.DeleteFunc(f.rels, func(x storing.Rel) bool { return x == r })
}

// Tell tells the server about rels, each gone unless the case placed it.
func (f *Files) Tell(rels ...storing.Rel) {
	f.t.Helper()
	told := make([]Told, len(rels))
	for i, r := range rels {
		told[i] = Told{Path: f.At(r), Gone: !slices.Contains(f.rels, r)}
	}
	if err := f.tg.Server.Tell(f.t.Context(), f.lib, told); err != nil {
		f.t.Fatal(err)
	}
}

// Clear removes the case's files and tells the server, so a real server
// forgets them. A case defers it.
func (f *Files) Clear() {
	var gone []Told
	for _, r := range f.rels {
		if to, ok := f.moved[r.String()]; ok {
			r = to
		}
		if err := f.disk.Remove(r); err == nil {
			gone = append(gone, Told{Path: f.At(r), Gone: true})
		}
	}
	if len(gone) > 0 {
		if err := f.tg.Server.Tell(f.t.Context(), f.lib, gone); err != nil {
			f.t.Logf("tell the server the case's files are gone: %v", err)
		}
	}
}

// Found waits for the item at r and answers it.
func (f *Files) Found(r storing.Rel) Item {
	f.t.Helper()
	var it Item
	Eventually(f.t, f.tg, fmt.Sprintf("an item at %s", f.At(r)), func() (bool, error) {
		var ok bool
		var err error
		it, ok, err = f.tg.Server.ItemAt(f.t.Context(), f.lib, f.At(r))
		return ok, err
	})
	return it
}

// Gone waits until the server holds no item at r.
func (f *Files) Gone(r storing.Rel) {
	f.t.Helper()
	Eventually(f.t, f.tg, fmt.Sprintf("no item at %s", f.At(r)), func() (bool, error) {
		_, ok, err := f.tg.Server.ItemAt(f.t.Context(), f.lib, f.At(r))
		return !ok, err
	})
}

// Eventually looks every Poll until ok, and fails the test past the
// target's Bound.
func Eventually(t *testing.T, tg Target, what string, ok func() (bool, error)) {
	t.Helper()
	deadline := time.Now().Add(tg.Bound)
	for {
		done, err := ok()
		if err != nil {
			t.Fatalf("waiting for %s: %v", what, err)
		}
		if done {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("no %s within %v", what, tg.Bound)
		}
		time.Sleep(Poll)
	}
}

// Case is one named case. The cases run one after another in one test,
// with no subtests, because a synctest bubble refuses t.Run.
type Case struct {
	Name string
	Run  func(*testing.T)
}

// Each runs cases in order, logging each name first. A case that calls Skip
// ends alone and the cases after it still run.
func Each(t *testing.T, cases []Case) {
	t.Helper()
	for _, c := range cases {
		t.Logf("case %s", c.Name)
		runCase(t, c)
	}
}

type skipped string

// Skip ends the running case only. t.Skip would end every case after it too,
// since they share one *testing.T.
func Skip(t *testing.T, why string) {
	t.Helper()
	panic(skipped(why))
}

func runCase(t *testing.T, c Case) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		why, ok := r.(skipped)
		if !ok {
			panic(r)
		}
		t.Logf("case %s skipped: %s", c.Name, why)
	}()
	c.Run(t)
}

// Shared runs every case the kinds share.
func Shared(t *testing.T, tg Target) {
	t.Helper()
	var cases []Case
	for _, c := range []struct {
		name string
		run  func(*testing.T, Target)
	}{
		{"Identity", Identity}, {"LibrariesAndFolders", LibrariesAndFolders}, {"AddedShowsUp", AddedShowsUp},
		{"ReplacedKeepsItem", ReplacedKeepsItem}, {"RemovedLeaves", RemovedLeaves}, {"FreshFolderNeedsScan", FreshFolderNeedsScan},
		{"ScanEndSignal", ScanEndSignal}, {"CarryKeepsWatchData", CarryKeepsWatchData}, {"SidecarRemoved", SidecarRemoved},
		{"Sessions", ReadsDecisions}, {"CopyMovedLands", CopyMovedLands}, {"Exposed", Exposed},
	} {
		cases = append(cases, Case{Name: c.name, Run: func(t *testing.T) { c.run(t, tg) }})
	}
	cases = append(cases, moveCases(tg)...)
	Each(t, cases)
}

// CopyMovedLands copies a file into the second library and deletes it
// from the first: the server then holds it in the second and not in the
// first.
func CopyMovedLands(t *testing.T, tg Target) {
	if tg.Second == nil {
		Skip(t, "the target has no second library")
	}
	from, to := tg.Case(t), tg.Case(t).In(*tg.Second)
	defer from.Clear()
	defer to.Clear()
	r := Rel(t, "Copied", ".mkv")
	from.Place(r)
	from.Tell(r)
	from.Found(r)
	if err := tg.Server.CopyMoved(t.Context(), from.Library(), to.Library(), r); err != nil {
		t.Fatal(err)
	}
	to.rels = append(to.rels, r)
	to.Found(r)
	from.Gone(r)
}

func Identity(t *testing.T, tg Target) {
	id, err := tg.Server.Reach(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if id.ID == "" || id.Version.IsZero() {
		t.Fatalf("identity %+v has no id or no version", id)
	}
}

func LibrariesAndFolders(t *testing.T, tg Target) {
	libs, err := tg.Server.Libraries(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(libs, func(l link.Library) bool { return l.Key == tg.Library })
	if i < 0 {
		t.Fatalf("no library %s in %+v", tg.Library, libs)
	}
	if !slices.ContainsFunc(libs[i].Folders, func(f link.ServerPath) bool { return f.Key() == tg.Folder.Key() }) {
		t.Fatalf("library %s has folders %v, without %s", tg.Library, libs[i].Folders, tg.Folder)
	}
}

func AddedShowsUp(t *testing.T, tg Target) {
	f := tg.Case(t)
	defer f.Clear()
	r := Rel(t, "Added", ".mkv")
	f.Place(r)
	f.Tell(r)
	f.Found(r)
}

func ReplacedKeepsItem(t *testing.T, tg Target) {
	f := tg.Case(t)
	defer f.Clear()
	r := Rel(t, "Replaced", ".mkv")
	f.Place(r)
	f.Tell(r)
	before := f.Found(r)
	f.Remove(r)
	f.Place(r)
	f.Tell(r)
	if after := f.Found(r); after.Key != before.Key {
		t.Fatalf("the replaced file is item %s, was %s", after.Key, before.Key)
	}
}

func RemovedLeaves(t *testing.T, tg Target) {
	f := tg.Case(t)
	defer f.Clear()
	r := Rel(t, "Removed", ".mkv")
	f.Place(r)
	f.Tell(r)
	f.Found(r)
	f.Remove(r)
	f.Tell(r)
	f.Gone(r)
}

// FreshFolderNeedsScan is the fresh folder rule: a change into a library
// never scanned with content is ignored until a full scan. A server that
// takes it anyway shows the item without one.
func FreshFolderNeedsScan(t *testing.T, tg Target) {
	if tg.Fresh == nil {
		Skip(t, "a real server's fresh library is a recorded manual run, since SlipStream creates no library")
	}
	f := tg.Case(t).In(*tg.Fresh)
	defer f.Clear()
	r := Rel(t, "Fresh", ".mkv")
	f.Place(r)
	f.Tell(r)
	if tg.FreshTakesChange {
		f.Found(r)
		return
	}
	time.Sleep(tg.Settle)
	if _, ok, err := tg.Server.ItemAt(t.Context(), tg.Fresh.Library, f.At(r)); err != nil || ok {
		t.Fatalf("a change into a never-scanned library made an item (err %v)", err)
	}
	if err := tg.Server.Scan(t.Context(), tg.Fresh.Library); err != nil {
		t.Fatal(err)
	}
	Eventually(t, tg, "the full scan's end", func() (bool, error) { return tg.Server.ScanEnded(t.Context(), tg.Fresh.Library) })
	f.Found(r)
}

func ScanEndSignal(t *testing.T, tg Target) {
	if err := tg.Server.Scan(t.Context(), tg.Library); err != nil {
		t.Fatal(err)
	}
	ended, err := tg.Server.ScanEnded(t.Context(), tg.Library)
	if err != nil {
		t.Fatal(err)
	}
	if ended {
		t.Fatal("the scan read as ended right after it was asked for")
	}
	Eventually(t, tg, "the full scan's end", func() (bool, error) { return tg.Server.ScanEnded(t.Context(), tg.Library) })
}

// CarryKeepsWatchData moves a played file in a collection to another
// folder. The item now at the new path is still played and in the
// collection.
func CarryKeepsWatchData(t *testing.T, tg Target) {
	f := tg.Case(t)
	defer f.Clear()
	from := Rel(t, "Carried", ".mkv")
	to := Elsewhere(t, from)
	f.Place(from)
	f.Tell(from)
	it := f.Found(from)
	if err := tg.Server.Watched(t.Context(), it.Key, "SlipStream conformance"); err != nil {
		t.Fatal(err)
	}
	if err := tg.Server.Moved(t.Context(), tg.Library, []link.ServerPath{f.At(from)}, []link.ServerPath{f.At(to)}, func() { f.Move(from, to) }); err != nil {
		t.Fatal(err)
	}
	now := f.Found(to)
	if !now.Played || !slices.Contains(now.Collections, "SlipStream conformance") {
		t.Fatalf("the moved item has played %v and collections %v", now.Played, now.Collections)
	}
	f.Gone(from)
}

// SidecarRemoved places a video with an .srt beside it. The server shows
// the item, then the .srt on it, and drops the .srt once told it is gone.
// Jellyfin lists a new item before it has probed the file, so the .srt
// may show some time after the item (online run 2026-11-14 on 12.4).
func SidecarRemoved(t *testing.T, tg Target) {
	f := tg.Case(t)
	defer f.Clear()
	video, srt := Rel(t, "Subtitled", ".mkv"), Rel(t, "Subtitled", ".en.srt")
	f.Place(video, srt)
	f.Tell(video)
	f.Found(video)
	Eventually(t, tg, "the .srt on the item", func() (bool, error) {
		it, ok, err := tg.Server.ItemAt(t.Context(), tg.Library, f.At(video))
		return ok && it.Sidecars == 1, err
	})
	f.Remove(srt)
	if err := tg.Server.SidecarRemoved(t.Context(), tg.Library, f.At(video), f.At(srt)); err != nil {
		t.Fatal(err)
	}
	Eventually(t, tg, "the Sidecar gone", func() (bool, error) {
		it, ok, err := tg.Server.ItemAt(t.Context(), tg.Library, f.At(video))
		return ok && it.Sidecars == 0, err
	})
}

// Exposed is what S51 (Reachability diagnostic) reads of a server: once a
// recorded run measured the kind's names, each fact the kind reads has a
// value, and Plex reports its mapping state.
func Exposed(t *testing.T, tg Target) {
	if !mediaserverlink.ExposedMeasured(tg.Kind) {
		Skip(t, "waits for a recorded run against the product")
	}
	x, err := tg.Server.Exposed(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if x.Remote == link.Unknown || x.IPv6 == link.Unknown {
		t.Errorf("remote access reads %v and IPv6 %v, want both read", x.Remote, x.IPv6)
	}
	if tg.Kind == "plex" && (x.Mapping == nil || x.Mapping.State == "" || x.MappingRoot == "") {
		t.Errorf("Plex's mapping state reads %+v and %q", x.Mapping, x.MappingRoot)
	}
}

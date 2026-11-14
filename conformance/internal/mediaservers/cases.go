package mediaservers

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

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

// InFresh is f's files in the target's fresh library.
func (f *Files) InFresh() *Files {
	f.disk, f.lib, f.root = f.tg.Fresh.Disk, f.tg.Fresh.Library, f.tg.Fresh.Folder
	return f
}

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

// Tell tells the server about rels.
func (f *Files) Tell(rels ...storing.Rel) {
	f.t.Helper()
	paths := make([]link.ServerPath, len(rels))
	for i, r := range rels {
		paths[i] = f.At(r)
	}
	if err := f.tg.Server.Tell(f.t.Context(), f.lib, paths); err != nil {
		f.t.Fatal(err)
	}
}

// Clear removes the case's files and tells the server, so a real server
// forgets them. A case defers it.
func (f *Files) Clear() {
	var gone []link.ServerPath
	for _, r := range f.rels {
		if to, ok := f.moved[r.String()]; ok {
			r = to
		}
		if err := f.disk.Remove(r); err == nil {
			gone = append(gone, f.At(r))
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

// Each runs cases in order, logging each name first.
func Each(t *testing.T, cases []Case) {
	t.Helper()
	for _, c := range cases {
		t.Logf("case %s", c.Name)
		c.Run(t)
	}
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
		{"Sessions", Sessions},
	} {
		cases = append(cases, Case{Name: c.name, Run: func(t *testing.T) { c.run(t, tg) }})
	}
	Each(t, cases)
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
// never scanned with content is ignored until a full scan.
func FreshFolderNeedsScan(t *testing.T, tg Target) {
	if tg.Fresh == nil {
		t.Skip("a real server's fresh library is a recorded manual run, since SlipStream creates no library: https://github.com/jatassi/SlipStream/issues/160")
	}
	f := tg.Case(t).InFresh()
	defer f.Clear()
	r := Rel(t, "Fresh", ".mkv")
	f.Place(r)
	f.Tell(r)
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
// folder. The item now at the new path keeps its play count and
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
	f.Move(from, to)
	if err := tg.Server.Moved(t.Context(), tg.Library, []link.ServerPath{f.At(from)}, []link.ServerPath{f.At(to)}); err != nil {
		t.Fatal(err)
	}
	now := f.Found(to)
	if now.PlayCount != 1 || !slices.Contains(now.Collections, "SlipStream conformance") {
		t.Fatalf("the moved item has play count %d and collections %v", now.PlayCount, now.Collections)
	}
	f.Gone(from)
}

func SidecarRemoved(t *testing.T, tg Target) {
	f := tg.Case(t)
	defer f.Clear()
	video, srt := Rel(t, "Subtitled", ".mkv"), Rel(t, "Subtitled", ".en.srt")
	f.Place(video, srt)
	f.Tell(video)
	if it := f.Found(video); it.Sidecars != 1 {
		t.Fatalf("the item has %d Sidecars, want the .srt", it.Sidecars)
	}
	f.Remove(srt)
	if err := tg.Server.SidecarRemoved(t.Context(), tg.Library, f.At(video), f.At(srt)); err != nil {
		t.Fatal(err)
	}
	Eventually(t, tg, "the Sidecar gone", func() (bool, error) {
		it, ok, err := tg.Server.ItemAt(t.Context(), tg.Library, f.At(video))
		return ok && it.Sidecars == 0, err
	})
}

func Sessions(t *testing.T, tg Target) {
	if tg.Play == nil {
		t.Skip("a real server's sessions need a player: https://github.com/jatassi/SlipStream/issues/160")
	}
	f := tg.Case(t)
	defer f.Clear()
	r := Rel(t, "Playing", ".mkv")
	f.Place(r)
	f.Tell(r)
	f.Found(r)
	for _, d := range link.Decisions() {
		stop := tg.Play(t, f.At(r), d)
		sessions, err := tg.Server.Sessions(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(sessions, func(s link.Session) bool { return s.Path.Key() == f.At(r).Key() })
		if i < 0 || sessions[i].Decision != d {
			t.Fatalf("playing %s as %s, the sessions are %+v", f.At(r), d, sessions)
		}
		stop()
		Eventually(t, tg, "the session's end", func() (bool, error) {
			sessions, err := tg.Server.Sessions(t.Context())
			return !slices.ContainsFunc(sessions, func(s link.Session) bool { return s.Path.Key() == f.At(r).Key() }), err
		})
	}
}

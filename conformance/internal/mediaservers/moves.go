package mediaservers

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/storing"
)

// moveCases are the cases of S48 (Moves): a file copied to another root
// and deleted from its source, as a Move does.
func moveCases(tg Target) []Case {
	var out []Case
	for _, c := range []struct {
		name string
		run  func(*testing.T, Target)
	}{
		{"CopyMovedKeepsWatchData", CopyMovedKeepsWatchData},
		{"PlexCopyInItsSection", PlexCopyInItsSection},
		{"PlexCopyToAnotherSection", PlexCopyToAnotherSection},
	} {
		out = append(out, Case{Name: c.name, Run: func(t *testing.T) { c.run(t, tg) }})
	}
	return out
}

const moveCollection = "SlipStream conformance"

// unmeasured skips a Plex case whose outcome the recorded Plex run
// decides.
func unmeasured(t *testing.T, tg Target) {
	if tg.Kind == "plex" && !mediaserverlink.PlexMoveMeasured() {
		Skip(t, "waits for the recorded Plex run")
	}
}

// CopyMovedKeepsWatchData watches a file, copies it into the second
// library and deletes it from the first, as a Move across libraries does:
// the item in the second library holds the play and the collection.
func CopyMovedKeepsWatchData(t *testing.T, tg Target) {
	if tg.Second == nil {
		Skip(t, "the target has no second library")
	}
	unmeasured(t, tg)
	from, to := tg.Case(t), tg.Case(t).In(*tg.Second)
	defer from.Clear()
	defer to.Clear()
	r := Rel(t, "Copied and deleted", ".mkv")
	from.Place(r)
	from.Tell(r)
	it := from.Found(r)
	if err := tg.Server.Watched(t.Context(), it.Key, moveCollection); err != nil {
		t.Fatal(err)
	}
	if err := tg.Server.CopyMoved(t.Context(), from.Library(), to.Library(), r); err != nil {
		t.Fatal(err)
	}
	now := to.Found(r)
	if !now.Played || !slices.Contains(now.Collections, moveCollection) {
		t.Fatalf("the moved item has played %v and collections %v", now.Played, now.Collections)
	}
	from.Gone(r)
}

// PlexCopyInItsSection copies a file to another folder of its section
// while the source is still there: Plex shows the copy as the recorded
// run measured, a second Part of the item or a new item.
func PlexCopyInItsSection(t *testing.T, tg Target) {
	if tg.Kind != "plex" {
		Skip(t, "a Plex case")
	}
	unmeasured(t, tg)
	f := tg.Case(t)
	defer f.Clear()
	r := Rel(t, "Copied in its section", ".mkv")
	f.Place(r)
	f.Tell(r)
	it := f.Found(r)
	to := Elsewhere(t, r)
	f.copyFrom(f, r, to)
	f.Tell(to)
	now := f.Found(to)
	if same := now.Key == it.Key; same != mediaserverlink.PlexSameSectionPart() {
		t.Fatalf("the copy's item %s against the source's %s: same item %v", now.Key, it.Key, same)
	}
}

// PlexCopyToAnotherSection copies a file into the second section: Plex
// shows it there as a new item, found by its path.
func PlexCopyToAnotherSection(t *testing.T, tg Target) {
	if tg.Kind != "plex" {
		Skip(t, "a Plex case")
	}
	if tg.Second == nil {
		Skip(t, "the target has no second library")
	}
	unmeasured(t, tg)
	from, to := tg.Case(t), tg.Case(t).In(*tg.Second)
	defer from.Clear()
	defer to.Clear()
	r := Rel(t, "Copied to another section", ".mkv")
	from.Place(r)
	from.Tell(r)
	it := from.Found(r)
	to.copyFrom(from, r, r)
	to.Tell(r)
	if now := to.Found(r); now.Key == it.Key {
		t.Fatalf("the copy in another section is the source's item %s", it.Key)
	}
}

// copyFrom copies src's file at from to to among f's files.
func (f *Files) copyFrom(src *Files, from, to storing.Rel) {
	f.t.Helper()
	in, err := src.disk.Open(from)
	if err != nil {
		f.t.Fatal(err)
	}
	defer func() { _ = in.Close() }()
	if err := f.disk.Write(to, in); err != nil {
		f.t.Fatal(err)
	}
	f.rels = append(f.rels, to)
}

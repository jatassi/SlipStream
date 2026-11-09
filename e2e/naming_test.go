package e2e

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// A file SlipStream doesn't track sits where the templates name The
// Lantern Keeper's Sized file, so its Grab fails at its probe with the
// path, before any bytes move, and the file stays as it was.
func TestAGrabWhoseNameAFileSlipStreamDoesntTrackHoldsFailsNameTaken(t *testing.T) {
	t.Parallel()
	w, _ := sized(t, "9000002", "Films", "Sized")
	const theirs = "movies/The Lantern Keeper (2019)/The Lantern Keeper (2019) - Sized.mkv"
	at := filepath.Join(w.m.MediaDir, filepath.FromSlash(theirs))
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte("theirs"), 0o600); err != nil {
		t.Fatal(err)
	}

	found := w.searched(t, w.search(t))
	if found.Outcome == nil || *found.Outcome != "grabbed" || found.GrabID == nil {
		t.Fatalf("the search ended %+v, want a release grabbed", found)
	}
	w.ended(t, *found.GrabID)
	if w.grab.Stage != "failed" || w.grab.Failure == nil || w.grab.Failure.Code != "name_taken" ||
		w.grab.Failure.Message != `a file SlipStream doesn't track is already at "The Lantern Keeper (2019)/The Lantern Keeper (2019) - Sized.mkv"` {
		t.Fatalf("the Grab ended %s with %+v after %v, want failed with its name taken", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	if got := w.grab.stages(); !slices.Equal(got[len(got)-2:], []string{"probing", "failed"}) {
		t.Fatalf("the Grab went through %v, want it to fail at its probe", got)
	}
	holdsOnly(t, w.m, theirs)
	if data, err := os.ReadFile(at); err != nil || string(data) != "theirs" {
		t.Fatalf("the file at the name holds %q, %v, want what it held before", data, err)
	}
}

package sim_test

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/sim/movestest"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/roots"
)

// absent fails t for each of rels the root in folder holds.
func absent(t *testing.T, w movestest.World, folder string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		if _, err := os.Stat(filepath.Join(w.Roots[folder].Path, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("%s holds %s: %v", folder, rel, err)
		}
	}
}

// TestAWorkDeletedMidMoveCancelsItsUnit deletes The Quiet Orchard, its
// files left where they are, while its Move to movies-f places: the unit
// ends canceled and undone, movies-f holds nothing of it, and movies-e
// keeps its files.
func TestAWorkDeletedMidMoveCancelsItsUnit(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		pause := movestest.PauseAt("place_file")
		w := movestest.Launch(t, pause.Option())
		batch := w.Start(t, "movies-f")
		pause.Reached(t)
		deleted := w.API().Admin("home", http.MethodPost, "/api/v1/works/delete", fmt.Sprintf(`{"workIds":[%q],"files":"leave"}`, w.Work))
		if deleted.Status != http.StatusOK {
			t.Fatalf("deleteWorks = %d %s, want 200", deleted.Status, deleted.Body)
		}
		pause.Resume()
		simtest.Eventually(t, "the unit is canceled and undone", func() bool {
			u := w.Unit(t, batch)
			return u.Phase == moves.PhaseCanceled && u.Undone
		})
		absent(t, w, "movies-f", movestest.Film, movestest.Subtitle)
		for rel, want := range w.Was {
			if got, err := os.ReadFile(filepath.Join(w.Roots["movies-e"].Path, filepath.FromSlash(rel))); err != nil || !bytes.Equal(got, want) {
				t.Errorf("movies-e lost %s (%d bytes, %v)", rel, len(got), err)
			}
		}
	})
}

// TestAFullDestinationWaitsForSpace fills seedbox's movies-d just as the
// Move to it takes its lane: the unit waits for space, and once the disk
// has room again it moves.
func TestAFullDestinationWaitsForSpace(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		pause := movestest.PauseAt("take_lane")
		w := movestest.Launch(t, pause.Option())
		batch := w.Start(t, "movies-d")
		pause.Reached(t)
		lift := w.Disk("seedbox", w.Roots["movies-d"].Path, fault.Volume{Total: 500 << 30, Free: 0})
		time.Sleep(roots.SampleEvery.Default + time.Minute)
		pause.Resume()
		simtest.Eventually(t, "the unit waits for space on seedbox", func() bool {
			u := w.Unit(t, batch)
			return u.Waiting == moves.WaitSpace
		})
		if u := w.Unit(t, batch); u.Words != "Waiting for free space on seedbox" {
			t.Errorf("the unit reads %q, want the space wait's words", u.Words)
		}
		w.Invariants(t, batch)
		lift()
		if u := w.Ended(t, batch); u.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v, want done", u.Phase, u.Failure)
		}
		w.Moved(t, "movies-d")
	})
}

// TestASourceChangedBeforeCleanupStays replaces the subtitle in movies-e
// after the switch, before cleanup removes it: cleanup leaves it, the
// unit ends done noting it, and movies-f keeps the bytes that moved.
func TestASourceChangedBeforeCleanupStays(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		pause := movestest.PauseAt("remove_source")
		w := movestest.Launch(t, pause.Option())
		batch := w.Start(t, "movies-f")
		pause.Reached(t)
		source := filepath.Join(w.Roots["movies-e"].Path, filepath.FromSlash(movestest.Subtitle))
		changed := []byte("1\n00:00:01,000 --> 00:00:02,000\nHello again\n")
		if err := os.WriteFile(source+".new", changed, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(source+".new", source); err != nil {
			t.Fatal(err)
		}
		pause.Resume()
		u := w.Ended(t, batch)
		if u.Phase != moves.PhaseDone || !slices.Contains(u.Notes.Skipped, movestest.Subtitle) {
			t.Fatalf("the unit ended %s noting %+v, want done with the subtitle skipped", u.Phase, u.Notes)
		}
		if got, err := os.ReadFile(source); err != nil || !bytes.Equal(got, changed) {
			t.Errorf("movies-e's subtitle is %q (%v), want the rewritten one kept", got, err)
		}
		placed := filepath.Join(w.Roots["movies-f"].Path, filepath.FromSlash(movestest.Subtitle))
		if got, err := os.ReadFile(placed); err != nil || !bytes.Equal(got, w.Was[movestest.Subtitle]) {
			t.Errorf("movies-f's subtitle is %q (%v), want the bytes that moved", got, err)
		}
		absent(t, w, "movies-e", movestest.Film)
	})
}

// TestAFileGoneAtTheSourceBeforeTheFinalPassIsDropped removes the
// subtitle from movies-e once both files are placed: the final pass drops
// it, its placed copy goes, and the film moves alone.
func TestAFileGoneAtTheSourceBeforeTheFinalPassIsDropped(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		pause := movestest.PauseAt("final_pass")
		w := movestest.Launch(t, pause.Option())
		batch := w.Start(t, "movies-f")
		pause.Reached(t)
		if err := os.Remove(filepath.Join(w.Roots["movies-e"].Path, filepath.FromSlash(movestest.Subtitle))); err != nil {
			t.Fatal(err)
		}
		pause.Resume()
		if u := w.Ended(t, batch); u.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v, want done", u.Phase, u.Failure)
		}
		w.Invariants(t, batch)
		absent(t, w, "movies-f", movestest.Subtitle)
		absent(t, w, "movies-e", movestest.Film, movestest.Subtitle)
		placed := filepath.Join(w.Roots["movies-f"].Path, filepath.FromSlash(movestest.Film))
		if got, err := os.ReadFile(placed); err != nil || !bytes.Equal(got, w.Was[movestest.Film]) {
			t.Errorf("movies-f holds %d bytes of the film (%v), want %d", len(got), err, len(w.Was[movestest.Film]))
		}
	})
}

// TestACancelAfterACrashBetweenAPlaceAndItsWriteLeavesNoStray stops the
// Controller as a Place has put the film in movies-f and before its Write
// records it, cancels the Move, and starts the Controller again: the undo
// removes the film no row names, and movies-e keeps both files.
func TestACancelAfterACrashBetweenAPlaceAndItsWriteLeavesNoStray(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		crash := movestest.CrashAt("place_file")
		w := movestest.Launch(t, crash.Option())
		batch := w.Start(t, "movies-f")
		crash.Reached(t)
		stray := w.StrayAt(t, "movies-f", movestest.Film)
		if _, _, err := w.Moves().CancelBatch(t.Context(), batch); err != nil {
			t.Fatal(err)
		}
		w.Restart(t, movestest.Fault{Name: "restart"})
		simtest.Eventually(t, "the canceled unit is undone", func() bool {
			u := w.Unit(t, batch)
			return u.Phase == moves.PhaseCanceled && u.Undone
		})
		if _, err := os.Stat(stray); !os.IsNotExist(err) {
			t.Errorf("the undo left the stray film in movies-f: %v", err)
		}
		w.Invariants(t, batch)
	})
}

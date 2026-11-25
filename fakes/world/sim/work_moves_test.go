package sim_test

import (
	"bytes"
	"context"
	"fmt"
	"maps"
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
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
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

// remove removes each of rels from the root in folder.
func remove(t *testing.T, w movestest.World, folder string, rels ...string) {
	t.Helper()
	for _, rel := range rels {
		if err := os.Remove(filepath.Join(w.Roots[folder].Path, filepath.FromSlash(rel))); err != nil {
			t.Fatal(err)
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

// TestAFilmGoneAtTheSourceBeforeItsPlaceFailsTheUnit removes the film
// from movies-e as its Move to movies-f is about to place it: the unit
// fails source_gone and is undone, movies-f holds nothing of it, and
// movies-e keeps its subtitle.
func TestAFilmGoneAtTheSourceBeforeItsPlaceFailsTheUnit(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		pause := movestest.PauseAt("place_file")
		w := movestest.Launch(t, pause.Option())
		batch := w.Start(t, "movies-f")
		pause.Reached(t)
		remove(t, w, "movies-e", movestest.Film)
		pause.Resume()
		if u := w.Ended(t, batch); u.Phase != moves.PhaseFailed || u.Failure == nil || u.Failure.Code != moves.SourceGone {
			t.Fatalf("the unit ended %s %+v, want failed source_gone", u.Phase, u.Failure)
		}
		simtest.Eventually(t, "the failed unit is undone", func() bool { return w.Unit(t, batch).Undone })
		absent(t, w, "movies-f", movestest.Film, movestest.Subtitle)
		if got, err := os.ReadFile(filepath.Join(w.Roots["movies-e"].Path, filepath.FromSlash(movestest.Subtitle))); err != nil || !bytes.Equal(got, w.Was[movestest.Subtitle]) {
			t.Errorf("movies-e lost its subtitle (%d bytes, %v)", len(got), err)
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

// upgrade searches The Quiet Orchard, whose 720p film the Scenario's
// 1080p release upgrades, waits until its Grab imported into movies-e,
// and answers the upgrade's path and bytes. A Move before its switch
// holds the upgrade's rename, so its bytes may still sit in incoming.
func upgrade(t *testing.T, w movestest.World) (rel string, b []byte) {
	t.Helper()
	simtest.Search(t, w.World, w.Work.String())
	simtest.ExpectImported(t, simtest.Await(t, w.World))
	var sits map[files.ID]files.Sitting
	err := w.Library(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		sits, err = files.Sits(ctx, r, []works.WorkID{w.Work})
		return err
	})
	if err != nil || len(sits) != 1 {
		t.Fatalf("The Quiet Orchard has %d files once the upgrade imported (%v), want its upgrade alone", len(sits), err)
	}
	for _, s := range sits {
		rel = s.Path.String()
		if b, err = os.ReadFile(filepath.Join(w.Roots["movies-e"].Path, filepath.FromSlash(s.Sits.String()))); err != nil {
			t.Fatal(err)
		}
	}
	return rel, b
}

type workFilesReply struct {
	Files []struct {
		Path string `json:"path"`
	} `json:"files"`
}

func workFiles(t *testing.T, w movestest.World) workFilesReply {
	t.Helper()
	return simtest.Decode[workFilesReply](t, "listWorkFiles", w.API().Admin("home", http.MethodGet, "/api/v1/works/"+w.Work.String()+"/files", ""))
}

// wantedReasons is the reason each Work is wanted, by its label.
func wantedReasons(t *testing.T, w movestest.World) map[string]string {
	t.Helper()
	rows := simtest.Decode[struct {
		Rows []struct {
			Work struct {
				Label string `json:"label"`
			} `json:"work"`
			Reason string `json:"reason"`
		} `json:"rows"`
	}](t, "listWanted", w.API().Admin("home", http.MethodGet, "/api/v1/wanted", "")).Rows
	out := map[string]string{}
	for _, r := range rows {
		out[r.Work.Label] = r.Reason
	}
	return out
}

// expectUpgradeMoved checks batch's unit ends done with the upgrade's
// film, rel with bytes b, in the root in folder, nothing of The Quiet
// Orchard's films left in movies-e, and what is wanted as wanted was.
func expectUpgradeMoved(t *testing.T, w movestest.World, batch moves.BatchID, folder, rel string, b []byte, wanted map[string]string) {
	t.Helper()
	if u := w.Ended(t, batch); u.Phase != moves.PhaseDone {
		t.Fatalf("the unit ended %s %+v, want done", u.Phase, u.Failure)
	}
	placed := filepath.Join(w.Roots[folder].Path, filepath.FromSlash(rel))
	simtest.Eventually(t, "the upgrade's rename runs in "+folder, func() bool {
		_, err := os.Stat(placed)
		return err == nil
	})
	w.Invariants(t, batch)
	if got, err := os.ReadFile(placed); err != nil || !bytes.Equal(got, b) {
		t.Errorf("%s holds %d bytes of the upgrade (%v), want %d", folder, len(got), err, len(b))
	}
	absent(t, w, "movies-e", movestest.Film, rel)
	absent(t, w, folder, movestest.Film)
	work := simtest.Decode[struct {
		Root struct {
			ID string `json:"id"`
		} `json:"root"`
	}](t, "getWork", w.API().Admin("home", http.MethodGet, "/api/v1/works/"+w.Work.String(), ""))
	if work.Root.ID != w.Roots[folder].ID {
		t.Errorf("The Quiet Orchard lives in root %s, want %s", work.Root.ID, folder)
	}
	if listed := workFiles(t, w); len(listed.Files) != 1 || listed.Files[0].Path != rel {
		t.Errorf("The Quiet Orchard's files are %+v, want the upgrade alone", listed.Files)
	}
	if got := wantedReasons(t, w); !maps.Equal(got, wanted) {
		t.Errorf("wanted is %v once the Move ended, want %v as before it", got, wanted)
	}
}

// TestALateImportMidCopyMovesTheUpgrade imports the upgrade of The Quiet
// Orchard's 720p film into movies-e, replacing it, while its Move to
// movies-f holds before its first copy: the Move plans again from the
// source and ends done with the upgrade alone in movies-f, and movies-e
// keeps neither film.
func TestALateImportMidCopyMovesTheUpgrade(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		pause := movestest.PauseAt("place_file")
		w := movestest.LaunchUpgradable(t, pause.Option())
		batch := w.Start(t, "movies-f")
		pause.Reached(t)
		rel, b := upgrade(t, w)
		wanted := wantedReasons(t, w)
		pause.Resume()
		expectUpgradeMoved(t, w, batch, "movies-f", rel, b, wanted)
	})
}

// playOrchard opens a play of The Quiet Orchard's film on Living room
// Plex, a server whose play nothing ends.
func playOrchard(t *testing.T, w movestest.World) {
	t.Helper()
	server, now := storage.NewUUID(), storage.FromTime(time.Now())
	err := w.Node("home").Half().Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO mediaservers_server (id, kind, name, address, identity, version, reaching_node_id, secrets_sealed, created_at, updated_at)
			SELECT ?, 'plex', 'Living room Plex', 'http://192.0.2.10:32400', 'plex-1', '1.41', id, x'00', ?, ? FROM nodes_node WHERE name = 'home'`, server, now, now); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO mediaservers_play (server_id, id, session_key, file_id, server_path, user_name, player, product, decision, started_at, last_seen_at)
			SELECT ?, ?, 'session-1', id, '/media/Movies 4K/' || path, 'ada', 'Living room', 'Plex Web', 'direct_play', ?, ? FROM files_file WHERE path = ?`,
			server, storage.NewUUID(), now, now, movestest.Film)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestAnImportDuringThePlayingWaitGoesBackBeforeSwitchNow imports the
// upgrade of The Quiet Orchard's 720p film, replacing it, while its Move
// to movies-f waits for a play to end, then the Admin presses Switch now:
// the Move plans again from the source and ends done with the upgrade
// alone in movies-f, and movies-e keeps neither film.
func TestAnImportDuringThePlayingWaitGoesBackBeforeSwitchNow(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := movestest.LaunchUpgradable(t)
		playOrchard(t, w)
		batch := w.Start(t, "movies-f")
		var unit string
		waitsPlaying := func() bool {
			b := simtest.Decode[struct {
				Units []struct {
					ID      string `json:"id"`
					Waiting string `json:"waiting"`
				} `json:"units"`
			}](t, "getMove", w.API().Admin("home", http.MethodGet, "/api/v1/moves/"+batch.String(), ""))
			unit = b.Units[0].ID
			return b.Units[0].Waiting == string(moves.WaitPlaying)
		}
		simtest.Eventually(t, "the unit waits for the play to end", waitsPlaying)
		rel, b := upgrade(t, w)
		wanted := wantedReasons(t, w)
		simtest.Eventually(t, "the unit waits for the play to end once the upgrade imported", waitsPlaying)
		if r := w.API().Admin("home", http.MethodPost, "/api/v1/moves/units/"+unit+"/switch-now", ""); r.Status != http.StatusOK {
			t.Fatalf("switchMoveUnitNow = %d %s, want 200", r.Status, r.Body)
		}
		expectUpgradeMoved(t, w, batch, "movies-f", rel, b, wanted)
	})
}

// restore writes what a restore's Write does to the Moves it caught, and
// to Reconciling, then starts home again, as a restore does.
func restore(t *testing.T, w movestest.World) {
	t.Helper()
	err := w.Node("home").Half().Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := reconciling.Begin(ctx, tx, reconciling.BeginArgs{ID: "restore-1", Cause: reconciling.Restore, Backup: "nightly", TakenAt: time.Now()}); err != nil {
			return err
		}
		if err := files.DropQueuedMoves(ctx, tx); err != nil {
			return err
		}
		return moves.Restored(ctx, tx)
	})
	if err != nil {
		t.Fatal(err)
	}
	w.Restart(t, movestest.Fault{Name: "restart"})
}

// expectReconciled checks batch's unit ends failed with want, or done
// with The Quiet Orchard moved to movies-f when want is empty.
func expectReconciled(t *testing.T, w movestest.World, batch moves.BatchID, want moves.FailureCode) {
	t.Helper()
	u := w.Ended(t, batch)
	if want != "" {
		if u.Phase != moves.PhaseFailed || u.Failure == nil || u.Failure.Code != want {
			t.Fatalf("the unit ended %s %+v, want failed %s", u.Phase, u.Failure, want)
		}
		return
	}
	if u.Phase != moves.PhaseDone {
		t.Fatalf("the unit ended %s %+v, want done", u.Phase, u.Failure)
	}
	w.Invariants(t, batch)
	w.Moved(t, "movies-f")
}

// TestARestoreMidMoveReconcilesItsUnit restores home's library as The
// Quiet Orchard's Move to movies-f has placed both files and not
// switched, with the disks as the restored rows find them: a whole source
// moves again, a gone source with a whole destination switches, and with
// neither whole the unit fails source_gone.
func TestARestoreMidMoveReconcilesItsUnit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		gone map[string][]string
		want moves.FailureCode
	}{
		{name: "the source whole"},
		{name: "the destination whole", gone: map[string][]string{"movies-e": {movestest.Film, movestest.Subtitle}}},
		{name: "neither whole", gone: map[string][]string{"movies-e": {movestest.Film}, "movies-f": {movestest.Film}}, want: moves.SourceGone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				crash := movestest.CrashAt("switch")
				w := movestest.Launch(t, crash.Option())
				batch := w.Start(t, "movies-f")
				crash.Reached(t)
				for folder, rels := range tc.gone {
					remove(t, w, folder, rels...)
				}
				restore(t, w)
				expectReconciled(t, w, batch, tc.want)
			})
		})
	}
}

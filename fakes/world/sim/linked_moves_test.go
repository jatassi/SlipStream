package sim_test

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// linked is The Quiet Orchard grabbed into seedbox, its import a hard link
// of the payload in movies-d, and the place its Move to den's movies-e
// puts it.
type linked struct {
	w    *sim.World
	work works.WorkID
	seed seeded
	// film is the file's path in movies-e, from its path in movies-d,
	// and was its bytes there.
	film, from string
	was        []byte
}

// grabLinked grabs The Quiet Orchard on a moves world launched with opts
// and waits until it is imported.
func grabLinked(t *testing.T, opts ...sim.Option) linked {
	t.Helper()
	l := linked{w: launchMoves(t, opts...)}
	work := simtest.AddOrchard(t, l.w)
	simtest.Search(t, l.w, work)
	simtest.ExpectImported(t, simtest.Await(t, l.w))
	id, err := storage.ParseUUID(work)
	if err != nil {
		t.Fatal(err)
	}
	l.work = works.WorkID{UUID: id}
	lib := l.w.Node("home").Half().Library
	var from, to, rel string
	var hash []byte
	if err := lib.QueryRowContext(t.Context(), `SELECT r.path, f.path FROM files_file f JOIN roots_root r ON r.id = f.root_id
		WHERE f.work_id = ?`, l.work).Scan(&from, &rel); err != nil {
		t.Fatal(err)
	}
	if err := lib.QueryRowContext(t.Context(), `SELECT path FROM roots_root WHERE path LIKE '%movies-e'`).Scan(&to); err != nil {
		t.Fatal(err)
	}
	if err := lib.QueryRowContext(t.Context(), `SELECT id, info_hash FROM downloadclients_torrent`).Scan(&l.seed.id, &hash); err != nil {
		t.Fatal(err)
	}
	l.seed.hash = [20]byte(hash)
	l.film, l.from = filepath.Join(to, filepath.FromSlash(rel)), filepath.Join(from, filepath.FromSlash(rel))
	if l.was, err = os.ReadFile(l.from); err != nil {
		t.Fatal(err)
	}
	return l
}

// start previews and starts the Work's Move to movies-e with its torrent,
// failing unless both are movable.
func (l linked) start(t *testing.T) moves.BatchID {
	t.Helper()
	m := l.w.Node("home").Half().Moves
	var to roots.ID
	if err := l.w.Node("home").Half().Library.QueryRowContext(t.Context(), `SELECT id FROM roots_root WHERE path LIKE '%movies-e'`).Scan(&to); err != nil {
		t.Fatal(err)
	}
	plan, err := m.Preview(t.Context(), moves.Ask{
		Origin: moves.OriginWork, Works: []moves.WorkTo{{Work: l.work, To: to}},
		Options: moves.Options{IncludeLinkedTorrents: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Units) != 2 || plan.Movable() != 2 {
		t.Fatalf("the preview is %+v, want the Work and its torrent movable", plan.Units)
	}
	started, err := m.Start(t.Context(), identity.Admin{}, plan.Token, 2)
	if err != nil {
		t.Fatal(err)
	}
	return started.Batch
}

// ended is batch's Work unit and torrent unit once both rest.
func (l linked) ended(t *testing.T, batch moves.BatchID) (work, torrent moves.View) {
	t.Helper()
	within(t, time.Hour, "both units end", func() bool {
		_, units, err := l.w.Node("home").Half().Moves.Get(t.Context(), batch, -1, 10)
		if err != nil || len(units) != 2 {
			t.Fatalf("batch %s has %d units (%v), want 2", batch, len(units), err)
		}
		for _, u := range units {
			if u.Torrent != nil {
				torrent = u
			} else {
				work = u
			}
		}
		return rests(work) && rests(torrent)
	})
	return work, torrent
}

func rests(u moves.View) bool {
	return u.Phase == moves.PhaseDone || u.Phase == moves.PhaseFailed || u.Phase == moves.PhaseCanceled && u.Undone
}

// expectPlaced checks movies-e holds the film whole, and that den's data
// holds want other names of it.
func (l linked) expectPlaced(t *testing.T, want int) {
	t.Helper()
	got, err := os.ReadFile(l.film)
	if err != nil || !bytes.Equal(got, l.was) {
		t.Fatalf("movies-e holds %d bytes of the film (%v), want movies-d's %d", len(got), err, len(l.was))
	}
	placed, err := os.Stat(l.film)
	if err != nil {
		t.Fatal(err)
	}
	if n := sameFiles(t, l.w.Node("den").DataDir().Path(), placed); n != want {
		t.Errorf("den's data holds %d names of movies-e's film, want %d", n, want)
	}
}

// TestAWorkMovesWithItsLinkedTorrent moves The Quiet Orchard to den's
// movies-e with its torrent, then back: both ways both units end done and
// the torrent seeds from its destination alone, from the payload file the
// library's film is a hard link of, and neither Node keeps another copy.
func TestAWorkMovesWithItsLinkedTorrent(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		l := grabLinked(t)
		batch := l.start(t)
		l.expectDone(t, batch)
		l.expectPlaced(t, 1)
		l.expectSeedsFilm(t, "den", l.film)
		m := l.w.Node("home").Half().Moves
		back, err := m.Back(t.Context(), batch)
		if err != nil {
			t.Fatal(err)
		}
		if back.Movable() != 2 {
			t.Fatalf("the way back is %+v, want the Work and its torrent movable", back.Units)
		}
		started, err := m.Start(t.Context(), identity.Admin{}, back.Token, 2)
		if err != nil {
			t.Fatal(err)
		}
		l.expectDone(t, started.Batch)
		if _, err := os.Stat(l.film); !os.IsNotExist(err) {
			t.Errorf("movies-e still holds the film: %v", err)
		}
		l.expectSeedsFilm(t, "seedbox", l.from)
		if left := payloads(t, l.w.Node("den").DataDir().Path()); len(left) > 0 {
			t.Errorf("den keeps %v after the Move back", left)
		}
	})
}

// TestABatchFinishesWhenItsLastCleanupEnds holds The Quiet Orchard's
// Work unit before it finishes its cleanup until its torrent unit is done:
// the batch reads unfinished and offers no way back until the Work's
// cleanup ends.
func TestABatchFinishesWhenItsLastCleanupEnds(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		hold, reached, free := holdWork("finish_cleanup")
		l := grabLinked(t, hold)
		batch := l.start(t)
		<-reached
		defer free()
		m := l.w.Node("home").Half().Moves
		within(t, time.Hour, "the torrent is done", func() bool {
			_, units, err := m.Get(t.Context(), batch, -1, 10)
			if err != nil {
				t.Fatal(err)
			}
			return slices.ContainsFunc(units, func(u moves.View) bool { return u.Torrent != nil && u.Phase == moves.PhaseDone })
		})
		if b, _, err := m.Get(t.Context(), batch, -1, 10); err != nil || b.FinishedAt != nil || b.CanBack {
			t.Errorf("with its Work cleaning up the batch finished at %v and can go back %t (%v), want neither", b.FinishedAt, b.CanBack, err)
		}
		free()
		l.ended(t, batch)
		if b, _, err := m.Get(t.Context(), batch, -1, 10); err != nil || b.FinishedAt == nil || !b.CanBack {
			t.Errorf("once both units are done the batch finished at %v and can go back %t (%v), want both", b.FinishedAt, b.CanBack, err)
		}
	})
}

// expectDone checks batch's Work and torrent both end done, the torrent
// named by its release, and its sentence counts one of each, and the torrent's hash has one Torrent,
// started.
func (l linked) expectDone(t *testing.T, batch moves.BatchID) {
	t.Helper()
	work, torrent := l.ended(t, batch)
	for _, u := range []moves.View{work, torrent} {
		if u.Phase != moves.PhaseDone {
			t.Errorf("%s ended %s %+v, want done", u.Label, u.Phase, u.Failure)
		}
	}
	if torrent.Label != orchard {
		t.Errorf("the torrent is named %q, want its release %q", torrent.Label, orchard)
	}
	if got := wantedOf(t, l.w, l.seed); len(got) != 1 || !strings.HasSuffix(got[0], ":started") {
		t.Errorf("the hash's Torrents are %v, want one, started", got)
	}
	b, _, err := l.w.Node("home").Half().Moves.Get(t.Context(), batch, -1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.Words, "1 movie ") || !strings.Contains(b.Words, "1 torrent ") {
		t.Errorf("the batch reads %q, want it to count 1 movie and 1 torrent", b.Words)
	}
}

// expectSeedsFilm checks the Torrent of the hash on the Node named on
// keeps its payload in a folder whose file is film, and every payload
// file in that Node's data is film too.
func (l linked) expectSeedsFilm(t *testing.T, on, film string) {
	t.Helper()
	of, err := os.Stat(film)
	if err != nil {
		t.Fatal(err)
	}
	var id storage.UUID
	if err := l.w.Node("home").Half().Library.QueryRowContext(t.Context(), `SELECT t.id FROM downloadclients_torrent t
		JOIN nodes_node n ON n.id = t.node_id WHERE t.info_hash = ? AND t.removed_at IS NULL AND n.name = ?`, l.seed.hash[:], on).Scan(&id); err != nil {
		t.Fatal(err)
	}
	var folder string
	within(t, time.Hour, on+" reports where it keeps the torrent", func() bool {
		folder = simtest.Decode[struct {
			SaveFolder string `json:"saveFolder"`
		}](t, "getTorrent", l.w.API().Admin("home", http.MethodGet, "/api/v1/torrents/"+id.String(), "")).SaveFolder
		return folder != ""
	})
	seeds, err := os.Stat(filepath.Join(folder, orchard+".mkv"))
	if err != nil || !os.SameFile(seeds, of) {
		t.Errorf("%s seeds the payload in %s, not the film %s (%v)", on, folder, film, err)
	}
	for _, p := range payloads(t, l.w.Node(on).DataDir().Path()) {
		if info, err := os.Stat(p); err != nil || !os.SameFile(info, of) {
			t.Errorf("%s keeps %s, a copy apart from the film (%v)", on, p, err)
		}
	}
}

// payloads is every video file under dir.
func payloads(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() && strings.HasSuffix(p, ".mkv") {
			out = append(out, p)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestARefusedStartAfterItsWorkSwitchedFailsOnlyTheTorrent has
// fake-tracker refuse den's start after the Work placed from the payload
// and switched: the torrent fails switch_refused and seeds from seedbox
// again, the Work stays moved, and movies-e's film keeps its link to den's
// paused copy.
func TestARefusedStartAfterItsWorkSwitchedFailsOnlyTheTorrent(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		l := grabLinked(t)
		l.w.Cue("tracker", tracker.RejectStart.Name, tracker.RejectStartArgs{Release: orchard, OriginNode: "den", Reason: "You already seed this torrent"})
		work, torrent := l.ended(t, l.start(t))
		if work.Phase != moves.PhaseDone {
			t.Errorf("the Work ended %s %+v, want done", work.Phase, work.Failure)
		}
		if torrent.Phase != moves.PhaseFailed || torrent.Failure.Code != moves.SwitchRefused {
			t.Errorf("the torrent ended %s %+v, want failed switch_refused", torrent.Phase, torrent.Failure)
		}
		if got := wantedOf(t, l.w, l.seed); len(got) != 2 || got[0] != "seedbox:started" || got[1] != "den:paused" {
			t.Errorf("the hash's Torrents are %v, want seedbox's started and den's copy paused", got)
		}
		l.expectPlaced(t, 1)
	})
}

// TestATorrentCanceledAfterItsPlaceLeavesItsWorkToTheLibrary cancels the
// torrent's unit once it placed its payload on den, as its Work takes its
// lane: the Work still moves, its film copied from movies-d, the torrent
// seeds from seedbox alone, and den keeps no payload.
func TestATorrentCanceledAfterItsPlaceLeavesItsWorkToTheLibrary(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		hold, reached, free := holdWorkLane()
		l := grabLinked(t, hold)
		batch := l.start(t)
		<-reached
		defer free()
		l.cancelTorrentAtSwitch(t, batch)
		free()
		work, torrent := l.ended(t, batch)
		if work.Phase != moves.PhaseDone {
			t.Errorf("the Work ended %s %+v, want done", work.Phase, work.Failure)
		}
		if torrent.Phase != moves.PhaseCanceled {
			t.Errorf("the torrent ended %s %+v, want canceled", torrent.Phase, torrent.Failure)
		}
		if got := wantedOf(t, l.w, l.seed); len(got) != 1 || got[0] != "seedbox:started" {
			t.Errorf("the hash's Torrents are %v, want seedbox's alone, started", got)
		}
		l.expectPlaced(t, 0)
	})
}

// holdWorkLane holds home's first Work unit as it takes its lane until
// free runs, closing reached once it holds.
func holdWorkLane() (hold sim.Option, reached <-chan struct{}, free func()) {
	return holdWork("take_lane")
}

// holdWork holds home's first Work unit before its step until free runs,
// closing reached once it holds.
func holdWork(step string) (hold sim.Option, reached <-chan struct{}, free func()) {
	at, release := make(chan struct{}), make(chan struct{})
	var held, released atomic.Bool
	hold = sim.Edit(func(name string, e *node.Env) {
		if name != "home" {
			return
		}
		e.Test.BeforeMoveStep = func(_ context.Context, s moves.Step) {
			if s.Action == step && !s.Work.IsZero() && held.CompareAndSwap(false, true) {
				close(at)
				<-release
			}
		}
	})
	return hold, at, func() {
		if released.CompareAndSwap(false, true) {
			close(release)
		}
	}
}

// cancelTorrentAtSwitch cancels batch's torrent unit, failing unless it
// placed its payload and waits at its switch.
func (l linked) cancelTorrentAtSwitch(t *testing.T, batch moves.BatchID) {
	t.Helper()
	_, units, err := l.w.Node("home").Half().Moves.Get(t.Context(), batch, -1, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range units {
		if u.Torrent == nil {
			continue
		}
		if u.Phase != moves.PhaseSwitch {
			t.Fatalf("the torrent is %s as its Work takes its lane, want placed and at its switch", u.Phase)
		}
		if err := l.w.Node("home").Half().Moves.Cancel(t.Context(), u.Unit); err != nil {
			t.Fatal(err)
		}
	}
}

// sameFiles counts the files under dir that are of.
func sameFiles(t *testing.T, dir string, of os.FileInfo) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", d.Name(), err)
		}
		if os.SameFile(info, of) {
			n++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

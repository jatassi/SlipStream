package sim_test

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
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
	// film is the file's path in movies-e, was its bytes in movies-d.
	film string
	was  []byte
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
	l.film = filepath.Join(to, filepath.FromSlash(rel))
	if l.was, err = os.ReadFile(filepath.Join(from, filepath.FromSlash(rel))); err != nil {
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
	if want := "Move 1 movie and 1 torrent to den."; plan.Recommendation != want {
		t.Errorf("the preview reads %q, want %q", plan.Recommendation, want)
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
// movies-e with its torrent: both units end done, the torrent seeds on den
// alone, and movies-e's file is a hard link of the payload den seeds.
func TestAWorkMovesWithItsLinkedTorrent(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		l := grabLinked(t)
		work, torrent := l.ended(t, l.start(t))
		for _, u := range []moves.View{work, torrent} {
			if u.Phase != moves.PhaseDone {
				t.Errorf("%s ended %s %+v, want done", u.Label, u.Phase, u.Failure)
			}
		}
		if got := wantedOf(t, l.w, l.seed); len(got) != 1 || got[0] != "den:started" {
			t.Errorf("the hash's Torrents are %v, want den's alone, started", got)
		}
		l.expectPlaced(t, 1)
	})
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
		reached, release := make(chan struct{}), make(chan struct{})
		var held atomic.Bool
		l := grabLinked(t, sim.Edit(func(name string, e *node.Env) {
			if name != "home" {
				return
			}
			e.Test.BeforeMoveStep = func(_ context.Context, s moves.Step) {
				if s.Action == "take_lane" && !s.Work.IsZero() && held.CompareAndSwap(false, true) {
					close(reached)
					<-release
				}
			}
		}))
		batch := l.start(t)
		<-reached
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
		close(release)
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

// sameFiles counts the files under dir that are of.
func sameFiles(t *testing.T, dir string, of os.FileInfo) int {
	t.Helper()
	n := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		info, err := d.Info()
		if err == nil && os.SameFile(info, of) {
			n++
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

//go:build scale

package scale_test

import (
	"context"
	"flag"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/movestest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/storage"
)

var moveCount = flag.Int("scale.moves", 5000, "generated movies to seed and move, half on seedbox and half on den")

// TestMovesScale is the scale simulation of S48 (Moves), run on demand and
// never in CI:
//
//	go test -tags scale -run TestMovesScale ./fakes/world/sim/scale/ -v -timeout 2h
//
// It seeds the moves Scenario with 5,000 movies, half in seedbox's
// movies-d and half in den's movies-e, and moves every one to den's
// movies-f in one batch. It reports the seeding, the preview with its
// queries, the Start, the longest time from a unit's switch to its
// cleanup, and the drain to done, in wall and fake time, and asserts
// only that every unit ends done.
func TestMovesScale(t *testing.T) {
	wall := startWallClock(t)
	synctest.Test(t, func(t *testing.T) {
		timer := &switchTimer{wall: wall, at: map[moves.UnitID]time.Time{}}
		var w movestest.World
		seeded := wall.took(func() { w = movestest.Seed(t, *moveCount, timer.option()) })
		t.Logf("seeded %d movies in %s (wall)", len(w.Seeded), seeded.Round(time.Millisecond))

		to := w.Root(t, "movies-f")
		ask := moves.Ask{Origin: moves.OriginWork}
		for _, id := range w.Seeded {
			ask.Works = append(ask.Works, moves.WorkTo{Work: id, To: to})
		}
		ctx, queries := storage.Counted(t.Context())
		var plan moves.Plan
		var err error
		previewed := wall.took(func() { plan, err = w.Moves().Preview(ctx, ask) })
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("preview: %d units, %d movable, in %s (wall) with %d queries: %s",
			len(plan.Units), plan.Movable(), previewed.Round(time.Millisecond), queries(), plan.Recommendation)

		var started moves.Started
		startedIn := wall.took(func() {
			started, err = w.Moves().Start(t.Context(), identity.Admin{}, plan.Token, plan.Movable())
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("start: %d units in one Write in %s (wall)", plan.Movable(), startedIn.Round(time.Millisecond))

		fakeStart, wallStart := time.Now(), wall.now()
		var b moves.BatchView
		for b.FinishedAt == nil {
			time.Sleep(10 * time.Second)
			if b, _, err = w.Moves().Get(t.Context(), started.Batch, -1, 1); err != nil {
				t.Fatal(err)
			}
		}
		t.Logf("drain: %v in %s (fake), %s (wall); the longest switch to cleanup took %s (wall)",
			b.Counts, time.Since(fakeStart).Round(time.Second), wall.now().Sub(wallStart).Round(time.Millisecond), timer.longestSoFar().Round(time.Millisecond))
		if b.Counts[moves.PhaseDone] != plan.Movable() {
			t.Errorf("the batch ended %v, want every unit done", b.Counts)
		}
	})
}

// switchTimer keeps the longest wall time from a unit's switch step to its
// cleanup step on home.
type switchTimer struct {
	wall    wallClock
	mu      sync.Mutex
	at      map[moves.UnitID]time.Time
	longest time.Duration
}

func (s *switchTimer) option() sim.Option {
	return sim.Edit(func(name string, e *node.Env) {
		if name == "home" {
			e.Test.BeforeMoveStep = func(_ context.Context, step moves.Step) { s.saw(step) }
		}
	})
}

func (s *switchTimer) saw(step moves.Step) {
	if step.Action != "switch" && step.Action != "remove_source" {
		return
	}
	now := s.wall.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if step.Action == "switch" {
		s.at[step.Unit] = now
		return
	}
	if from, ok := s.at[step.Unit]; ok {
		s.longest = max(s.longest, now.Sub(from))
		delete(s.at, step.Unit)
	}
}

func (s *switchTimer) longestSoFar() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.longest
}

package sim_test

import (
	"context"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim/movestest"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestARootMoveDeletesTheRootItEmpties moves every Work of movies-e to
// movies-f with "Delete this root when it's empty": once the unit ends,
// movies-e is gone, The Quiet Orchard lives in movies-f, and the batch
// finishes.
func TestARootMoveDeletesTheRootItEmpties(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := movestest.Launch(t)
		m := w.Moves()
		from, to := w.Root(t, "movies-e"), w.Root(t, "movies-f")
		plan, err := m.Preview(t.Context(), moves.Ask{
			Origin: moves.OriginRoot, Roots: []moves.RootTo{{Root: from, To: to}},
			Options: moves.Options{DeleteEmptiedRoots: true},
		})
		if err != nil || plan.Movable() != 1 {
			t.Fatalf("the preview is %+v (%v), want The Quiet Orchard moving", plan.Units, err)
		}
		started, err := m.Start(t.Context(), identity.Admin{}, plan.Token, 1)
		if err != nil {
			t.Fatal(err)
		}
		simtest.Eventually(t, "the batch finishes", func() bool {
			b, _, err := m.Get(t.Context(), started.Batch, -1, 10)
			return err == nil && b.FinishedAt != nil
		})
		if unit := w.Unit(t, started.Batch); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v, want done", unit.Phase, unit.Failure)
		}
		for _, r := range simtest.Decode[struct {
			Roots []simtest.RootRow `json:"roots"`
		}](t, "listRoots", w.API().Admin("home", http.MethodGet, "/api/v1/roots", "")).Roots {
			if r.ID == from.String() {
				t.Errorf("movies-e is still a root after the Move emptied it: %+v", r)
			}
		}
		var root storage.UUID
		err = w.Library(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			return r.QueryRowContext(ctx, `SELECT root_id FROM roots_work WHERE work_id = ?`, w.Work).Scan(&root)
		})
		if err != nil || root != to.UUID {
			t.Errorf("The Quiet Orchard lives in %s (%v), want movies-f", root, err)
		}
		w.Moved(t, "movies-f")
	})
}

// TestARootDeleteWaitsForAMediaServersCarryFromIt moves movies-e to
// movies-f with "Delete this root when it's empty" while a Living room
// Plex Move Carry from movies-e is still open: once the unit ends the
// batch waits and keeps movies-e, whose mapping the Carry still needs, and
// deletes it once the Carry is sent.
func TestARootDeleteWaitsForAMediaServersCarryFromIt(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := movestest.Launch(t)
		m := w.Moves()
		from, to := w.Root(t, "movies-e"), w.Root(t, "movies-f")
		home := w.Node("home").Half()
		change := storage.NewUUID()
		write := func(query string, args ...any) {
			t.Helper()
			err := home.Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				_, err := tx.ExecContext(ctx, query, args...)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		server := storage.NewUUID()
		now, later := storage.FromTime(time.Now()), storage.FromTime(time.Now().Add(1000*time.Hour))
		write(`INSERT INTO mediaservers_server (id, kind, name, address, identity, version, reaching_node_id, secrets_sealed, created_at, updated_at)
			SELECT ?, 'plex', 'Living room Plex', 'http://192.0.2.10:32400', 'plex-1', '1.41', id, x'00', ?, ? FROM nodes_node WHERE name = 'home'`, server, now, now)
		write(`INSERT INTO mediaservers_change (id, server_id, kind, root_id, path, from_root_id, from_path, file_id, state, next_attempt_at, created_at)
			VALUES (?, ?, 'moved', ?, ?, ?, ?, ?, 'due', ?, ?)`, change, server, to, movestest.Film, from, movestest.Film, storage.NewUUID(), later, now)

		plan, err := m.Preview(t.Context(), moves.Ask{
			Origin: moves.OriginRoot, Roots: []moves.RootTo{{Root: from, To: to}},
			Options: moves.Options{DeleteEmptiedRoots: true},
		})
		if err != nil || plan.Movable() != 1 {
			t.Fatalf("the preview is %+v (%v), want The Quiet Orchard moving", plan.Units, err)
		}
		started, err := m.Start(t.Context(), identity.Admin{}, plan.Token, 1)
		if err != nil {
			t.Fatal(err)
		}
		if unit := w.Ended(t, started.Batch); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v, want done", unit.Phase, unit.Failure)
		}
		time.Sleep(10 * time.Minute)
		if b, _, err := m.Get(t.Context(), started.Batch, -1, 10); err != nil || b.FinishedAt != nil {
			t.Fatalf("with the Carry from movies-e open the batch is %+v (%v), want it unfinished", b, err)
		}
		if !hasRoot(t, w, from) {
			t.Fatal("movies-e was deleted while a Carry from it was open")
		}

		write(`UPDATE mediaservers_change SET state = 'sent', sent_at = ?, result = 'accepted' WHERE id = ?`, storage.FromTime(time.Now()), change)
		simtest.Eventually(t, "the batch finishes once the Carry is sent", func() bool {
			b, _, err := m.Get(t.Context(), started.Batch, -1, 10)
			return err == nil && b.FinishedAt != nil
		})
		if hasRoot(t, w, from) {
			t.Error("movies-e is still a root once the Carry from it was sent")
		}
	})
}

func hasRoot(t *testing.T, w movestest.World, id roots.ID) bool {
	t.Helper()
	for _, r := range simtest.Decode[struct {
		Roots []simtest.RootRow `json:"roots"`
	}](t, "listRoots", w.API().Admin("home", http.MethodGet, "/api/v1/roots", "")).Roots {
		if r.ID == id.String() {
			return true
		}
	}
	return false
}

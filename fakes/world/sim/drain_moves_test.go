package sim_test

import (
	"context"
	"net/http"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim/movestest"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/storage"
)

func nodeRow(t *testing.T, a *worldtest.API, name string) worldtest.Row {
	t.Helper()
	for _, r := range a.Nodes("home") {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("listNodes on home has no row %s", name)
	return worldtest.Row{}
}

// TestAStorageDrainMovesWhatTheNodeKeepsAndDeletesTheRootsItEmpties
// unticks den's Storage and moves each of its roots to seedbox with
// "Delete the roots it empties": The Quiet Orchard copies into movies-d,
// Harbor Lights, which has no files, is homed there in the start Write,
// den's roots are deleted once every unit ends, and den stops draining
// Storage.
func TestAStorageDrainMovesWhatTheNodeKeepsAndDeletesTheRootsItEmpties(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := movestest.Launch(t)
		a := w.API()
		harbor := simtest.Decode[struct {
			Work struct {
				ID string `json:"id"`
			} `json:"work"`
		}](t, "addWork", a.Admin("home", http.MethodPost, "/api/v1/works",
			`{"contentType":"movie","key":"9000002","preset":"all","rootId":"`+w.Roots["movies-e"].ID+`"}`)).Work.ID
		den := nodeRow(t, a, "den").ID
		a.SetRoles("home", den, "downloading")
		if st := stateOf(nodeRow(t, a, "den"), "storage"); st != "draining" {
			t.Fatalf("unticked, den stands %s in Storage, want draining", st)
		}

		m := w.Moves()
		plan, err := m.Preview(t.Context(), moves.Ask{
			Origin: moves.OriginDrain,
			Roots: []moves.RootTo{
				{Root: w.Root(t, "movies-e"), To: w.Root(t, "movies-d")},
				{Root: w.Root(t, "movies-f"), To: w.Root(t, "movies-d")},
				{Root: w.Root(t, "series-e"), To: w.Root(t, "series-d")},
			},
			Options: moves.Options{DeleteEmptiedRoots: true},
		})
		if err != nil || plan.Movable() != 1 || len(plan.Homed) != 1 || plan.Homed[0].Work.String() != harbor {
			t.Fatalf("the preview moves %d and homes %+v (%v), want The Quiet Orchard moving and Harbor Lights homed", plan.Movable(), plan.Homed, err)
		}
		started, err := m.Start(t.Context(), identity.Admin{}, plan.Token, 1)
		if err != nil {
			t.Fatal(err)
		}
		if got := rootOf(t, w, harbor); got != w.Roots["movies-d"].ID {
			t.Errorf("after the Start Harbor Lights lives in %s, want movies-d", got)
		}
		simtest.Eventually(t, "the batch finishes", func() bool {
			b, _, err := m.Get(t.Context(), started.Batch, -1, 10)
			return err == nil && b.FinishedAt != nil
		})
		if unit := w.Unit(t, started.Batch); unit.Phase != moves.PhaseDone {
			t.Fatalf("The Quiet Orchard's unit ended %s %+v, want done", unit.Phase, unit.Failure)
		}
		w.Moved(t, "movies-d")
		for _, r := range simtest.Decode[struct {
			Roots []simtest.RootRow `json:"roots"`
		}](t, "listRoots", a.Admin("home", http.MethodGet, "/api/v1/roots", "")).Roots {
			if r.NodeID == den {
				t.Errorf("%s is still a root on den after the drain emptied it", r.Path)
			}
		}
		simtest.Eventually(t, "den stops draining Storage", func() bool {
			return stateOf(nodeRow(t, a, "den"), "storage") == "none"
		})
	})
}

func rootOf(t *testing.T, w movestest.World, work string) string {
	t.Helper()
	id, err := storage.ParseUUID(work)
	if err != nil {
		t.Fatal(err)
	}
	var root storage.UUID
	err = w.Library(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, `SELECT root_id FROM roots_work WHERE work_id = ?`, id).Scan(&root)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root.String()
}

// TestADownloadingDrainKeepsItsRoleUntilItsTorrentsMove deletes seedbox's
// external clients, unticks its Downloading and moves its built-in
// client's torrents to den's:
// seedbox drains Downloading while the unit runs, since the source Torrent
// stays unparted until its switch, and stops once The Quiet Orchard seeds
// on den.
func TestADownloadingDrainKeepsItsRoleUntilItsTorrentsMove(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		s := seedOrchard(t, w, client{"seedbox", ""})
		a := w.API()
		for _, kind := range []string{"qbittorrent", "deluge"} {
			if r := a.Admin("home", http.MethodDelete, "/api/v1/external-download-clients/"+clientOn(t, w, "seedbox", kind).String(), ""); r.Status != http.StatusNoContent {
				t.Fatalf("deleting seedbox's %s = %d %s, want 204", kind, r.Status, r.Body)
			}
		}
		box := nodeRow(t, a, "seedbox").ID
		a.SetRoles("home", box, "searching", "storage")
		if st := stateOf(nodeRow(t, a, "seedbox"), "downloading"); st != "draining" {
			t.Fatalf("unticked, seedbox stands %s in Downloading, want draining", st)
		}
		m := w.Node("home").Half().Moves
		plan, err := m.Preview(t.Context(), moves.Ask{Origin: moves.OriginDrain, Clients: []moves.ClientTo{
			{From: client{"seedbox", ""}.side(t, w), To: client{"den", ""}.side(t, w)},
		}})
		if err != nil || plan.Movable() != 1 || plan.Units[0].Torrent == nil || plan.Units[0].Torrent.Torrent != s.id {
			t.Fatalf("the preview is %+v (%v), want The Quiet Orchard moving", plan.Units, err)
		}
		started, err := m.Start(t.Context(), identity.Admin{}, plan.Token, 1)
		if err != nil {
			t.Fatal(err)
		}
		drainsWhileItsUnitRuns(t, a, m, started.Batch)
		if unit := ended(t, w, started.Batch); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v, want done", unit.Phase, unit.Failure)
		}
		simtest.Eventually(t, "seedbox stops draining Downloading", func() bool {
			return stateOf(nodeRow(t, a, "seedbox"), "downloading") == "none"
		})
	})
}

// drainsWhileItsUnitRuns follows batch's one unit until it ends and fails
// t if seedbox stops draining Downloading before the unit's cleanup.
func drainsWhileItsUnitRuns(t *testing.T, a *worldtest.API, m *moves.Moves, batch moves.BatchID) {
	t.Helper()
	for range 240 {
		_, units, err := m.Get(t.Context(), batch, -1, 10)
		if err != nil || len(units) != 1 {
			t.Fatalf("the batch has %d units (%v), want 1", len(units), err)
		}
		phase := units[0].Phase
		if phase == moves.PhaseDone || phase == moves.PhaseFailed {
			return
		}
		if st := stateOf(nodeRow(t, a, "seedbox"), "downloading"); phase != moves.PhaseCleanup && st != "draining" {
			t.Fatalf("with its torrent's unit %s, seedbox stands %s in Downloading, want draining", phase, st)
		}
		time.Sleep(30 * time.Second)
	}
}

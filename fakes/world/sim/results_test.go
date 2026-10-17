package sim_test

import (
	"context"
	"os"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/role/roletest"
	"github.com/jatassi/slipstream/internal/storage"
)

// partitioned is past keepalive and write_timeout, so the stalled
// session has ended on both sides.
const partitioned = 36 * time.Second

// redialed is past any re-dial a heal waits for.
const redialed = 2 * time.Minute

// nodeID is the id the Controller gave the Node named name.
func nodeID(t *testing.T, w *sim.World, name string) role.NodeID {
	t.Helper()
	id, err := storage.ParseUUID(w.API().Node(name).NodeID)
	if err != nil {
		t.Fatal(err)
	}
	return role.NodeID{UUID: id}
}

// applied is every Probe result home applied from seedbox.
func applied(t *testing.T, w *sim.World) []uint64 {
	t.Helper()
	got, err := roletest.Applied(t.Context(), w.Node("home").Half().Library, nodeID(t, w, "seedbox"))
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func report(t *testing.T, w *sim.World, name string, results ...uint64) {
	t.Helper()
	for _, n := range results {
		if err := w.Node(name).Probe().Report(t.Context(), n); err != nil {
			t.Fatalf("%s reports %d: %v", name, n, err)
		}
	}
}

// TestResultsApplyOnce reports results from seedbox's Probe while it is
// partitioned. home applies each once: after the heal, again after a
// re-dial that follows an Ack cut on its first byte, with the outbox going
// on past the cut, and a new epoch's result after seedbox lost its local.db.
func TestResultsApplyOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithProbe())
		path := w.Path("home", "seedbox")
		synctest.Wait()
		flip(t, path, fault.Stall{})
		time.Sleep(partitioned)
		report(t, w, "seedbox", 1, 2, 3)
		synctest.Wait()
		if got := applied(t, w); len(got) != 0 {
			t.Fatalf("home applied %v while partitioned", got)
		}

		resume := w.Node("home").Control().PauseAt(3)
		path.Clear("")
		time.Sleep(redialed)
		synctest.Wait()
		flip(t, path, fault.Cut(0))
		resume()
		synctest.Wait()
		if got := applied(t, w); !slices.Equal(got, []uint64{1, 2, 3}) {
			t.Fatalf("after the heal, home applied %v, want [1 2 3]", got)
		}
		if got := standing(t, w, "seedbox"); got != role.Unreachable {
			t.Fatalf("with the Ack cut, seedbox's Probe stands %s, want unreachable", got)
		}

		path.Clear("")
		time.Sleep(redialed)
		synctest.Wait()
		if got := applied(t, w); !slices.Equal(got, []uint64{1, 2, 3}) {
			t.Fatalf("after the next re-dial, home applied %v, want [1 2 3]", got)
		}
		report(t, w, "seedbox", 4)
		synctest.Wait()
		if got := applied(t, w); !slices.Equal(got, []uint64{1, 2, 3, 4}) {
			t.Fatalf("after reporting 4, home applied %v, want [1 2 3 4]", got)
		}

		box := w.Node("seedbox")
		box.Stop()
		for _, f := range []string{"", "-wal", "-shm"} {
			if err := os.Remove(box.DataDir().LocalDB() + f); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
		}
		box.Start()
		synctest.Wait()
		report(t, w, "seedbox", 1)
		synctest.Wait()
		if got := applied(t, w); !slices.Equal(got, []uint64{1, 2, 3, 4, 1}) {
			t.Fatalf("with a new local.db, home applied %v, want [1 2 3 4 1]", got)
		}
	})
}

// TestSupervisorRestarts panics seedbox's Probe once: its facts show it
// restarting, then running again restart_min later, and the Node runs on.
func TestSupervisorRestarts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithProbe())
		a := w.API()
		synctest.Wait()
		w.Node("seedbox").Probe().Crash()
		synctest.Wait()
		if f := seedbox(t, a).Facts; f == nil || !slices.Equal(f.Running, []string{"downloading"}) || !slices.Equal(f.Restarting, []string{"probe"}) {
			t.Fatalf("after the panic, seedbox's facts are %+v, want downloading running and probe restarting", f)
		}
		time.Sleep(time.Second)
		synctest.Wait()
		if f := seedbox(t, a).Facts; f == nil || !slices.Equal(f.Running, []string{"downloading", "probe"}) || len(f.Restarting) != 0 {
			t.Fatalf("restart_min later, seedbox's facts are %+v, want downloading and probe running", f)
		}
		if starts := w.Node("seedbox").Probe().Starts(); starts != 2 {
			t.Fatalf("the Probe started %d times, want 2", starts)
		}
		worldtest.Expect(t, "seedbox GET /api/v1/status", a.Get("seedbox", "/api/v1/status"), worldtest.Reply{Status: 200, Body: `{"claim":"worker","passkeys":{"available":false,"reason":"ip_address"}}` + "\n"})
	})
}

// CheckFresh fails Controller logic that changes a Probe table without
// Touch, naming the Node it left behind.
func TestCheckFreshFindsAProbeChangeWithoutTouch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithProbe())
		synctest.Wait()
		home := w.Node("home").Half()
		box := nodeID(t, w, "seedbox")
		names := map[role.NodeID]string{nodeID(t, w, "home"): "home", box: "seedbox"}
		setIntent := func(value string) {
			err := home.Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				return roletest.SetIntent(ctx, tx, box, value)
			})
			if err != nil {
				t.Fatal(err)
			}
		}

		setIntent("touched")
		w.Node("home").Control().Touch(box)
		synctest.Wait()
		touched := &recorder{TB: t}
		roletest.CheckFresh(touched, home.Controls, names)
		if len(touched.errs) != 0 {
			t.Fatalf("after a Touch, CheckFresh reported %q", touched.errs)
		}

		setIntent("forgotten")
		synctest.Wait()
		forgot := &recorder{TB: t}
		roletest.CheckFresh(forgot, home.Controls, names)
		if want := []string{"probe intent for seedbox changed without Touch"}; !slices.Equal(forgot.errs, want) {
			t.Fatalf("without Touch, CheckFresh reported %q, want %q", forgot.errs, want)
		}
	})
}

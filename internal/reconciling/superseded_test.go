package reconciling_test

import (
	"context"
	"testing"
	"testing/synctest"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// TestSupersededKeepsAllAndPausesOnlyItself retires the Controller: every
// gate reads Superseded, its own Downloading intent keeps and pauses every
// torrent, and a Worker's keeps every torrent and pauses none.
func TestSupersededKeepsAllAndPausesOnlyItself(t *testing.T) {
	synctest.Test(t, func(t *testing.T) { supersededKeepsAll(t, newWorld(t)) })
}

func supersededKeepsAll(t *testing.T, w *world) {
	worker := role.NodeID{UUID: storage.NewUUID()}
	if err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO nodes_fence (id, state, addresses, fenced_at) VALUES (1, 'retired', '[]', 0)")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	derive := dc.Derive{Keeps: w.keeper, HoldingsPage: 1000}
	if err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		if p, err := reconciling.State(ctx, r); err != nil || p != reconciling.Superseded {
			t.Fatalf("State = %v, %v; want Superseded", p, err)
		}
		own, err := derive.Intent(ctx, r, controllerNode)
		if err != nil {
			return err
		}
		other, err := derive.Intent(ctx, r, worker)
		if err != nil {
			return err
		}
		if !own.GetKeepAll() || !own.GetPauseAll() || own.GetHoldings() != nil {
			t.Errorf("the Controller's intent keeps all %v, pauses all %v, asks Holdings %v; want keep and pause, no Holdings", own.GetKeepAll(), own.GetPauseAll(), own.GetHoldings())
		}
		if !other.GetKeepAll() || other.GetPauseAll() {
			t.Errorf("a Worker's intent keeps all %v, pauses all %v; want keep, no pause", other.GetKeepAll(), other.GetPauseAll())
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

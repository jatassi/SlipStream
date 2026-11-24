package reachability_test

import (
	"context"
	"math/rand/v2"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/reachability/judge/judgetest"
	"github.com/jatassi/slipstream/internal/reachability/reachabilitytest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

func world(t *testing.T, b *judgetest.B) *reachabilitytest.World {
	t.Helper()
	return reachabilitytest.New(t, b, reachabilitytest.Config{
		Timings: reachability.DefaultTimings(),
		Rand:    rand.New(rand.NewPCG(1, 2)), //nolint:gosec // a fixed draw the test replays
	})
}

func exec(t *testing.T, w *reachabilitytest.World, sql string, args ...any) {
	t.Helper()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, sql, args...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// count is how many rows from, a table and its WHERE, holds.
func count(t *testing.T, w *reachabilitytest.World, from string, args ...any) int {
	t.Helper()
	var n int
	if err := w.Lib.QueryRowContext(t.Context(), "SELECT count(*) FROM "+from, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func address(t *testing.T, w *reachabilitytest.World, addr string) {
	t.Helper()
	exec(t, w, "UPDATE identity_settings SET address = ? WHERE id = 1", addr)
	identity.AddressChangedEvent.Bind(w.Hub).Publish(t.Context(), identity.AddressChanged{})
}

// plex adds a Plex reached through atlas that runs on runsOn.
func plex(t *testing.T, w *reachabilitytest.World, name string, runsOn role.NodeID) link.ServerID {
	t.Helper()
	id := link.ServerID{UUID: storage.NewUUID()}
	exec(t, w, `INSERT INTO mediaservers_server (id, kind, name, address, identity, version, reaching_node_id, runs_on_node_id, secrets_sealed, created_at, updated_at)
		VALUES (?, 'plex', ?, 'http://192.168.1.20:32400', ?, '1.43.5', ?, ?, x'00', 1, 1)`, id, name, reachabilitytest.Identity, judgetest.Atlas, runsOn)
	mediaservers.ChangedEvent.Bind(w.Hub).Publish(t.Context(), mediaservers.Changed{})
	return id
}

func removeServer(t *testing.T, w *reachabilitytest.World, id link.ServerID) {
	t.Helper()
	exec(t, w, "DELETE FROM mediaservers_server WHERE id = ?", id)
	mediaservers.ChangedEvent.Bind(w.Hub).Publish(t.Context(), mediaservers.Changed{})
}

func removeNode(t *testing.T, w *reachabilitytest.World, id role.NodeID) {
	t.Helper()
	exec(t, w, "UPDATE nodes_node SET removed_at = 1, removed_key = public_key, public_key = NULL WHERE id = ?", id)
	w.Removed(t, id)
}

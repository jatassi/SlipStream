package indexers_test

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestUnassigningSearchingMovesEveryIndexerHomedOnTheNode(t *testing.T) {
	lib := storagetest.Library(t, indexers.Schema())
	seedbox, home := role.NodeID{UUID: storage.NewUUID()}, role.NodeID{UUID: storage.NewUUID()}
	owner := indexers.SearchingOwner
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for i, name := range []string{"one", "two"} {
			_, err := tx.ExecContext(ctx, "INSERT INTO indexers_indexer (id, name, protocol, url, home_node_id, caps, created_at) VALUES (?, ?, 'torznab', 'http://x', ?, '{}', ?)",
				storage.NewUUID(), name, seedbox, i)
			if err != nil {
				return err
			}
		}
		return owner.Unassigned(ctx, tx, seedbox, home)
	})
	if err != nil {
		t.Fatal(err)
	}
	for node, want := range map[role.NodeID]int64{seedbox: 0, home: 2} {
		held, err := owner.Held(t.Context(), lib, node)
		if err != nil {
			t.Fatal(err)
		}
		if held.Units != want {
			t.Errorf("node %s has %d Indexers homed on it, want %d", node, held.Units, want)
		}
	}
}

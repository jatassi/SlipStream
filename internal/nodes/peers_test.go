package nodes_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestPeersListDialAddressesBeforeLinkAddresses(t *testing.T) {
	lib := storagetest.Library(t, nodes.Schema())
	box, gone := role.NodeID{UUID: storage.NewUUID()}, role.NodeID{UUID: storage.NewUUID()}
	var key link.Key
	key[0] = 7
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, stmt := range []struct {
			sql  string
			args []any
		}{
			{"INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, ?, 'box', 0)", []any{box, key[:]}},
			{"INSERT INTO nodes_node (id, name, paired_at, removed_at, removed_key) VALUES (?, 'old', 0, 1, randomblob(32))", []any{gone}},
			{"INSERT INTO nodes_address (node_id, position, address) VALUES (?, 0, '10.0.0.2:7000')", []any{box}},
			{"INSERT INTO nodes_dial (node_id, position, address) VALUES (?, 0, 'box.example:7000')", []any{box}},
		} {
			if _, err := tx.ExecContext(ctx, stmt.sql, stmt.args...); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var peers map[role.NodeID]nodes.Peer
	err = lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		peers, err = nodes.PeersIn(ctx, rd)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	got := peers[box]
	if got.Name != "box" || got.Key != key || !slices.Equal(got.Addresses, []string{"box.example:7000", "10.0.0.2:7000"}) {
		t.Errorf("box reads %+v, want its key with its dial address first", got)
	}
	if old := peers[gone]; old.Name != "old" || !old.Removed || old.Key != (link.Key{}) {
		t.Errorf("the removed Node reads %+v, want its name and no key", old)
	}
}

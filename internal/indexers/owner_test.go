package indexers_test

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestUnassigningSearchingMovesEveryIndexerHomedOnTheNode(t *testing.T) {
	sealer := secrettest.Sealer(t)
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), indexers.Schema())
	seedbox, home := role.NodeID{UUID: storage.NewUUID()}, role.NodeID{UUID: storage.NewUUID()}
	owner := indexers.SearchingOwner
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for node, name := range map[role.NodeID]string{seedbox: "seedbox", home: "home"} {
			if err := nodestest.Insert(ctx, tx, node, name); err != nil {
				return err
			}
		}
		for _, name := range []string{"one", "two"} {
			id := indexers.ID{UUID: storage.NewUUID()}
			if err := indexerstest.Torznab(ctx, tx, sealer, id, name, seedbox, "https://site.test/torznab", "k"); err != nil {
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

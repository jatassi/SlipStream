// Package nodestest gives tests that open a roster what nodes.Open asks
// for, and a paired Node row.
package nodestest

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Unowned is an owner for every Role, each holding nothing.
func Unowned() []nodes.Owner {
	out := make([]nodes.Owner, 0, len(role.Names()))
	for _, r := range role.Names() {
		out = append(out, nodes.Unowned(r))
	}
	return out
}

// Insert pairs a Node named name in tx, for tables that reference one.
func Insert(ctx context.Context, tx *storage.Tx[storage.Library], id role.NodeID, name string) error {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	if err := db.New(tx).InsertNode(ctx, db.InsertNodeParams{ID: id, PublicKey: key, Name: name, PairedAt: storage.Now()}); err != nil {
		return fmt.Errorf("pair Node %s: %w", name, err)
	}
	return nil
}

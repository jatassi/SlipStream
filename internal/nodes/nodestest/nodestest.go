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

// Address records addr, such as 10.0.0.7:7878, as one of node's link
// addresses.
func Address(ctx context.Context, tx *storage.Tx[storage.Library], id role.NodeID, position int64, addr string) error {
	if err := db.New(tx).InsertAddress(ctx, db.InsertAddressParams{NodeID: id, Position: position, Address: addr}); err != nil {
		return fmt.Errorf("record an address of Node %s: %w", id, err)
	}
	return nil
}

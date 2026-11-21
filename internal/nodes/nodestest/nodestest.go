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

// Paired is nodes.Config.Paired for a roster whose pairings no other
// owner keeps a row for.
func Paired(context.Context, *storage.Tx[storage.Library], role.NodeID) error { return nil }

// Supersede records key as an old Controller key that the Controller row
// controller replaced, with proof.
func Supersede(ctx context.Context, tx *storage.Tx[storage.Library], controller role.NodeID, key, proof []byte) error {
	if err := db.New(tx).InsertSuperseded(ctx, db.InsertSupersededParams{PublicKey: key, NodeID: controller, Proof: proof, SupersededAt: storage.Now()}); err != nil {
		return fmt.Errorf("supersede a Controller key: %w", err)
	}
	return nil
}

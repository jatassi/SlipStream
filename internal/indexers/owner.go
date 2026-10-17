package indexers

import (
	"context"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// SearchingOwner counts the Indexers homed on a Node, the work that moves
// to another Searching Node when it lets Searching go.
var SearchingOwner = nodes.Owner{Role: role.Searching, Held: homedOn, Unassigned: rehome}

func homedOn(ctx context.Context, r storage.Reader[file], node role.NodeID) (nodes.Held, error) {
	n, err := db.New(r).CountHomedOn(ctx, node)
	return nodes.Held{Units: n, Noun: nodes.Noun{One: "Indexer", Many: "Indexers"}}, err
}

func rehome(ctx context.Context, tx *storage.Tx[file], node, heir role.NodeID) error {
	return db.New(tx).Rehome(ctx, db.RehomeParams{Heir: heir, HomeNodeID: node})
}

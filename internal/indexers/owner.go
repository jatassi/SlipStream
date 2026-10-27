package indexers

import (
	"context"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// SearchingOwner counts the Indexers homed on a Node, the work that moves
// to another Searching Node when it lets Searching go or is written off.
var SearchingOwner = nodes.Owner{Role: role.Searching, Held: homedOn, Unassigned: rehome}

func homedOn(ctx context.Context, r storage.Reader[file], node role.NodeID) (nodes.Held, error) {
	n, err := db.New(r).CountHomedOn(ctx, node)
	return nodes.Held{Units: n, Noun: nodes.Noun{One: "Indexer", Many: "Indexers"}}, err
}

// rehome moves every Indexer homed on node to heir, the default home, and
// deletes node's sessions, since a session holds only on the Node that
// earned it.
func rehome(ctx context.Context, tx *storage.Tx[file], node, heir role.NodeID) error {
	q := db.New(tx)
	if err := q.Rehome(ctx, db.RehomeParams{Heir: heir, HomeNodeID: node}); err != nil {
		return err
	}
	return q.DeleteSessionsOn(ctx, node)
}

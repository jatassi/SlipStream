package conversions

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
)

var Owner = nodes.Owner{Role: role.Converting, Held: held, Unassigned: unassigned, Requeues: true}

var conversionNoun = nodes.Noun{One: "Conversion", Many: "Conversions"}

// held counts every Conversion placed on node as Units, and the running
// and paused ones as Started.
func held(ctx context.Context, r storage.Reader[storage.Library], node nodes.ID) (nodes.Held, error) {
	n, err := db.New(r).CountHeld(ctx, node)
	if err != nil {
		return nodes.Held{}, err
	}
	return nodes.Held{Units: n.Units, Started: n.Started, Noun: conversionNoun}, nil
}

// unassigned puts every Conversion placed on node back in the queue, heir
// or not. Owner is built before Conversions, so the source copies it
// leaves are ended by the reconciler's next pass.
func unassigned(ctx context.Context, tx *storage.Tx[storage.Library], node, _ nodes.ID) error {
	return (&Conversions{cfg: Config{Clock: time.Now, Transfers: unreached{}}}).putBack(ctx, tx, node)
}

// unreached is the Transfers of a put back that can't reach them.
type unreached struct{}

func (unreached) Want(context.Context, *storage.Tx[storage.Library], transfers.Request) (transfers.ID, error) {
	return transfers.ID{}, nil
}

func (unreached) Touch() {}

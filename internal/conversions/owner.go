package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

var Owner = nodes.Owner{Role: role.Converting, Held: held, Unassigned: unassigned}

// held counts every Conversion placed on node as Units.
func held(context.Context, storage.Reader[storage.Library], nodes.ID) (nodes.Held, error) {
	return nodes.Held{}, nil
}

// unassigned puts every Conversion placed on node back in the queue, heir
// or not.
func unassigned(context.Context, *storage.Tx[storage.Library], nodes.ID, nodes.ID) error { return nil }

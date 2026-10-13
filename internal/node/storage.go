package node

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

func librarySchemas() []storage.Schema[storage.Library] {
	return []storage.Schema[storage.Library]{
		secret.Schema(),
		identity.Schema(),
		role.LibrarySchema(),
	}
}

func localSchemas() []storage.Schema[storage.Local] {
	return []storage.Schema[storage.Local]{
		role.LocalSchema(),
	}
}

func preMigrationHook(context.Context, *storage.Unmigrated, storage.Plan) error {
	return nil
}

func (n *Node) stoppedOr(err error) error {
	if stop := n.checkpoint(); stop != nil {
		return stop
	}
	return err
}

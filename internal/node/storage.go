package node

import (
	"context"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func LibrarySchemas(types contenttype.Set) []storage.Schema[storage.Library] {
	core := []storage.Schema[storage.Library]{
		secret.Schema(),
		identity.Schema(),
		nodes.Schema(),
		role.LibrarySchema(),
		works.Schema(),
		targets.Schema(),
		roots.Schema(),
		indexers.Schema(),
		files.Schema(),
	}
	return append(core, types.Schemas()...)
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

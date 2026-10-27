package node

import (
	"context"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

func LibrarySchemas(types contenttype.Set) []storage.Schema[storage.Library] {
	core := []storage.Schema[storage.Library]{
		secret.Schema(),
		identity.Schema(),
		mail.Schema(),
		nodes.Schema(),
		role.LibrarySchema(),
		works.Schema(),
		metadata.Schema(),
		targets.Schema(),
		roots.Schema(),
		downloadclients.Schema(),
		grabs.Schema(),
		indexers.Schema(),
		files.Schema(),
		tasks.Schema(),
		health.Schema(),
		firstrun.Schema(),
	}
	return append(core, types.Schemas()...)
}

func localSchemas() []storage.Schema[storage.Local] {
	return []storage.Schema[storage.Local]{
		role.LocalSchema(),
		transfers.Schema(),
		downloading.Schema(),
		metadata.LocalSchema(),
		artwork.Schema(),
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

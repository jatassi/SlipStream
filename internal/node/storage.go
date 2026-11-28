package node

import (
	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/home"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/netfacts"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/trackers"
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
		transfers.LibrarySchema(),
		grabs.Schema(),
		indexers.Schema(),
		trackers.Schema(),
		files.Schema(),
		conversions.Schema(),
		subtitles.Schema(),
		mediaservers.Schema(),
		reachability.Schema(),
		naming.Schema(),
		imports.Schema(),
		moves.Schema(),
		tasks.Schema(),
		health.Schema(),
		history.Schema(),
		home.Schema(),
		notifications.Schema(),
		firstrun.Schema(),
		backups.Schema(),
		reconciling.Schema(),
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
		storing.SidecarSchema(),
		netfacts.Schema(),
	}
}

func (n *Node) stoppedOr(err error) error {
	if stop := n.checkpoint(); stop != nil {
		return stop
	}
	return err
}

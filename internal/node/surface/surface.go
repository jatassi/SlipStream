// Package surface is the whole API: every owner's operations and every
// declared event. It never imports internal/node, which links libtorrent,
// so the document builds without cgo.
package surface

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/downloadclientsapi"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/filesapi"
	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/firstrunapi"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabsapi"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/healthapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identityapi"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexersapi"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/liveapi"
	"github.com/jatassi/slipstream/internal/mailapi"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadataapi"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesapi"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/pairingapi"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rootsapi"
	"github.com/jatassi/slipstream/internal/servingapi"
	"github.com/jatassi/slipstream/internal/shellapi"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/internal/systemapi"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/targetsapi"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/tasksapi"
	"github.com/jatassi/slipstream/internal/works"
	"github.com/jatassi/slipstream/internal/worksapi"
)

// Events lists every declared event, one line each. The hub, the document
// and every Bind read it.
var Events = []live.Event{
	identity.SessionsChangedEvent,
	identity.AccountsChangedEvent,
	identity.LinksChangedEvent,
	identity.PasskeysChangedEvent,
	identity.AddressChangedEvent,
	nodes.ChangedEvent,
	nodes.SettingsChangedEvent,
	nodes.TrafficEvent,
	works.ChangedEvent,
	works.ContentTypesChangedEvent,
	works.TypeChangedEvent,
	metadata.ChangedEvent,
	targets.ChangedEvent,
	roots.ChangedEvent,
	roots.VolumesChangedEvent,
	indexers.ChangedEvent,
	indexers.StatusChangedEvent,
	files.ChangedEvent,
	grabs.ChangedEvent,
	grabs.SearchChangedEvent,
	downloadclients.ChangedEvent,
	downloadclients.LiveEvent,
	downloadclients.ClientsChangedEvent,
	tasks.ChangedEvent,
	tasks.ProgressEvent,
	health.ChangedEvent,
	firstrun.ChangedEvent,
}

// Node is the running Node as the operations reach it: its claim, the
// composition root's one definition of it, its library.db, its roster,
// its Join and its Work tree.
type Node interface {
	Claim(context.Context) (system.Claim, error)
	identityapi.Library
	mailapi.Library
	nodesapi.Roster
	pairingapi.Joiner
	worksapi.Shelf
	metadataapi.Registry
	targetsapi.Goals
	rootsapi.Folders
	indexersapi.Registry
	filesapi.Library
	grabsapi.Pipeline
	downloadclientsapi.Clients
	shellapi.Shell
	systemapi.Systems
	Removal() (pairing.Removal, bool)
	tasksapi.Clock
	servingapi.Worker
	healthapi.Watch
	firstrunapi.Guide
}

// Services are what the operations run against. Document passes none.
type Services struct {
	Node   Node
	Hub    *live.Hub
	Live   live.Timings
	System system.Facts
}

// Mount registers every owner's operations, one line each.
func Mount(r *api.Router, s Services) {
	identityapi.Routes(r, s.Node)
	mailapi.Routes(r, s.Node)
	liveapi.Routes(r, s.Hub, s.Node, s.Live)
	systemapi.Routes(r, s.System, s.Node)
	nodesapi.Routes(r, s.Node)
	pairingapi.Routes(r, s.Node)
	worksapi.Routes(r, s.Node)
	metadataapi.Routes(r, s.Node)
	targetsapi.Routes(r, s.Node)
	rootsapi.Routes(r, s.Node)
	indexersapi.Routes(r, s.Node)
	filesapi.Routes(r, s.Node)
	grabsapi.Routes(r, s.Node)
	downloadclientsapi.Routes(r, s.Node)
	shellapi.Routes(r, s.Node)
	tasksapi.Routes(r, s.Node)
	servingapi.Routes(r, s.Node)
	healthapi.Routes(r, s.Node)
	firstrunapi.Routes(r, s.Node)
}

// Document is the OpenAPI document. Registering only captures each
// handler, so none runs and no service is needed.
func Document() []byte {
	r := api.NewRouter(api.Services{})
	Mount(r, Services{Hub: live.NewHub(Events)})
	return r.Document()
}

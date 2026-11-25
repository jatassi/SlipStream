// Package surface is the whole API: every owner's operations and every
// declared event. It never imports internal/node, which links libtorrent,
// so the document builds without cgo.
package surface

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/backupsapi"
	"github.com/jatassi/slipstream/internal/calendarapi"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/downloadclientsapi"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/filesapi"
	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/firstrunapi"
	"github.com/jatassi/slipstream/internal/formatsapi"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabsapi"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/healthapi"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/historyapi"
	"github.com/jatassi/slipstream/internal/home"
	"github.com/jatassi/slipstream/internal/homeapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identityapi"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/importsapi"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexersapi"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/liveapi"
	"github.com/jatassi/slipstream/internal/mailapi"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaserversapi"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadataapi"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/movesapi"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/namingapi"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesapi"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notificationsapi"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/pairingapi"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/reachabilityapi"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/reconcilingapi"
	"github.com/jatassi/slipstream/internal/restoreapi"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rootsapi"
	"github.com/jatassi/slipstream/internal/servingapi"
	"github.com/jatassi/slipstream/internal/shellapi"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitlesapi"
	"github.com/jatassi/slipstream/internal/supersededapi"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/internal/systemapi"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/targetsapi"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/tasksapi"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/trackersapi"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfersapi"
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
	nodes.FencedEvent,
	nodes.TrafficEvent,
	works.ChangedEvent,
	works.RemovedEvent,
	works.ContentTypesChangedEvent,
	works.TypeChangedEvent,
	metadata.ChangedEvent,
	targets.ChangedEvent,
	roots.ChangedEvent,
	roots.VolumesChangedEvent,
	indexers.ChangedEvent,
	indexers.StatusChangedEvent,
	indexers.CatalogChangedEvent,
	indexers.SettingsChangedEvent,
	indexers.RSSChangedEvent,
	indexers.TrackersEvent,
	trackers.SeedingChangedEvent,
	files.ChangedEvent,
	files.MovedEvent,
	naming.ChangedEvent,
	imports.ChangedEvent,
	moves.ChangedEvent,
	grabs.ChangedEvent,
	grabs.SearchChangedEvent,
	grabs.BlocklistChangedEvent,
	grabs.QueueChangedEvent,
	downloadclients.ChangedEvent,
	downloadclients.LiveEvent,
	downloadclients.ClientsChangedEvent,
	transfers.ChangedEvent,
	tasks.ChangedEvent,
	tasks.ProgressEvent,
	health.ChangedEvent,
	firstrun.ChangedEvent,
	mediaservers.ChangedEvent,
	mediaservers.SessionsChangedEvent,
	mediaservers.UnhealthyEvent,
	mediaservers.RecoveredEvent,
	mediaservers.ItemsFoundEvent,
	reachability.ChangedEvent,
	notifications.ChangedEvent,
	notifications.SeenEvent,
	history.ChangedEvent,
	home.SettingsChangedEvent,
	subtitles.ChangedEvent,
	subtitles.ProvidersChangedEvent,
	subtitles.AttemptEvent,
	reconciling.ChangedEvent,
	backups.ChangedEvent,
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
	trackersapi.Registry
	filesapi.Library
	namingapi.Namer
	importsapi.Pending
	grabsapi.Pipeline
	downloadclientsapi.Clients
	transfersapi.Copies
	movesapi.Mover
	backupsapi.Keeper
	shellapi.Shell
	systemapi.Systems
	Removal() (pairing.Removal, bool)
	Superseded() bool
	supersededapi.Fencer
	tasksapi.Clock
	servingapi.Worker
	healthapi.Watch
	firstrunapi.Guide
	mediaserversapi.Servers
	reachabilityapi.Owner
	notificationsapi.Notes
	calendarapi.Calendar
	historyapi.History
	homeapi.Home
	subtitlesapi.Owner
	reconcilingapi.Reconciler
	restoreapi.Restorer
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
	formatsapi.Routes(r, s.Node)
	rootsapi.Routes(r, s.Node)
	indexersapi.Routes(r, s.Node)
	trackersapi.Routes(r, s.Node)
	filesapi.Routes(r, s.Node)
	namingapi.Routes(r, s.Node)
	importsapi.Routes(r, s.Node)
	grabsapi.Routes(r, s.Node)
	downloadclientsapi.Routes(r, s.Node)
	transfersapi.Routes(r, s.Node)
	movesapi.Routes(r, s.Node)
	backupsapi.Routes(r, s.Node)
	shellapi.Routes(r, s.Node)
	tasksapi.Routes(r, s.Node)
	servingapi.Routes(r, s.Node)
	healthapi.Routes(r, s.Node)
	firstrunapi.Routes(r, s.Node)
	mediaserversapi.Routes(r, s.Node)
	reachabilityapi.Routes(r, s.Node)
	notificationsapi.Routes(r, s.Node)
	calendarapi.Routes(r, s.Node)
	historyapi.Routes(r, s.Node)
	homeapi.Routes(r, s.Node)
	subtitlesapi.Routes(r, s.Node)
	reconcilingapi.Routes(r, s.Node)
	restoreapi.Routes(r, s.Node)
	supersededapi.Routes(r, s.Node)
}

// Document is the OpenAPI document. Registering only captures each
// handler, so none runs and no service is needed.
func Document() []byte {
	r := api.NewRouter(api.Services{})
	Mount(r, Services{Hub: live.NewHub(Events)})
	return r.Document()
}

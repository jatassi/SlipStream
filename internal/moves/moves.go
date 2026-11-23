// Package moves owns S48 (Moves): an Admin's batch of units, each moving
// one Work's live files to another library root. A preview gives every
// unit a verdict before anything moves. One runner drives every unit
// through its Stage: it copies while the source keeps playing, switches
// the Work's rows in one Write, and deletes the source only after that
// Write commits. Owners reach moves only through the hooks they declare,
// which Hooks answers.
package moves

import (
	"context"
	"embed"
	"log/slog"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("moves", schemaFiles) }

// Changed names what one Write of moves changed.
type Changed struct {
	BatchIDs []string `json:"batchIds" nullable:"false" format:"uuid"`
	UnitIDs  []string `json:"unitIds" nullable:"false" format:"uuid"`
	WorkIDs  []string `json:"workIds" nullable:"false" format:"uuid"`
}

var ChangedEvent = live.AdminEvent[Changed]("moves.changed")

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

// Disk is the byte calls a unit makes on the Nodes, each on the Node that
// holds the root now: *roots.Roots.
type Disk interface {
	ProbeMove(ctx context.Context, node role.NodeID, p storing.ProbeMove) (storing.Probed, error)
	Place(ctx context.Context, root roots.ID, to storing.Rel, from storing.Source, allowCopy bool) (storing.Placed, error)
	PlaceSidecar(ctx context.Context, root roots.ID, to storing.Rel, from storing.Source, allowCopy bool) (storing.Placed, error)
	Move(ctx context.Context, root roots.ID, from, to storing.Rel, carry bool) (storing.Placed, error)
	MoveSidecar(ctx context.Context, root roots.ID, from, to storing.Rel) (storing.Placed, error)
	RemoveIf(ctx context.Context, root roots.ID, rel storing.Rel, size int64, modified time.Time) (storing.Removal, error)
	Prune(ctx context.Context, root roots.ID, folder storing.Rel, keep []storing.Rel) ([]storing.Rel, error)
}

// MediaServers is what a unit reads and asks of the Media Servers:
// *mediaservers.MediaServers.
type MediaServers interface {
	Coverage(ctx context.Context, r storage.Reader[file], rs []roots.ID) ([]mediaservers.Cover, error)
	Playing(ctx context.Context, r storage.Reader[file], fs []files.ID) (map[files.ID][]link.ServerID, error)
	PrepareMove(ctx context.Context, fs []files.ID, from, to roots.ID)
	Snapshotted(ctx context.Context, r storage.Reader[file], fs []files.ID, from roots.ID) (map[link.ServerID]bool, error)
	Safe(ctx context.Context, r storage.Reader[file], from roots.ID, fs []files.ID) (map[link.ServerID]bool, error)
	SourcesGone(ctx context.Context, tx *storage.Tx[file], from roots.ID, fs []files.ID) error
}

// Roster is whether a Node is connected and in step now.
type Roster interface {
	InStep(node role.NodeID) bool
}

type Config struct {
	Library *storage.DB[file]
	Works   *works.Works
	Files   *files.Files
	Roots   *roots.Roots
	Disk    Disk
	// Destinations is each of ids with its State and volume, read in rd.
	Destinations func(ctx context.Context, rd storage.Reader[file], ids []roots.ID) (map[roots.ID]roots.Destination, error)
	Imports      *imports.Imports
	Transfers    *transfers.Transfers
	MediaServers MediaServers
	// Writing is which of fs a Speech Map is built for or a Sidecar is
	// written beside now: subtitles.Writing.
	Writing func(ctx context.Context, r storage.Reader[file], fs []files.ID) (map[files.ID]bool, error)
	Roster  Roster
	// Videos is the Admin's video extensions, which say a file's
	// companions.
	Videos func(ctx context.Context, r storage.Reader[file]) (media.Videos, error)
	// Reconciling is whether a restored Controller is Reconciling: no
	// step runs, and Start refuses.
	Reconciling  func(ctx context.Context, r storage.Reader[file]) (bool, error)
	Hub          *live.Hub
	Changed      Publisher[Changed]
	FilesChanged Publisher[files.Changed]
	// Touch has the Nodes' Storage intents derived again.
	Touch func(nodes ...role.NodeID)
	// Clients holds the Torrents a torrent unit moves.
	Clients *downloadclients.Clients
	// Downloading is a Node's downloading service, which places a
	// torrent's payload in its destination client's folder.
	Downloading func(node role.NodeID) downloadingv1connect.DownloadingServiceHandler
	// Repoint and CopyGoalRun are a torrent switch's Grabs and seeding
	// goal: grabs.RepointTorrentIn and trackers.CopyGoalRunIn.
	Repoint     func(ctx context.Context, tx *storage.Tx[file], from, to downloadclients.ID, node role.NodeID) error
	CopyGoalRun func(ctx context.Context, tx *storage.Tx[file], from, to downloadclients.ID) error
	// Reroute has each Tracker route to one client go to another, in a
	// Start that repoints routing: trackers.RerouteIn.
	Reroute  func(ctx context.Context, tx *storage.Tx[file], from, to downloadclients.Choice) error
	Recorder entry.Recorder
	Timings  Timings
	// BeforeStep sees each IO step before it runs, each torrent handoff
	// step inside its Write, and a torrent's record of its place, in
	// tests.
	BeforeStep func(ctx context.Context, s Step)
	Clock      func() time.Time
	Log        *slog.Logger
}

// Step is one IO step of a unit, as a test sees it before it runs.
type Step struct {
	Unit   UnitID
	Work   works.WorkID
	Action string
}

type Moves struct {
	cfg   Config
	plans *plans
	poke  chan struct{}

	mu sync.Mutex
	// running is each unit an IO step runs for now.
	running map[UnitID]bool
	// fired is each wait an event fired since the last pass, with the ids
	// its events named: nil ids for every unit with the wait.
	fired map[Wait]*scope
	// reconciled is set when reconciling.changed fired: Reconciling may
	// have ended, so a restored unit is due.
	reconciled bool
	// copiedUnits is the units the Transfers hooks changed, published
	// once their Write commits.
	copiedUnits []Unit
	jobs        chan struct{}
}

func New(cfg Config) *Moves {
	return &Moves{
		cfg: cfg, plans: newPlans(cfg.Timings.Plans), poke: make(chan struct{}, 1),
		running: map[UnitID]bool{}, fired: map[Wait]*scope{}, jobs: make(chan struct{}, cfg.Timings.Jobs),
	}
}

// Poke has the runner pass soon.
func (m *Moves) Poke() {
	select {
	case m.poke <- struct{}{}:
	default:
	}
}

func (m *Moves) now() time.Time { return m.cfg.Clock() }

// fence refuses with ErrReconciling while a restored Controller is
// Reconciling.
func (m *Moves) fence(ctx context.Context, r storage.Reader[file]) error {
	on, err := m.cfg.Reconciling(ctx, r)
	if err == nil && on {
		return ErrReconciling
	}
	return err
}

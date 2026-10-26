// Package grabs owns search runs, the decision and Grabs. It is the only
// package that knows the whole walk from a search to a placed file, and
// nothing imports it but its API and the composition root.
package grabs

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("grabs", schemaFiles) }

type (
	GrabID   = db.GrabID
	SearchID = db.SearchID
	Stage    = db.Stage
	Code     = db.Code
	Outcome  = db.Outcome
)

const (
	Fetching     = db.Fetching
	Downloading  = db.Downloading
	Probing      = db.Probing
	Transferring = db.Transferring
	Importing    = db.Importing
	Imported     = db.Imported
	Failed       = db.Failed
)

const (
	IndexerUnavailable Code = "indexer_unavailable"
	TorrentUnreachable Code = "torrent_unreachable"
	TorrentInvalid     Code = "torrent_invalid"
	InfohashMismatch   Code = "infohash_mismatch"
	TorrentV2Only      Code = "torrent_v2_only"
	NoDownloadClient   Code = "no_download_client"
	DownloadFailed     Code = "download_failed"
	Removed            Code = "removed"
	NoVideoFile        Code = "no_video_file"
	ProbeFailed        Code = "probe_failed"
	FillsNothing       Code = "fills_nothing"
	TransferFailed     Code = "transfer_failed"
	ImportFailed       Code = "import_failed"
	NodeWrittenOff     Code = "node_written_off"
	RootChanged        Code = "root_changed"
)

const (
	Grabbed                  Outcome = "grabbed"
	NothingAcceptable        Outcome = "nothing_acceptable"
	SearchIndexerUnavailable Outcome = "indexer_unavailable"
	SearchIndexerRefused     Outcome = "indexer_refused"
)

type Timings struct {
	// RetryMin and RetryMax bound the backoff before a step retries after
	// a link outage.
	RetryMin time.Duration `yaml:"retry_min"`
	RetryMax time.Duration `yaml:"retry_max"`
	// Progress is the least time between two grabs.changed for one Grab's
	// download progress.
	Progress time.Duration `yaml:"progress"`
}

func DefaultTimings() Timings {
	return Timings{RetryMin: time.Second, RetryMax: time.Minute, Progress: time.Second}
}

type Changed struct {
	GrabID string `json:"grabId" format:"uuid"`
	WorkID string `json:"workId" format:"uuid"`
}

type SearchChanged struct {
	SearchID string `json:"searchId" format:"uuid"`
	WorkID   string `json:"workId" format:"uuid"`
}

var (
	ChangedEvent       = live.AdminEvent[Changed]("grabs.changed")
	SearchChangedEvent = live.AdminEvent[SearchChanged]("grabs.search-changed")
)

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

var (
	ErrNoWork        = works.ErrNoWork
	ErrNothingWanted = errors.New("the Work has no wanted cell to search for")
	ErrNoIndexer     = errors.New("no Indexer is set up")
	ErrNoRoot        = roots.ErrNoRoot
	ErrNotSearchable = errors.New("the Work's Content Type can't be searched")
	ErrNoGrab        = errors.New("no Grab has that id")
)

type RootUnreachableError = roots.UnreachableError

// Roster is the Nodes and the Roles they hold.
type Roster interface {
	// Route is where r work homed on home goes now. It fails with
	// *nodes.Unrouted while no Node may take it.
	Route(r role.Name, home role.NodeID) (role.NodeID, error)
	// InStep reports whether node's Sync session is attached.
	InStep(node role.NodeID) bool
	// Changed is closed at the roster's next change.
	Changed() <-chan struct{}
	// WrittenOff reports whether the Admin wrote off what node holds for
	// r, so its owner gives that work up.
	WrittenOff(r role.Name, node role.NodeID) bool
}

// Downloads is the Downloading Role's Controller half.
type Downloads interface {
	Touch(nodes ...role.NodeID)
	// Moved is closed once a Node's reports that may move a Torrent's
	// outcome commit.
	Moved() <-chan struct{}
	// Sample is a Torrent's live sample, false while it has none.
	Sample(id downloadclients.ID) (downloadclients.Sample, bool)
	// Resolve is one long poll of node's search for link's metadata,
	// which ends at endsAt unless a peer sent it.
	Resolve(ctx context.Context, node role.NodeID, link magnet.Link, endsAt time.Time) (downloadclients.Resolution, error)
}

// Roots is the library roots as a Grab reaches them: bytes go into a
// root only through the Node that holds it.
type Roots interface {
	// Reachable is nil when root's Node may take new downloads into it.
	Reachable(ctx context.Context, r storage.Reader[file], root roots.Root) error
	// Watch is closed at the next change of a root's report.
	Watch() <-chan struct{}
	Place(ctx context.Context, root roots.ID, to storing.Rel, from storing.Payload) (int64, error)
	Move(ctx context.Context, root roots.ID, from, to storing.Rel) (int64, error)
}

type Config struct {
	Library   *storage.DB[file]
	Works     *works.Works
	Targets   *targets.Targets
	Files     *files.Files
	Indexers  *indexers.Indexers
	Terms     map[works.TypeID]indexers.Facet
	Roster    Roster
	Downloads Downloads
	Media     media.Remote
	Roots     Roots
	// Transfers copies a Grab's file from the Node that downloaded it to
	// the Node that holds its root.
	Transfers     *transfers.Transfers
	Timings       Timings
	Rand          *rand.Rand
	Changed       Publisher[Changed]
	SearchChanged Publisher[SearchChanged]
	FilesChanged  Publisher[files.Changed]
	// BeforeStep sees each step before it runs: the simulation tier's
	// hook, nil in production.
	BeforeStep func(context.Context, Stage)
	Log        *slog.Logger
}

type Grabs struct {
	cfg       Config
	poke      chan struct{}
	searching searching

	mu sync.Mutex
	// moved is each Grab a transfers hook moved, for grabs.changed once
	// the hook's Write commits.
	moved map[string]Changed
}

// New builds the Grab pipeline and registers its Transfers purpose.
func New(cfg Config) *Grabs {
	gr := &Grabs{cfg: cfg, poke: make(chan struct{}, 1), moved: map[string]Changed{}, searching: searching{peers: map[GrabID]int{}}}
	cfg.Transfers.Register(transfers.Purpose{
		Kind:     transfers.Grab,
		Verified: gr.transferVerified,
		Failed:   gr.transferFailed,
		After:    gr.transfersCommitted,
		Names:    transferNames,
	})
	return gr
}

// Release is the release a Grab took, as the Indexer answered it.
type Release struct {
	Title   string
	Size    int64
	Indexer indexers.ID
	Link    string
	// InfoHash is the Indexer's infohash attribute, nil when it sent none.
	InfoHash *bencode.InfoHash
}

// Torrent is the Torrent a Grab downloads, as the Grab reads it.
type Torrent struct {
	ID downloadclients.ID
	downloadclients.Attached
}

// Plan is placement, decided at probing before any Transfer (ADR 0004):
// the file and its probe, the root it goes under, its path there, and
// the Targets it fills in Profile order, the first naming the file.
type Plan struct {
	FileIndex int
	Probe     media.Probe
	// Root is zero once a finished Grab's root is deleted.
	Root roots.Root
	// Path is relative to the root, '/' separated.
	Path  string
	Fills []targets.TargetID
}

type Failure struct {
	Stage   Stage
	Code    Code
	Message string
}

// Entered is when a Grab entered a stage.
type Entered struct {
	Stage Stage
	At    time.Time
}

type Grab struct {
	ID     GrabID
	WorkID works.WorkID
	ItemID works.ItemID
	// Edition is what its cells want: its release's cut, or any.
	Edition targets.Edition
	Targets []targets.TargetID
	Release Release
	// Node downloads the release: the Indexer's home Node until the fetch,
	// then the Downloading Node the fetch handed the .torrent to.
	Node role.NodeID
	// FetchNode is the Node that fetched the .torrent: the Indexer's
	// Searching Node, or the Downloading Node that got a magnet link's
	// metadata from peers. Zero until one did.
	FetchNode role.NodeID
	// MetadataSince is when the Grab first asked for its magnet link's
	// metadata, zero until it did.
	MetadataSince time.Time
	Stage         Stage
	// Torrent is nil until the fetch attached one.
	Torrent *Torrent
	Plan    *Plan
	Failed  *Failure
	History []Entered
	Created time.Time
}

// Done reports whether the Grab has no step left.
func (g Grab) Done() bool { return g.Stage == Imported || g.Stage == Failed }

// related is every Grab's targets, fills and history, by Grab, and the
// Torrents of the Grabs read.
type related struct {
	targets  map[GrabID][]targets.TargetID
	fills    map[GrabID][]targets.TargetID
	history  map[GrabID][]Entered
	torrents map[downloadclients.ID]downloadclients.Attached
}

func readRelated(ctx context.Context, r storage.Reader[file], rows []db.GrabsGrab) (related, error) {
	q := db.New(r)
	rel := related{targets: map[GrabID][]targets.TargetID{}, fills: map[GrabID][]targets.TargetID{}, history: map[GrabID][]Entered{}}
	var ids []downloadclients.ID
	for _, row := range rows {
		if !row.TorrentID.IsZero() {
			ids = append(ids, row.TorrentID)
		}
	}
	var err error
	if rel.torrents, err = downloadclients.Outcomes(ctx, r, ids); err != nil {
		return related{}, err
	}
	ts, err := q.ListTargets(ctx)
	if err != nil {
		return related{}, err
	}
	for _, t := range ts {
		rel.targets[t.GrabID] = append(rel.targets[t.GrabID], t.TargetID)
	}
	fs, err := q.ListFills(ctx)
	if err != nil {
		return related{}, err
	}
	for _, f := range fs {
		rel.fills[f.GrabID] = append(rel.fills[f.GrabID], f.TargetID)
	}
	ss, err := q.ListStages(ctx)
	if err != nil {
		return related{}, err
	}
	for _, s := range ss {
		rel.history[s.GrabID] = append(rel.history[s.GrabID], Entered{Stage: s.Stage, At: s.EnteredAt.Time()})
	}
	return rel, nil
}

// load parses a row. The schema's checks guarantee each stage's columns,
// so a row that parses holds what its stage needs.
func load(ctx context.Context, r storage.Reader[file], row db.GrabsGrab, rel related) (Grab, error) {
	g := Grab{
		ID: row.ID, WorkID: row.WorkID, ItemID: row.ItemID, Edition: row.Edition, Targets: rel.targets[row.ID],
		Release: Release{Title: row.ReleaseTitle, Size: row.ReleaseSize, Indexer: row.IndexerID, Link: row.ReleaseLink},
		Node:    row.NodeID, FetchNode: row.FetchNodeID, MetadataSince: row.MetadataStartedAt.Time(), Stage: row.Stage,
		History: rel.history[row.ID], Created: row.CreatedAt.Time(),
	}
	if row.ReleaseInfoHash != nil {
		h := bencode.InfoHash(row.ReleaseInfoHash)
		g.Release.InfoHash = &h
	}
	if !row.TorrentID.IsZero() {
		g.Torrent = &Torrent{ID: row.TorrentID, Attached: rel.torrents[row.TorrentID]}
	}
	if row.Path != nil {
		g.Plan = &Plan{FileIndex: int(*row.FileIndex), Path: *row.Path, Fills: rel.fills[row.ID]}
		if !row.RootID.IsZero() {
			root, err := roots.Get(ctx, r, row.RootID)
			if err != nil {
				return Grab{}, err
			}
			g.Plan.Root = root
		}
		if err := json.Unmarshal([]byte(*row.Probe), &g.Plan.Probe); err != nil {
			return Grab{}, fmt.Errorf("grab %s: probe: %w", row.ID, err)
		}
	}
	if row.FailureCode != nil {
		g.Failed = &Failure{Stage: *row.FailedStage, Code: *row.FailureCode, Message: *row.FailureMessage}
	}
	return g, nil
}

func loadAll(ctx context.Context, r storage.Reader[file], rows []db.GrabsGrab) ([]Grab, error) {
	rel, err := readRelated(ctx, r, rows)
	if err != nil {
		return nil, err
	}
	out := make([]Grab, len(rows))
	for i, row := range rows {
		if out[i], err = load(ctx, r, row, rel); err != nil {
			return nil, err
		}
	}
	return out, nil
}

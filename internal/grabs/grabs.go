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
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/media"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/torrent/engine"
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
	Facts(node role.NodeID) (*dlv1.Facts, bool)
}

// Payloads is where this Node's downloaded torrent files lie.
type Payloads interface {
	Path(h bencode.InfoHash, index int) (string, error)
}

type Config struct {
	Library  *storage.DB[file]
	Works    *works.Works
	Targets  *targets.Targets
	Files    *files.Files
	Indexers *indexers.Indexers
	Terms    map[works.TypeID]indexers.Facet
	Roster   Roster
	// Downloads is the Downloading Role's *role.Control.
	Downloads Downloads
	Media     media.Remote
	// Payloads is this Node's own client, for an import from a local
	// download.
	Payloads Payloads
	// Transfers receives a file from the Node that downloaded it.
	Transfers     *transfers.Receiver
	Linker        files.Linker
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
	cfg  Config
	poke chan struct{}
}

func New(cfg Config) *Grabs { return &Grabs{cfg: cfg, poke: make(chan struct{}, 1)} }

// Release is the release a Grab took, as the Indexer answered it.
type Release struct {
	Title   string
	Size    int64
	Indexer indexers.ID
	Link    string
	// InfoHash is the Indexer's infohash attribute, nil when it sent none.
	InfoHash *bencode.InfoHash
}

// Torrent is the fetched .torrent: its v1 info hash and the metainfo.
type Torrent struct {
	InfoHash bencode.InfoHash
	Metainfo []byte
}

// Download is how the downloading Node's client ended the download: its
// files, or the error it failed with.
type Download struct {
	Files []engine.File
	Error string
}

// Plan is placement, decided at probing before any Transfer (ADR 0004):
// the file and its probe, the root it goes under, its path there, and
// the Targets it fills in Profile order, the first naming the file.
type Plan struct {
	FileIndex int
	Probe     media.Probe
	Root      roots.Root
	// Path is relative to the root, '/' separated.
	Path  string
	Fills []targets.TargetID
}

// moves reports whether the file goes to a root on another Node than from,
// the one that downloaded it, so a Transfer brings it first.
func (p Plan) moves(from role.NodeID) bool { return p.Root.Node != from }

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
	// FetchNode is the Searching Node that fetched the .torrent, zero
	// until it did.
	FetchNode role.NodeID
	Stage     Stage
	Torrent   *Torrent
	Download  *Download
	Plan      *Plan
	Failed    *Failure
	History   []Entered
	Created   time.Time
}

// Done reports whether the Grab has no step left.
func (g Grab) Done() bool { return g.Stage == Imported || g.Stage == Failed }

// related is every Grab's targets, fills and history, by Grab.
type related struct {
	targets map[GrabID][]targets.TargetID
	fills   map[GrabID][]targets.TargetID
	history map[GrabID][]Entered
}

func readRelated(ctx context.Context, q *db.Queries) (related, error) {
	rel := related{targets: map[GrabID][]targets.TargetID{}, fills: map[GrabID][]targets.TargetID{}, history: map[GrabID][]Entered{}}
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
		Node:    row.NodeID, FetchNode: row.FetchNodeID, Stage: row.Stage, History: rel.history[row.ID], Created: row.CreatedAt.Time(),
	}
	if row.ReleaseInfoHash != nil {
		h := bencode.InfoHash(row.ReleaseInfoHash)
		g.Release.InfoHash = &h
	}
	if row.InfoHash != nil {
		g.Torrent = &Torrent{InfoHash: bencode.InfoHash(row.InfoHash), Metainfo: row.Metainfo}
	}
	switch {
	case row.DownloadFiles != nil:
		g.Download = &Download{}
		if err := json.Unmarshal([]byte(*row.DownloadFiles), &g.Download.Files); err != nil {
			return Grab{}, fmt.Errorf("grab %s: download files: %w", row.ID, err)
		}
	case row.DownloadError != nil:
		g.Download = &Download{Error: *row.DownloadError}
	}
	if row.Path != nil {
		root, err := roots.Get(ctx, r, row.RootID)
		if err != nil {
			return Grab{}, err
		}
		g.Plan = &Plan{FileIndex: int(*row.FileIndex), Root: root, Path: *row.Path, Fills: rel.fills[row.ID]}
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
	rel, err := readRelated(ctx, db.New(r))
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

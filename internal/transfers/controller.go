package transfers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/transfers/internal/ledger"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// Def is the Node-wide channel every claimed Node sends and receives
// Transfers on.
var Def = role.WideDef[*tv1.Intent, *tv1.Facts, *tv1.Result]{Name: role.Transfers}

// PurposeKind is a registered purpose: the kind of job that asks for
// Transfers.
type PurposeKind string

const Grab PurposeKind = "grab"

// Destination is where a Transfer's files go. Its Node is never stored:
// judge reads it from the destination each time, so a re-home moves it.
type Destination interface{ kind() string }

// Root is a library root destination. Its receiver stages file position
// at StagedRel(id, position) in the root.
type Root struct{ ID roots.ID }

func (Root) kind() string { return rootKind }

const rootKind = "root"

// rootOf is the root d names.
func rootOf(d Destination) roots.ID {
	r, ok := d.(Root)
	if !ok {
		panic("transfers: a destination of no known kind")
	}
	return r.ID
}

// FileRequest is one file a purpose asks to copy.
type FileRequest struct {
	// Group is the purpose's key for the files Verified fires for
	// together, empty for one group.
	Group string
	// Name is the file's name in words, such as "Heat (1995).mkv".
	Name   string
	Source source.Ref
	Size   int64
}

// Request is what a purpose asks to copy. Its files go in the order
// given.
type Request struct {
	Purpose     PurposeKind
	Job         string
	Source      role.NodeID
	Destination Destination
	Files       []FileRequest
}

type Transfer struct {
	ID          ID
	Purpose     PurposeKind
	Job         string
	Source      role.NodeID
	Destination Destination
	Created     time.Time
	Files       []File
	Failure     *Failure
}

type File struct {
	Position int
	Group    string
	Name     string
	Source   source.Ref
	Size     int64
	// Verified is at the destination's current place.
	Verified bool
	Placed   bool
}

// Code is a stored failure's kind.
type Code string

const (
	SourceGone       Code = "source_gone"
	SourceFailed     Code = "source_failed"
	Damaged          Code = "damaged"
	SourceWrittenOff Code = "source_written_off"
	Rerouted         Code = "rerouted"
)

// readBack is a damaged failure's detail when only the receiver's read of
// its own staged file failed the check.
const readBack = "read_back"

// Failure is a Transfer's stored failure.
type Failure struct {
	Code Code
	// Position is the file it names, -1 for none.
	Position int
	// Detail is the engine's text, or "read_back".
	Detail string
	// Sentence is its words, with the names it reads in the Write that
	// set it.
	Sentence string
}

// Purpose is a registered kind and its hooks. The hooks run inside a
// library Write, write only their own purpose's rows, may Cancel their own
// Transfer, do no IO, and treat a state they don't expect as nothing to
// do.
type Purpose struct {
	Kind PurposeKind
	// Verified fires once per group, in the Write that verified its last
	// file, so groups fire in the order they complete.
	Verified func(ctx context.Context, tx *storage.Tx[storage.Library], t Transfer, group string) error
	// Failed fires once, in the Write that sets the failure.
	Failed func(ctx context.Context, tx *storage.Tx[storage.Library], t Transfer, f Failure) error
	// After runs once that Write commits, with the jobs whose hooks fired.
	// It must not block.
	After func(jobs []string)
	// Names is each of jobs' name in words, such as "Heat (1995)", read
	// in rd. A job it leaves out has no name.
	Names func(ctx context.Context, rd storage.Reader[storage.Library], jobs []string) (map[string]string, error)
}

// Publisher publishes transfers.changed.
type Publisher interface {
	Publish(ctx context.Context, ev Changed)
}

// Changed is transfers.changed's payload: the Transfers whose view moved
// and the Nodes whose send limit changed.
type Changed struct {
	TransferIDs []string `json:"transferIds" nullable:"false"`
	NodeIDs     []string `json:"nodeIds" nullable:"false"`
}

var ChangedEvent = live.AdminEvent[Changed]("transfers.changed")

// ControllerConfig builds the Controller half. Each function field reads
// another feature's state, so a test can stand in for it.
type ControllerConfig struct {
	Library *storage.DB[storage.Library]
	// Touch and Facts reach Def's Control, which is served after New.
	Touch func(nodes ...role.NodeID)
	Facts func(node role.NodeID) (*tv1.Facts, bool)
	// Board is the roster now.
	Board func() Board
	// Destinations is each root of ids with its State and volume, read in
	// rd. RootsMoved closes when a State it reads moves.
	Destinations func(ctx context.Context, rd storage.Reader[storage.Library], ids []roots.ID) (map[roots.ID]roots.Destination, error)
	RootsMoved   func() <-chan struct{}
	// Readiness is each of refs' readiness on node, read in rd. Moved
	// closes when it changes.
	Readiness func(ctx context.Context, rd storage.Reader[storage.Library], node role.NodeID, refs []source.Ref) (map[source.Ref]source.Readiness, error)
	Moved     func() <-chan struct{}
	// Sample asks node for a fresh sample of its volumes.
	Sample func(ctx context.Context, node role.NodeID) error
	// Hub wakes the loop on the events of the inputs above that publish
	// one: roots, volumes, Health and Nodes.
	Hub     *live.Hub
	Changed Publisher
	Timings Timings
	Log     *slog.Logger
}

// Transfers is the Controller half: the record of every Transfer, the
// one loop that judges them, and the sync channel's derive and apply.
type Transfers struct {
	cfg      ControllerConfig
	purposes map[PurposeKind]Purpose
	poke     chan struct{}

	mu sync.Mutex
	// fullAt is when a receiver first reported each Transfer's disk full,
	// so only a volume sample newer than it lifts the wait.
	fullAt map[ID]time.Time
	// fired is each Node's jobs whose hooks fired in its Results batch,
	// for After once the batch commits.
	fired map[role.NodeID]map[PurposeKind][]string
	// limited is the Nodes whose send limit changed since the last
	// transfers.changed.
	limited []role.NodeID
	// judged is every Node's intent as the loop last judged it, and
	// strayed the Nodes served another intent since.
	judged  intents
	strayed map[role.NodeID]bool
}

func New(cfg ControllerConfig) *Transfers {
	return &Transfers{
		cfg:      cfg,
		purposes: map[PurposeKind]Purpose{},
		poke:     make(chan struct{}, 1),
		fullAt:   map[ID]time.Time{},
		fired:    map[role.NodeID]map[PurposeKind][]string{},
		strayed:  map[role.NodeID]bool{},
	}
}

// Register adds a purpose. Call it before Run. A kind registered twice,
// or with a hook missing, panics.
func (t *Transfers) Register(p Purpose) {
	if _, dup := t.purposes[p.Kind]; dup {
		panic("transfers: purpose " + string(p.Kind) + " is registered twice")
	}
	if p.Verified == nil || p.Failed == nil || p.After == nil || p.Names == nil {
		panic("transfers: purpose " + string(p.Kind) + " is missing a hook")
	}
	t.purposes[p.Kind] = p
}

// Touch tells the loop the library changed. Call it after any commit that
// called Start, Placed or Cancel.
func (t *Transfers) Touch() {
	select {
	case t.poke <- struct{}{}:
	default:
	}
}

// Start records r in tx and returns its id. A Request for a (purpose, job)
// that already has a record returns that record's id, so a repeated step
// converges. Call Touch after the commit.
func (t *Transfers) Start(ctx context.Context, tx *storage.Tx[storage.Library], r Request) (ID, error) {
	if _, ok := t.purposes[r.Purpose]; !ok {
		panic("transfers: purpose " + string(r.Purpose) + " isn't registered")
	}
	q := ledger.New(tx)
	id, err := q.TransferOfJob(ctx, ledger.TransferOfJobParams{Purpose: string(r.Purpose), Job: r.Job})
	switch {
	case err == nil:
		return id, nil
	case !errors.Is(err, storage.ErrNoRows):
		return ID{}, err
	}
	id = ID{UUID: storage.NewUUID()}
	root := rootOf(r.Destination)
	row := ledger.InsertTransferParams{ID: id, Purpose: string(r.Purpose), Job: r.Job, SourceNode: r.Source, DestinationKind: r.Destination.kind(), DestinationRoot: &root, CreatedAt: storage.Now()}
	if err := q.InsertTransfer(ctx, row); err != nil {
		return ID{}, fmt.Errorf("record the Transfer: %w", err)
	}
	for i, f := range r.Files {
		s, ok := f.Source.(source.Torrent)
		if !ok {
			panic("transfers: a source of no known kind")
		}
		file := ledger.InsertFileParams{TransferID: id, Position: int64(i), Item: f.Group, Name: f.Name, SourceKind: string(f.Source.Kind()), InfoHash: s.Hash[:], FileIndex: new(int64(s.File)), Size: f.Size}
		if err := q.InsertFile(ctx, file); err != nil {
			return ID{}, fmt.Errorf("record the Transfer's file %d: %w", i, err)
		}
	}
	return id, nil
}

// Placed marks file position of id placed. The record goes with its last
// unplaced file. A Transfer or file it doesn't have is nothing to do.
func (t *Transfers) Placed(ctx context.Context, tx *storage.Tx[storage.Library], id ID, position int) error {
	q := ledger.New(tx)
	if err := q.SetPlaced(ctx, ledger.SetPlacedParams{TransferID: id, Position: int64(position)}); err != nil {
		return err
	}
	left, err := q.CountUnplaced(ctx, id)
	if err != nil || left > 0 {
		return err
	}
	return q.DeleteTransfer(ctx, id)
}

// Cancel deletes id's record. Its receiver deletes the staging folder once
// the id leaves its intent.
func (t *Transfers) Cancel(ctx context.Context, tx *storage.Tx[storage.Library], id ID) error {
	return ledger.New(tx).DeleteTransfer(ctx, id)
}

// Find is p's record for job, false when it has none.
func (t *Transfers) Find(ctx context.Context, rd storage.Reader[storage.Library], p PurposeKind, job string) (Transfer, bool, error) {
	q := ledger.New(rd)
	id, err := q.TransferOfJob(ctx, ledger.TransferOfJobParams{Purpose: string(p), Job: job})
	if errors.Is(err, storage.ErrNoRows) {
		return Transfer{}, false, nil
	}
	if err != nil {
		return Transfer{}, false, err
	}
	tr, err := load(ctx, rd, id)
	return tr, err == nil, err
}

// LiveIn is every Transfer id with a record, for the files sweep.
func LiveIn(ctx context.Context, rd storage.Reader[storage.Library]) (map[ID]bool, error) {
	ids, err := ledger.New(rd).ListTransferIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[ID]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// StagedRel is where a root destination stages file position of id,
// relative to the root.
func StagedRel(id ID, position int) storing.Rel {
	rel, err := storing.ParseRel(path.Join(storing.StagingDir, id.String(), strconv.Itoa(position)+".part"))
	if err != nil {
		panic(fmt.Sprintf("transfers: staged path: %v", err))
	}
	return rel
}

// load reads id's record in rd, its files verified at its destination's
// place.
func load(ctx context.Context, rd storage.Reader[storage.Library], id ID) (Transfer, error) {
	q := ledger.New(rd)
	row, err := q.GetTransfer(ctx, id)
	if err != nil {
		return Transfer{}, err
	}
	files, err := q.FilesOf(ctx, id)
	if err != nil {
		return Transfer{}, err
	}
	root, err := roots.Get(ctx, rd, *row.DestinationRoot)
	if err != nil {
		return Transfer{}, err
	}
	return transferOf(row, files, root.Place)
}

func transferOf(row ledger.Transfer, files []ledger.File, place int64) (Transfer, error) {
	t := Transfer{
		ID: row.ID, Purpose: PurposeKind(row.Purpose), Job: row.Job, Source: row.SourceNode,
		Destination: Root{ID: *row.DestinationRoot}, Created: row.CreatedAt.Time(),
	}
	if row.FailureCode != nil {
		t.Failure = &Failure{Code: Code(*row.FailureCode), Position: -1, Detail: row.FailureDetail}
		if row.FailurePosition != nil {
			t.Failure.Position = int(*row.FailurePosition)
		}
	}
	for _, f := range files {
		ref, err := refOf(f)
		if err != nil {
			return Transfer{}, err
		}
		t.Files = append(t.Files, File{
			Position: int(f.Position), Group: f.Item, Name: f.Name, Source: ref, Size: f.Size,
			Verified: f.VerifiedPlace != nil && *f.VerifiedPlace == place, Placed: f.Placed,
		})
	}
	return t, nil
}

func refOf(f ledger.File) (source.Ref, error) {
	if source.Kind(f.SourceKind) != source.TorrentKind {
		return nil, fmt.Errorf("file %d of Transfer %s has the source kind %q", f.Position, f.TransferID, f.SourceKind)
	}
	var t source.Torrent
	if copy(t.Hash[:], f.InfoHash) != len(t.Hash) {
		return nil, fmt.Errorf("file %d of Transfer %s has a %d-byte info hash", f.Position, f.TransferID, len(f.InfoHash))
	}
	t.File = int(*f.FileIndex)
	return t, nil
}

// Package indexers owns the Indexers. The Controller keeps each Indexer.
// The Node it routes to runs its protocol (package searching), and this
// package sends that Node a job and reads the typed answer. It never
// parses a site's response (ADR 0017).
package indexers

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("indexers", schemaFiles) }

type ID = db.IndexerID

// Facet is how a Content Type's Items are searched. A type that can't be
// searched answers the zero Terms.
type Facet interface {
	SearchTerms(ctx context.Context, r storage.Reader[storage.Library], item works.ItemID) (Terms, error)
}

type Terms struct {
	// Mode is the Torznab t= value the type searches with.
	Mode string
	// IDs are Torznab id parameters, by name.
	IDs map[string]string
	// Query is the free-text search when the Indexer takes none of IDs.
	Query string
}

type Timings struct {
	// Job is a job's deadline from when its Node started it.
	Job time.Duration `yaml:"job"`
	// Poll is the least gap between two Run calls of one job.
	Poll time.Duration `yaml:"poll"`
}

func DefaultTimings() Timings { return Timings{Job: 2 * time.Minute, Poll: time.Second} }

const protocol = "torznab"

// Caps are an Indexer's search modes, keyed by their t= value, each with
// the parameters it takes in the order the Indexer lists them.
type Caps struct {
	Modes map[string][]string `json:"modes"`
}

// Indexer is a Torznab Indexer and the Node that reaches it.
type Indexer struct {
	ID       ID
	Name     string
	Protocol string
	URL      string
	// Key is zero when the migration that sealed keys dropped it, until an
	// Admin enters it again.
	Key       secret.Sealed
	HomeNode  role.NodeID
	Caps      Caps
	CreatedAt time.Time
}

type Changed struct {
	IndexerID string `json:"indexerId" format:"uuid"`
}

var ChangedEvent = live.AdminEvent[Changed]("indexers.changed")

type Publisher interface {
	Publish(ctx context.Context, v Changed)
}

var (
	ErrNameTaken       = errors.New("another Indexer has that name")
	ErrNoSearchingNode = errors.New("no Node holds Searching")
	ErrAPIKeyDropped   = errors.New("the Indexer's API key was dropped")
	ErrNoIndexer       = errors.New("no such Indexer")
	// ErrLate is a job that passed its deadline.
	ErrLate = errors.New("the Indexer didn't respond in time")
)

// UnavailableError is a call that didn't reach the Indexer, brought back
// nothing usable, or met a site that limited or failed it. A site that
// stopped the Indexer is a *searching.Stopped.
type UnavailableError struct {
	// Wait is true when no Node could take the call, or its link was down,
	// mismatched or lost the call: the same call may succeed once the
	// roster changes.
	Wait bool
	Err  error
}

func (e *UnavailableError) Error() string { return "the Indexer is unavailable: " + e.Err.Error() }

func (e *UnavailableError) Unwrap() error { return e.Err }

// Roster is the Nodes and the Roles they hold.
type Roster interface {
	Self() role.NodeID
	// Pick is prefer when it may take new r work, else the Controller when
	// it may, else the first Node by id that may.
	Pick(r role.Name, prefer role.NodeID) (role.NodeID, bool)
	// Route is where r work homed on home goes now, or why it waits.
	Route(r role.Name, home role.NodeID) (role.NodeID, error)
	// Away is how node stands while it is assigned r and out of step.
	Away(r role.Name, node role.NodeID) (nodes.Absence, bool)
}

type Config struct {
	Library *storage.DB[file]
	// Sealer opens an Indexer's key only to put it on the wire.
	Sealer *secret.Sealer
	Roster Roster
	// Searching reaches the Searching service on a Node.
	Searching func(role.NodeID) searching.Remote
	Changed   Publisher
	Timings   Timings
	Log       *slog.Logger
}

type Indexers struct{ cfg Config }

func New(cfg Config) *Indexers { return &Indexers{cfg: cfg} }

// Draft is an Indexer as an Admin enters it.
type Draft struct {
	Name   string
	URL    string
	APIKey string
}

// Create saves d once its home Node has read its caps. The home Node is
// ADR 0017's default: this Node when it holds Searching, else the first
// Node paired that does.
func (x *Indexers) Create(ctx context.Context, _ identity.Admin, d Draft) (Indexer, error) {
	switch taken, err := db.New(x.cfg.Library).NameTaken(ctx, d.Name); {
	case err != nil:
		return Indexer{}, err
	case taken:
		return Indexer{}, ErrNameTaken
	}
	home, err := x.home()
	if err != nil {
		return Indexer{}, err
	}
	idx := Indexer{ID: ID{UUID: storage.NewUUID()}, Name: d.Name, Protocol: protocol, URL: d.URL, HomeNode: home}
	idx.Key = x.cfg.Sealer.Seal([]byte(d.APIKey), apiKeyColumn.Row(idx.ID.String()))
	if idx.Caps, err = x.caps(ctx, idx); err != nil {
		return Indexer{}, err
	}
	caps, err := json.Marshal(idx.Caps)
	if err != nil {
		return Indexer{}, fmt.Errorf("encode the caps: %w", err)
	}
	created := storage.Now()
	idx.CreatedAt = created.Time()
	err = x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		switch taken, err := q.NameTaken(ctx, d.Name); {
		case err != nil:
			return err
		case taken:
			return ErrNameTaken
		}
		return q.InsertIndexer(ctx, db.InsertIndexerParams{
			ID: idx.ID, Name: idx.Name, Protocol: protocol, Url: idx.URL, ApiKeySealed: idx.Key, HomeNodeID: home, Caps: string(caps), CreatedAt: created,
		})
	})
	if err != nil {
		return Indexer{}, err
	}
	x.cfg.Changed.Publish(ctx, Changed{IndexerID: idx.ID.String()})
	return idx, nil
}

// SetAPIKey enters Indexer id's key again, once its home Node has read the
// caps with it.
func (x *Indexers) SetAPIKey(ctx context.Context, _ identity.Admin, id ID, key string) (Indexer, error) {
	idx, err := Get(ctx, x.cfg.Library, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Indexer{}, ErrNoIndexer
	}
	if err != nil {
		return Indexer{}, err
	}
	idx.Key = x.cfg.Sealer.Seal([]byte(key), apiKeyColumn.Row(idx.ID.String()))
	if idx.Caps, err = x.caps(ctx, idx); err != nil {
		return Indexer{}, err
	}
	caps, err := json.Marshal(idx.Caps)
	if err != nil {
		return Indexer{}, fmt.Errorf("encode the caps: %w", err)
	}
	err = x.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		n, err := db.New(tx).SetIndexerKey(ctx, db.SetIndexerKeyParams{ApiKeySealed: idx.Key, Caps: string(caps), ID: id})
		if err == nil && n == 0 {
			return ErrNoIndexer
		}
		return err
	})
	if err != nil {
		return Indexer{}, err
	}
	x.cfg.Changed.Publish(ctx, Changed{IndexerID: idx.ID.String()})
	return idx, nil
}

func (x *Indexers) home() (role.NodeID, error) {
	if node, ok := x.cfg.Roster.Pick(role.Searching, x.cfg.Roster.Self()); ok {
		return node, nil
	}
	return role.NodeID{}, ErrNoSearchingNode
}

// List lists every Indexer by name.
func (x *Indexers) List(ctx context.Context, _ identity.Admin) ([]Indexer, error) {
	return All(ctx, x.cfg.Library)
}

// All reads every Indexer in r, by name.
func All(ctx context.Context, r storage.Reader[file]) ([]Indexer, error) {
	rows, err := db.New(r).ListIndexers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Indexer, len(rows))
	for i, row := range rows {
		if out[i], err = indexer(row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Get reads Indexer id in r.
func Get(ctx context.Context, r storage.Reader[file], id ID) (Indexer, error) {
	row, err := db.New(r).GetIndexer(ctx, id)
	if err != nil {
		return Indexer{}, err
	}
	return indexer(row)
}

func indexer(row db.IndexersIndexer) (Indexer, error) {
	idx := Indexer{ID: row.ID, Name: row.Name, Protocol: row.Protocol, URL: row.Url, Key: row.ApiKeySealed, HomeNode: row.HomeNodeID, CreatedAt: row.CreatedAt.Time()}
	if err := json.Unmarshal([]byte(row.Caps), &idx.Caps); err != nil {
		return Indexer{}, fmt.Errorf("indexer %s: decode the caps: %w", row.ID, err)
	}
	return idx, nil
}

// caps are idx's search modes, as its home Node's Test read them.
func (x *Indexers) caps(ctx context.Context, idx Indexer) (Caps, error) {
	a, _, err := x.run(ctx, idx, searching.TestCall{})
	if err != nil {
		return Caps{}, err
	}
	return Caps{Modes: a.Report.Capabilities.Modes}, nil
}

// Release is one normalized Release from one Indexer.
type Release struct {
	Indexer ID
	searching.Release
}

// Search asks the Node idx routes through to search for terms. It fails
// with *searching.Stopped or *UnavailableError.
func (x *Indexers) Search(ctx context.Context, idx Indexer, terms Terms) ([]Release, error) {
	a, _, err := x.run(ctx, idx, searching.SearchCall{Query: Query(idx.Caps, terms)})
	if err != nil {
		return nil, err
	}
	out := make([]Release, len(a.Releases))
	for i, r := range a.Releases {
		out[i] = Release{Indexer: idx.ID, Release: r}
	}
	return out, nil
}

// Query is the query for terms under caps: terms' mode with the first id
// parameter caps list for that mode, else a free-text search. It chooses
// by declared capability, so it never retries.
func Query(c Caps, terms Terms) searching.Query {
	for _, p := range c.Modes[terms.Mode] {
		if id, ok := terms.IDs[p]; ok {
			return searching.Query{Mode: terms.Mode, IDs: map[string]string{p: id}}
		}
	}
	return searching.Query{Mode: "search", Text: terms.Query}
}

// Got is what a release's link leads to: a Torrent or a Magnet.
type Got interface{ got() }

// Torrent is a .torrent's bytes as the Indexer sent them, and the
// Searching Node that fetched them.
type Torrent struct {
	Raw  []byte
	Node role.NodeID
}

// Magnet is a magnet link: the release's own, or the one its link
// redirected to.
type Magnet struct{ Link magnet.Link }

func (Torrent) got() {}
func (Magnet) got()  {}

// Fetch is what link leads to. A magnet link needs no Node. Any other
// link goes to the Node idx routes through, which may fetch it from any
// host and follow it to a magnet link. It fails with *searching.Stopped,
// *UnavailableError, *magnet.InvalidError or bencode.ErrV2Only.
func (x *Indexers) Fetch(ctx context.Context, idx Indexer, link searching.Link) (Got, error) {
	if link.IsMagnet() {
		return parseMagnet(link)
	}
	a, node, err := x.run(ctx, idx, searching.FetchCall{Link: link})
	if err != nil {
		return nil, err
	}
	switch got := a.Got.(type) {
	case searching.Torrent:
		return Torrent{Raw: got.Raw, Node: node}, nil
	case searching.Magnet:
		return parseMagnet(got.Link)
	}
	return nil, &UnavailableError{Err: errors.New("the Searching Node's response holds neither a .torrent nor a magnet link")}
}

func parseMagnet(link searching.Link) (Got, error) {
	l, err := link.Magnet()
	if err != nil {
		return nil, err
	}
	return Magnet{Link: l}, nil
}

// Waiting is the absence of idx's home Node while idx's calls wait for it:
// out of step, but not yet past the offline grace.
func (x *Indexers) Waiting(idx Indexer) (nodes.Absence, bool) {
	a, away := x.cfg.Roster.Away(role.Searching, idx.HomeNode)
	return a, away && !a.Lapsed
}

// run sends call on idx to the Node idx routes through now, routed afresh
// each time so it goes home once home is back, and the Node that ran it.
// An outcome other than ok fails it: a stop as *searching.Stopped, any
// other as *UnavailableError.
func (x *Indexers) run(ctx context.Context, idx Indexer, call searching.Call) (searching.Answer, role.NodeID, error) {
	node, err := x.cfg.Roster.Route(role.Searching, idx.HomeNode)
	if err != nil {
		return searching.Answer{}, role.NodeID{}, &UnavailableError{Wait: true, Err: err}
	}
	job, err := x.job(idx, call)
	if err != nil {
		return searching.Answer{}, role.NodeID{}, err
	}
	a, err := x.ask(ctx, node, job)
	if err != nil {
		return searching.Answer{}, role.NodeID{}, unavailable(err)
	}
	if stopped, ok := errors.AsType[*searching.Stopped](a.Outcome); ok {
		return searching.Answer{}, role.NodeID{}, stopped
	}
	if a.Outcome != nil {
		return searching.Answer{}, role.NodeID{}, &UnavailableError{Err: a.Outcome}
	}
	return a, node, nil
}

var apiKeyColumn = secret.NewColumn("indexers_indexer.api_key_sealed")

// job is call on idx as the Node runs it, and the one place an Indexer's
// key is opened, for the length of one job. A dropped key fails as an
// *UnavailableError the same call can't recover from.
func (x *Indexers) job(idx Indexer, call searching.Call) (searching.Job, error) {
	if !idx.Key.Set() {
		return searching.Job{}, &UnavailableError{Err: ErrAPIKeyDropped}
	}
	key, err := x.cfg.Sealer.Open(idx.Key, apiKeyColumn.Row(idx.ID.String()))
	if err != nil {
		return searching.Job{}, fmt.Errorf("indexer %s: %w", idx.ID, err)
	}
	return searching.Job{
		ID: storage.NewUUID().String(), Indexer: idx.ID.String(), Kind: searching.Torznab, Address: idx.URL, APIKey: string(key), Call: call,
	}, nil
}

func unavailable(err error) error {
	if why, ok := rpc.NotSent(err); ok {
		return &UnavailableError{Wait: why != rpc.NotHolder, Err: err}
	}
	return &UnavailableError{Wait: rpc.Lost(err), Err: err}
}

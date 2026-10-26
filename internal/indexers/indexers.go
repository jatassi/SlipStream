// Package indexers owns the Indexers and the Searching Role's wire calls.
// The Controller keeps each Indexer and parses every Torznab answer; the
// Indexer's home Node only speaks the protocol (ADR 0017).
package indexers

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/magnet"
	"github.com/jatassi/slipstream/internal/nodes"
	searchv1 "github.com/jatassi/slipstream/internal/pb/slipstream/search/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/search/v1/searchv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
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
	// Request bounds one request to an Indexer, its body included.
	Request time.Duration `yaml:"request"`
}

func DefaultTimings() Timings { return Timings{Request: 30 * time.Second} }

const protocol = "torznab"

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
)

// UnavailableError is a call that didn't reach the Indexer or brought
// back nothing usable.
type UnavailableError struct {
	// Wait is true when no Node could take the call, or its link was down,
	// mismatched or lost the call: the same call may succeed once the
	// roster changes.
	Wait bool
	Err  error
}

func (e *UnavailableError) Error() string {
	return "the Indexer is unavailable: " + rpc.MessageOf(e.Err)
}

// Reason is the cause alone as a sentence a person reads, for a problem's
// detail or a failure's Detail, whose title already says the Indexer is
// unavailable.
func (e *UnavailableError) Reason() string {
	m := rpc.MessageOf(e.Err)
	return strings.ToUpper(m[:1]) + m[1:] + "."
}

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
	Searching func(role.NodeID) searchv1connect.SearchingServiceHandler
	Changed   Publisher
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

func (x *Indexers) caps(ctx context.Context, idx Indexer) (Caps, error) {
	node, w, err := x.call(idx)
	if err != nil {
		return Caps{}, err
	}
	res, err := x.cfg.Searching(node).Caps(ctx, &searchv1.CapsRequest{Indexer: w})
	if err != nil {
		return Caps{}, unavailable(err)
	}
	if err := answered(res.GetBody(), res.GetStatus()); err != nil {
		return Caps{}, err
	}
	c, err := ParseCaps(res.GetBody())
	if err != nil {
		return Caps{}, &UnavailableError{Err: err}
	}
	return c, nil
}

// Search asks the Node idx routes through to search for terms and parses
// the answer. It fails with *UnavailableError or *RefusedError.
func (x *Indexers) Search(ctx context.Context, idx Indexer, terms Terms) ([]Item, error) {
	node, w, err := x.call(idx)
	if err != nil {
		return nil, err
	}
	res, err := x.cfg.Searching(node).Search(ctx, &searchv1.SearchRequest{Indexer: w, Params: Query(idx.Caps, terms)})
	if err != nil {
		return nil, unavailable(err)
	}
	if err := answered(res.GetBody(), res.GetStatus()); err != nil {
		return nil, err
	}
	items, err := ParseItems(res.GetBody())
	if err != nil {
		return nil, &UnavailableError{Err: err}
	}
	return items, nil
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
// link goes to the Node idx routes through, must be on the Indexer's own
// host, and may redirect to a magnet link. It fails with
// *UnavailableError, *magnet.InvalidError or bencode.ErrV2Only.
func (x *Indexers) Fetch(ctx context.Context, idx Indexer, link string) (Got, error) {
	if magnet.Is(link) {
		return parseMagnet(link)
	}
	node, w, err := x.call(idx)
	if err != nil {
		return nil, err
	}
	res, err := x.cfg.Searching(node).FetchTorrent(ctx, &searchv1.FetchTorrentRequest{Indexer: w, Url: link})
	if err != nil {
		return nil, unavailable(err)
	}
	switch got := res.GetGot().(type) {
	case *searchv1.FetchTorrentResponse_Torrent:
		return Torrent{Raw: got.Torrent, Node: node}, nil
	case *searchv1.FetchTorrentResponse_Magnet:
		return parseMagnet(got.Magnet)
	}
	return nil, &UnavailableError{Err: errors.New("the Searching Node's response holds neither a .torrent nor a magnet link")}
}

func parseMagnet(link string) (Got, error) {
	l, err := magnet.Parse(link)
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

// call is the Node a call to idx goes through now, routed afresh each
// time so it goes home once home is back, and idx on the wire.
func (x *Indexers) call(idx Indexer) (role.NodeID, *searchv1.Indexer, error) {
	node, err := x.cfg.Roster.Route(role.Searching, idx.HomeNode)
	if err != nil {
		return role.NodeID{}, nil, &UnavailableError{Wait: true, Err: err}
	}
	w, err := x.wire(idx)
	return node, w, err
}

var apiKeyColumn = secret.NewColumn("indexers_indexer.api_key_sealed")

// wire is the one place an Indexer's key is opened, for the length of one
// call to its home Node. A dropped key fails as an *UnavailableError the
// same call can't recover from.
func (x *Indexers) wire(idx Indexer) (*searchv1.Indexer, error) {
	if !idx.Key.Set() {
		return nil, &UnavailableError{Err: ErrAPIKeyDropped}
	}
	key, err := x.cfg.Sealer.Open(idx.Key, apiKeyColumn.Row(idx.ID.String()))
	if err != nil {
		return nil, fmt.Errorf("indexer %s: %w", idx.ID, err)
	}
	return &searchv1.Indexer{Url: idx.URL, ApiKey: string(key)}, nil
}

// answered is the Torznab error body holds, else an *UnavailableError for
// a status other than success.
func answered(body []byte, status int32) error {
	if r := refusal(body); r != nil {
		return r
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return &UnavailableError{Err: fmt.Errorf("the Indexer responded with status %d", status)}
	}
	return nil
}

func unavailable(err error) error {
	if why, ok := rpc.NotSent(err); ok {
		return &UnavailableError{Wait: why != rpc.NotHolder, Err: err}
	}
	return &UnavailableError{Wait: rpc.Lost(err), Err: err}
}

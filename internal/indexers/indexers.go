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
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	searchv1 "github.com/jatassi/slipstream/internal/pb/slipstream/search/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/search/v1/searchv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
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
	ID        ID
	Name      string
	Protocol  string
	URL       string
	APIKey    string
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
)

// UnavailableError is a call that didn't reach the Indexer or brought
// back nothing usable.
type UnavailableError struct {
	// Wait is true when the home Node's link was down, mismatched or lost
	// the call: the same call may succeed once the link is up again.
	Wait bool
	Err  error
}

func (e *UnavailableError) Error() string { return "the Indexer is unavailable: " + e.Err.Error() }

func (e *UnavailableError) Unwrap() error { return e.Err }

// Roster is the Nodes and the Roles they hold.
type Roster interface {
	Self() role.NodeID
	// HoldersByPairing is every Node holding r, first paired first.
	HoldersByPairing(ctx context.Context, r role.Name) ([]role.NodeID, error)
}

type Config struct {
	Library *storage.DB[file]
	Roster  Roster
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
	home, err := x.home(ctx)
	if err != nil {
		return Indexer{}, err
	}
	idx := Indexer{ID: ID{UUID: storage.NewUUID()}, Name: d.Name, Protocol: protocol, URL: d.URL, APIKey: d.APIKey, HomeNode: home}
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
			ID: idx.ID, Name: idx.Name, Protocol: protocol, Url: idx.URL, ApiKey: idx.APIKey, HomeNodeID: home, Caps: string(caps), CreatedAt: created,
		})
	})
	if err != nil {
		return Indexer{}, err
	}
	x.cfg.Changed.Publish(ctx, Changed{IndexerID: idx.ID.String()})
	return idx, nil
}

func (x *Indexers) home(ctx context.Context) (role.NodeID, error) {
	holders, err := x.cfg.Roster.HoldersByPairing(ctx, role.Searching)
	switch {
	case err != nil:
		return role.NodeID{}, err
	case len(holders) == 0:
		return role.NodeID{}, ErrNoSearchingNode
	case slices.Contains(holders, x.cfg.Roster.Self()):
		return x.cfg.Roster.Self(), nil
	}
	return holders[0], nil
}

// List lists every Indexer by name.
func (x *Indexers) List(ctx context.Context, _ identity.Admin) ([]Indexer, error) {
	rows, err := db.New(x.cfg.Library).ListIndexers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Indexer, len(rows))
	for i, row := range rows {
		out[i] = Indexer{ID: row.ID, Name: row.Name, Protocol: row.Protocol, URL: row.Url, APIKey: row.ApiKey, HomeNode: row.HomeNodeID, CreatedAt: row.CreatedAt.Time()}
		if err := json.Unmarshal([]byte(row.Caps), &out[i].Caps); err != nil {
			return nil, fmt.Errorf("indexer %s: decode the caps: %w", row.ID, err)
		}
	}
	return out, nil
}

func (x *Indexers) caps(ctx context.Context, idx Indexer) (Caps, error) {
	res, err := x.cfg.Searching(idx.HomeNode).Caps(ctx, &searchv1.CapsRequest{Indexer: wire(idx)})
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

// Search asks idx's home Node to search for terms and parses the answer.
// It fails with *UnavailableError or *RefusedError.
func (x *Indexers) Search(ctx context.Context, idx Indexer, terms Terms) ([]Item, error) {
	res, err := x.cfg.Searching(idx.HomeNode).Search(ctx, &searchv1.SearchRequest{Indexer: wire(idx), Params: Query(idx.Caps, terms)})
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

// FetchTorrent asks idx's home Node for the .torrent at link, which must
// be on the Indexer's own host. It fails with *UnavailableError.
func (x *Indexers) FetchTorrent(ctx context.Context, idx Indexer, link string) ([]byte, error) {
	res, err := x.cfg.Searching(idx.HomeNode).FetchTorrent(ctx, &searchv1.FetchTorrentRequest{Indexer: wire(idx), Url: link})
	if err != nil {
		return nil, unavailable(err)
	}
	return res.GetTorrent(), nil
}

func wire(idx Indexer) *searchv1.Indexer {
	return &searchv1.Indexer{Url: idx.URL, ApiKey: idx.APIKey}
}

// answered is the Torznab error body holds, else an *UnavailableError for
// a status other than success.
func answered(body []byte, status int32) error {
	if r := refusal(body); r != nil {
		return r
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return &UnavailableError{Err: fmt.Errorf("the Indexer answered status %d", status)}
	}
	return nil
}

func unavailable(err error) error {
	if why, ok := rpc.NotSent(err); ok {
		return &UnavailableError{Wait: why != rpc.NotHolder, Err: err}
	}
	return &UnavailableError{Wait: rpc.Lost(err), Err: err}
}

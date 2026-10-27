// Package indexers owns every Indexer, its status, its sessions on each
// Node, its limits, the Indexer catalog and the Trackers Indexers reach.
// The Controller keeps all of it. Every protocol runs on the Node an
// Indexer routes to (package searching): this package sends that Node a
// job and records the typed answer. It never parses a site's response
// (ADR 0017).
//
// Each record has one writer. Status is written only by record, from a
// job's answer, through the pure next. A session is upserted only when it
// was earned later than the stored one. A limit is a count of reserved
// indexers_query rows, reserved inside a Write, so concurrent calls never
// pass it.
package indexers

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("indexers", schemaFiles) }

type (
	ID        = db.IndexerID
	TrackerID = db.TrackerID
	QueryID   = db.QueryID
)

// Kind is how an Indexer is reached.
type Kind string

const (
	Catalog Kind = "catalog"
	Torznab Kind = "torznab"
	RSS     Kind = "rss"
)

// DefaultPriority is an Indexer's priority until an Admin sets one.
const DefaultPriority = 25

// Indexer is one Indexer as the Controller keeps it.
type Indexer struct {
	ID            ID
	Name          string
	Kind          Kind
	Home          role.NodeID
	On            bool
	AutoSearch    bool
	RSS           bool
	AllowFailover bool
	// Priority is 1 to 50, lower wins.
	Priority int
	// Categories is nil for every category the Indexer advertises.
	Categories []newznab.ID
	// Media is the Content Types an RSS feed serves, nil for every other
	// kind.
	Media    []works.TypeID
	Searches Limit
	Grabs    Limit
	Source   Source
	// Secrets is one sealed Secrets.
	Secrets secret.Sealed
	// FeedHost is shown in place of a feed's sealed address.
	FeedHost string
	Caps     Caps
	// Notes are what a pin move changed, for the Indexer page.
	Notes    []string
	Revision int64
	Created  time.Time
}

// Limit is how many calls an Indexer takes in a sliding window. A zero
// Count is no limit.
type Limit struct {
	Count int
	Per   Window
}

type Window string

const (
	Hour Window = "hour"
	Day  Window = "day"
)

func (w Window) Duration() time.Duration {
	if w == Day {
		return 24 * time.Hour
	}
	return time.Hour
}

// Source is a Definition or a Feed.
type Source interface{ source() }

type Origin string

const (
	FromCatalog Origin = "catalog"
	FromCustom  Origin = "custom"
)

// Definition is the definition version a catalog Indexer pins and its
// plain settings.
type Definition struct {
	ID, SHA     string
	BaseAddress string
	// Plain is the select and checkbox values and the username.
	Plain map[string]string
}

// Feed is a Torznab or RSS feed.
type Feed struct {
	// Tracker is the Tracker name the Admin gave, for an RSS feed or a
	// Torznab feed whose items name no source.
	Tracker     string
	AllowNoSize bool
}

func (Definition) source() {}
func (Feed) source()       {}

// Secrets are an Indexer's sealed values, opened only to put on the wire.
type Secrets struct {
	// Address is a feed's whole address, which may carry a passkey.
	Address string            `json:"address,omitempty"`
	APIKey  string            `json:"apiKey,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

var secretsColumn = secret.NewColumn("indexers_indexer.secrets_sealed")

// Caps are an Indexer's search modes, keyed by their t= value, each with
// the parameters it takes in the order the Indexer lists them, and its
// categories.
type Caps struct {
	Modes      map[string][]string `json:"modes"`
	Categories []newznab.ID        `json:"categories,omitempty"`
}

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
	// Limited is how long a limited Indexer waits when the site gave no
	// Retry-After.
	Limited time.Duration `yaml:"limited"`
	// Ladder is how long a failing Indexer backs off at each rung.
	Ladder []time.Duration `yaml:"ladder"`
	// PackageShare is the percent of the catalog's files that, once
	// changed, a refresh fetches as package.zip in place of one by one.
	PackageShare int `yaml:"package_share"`
	// CatalogRequest is how long one request to the Indexer catalog may
	// take.
	CatalogRequest time.Duration `yaml:"catalog_request"`
}

func DefaultTimings() Timings {
	return Timings{
		Job: 2 * time.Minute, Poll: time.Second, Limited: time.Minute,
		Ladder: []time.Duration{
			time.Minute, 5 * time.Minute, 15 * time.Minute, 30 * time.Minute, time.Hour,
			3 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour,
		},
		PackageShare: 25, CatalogRequest: 2 * time.Minute,
	}
}

// Endpoints is read on the Controller only. A Worker's value is unused.
type Endpoints struct {
	// Catalog is the Indexer catalog's base address, to which SlipStream
	// adds the schema version it reads.
	Catalog endpoint.URL `yaml:"catalog"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{Catalog: endpoint.MustParseURL("https://indexers.prowlarr.com/master")}
}

type Changed struct {
	IndexerID string `json:"indexerId" format:"uuid"`
}

type StatusChanged struct {
	IndexerID string `json:"indexerId" format:"uuid"`
}

type CatalogChanged struct {
	RefreshedAt *time.Time `json:"refreshedAt" doc:"Null until a refresh has read the catalog"`
}

var (
	ChangedEvent        = live.AdminEvent[Changed]("indexers.changed")
	StatusChangedEvent  = live.AdminEvent[StatusChanged]("indexers.status-changed")
	CatalogChangedEvent = live.AdminEvent[CatalogChanged]("indexers.catalog-changed")
)

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

var (
	ErrNameTaken       = errors.New("another Indexer has that name")
	ErrNoSearchingNode = errors.New("no Node holds Searching")
	ErrNoIndexer       = errors.New("no such Indexer")
	ErrNotSearching    = errors.New("that Node doesn't hold Searching")
	// ErrLate is a job that passed its deadline.
	ErrLate = errors.New("the Indexer didn't respond in time")
)

// UnavailableError is a call that didn't reach the Indexer's site: its
// Node's link was down, mismatched or lost the call, or the Node no longer
// holds Searching.
type UnavailableError struct {
	// Wait is true when the same call may succeed once the roster changes.
	Wait bool
	Err  error
}

func (e *UnavailableError) Error() string { return "the Indexer is unavailable: " + e.Err.Error() }

func (e *UnavailableError) Unwrap() error { return e.Err }

// WaitError is a call no Node may take now, and why.
type WaitError struct{ Outcome Outcome }

func (e *WaitError) Error() string { return "the Indexer waits for its home Node" }

type Config struct {
	Library *storage.DB[file]
	// Sealer opens an Indexer's secrets and sessions only to put them on
	// the wire.
	Sealer *secret.Sealer
	Roster Roster
	// Searching reaches the Searching service on a Node.
	Searching func(role.NodeID) searching.Remote
	Changed   Publisher[Changed]
	Status    Publisher[StatusChanged]
	Catalogs  Publisher[CatalogChanged]
	// HTTP reaches the Indexer catalog from the Controller.
	HTTP      *http.Client
	Endpoints Endpoints
	// CatalogNext is when the Indexer catalog refresh runs next, zero when
	// it isn't waiting for a slot.
	CatalogNext func(context.Context) (time.Time, error)
	Timings     Timings
	// Zone is the zone of the times inside sentences.
	Zone *time.Location
	Log  *slog.Logger
}

type Indexers struct {
	cfg  Config
	late background
}

func New(cfg Config) *Indexers { return &Indexers{cfg: cfg} }

// Run keeps late jobs polled until ctx ends, then waits for the ones still
// polling, which end with ctx.
func (x *Indexers) Run(ctx context.Context) error {
	x.late.run(ctx)
	return nil
}

// background runs work on the Indexers' own context while Run runs, so a
// shutdown waits for it rather than leaking it.
type background struct {
	mu  sync.Mutex
	ctx context.Context
	wg  sync.WaitGroup
}

func (b *background) run(ctx context.Context) {
	b.mu.Lock()
	b.ctx = ctx
	b.mu.Unlock()
	<-ctx.Done()
	b.mu.Lock()
	b.ctx = nil
	b.mu.Unlock()
	b.wg.Wait()
}

// goOn runs f on Run's context, reporting false when Run isn't running.
func (b *background) goOn(f func(ctx context.Context)) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.ctx == nil {
		return false
	}
	ctx := b.ctx
	b.wg.Go(func() { f(ctx) })
	return true
}

// All reads every Indexer in r, by name.
func All(ctx context.Context, r storage.Reader[file]) ([]Indexer, error) {
	rows, err := db.New(r).ListIndexers(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Indexer, len(rows))
	for i, row := range rows {
		if out[i], err = indexerOf(row); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Get reads Indexer id in r: ErrNoIndexer when there is none.
func Get(ctx context.Context, r storage.Reader[file], id ID) (Indexer, error) {
	row, err := db.New(r).GetIndexer(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Indexer{}, ErrNoIndexer
	}
	if err != nil {
		return Indexer{}, err
	}
	return indexerOf(row)
}

func indexerOf(row db.IndexersIndexer) (Indexer, error) {
	idx := Indexer{
		ID: row.ID, Name: row.Name, Kind: Kind(row.Kind), Home: row.HomeNodeID, On: row.IsOn, AutoSearch: row.AutoSearch,
		RSS: row.Rss, AllowFailover: row.AllowFailover, Priority: int(row.Priority),
		Searches: Limit{Count: int(row.SearchLimit), Per: Window(row.SearchWindow)},
		Grabs:    Limit{Count: int(row.GrabLimit), Per: Window(row.GrabWindow)},
		Secrets:  row.SecretsSealed, FeedHost: deref(row.FeedHost), Revision: row.Revision, Created: row.CreatedAt.Time(),
	}
	if row.DefinitionID != nil {
		d := Definition{ID: *row.DefinitionID, SHA: deref(row.DefinitionSha), BaseAddress: deref(row.BaseAddress)}
		if err := decode(row.Plain, &d.Plain); err != nil {
			return Indexer{}, fmt.Errorf("indexer %s: plain: %w", row.ID, err)
		}
		idx.Source = d
	} else {
		idx.Source = Feed{Tracker: deref(row.TrackerName), AllowNoSize: row.AllowNoSize}
	}
	for _, f := range []struct {
		raw *string
		v   any
	}{{row.Categories, &idx.Categories}, {row.Media, &idx.Media}, {&row.Caps, &idx.Caps}, {&row.Notes, &idx.Notes}} {
		if f.raw == nil {
			continue
		}
		if err := decode(*f.raw, f.v); err != nil {
			return Indexer{}, fmt.Errorf("indexer %s: %w", row.ID, err)
		}
	}
	return idx, nil
}

// row is idx as indexers_indexer holds it.
func row(idx Indexer, updated storage.Time) (db.InsertIndexerParams, error) {
	p := db.InsertIndexerParams{
		ID: idx.ID, Name: idx.Name, Kind: string(idx.Kind), HomeNodeID: idx.Home, IsOn: idx.On, AutoSearch: idx.AutoSearch,
		Rss: idx.RSS, AllowFailover: idx.AllowFailover, Priority: int64(idx.Priority),
		SearchLimit: int64(idx.Searches.Count), SearchWindow: string(window(idx.Searches.Per)),
		GrabLimit: int64(idx.Grabs.Count), GrabWindow: string(window(idx.Grabs.Per)),
		SecretsSealed: idx.Secrets, FeedHost: orNil(idx.FeedHost), Plain: "{}", Notes: "[]",
		Revision: idx.Revision, CreatedAt: storage.FromTime(idx.Created), UpdatedAt: updated,
	}
	switch s := idx.Source.(type) {
	case Definition:
		p.DefinitionID, p.DefinitionSha, p.BaseAddress = &s.ID, &s.SHA, &s.BaseAddress
		plain, err := encode(s.Plain)
		if err != nil {
			return db.InsertIndexerParams{}, err
		}
		p.Plain = *plain
	case Feed:
		p.TrackerName, p.AllowNoSize = orNil(s.Tracker), s.AllowNoSize
	}
	var err error
	if p.Categories, err = encode(idx.Categories); err != nil {
		return db.InsertIndexerParams{}, err
	}
	if p.Media, err = encode(idx.Media); err != nil {
		return db.InsertIndexerParams{}, err
	}
	caps, err := json.Marshal(idx.Caps)
	if err != nil {
		return db.InsertIndexerParams{}, fmt.Errorf("encode the caps: %w", err)
	}
	p.Caps = string(caps)
	if len(idx.Notes) > 0 {
		notes, err := encode(idx.Notes)
		if err != nil {
			return db.InsertIndexerParams{}, err
		}
		p.Notes = *notes
	}
	return p, nil
}

func window(w Window) Window {
	if w == "" {
		return Hour
	}
	return w
}

// encode is v as JSON, nil for a nil slice or map.
func encode[T any](v T) (*string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("encode %T: %w", v, err)
	}
	if string(b) == "null" {
		return nil, nil
	}
	s := string(b)
	return &s, nil
}

func decode(raw string, v any) error {
	if err := json.Unmarshal([]byte(raw), v); err != nil {
		return fmt.Errorf("decode %T: %w", v, err)
	}
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func orNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// open is idx's secrets in clear.
func (x *Indexers) open(idx Indexer) (Secrets, error) {
	plain, err := x.cfg.Sealer.Open(idx.Secrets, secretsColumn.Row(idx.ID.String()))
	if err != nil {
		return Secrets{}, fmt.Errorf("indexer %s: open the secrets: %w", idx.ID, err)
	}
	var s Secrets
	err = decode(string(plain), &s)
	return s, err
}

func (x *Indexers) seal(id ID, s Secrets) (secret.Sealed, error) {
	b, err := json.Marshal(s) //nolint:gosec // G117: sealed below
	if err != nil {
		return secret.Sealed{}, fmt.Errorf("encode the secrets: %w", err)
	}
	return x.cfg.Sealer.Seal(b, secretsColumn.Row(id.String())), nil
}

func (x *Indexers) changed(ctx context.Context, id ID) {
	x.cfg.Changed.Publish(ctx, Changed{IndexerID: id.String()})
}

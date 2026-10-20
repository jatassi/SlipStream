package metadata

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/metadata/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	metadatav1 "github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// DefaultLanguage and DefaultRegion are the metadata language and release
// region an Ask carries until an Admin sets others.
const (
	DefaultLanguage = "en-US"
	DefaultRegion   = "US"
)

// Owner is the Metadata Role's owner. A holder keeps no work, so the Role
// moves freely.
var Owner = nodes.Owner{Role: role.Metadata, Held: nodes.HoldsNothing}

// Roster is the Nodes and the Roles they hold.
type Roster interface {
	Self() role.NodeID
	// Pick is prefer when it may take new r work, else the Controller when
	// it may, else the first Node by id that may.
	Pick(r role.Name, prefer role.NodeID) (role.NodeID, bool)
	// PickOther is Pick for a call besides couldn't take.
	PickOther(r role.Name, besides role.NodeID) (role.NodeID, bool)
	// Why is why no Node may take new r work, in words.
	Why(r role.Name) string
}

type Config struct {
	// Library holds the Admin keys, sealed under Sealer.
	Library *storage.DB[storage.Library]
	Sealer  *secret.Sealer
	Roster  Roster
	// Service reaches MetadataService on a Node.
	Service func(role.NodeID) metadatav1connect.MetadataServiceHandler
	// Providers are the providers status lists and keys are set for, in
	// display order.
	Providers []Provider
	// Changed announces a change to a provider's status.
	Changed Publisher
	Timings Timings
	// Rand draws the cache generation.
	Rand *rand.Rand
	// Types are the Content Types a new region changes, announced on
	// TypeChanged.
	Types       []works.TypeID
	TypeChanged works.Publisher[works.TypeChanged]
	// Minter opens the artwork tokens the API hands out.
	Minter artwork.Minter
	// Local indexes the artwork store, whose images are files under
	// ArtworkDir.
	Local      *storage.DB[storage.Local]
	ArtworkDir string
}

type Publisher interface {
	Publish(ctx context.Context, v Changed)
}

// Metadata is the Controller's half of the Metadata Role: it fills an Ask
// for each call, sends the call to a holder and keeps each provider's
// status from the outcomes holders answer with.
type Metadata struct {
	cfg     Config
	bundled map[ProviderID]string
	// wake tells Run a retest was scheduled.
	wake chan struct{}
	// applying holds one settings change at a time.
	applying sync.Mutex

	mu sync.Mutex
	// generation is the cache generation every Ask carries. A holder whose
	// cache holds another drops it, so a new Controller process, or a
	// cleared cache, starts every holder's cache afresh.
	generation uint64
	settings   stored
	// yours are the Admin keys, opened.
	yours    map[ProviderID]string
	statuses map[ProviderID]*status
	retests  map[reach]*retest

	gallery *gallery
}

// New is the Controller's metadata with no Admin key read. Open reads
// them; New alone serves a test that sets none.
func New(cfg Config) *Metadata {
	m := &Metadata{
		cfg: cfg, generation: cfg.Rand.Uint64(), bundled: parseBundled(bundled), wake: make(chan struct{}, 1),
		yours: map[ProviderID]string{}, statuses: map[ProviderID]*status{}, retests: map[reach]*retest{}, gallery: newGallery(),
	}
	for _, p := range cfg.Providers {
		m.statuses[p.ID] = newStatus(p)
	}
	return m
}

// Open is New with the Admin keys library.db holds and the artwork store.
// It reaches no provider: status starts unused, and Run retests what calls
// find unreachable.
func Open(ctx context.Context, cfg Config) (*Metadata, error) {
	m := New(cfg)
	q := db.New(cfg.Library)
	settings, err := q.GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the metadata settings: %w", err)
	}
	m.settings = settings
	if m.gallery.store, err = m.openStore(ctx); err != nil {
		return nil, err
	}
	rows, err := q.ListKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the metadata keys: %w", err)
	}
	for _, row := range rows {
		key, err := cfg.Sealer.Open(row.ApiKeySealed, keyColumn.Row(row.Provider))
		if err != nil {
			return nil, fmt.Errorf("open the %s key: %w", row.Provider, err)
		}
		m.yours[ProviderID(row.Provider)] = string(key)
	}
	return m, nil
}

var _ works.Source = (*Metadata)(nil)

// Search reads no ratings provider and may answer from the cache, as a
// preview does.
func (m *Metadata) Search(ctx context.Context, t works.TypeID, query string) ([]works.Candidate, error) {
	res, err := send(ctx, m, func(ctx context.Context, s metadatav1connect.MetadataServiceHandler) (*metadatav1.SearchResponse, error) {
		return s.Search(ctx, &metadatav1.SearchRequest{Ask: m.ask(works.ForPreview), ContentType: string(t), Query: query})
	})
	if err != nil {
		return nil, err
	}
	if f := res.GetFailure(); f != nil {
		return nil, failureOf(f)
	}
	return works.DecodeCandidates(res.GetCandidates())
}

func (m *Metadata) Fetch(ctx context.Context, t works.TypeID, key works.Key, f works.Fetching) (works.Fetched, error) {
	res, err := send(ctx, m, func(ctx context.Context, s metadatav1connect.MetadataServiceHandler) (*metadatav1.FetchResponse, error) {
		return s.Fetch(ctx, &metadatav1.FetchRequest{Ask: m.ask(f.Purpose), ContentType: string(t), Key: string(key)})
	})
	if err != nil {
		return works.Fetched{}, err
	}
	if f := res.GetFailure(); f != nil {
		return works.Fetched{}, failureOf(f)
	}
	return works.DecodeFetched(res.GetFetched())
}

func (m *Metadata) ask(p works.Purpose) *metadatav1.Ask {
	keys := m.inEffect()
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.settings
	return askToWire(Ask{
		Language: LanguageChoice.Value(s.Language), Region: RegionChoice.Value(s.Region), Ratings: p != works.ForPreview,
		keys: keys, fresh: p == works.ForRefresh, cacheFor: CacheFor.Value(s.CacheForSeconds), generation: m.generation,
	})
}

func (m *Metadata) cacheGeneration() uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.generation
}

// reply is a holder's answer, which carries every outcome of the call.
type reply interface {
	GetOutcomes() []*metadatav1.Outcome
}

// send is call with the answer's outcomes folded into provider status.
func send[R reply](ctx context.Context, m *Metadata, do func(context.Context, metadatav1connect.MetadataServiceHandler) (R, error)) (R, error) {
	res, node, err := call(ctx, m, do)
	if err != nil {
		return res, err
	}
	m.fold(ctx, node, outcomesOf(res.GetOutcomes()))
	return res, nil
}

// call sends do to the holder Pick chooses and says which holder answered.
// A holder that wasn't reached, or lost the call, is tried once more on
// the holder PickOther chooses. While no Node may take metadata, or no
// holder answered in time, the call is ErrUnavailable saying why. A holder
// that answered with an error ran into a bug of the type.
func call[R any](ctx context.Context, m *Metadata, do func(context.Context, metadatav1connect.MetadataServiceHandler) (R, error)) (R, role.NodeID, error) {
	var none R
	node, ok := m.cfg.Roster.Pick(role.Metadata, m.cfg.Roster.Self())
	if !ok {
		return none, role.NodeID{}, unavailable(m.cfg.Roster.Why(role.Metadata))
	}
	res, err := do(ctx, m.cfg.Service(node))
	if unreached(err) && ctx.Err() == nil {
		if other, ok := m.cfg.Roster.PickOther(role.Metadata, node); ok {
			node = other
			res, err = do(ctx, m.cfg.Service(other))
		}
	}
	switch {
	case err == nil:
		return res, node, nil
	case !unreached(err) && rpc.Answered(err) && ctx.Err() == nil:
		return none, node, errors.New(rpc.MessageOf(err))
	case ctx.Err() != nil:
		return none, node, unavailable("the Node reading metadata didn't answer in time")
	}
	return none, node, unavailable("the Node reading metadata didn't answer: " + err.Error())
}

func unreached(err error) bool {
	_, notSent := rpc.NotSent(err)
	return notSent || rpc.Lost(err)
}

func unavailable(why string) error {
	return &works.SourceError{Source: "Metadata", Words: why, Err: works.ErrUnavailable}
}

// Package contenttypetest holds Fixture, the in-memory Content Type every
// core test registers instead of a real one.
package contenttypetest

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// ID is the Fixture's Content Type id.
const ID works.TypeID = "fixture"

// Description is the Fixture's: one level, and a field of each kind.
var Description = works.Description{
	Name:   "Fixture",
	Work:   works.Noun{One: "Shelf", Many: "Shelves"},
	Levels: []works.Noun{{One: "Box", Many: "Boxes"}},
	Item:   works.Noun{One: "Thing", Many: "Things"},
	Fields: []works.FieldSpec{
		{Key: "origin", Label: "Origin", Tier: works.TierWork, Kind: works.KindText},
		{Key: "packed", Label: "Packed", Tier: works.TierLevel(1), Kind: works.KindDate},
		{Key: "count", Label: "Count", Tier: works.TierItem, Kind: works.KindNumber},
		{Key: "runtime", Label: "Runtime", Tier: works.TierItem, Kind: works.KindDuration},
	},
}

// Endpoints is endpoints.content_types.fixture.
type Endpoints struct {
	API endpoint.URL `yaml:"api"`
}

// Record is what the Fixture's source holds for one Work now: its tree
// and each node's fields, levels and Items keyed by their keys.
type Record struct {
	Tree        works.Fetched
	WorkFields  []works.FieldValue
	LevelFields map[works.Key][]works.FieldValue
	ItemFields  map[works.Key][]works.FieldValue
}

// Fixture is an in-memory Content Type with id fixture and no tables. It
// keeps the fields Store hands it in memory.
type Fixture struct {
	mu        sync.Mutex
	records   map[works.Key]Record
	fails     map[works.Key]error
	searchErr error
	fetches   map[works.Key]int
	stored    map[works.WorkID]works.FieldSet
	hold      *hold
	opened    []Opened
}

// Opened is one build of the Fixture by its Def.
type Opened struct {
	Deps      contenttype.Deps
	Endpoints Endpoints
}

type hold struct {
	arrived sync.WaitGroup
	release chan struct{}
}

func New() *Fixture {
	return &Fixture{
		records: map[works.Key]Record{},
		fails:   map[works.Key]error{},
		fetches: map[works.Key]int{},
		stored:  map[works.WorkID]works.FieldSet{},
	}
}

// Def registers f, with an api endpoint defaulting to
// https://fixture.invalid/v1.
func (f *Fixture) Def() contenttype.Def[Endpoints] {
	return contenttype.Define(string(ID), nil, func() Endpoints {
		return Endpoints{API: endpoint.MustParseURL("https://fixture.invalid/v1")}
	}, func(d contenttype.Deps, e Endpoints) contenttype.Type {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.opened = append(f.opened, Opened{Deps: d, Endpoints: e})
		return f
	})
}

// Opens lists each build of f by its Def so far.
func (f *Fixture) Opens() []Opened {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.opened)
}

// Put sets what the source serves for key from now on.
func (f *Fixture) Put(key works.Key, r Record) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.records[key] = r
}

// Fail makes every Fetch of key answer err, until Fail(key, nil).
func (f *Fixture) Fail(key works.Key, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fails[key] = err
}

// FailSearch makes every Search answer err, until FailSearch(nil).
func (f *Fixture) FailSearch(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.searchErr = err
}

// Fetches counts the Fetches of key so far.
func (f *Fixture) Fetches(key works.Key) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fetches[key]
}

// Hold makes the next n Fetches wait. release waits until all n have
// started, then lets them answer together.
func (f *Fixture) Hold(n int) (release func()) {
	h := &hold{release: make(chan struct{})}
	h.arrived.Add(n)
	f.mu.Lock()
	f.hold = h
	f.mu.Unlock()
	return func() {
		h.arrived.Wait()
		f.mu.Lock()
		f.hold = nil
		f.mu.Unlock()
		close(h.release)
	}
}

func (f *Fixture) Describe() works.Description { return Description }

// Search matches a case-folded substring of each Work's label, in key
// order.
func (f *Fixture) Search(_ context.Context, query string) ([]works.Candidate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	var out []works.Candidate
	for _, key := range slices.Sorted(maps.Keys(f.records)) {
		r := f.records[key]
		if strings.Contains(strings.ToLower(r.Tree.Label), strings.ToLower(query)) {
			out = append(out, works.Candidate{Key: key, Label: r.Tree.Label, Fields: r.WorkFields})
		}
	}
	return out, nil
}

func (f *Fixture) Fetch(ctx context.Context, key works.Key) (works.Fetched, error) {
	f.mu.Lock()
	f.fetches[key]++
	h := f.hold
	f.mu.Unlock()
	if h != nil {
		h.arrived.Done()
		select {
		case <-h.release:
		case <-ctx.Done():
			return works.Fetched{}, works.ErrUnavailable
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.fails[key]; err != nil {
		return works.Fetched{}, err
	}
	r, ok := f.records[key]
	if !ok {
		return works.Fetched{}, works.ErrNotFound
	}
	detail, err := json.Marshal(r)
	if err != nil {
		return works.Fetched{}, fmt.Errorf("encode the record of %s: %w", key, err)
	}
	fetched := r.Tree
	fetched.Detail = detail
	return fetched, nil
}

// Store keeps the fields of every placed node in memory, beside the
// fields of nodes stored earlier.
func (f *Fixture) Store(_ context.Context, _ *storage.Tx[storage.Library], at works.Placed, detail []byte) error {
	var r Record
	if err := json.Unmarshal(detail, &r); err != nil {
		return fmt.Errorf("decode the detail of work %s: %w", at.Work, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	fs := f.stored[at.Work]
	if fs.Levels == nil {
		fs = works.FieldSet{Levels: map[works.LevelID][]works.FieldValue{}, Items: map[works.ItemID][]works.FieldValue{}}
	}
	fs.Work = r.WorkFields
	for key, id := range at.Levels {
		fs.Levels[id] = r.LevelFields[key]
	}
	for key, id := range at.Items {
		fs.Items[id] = r.ItemFields[key]
	}
	f.stored[at.Work] = fs
	return nil
}

func (f *Fixture) Fields(_ context.Context, _ storage.Reader[storage.Library], w works.WorkID) (works.FieldSet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fs := f.stored[w]
	return works.FieldSet{Work: fs.Work, Levels: maps.Clone(fs.Levels), Items: maps.Clone(fs.Items)}, nil
}

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
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

const ID works.TypeID = "fixture"

var Description = works.Description{
	Name:   "Fixture",
	Icon:   works.IconShapes,
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

type Endpoints struct {
	API endpoint.URL `yaml:"api"`
}

type Record struct {
	Tree        works.Fetched
	WorkFields  []works.FieldValue
	LevelFields map[works.Key][]works.FieldValue
	ItemFields  map[works.Key][]works.FieldValue
}

type Fixture struct {
	mu        sync.Mutex
	records   map[works.Key]Record
	fails     map[works.Key]error
	searchErr error
	fetches   map[works.Key]int
	stored    map[works.WorkID]works.FieldSet
	hold      *hold
	opened    []Opened
	timings   map[works.ItemID]targets.Timing
	cuts      []release.Cut
}

// Released is the date every Item is released on until SetTiming says
// otherwise.
var Released = works.Date{Year: 2000, Month: 1, Day: 1}

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
		timings: map[works.ItemID]targets.Timing{},
	}
}

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

func (f *Fixture) Opens() []Opened {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.opened)
}

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

func (f *Fixture) Fetches(key works.Key) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.fetches[key]
}

// Hold makes the next n Fetches wait. letGo waits until all n have
// started, then lets them answer together.
func (f *Fixture) Hold(n int) (letGo func()) {
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

// NamingTokens answers naming's empty answer: the Fixture names no files.
func (*Fixture) NamingTokens(context.Context, storage.Reader[storage.Library], works.ItemID) (map[string]string, error) {
	return map[string]string{}, nil
}

func (*Fixture) DefaultTemplates() naming.Templates { return naming.Templates{} }

// SearchTerms answers indexers' empty answer: the Fixture can't be
// searched.
func (*Fixture) SearchTerms(context.Context, storage.Reader[storage.Library], works.ItemID) (indexers.Terms, error) {
	return indexers.Terms{}, nil
}

// SetTiming makes Timings answer t for item, in place of Released and an
// unknown runtime.
func (f *Fixture) SetTiming(item works.ItemID, t targets.Timing) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.timings[item] = t
}

func (f *Fixture) SetCuts(cuts []release.Cut) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cuts = slices.Clone(cuts)
}

// Timings answers every stored Item of ws: its SetTiming, else Released
// and an unknown runtime.
func (f *Fixture) Timings(_ context.Context, _ storage.Reader[storage.Library], ws []works.WorkID) (map[works.ItemID]targets.Timing, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[works.ItemID]targets.Timing{}
	for _, w := range ws {
		for id := range f.stored[w].Items {
			t, ok := f.timings[id]
			if !ok {
				t = targets.Timing{Released: release.Known(Released)}
			}
			out[id] = t
		}
	}
	return out, nil
}

func (f *Fixture) Cuts() []release.Cut {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.cuts)
}

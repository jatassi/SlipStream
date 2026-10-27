// Package contenttypetest holds Fixture, the in-memory Content Type every
// core test registers instead of a real one.
package contenttypetest

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/metadata/metadatatest"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/setting"
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
		{Key: "tags", Label: "Tags", Tier: works.TierWork, Kind: works.KindTexts},
		{Key: "makers", Label: "Makers", Tier: works.TierWork, Kind: works.KindPeople},
		{Key: "page", Label: "Page", Tier: works.TierWork, Kind: works.KindLink},
	},
	Artwork: []artwork.Kind{
		{Key: "photo", Label: "Photo", Shape: artwork.ShapePoster, Prefetch: artwork.SizeMedium},
		{Key: "snap", Label: "Snap", Item: true, Shape: artwork.ShapeBackdrop},
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

// Fixture is the library half, and each provider half it opens reads its
// records.
type Fixture struct {
	mu        sync.Mutex
	records   map[works.Key]Record
	fails     map[works.Key]error
	searchErr error
	fetches   map[works.Key]int
	calls     []Call
	stored    map[works.WorkID]works.FieldSet
	hold      *hold
	opened    []Opened
	libraries []contenttype.LibraryDeps
	timings   map[works.ItemID]targets.Timing
	cuts      []release.Cut
	prefetch  []artwork.Want
	depth     *int64
	changed   []works.Key
	changeErr error
	growing   map[works.WorkID]bool
}

// Released is the date every Item is released on until SetTiming says
// otherwise.
var Released = works.Date{Year: 2000, Month: 1, Day: 1}

// Opened is one provider half a Node opened.
type Opened struct {
	Deps      contenttype.ProviderDeps
	Endpoints Endpoints
}

// Call is one Search, Fetch or page of Changes a provider half answered.
type Call struct {
	// Open is the provider half's index in Opens, -1 for Source's.
	Open  int
	Ask   metadata.Ask
	Query string
	Key   works.Key
	// Since and Page are a Changes call's.
	Since time.Time
	Page  int
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
		growing: map[works.WorkID]bool{},
	}
}

func (f *Fixture) Def() contenttype.Def[Endpoints] {
	return contenttype.Define(string(ID), nil, func() Endpoints {
		return Endpoints{API: endpoint.MustParseURL("https://fixture.invalid/v1")}
	}, contenttype.Halves[Endpoints]{
		Provider: func(d contenttype.ProviderDeps, e Endpoints) metadata.Source {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.opened = append(f.opened, Opened{Deps: d, Endpoints: e})
			return &source{f: f, open: len(f.opened) - 1}
		},
		Library: func(d contenttype.LibraryDeps) contenttype.Library {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.libraries = append(f.libraries, d)
			return f
		},
	})
}

// Opens is every provider half opened, in order.
func (f *Fixture) Opens() []Opened {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.opened)
}

// Libraries is every library half opened, in order.
func (f *Fixture) Libraries() []contenttype.LibraryDeps {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.libraries)
}

// Source is the Controller's metadata with a provider half of its own,
// for a test that opens the library half without a Node.
func (f *Fixture) Source() works.Source { return f.SourceAs(ID) }

// SourceAs is Source serving the Fixture's records as each of ids.
func (f *Fixture) SourceAs(ids ...works.TypeID) works.Source {
	sources := map[works.TypeID]metadata.Source{}
	for _, id := range ids {
		sources[id] = &source{f: f, open: -1}
	}
	return metadatatest.Serve(sources)
}

// Prefetch records wants, as works.Config's Artwork.
func (f *Fixture) Prefetch(wants []artwork.Want) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.prefetch = append(f.prefetch, wants...)
}

// Prefetched is every image works asked to prefetch, in order.
func (f *Fixture) Prefetched() []artwork.Want {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.prefetch)
}

// Calls is every Search and Fetch the provider halves answered, in order.
func (f *Fixture) Calls() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
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

type source struct {
	f    *Fixture
	open int
}

func (*source) Sources() []artwork.Source { return nil }

// Search matches a case-folded substring of each Work's label, in key
// order.
func (s *source) Search(_ context.Context, ask metadata.Ask, query string) ([]works.Candidate, error) {
	f := s.f
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, Call{Open: s.open, Ask: ask, Query: query})
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

func (s *source) Fetch(ctx context.Context, ask metadata.Ask, key works.Key) (works.Fetched, error) {
	f := s.f
	f.mu.Lock()
	f.fetches[key]++
	f.calls = append(f.calls, Call{Open: s.open, Ask: ask, Key: key})
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

// ChangesWindow is how far back the Fixture's changes list reaches, as
// TMDB's does.
const ChangesWindow = 14 * 24 * time.Hour

// ChangesPerPage is how many keys a page of the changes list holds.
const ChangesPerPage = 2

// Changes answers the keys Change set, ChangesPerPage a page, whatever
// since is within ChangesWindow of until.
func (s *source) Changes(_ context.Context, ask metadata.Ask, since, until time.Time, page int) (metadata.ChangesPage, error) {
	f := s.f
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, Call{Open: s.open, Ask: ask, Since: since, Page: page})
	switch {
	case f.changeErr != nil:
		return metadata.ChangesPage{}, f.changeErr
	case until.Sub(since) > ChangesWindow:
		return metadata.ChangesPage{}, metadata.ErrNoChanges
	}
	pages := max(1, (len(f.changed)+ChangesPerPage-1)/ChangesPerPage)
	from := min((page-1)*ChangesPerPage, len(f.changed))
	return metadata.ChangesPage{Keys: slices.Clone(f.changed[from:min(from+ChangesPerPage, len(f.changed))]), Pages: pages}, nil
}

// Change sets the keys every changes list answers.
func (f *Fixture) Change(keys ...works.Key) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.changed = keys
}

// FailChanges makes every changes list answer err, until
// FailChanges(nil).
func (f *Fixture) FailChanges(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.changeErr = err
}

// SetGrowing says whether Work w can still gain Items.
func (f *Fixture) SetGrowing(w works.WorkID, on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.growing[w] = on
}

func (f *Fixture) Growing(_ context.Context, _ storage.Reader[storage.Library], ws []works.WorkID) (map[works.WorkID]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[works.WorkID]bool{}
	for _, w := range ws {
		if f.growing[w] {
			out[w] = true
		}
	}
	return out, nil
}

// Store keeps the fields of every placed node in memory, beside the
// fields of nodes stored earlier, and reports whether any differed.
func (f *Fixture) Store(_ context.Context, _ *storage.Tx[storage.Library], at works.Placed, detail []byte) (bool, error) {
	var r Record
	if err := json.Unmarshal(detail, &r); err != nil {
		return false, fmt.Errorf("decode the detail of work %s: %w", at.Work, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	fs, ok := f.stored[at.Work]
	if !ok {
		fs = works.FieldSet{Levels: map[works.LevelID][]works.FieldValue{}, Items: map[works.ItemID][]works.FieldValue{}}
	}
	changed := !ok || !reflect.DeepEqual(fs.Work, r.WorkFields)
	fs.Work = r.WorkFields
	for key, id := range at.Levels {
		changed = changed || !reflect.DeepEqual(fs.Levels[id], r.LevelFields[key])
		fs.Levels[id] = r.LevelFields[key]
	}
	for key, id := range at.Items {
		changed = changed || !reflect.DeepEqual(fs.Items[id], r.ItemFields[key])
		fs.Items[id] = r.ItemFields[key]
	}
	f.stored[at.Work] = fs
	return changed, nil
}

func (f *Fixture) Fields(_ context.Context, _ storage.Reader[storage.Library], w works.WorkID) (works.FieldSet, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fs := f.stored[w]
	return works.FieldSet{Work: fs.Work, Levels: maps.Clone(fs.Levels), Items: maps.Clone(fs.Items)}, nil
}

// Depth is the Fixture's one setting. It is held in memory, so a Write
// that fails after ApplySettings still keeps it.
var Depth = setting.Count{Key: "fixture.depth", Label: "Boxes deep", Default: 3, Min: 1, Max: 9, Unit: setting.Noun{One: "box", Many: "boxes"}}

func (f *Fixture) SettingFields(context.Context, storage.Reader[storage.Library]) ([]setting.Field, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return []setting.Field{Depth.Field(f.depth)}, nil
}

func (f *Fixture) ApplySettings(_ context.Context, _ *storage.Tx[storage.Library], p setting.Patch) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	next := f.depth
	for key, raw := range p {
		if key != Depth.Key {
			return false, &setting.UnknownKeyError{Key: key}
		}
		s, err := Depth.ParseJSON(raw)
		if err != nil {
			return false, err
		}
		next = s.Override()
	}
	if reflect.DeepEqual(next, f.depth) {
		return false, nil
	}
	f.depth = next
	return true, nil
}

// NamingTokens names an Item's files after its Work's label, as Shelf,
// and its own, as Thing.
func (*Fixture) NamingTokens(ctx context.Context, r storage.Reader[storage.Library], item works.ItemID) (map[string]string, error) {
	work, it, err := works.ItemOf(ctx, r, item)
	if err != nil {
		return nil, err
	}
	return map[string]string{"Shelf": work.Label, "Thing": it.Label}, nil
}

func (*Fixture) DefaultTemplates() naming.Templates {
	return naming.Templates{Folder: "{Shelf}", File: "{Shelf} - {Thing} - {Edition Tag} {Target}"}
}

// SearchTerms answers indexers' empty answer: the Fixture can't be
// searched.
func (*Fixture) SearchTerms(context.Context, storage.Reader[storage.Library], works.ItemID) (indexers.Terms, error) {
	return indexers.Terms{}, nil
}

// Categories are none: the Fixture can't be searched.
func (*Fixture) Categories() []newznab.ID { return nil }

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

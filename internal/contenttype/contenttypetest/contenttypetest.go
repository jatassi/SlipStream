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
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

const ID works.TypeID = "fixture"

var Description = works.Description{
	Name:   "Fixture",
	Icon:   works.IconShapes,
	Source: "Fixture source",
	Work:   works.Noun{One: "Shelf", Many: "Shelves"},
	Levels: []works.Noun{{One: "Box", Many: "Boxes"}},
	Aside:  works.Noun{One: "Spare box", Many: "Spare boxes"},
	Item:   works.Noun{One: "Thing", Many: "Things"},
	Items:  works.ItemsMany,
	Fields: []works.FieldSpec{
		{Key: "origin", Label: "Origin", Tier: works.TierWork, Kind: works.KindText, In: []works.Place{works.PlaceFacts, works.PlaceAbout}, List: works.ListColumn | works.ListSort | works.ListFilter},
		{Key: "packed", Label: "Packed", Tier: works.TierLevel(1), Kind: works.KindDate, In: []works.Place{works.PlaceFacts}},
		{Key: "count", Label: "Count", Tier: works.TierItem, Kind: works.KindNumber, In: []works.Place{works.PlaceDetails}},
		{Key: "runtime", Label: "Runtime", Tier: works.TierItem, Kind: works.KindDuration, In: []works.Place{works.PlaceFacts}},
		{Key: "tags", Label: "Tags", Tier: works.TierWork, Kind: works.KindTexts, In: []works.Place{works.PlaceAbout}, List: works.ListColumn | works.ListFilter},
		{Key: "makers", Label: "Makers", Tier: works.TierWork, Kind: works.KindPeople, In: []works.Place{works.PlaceCredits}},
		{Key: "page", Label: "Page", Tier: works.TierWork, Kind: works.KindLink, In: []works.Place{works.PlaceDetails}},
		{Key: "code", Label: "Code", Tier: works.TierItem, Kind: works.KindText, In: []works.Place{works.PlaceCode}},
	},
	Artwork: []artwork.Kind{
		{Key: "photo", Label: "Photo", Shape: artwork.ShapePoster, Prefetch: artwork.SizeMedium},
		{Key: "label", Label: "Label", Tier: works.TierLevel(1), Shape: artwork.ShapePoster},
		{Key: "snap", Label: "Snap", Tier: works.TierItem, Shape: artwork.ShapeBackdrop},
	},
	Art:     works.ArtRoles{Poster: "photo"},
	Ratings: []works.RatingSource{"tmdb", "imdb"},
}

type Endpoints struct {
	API endpoint.URL `yaml:"api"`
}

type Record struct {
	Tree works.Fetched
	// Stacked is the record in OrderStack, nil when its source lists it in
	// OrderShelf only.
	Stacked     *works.Fetched
	WorkFields  []works.FieldValue
	LevelFields map[works.Key][]works.FieldValue
	ItemFields  map[works.Key][]works.FieldValue
	// Titles are the Work's titles beyond its label, which Titles answers.
	Titles []string
	// Order is the order a Fetch answered the record in; Fetch sets it.
	Order Order
}

// Order is the Fixture's one per-Work option: which of a record's trees a
// Fetch answers.
type Order string

const (
	// OrderShelf answers Record.Tree, and is every Work's default.
	OrderShelf Order = "shelf"
	// OrderStack answers Record.Stacked.
	OrderStack Order = "stack"
)

var orderChoice = setting.Choice[Order]{Key: "order", Label: "Order", Default: OrderShelf, Options: []setting.Option[Order]{
	{Value: OrderShelf, Label: "Shelf order"}, {Value: OrderStack, Label: "Stack order"},
}}

type options struct {
	Order Order `json:"order"`
}

// box is a stored level: the order of the tree that placed it, and its
// label. Level keys differ between a record's orders, so a Work's live
// levels name the order it was last stored in, and a Write that rolls back
// leaves no trace.
type box struct {
	order Order
	label string
}

// Fixture is the library half, and each provider half it opens reads its
// records. Its scan facet is scanning's empty answer.
type Fixture struct {
	scanning.None
	mu        sync.Mutex
	records   map[works.Key]Record
	fails     map[works.Key]error
	searchErr error
	fetches   map[works.Key]int
	calls     []Call
	stored    map[works.WorkID]works.FieldSet
	titles    map[works.WorkID][]string
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
	aliases   map[works.Key]works.Key
	boxes     map[works.LevelID]box
	stacked   map[works.WorkID]bool
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
	// Options are a Fetch's.
	Options []byte
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
		titles:  map[works.WorkID][]string{},
		timings: map[works.ItemID]targets.Timing{},
		growing: map[works.WorkID]bool{},
		aliases: map[works.Key]works.Key{},
		boxes:   map[works.LevelID]box{},
		stacked: map[works.WorkID]bool{},
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
		Library: func(d contenttype.LibraryDeps, _ Endpoints) contenttype.Library {
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

// Fetch answers the record's tree in the order options name, OrderShelf
// when they name none.
func (s *source) Fetch(ctx context.Context, ask metadata.Ask, key works.Key, opts []byte) (works.Fetched, error) {
	order := OrderShelf
	if opts != nil {
		var o options
		if err := json.Unmarshal(opts, &o); err != nil {
			return works.Fetched{}, fmt.Errorf("decode the options of %s: %w", key, err)
		}
		order = o.Order
	}
	f := s.f
	f.mu.Lock()
	f.fetches[key]++
	f.calls = append(f.calls, Call{Open: s.open, Ask: ask, Key: key, Options: opts})
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
	fetched := r.Tree
	if order == OrderStack {
		if r.Stacked == nil {
			return works.Fetched{}, works.ErrOrderUnavailable
		}
		fetched = *r.Stacked
	}
	r.Order = order
	detail, err := json.Marshal(r)
	if err != nil {
		return works.Fetched{}, fmt.Errorf("encode the record of %s: %w", key, err)
	}
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
	changed := !ok || !reflect.DeepEqual(fs.Work, r.WorkFields) || !slices.Equal(f.titles[at.Work], r.Titles) || f.stacked[at.Work] != (r.Stacked != nil)
	fs.Work = r.WorkFields
	f.titles[at.Work] = r.Titles
	f.stacked[at.Work] = r.Stacked != nil
	labels := labelsOf(r.tree(r.Order).Children, map[works.Key]string{})
	for key, id := range at.Levels {
		b := box{order: r.Order, label: labels[key]}
		changed = changed || !reflect.DeepEqual(fs.Levels[id], r.LevelFields[key]) || f.boxes[id] != b
		fs.Levels[id] = r.LevelFields[key]
		f.boxes[id] = b
	}
	for key, id := range at.Items {
		changed = changed || !reflect.DeepEqual(fs.Items[id], r.ItemFields[key])
		fs.Items[id] = r.ItemFields[key]
	}
	f.stored[at.Work] = fs
	return changed, nil
}

// tree is the record's tree in order o.
func (r Record) tree(o Order) works.Fetched {
	if o == OrderStack {
		return *r.Stacked
	}
	return r.Tree
}

// labelsOf adds the label of every level in es to into.
func labelsOf(es []works.Entry, into map[works.Key]string) map[works.Key]string {
	for _, e := range es {
		if len(e.Children) > 0 {
			into[e.Key] = e.Label
			labelsOf(e.Children, into)
		}
	}
	return into
}

// orderIn is the order Work w's live levels were placed in, in r:
// OrderShelf for a Work with none.
func (f *Fixture) orderIn(ctx context.Context, r storage.Reader[storage.Library], w works.WorkID) (Order, error) {
	levels, err := works.LevelsOf(ctx, r, w)
	if err != nil {
		return "", fmt.Errorf("contenttypetest: levels of work %s: %w", w, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, l := range levels {
		if l.WithdrawnAt.IsZero() {
			return f.boxes[l.ID].order, nil
		}
	}
	return OrderShelf, nil
}

// Fields answers the stored fields of each of w's nodes r holds, leaving
// out the nodes of a Write that rolled back.
func (f *Fixture) Fields(ctx context.Context, r storage.Reader[storage.Library], w works.WorkID) (works.FieldSet, error) {
	levels, err := works.LevelsOf(ctx, r, w)
	if err != nil {
		return works.FieldSet{}, fmt.Errorf("contenttypetest: levels of work %s: %w", w, err)
	}
	items, err := works.ItemsOf(ctx, r, works.Only([]works.WorkID{w}))
	if err != nil {
		return works.FieldSet{}, fmt.Errorf("contenttypetest: items of work %s: %w", w, err)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	fs := f.stored[w]
	out := works.FieldSet{Work: fs.Work, Levels: map[works.LevelID][]works.FieldValue{}, Items: map[works.ItemID][]works.FieldValue{}}
	for _, l := range levels {
		if v, ok := fs.Levels[l.ID]; ok {
			out.Levels[l.ID] = v
		}
	}
	for _, it := range items[w] {
		if v, ok := fs.Items[it.ID]; ok {
			out.Items[it.ID] = v
		}
	}
	return out, nil
}

// ListFields answers each Work's stored Work-tier fields that Description
// lists.
func (f *Fixture) ListFields(_ context.Context, _ storage.Reader[storage.Library], ws works.Among[works.WorkID]) (map[works.WorkID][]works.FieldValue, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[works.WorkID][]works.FieldValue{}
	for _, w := range ws.IDs() {
		for _, v := range f.stored[w].Work {
			if slices.ContainsFunc(Description.Fields, func(s works.FieldSpec) bool { return s.Tier == works.TierWork && s.Key == v.Key && s.List != 0 }) {
				out[w] = append(out[w], v)
			}
		}
	}
	return out, nil
}

// Titles answers the stored record's Titles.
func (f *Fixture) Titles(_ context.Context, _ storage.Reader[storage.Library], w works.WorkID) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.titles[w]), nil
}

// OptionFields answers w's order, offering OrderStack only when its source
// lists w that way.
func (f *Fixture) OptionFields(ctx context.Context, r storage.Reader[storage.Library], w works.WorkID) ([]setting.Field, error) {
	order, err := f.orderIn(ctx, r, w)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	stacked := f.stacked[w]
	f.mu.Unlock()
	c := orderChoice
	if !stacked {
		c.Options = c.Options[:1]
	}
	var override *Order
	if order != OrderShelf {
		override = &order
	}
	return []setting.Field{c.Field(override)}, nil
}

// FetchOptions answers w's order, nil for OrderShelf.
func (f *Fixture) FetchOptions(ctx context.Context, r storage.Reader[storage.Library], w works.WorkID) ([]byte, error) {
	order, err := f.orderIn(ctx, r, w)
	if err != nil {
		return nil, err
	}
	return encodeOrder(order)
}

// PatchOptions answers the options p's order makes of w's: ErrOrderUnavailable
// for OrderStack when w's source lists it in OrderShelf only.
func (f *Fixture) PatchOptions(ctx context.Context, r storage.Reader[storage.Library], w works.WorkID, p setting.Patch) ([]byte, error) {
	order, err := f.orderIn(ctx, r, w)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	stacked := f.stacked[w]
	f.mu.Unlock()
	for key, raw := range p {
		if key != orderChoice.Key {
			return nil, &setting.UnknownKeyError{Key: key}
		}
		v, err := orderChoice.ParseJSON(raw)
		if err != nil {
			return nil, err
		}
		order = orderChoice.Value(v.Override())
	}
	if order == OrderStack && !stacked {
		return nil, works.ErrOrderUnavailable
	}
	return encodeOrder(order)
}

func encodeOrder(o Order) ([]byte, error) {
	if o == OrderShelf || o == "" {
		return nil, nil
	}
	b, err := json.Marshal(options{Order: o})
	if err != nil {
		return nil, fmt.Errorf("contenttypetest: encode order %s: %w", o, err)
	}
	return b, nil
}

// Alias has KeysOf map other to the Shelf keyed key, as a series' TMDB
// id maps to its TVDB key.
func (f *Fixture) Alias(other, key works.Key) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.aliases[other] = key
}

// KeysOf maps each key Alias named to its Shelf's key, and others to none.
func (f *Fixture) KeysOf(_ context.Context, _ storage.Reader[storage.Library], keys []works.Key) ([]works.Key, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []works.Key
	for _, k := range keys {
		if key, ok := f.aliases[k]; ok {
			out = append(out, key)
		}
	}
	return out, nil
}

// Links answers none: a Shelf links nowhere.
func (*Fixture) Links(works.Key, []works.FieldValue) []works.WebLink { return nil }

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
// its level's, as Box, and its own, as Thing.
func (f *Fixture) NamingTokens(ctx context.Context, r storage.Reader[storage.Library], item works.ItemID) (map[string]string, error) {
	work, it, err := works.ItemOf(ctx, r, item)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return map[string]string{"Shelf": work.Label, "Box": f.boxes[it.LevelID].label, "Thing": it.Label}, nil
}

// DefaultTemplates put a file in a folder per box, and one in a spare box
// in Spare boxes.
func (*Fixture) DefaultTemplates() naming.Templates {
	return naming.Templates{Folder: "{Shelf}", Levels: []string{"{Box}"}, Aside: "Spare boxes", File: "{Shelf} - {Thing} - {Edition Tag} {Target}"}
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
func (f *Fixture) Timings(_ context.Context, _ storage.Reader[storage.Library], ws works.Among[works.WorkID]) (map[works.ItemID]targets.Timing, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[works.ItemID]targets.Timing{}
	for _, w := range ws.IDs() {
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

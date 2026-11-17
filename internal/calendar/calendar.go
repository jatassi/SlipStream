// Package calendar answers what comes out in a span of days and whether
// the library has it: each Content Type's dated Items, collapsed into rows
// with their states, and a sentence that leads with what needs attention.
package calendar

import (
	"cmp"
	"context"
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Facet is a Content Type's Calendar capability, embedded in
// contenttype.Library. A type without dates embeds None.
type Facet interface {
	// Dated answers each Item with a dated event on a full day in s, one
	// per Item and kind, through an index on the type's own date column.
	// Kinds are the type's Description.Dated keys.
	Dated(ctx context.Context, r storage.Reader[storage.Library], s Span) ([]Dated, error)
}

type Dated struct {
	Work works.WorkID
	Item works.ItemID
	Kind works.DatedKey
	Day  Day
}

// None is the documented empty answer: the type dates nothing.
type None struct{}

func (None) Dated(context.Context, storage.Reader[storage.Library], Span) ([]Dated, error) {
	return nil, nil
}

type Config struct {
	Library *storage.DB[storage.Library]
	// Types is every registered Content Type's facet, off ones included.
	Types map[works.TypeID]Facet
	Works *works.Works
	Files *files.Files
	// Today is the time now in the Controller's zone.
	Today func() time.Time
}

type Query struct {
	Span Span
	// Type is zero for every type that is on.
	Type works.TypeID
	// Kinds is nil for every kind.
	Kinds       []works.DatedKey
	Unmonitored bool
}

var (
	ErrUnknownType = errors.New("no Content Type has that id")
	ErrTypeOff     = errors.New("that Content Type is turned off")
	ErrUnknownKind = errors.New("no Content Type dates that kind")
)

// TypeOffError is ErrTypeOff for the type Name names, such as Movies.
type TypeOffError struct{ Name string }

func (e *TypeOffError) Error() string { return e.Name + " is turned off" }

func (*TypeOffError) Unwrap() error { return ErrTypeOff }

// UnknownKindError is ErrUnknownKind for Key.
type UnknownKindError struct{ Key works.DatedKey }

func (e *UnknownKindError) Error() string { return "no Content Type dates kind " + string(e.Key) }

func (*UnknownKindError) Unwrap() error { return ErrUnknownKind }

type Answer struct {
	// Today and NextDayAt are the Controller's: NextDayAt is when its next
	// day starts.
	Today     Day
	NextDayAt time.Time
	// Sentence counts monitored Items only. It is nil when none in the span
	// needs attention and none comes out in the next 7 days.
	Sentence *Sentence
	// Days are only the days with rows, in order.
	Days []DayRows
}

type DayRows struct {
	Day  Day
	Rows []Row
}

// Row is a Work's Items of one level, kind and day, monitored or not.
type Row struct {
	Work   works.WorkID
	Type   works.TypeID
	Label  string
	Poster artwork.Ref
	Kind   works.DatedKind
	Items  []ItemRef
	// ItemsWords is the Items in words with their level, such as "Season
	// 2 · Episodes 1 to 10", empty for one Item without a level.
	ItemsWords string
	// State is the most urgent state of its Items that have a cell, zero
	// when none has one. Met and Of count those Items.
	State     files.State
	Met, Of   int
	Monitored bool
}

type ItemRef struct {
	ID works.ItemID
	// Label is "Episode 1 · Hello, Ms. Cobel", or "Episode 1" without its
	// own title.
	Label string
}

type Sentence struct {
	// NeedsYou is whether it names Items that are out and still missing.
	NeedsYou bool
	Words    string
}

type Calendar struct{ cfg Config }

func New(cfg Config) *Calendar { return &Calendar{cfg: cfg} }

// week is how far ahead the sentence looks when nothing needs attention.
const week = 7

// Read answers q in one snapshot. It refuses an unknown type, a type that
// is off and an unknown kind, and never asks an off type's facet.
func (c *Calendar) Read(ctx context.Context, _ identity.Admin, q Query) (Answer, error) {
	now := c.cfg.Today()
	today := DayOf(now)
	out := Answer{Today: today, NextDayAt: today.Add(1).midnight(now.Location())}
	ahead := Span{from: today, to: today.Add(week)}
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		asked, err := c.asked(ctx, r, q)
		if err != nil {
			return err
		}
		dated, err := datedIn(ctx, r, asked, q.Span)
		if err != nil {
			return err
		}
		soon := slices.DeleteFunc(slices.Clone(dated), func(d Dated) bool { return !ahead.Contains(d.Day) })
		if !q.Span.covers(ahead) {
			soon, err = datedIn(ctx, r, asked, ahead)
			if err != nil {
				return err
			}
		}
		of, err := c.worksOf(ctx, r, slices.Concat(dated, soon))
		if err != nil {
			return err
		}
		es := kept(dated, of, q.Unmonitored)
		out.Days = collapse(es)
		out.Sentence = sentenceOf(today, es, kept(soon, of, q.Unmonitored))
		return nil
	})
	return out, err
}

// asking is one type a Read asks, with the kinds it keeps.
type asking struct {
	facet Facet
	kinds []works.DatedKey
}

func (c *Calendar) asked(ctx context.Context, r storage.Reader[storage.Library], q Query) ([]asking, error) {
	if err := c.refuse(q); err != nil {
		return nil, err
	}
	var out []asking
	for _, t := range slices.Sorted(maps.Keys(c.cfg.Types)) {
		if q.Type != "" && t != q.Type {
			continue
		}
		on, err := works.Enabled(ctx, r, t)
		if err != nil {
			return nil, err
		}
		d, _ := c.cfg.Works.Describe(t)
		if !on && q.Type != "" {
			return nil, &TypeOffError{Name: d.Name}
		}
		var kinds []works.DatedKey
		for _, k := range d.Dated {
			if q.Kinds == nil || slices.Contains(q.Kinds, k.Key) {
				kinds = append(kinds, k.Key)
			}
		}
		if on && len(kinds) > 0 {
			out = append(out, asking{facet: c.cfg.Types[t], kinds: kinds})
		}
	}
	return out, nil
}

// refuse answers an unknown type, or a kind no type declares.
func (c *Calendar) refuse(q Query) error {
	if _, ok := c.cfg.Types[q.Type]; q.Type != "" && !ok {
		return ErrUnknownType
	}
	declared := map[works.DatedKey]bool{}
	for t := range c.cfg.Types {
		d, _ := c.cfg.Works.Describe(t)
		for _, k := range d.Dated {
			declared[k.Key] = true
		}
	}
	for _, k := range q.Kinds {
		if !declared[k] {
			return &UnknownKindError{Key: k}
		}
	}
	return nil
}

func datedIn(ctx context.Context, r storage.Reader[storage.Library], asked []asking, s Span) ([]Dated, error) {
	var out []Dated
	for _, a := range asked {
		ds, err := a.facet.Dated(ctx, r, s)
		if err != nil {
			return nil, err
		}
		for _, d := range ds {
			if slices.Contains(a.kinds, d.Kind) {
				out = append(out, d)
			}
		}
	}
	return out, nil
}

// work is what a row reads of a dated Work.
type work struct {
	row  works.Row
	rank int
	desc works.Description
	// levels is each level's label, items each Item's state and its index
	// in tree order, and places each Item's place in its level.
	levels map[works.LevelID]string
	items  map[works.ItemID]placed
	places map[works.ItemID]int
}

type placed struct {
	files.ItemState
	index int
}

// worksOf reads the Works dated names, and only those.
func (c *Calendar) worksOf(ctx context.Context, r storage.Reader[storage.Library], dated []Dated) (map[works.WorkID]*work, error) {
	seen := map[works.WorkID]bool{}
	var ids []works.WorkID
	for _, d := range dated {
		if !seen[d.Work] {
			seen[d.Work] = true
			ids = append(ids, d.Work)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	states, err := c.cfg.Files.ItemStates(ctx, r, works.Only(ids))
	if err != nil {
		return nil, err
	}
	rows, err := c.cfg.Works.RowsIn(ctx, r, works.Scope{Works: ids})
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID]*work, len(rows))
	for i, row := range rows {
		w, _ := c.cfg.Works.Describe(row.ContentType)
		out[row.ID] = newWork(row, i, w, states[row.ID])
	}
	return out, nil
}

// newWork is row's work, rank its place in the Library's order.
func newWork(row works.Row, rank int, desc works.Description, st files.WorkStates) *work {
	w := &work{row: row, rank: rank, desc: desc, levels: map[works.LevelID]string{}, items: make(map[works.ItemID]placed, len(st.Items))}
	for _, l := range st.Levels {
		w.levels[l.ID] = l.Label
	}
	bare := make([]works.Item, len(st.Items))
	for j, it := range st.Items {
		w.items[it.Item.ID] = placed{ItemState: it, index: j}
		bare[j] = it.Item
	}
	w.places = works.Places(bare)
	return w
}

// entry is a Dated the answer keeps, with its Item and its Work.
type entry struct {
	Dated
	item placed
	work *work
}

// kept is dated's live Items, and unmonitored ones only when asked.
func kept(dated []Dated, of map[works.WorkID]*work, unmonitored bool) []entry {
	var out []entry
	for _, d := range dated {
		w := of[d.Work]
		it := w.items[d.Item]
		if it.Item.Live() && (it.Item.Monitored || unmonitored) {
			out = append(out, entry{Dated: d, item: it, work: w})
		}
	}
	slices.SortFunc(out, func(a, b entry) int {
		return cmp.Or(a.Day.Compare(b.Day), cmp.Compare(a.work.rank, b.work.rank), cmp.Compare(a.item.index, b.item.index))
	})
	return out
}

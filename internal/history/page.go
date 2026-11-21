package history

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/history/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Before is a page cursor: the entries with a smaller id. Zero is the
// newest page.
type Before EventID

// Filter is what History lists. Each zero field lists everything.
type Filter struct {
	// Type is zero for every Content Type, Health included.
	Type works.TypeID
	// Groups is nil for all five.
	Groups []entry.Group
	// Kinds each narrow their own group, which Groups must hold.
	Kinds []entry.Kind
	Since time.Time
	Work  works.WorkID
}

// UnknownKindError is a kind no catalog row has.
type UnknownKindError struct{ Kind entry.Kind }

func (e *UnknownKindError) Error() string { return fmt.Sprintf("history: unknown kind %s", e.Kind) }

// KindOffGroupError is a kind whose group the Filter's Groups leave out.
type KindOffGroupError struct{ Kind entry.Spec }

func (e *KindOffGroupError) Error() string {
	return fmt.Sprintf("history: kind %s is in group %s, which isn't asked for", e.Kind.Kind, e.Kind.Group)
}

// ErrGone is an entry that was cleared, pruned or never recorded.
var ErrGone = errors.New("history: entry gone")

// Entry is one History entry in words.
type Entry struct {
	ID   EventID
	Spec entry.Spec
	At   time.Time
	// Work is zero for an entry about no Work or many.
	Work   works.WorkID
	Grab   storage.UUID
	Import storage.UUID
	// Health is a Health entry's item keys, "<category>.<key>".
	Health  []string
	Message entry.Message
}

type Page struct {
	Entries []Entry
	// Next is zero on the last page.
	Next Before
}

// narrowed is the kinds f lists, in catalog order, and whether that is
// every kind of every group, so no kind needs naming.
func (f Filter) narrowed(c *entry.Catalog) (kinds []entry.Kind, every bool, err error) {
	groups := f.Groups
	if groups == nil {
		groups = entry.Groups()
	}
	named := map[entry.Group]bool{}
	for _, k := range f.Kinds {
		spec, ok := c.Spec(k)
		switch {
		case !ok:
			return nil, false, &UnknownKindError{Kind: k}
		case !slices.Contains(groups, spec.Group):
			return nil, false, &KindOffGroupError{Kind: spec}
		}
		named[spec.Group] = true
	}
	for _, spec := range c.Specs() {
		if slices.Contains(groups, spec.Group) && (!named[spec.Group] || slices.Contains(f.Kinds, spec.Kind)) {
			kinds = append(kinds, spec.Kind)
		}
	}
	return kinds, len(f.Kinds) == 0 && len(groups) == len(entry.Groups()), nil
}

// Page is up to limit entries f lists, newest first, below b. Each read is
// an index range of at most limit+1 ids: one per kind when kinds or a
// Content Type narrow it, merged here.
func (s *Service) Page(ctx context.Context, _ identity.Admin, f Filter, b Before, limit int) (Page, error) {
	kinds, every, err := f.narrowed(s.cfg.Catalog)
	if err != nil {
		return Page{}, err
	}
	var out Page
	err = s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		floor, err := floorOf(ctx, r, f.Since)
		if err != nil || floor == 0 {
			return err
		}
		ids, err := pageIDs(ctx, db.New(r), f, kinds, every, span{floor: floor, before: beforeOf(b), limit: limit + 1})
		if err != nil {
			return err
		}
		if len(ids) > limit {
			ids, out.Next = ids[:limit], Before(ids[limit-1])
		}
		out.Entries, err = s.entries(ctx, r, ids, time.UTC)
		return err
	})
	return out, err
}

// span is one page read's id range and size.
type span struct {
	floor, before EventID
	limit         int
}

func beforeOf(b Before) EventID {
	if b == 0 {
		return math.MaxInt64
	}
	return EventID(b)
}

// floorOf is the lowest id History lists: above the Clear floor, and from
// since when it is set. Zero: nothing is that recent.
func floorOf(ctx context.Context, r storage.Reader[file], since time.Time) (EventID, error) {
	q := db.New(r)
	cleared, err := q.GetCleared(ctx)
	if err != nil || since.IsZero() {
		return cleared + 1, err
	}
	first, err := q.FirstIDFrom(ctx, storage.FromTime(since))
	if errors.Is(err, storage.ErrNoRows) {
		return 0, nil
	}
	return max(cleared+1, first), err
}

func pageIDs(ctx context.Context, q *db.Queries, f Filter, kinds []entry.Kind, every bool, sp span) ([]EventID, error) {
	n := int64(sp.limit)
	switch {
	case !f.Work.IsZero():
		var t any
		if f.Type != "" {
			t = string(f.Type)
		}
		// A JSON list: sqlc numbers the parameters after a slice wrongly.
		listed, err := json.Marshal(kinds)
		if err != nil {
			return nil, fmt.Errorf("history: listing kinds: %w", err)
		}
		return q.ListWorkIDs(ctx, db.ListWorkIDsParams{WorkID: f.Work, Floor: sp.floor, Before: sp.before, Kinds: string(listed), ContentType: t, Limit: n})
	case f.Type == "" && every:
		return q.ListIDs(ctx, db.ListIDsParams{Floor: sp.floor, Before: sp.before, Limit: n})
	}
	var all []EventID
	for _, k := range kinds {
		var ids []EventID
		var err error
		if f.Type == "" {
			ids, err = q.ListKindIDs(ctx, db.ListKindIDsParams{Kind: k, Floor: sp.floor, Before: sp.before, Limit: n})
		} else if !IsHealth(k) {
			ids, err = q.ListTypeKindIDs(ctx, db.ListTypeKindIDsParams{ContentType: &f.Type, Kind: k, Floor: sp.floor, Before: sp.before, Limit: n})
		}
		if err != nil {
			return nil, err
		}
		all = append(all, ids...)
	}
	return newest(all, sp.limit), nil
}

// newest is the first n of ids, newest first.
func newest(ids []EventID, n int) []EventID {
	slices.SortFunc(ids, func(a, b EventID) int { return cmp.Compare(b, a) })
	return ids[:min(len(ids), n)]
}

// entries is ids in words, newest first, rendered in zone.
func (s *Service) entries(ctx context.Context, r storage.Reader[file], ids []EventID, zone *time.Location) ([]Entry, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := db.New(r)
	rows, err := q.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	items, err := Items(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	c := entry.Context{Now: time.Now(), Zone: zone}
	out := make([]Entry, len(rows))
	for i, row := range rows {
		if out[i], err = s.entryOf(row, items[row.ID], c); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) entryOf(row db.HistoryEvent, items []Item, c entry.Context) (Entry, error) {
	spec, _ := s.cfg.Catalog.Spec(row.Kind)
	e := Entry{ID: row.ID, Spec: spec, At: row.At.Time(), Work: row.WorkID, Grab: row.GrabID, Import: row.ImportID}
	subject := []byte(row.Subject)
	if IsHealth(row.Kind) {
		for _, it := range items {
			e.Health = append(e.Health, it.Key)
		}
		var err error
		if subject, err = json.Marshal(HealthFacts{Items: items}); err != nil {
			return Entry{}, fmt.Errorf("history: entry %d's Health items: %w", row.ID, err)
		}
	}
	m, err := s.cfg.Catalog.Render(row.Kind, subject, c)
	e.Message = m
	return e, err
}

// One is entry id in full, its clock times in zone. ErrGone: it was
// cleared, pruned or never recorded.
func (s *Service) One(ctx context.Context, _ identity.Admin, id EventID, zone *time.Location) (Entry, error) {
	var out Entry
	err := s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		cleared, err := db.New(r).GetCleared(ctx)
		if err != nil {
			return err
		}
		if id <= cleared {
			return ErrGone
		}
		es, err := s.entries(ctx, r, []EventID{id}, zone)
		if err != nil {
			return err
		}
		if len(es) == 0 {
			return ErrGone
		}
		out = es[0]
		return nil
	})
	return out, err
}

// Read runs f in one Read of the Library, for the API that composes an
// entry's links from their owners.
func (s *Service) Read(ctx context.Context, f func(ctx context.Context, r storage.Reader[storage.Library]) error) error {
	return s.cfg.Library.Read(ctx, f)
}

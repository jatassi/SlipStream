package works

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// Batch is the most ids one read binds, well under SQLite's 32,766
// variables. A read of more runs once per Batch inside one snapshot.
const Batch = 2000

// Bare is a Work's summary and its Items in tree order, without fields.
type Bare struct {
	Summary
	Items []Item
}

func BareOf(t Tree) Bare { return Bare{Summary: t.Summary, Items: t.Items} }

// Row is what the Library lists of one Work.
type Row struct {
	Bare
	Monitored Monitoring
	// Poster is the Work's art of its type's Art.Poster kind, zero when it
	// has none.
	Poster artwork.Ref
	// Rating is the first of its type's Ratings the Work holds, nil when
	// it holds none.
	Rating *StoredRating
	// Values are its type's listed fields, in declared order.
	Values []FieldValue
}

// Monitoring is how many of a Work's Items are monitored.
type Monitoring uint8

const (
	MonitoredNone Monitoring = iota + 1
	MonitoredSome
	MonitoredAll
)

var monitoringNames = map[Monitoring]string{MonitoredNone: "none", MonitoredSome: "some", MonitoredAll: "all"}

func (m Monitoring) String() string { return monitoringNames[m] }

func monitoringOf(items []Item) Monitoring {
	on := 0
	for _, it := range items {
		if it.Monitored {
			on++
		}
	}
	switch on {
	case 0:
		return MonitoredNone
	case len(items):
		return MonitoredAll
	}
	return MonitoredSome
}

// Scope is every Work, one Content Type's, or the named ones when Works
// is not nil. An id no Work has is left out.
type Scope struct {
	Type  *TypeID
	Works []WorkID
}

// RowsIn reads the rows of s in r, by sort label then id.
func (w *Works) RowsIn(ctx context.Context, r storage.Reader[file], s Scope) ([]Row, error) {
	works, err := w.scoped(ctx, r, s)
	if err != nil {
		return nil, err
	}
	return w.rowsOf(ctx, r, works, s.Among(idsOf(works)))
}

// Among is ids, the Works s holds in a snapshot.
func (s Scope) Among(ids []WorkID) Among[WorkID] {
	return Among[WorkID]{ids: ids, every: s.Type == nil && s.Works == nil}
}

func idsOf(works []db.WorksWork) []WorkID {
	out := make([]WorkID, len(works))
	for i, row := range works {
		out[i] = row.ID
	}
	return out
}

func (w *Works) scoped(ctx context.Context, r storage.Reader[file], s Scope) ([]db.WorksWork, error) {
	q := db.New(r)
	switch {
	case s.Works != nil:
		var out []db.WorksWork
		for chunk := range slices.Chunk(s.Works, Batch) {
			rows, err := q.ListWorksByID(ctx, chunk)
			if err != nil {
				return nil, err
			}
			out = append(out, rows...)
		}
		slices.SortFunc(out, func(a, b db.WorksWork) int {
			return cmp.Or(cmp.Compare(a.SortLabel, b.SortLabel), a.ID.Compare(b.ID.UUID))
		})
		return out, nil
	case s.Type != nil:
		if _, ok := w.types[*s.Type]; !ok {
			return nil, ErrNoType
		}
		return q.ListWorksOfType(ctx, *s.Type)
	}
	return q.ListWorks(ctx)
}

// rowsOf reads the rows of works, which among names, in their order.
func (w *Works) rowsOf(ctx context.Context, r storage.Reader[file], works []db.WorksWork, among Among[WorkID]) ([]Row, error) {
	bares, err := baresOf(ctx, r, works, among)
	if err != nil {
		return nil, err
	}
	out := make([]Row, len(bares))
	at := make(map[WorkID]*Row, len(bares))
	byType := map[TypeID][]WorkID{}
	for i, b := range bares {
		out[i] = Row{Bare: b, Monitored: monitoringOf(b.Items)}
		at[b.ID] = &out[i]
		byType[b.ContentType] = append(byType[b.ContentType], b.ID)
	}
	q := db.New(r)
	ratings, err := Read(among, func() ([]db.ListEveryWorkRatingRow, error) { return q.ListEveryWorkRating(ctx) }, func(ids []WorkID) ([]db.ListEveryWorkRatingRow, error) {
		rows, err := q.ListWorkRatingsOfWorks(ctx, ids)
		return convert(rows, func(r db.ListWorkRatingsOfWorksRow) db.ListEveryWorkRatingRow { return db.ListEveryWorkRatingRow(r) }), err
	})
	if err != nil {
		return nil, err
	}
	for _, typ := range slices.Sorted(maps.Keys(byType)) {
		reg, ofType := w.types[typ], Like(among, byType[typ])
		if err := listValues(ctx, r, reg, ofType, at); err != nil {
			return nil, err
		}
		if reg.desc.Art.Poster == "" {
			continue
		}
		kind := reg.desc.Art.Poster
		art, err := Read(ofType, func() ([]db.ListEveryWorkArtRow, error) { return q.ListEveryWorkArt(ctx, kind) }, func(ids []WorkID) ([]db.ListEveryWorkArtRow, error) {
			rows, err := q.ListWorkArtOfWorks(ctx, db.ListWorkArtOfWorksParams{Kind: kind, WorkIds: ids})
			return convert(rows, func(r db.ListWorkArtOfWorksRow) db.ListEveryWorkArtRow { return db.ListEveryWorkArtRow(r) }), err
		})
		if err != nil {
			return nil, err
		}
		shape := posterShape(reg.desc)
		for _, a := range art {
			if row := at[a.WorkID]; row.ContentType == typ {
				row.Poster = artwork.Ref{Source: artwork.SourceID(a.Source), Shape: shape, Path: a.Path}
			}
		}
	}
	for _, rt := range ratings {
		row := at[rt.WorkID]
		order := w.types[row.ContentType].desc.Ratings
		rank := slices.Index(order, RatingSource(rt.Source))
		if rank < 0 || row.Rating != nil && rank > slices.Index(order, row.Rating.Source) {
			continue
		}
		row.Rating = &StoredRating{Source: RatingSource(rt.Source), Value: rt.Value, Votes: rt.Votes, FetchedAt: rt.FetchedAt}
	}
	return out, nil
}

func convert[F, T any](rows []F, to func(F) T) []T {
	out := make([]T, len(rows))
	for i, r := range rows {
		out[i] = to(r)
	}
	return out
}

func posterShape(d Description) artwork.Shape {
	i := slices.IndexFunc(d.Artwork, func(k artwork.Kind) bool { return !k.Item && k.Key == d.Art.Poster })
	return d.Artwork[i].Shape
}

// listValues puts the listed fields reg's catalog answers for ws on their
// rows, refusing a field the type doesn't list.
func listValues(ctx context.Context, r storage.Reader[file], reg registered, ws Among[WorkID], at map[WorkID]*Row) error {
	var listed []FieldSpec
	for _, f := range reg.specs(TierWork) {
		if f.List != 0 {
			listed = append(listed, f)
		}
	}
	if len(listed) == 0 {
		return nil
	}
	values, err := reg.catalog.ListFields(ctx, r, ws)
	if err != nil {
		return fmt.Errorf("content type %s: listed fields: %w", reg.desc.Name, err)
	}
	for id, vs := range values {
		row, ok := at[id]
		if !ok {
			return fmt.Errorf("content type %s: listed fields name work %s, which it wasn't asked for", reg.desc.Name, id)
		}
		if row.Values, err = ordered(listed, vs); err != nil {
			return fmt.Errorf("content type %s: listed fields of work %s: %w", reg.desc.Name, id, err)
		}
	}
	return nil
}

// BareIn reads Work id's bare tree inside a caller's read, ErrNoWork when
// no Work has it.
func BareIn(ctx context.Context, r storage.Reader[file], id WorkID) (Bare, error) {
	rows, err := db.New(r).ListWorksByID(ctx, []WorkID{id})
	if err != nil {
		return Bare{}, err
	}
	if len(rows) == 0 {
		return Bare{}, ErrNoWork
	}
	bares, err := baresOf(ctx, r, rows, Only([]WorkID{id}))
	if err != nil {
		return Bare{}, err
	}
	return bares[0], nil
}

// baresOf reads the Items of works, which among names, each Work's in
// tree order. Only a Work of many Items has its levels read: they order
// its Items.
func baresOf(ctx context.Context, r storage.Reader[file], works []db.WorksWork, among Among[WorkID]) ([]Bare, error) {
	q := db.New(r)
	items, err := Read(among, func() ([]db.WorksItem, error) { return q.ListEveryItem(ctx) }, func(ids []WorkID) ([]db.WorksItem, error) {
		return q.ListItemsOfWorks(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	itemsOf := make(map[WorkID][]db.WorksItem, len(works))
	var many []WorkID
	for _, it := range items {
		itemsOf[it.WorkID] = append(itemsOf[it.WorkID], it)
		if len(itemsOf[it.WorkID]) == 2 {
			many = append(many, it.WorkID)
		}
	}
	var levels []db.WorksLevel
	switch {
	case len(many) == 0:
	case among.Every():
		levels, err = q.ListEveryLevel(ctx)
	default:
		levels, err = Read(Only(many), nil, func(ids []WorkID) ([]db.WorksLevel, error) { return q.ListLevelsOfWorks(ctx, ids) })
	}
	if err != nil {
		return nil, err
	}
	levelsOf := map[WorkID][]db.WorksLevel{}
	for _, l := range levels {
		levelsOf[l.WorkID] = append(levelsOf[l.WorkID], l)
	}
	out := make([]Bare, len(works))
	for i, row := range works {
		of := itemsOf[row.ID]
		if len(of) > 1 {
			treeOrder(readOrder(levelsOf[row.ID], of))
		}
		out[i] = Bare{Summary: Summary(row), Items: make([]Item, len(of))}
		for j, it := range of {
			out[i].Items[j] = bare(it)
		}
	}
	return out, nil
}

// readOrder sorts one Work's levels and Items, read in any order, the way
// treeOrder takes them.
func readOrder(levels []db.WorksLevel, items []db.WorksItem) ([]db.WorksLevel, []db.WorksItem) {
	slices.SortFunc(levels, func(a, b db.WorksLevel) int {
		return cmp.Or(cmp.Compare(a.Depth, b.Depth), placed(a.WithdrawnAt, a.Position, a.ID.UUID, b.WithdrawnAt, b.Position, b.ID.UUID))
	})
	slices.SortFunc(items, siblings)
	return levels, items
}

func siblings(a, b db.WorksItem) int {
	return placed(a.WithdrawnAt, a.Position, a.ID.UUID, b.WithdrawnAt, b.Position, b.ID.UUID)
}

// placed orders siblings: present before withdrawn, then by position and id.
func placed(aw storage.Time, ap int64, aid storage.UUID, bw storage.Time, bp int64, bid storage.UUID) int {
	return cmp.Or(cmp.Compare(gone(aw), gone(bw)), cmp.Compare(ap, bp), aid.Compare(bid))
}

func gone(at storage.Time) int {
	if at.IsZero() {
		return 0
	}
	return 1
}

// treeOrder sorts one Work's levels, read by depth, by their parent's
// order within each depth, and its Items by their level's order, each
// keeping the order read otherwise. It answers each level's rank.
func treeOrder(levels []db.WorksLevel, items []db.WorksItem) map[LevelID]int {
	rank := map[LevelID]int{}
	for rest := levels; len(rest) > 0; {
		n := 1 + slices.IndexFunc(rest[1:], func(l db.WorksLevel) bool { return l.Depth != rest[0].Depth })
		if n == 0 {
			n = len(rest)
		}
		depth := rest[:n]
		rest = rest[n:]
		slices.SortStableFunc(depth, func(a, b db.WorksLevel) int { return cmp.Compare(rank[a.ParentID], rank[b.ParentID]) })
		for _, l := range depth {
			rank[l.ID] = len(rank) + 1
		}
	}
	slices.SortStableFunc(items, func(a, b db.WorksItem) int { return cmp.Compare(rank[a.LevelID], rank[b.LevelID]) })
	return rank
}

// Links is t's outbound links, from its type's Links.
func (w *Works) Links(t Tree) []WebLink {
	return w.types[t.ContentType].catalog.Links(t.Key, t.Fields)
}

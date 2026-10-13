package works

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

type ContentType struct {
	ID TypeID
	Description
	Enabled bool
}

type Summary struct {
	ID          WorkID
	ContentType TypeID
	Key         Key
	Label       string
	SortLabel   string
	AddedAt     storage.Time
	RefreshedAt storage.Time
}

// Tree is a Work with its levels, by depth, then their parent's order,
// then position, and its Items, by their level's order, then position.
// Withdrawn nodes stay in it, after their live siblings: a withdrawn
// node keeps the position it last had, which a live sibling may now hold.
type Tree struct {
	Summary
	Fields []FieldValue
	Levels []Level
	Items  []Item
}

type Level struct {
	ID LevelID
	// ParentID is zero under the Work.
	ParentID    LevelID
	Depth       int
	Key         Key
	Label       string
	Position    int
	WithdrawnAt storage.Time
	Fields      []FieldValue
}

type Item struct {
	ID ItemID
	// LevelID is zero when the type declares no levels.
	LevelID     LevelID
	Key         Key
	Label       string
	Position    int
	WithdrawnAt storage.Time
	Fields      []FieldValue
}

// Types lists every registered Content Type in id order, off ones
// included.
func (w *Works) Types(ctx context.Context) ([]ContentType, error) {
	off, err := db.New(w.cfg.Library).ListDisabledTypes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ContentType, len(w.ids))
	for i, id := range w.ids {
		out[i] = ContentType{ID: id, Description: w.types[id].desc, Enabled: !slices.Contains(off, id)}
	}
	return out, nil
}

// SetEnabled turns t on or off. Turning off a type that holds a Work is
// ErrTypeInUse. A change publishes ContentTypesChanged; a repeat changes
// nothing.
func (w *Works) SetEnabled(ctx context.Context, _ identity.Admin, t TypeID, on bool) (ContentType, error) {
	reg, ok := w.types[t]
	if !ok {
		return ContentType{}, ErrNoType
	}
	var changed int64
	err := w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		var err error
		if on {
			changed, err = q.EnableType(ctx, t)
			return err
		}
		holds, err := q.HoldsType(ctx, t)
		switch {
		case err != nil:
			return err
		case holds:
			return ErrTypeInUse
		}
		changed, err = q.DisableType(ctx, db.DisableTypeParams{ContentType: t, DisabledAt: storage.Now()})
		return err
	})
	if err != nil {
		return ContentType{}, err
	}
	if changed > 0 {
		w.cfg.TypesChanged.Publish(ctx, ContentTypesChanged{ContentType: string(t)})
	}
	return ContentType{ID: t, Description: reg.desc, Enabled: on}, nil
}

// Search asks t's metadata source for query. Each candidate already in
// the library carries its WorkID.
func (w *Works) Search(ctx context.Context, _ identity.Admin, t TypeID, query string) ([]Candidate, error) {
	reg, err := w.open(ctx, t)
	if err != nil {
		return nil, err
	}
	fetchCtx, cancel := context.WithTimeout(ctx, w.cfg.Timings.Fetch)
	found, err := reg.catalog.Search(fetchCtx, query)
	cancel()
	if err != nil {
		return nil, sourced(t, "search", err)
	}
	if len(found) > maxCandidates {
		return nil, fmt.Errorf("content type %s: search returned %d candidates, more than %d", t, len(found), maxCandidates)
	}
	work := reg.specs(TierWork)
	err = w.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		for i, c := range found {
			fields, err := ordered(work, c.Fields)
			if err != nil {
				return fmt.Errorf("content type %s: candidate %s: %w", t, c.Key, err)
			}
			found[i].Fields = fields
			row, err := q.GetWorkByKey(ctx, db.GetWorkByKeyParams{ContentType: t, Key: c.Key})
			switch {
			case errors.Is(err, storage.ErrNoRows):
			case err != nil:
				return err
			default:
				found[i].WorkID = row.ID
			}
		}
		return nil
	})
	return found, err
}

const maxCandidates = 25

// Add puts the Work t's source knows by key in the library with its whole
// tree, or returns the Work already there with created false. Two racing
// adds converge on one Work.
func (w *Works) Add(ctx context.Context, by identity.Admin, t TypeID, key Key) (tree Tree, created bool, err error) {
	reg, err := w.open(ctx, t)
	if err != nil {
		return Tree{}, false, err
	}
	row, err := db.New(w.cfg.Library).GetWorkByKey(ctx, db.GetWorkByKeyParams{ContentType: t, Key: key})
	if err == nil {
		tree, err = w.Get(ctx, by, row.ID)
		return tree, false, err
	}
	if !errors.Is(err, storage.ErrNoRows) {
		return Tree{}, false, err
	}
	p, err := w.fetch(ctx, t, reg, key)
	if err != nil {
		return Tree{}, false, err
	}
	id := WorkID{UUID: storage.NewUUID()}
	err = w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		on, err := Enabled(ctx, tx, t)
		switch {
		case err != nil:
			return err
		case !on:
			return ErrTypeOff
		}
		row, err := q.GetWorkByKey(ctx, db.GetWorkByKeyParams{ContentType: t, Key: key})
		if err == nil {
			id = row.ID
			return nil
		}
		if !errors.Is(err, storage.ErrNoRows) {
			return err
		}
		now := storage.Now()
		created = true
		if err := q.InsertWork(ctx, db.InsertWorkParams{ID: id, ContentType: t, Key: key, Label: p.Label, SortLabel: p.SortLabel, AddedAt: now, RefreshedAt: now}); err != nil {
			return err
		}
		return converge(ctx, tx, reg.catalog, id, p, now)
	})
	if err != nil {
		return Tree{}, false, err
	}
	if created {
		w.cfg.Changed.Publish(ctx, WorkChanged{WorkID: id.String()})
	}
	tree, err = w.Get(ctx, by, id)
	return tree, created, err
}

// Refresh converges a Work on what its source has now, keeping every id.
// A source that no longer has the Work is ErrGoneUpstream and changes
// nothing.
func (w *Works) Refresh(ctx context.Context, by identity.Admin, id WorkID) (Tree, error) {
	row, err := db.New(w.cfg.Library).GetWork(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Tree{}, ErrNoWork
	}
	if err != nil {
		return Tree{}, err
	}
	reg := w.types[row.ContentType]
	p, err := w.fetch(ctx, row.ContentType, reg, row.Key)
	if errors.Is(err, ErrNotFound) {
		return Tree{}, ErrGoneUpstream
	}
	if err != nil {
		return Tree{}, err
	}
	err = w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return converge(ctx, tx, reg.catalog, id, p, storage.Now())
	})
	if err != nil {
		return Tree{}, err
	}
	w.cfg.Changed.Publish(ctx, WorkChanged{WorkID: id.String()})
	return w.Get(ctx, by, id)
}

func (w *Works) open(ctx context.Context, t TypeID) (registered, error) {
	reg, ok := w.types[t]
	if !ok {
		return registered{}, ErrNoType
	}
	on, err := Enabled(ctx, w.cfg.Library, t)
	switch {
	case err != nil:
		return registered{}, err
	case !on:
		return registered{}, ErrTypeOff
	}
	return reg, nil
}

func (w *Works) fetch(ctx context.Context, t TypeID, reg registered, key Key) (parsed, error) {
	ctx, cancel := context.WithTimeout(ctx, w.cfg.Timings.Fetch)
	defer cancel()
	f, err := reg.catalog.Fetch(ctx, key)
	if err != nil {
		return parsed{}, sourced(t, "fetch "+string(key), err)
	}
	p, err := parseFetched(reg.desc, key, f)
	if err != nil {
		w.cfg.Log.Warn("metadata invalid", "contentType", string(t), "key", string(key), "err", err.Error())
	}
	return p, err
}

func sourced(t TypeID, op string, err error) error {
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrUnavailable) {
		return err
	}
	return fmt.Errorf("content type %s: %s: %w", t, op, err)
}

func converge(ctx context.Context, tx *storage.Tx[file], c Catalog, id WorkID, p parsed, now storage.Time) error {
	q := db.New(tx)
	levels, err := q.ListLevels(ctx, id)
	if err != nil {
		return err
	}
	items, err := q.ListItems(ctx, id)
	if err != nil {
		return err
	}
	levelIDs := map[Key]LevelID{}
	levelKeys := map[LevelID]Key{}
	var stored nodes
	for _, l := range levels {
		levelIDs[l.Key], levelKeys[l.ID] = l.ID, l.Key
	}
	for _, l := range levels {
		stored.Levels = append(stored.Levels, level{node: node{Key: l.Key, Parent: levelKeys[l.ParentID], Label: l.Label, Position: int(l.Position), WithdrawnAt: l.WithdrawnAt}, Depth: int(l.Depth)})
	}
	itemIDs := map[Key]ItemID{}
	for _, it := range items {
		itemIDs[it.Key] = it.ID
		stored.Items = append(stored.Items, node{Key: it.Key, Parent: levelKeys[it.LevelID], Label: it.Label, Position: int(it.Position), WithdrawnAt: it.WithdrawnAt})
	}
	pl := planMerge(stored, p, now)
	if err := q.UpdateWork(ctx, db.UpdateWorkParams{Label: pl.Label, SortLabel: pl.SortLabel, RefreshedAt: pl.RefreshedAt, ID: id}); err != nil {
		return err
	}
	for _, c := range pl.Levels {
		l := c.Node
		if c.Insert {
			levelIDs[l.Key] = LevelID{UUID: storage.NewUUID()}
			err = q.InsertLevel(ctx, db.InsertLevelParams{ID: levelIDs[l.Key], WorkID: id, ParentID: levelIDs[l.Parent], Depth: int64(l.Depth), Key: l.Key, Label: l.Label, Position: int64(l.Position), WithdrawnAt: l.WithdrawnAt})
		} else {
			err = q.UpdateLevel(ctx, db.UpdateLevelParams{ParentID: levelIDs[l.Parent], Depth: int64(l.Depth), Label: l.Label, Position: int64(l.Position), WithdrawnAt: l.WithdrawnAt, ID: levelIDs[l.Key]})
		}
		if err != nil {
			return err
		}
	}
	for _, c := range pl.Items {
		it := c.Node
		if c.Insert {
			itemIDs[it.Key] = ItemID{UUID: storage.NewUUID()}
			err = q.InsertItem(ctx, db.InsertItemParams{ID: itemIDs[it.Key], WorkID: id, LevelID: levelIDs[it.Parent], Key: it.Key, Label: it.Label, Position: int64(it.Position), WithdrawnAt: it.WithdrawnAt})
		} else {
			err = q.UpdateItem(ctx, db.UpdateItemParams{LevelID: levelIDs[it.Parent], Label: it.Label, Position: int64(it.Position), WithdrawnAt: it.WithdrawnAt, ID: itemIDs[it.Key]})
		}
		if err != nil {
			return err
		}
	}
	at := Placed{Work: id, Levels: map[Key]LevelID{}, Items: map[Key]ItemID{}}
	for _, n := range p.Levels {
		at.Levels[n.Key] = levelIDs[n.Key]
	}
	for _, n := range p.Items {
		at.Items[n.Key] = itemIDs[n.Key]
	}
	return c.Store(ctx, tx, at, p.Detail)
}

// List lists Works by sort label, then id: every Work, or t's.
func (w *Works) List(ctx context.Context, _ identity.Admin, t *TypeID) ([]Summary, error) {
	q := db.New(w.cfg.Library)
	var rows []db.WorksWork
	var err error
	if t == nil {
		rows, err = q.ListWorks(ctx)
	} else if _, ok := w.types[*t]; !ok {
		return nil, ErrNoType
	} else {
		rows, err = q.ListWorksOfType(ctx, *t)
	}
	if err != nil {
		return nil, err
	}
	out := make([]Summary, len(rows))
	for i, r := range rows {
		out[i] = Summary(r)
	}
	return out, nil
}

// Get reads one Work's tree and its display fields from one snapshot.
func (w *Works) Get(ctx context.Context, _ identity.Admin, id WorkID) (Tree, error) {
	var tree Tree
	err := w.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		tree, err = w.read(ctx, r, id)
		return err
	})
	return tree, err
}

func (w *Works) read(ctx context.Context, r storage.Reader[file], id WorkID) (Tree, error) {
	q := db.New(r)
	row, err := q.GetWork(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Tree{}, ErrNoWork
	}
	if err != nil {
		return Tree{}, err
	}
	levels, err := q.ListLevels(ctx, id)
	if err != nil {
		return Tree{}, err
	}
	items, err := q.ListItems(ctx, id)
	if err != nil {
		return Tree{}, err
	}
	reg := w.types[row.ContentType]
	fields, err := reg.catalog.Fields(ctx, r, id)
	if err != nil {
		return Tree{}, fmt.Errorf("content type %s: fields of work %s: %w", row.ContentType, id, err)
	}
	t, err := assemble(reg, row, levels, items, fields)
	if err != nil {
		return Tree{}, fmt.Errorf("content type %s: fields of work %s: %w", row.ContentType, id, err)
	}
	return t, nil
}

func assemble(reg registered, row db.WorksWork, levels []db.WorksLevel, items []db.WorksItem, fs FieldSet) (Tree, error) {
	t := Tree{Summary: Summary(row)}
	var err error
	if t.Fields, err = ordered(reg.specs(TierWork), fs.Work); err != nil {
		return Tree{}, fmt.Errorf("the Work: %w", err)
	}
	rank := map[LevelID]int{}
	for len(levels) > 0 {
		n := 1 + slices.IndexFunc(levels[1:], func(l db.WorksLevel) bool { return l.Depth != levels[0].Depth })
		if n == 0 {
			n = len(levels)
		}
		depth := levels[:n]
		levels = levels[n:]
		slices.SortStableFunc(depth, func(a, b db.WorksLevel) int { return cmp.Compare(rank[a.ParentID], rank[b.ParentID]) })
		for _, l := range depth {
			rank[l.ID] = len(rank) + 1
			values, err := ordered(reg.specs(TierLevel(int(l.Depth))), fs.Levels[l.ID])
			if err != nil {
				return Tree{}, fmt.Errorf("level %s: %w", l.Key, err)
			}
			t.Levels = append(t.Levels, Level{ID: l.ID, ParentID: l.ParentID, Depth: int(l.Depth), Key: l.Key, Label: l.Label, Position: int(l.Position), WithdrawnAt: l.WithdrawnAt, Fields: values})
		}
	}
	slices.SortStableFunc(items, func(a, b db.WorksItem) int { return cmp.Compare(rank[a.LevelID], rank[b.LevelID]) })
	held := map[ItemID]bool{}
	for _, it := range items {
		held[it.ID] = true
		values, err := ordered(reg.specs(TierItem), fs.Items[it.ID])
		if err != nil {
			return Tree{}, fmt.Errorf("item %s: %w", it.Key, err)
		}
		t.Items = append(t.Items, Item{ID: it.ID, LevelID: it.LevelID, Key: it.Key, Label: it.Label, Position: int(it.Position), WithdrawnAt: it.WithdrawnAt, Fields: values})
	}
	for id := range fs.Levels {
		if _, ok := rank[id]; !ok {
			return Tree{}, fmt.Errorf("fields name level %s, which is not in the Work", id)
		}
	}
	for id := range fs.Items {
		if !held[id] {
			return Tree{}, fmt.Errorf("fields name item %s, which is not in the Work", id)
		}
	}
	return t, nil
}

func (r registered) specs(t Tier) []FieldSpec {
	var out []FieldSpec
	for _, f := range r.desc.Fields {
		if f.Tier == t {
			out = append(out, f)
		}
	}
	return out
}

func ordered(specs []FieldSpec, values []FieldValue) ([]FieldValue, error) {
	out := make([]FieldValue, 0, len(values))
	for _, s := range specs {
		i := slices.IndexFunc(values, func(v FieldValue) bool { return v.Key == s.Key })
		if i < 0 {
			continue
		}
		if values[i].Kind != s.Kind {
			return nil, fmt.Errorf("field %s is %s; its spec says %s", s.Key, values[i].Kind, s.Kind)
		}
		out = append(out, values[i])
	}
	if len(out) != len(values) {
		return nil, fmt.Errorf("its %d field values name %d declared fields", len(values), len(out))
	}
	return out, nil
}

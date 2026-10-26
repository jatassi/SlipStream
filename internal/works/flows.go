package works

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/role"
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
	// GoneUpstreamAt is when the source last answered that it has no such
	// record, zero once it has it again.
	GoneUpstreamAt storage.Time
	// RatingsReadAt is when the ratings providers last answered.
	RatingsReadAt storage.Time
}

// StoredRating is a Rating as the library holds it.
type StoredRating struct {
	Source    RatingSource
	Value     float64
	Votes     *int64
	FetchedAt storage.Time
}

// Tree is a Work with its levels, by depth, then their parent's order,
// then position, and its Items, by their level's order, then position.
// Withdrawn nodes stay in it, after their live siblings: a withdrawn
// node keeps the position it last had, which a live sibling may now hold.
type Tree struct {
	Summary
	Fields  []FieldValue
	Ratings []StoredRating
	// Artwork is keyed by the artwork.Kind's key.
	Artwork map[string]artwork.Ref
	Levels  []Level
	Items   []Item
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
	Monitored   bool
	Fields      []FieldValue
	Ratings     []StoredRating
	Artwork     map[string]artwork.Ref
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

// Describe is t's description, false when no registered type has id t.
func (w *Works) Describe(t TypeID) (Description, bool) {
	reg, ok := w.types[t]
	return reg.desc, ok
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
	found, err := w.cfg.Source.Search(fetchCtx, t, query)
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

// Adding is how Add creates a Work: its Items monitored when Monitor is
// set, and Also, when set, run in the same Write once the Work exists, so
// another owner's rows for it commit or roll back with it.
type Adding struct {
	Monitor bool
	Also    func(ctx context.Context, tx *storage.Tx[file], w Summary) error
}

// Add puts the Work t's source knows by key in the library with its whole
// tree as a says, or returns the Work already there, unchanged, with
// created false. Two racing adds converge on one Work.
func (w *Works) Add(ctx context.Context, by identity.Admin, t TypeID, key Key, a Adding) (tree Tree, created bool, err error) {
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
	p, err := w.fetch(ctx, t, reg, key, Fetching{Purpose: ForAdd})
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
		created = true
		return insert(ctx, tx, reg, Summary{ID: id, ContentType: t, Key: key, Label: p.Label, SortLabel: p.SortLabel}, p, a)
	})
	if err != nil {
		return Tree{}, false, err
	}
	if created {
		w.cfg.Changed.Publish(ctx, ChangedOf(id))
		if wants := prefetches(reg.desc, p.Artwork); len(wants) > 0 {
			w.cfg.Artwork.Prefetch(wants)
		}
	}
	tree, err = w.Get(ctx, by, id)
	return tree, created, err
}

// prefetches are the images of as whose kind d prefetches, at its size.
func prefetches(d Description, as []Art) []artwork.Want {
	var wants []artwork.Want
	for _, a := range as {
		i := slices.IndexFunc(d.Artwork, func(k artwork.Kind) bool { return k.Key == a.Kind && k.Item == (a.Item != "") })
		if size := d.Artwork[i].Prefetch; size != "" {
			wants = append(wants, artwork.Want{Ref: a.Ref, Size: size})
		}
	}
	return wants
}

var errPreview = errors.New("works: a preview rolls back")

// Preview is the Work Add would make of the record t's source knows by
// key, without its ratings providers: Add's Write, rolled back, so the
// tree has an added Work's shape and nothing commits. Its ids name
// nothing. A key already in the library answers the stored Work, with
// stored true.
func (w *Works) Preview(ctx context.Context, _ identity.Admin, t TypeID, key Key, within func(context.Context, storage.Reader[file], Tree) error) (tree Tree, stored bool, err error) {
	reg, err := w.open(ctx, t)
	if err != nil {
		return Tree{}, false, err
	}
	row, err := db.New(w.cfg.Library).GetWorkByKey(ctx, db.GetWorkByKeyParams{ContentType: t, Key: key})
	if err == nil {
		err = w.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
			if tree, err = w.read(ctx, r, row.ID); err != nil {
				return err
			}
			return within(ctx, r, tree)
		})
		return tree, true, err
	}
	if !errors.Is(err, storage.ErrNoRows) {
		return Tree{}, false, err
	}
	p, err := w.fetch(ctx, t, reg, key, Fetching{Purpose: ForPreview})
	if err != nil {
		return Tree{}, false, err
	}
	err = w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		id := WorkID{UUID: storage.NewUUID()}
		row, err := db.New(tx).GetWorkByKey(ctx, db.GetWorkByKeyParams{ContentType: t, Key: key})
		switch {
		case err == nil:
			id, stored = row.ID, true
		case !errors.Is(err, storage.ErrNoRows):
			return err
		default:
			err = insert(ctx, tx, reg, Summary{ID: id, ContentType: t, Key: key, Label: p.Label, SortLabel: p.SortLabel}, p, Adding{})
			if err != nil {
				return err
			}
		}
		if tree, err = w.read(ctx, tx, id); err != nil {
			return err
		}
		if err := within(ctx, tx, tree); err != nil {
			return err
		}
		return errPreview
	})
	if !errors.Is(err, errPreview) {
		return Tree{}, false, err
	}
	return tree, stored, nil
}

// insert writes Work w, as p has it, now.
func insert(ctx context.Context, tx *storage.Tx[file], reg registered, w Summary, p parsed, a Adding) error {
	q := db.New(tx)
	w.AddedAt = storage.Now()
	w.RefreshedAt = w.AddedAt
	work := db.InsertWorkParams{ID: w.ID, ContentType: w.ContentType, Key: w.Key, Label: w.Label, SortLabel: w.SortLabel, AddedAt: w.AddedAt, RefreshedAt: w.RefreshedAt}
	if err := q.InsertWork(ctx, work); err != nil {
		return err
	}
	if _, err := converge(ctx, tx, reg.catalog, w.ID, p, w.AddedAt); err != nil {
		return err
	}
	if a.Monitor {
		if err := q.MonitorItems(ctx, w.ID); err != nil {
			return err
		}
	}
	if a.Also == nil {
		return nil
	}
	return a.Also(ctx, tx, w)
}

// Refresh converges a Work on what its source has now, keeping every id,
// and publishes works.changed only when something but the refresh time
// moved. A source that no longer has the Work marks it gone upstream,
// changes nothing else, and is ErrGoneUpstream.
func (w *Works) Refresh(ctx context.Context, by identity.Admin, id WorkID) (Tree, error) {
	changed, err := w.refresh(ctx, id, role.NodeID{})
	if changed {
		w.cfg.Changed.Publish(ctx, ChangedOf(id))
	}
	if err != nil {
		return Tree{}, err
	}
	return w.Get(ctx, by, id)
}

// RefreshDue is Refresh for the Metadata refresh task, which needs no
// Admin: the call leaves from via, the Node the task's runner placed the
// Work on. A call via couldn't take fails with the rpc error, so the
// runner can tell a Node that no longer holds Metadata from a source that
// failed.
func (w *Works) RefreshDue(ctx context.Context, id WorkID, via role.NodeID) error {
	changed, err := w.refresh(ctx, id, via)
	if changed {
		w.cfg.Changed.Publish(ctx, ChangedOf(id))
	}
	return err
}

// refresh converges Work id on its source and reports whether it reads
// differently now, a new gone-upstream mark included.
func (w *Works) refresh(ctx context.Context, id WorkID, via role.NodeID) (bool, error) {
	row, err := db.New(w.cfg.Library).GetWork(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return false, ErrNoWork
	}
	if err != nil {
		return false, err
	}
	reg := w.types[row.ContentType]
	p, err := w.fetch(ctx, row.ContentType, reg, row.Key, Fetching{Purpose: ForRefresh, RatingsReadAt: row.RatingsReadAt, Via: via})
	if errors.Is(err, ErrNotFound) {
		return w.goneUpstream(ctx, id)
	}
	if err != nil {
		return false, err
	}
	var changed bool
	err = w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		changed, err = converge(ctx, tx, reg.catalog, id, p, storage.Now())
		return err
	})
	return changed, err
}

// goneUpstream marks Work id gone upstream, reports whether the mark is
// new, and is ErrGoneUpstream.
func (w *Works) goneUpstream(ctx context.Context, id WorkID) (bool, error) {
	var marked bool
	err := w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		row, err := q.GetWork(ctx, id)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNoWork
		}
		if err != nil {
			return err
		}
		marked = row.GoneUpstreamAt.IsZero()
		return q.MarkGoneUpstream(ctx, db.MarkGoneUpstreamParams{GoneUpstreamAt: storage.Now(), ID: id})
	})
	if err != nil {
		return false, err
	}
	return marked, ErrGoneUpstream
}

// SetMonitored monitors or unmonitors items, publishing one works.changed
// naming each Work whose Items it changed. An id no Item has changes
// nothing.
func (w *Works) SetMonitored(ctx context.Context, _ identity.Admin, items []ItemID, on bool) error {
	var touched []WorkID
	err := w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		var err error
		if touched, err = q.WorksToMonitor(ctx, db.WorksToMonitorParams{ItemIds: items, Monitored: on}); err != nil {
			return err
		}
		return q.SetMonitored(ctx, db.SetMonitoredParams{Monitored: on, ItemIds: items})
	})
	if err != nil {
		return err
	}
	if len(touched) > 0 {
		w.cfg.Changed.Publish(ctx, ChangedOf(touched...))
	}
	return nil
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

func (w *Works) fetch(ctx context.Context, t TypeID, reg registered, key Key, f Fetching) (parsed, error) {
	ctx, cancel := context.WithTimeout(ctx, w.cfg.Timings.Fetch)
	defer cancel()
	fetched, err := w.cfg.Source.Fetch(ctx, t, key, f)
	if err != nil {
		return parsed{}, sourced(t, "fetch "+string(key), err)
	}
	p, err := parseFetched(reg.desc, key, fetched)
	if err != nil {
		w.cfg.Log.Warn("metadata invalid", "contentType", string(t), "key", string(key), "err", err.Error())
	}
	return p, err
}

func sourced(t TypeID, op string, err error) error {
	if errors.Is(err, ErrNotFound) || errors.Is(err, ErrUnavailable) || errors.Is(err, ErrRefused) {
		return err
	}
	return fmt.Errorf("content type %s: %s: %w", t, op, err)
}

// converge writes p over Work id and reports whether anything but the
// refresh times moved: the labels, a level or Item, a rating's value or
// votes, an artwork reference, the gone-upstream mark, or the type's own
// rows. A rating source or artwork kind p lacks keeps its stored row.
func converge(ctx context.Context, tx *storage.Tx[file], c Catalog, id WorkID, p parsed, now storage.Time) (bool, error) {
	q := db.New(tx)
	row, err := q.GetWork(ctx, id)
	if err != nil {
		return false, err
	}
	at, moved, err := writeNodes(ctx, q, id, p, now)
	if err != nil {
		return false, err
	}
	if p.RatingsRead {
		if err := q.SetRatingsRead(ctx, db.SetRatingsReadParams{RatingsReadAt: now, ID: id}); err != nil {
			return false, err
		}
	}
	rated, err := writeRatings(ctx, q, id, at.Items, p.Ratings, now)
	if err != nil {
		return false, err
	}
	drawn, err := writeArtwork(ctx, q, id, at.Items, p.Artwork)
	if err != nil {
		return false, err
	}
	own, err := c.Store(ctx, tx, at, p.Detail)
	if err != nil {
		return false, err
	}
	relabeled := p.Label != row.Label || p.SortLabel != row.SortLabel
	return relabeled || !row.GoneUpstreamAt.IsZero() || moved || rated || drawn || own, nil
}

// writeNodes merges p's levels and Items over Work id's, clears its
// gone-upstream mark, and reports where each node sits and whether any
// level or Item moved.
func writeNodes(ctx context.Context, q *db.Queries, id WorkID, p parsed, now storage.Time) (Placed, bool, error) {
	stored, levelIDs, itemIDs, err := storedNodes(ctx, q, id)
	if err != nil {
		return Placed{}, false, err
	}
	pl := planMerge(stored, p, now)
	if err := q.UpdateWork(ctx, db.UpdateWorkParams{Label: pl.Label, SortLabel: pl.SortLabel, RefreshedAt: pl.RefreshedAt, ID: id}); err != nil {
		return Placed{}, false, err
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
			return Placed{}, false, err
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
			return Placed{}, false, err
		}
	}
	at := Placed{Work: id, Levels: map[Key]LevelID{}, Items: map[Key]ItemID{}}
	for _, n := range p.Levels {
		at.Levels[n.Key] = levelIDs[n.Key]
	}
	for _, n := range p.Items {
		at.Items[n.Key] = itemIDs[n.Key]
	}
	return at, len(pl.Levels) > 0 || len(pl.Items) > 0, nil
}

func storedNodes(ctx context.Context, q *db.Queries, id WorkID) (stored nodes, levelIDs map[Key]LevelID, itemIDs map[Key]ItemID, err error) {
	levels, err := q.ListLevels(ctx, id)
	if err != nil {
		return nodes{}, nil, nil, err
	}
	items, err := q.ListItems(ctx, id)
	if err != nil {
		return nodes{}, nil, nil, err
	}
	levelIDs = map[Key]LevelID{}
	levelKeys := map[LevelID]Key{}
	for _, l := range levels {
		levelIDs[l.Key], levelKeys[l.ID] = l.ID, l.Key
	}
	for _, l := range levels {
		stored.Levels = append(stored.Levels, level{node: node{Key: l.Key, Parent: levelKeys[l.ParentID], Label: l.Label, Position: int(l.Position), WithdrawnAt: l.WithdrawnAt}, Depth: int(l.Depth)})
	}
	itemIDs = map[Key]ItemID{}
	for _, it := range items {
		itemIDs[it.Key] = it.ID
		stored.Items = append(stored.Items, node{Key: it.Key, Parent: levelKeys[it.LevelID], Label: it.Label, Position: int(it.Position), WithdrawnAt: it.WithdrawnAt})
	}
	return stored, levelIDs, itemIDs, nil
}

func writeRatings(ctx context.Context, q *db.Queries, id WorkID, itemIDs map[Key]ItemID, rs []Rating, now storage.Time) (changed bool, err error) {
	type slot struct {
		item   ItemID
		source string
	}
	rows, err := q.ListRatings(ctx, id)
	if err != nil {
		return false, err
	}
	held := map[slot]db.WorksRating{}
	for _, r := range rows {
		held[slot{r.ItemID, r.Source}] = r
	}
	for _, r := range rs {
		item, scale := itemIDs[r.Item], RatingSources[r.Source].Scale
		old, ok := held[slot{item, string(r.Source)}]
		if !ok {
			changed = true
			err = q.InsertRating(ctx, db.InsertRatingParams{WorkID: id, ItemID: item, Source: string(r.Source), Value: r.Value, Scale: scale, Votes: r.Votes, FetchedAt: now})
		} else {
			changed = changed || old.Value != r.Value || old.Scale != scale || !samePtr(old.Votes, r.Votes)
			err = q.UpdateRating(ctx, db.UpdateRatingParams{Value: r.Value, Scale: scale, Votes: r.Votes, FetchedAt: now, WorkID: id, ItemID: item, Source: string(r.Source)})
		}
		if err != nil {
			return false, err
		}
	}
	return changed, nil
}

func samePtr[T comparable](a, b *T) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

func writeArtwork(ctx context.Context, q *db.Queries, id WorkID, itemIDs map[Key]ItemID, as []Art) (changed bool, err error) {
	type slot struct {
		item ItemID
		kind string
	}
	rows, err := q.ListArtwork(ctx, id)
	if err != nil {
		return false, err
	}
	held := map[slot]db.WorksArtwork{}
	for _, r := range rows {
		held[slot{r.ItemID, r.Kind}] = r
	}
	for _, a := range as {
		item := itemIDs[a.Item]
		old, ok := held[slot{item, a.Kind}]
		switch {
		case !ok:
			err = q.InsertArtwork(ctx, db.InsertArtworkParams{WorkID: id, ItemID: item, Kind: a.Kind, Source: string(a.Ref.Source), Path: a.Ref.Path})
		case old.Source != string(a.Ref.Source) || old.Path != a.Ref.Path:
			err = q.UpdateArtwork(ctx, db.UpdateArtworkParams{Source: string(a.Ref.Source), Path: a.Ref.Path, WorkID: id, ItemID: item, Kind: a.Kind})
		default:
			continue
		}
		if err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
}

// Listed is a Work with its own fields, without its levels and Items.
type Listed struct {
	Summary
	Fields []FieldValue
	// Artwork is the Work's own, keyed by the artwork.Kind's key.
	Artwork map[string]artwork.Ref
}

// List lists Works by sort label, then id, with their fields and their
// own artwork, from one snapshot: every Work, or t's.
func (w *Works) List(ctx context.Context, _ identity.Admin, t *TypeID) ([]Listed, error) {
	if t != nil {
		if _, ok := w.types[*t]; !ok {
			return nil, ErrNoType
		}
	}
	var out []Listed
	err := w.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		var rows []db.WorksWork
		var err error
		if t == nil {
			rows, err = q.ListWorks(ctx)
		} else {
			rows, err = q.ListWorksOfType(ctx, *t)
		}
		if err != nil {
			return err
		}
		out = make([]Listed, len(rows))
		for i, row := range rows {
			if out[i], err = w.listed(ctx, r, row); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}

func (w *Works) listed(ctx context.Context, r storage.Reader[file], row db.WorksWork) (Listed, error) {
	reg := w.types[row.ContentType]
	fs, err := reg.catalog.Fields(ctx, r, row.ID)
	if err != nil {
		return Listed{}, fmt.Errorf("content type %s: fields of work %s: %w", row.ContentType, row.ID, err)
	}
	values, err := ordered(reg.specs(TierWork), fs.Work)
	if err != nil {
		return Listed{}, fmt.Errorf("content type %s: fields of work %s: %w", row.ContentType, row.ID, err)
	}
	art, err := db.New(r).ListArtwork(ctx, row.ID)
	if err != nil {
		return Listed{}, err
	}
	l := Listed{Summary: Summary(row), Fields: values, Artwork: map[string]artwork.Ref{}}
	for _, a := range art {
		if a.ItemID.IsZero() {
			l.Artwork[a.Kind] = refOf(reg.desc, a)
		}
	}
	return l, nil
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
	ratings, err := q.ListRatings(ctx, id)
	if err != nil {
		return Tree{}, err
	}
	art, err := q.ListArtwork(ctx, id)
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
	t.attach(reg.desc, ratings, art)
	return t, nil
}

func (t *Tree) attach(d Description, ratings []db.WorksRating, art []db.WorksArtwork) {
	at := map[ItemID]int{}
	for i, it := range t.Items {
		at[it.ID] = i
	}
	for _, r := range ratings {
		v := StoredRating{Source: RatingSource(r.Source), Value: r.Value, Votes: r.Votes, FetchedAt: r.FetchedAt}
		if r.ItemID.IsZero() {
			t.Ratings = append(t.Ratings, v)
			continue
		}
		it := &t.Items[at[r.ItemID]]
		it.Ratings = append(it.Ratings, v)
	}
	for _, a := range art {
		owner := &t.Artwork
		if !a.ItemID.IsZero() {
			owner = &t.Items[at[a.ItemID]].Artwork
		}
		if *owner == nil {
			*owner = map[string]artwork.Ref{}
		}
		(*owner)[a.Kind] = refOf(d, a)
	}
}

// refOf is a stored artwork row as a reference, shaped as its kind is.
func refOf(d Description, a db.WorksArtwork) artwork.Ref {
	item := !a.ItemID.IsZero()
	i := slices.IndexFunc(d.Artwork, func(k artwork.Kind) bool { return k.Key == a.Kind && k.Item == item })
	return artwork.Ref{Source: artwork.SourceID(a.Source), Shape: d.Artwork[i].Shape, Path: a.Path}
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
		t.Items = append(t.Items, Item{ID: it.ID, LevelID: it.LevelID, Key: it.Key, Label: it.Label, Position: int(it.Position), WithdrawnAt: it.WithdrawnAt, Monitored: it.Monitored, Fields: values})
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

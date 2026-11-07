package works

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// Preset is a monitoring preset: a rule over level order, the aside mark
// and each Item's facts. It is not targets.Preset, which lays out quality
// tiers.
type Preset string

const (
	PresetAll      Preset = "all"
	PresetFuture   Preset = "future"
	PresetMissing  Preset = "missing"
	PresetExisting Preset = "existing"
	PresetFirst    Preset = "first"
	PresetLatest   Preset = "latest"
	PresetNone     Preset = "none"
)

var presets = []Preset{PresetAll, PresetFuture, PresetMissing, PresetExisting, PresetFirst, PresetLatest, PresetNone}

// Presets is every Preset in the order choices show them.
func Presets() []Preset { return slices.Clone(presets) }

func ParsePreset(s string) (Preset, error) {
	if p := Preset(s); slices.Contains(presets, p) {
		return p, nil
	}
	return "", fmt.Errorf("%q is not a monitoring preset", s)
}

// MonitorsNew is the preset's monitor_new: on for All, Future, Missing and
// Latest.
func (p Preset) MonitorsNew() bool {
	return p == PresetAll || p == PresetFuture || p == PresetMissing || p == PresetLatest
}

// NeedsFacts is whether Monitors reads ItemFacts for the preset. All,
// First and None read the tree alone.
func (p Preset) NeedsFacts() bool {
	return p == PresetFuture || p == PresetMissing || p == PresetExisting || p == PresetLatest
}

// setBased is whether the preset picks by the aside mark alone, so
// ApplyPreset writes it over whole Works without reading their trees.
func (p Preset) setBased() bool { return p == PresetAll || p == PresetNone }

type Choice struct {
	Preset Preset
	// Asides, when set, adds every aside Item to the set and becomes the
	// Work's MonitorAsides; nil keeps each Work's own.
	Asides *bool
	// New overrides Preset.MonitorsNew when set.
	New *bool
}

// ItemFacts is what the presets read of one Item that works doesn't hold.
type ItemFacts struct {
	// Released: out on the Controller's calendar date.
	Released bool
	// Filled: a file fills at least one of its cells, Met or Upgradable.
	Filled bool
}

type Facts map[ItemID]ItemFacts

func (c Choice) asidesOf(w Summary) bool {
	if c.Asides == nil {
		return w.MonitorAsides
	}
	return *c.Asides
}

func (c Choice) monitorNew() bool {
	if c.New == nil {
		return c.Preset.MonitorsNew()
	}
	return *c.New
}

// run is a Work's live Items outside asides, and the same Items under each
// main level: its live outermost levels without the aside mark that hold
// one, in position order. A type with no levels has one main level
// holding every Item.
type run struct {
	base   []Item
	levels [][]Item
}

func runOf(t Tree) run {
	var r run
	if len(t.Levels) == 0 {
		for _, it := range t.Items {
			if it.Live() {
				r.base = append(r.base, it)
			}
		}
		r.levels = [][]Item{r.base}
		return r
	}
	parent := make(map[LevelID]LevelID, len(t.Levels))
	for _, l := range t.Levels {
		parent[l.ID] = l.ParentID
	}
	under := map[LevelID][]Item{}
	for _, it := range t.Items {
		if !it.Live() || it.Aside {
			continue
		}
		r.base = append(r.base, it)
		top := it.LevelID
		for !parent[top].IsZero() {
			top = parent[top]
		}
		under[top] = append(under[top], it)
	}
	var mains []Level
	for _, l := range t.Levels {
		if l.ParentID.IsZero() && l.WithdrawnAt.IsZero() && !l.Aside && len(under[l.ID]) > 0 {
			mains = append(mains, l)
		}
	}
	slices.SortStableFunc(mains, func(a, b Level) int { return cmp.Compare(a.Position, b.Position) })
	for _, l := range mains {
		r.levels = append(r.levels, under[l.ID])
	}
	return r
}

func (r run) level(i int) []Item {
	if i >= len(r.levels) {
		return nil
	}
	return r.levels[i]
}

func where(items []Item, keep func(Item) bool) []Item {
	var out []Item
	for _, it := range items {
		if keep(it) {
			out = append(out, it)
		}
	}
	return out
}

// rules is each Preset's pick among a Work's main run, asides left out.
var rules = map[Preset]func(r run, f Facts) []Item{
	PresetAll: func(r run, _ Facts) []Item { return r.base },
	PresetFuture: func(r run, f Facts) []Item {
		return where(r.base, func(it Item) bool { return !f[it.ID].Released })
	},
	PresetMissing: func(r run, f Facts) []Item {
		return where(r.base, func(it Item) bool { return !f[it.ID].Filled || !f[it.ID].Released })
	},
	PresetExisting: func(r run, f Facts) []Item {
		return where(r.base, func(it Item) bool { return f[it.ID].Filled })
	},
	PresetFirst: func(r run, _ Facts) []Item { return r.level(0) },
	PresetLatest: func(r run, f Facts) []Item {
		for i := len(r.levels) - 1; i >= 0; i-- {
			if slices.ContainsFunc(r.levels[i], func(it Item) bool { return f[it.ID].Released }) {
				return r.levels[i]
			}
		}
		return r.level(0)
	},
	PresetNone: func(run, Facts) []Item { return nil },
}

// Monitors is the set c monitors among t's live Items, under facts. Pure.
func Monitors(t Tree, facts Facts, c Choice) map[ItemID]bool {
	on := map[ItemID]bool{}
	for _, it := range rules[c.Preset](runOf(t), facts) {
		on[it.ID] = true
	}
	if c.asidesOf(t.Summary) {
		for _, it := range t.Items {
			if it.Live() && it.Aside {
				on[it.ID] = true
			}
		}
	}
	return on
}

type PresetCount struct {
	Preset      Preset
	Monitors    int
	MonitorsNew bool
}

// PresetCounts is what each Preset monitors without asides, and how many
// live aside Items Include adds to any of them.
type PresetCounts struct {
	Choices []PresetCount
	Asides  int
}

// CountPresets is Monitors' size for each Preset. Pure.
func CountPresets(t Tree, facts Facts) PresetCounts {
	r := runOf(t)
	out := PresetCounts{Choices: make([]PresetCount, len(presets))}
	for i, p := range presets {
		out.Choices[i] = PresetCount{Preset: p, Monitors: len(rules[p](r, facts)), MonitorsNew: p.MonitorsNew()}
	}
	for _, it := range t.Items {
		if it.Live() && it.Aside {
			out.Asides++
		}
	}
	return out
}

// Delta is what applying a Choice changes on one Work.
type Delta struct {
	Start, Stop int
	// Rule: MonitorNew or MonitorAsides moved.
	Rule bool
}

func (d Delta) Zero() bool { return d == Delta{} }

// moving is d with n Items that are monitored as was and should be as
// want.
func (d Delta) moving(want, was bool, n int) Delta {
	switch {
	case want && !was:
		d.Start += n
	case !want && was:
		d.Stop += n
	}
	return d
}

// ApplyPreset makes every live Item of each of ws monitored exactly when
// Monitors says under c, and stores c's MonitorNew and MonitorAsides on
// each, in tx. Withdrawn Items keep their flags. All and None are
// set-based over ws and read no facts; facts is read only when
// c.Preset.NeedsFacts. It publishes nothing: the composer publishes once
// after commit, or rolls back for a preview. Applying it twice changes
// nothing the second time. A Work ws names that doesn't exist is absent
// from the answer.
func (w *Works) ApplyPreset(ctx context.Context, tx *storage.Tx[file], ws Among[WorkID], c Choice, facts map[WorkID]Facts) (map[WorkID]Delta, error) {
	q := db.New(tx)
	rows, err := Read(ws, func() ([]db.WorksWork, error) { return q.ListWorks(ctx) }, func(ids []WorkID) ([]db.WorksWork, error) {
		return q.ListWorksOf(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	out := make(map[WorkID]Delta, len(rows))
	for _, row := range rows {
		out[row.ID] = Delta{Rule: row.MonitorNew != c.monitorNew() || row.MonitorAsides != c.asidesOf(Summary(row))}
	}
	present := Like(ws, idsOf(rows))
	if c.Preset.setBased() {
		err = applySet(ctx, q, rows, present, c, out)
	} else {
		err = applyTrees(ctx, q, rows, present, c, facts, out)
	}
	if err != nil {
		return nil, err
	}
	return out, writeRules(ctx, q, rows, c, out)
}

// applySet writes All or None over whole Works: Items outside asides take
// the preset's flag, and aside Items each Work's asides.
func applySet(ctx context.Context, q *db.Queries, rows []db.WorksWork, present Among[WorkID], c Choice, out map[WorkID]Delta) error {
	counts, err := Read(present, func() ([]db.CountMonitoredOfWorksRow, error) {
		rows, err := q.CountEveryMonitored(ctx)
		return convert(rows, func(r db.CountEveryMonitoredRow) db.CountMonitoredOfWorksRow { return db.CountMonitoredOfWorksRow(r) }), err
	}, func(ids []WorkID) ([]db.CountMonitoredOfWorksRow, error) { return q.CountMonitoredOfWorks(ctx, ids) })
	if err != nil {
		return err
	}
	main := c.Preset == PresetAll
	asides := make(map[WorkID]bool, len(rows))
	for _, row := range rows {
		asides[row.ID] = c.asidesOf(Summary(row))
	}
	for _, n := range counts {
		want := main
		if n.Aside {
			want = asides[n.WorkID]
		}
		out[n.WorkID] = out[n.WorkID].moving(want, n.Monitored, int(n.Items))
	}
	var moved []WorkID
	for _, row := range rows {
		if d := out[row.ID]; d.Start+d.Stop > 0 {
			moved = append(moved, row.ID)
		}
	}
	for chunk := range slices.Chunk(moved, Batch) {
		if err := q.SetMonitoredOfWorks(ctx, db.SetMonitoredOfWorksParams{WorkIds: chunk, Aside: false, Monitored: main}); err != nil {
			return err
		}
		if err := setAsides(ctx, q, chunk, asides); err != nil {
			return err
		}
	}
	return nil
}

// setAsides turns the aside Items of each of ws on or off as asides says.
func setAsides(ctx context.Context, q *db.Queries, ws []WorkID, asides map[WorkID]bool) error {
	for _, on := range []bool{true, false} {
		of := slices.DeleteFunc(slices.Clone(ws), func(id WorkID) bool { return asides[id] != on })
		if len(of) == 0 {
			continue
		}
		if err := q.SetMonitoredOfWorks(ctx, db.SetMonitoredOfWorksParams{WorkIds: of, Aside: true, Monitored: on}); err != nil {
			return err
		}
	}
	return nil
}

// applyTrees writes a preset that reads each Work's levels or facts, Item
// by Item.
func applyTrees(ctx context.Context, q *db.Queries, rows []db.WorksWork, present Among[WorkID], c Choice, facts map[WorkID]Facts, out map[WorkID]Delta) error {
	trees, err := outlines(ctx, q, rows, present)
	if err != nil {
		return err
	}
	var start, stop []ItemID
	for _, t := range trees {
		on := Monitors(t, facts[t.ID], c)
		d := out[t.ID]
		for _, it := range t.Items {
			switch {
			case !it.Live() || on[it.ID] == it.Monitored:
			case on[it.ID]:
				start = append(start, it.ID)
				d.Start++
			default:
				stop = append(stop, it.ID)
				d.Stop++
			}
		}
		out[t.ID] = d
	}
	if err := setItems(ctx, q, start, true); err != nil {
		return err
	}
	return setItems(ctx, q, stop, false)
}

// writeRules stores c's rule on each Work whose rule moved, in one
// statement per Batch of them.
func writeRules(ctx context.Context, q *db.Queries, rows []db.WorksWork, c Choice, out map[WorkID]Delta) error {
	var moved []WorkID
	for _, row := range rows {
		if out[row.ID].Rule {
			moved = append(moved, row.ID)
		}
	}
	for chunk := range slices.Chunk(moved, Batch) {
		var err error
		if c.Asides == nil {
			err = q.SetMonitorNew(ctx, db.SetMonitorNewParams{MonitorNew: c.monitorNew(), Ids: chunk})
		} else {
			err = q.SetWorkRules(ctx, db.SetWorkRulesParams{MonitorNew: c.monitorNew(), MonitorAsides: *c.Asides, Ids: chunk})
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func setItems(ctx context.Context, q *db.Queries, ids []ItemID, on bool) error {
	for chunk := range slices.Chunk(ids, Batch) {
		if err := q.SetItemsMonitored(ctx, db.SetItemsMonitoredParams{Monitored: on, ItemIds: chunk}); err != nil {
			return err
		}
	}
	return nil
}

// OutlinesIn reads each of ws a Work has inside a caller's read: its
// summary, its levels and its Items in tree order, without fields,
// ratings or artwork. It is what Monitors and CountPresets read.
func OutlinesIn(ctx context.Context, r storage.Reader[file], ws Among[WorkID]) ([]Tree, error) {
	q := db.New(r)
	rows, err := Read(ws, func() ([]db.WorksWork, error) { return q.ListWorks(ctx) }, func(ids []WorkID) ([]db.WorksWork, error) {
		return q.ListWorksOf(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	return outlines(ctx, q, rows, Like(ws, idsOf(rows)))
}

// outlines is the outline of each of rows, which among names.
func outlines(ctx context.Context, q *db.Queries, rows []db.WorksWork, among Among[WorkID]) ([]Tree, error) {
	items, err := Read(among, func() ([]db.WorksItem, error) { return q.ListEveryItem(ctx) }, func(ids []WorkID) ([]db.WorksItem, error) {
		return q.ListItemsOfWorks(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	levels, err := Read(among, func() ([]db.WorksLevel, error) { return q.ListEveryLevel(ctx) }, func(ids []WorkID) ([]db.WorksLevel, error) {
		return q.ListLevelsOfWorks(ctx, ids)
	})
	if err != nil {
		return nil, err
	}
	itemsOf := make(map[WorkID][]db.WorksItem, len(rows))
	for _, it := range items {
		itemsOf[it.WorkID] = append(itemsOf[it.WorkID], it)
	}
	levelsOf := map[WorkID][]db.WorksLevel{}
	for _, l := range levels {
		levelsOf[l.WorkID] = append(levelsOf[l.WorkID], l)
	}
	out := make([]Tree, len(rows))
	for i, row := range rows {
		ls, its := readOrder(levelsOf[row.ID], itemsOf[row.ID])
		treeOrder(ls, its)
		t := Tree{Summary: Summary(row), Levels: make([]Level, len(ls)), Items: make([]Item, len(its))}
		for j, l := range ls {
			t.Levels[j] = Level{ID: l.ID, ParentID: l.ParentID, Depth: int(l.Depth), Key: l.Key, Label: l.Label, Position: int(l.Position), WithdrawnAt: l.WithdrawnAt, Aside: l.Aside}
		}
		for j, it := range its {
			t.Items[j] = bare(it)
		}
		out[i] = t
	}
	return out, nil
}

// Within names part of a Work: the Items under Levels and the Items
// named.
type Within struct {
	Levels []LevelID
	Items  []ItemID
}

// NotInWorkError is ErrNotInWork for one id of a Within: the Index-th of
// its Levels when Level, else of its Items.
type NotInWorkError struct {
	Level bool
	Index int
}

func (e *NotInWorkError) Error() string { return ErrNotInWork.Error() }

func (e *NotInWorkError) Unwrap() error { return ErrNotInWork }

// within is the Items part names of t: the live Items under each named
// level, at any depth, and each named Item.
func within(t Tree, part Within) ([]Item, error) {
	parent := make(map[LevelID]LevelID, len(t.Levels))
	for _, l := range t.Levels {
		parent[l.ID] = l.ParentID
	}
	named := map[LevelID]bool{}
	for i, id := range part.Levels {
		if _, ok := parent[id]; !ok {
			return nil, &NotInWorkError{Level: true, Index: i}
		}
		named[id] = true
	}
	at := make(map[ItemID]int, len(t.Items))
	for i, it := range t.Items {
		at[it.ID] = i
	}
	picked := map[ItemID]bool{}
	for i, id := range part.Items {
		if _, ok := at[id]; !ok {
			return nil, &NotInWorkError{Index: i}
		}
		picked[id] = true
	}
	under := func(l LevelID) bool {
		for ; !l.IsZero(); l = parent[l] {
			if named[l] {
				return true
			}
		}
		return false
	}
	var out []Item
	for _, it := range t.Items {
		if picked[it.ID] || it.Live() && under(it.LevelID) {
			out = append(out, it)
		}
	}
	return out, nil
}

// SetMonitored turns the Items within part of Work id on or off, leaving
// the Work's rule, and publishes works.changed when any moved. changed is
// how many did. ErrNoWork, or a *NotInWorkError.
func (w *Works) SetMonitored(ctx context.Context, _ identity.Admin, id WorkID, part Within, on bool) (changed int, err error) {
	var moved []ItemID
	err = w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		rows, err := q.ListWorksOf(ctx, []WorkID{id})
		switch {
		case err != nil:
			return err
		case len(rows) == 0:
			return ErrNoWork
		}
		trees, err := outlines(ctx, q, rows, Only([]WorkID{id}))
		if err != nil {
			return err
		}
		items, err := within(trees[0], part)
		if err != nil {
			return err
		}
		for _, it := range items {
			if it.Monitored != on {
				moved = append(moved, it.ID)
			}
		}
		return setItems(ctx, q, moved, on)
	})
	if err != nil {
		return 0, err
	}
	if len(moved) > 0 {
		w.Touched(ctx, []WorkID{id})
	}
	return len(moved), nil
}

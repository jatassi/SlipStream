package grabs

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/pacing"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Queued is one search group a press landed on: its key, its label, the
// Items of it the press searches, and whether it merged into a row
// already queued or searching with its scope.
type Queued struct {
	Key    indexers.GroupKey
	Label  string
	Items  []works.ItemID
	Merged bool
}

// landed is a trigger as enqueue lands it: its laneRules row, in the lane
// of a call for one Work or several.
type landed struct {
	trigger searches.Trigger
	lane    pace.Lane
	gone    bool
	resets  bool
	holds   bool
}

func landing(t searches.Trigger, several bool) landed {
	r := laneRules[t]
	lane := r.single
	if several {
		lane = r.several
	}
	return landed{trigger: t, lane: lane, gone: r.gone, resets: r.resets, holds: r.holds}
}

// enqueue lands e on each search group of cells' Items, cells of work of
// Content Type typ, in tx, under Backlog search's interval, and answers
// each group's state.
func (gr *Grabs) enqueue(ctx context.Context, tx *storage.Tx[file], work works.WorkID, typ works.TypeID, cells []files.Cell, e landed, interval time.Duration) ([]Queued, error) {
	groups, err := gr.cfg.Terms[typ].Groups(ctx, tx, work, itemsOf(cells))
	if err != nil {
		return nil, err
	}
	return land(ctx, tx, work, groups, cells, e, interval)
}

// land lands e on each of groups, the search groups of cells' Items, in
// tx. It is the one writer of a due row's queued entry, and promotes an
// open run e outranks.
func land(ctx context.Context, tx *storage.Tx[file], work works.WorkID, groups []indexers.Group, cells []files.Cell, e landed, interval time.Duration) ([]Queued, error) {
	q, now := db.New(tx), storage.Now().Time()
	out := make([]Queued, 0, len(groups))
	for _, g := range groups {
		row, run, err := dueOf(ctx, q, work, g.Key)
		if err != nil {
			return nil, err
		}
		next, state, promote := pace.Merge(row, pace.Entry{
			Lane: e.lane, Trigger: e.trigger, Holds: e.holds, Items: keysOf(g.Items), Gone: e.gone,
			Order: orderOf(within(cells, g.Items)), ResetGap: e.resets, At: now,
		}, interval, pacing.Spread(work.String(), string(g.Key), interval))
		if err := putDue(ctx, q, work, g.Key, next); err != nil {
			return nil, err
		}
		if promote {
			p := db.PromoteSearchParams{ID: run, Lane: new(int64(e.lane)), Trigger: new(string(e.trigger)), IncludeGone: e.gone}
			if err := q.PromoteSearch(ctx, p); err != nil {
				return nil, err
			}
		}
		out = append(out, Queued{Key: g.Key, Label: g.Label, Items: g.Items, Merged: state == pace.Merged})
	}
	return out, nil
}

// dueOf is the due row of work's group key as Merge reads it, with its
// open run, and that run's id. A group with no row reads as a Row that
// doesn't exist.
func dueOf(ctx context.Context, q *db.Queries, work works.WorkID, key indexers.GroupKey) (pace.Row, SearchID, error) {
	d, err := q.GetDue(ctx, db.GetDueParams{WorkID: work, GroupKey: key})
	if errors.Is(err, storage.ErrNoRows) {
		return pace.Row{}, SearchID{}, nil
	}
	if err != nil {
		return pace.Row{}, SearchID{}, err
	}
	r := pace.Row{
		Exists: true, Lane: laneOf(d.Lane), Trigger: searches.Trigger(deref(d.Trigger)), Gone: d.IncludeGone,
		Order:  pace.Order{UpgradeOnly: d.UpgradeOnly, Fresh: d.Fresh, Released: d.ReleasedAt.Time()},
		Queued: d.QueuedAt.Time(), Due: d.DueAt.Time(), Gap: time.Duration(d.GapSeconds) * time.Second, Changed: d.ChangedAt.Time(),
	}
	r.Holds = laneRules[r.Trigger].holds
	if r.Lane != 0 {
		items, err := q.ListDueItems(ctx, db.ListDueItemsParams{WorkID: work, GroupKey: key})
		if err != nil {
			return pace.Row{}, SearchID{}, err
		}
		if len(items) > 0 {
			r.Items = keysOf(items)
		}
	}
	switch last, err := q.LastSearchOfGroup(ctx, db.LastSearchOfGroupParams{WorkID: work, GroupKey: &key}); {
	case err == nil:
		r.Searched = last.Time()
	case !errors.Is(err, storage.ErrNoRows):
		return pace.Row{}, SearchID{}, err
	}
	if d.SearchID.IsZero() {
		return r, SearchID{}, nil
	}
	run, err := q.GetSearch(ctx, d.SearchID)
	if err != nil {
		return pace.Row{}, SearchID{}, err
	}
	items, err := q.ListSearchItems(ctx, []db.SearchID{run.ID})
	if err != nil {
		return pace.Row{}, SearchID{}, err
	}
	searched := make([]works.ItemID, len(items))
	for i, it := range items {
		searched[i] = it.ItemID
	}
	r.Searching = &pace.Run{Lane: laneOf(run.Lane), Holds: laneRules[searches.Trigger(deref(run.Trigger))].holds, Items: keysOf(searched)}
	return r, run.ID, nil
}

// putDue writes r as work's due row of key, with its queued scope.
func putDue(ctx context.Context, q *db.Queries, work works.WorkID, key indexers.GroupKey, r pace.Row) error {
	p := db.PutDueParams{
		WorkID: work, GroupKey: key, IncludeGone: r.Gone, UpgradeOnly: r.Order.UpgradeOnly, Fresh: r.Order.Fresh,
		ReleasedAt: storage.FromTime(r.Order.Released), DueAt: storage.FromTime(r.Due), GapSeconds: int64(r.Gap / time.Second),
		ChangedAt: storage.FromTime(r.Changed),
	}
	if r.Lane != 0 {
		p.Lane, p.QueuedAt, p.Trigger = new(int64(r.Lane)), storage.FromTime(r.Queued), new(string(r.Trigger))
	}
	if err := q.PutDue(ctx, p); err != nil {
		return err
	}
	if err := q.DeleteDueItems(ctx, db.DeleteDueItemsParams{WorkID: work, GroupKey: key}); err != nil {
		return err
	}
	if r.Lane == 0 {
		return nil
	}
	for _, s := range r.Items {
		id, err := storage.ParseUUID(s)
		if err != nil {
			return err
		}
		if err := q.InsertDueItem(ctx, db.InsertDueItemParams{WorkID: work, GroupKey: key, ItemID: works.ItemID{UUID: id}}); err != nil {
			return err
		}
	}
	return nil
}

// orderOf is a group's place in its lane from its cells: upgrade only
// when none is Missing or Failed, and its newest release.
func orderOf(cells []files.Cell) pace.Order {
	o := pace.Order{UpgradeOnly: true}
	for _, c := range cells {
		if c.State == files.Missing || c.State == files.Failed {
			o.UpgradeOnly = false
		}
		if from, ok := c.Timing.From(); ok && from.After(o.Released) {
			o.Released = from
		}
	}
	return o
}

func laneOf(rank *int64) pace.Lane {
	if rank == nil {
		return 0
	}
	return pace.Lane(*rank)
}

// itemsOf is cells' Items, each once, in cells' order.
func itemsOf(cells []files.Cell) []works.ItemID {
	var out []works.ItemID
	seen := map[works.ItemID]bool{}
	for _, c := range cells {
		if !seen[c.Item.ID] {
			seen[c.Item.ID] = true
			out = append(out, c.Item.ID)
		}
	}
	return out
}

// within is the cells of items.
func within(cells []files.Cell, items []works.ItemID) []files.Cell {
	in := make(map[works.ItemID]bool, len(items))
	for _, i := range items {
		in[i] = true
	}
	var out []files.Cell
	for _, c := range cells {
		if in[c.Item.ID] {
			out = append(out, c)
		}
	}
	return out
}

// keysOf is items as pace keys them.
func keysOf(items []works.ItemID) []string {
	out := make([]string, len(items))
	for i, id := range items {
		out[i] = id.String()
	}
	return out
}

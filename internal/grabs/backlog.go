package grabs

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// dueKey is one queue row's key: a Work's search group.
type dueKey struct {
	work  works.WorkID
	group indexers.GroupKey
}

func (k dueKey) compare(o dueKey) int {
	if c := k.work.Compare(o.work.UUID); c != 0 {
		return c
	}
	return strings.Compare(string(k.group), string(o.group))
}

// want is a group the snapshot found wanted: its place in its lane, and
// the gap a first row gets. A group only unseen listed cuts want is paced
// at the longest gap.
type want struct {
	order pace.Order
	gap   time.Duration
}

// dueStep is what a reconcile does to one row: keep it in step with its
// want, or, with none, delete it if nothing touched it since the snapshot.
type dueStep struct {
	key  dueKey
	want *want
}

// backlogPlan is one Backlog search's snapshot, taken at seen, as the
// keys its chunked Writes apply: every row's step, sorted, and the cells
// whose failure rows go, since the snapshot found them not Paced.
type backlogPlan struct {
	seen     storage.Time
	upgrades bool
	steps    []dueStep
	cleared  []files.CellKey
}

// backlog is Backlog search's one unit: it reconciles the queue with what
// is wanted and queues the rows that are due, then ends. Dispatch drains
// the lane, which Wanted and the Work page show.
func (gr *Grabs) backlog(ctx context.Context, _ tasks.Unit, p *tasks.Progress) error {
	plan, on, err := gr.planBacklog(ctx)
	if err != nil {
		return err
	}
	if !on {
		return tasks.SkipOwned("Backlog search is off in Settings.")
	}
	n, err := gr.applyBacklog(ctx, plan)
	if err != nil {
		return err
	}
	p.Note(queuedWords(n))
	return nil
}

func queuedWords(n int) string {
	if n == 1 {
		return "Queued 1 group for search."
	}
	return fmt.Sprintf("Queued %d groups for search.", n)
}

// planBacklog reads the snapshot in one Read: the settings, every Work's
// wanted groups, every row's key and every cell with failure rows. It is
// false while Backlog search is off.
func (gr *Grabs) planBacklog(ctx context.Context) (backlogPlan, bool, error) {
	interval, err := gr.interval(ctx)
	if err != nil {
		return backlogPlan{}, false, err
	}
	var plan backlogPlan
	on := false
	err = gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		s, err := q.GetSettings(ctx)
		if err != nil {
			return err
		}
		if on = BacklogOn.Value(s.BacklogOn); !on {
			return nil
		}
		plan.seen, plan.upgrades = storage.Now(), BacklogUpgrades.Value(s.BacklogUpgrades)
		ws, err := gr.cfg.Files.Wanted(ctx, r, works.Every[works.WorkID](nil))
		if err != nil {
			return err
		}
		wants, paced, err := gr.wantsOf(ctx, r, ws, interval, max(LongestGap.Value(s.LongestGapSeconds), interval))
		if err != nil {
			return err
		}
		keys, err := q.ListDueKeys(ctx)
		if err != nil {
			return err
		}
		plan.steps = stepsOf(wants, keys)
		failing, err := q.ListFailureCells(ctx)
		for _, f := range failing {
			if k := (files.CellKey{Item: f.ItemID, Edition: f.Edition, Target: f.TargetID}); !paced[k] {
				plan.cleared = append(plan.cleared, k)
			}
		}
		return err
	})
	return plan, on, err
}

// wantsOf is the wanted groups of ws, by Facet.Groups per Work: the
// groups of its Paced cells, stopped ones included, and the groups of its
// Items with an unseen listed cut and no Paced cell, at longest. It also
// answers every Paced cell.
func (gr *Grabs) wantsOf(ctx context.Context, r storage.Reader[file], ws []files.WorkWanted, interval, longest time.Duration) (wants map[dueKey]want, paced map[files.CellKey]bool, err error) {
	wants, paced = map[dueKey]want{}, map[files.CellKey]bool{}
	for _, w := range ws {
		cells := slices.DeleteFunc(slices.Clone(w.Cells), func(c files.Cell) bool { return !c.Paced() })
		for _, c := range cells {
			paced[files.CellKey{Item: c.Item.ID, Edition: c.Edition, Target: c.Target.ID}] = true
		}
		facet := gr.cfg.Terms[w.Type]
		groups, err := facet.Groups(ctx, r, w.Work, itemsOf(cells))
		if err != nil {
			return nil, nil, err
		}
		for _, g := range groups {
			wants[dueKey{w.Work, g.Key}] = want{order: orderOf(within(cells, g.Items)), gap: interval}
		}
		unseen := slices.DeleteFunc(slices.Clone(w.Unseen), func(it works.ItemID) bool {
			return slices.ContainsFunc(cells, func(c files.Cell) bool { return c.Item.ID == it })
		})
		if len(unseen) == 0 {
			continue
		}
		groups, err = facet.Groups(ctx, r, w.Work, unseen)
		if err != nil {
			return nil, nil, err
		}
		for _, g := range groups {
			if _, ok := wants[dueKey{w.Work, g.Key}]; !ok {
				wants[dueKey{w.Work, g.Key}] = want{order: pace.Order{UpgradeOnly: true}, gap: longest}
			}
		}
	}
	return wants, paced, nil
}

// stepsOf is a step for each wanted group and each row of keys no group
// wants, sorted by key.
func stepsOf(wants map[dueKey]want, keys []db.ListDueKeysRow) []dueStep {
	steps := make([]dueStep, 0, len(wants))
	for k, w := range wants {
		steps = append(steps, dueStep{key: k, want: &w})
	}
	for _, row := range keys {
		if k := (dueKey{row.WorkID, row.GroupKey}); !has(wants, k) {
			steps = append(steps, dueStep{key: k})
		}
	}
	slices.SortFunc(steps, func(a, b dueStep) int { return a.key.compare(b.key) })
	return steps
}

func has(wants map[dueKey]want, k dueKey) bool {
	_, ok := wants[k]
	return ok
}

// applyBacklog applies plan in Writes of at most BacklogChunk keys each,
// and answers how many rows it queued. In each Write a wanted group lacking
// a row gets one, spread over its gap; a row refreshes its order keys; a
// due, idle row joins the backlog lane; a row nothing wants and nothing
// touched since the snapshot goes; and a cell's failure rows go unless it
// failed since or a Grab holds it. A crash between Writes leaves the
// earlier ones applied, and the next run reconciles the rest.
func (gr *Grabs) applyBacklog(ctx context.Context, plan backlogPlan) (int, error) {
	queued := 0
	for chunk := range slices.Chunk(plan.steps, gr.cfg.Timings.BacklogChunk) {
		err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			n, err := applySteps(ctx, db.New(tx), plan, chunk)
			queued += n
			return err
		})
		if err != nil {
			return queued, err
		}
		gr.cfg.QueueChanged.Publish(ctx, QueueChanged{WorkIDs: []string{}})
		gr.wake()
	}
	for chunk := range slices.Chunk(plan.cleared, gr.cfg.Timings.BacklogChunk) {
		err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
			return clearStale(ctx, tx, plan.seen, chunk)
		})
		if err != nil {
			return queued, err
		}
	}
	return queued, nil
}

func applySteps(ctx context.Context, q *db.Queries, plan backlogPlan, steps []dueStep) (int, error) {
	now := storage.Now()
	queued := 0
	for _, s := range steps {
		k := s.key
		if s.want == nil {
			if err := q.DeleteIdleDue(ctx, db.DeleteIdleDueParams{WorkID: k.work, GroupKey: k.group, Seen: plan.seen}); err != nil {
				return queued, err
			}
			continue
		}
		o := s.want.order
		err := q.InsertBacklogDue(ctx, db.InsertBacklogDueParams{
			WorkID: k.work, GroupKey: k.group, UpgradeOnly: o.UpgradeOnly, Fresh: o.Fresh, ReleasedAt: storage.FromTime(o.Released),
			DueAt:      storage.FromTime(plan.seen.Time().Add(pace.Spread(k.work.String(), string(k.group), s.want.gap))),
			GapSeconds: int64(s.want.gap / time.Second), ChangedAt: now,
		})
		if err != nil {
			return queued, err
		}
		err = q.RefreshDueOrder(ctx, db.RefreshDueOrderParams{
			WorkID: k.work, GroupKey: k.group, UpgradeOnly: o.UpgradeOnly, Fresh: o.Fresh, ReleasedAt: storage.FromTime(o.Released), Seen: plan.seen,
		})
		if err != nil {
			return queued, err
		}
		n, err := q.QueueBacklogDue(ctx, db.QueueBacklogDueParams{
			WorkID: k.work, GroupKey: k.group, Lane: new(int64(pace.Backlog)), Trigger: new(string(searches.Backlog)),
			QueuedAt: now, Upgrades: plan.upgrades,
		})
		if err != nil {
			return queued, err
		}
		queued += int(n)
	}
	return queued, nil
}

// clearStale clears the failure rows of cells, which the snapshot taken
// at seen found not Paced, but for a cell that failed since or that a
// Grab still holds.
func clearStale(ctx context.Context, tx *storage.Tx[file], seen storage.Time, cells []files.CellKey) error {
	q := db.New(tx)
	var gone []files.CellKey
	for _, k := range cells {
		kept, err := q.FailuresKept(ctx, db.FailuresKeptParams{ItemID: k.Item, Edition: k.Edition, TargetID: k.Target, Seen: seen})
		if err != nil {
			return err
		}
		if !kept {
			gone = append(gone, k)
		}
	}
	return clearFailures(ctx, tx, gone)
}

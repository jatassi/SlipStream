package grabs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// opens reports whether a run of trigger t, with gone cells when gone,
// searches c: a Searchable cell, a stopped one at Backlog pace, and a gone
// one when the run opens gone cells.
func opens(c files.Cell, t searches.Trigger, gone bool) bool {
	if c.Gone {
		return gone && c.Wanted()
	}
	return c.Searchable() || c.Paced() && laneRules[t].paced
}

// startRun turns row, queued, into a run in p's Write: the Items of its
// group within its queued scope that have a cell its trigger opens, and a
// line for each of lines, the asked ones planned and the rest skipped with
// why. A Work whose root is gone or unreachable, or a Content Type no
// Indexer serves, refuses it with a finished no_root or no_indexer run.
// Either way the row leaves its lane and keeps its due time. A group with
// nothing to open leaves its lane with no run.
func (gr *Grabs) startRun(ctx context.Context, p *pass, row db.GrabsDue, lines []indexers.Line) error {
	trigger, lane := searches.Trigger(deref(row.Trigger)), laneOf(row.Lane)
	items, err := gr.runItems(ctx, p, row, trigger)
	if err != nil {
		return err
	}
	u := db.UnqueueDueParams{ChangedAt: p.now, WorkID: row.WorkID, GroupKey: row.GroupKey}
	p.changed[row.WorkID] = true
	if len(items) == 0 {
		return unqueue(ctx, p.q, u)
	}
	refused, err := gr.refusal(ctx, p.tx, row.WorkID, lines)
	if err != nil {
		return err
	}
	run := SearchID{UUID: storage.NewUUID()}
	err = p.q.InsertSearch(ctx, db.InsertSearchParams{
		ID: run, WorkID: row.WorkID, GroupKey: &row.GroupKey, Lane: new(int64(lane)), Trigger: new(string(trigger)),
		IncludeGone: row.IncludeGone, StartedAt: p.now,
	})
	if err != nil {
		return err
	}
	for _, it := range items {
		if err := p.q.InsertSearchItem(ctx, db.InsertSearchItemParams{SearchID: run, ItemID: it}); err != nil {
			return err
		}
	}
	p.runs[run] = row.WorkID
	if refused != "" {
		if err := p.q.FinishSearch(ctx, db.FinishSearchParams{ID: run, FinishedAt: p.now, Outcome: &refused}); err != nil {
			return err
		}
		if err := unqueue(ctx, p.q, u); err != nil {
			return err
		}
		return p.q.PruneGroup(ctx, db.PruneGroupParams{WorkID: row.WorkID, GroupKey: &row.GroupKey})
	}
	for i, l := range lines {
		if err := p.q.InsertSearchIndexer(ctx, lineRow(run, i, l, p.now)); err != nil {
			return err
		}
		if l.Skipped == nil {
			p.waiting[l.Indexer]++
		}
	}
	u.SearchID = run
	return unqueue(ctx, p.q, u)
}

// unqueue takes u's row out of its lane with its queued scope.
func unqueue(ctx context.Context, q *db.Queries, u db.UnqueueDueParams) error {
	if err := q.UnqueueDue(ctx, u); err != nil {
		return err
	}
	return q.DeleteDueItems(ctx, db.DeleteDueItemsParams{WorkID: u.WorkID, GroupKey: u.GroupKey})
}

// runItems is the Items a run of row searches: those of its group, within
// its queued scope, that have a cell trigger opens, and at Backlog pace
// those with a listed cut not yet seen, which a search's sighting makes
// wanted, in tree order.
func (gr *Grabs) runItems(ctx context.Context, p *pass, row db.GrabsDue, trigger searches.Trigger) ([]works.ItemID, error) {
	cells, err := gr.cfg.Files.Cells(ctx, p.tx, row.WorkID)
	if err != nil {
		return nil, err
	}
	unseen, err := gr.unseenOf(ctx, p.tx, row.WorkID, trigger)
	if err != nil {
		return nil, err
	}
	var open []files.Cell
	for _, c := range cells {
		if opens(c, trigger, row.IncludeGone) || unseen[c.Item.ID] {
			open = append(open, c)
		}
	}
	scope, err := p.q.ListDueItems(ctx, db.ListDueItemsParams{WorkID: row.WorkID, GroupKey: row.GroupKey})
	if err != nil {
		return nil, err
	}
	if len(scope) > 0 {
		open = within(open, scope)
	}
	if len(open) == 0 {
		return nil, nil
	}
	typ, err := p.typeOf(ctx, row.WorkID)
	if err != nil {
		return nil, err
	}
	groups, err := gr.cfg.Terms[typ].Groups(ctx, p.tx, row.WorkID, itemsOf(open))
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		if g.Key == row.GroupKey {
			return g.Items, nil
		}
	}
	return nil, nil
}

// unseenOf is work's Items with a listed cut not yet seen, read in r when
// trigger runs at Backlog pace.
func (gr *Grabs) unseenOf(ctx context.Context, r storage.Reader[file], work works.WorkID, trigger searches.Trigger) (map[works.ItemID]bool, error) {
	unseen := map[works.ItemID]bool{}
	if !laneRules[trigger].paced {
		return unseen, nil
	}
	ws, err := gr.cfg.Files.Wanted(ctx, r, works.Only([]works.WorkID{work}))
	if err != nil {
		return nil, err
	}
	for _, w := range ws {
		for _, it := range w.Unseen {
			unseen[it] = true
		}
	}
	return unseen, nil
}

// refusal is the outcome a run of work refuses with, "" when it may
// start: no_root while its root is gone or unreachable, no_indexer while
// no Indexer that is on with Automatic search serves its Content Type.
func (gr *Grabs) refusal(ctx context.Context, r storage.Reader[file], work works.WorkID, lines []indexers.Line) (Outcome, error) {
	root, err := roots.OfWork(ctx, r, work)
	if errors.Is(err, ErrNoRoot) {
		return NoRoot, nil
	}
	if err != nil {
		return "", err
	}
	err = gr.cfg.Roots.Reachable(ctx, r, root)
	if _, unreachable := errors.AsType[*RootUnreachableError](err); unreachable {
		return NoRoot, nil
	}
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return NoIndexer, nil
	}
	return "", nil
}

// sent is one line in flight: its run and position, its run's Work, its
// Item for a single's query, zero for the group's, and its durable
// deadline.
type sent struct {
	run      SearchID
	position int64
	work     works.WorkID
	item     works.ItemID
	line     indexers.Line
	deadline time.Time
}

// inFlight is every line sent and not answered, read in r.
func inFlight(ctx context.Context, r storage.Reader[file]) ([]sent, error) {
	rows, err := db.New(r).ListSentLines(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]sent, len(rows))
	for i, l := range rows {
		out[i] = sent{
			run: l.SearchID, position: l.Position, work: l.WorkID, item: l.ItemID, deadline: l.DeadlineAt.Time(),
			line: indexers.Line{Indexer: l.IndexerID, Name: l.IndexerName, Query: l.QueryID},
		}
	}
	return out, nil
}

// ask asks s's Indexer until its deadline and writes the answer with its
// Releases. Asked again after a restart, it joins the job its query runs
// on the Searching Node, so a line never searches twice.
func (gr *Grabs) ask(ctx context.Context, s sent) result {
	search := indexers.Search{}
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		run, err := searchIn(ctx, r, s.run)
		if err != nil {
			return err
		}
		tree, err := gr.cfg.Works.TreeIn(ctx, r, s.work)
		if err != nil {
			return err
		}
		search.Type = tree.ContentType
		if time.Now().Before(s.deadline) {
			search.Terms, err = gr.termsOf(ctx, r, tree.ContentType, run, s.item)
		}
		return err
	})
	if errors.Is(err, errRunGone) || errors.Is(err, works.ErrNoWork) {
		return committed
	}
	if err != nil {
		return gr.lineWaits(ctx, s, err)
	}
	a, err := gr.cfg.Indexers.SearchOne(ctx, s.line, search, s.deadline)
	if ctx.Err() != nil {
		return halted
	}
	if errors.Is(err, indexers.ErrAway) {
		return waited
	}
	if err != nil {
		return gr.lineWaits(ctx, s, err)
	}
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return gr.heardLine(ctx, tx, s, a)
	})
	if err != nil {
		return gr.lineWaits(ctx, s, err)
	}
	gr.cfg.QueueChanged.Publish(ctx, QueueChanged{WorkIDs: []string{s.work.String()}})
	return committed
}

func (gr *Grabs) lineWaits(ctx context.Context, s sent, err error) result {
	if ctx.Err() != nil {
		return halted
	}
	gr.cfg.Log.Error("ask a search line", "search", s.run.String(), "indexer", s.line.Name, "err", err.Error())
	return waited
}

// searchIn is run as read in r, errRunGone once it is gone.
func searchIn(ctx context.Context, r storage.Reader[file], id SearchID) (Search, error) {
	q := db.New(r)
	row, err := q.GetSearch(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Search{}, errRunGone
	}
	if err != nil {
		return Search{}, err
	}
	runs, err := runsOf(ctx, q, []db.GrabsSearch{row}, nil)
	if err != nil {
		return Search{}, err
	}
	return runs[0], nil
}

// termsOf is how a line of run searches: item's own Terms for a single,
// the one Item's for a run of one Item, else its group's.
func (gr *Grabs) termsOf(ctx context.Context, r storage.Reader[file], typ works.TypeID, run Search, item works.ItemID) (indexers.Terms, error) {
	f := gr.cfg.Terms[typ]
	switch {
	case !item.IsZero():
		return f.SearchTerms(ctx, r, item)
	case len(run.Items) == 1:
		return f.SearchTerms(ctx, r, run.Items[0])
	}
	groups, err := f.Groups(ctx, r, run.WorkID, run.Items)
	if err != nil {
		return indexers.Terms{}, err
	}
	for _, g := range groups {
		if g.Key == run.Group {
			return f.GroupTerms(ctx, r, run.WorkID, g)
		}
	}
	return indexers.Terms{}, fmt.Errorf("search %s: its Items no longer make its group %s", run.ID, run.Group)
}

// heardLine writes a on s's line and keeps each Release it lists whole, as
// heard, until its run decides. A line already answered writes nothing.
func (gr *Grabs) heardLine(ctx context.Context, tx *storage.Tx[file], s sent, a indexers.Answer) error {
	q := db.New(tx)
	p := outcomeRow(a.Outcome)
	p.SearchID, p.Position = s.run, s.position
	n, err := q.AnswerSearchIndexer(ctx, p)
	if err != nil || n == 0 {
		return err
	}
	next, err := q.NextReleasePosition(ctx, s.run)
	if err != nil {
		return err
	}
	now := storage.Now()
	for i, r := range a.Releases {
		kept, err := encodeKept(r)
		if err != nil {
			return err
		}
		at := int(next) + i
		err = q.InsertHeard(ctx, db.InsertHeardParams{
			SearchID: s.run, Position: int64(at), Title: r.Title, IndexerID: r.Indexer, IndexerName: s.line.Name, Size: r.Size,
			Seeders: int64(r.Seeders), Release: &kept, LinkSealed: gr.cfg.Sealer.Seal([]byte(r.Offered().Reveal()), keptLinkAD(s.run, at)),
			HeardAt: now,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

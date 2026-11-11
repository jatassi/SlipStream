package grabs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	ErrNoManualSearch       = errors.New("the Work has no Manual search")
	ErrManualSearchReplaced = errors.New("another Manual search of the Work replaced this one")
	ErrQueryNeedsOneItem    = errors.New("a typed query searches one Item")
	ErrNoCell               = errors.New("the Work's Item has no cell of that Edition and Target")
)

// ManualScope is the part of a Work a Manual search covers. Cell narrows
// the plan to one cell of the one Item Part names. Query is typed text
// for one Item. Again starts a new search even when the Work's Manual
// search has this scope and query.
type ManualScope struct {
	Part  Scope
	Cell  *files.CellKey
	Query string
	Again bool
}

// storedScope is a ManualScope's part as grabs_manual.scope keeps it, its
// ids sorted so one part always reads the same.
type storedScope struct {
	Levels []string    `json:"levels,omitempty"`
	Items  []string    `json:"items,omitempty"`
	Cell   *storedCell `json:"cell,omitempty"`
}

type storedCell struct {
	Item    string `json:"item"`
	Edition string `json:"edition"`
	Target  string `json:"target"`
}

func (s ManualScope) stored() (string, error) {
	var out storedScope
	if p := s.Part.part; p != nil {
		for _, l := range p.Levels {
			out.Levels = append(out.Levels, l.String())
		}
		for _, it := range p.Items {
			out.Items = append(out.Items, it.String())
		}
		slices.Sort(out.Levels)
		slices.Sort(out.Items)
	}
	if c := s.Cell; c != nil {
		out.Cell = &storedCell{Item: c.Item.String(), Edition: string(c.Edition), Target: c.Target.String()}
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", fmt.Errorf("encode a Manual search's scope: %w", err)
	}
	return string(b), nil
}

// items is the Items of t that s searches: the Work's live Items, or
// those its part names.
func (s ManualScope) items(t works.Tree) ([]works.Item, error) {
	if s.Part.part != nil {
		return t.ItemsWithin(*s.Part.part)
	}
	var out []works.Item
	for _, it := range t.Items {
		if it.Live() {
			out = append(out, it)
		}
	}
	return out, nil
}

// StartManualSearch records a Manual search of work within s: one run per
// Item in scope, monitoring and release dates aside, each asking every on
// Indexer that can search, Automatic search off included. Unless s.Again,
// a Manual search of work with s's scope and query is answered, started
// false. Otherwise it replaces the Work's Manual search in the same Write
// and, once committed, stops the runs it replaced. It fails with
// ErrReconciling, ErrQueryNeedsOneItem, ErrNoCell, ErrNotSearchable,
// ErrNoIndexer, ErrNoRoot, ErrNoWork and *works.NotInWorkError. An
// unreachable root doesn't stop it.
func (gr *Grabs) StartManualSearch(ctx context.Context, by identity.Admin, work works.WorkID, s ManualScope) (id ManualID, started bool, err error) {
	scope, err := s.stored()
	if err != nil {
		return ManualID{}, false, err
	}
	var replaced, runs []SearchID
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		tree, items, err := gr.manualItems(ctx, tx, work, s)
		if err != nil {
			return err
		}
		var kept bool
		if id, kept, replaced, err = dropManual(ctx, db.New(tx), work, s, scope); err != nil || kept {
			return err
		}
		id, started = ManualID{UUID: storage.NewUUID()}, true
		runs, err = gr.insertManual(ctx, tx, tree, items, id, scope, s.Query, actorOf(by))
		return err
	})
	if err != nil || !started {
		return id, false, err
	}
	gr.runs.stop(replaced...)
	for _, run := range runs {
		gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: run.String(), WorkID: work.String(), ManualSearchID: new(id.String())})
	}
	gr.wake()
	return id, true, nil
}

// manualItems reads, in r, work's tree and the Items s searches, or why
// a Manual search of them can't start.
func (gr *Grabs) manualItems(ctx context.Context, r storage.Reader[file], work works.WorkID, s ManualScope) (works.Tree, []works.Item, error) {
	reconciling, err := gr.cfg.Reconciling(ctx, r)
	if err != nil {
		return works.Tree{}, nil, err
	}
	if reconciling {
		return works.Tree{}, nil, ErrReconciling
	}
	tree, err := gr.cfg.Works.TreeIn(ctx, r, work)
	if err != nil {
		return works.Tree{}, nil, err
	}
	items, err := s.items(tree)
	if err != nil {
		return works.Tree{}, nil, err
	}
	if err := gr.hasCell(ctx, r, work, s.Cell); err != nil {
		return works.Tree{}, nil, err
	}
	if s.Query != "" && len(items) != 1 {
		return works.Tree{}, nil, ErrQueryNeedsOneItem
	}
	_, err = gr.searchable(ctx, r, tree)
	return tree, items, err
}

// dropManual deletes work's Manual search and returns its runs, unless it
// has s's scope, stored as scope, and query and s isn't Again: then it is
// kept, ok is true and its id returned.
func dropManual(ctx context.Context, q *db.Queries, work works.WorkID, s ManualScope, scope string) (kept ManualID, ok bool, runs []SearchID, err error) {
	old, err := q.GetManualOfWork(ctx, work)
	if errors.Is(err, storage.ErrNoRows) {
		return ManualID{}, false, nil, nil
	}
	if err != nil {
		return ManualID{}, false, nil, err
	}
	if !s.Again && old.Scope == scope && deref(old.Query) == s.Query {
		return old.ID, true, nil, nil
	}
	if runs, err = q.ListRunsOfManual(ctx, old.ID); err != nil {
		return ManualID{}, false, nil, err
	}
	return ManualID{}, false, runs, q.DeleteManual(ctx, old.ID)
}

// insertManual inserts Manual search id of tree, started by a with scope
// and query, and a run of each of items.
func (gr *Grabs) insertManual(ctx context.Context, tx *storage.Tx[file], tree works.Tree, items []works.Item, id ManualID, scope, query string, a Actor) ([]SearchID, error) {
	now := storage.Now()
	p := db.InsertManualParams{ID: id, WorkID: tree.ID, Scope: scope, Query: orNil(query), StartedAt: now}
	if a.Account != nil {
		p.StartedBy, p.StartedByName = *a.Account, &a.Name
	}
	if err := db.New(tx).InsertManual(ctx, p); err != nil {
		return nil, err
	}
	runs := make([]SearchID, len(items))
	for i, it := range items {
		var err error
		if runs[i], err = gr.planManual(ctx, tx, tree, it.ID, id, now); err != nil {
			return nil, err
		}
	}
	return runs, nil
}

// hasCell is ErrNoCell when cell is set and work has no such cell.
func (gr *Grabs) hasCell(ctx context.Context, r storage.Reader[file], work works.WorkID, cell *files.CellKey) error {
	if cell == nil {
		return nil
	}
	cells, err := gr.cfg.Files.Cells(ctx, r, work)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(cells, func(c files.Cell) bool {
		return c.Item.ID == cell.Item && c.Edition == cell.Edition && c.Target.ID == cell.Target
	}) {
		return ErrNoCell
	}
	return nil
}

// planManual records manual's run of item with the lines a Manual search
// plans: every on Indexer that can search, an RSS feed skipped.
func (gr *Grabs) planManual(ctx context.Context, tx *storage.Tx[file], tree works.Tree, item works.ItemID, manual ManualID, started storage.Time) (SearchID, error) {
	q := db.New(tx)
	id := SearchID{UUID: storage.NewUUID()}
	if err := q.InsertManualRun(ctx, db.InsertManualRunParams{ID: id, WorkID: tree.ID, ItemID: item, ManualID: manual, StartedAt: started}); err != nil {
		return SearchID{}, err
	}
	lines, err := gr.cfg.Indexers.Plan(ctx, tx, tree.ContentType, indexers.ManualSearch)
	if err != nil {
		return SearchID{}, err
	}
	for i, l := range lines {
		if err := q.InsertSearchIndexer(ctx, lineRow(id, i, l)); err != nil {
			return SearchID{}, err
		}
	}
	return id, nil
}

// runs is the cancel of each Manual run's goroutine.
type runs struct {
	mu     sync.Mutex
	cancel map[SearchID]context.CancelFunc
}

// track is ctx for run id's goroutine, which stop cancels, and its end.
func (r *runs) track(ctx context.Context, id SearchID) (tracked context.Context, end func()) {
	tracked, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.cancel[id] = cancel
	r.mu.Unlock()
	return tracked, func() {
		r.mu.Lock()
		delete(r.cancel, id)
		r.mu.Unlock()
		cancel()
	}
}

// stop cancels the goroutine of each of ids that runs.
func (r *runs) stop(ids ...SearchID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, id := range ids {
		if cancel, ok := r.cancel[id]; ok {
			cancel()
		}
	}
}

// errRunGone is a run that a Search again replaced or whose Work was
// deleted: it ends quietly.
var errRunGone = errors.New("the search run is gone")

// listen is a Manual run's goroutine: it asks the run's lines with no
// outcome and stores each answer as it lands. A restart starts it again,
// and it asks only the lines still unanswered.
func (gr *Grabs) listen(ctx context.Context, run Search) result {
	ctx, done := gr.runs.track(ctx, run.ID)
	defer done()
	a, err := gr.asking(ctx, run)
	switch {
	case err != nil:
	case len(a.search.Asked) == 0:
		err = gr.heard(ctx, run, nil)
	default:
		_, err = gr.cfg.Indexers.Search(ctx, a.search, func(ans indexers.Answer) error {
			return gr.heard(ctx, run, &answered{line: a.lines[ans.Indexer], answer: ans, assessed: assessAll(a.judging, listedOf(ans.Releases))})
		})
	}
	switch {
	case errors.Is(err, errRunGone), ctx.Err() != nil:
		return halted
	case err != nil:
		return gr.searchWaits(ctx, run, err)
	}
	return committed
}

// asking is what a Manual run asks and how it reads the answers: the
// search of its lines with no outcome, those lines by Indexer, and the
// judging of the Releases they list.
type asking struct {
	search  indexers.Search
	lines   map[indexers.ID]db.GrabsSearchIndexer
	judging judging
}

// asking reads run's asking in one Read, errRunGone once run is gone.
func (gr *Grabs) asking(ctx context.Context, run Search) (asking, error) {
	a := asking{lines: map[indexers.ID]db.GrabsSearchIndexer{}}
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		m, err := q.GetManual(ctx, run.Manual)
		if errors.Is(err, storage.ErrNoRows) {
			return errRunGone
		}
		if err != nil {
			return err
		}
		if _, err := q.GetSearch(ctx, run.ID); errors.Is(err, storage.ErrNoRows) {
			return errRunGone
		} else if err != nil {
			return err
		}
		tree, err := gr.cfg.Works.TreeIn(ctx, r, run.WorkID)
		if err != nil {
			return err
		}
		a.search.Type, a.search.Text = tree.ContentType, deref(m.Query)
		if a.search.Terms, err = gr.cfg.Terms[tree.ContentType].SearchTerms(ctx, r, run.ItemID); err != nil {
			return err
		}
		if a.judging, err = gr.judgingOf(ctx, r, tree.ContentType, run, a.search.Terms.Expect); err != nil {
			return err
		}
		rows, err := q.ListSearchIndexersOfSearch(ctx, run.ID)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if row.Outcome == nil {
				a.lines[row.IndexerID] = row
				a.search.Asked = append(a.search.Asked, indexers.Line{Indexer: row.IndexerID, Name: row.IndexerName, Query: row.QueryID})
			}
		}
		return nil
	})
	return a, err
}

// answered is one Answer, its line and its Releases assessed.
type answered struct {
	line     db.GrabsSearchIndexer
	answer   indexers.Answer
	assessed []assessed
}

// heard is one answer's Write, nil to finish a run whose lines were all
// skipped. It fails with errRunGone once run is gone.
func (gr *Grabs) heard(ctx context.Context, run Search, h *answered) error {
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if _, err := q.GetSearch(ctx, run.ID); errors.Is(err, storage.ErrNoRows) {
			return errRunGone
		} else if err != nil {
			return err
		}
		if h != nil {
			if err := gr.store(ctx, tx, run.ID, *h); err != nil {
				return err
			}
		}
		return q.FinishHeardSearch(ctx, db.FinishHeardSearchParams{ID: run.ID, FinishedAt: storage.Now()})
	})
	if err != nil {
		return err
	}
	gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: run.ID.String(), WorkID: run.WorkID.String(), ManualSearchID: new(run.Manual.String())})
	return nil
}

// store records h on run's line and appends its Releases, with their
// sightings and format timeouts. An answer the line already has stores
// nothing, so a repeated answer is one answer.
func (gr *Grabs) store(ctx context.Context, tx *storage.Tx[file], run SearchID, h answered) error {
	q := db.New(tx)
	p := outcomeRow(h.answer.Outcome)
	p.SearchID, p.Position = run, h.line.Position
	n, err := q.AnswerSearchIndexer(ctx, p)
	if err != nil || n == 0 {
		return err
	}
	next, err := q.NextReleasePosition(ctx, run)
	if err != nil {
		return err
	}
	if err := gr.storeListed(ctx, q, run, int(next), h.line.IndexerName, h.assessed); err != nil {
		return err
	}
	if err := sight(ctx, tx, h.assessed); err != nil {
		return err
	}
	results := make([]formats.Result, len(h.assessed))
	for i, a := range h.assessed {
		results[i] = a.formats
	}
	return targets.AddTimeouts(ctx, tx, results)
}

// storeListed inserts as, listed by indexer, at positions from next on
// with outcome listed: each one's kept Release, its link sealed to its
// run and position, its title's reading and the formats it matched.
func (gr *Grabs) storeListed(ctx context.Context, q *db.Queries, run SearchID, next int, indexer string, as []assessed) error {
	for _, a := range as {
		r, at := a.Release, next+a.at
		kept, err := encodeKept(r)
		if err != nil {
			return err
		}
		matched := make([]string, len(a.formats.Matched))
		for i, f := range a.formats.Matched {
			matched[i] = f.String()
		}
		fs, err := json.Marshal(matched)
		if err != nil {
			return fmt.Errorf("encode a listed Release's formats: %w", err)
		}
		err = q.InsertListed(ctx, db.InsertListedParams{
			SearchID: run, Position: int64(at), Title: r.Title, IndexerID: r.Indexer, IndexerName: indexer, Size: r.Size,
			Seeders: int64(r.Seeders), Formats: string(fs), Release: &kept,
			LinkSealed: gr.cfg.Sealer.Seal([]byte(r.Offered().Reveal()), keptLinkAD(run, at)), Read: &db.Reading{Read: a.read},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

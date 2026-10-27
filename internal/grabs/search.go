package grabs

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Search is one search run: one Item searched on every Indexer it planned
// to ask.
type Search struct {
	ID      SearchID
	WorkID  works.WorkID
	ItemID  works.ItemID
	Started time.Time
	// Indexers are the Indexers the run asked or skipped, by priority.
	Indexers []SearchIndexer
	// Finished, Outcome, Candidates and Refused are set once the run
	// finished, and GrabID and Notes, its Grab's, while it has one: the
	// decision's, or the rerun's once a dropped Grab reran it.
	Finished   time.Time
	Outcome    Outcome
	Candidates int
	Refused    []Refusal
	GrabID     GrabID
	Notes      []Reason
}

// SearchIndexer is one Indexer a run asked or skipped.
type SearchIndexer struct {
	// Indexer is zero once the Indexer is deleted.
	Indexer indexers.ID
	// Name is the Indexer's name when the run was planned.
	Name string
	// Outcome is nil while the Indexer's job runs.
	Outcome *indexers.Outcome
	// Line is Outcome in words.
	Line string
}

func searchOf(row db.GrabsSearch, refused []Refusal, lines []SearchIndexer) (Search, error) {
	s := Search{
		ID: row.ID, WorkID: row.WorkID, ItemID: row.ItemID, Started: row.StartedAt.Time(), Indexers: lines,
		Finished: row.FinishedAt.Time(), Candidates: int(row.Candidates), Refused: refused, GrabID: row.GrabID,
	}
	if row.Outcome != nil {
		s.Outcome = *row.Outcome
	}
	if row.Notes != nil {
		notes, err := decodeReasons(*row.Notes)
		if err != nil {
			return Search{}, fmt.Errorf("search %s: %w", row.ID, err)
		}
		s.Notes = notes
	}
	return s, nil
}

// lineRow is line l of run, at position.
func lineRow(run SearchID, position int, l indexers.Line) db.InsertSearchIndexerParams {
	p := db.InsertSearchIndexerParams{SearchID: run, Position: int64(position), IndexerID: l.Indexer, IndexerName: l.Name, QueryID: l.Query}
	if l.Skipped != nil {
		a := outcomeRow(*l.Skipped)
		p.Outcome, p.Releases, p.UntilAt, p.SinceAt, p.NodeID, p.Lapsed, p.Cause, p.Text =
			a.Outcome, a.Releases, a.UntilAt, a.SinceAt, a.NodeID, a.Lapsed, a.Cause, a.Text
	}
	return p
}

// outcomeRow is o as grabs_search_indexer keeps it.
func outcomeRow(o indexers.Outcome) db.AnswerSearchIndexerParams {
	p := db.AnswerSearchIndexerParams{Outcome: &o.Kind, UntilAt: storage.FromTime(o.Until), SinceAt: storage.FromTime(o.Since), NodeID: o.Node}
	switch o.Kind {
	case indexers.Found, indexers.NoReleases:
		p.Releases = new(int64(o.Releases))
	case indexers.Waiting:
		p.Lapsed = &o.Lapsed
	case indexers.IsStopped:
		p.Cause, p.Text = &o.Cause, &o.Text
	case indexers.IsLimited, indexers.IsFailing, indexers.Late, indexers.LimitReached, indexers.NoCategories, indexers.Removed:
	}
	return p
}

// outcomeOf is row's outcome, nil while its job runs.
func outcomeOf(row db.GrabsSearchIndexer) *indexers.Outcome {
	if row.Outcome == nil {
		return nil
	}
	o := &indexers.Outcome{Kind: *row.Outcome, Until: row.UntilAt.Time(), Since: row.SinceAt.Time(), Node: row.NodeID, Text: deref(row.Text)}
	if row.Releases != nil {
		o.Releases = int(*row.Releases)
	}
	if row.Lapsed != nil {
		o.Lapsed = *row.Lapsed
	}
	if row.Cause != nil {
		o.Cause = *row.Cause
	}
	return o
}

// SearchWork records a search run for each Item of work with a wanted cell
// no unfinished Grab holds, each with the Indexers it asks and skips and a
// query reserved on each it asks, and returns their ids. Run searches
// them.
func (gr *Grabs) SearchWork(ctx context.Context, _ identity.Admin, work works.WorkID) ([]SearchID, error) {
	var ids []SearchID
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		tree, err := gr.cfg.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return err
		}
		if err := gr.searchable(ctx, tx, tree); err != nil {
			return err
		}
		open, err := gr.openCells(ctx, tx, work)
		if err != nil {
			return err
		}
		started := storage.Now()
		for _, it := range tree.Items {
			if !slices.ContainsFunc(open, func(c files.Cell) bool { return c.Item.ID == it.ID }) {
				continue
			}
			id, err := gr.plan(ctx, tx, tree, it.ID, started)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		if len(ids) == 0 {
			return ErrNothingWanted
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: id.String(), WorkID: work.String()})
	}
	gr.wake()
	return ids, nil
}

// plan records a search run of item with the lines Plan read and
// reserved.
func (gr *Grabs) plan(ctx context.Context, tx *storage.Tx[file], tree works.Tree, item works.ItemID, started storage.Time) (SearchID, error) {
	q := db.New(tx)
	id := SearchID{UUID: storage.NewUUID()}
	if err := q.InsertSearch(ctx, db.InsertSearchParams{ID: id, WorkID: tree.ID, ItemID: item, StartedAt: started}); err != nil {
		return SearchID{}, err
	}
	lines, err := gr.cfg.Indexers.Plan(ctx, tx, tree.ContentType)
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

// searchable is why tree can't be searched, nil when it can: no Indexer
// exists at all, or the Work is rootless, or its root's Node is offline or
// written off.
func (gr *Grabs) searchable(ctx context.Context, r storage.Reader[file], tree works.Tree) error {
	if len(tree.Items) > 0 {
		terms, err := gr.cfg.Terms[tree.ContentType].SearchTerms(ctx, r, tree.Items[0].ID)
		if err != nil {
			return err
		}
		if terms.Mode == "" && terms.Query == "" && len(terms.IDs) == 0 {
			return ErrNotSearchable
		}
	}
	all, err := indexers.All(ctx, r)
	if err != nil {
		return err
	}
	if len(all) == 0 {
		return ErrNoIndexer
	}
	root, err := roots.OfWork(ctx, r, tree.ID)
	if err != nil {
		return err
	}
	return gr.cfg.Roots.Reachable(ctx, r, root)
}

// openCells is work's wanted cells that no unfinished Grab holds, in tree
// and rank order: the cells a search may fill.
func (gr *Grabs) openCells(ctx context.Context, r storage.Reader[file], work works.WorkID) ([]files.Cell, error) {
	cells, err := gr.cfg.Files.Cells(ctx, r, work)
	if err != nil {
		return nil, err
	}
	rows, err := db.New(r).ListTargetsOfWork(ctx, work)
	if err != nil {
		return nil, err
	}
	held := map[files.CellKey]bool{}
	for _, row := range rows {
		if row.Stage != Imported && row.Stage != Failed {
			held[files.CellKey{Item: row.ItemID, Edition: row.Edition, Target: row.TargetID}] = true
		}
	}
	var open []files.Cell
	for _, c := range cells {
		if c.Wanted() && !held[files.CellKey{Item: c.Item.ID, Edition: c.Edition, Target: c.Target.ID}] {
			open = append(open, c)
		}
	}
	return open, nil
}

// search asks every Indexer run planned to ask, then finishes the run in
// one Write with the Grab its decision took. A search never waits on an
// Indexer: each one that couldn't answer is its line's outcome, and the
// Admin searches again.
func (gr *Grabs) search(ctx context.Context, run Search) result {
	var s indexers.Search
	var asked []db.GrabsSearchIndexer
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		tree, err := gr.cfg.Works.TreeIn(ctx, r, run.WorkID)
		if err != nil {
			return err
		}
		s.Type = tree.ContentType
		if s.Terms, err = gr.cfg.Terms[tree.ContentType].SearchTerms(ctx, r, run.ItemID); err != nil {
			return err
		}
		rows, err := db.New(r).ListSearchIndexersOfSearch(ctx, run.ID)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if row.Outcome == nil {
				asked = append(asked, row)
				s.Asked = append(s.Asked, indexers.Line{Indexer: row.IndexerID, Name: row.IndexerName, Query: row.QueryID})
			}
		}
		return nil
	})
	if err != nil {
		return gr.searchWaits(ctx, run, err)
	}
	answers, err := gr.cfg.Indexers.Search(ctx, s)
	if ctx.Err() != nil {
		return halted
	}
	if err != nil {
		return gr.searchWaits(ctx, run, err)
	}
	var grab *Grab
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		grab, err = gr.finish(ctx, tx, run, s.Terms, asked, answers)
		return err
	})
	if err != nil {
		return gr.searchWaits(ctx, run, err)
	}
	gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: run.ID.String(), WorkID: run.WorkID.String()})
	if grab != nil {
		gr.cfg.Indexers.Taken(ctx, grab.Ticket)
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: grab.ID.String(), WorkID: grab.WorkID.String()})
	}
	return committed
}

// finish ends run in tx with each asked line's answer and decides once
// over every Release they found, keeping the ones its Grab didn't take
// for a rerun. It returns the Grab, nil when the run takes none.
func (gr *Grabs) finish(ctx context.Context, tx *storage.Tx[file], run Search, terms indexers.Terms, asked []db.GrabsSearchIndexer, answers []indexers.Answer) (*Grab, error) {
	q := db.New(tx)
	var releases []indexers.Release
	for i, a := range answers {
		p := outcomeRow(a.Outcome)
		p.SearchID, p.Position = run.ID, asked[i].Position
		if err := q.AnswerSearchIndexer(ctx, p); err != nil {
			return nil, err
		}
		releases = append(releases, a.Releases...)
	}
	candidates, err := gr.candidates(ctx, tx, run, releases)
	if err != nil {
		return nil, err
	}
	if err := sight(ctx, tx, run, terms, candidates); err != nil {
		return nil, err
	}
	kept := dedupe(candidates)
	picks, refused, err := gr.decision(ctx, tx, run, terms, kept)
	if err != nil {
		return nil, err
	}
	for i, r := range refused {
		reasons, err := encodeReasons(r.Reasons)
		if err != nil {
			return nil, err
		}
		if err := q.InsertRefusal(ctx, db.InsertRefusalParams{SearchID: run.ID, Position: int64(i), Title: r.Title, Reasons: reasons}); err != nil {
			return nil, err
		}
	}
	finished := storage.Now()
	grab, pick, err := gr.take(ctx, tx, run, picks, finished)
	if err != nil {
		return nil, err
	}
	finish := db.FinishSearchParams{ID: run.ID, FinishedAt: finished, Outcome: new(NothingAcceptable), Candidates: int64(len(kept))}
	if grab == nil {
		return nil, q.FinishSearch(ctx, finish)
	}
	notes, err := encodeReasons(pick.Notes)
	if err != nil {
		return nil, err
	}
	finish.Outcome, finish.GrabID, finish.Notes = new(Grabbed), grab.ID, &notes
	for _, c := range candidates {
		if c.at == pick.at {
			continue
		}
		if err := gr.keep(ctx, q, run.ID, c); err != nil {
			return nil, err
		}
	}
	return grab, q.FinishSearch(ctx, finish)
}

// candidates is releases as run's Item's candidates, each at its position
// in the run.
func (gr *Grabs) candidates(ctx context.Context, r storage.Reader[file], run Search, releases []indexers.Release) ([]candidate, error) {
	tree, err := gr.cfg.Works.TreeIn(ctx, r, run.WorkID)
	if err != nil {
		return nil, err
	}
	known, err := gr.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return nil, err
	}
	timings, err := gr.cfg.Targets.TimingsIn(ctx, r, tree.ContentType, []works.WorkID{run.WorkID})
	if err != nil {
		return nil, err
	}
	out := make([]candidate, len(releases))
	for i, rel := range releases {
		out[i] = candidateOf(rel, known.For(tree.ContentType), run.ItemID, timings[run.ItemID].Runtime)
		out[i].at = i
	}
	return out, nil
}

// sight records a sighting of the cut of every listed release of run's
// Work, so a cut seen now is wanted now.
func sight(ctx context.Context, tx *storage.Tx[file], run Search, terms indexers.Terms, candidates []candidate) error {
	for _, c := range candidates {
		if wrongWork(c.Release, terms) {
			continue
		}
		if err := targets.See(ctx, tx, run.ItemID, c.Facts.Cut, releaseKey(c.Release)); err != nil {
			return err
		}
	}
	return nil
}

// decision judges candidates against run's Item's open cells: the
// grabbable in rank order and the refused.
func (gr *Grabs) decision(ctx context.Context, r storage.Reader[file], run Search, terms indexers.Terms, candidates []candidate) ([]Pick, []Refusal, error) {
	open, err := gr.openCells(ctx, r, run.WorkID)
	if err != nil {
		return nil, nil, err
	}
	var item []files.Cell
	for _, c := range open {
		if c.Item.ID == run.ItemID {
			item = append(item, c)
		}
	}
	picks, refused := decide(item, terms, candidates)
	return picks, refused, nil
}

// take grabs the first of picks whose Indexer takes it, nil when none
// does: an Indexer that is stopped, at its grab limit or deleted would
// make the Grab wait for the Admin.
func (gr *Grabs) take(ctx context.Context, tx *storage.Tx[file], run Search, picks []Pick, now storage.Time) (*Grab, Pick, error) {
	for _, p := range picks {
		t, err := gr.cfg.Indexers.Take(ctx, tx, p.Release)
		if errors.As(err, new(*indexers.NotTakenError)) {
			continue
		}
		if err != nil {
			return nil, Pick{}, err
		}
		g, err := gr.insertGrab(ctx, tx, run, p, t, now)
		return &g, p, err
	}
	return nil, Pick{}, nil
}

// releaseKey is what a sighting keys it by: its info hash in hex, else its
// title.
func releaseKey(r indexers.Release) string {
	if r.InfoHash != nil {
		return r.InfoHash.String()
	}
	return r.Title
}

func (gr *Grabs) searchWaits(ctx context.Context, run Search, err error) result {
	if ctx.Err() != nil {
		return halted
	}
	gr.cfg.Log.Error("run a search", "search", run.ID.String(), "err", err.Error())
	return waited
}

// insertGrab inserts the Grab of pick that t readied, at fetching on the
// Indexer's home Node.
func (gr *Grabs) insertGrab(ctx context.Context, tx *storage.Tx[file], run Search, pick Pick, t indexers.Ticket, now storage.Time) (Grab, error) {
	q := db.New(tx)
	r := pick.Release
	g := Grab{
		ID: GrabID{UUID: storage.NewUUID()}, WorkID: run.WorkID, ItemID: run.ItemID, Edition: pick.Edition, Targets: pick.Targets,
		Release: Release{Title: r.Title, Size: r.Size, InfoHash: r.InfoHash}, Ticket: t, Stage: Fetching, Created: now.Time(),
	}
	idx, err := indexers.Get(ctx, tx, t.Indexer)
	if err != nil {
		return Grab{}, err
	}
	g.Node = idx.Home
	if link := r.Offered(); link.IsMagnet() {
		g.Release.magnet = link.Reveal()
	} else {
		g.Release.link = gr.cfg.Sealer.Seal([]byte(link.Reveal()), linkColumn.Row(g.ID.String()))
	}
	var hint []byte
	if h := g.Release.InfoHash; h != nil {
		hint = h[:]
	}
	err = q.InsertGrab(ctx, db.InsertGrabParams{
		ID: g.ID, WorkID: g.WorkID, ItemID: g.ItemID, Edition: g.Edition, IndexerID: t.Indexer, TrackerID: t.Tracker, NodeID: g.Node,
		ReleaseTitle: g.Release.Title, ReleaseSize: g.Release.Size, ReleaseLinkSealed: g.Release.link, ReleaseMagnet: orNil(g.Release.magnet),
		ReleaseInfoHash: hint, FetchQueryID: t.Query, CreatedAt: now,
		MinRatio: orNil(r.Floor.MinRatio), MinSeedSeconds: orNil(int64(r.Floor.MinSeed.Seconds())),
		DownloadFactor: new(r.Factors.Download), UploadFactor: new(r.Factors.Upload),
	})
	if err != nil {
		return Grab{}, err
	}
	for i, target := range g.Targets {
		if err := q.InsertTarget(ctx, db.InsertTargetParams{GrabID: g.ID, Position: int64(i), TargetID: target}); err != nil {
			return Grab{}, err
		}
	}
	return g, q.InsertStage(ctx, db.InsertStageParams{GrabID: g.ID, Stage: Fetching, EnteredAt: now})
}

// Searches lists work's search runs, newest first.
func (gr *Grabs) Searches(ctx context.Context, _ identity.Admin, work works.WorkID) ([]Search, error) {
	var out []Search
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		tree, err := gr.cfg.Works.TreeIn(ctx, r, work)
		if err != nil {
			return err
		}
		q := db.New(r)
		rows, err := q.ListSearchesOfWork(ctx, work)
		if err != nil {
			return err
		}
		refusals, err := q.ListRefusalsOfWork(ctx, work)
		if err != nil {
			return err
		}
		asked, err := q.ListSearchIndexersOfWork(ctx, work)
		if err != nil {
			return err
		}
		lines := map[SearchID][]SearchIndexer{}
		for _, row := range asked {
			o := outcomeOf(row)
			lines[row.SearchID] = append(lines[row.SearchID], SearchIndexer{
				Indexer: row.IndexerID, Name: row.IndexerName, Outcome: o, Line: gr.cfg.Indexers.Line(row.IndexerName, tree.ContentType, o),
			})
		}
		refused := map[SearchID][]Refusal{}
		for _, row := range refusals {
			reasons, err := decodeReasons(row.Reasons)
			if err != nil {
				return fmt.Errorf("search %s: %w", row.SearchID, err)
			}
			refused[row.SearchID] = append(refused[row.SearchID], Refusal{Title: row.Title, Reasons: reasons})
		}
		for _, row := range rows {
			s, err := searchOf(row, refused[row.ID], lines[row.ID])
			if err != nil {
				return err
			}
			out = append(out, s)
		}
		return nil
	})
	return out, err
}

func orNil[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

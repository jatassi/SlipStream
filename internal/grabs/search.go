package grabs

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
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
	ID     SearchID
	WorkID works.WorkID
	ItemID works.ItemID
	// Manual is the Manual search the run lists Releases for, zero for an
	// automatic run.
	Manual  ManualID
	Started time.Time
	// Indexers are the Indexers the run asked or skipped, by priority.
	Indexers []SearchIndexer
	// Finished, Outcome and Candidates are set once the run finished, and
	// GrabID and Notes, its Grab's, while it has one: the decision's, or
	// the rerun's once a dropped Grab reran it.
	Finished   time.Time
	Outcome    Outcome
	Candidates int
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

func searchOf(row db.GrabsSearch, lines []SearchIndexer) (Search, error) {
	s := Search{
		ID: row.ID, WorkID: row.WorkID, ItemID: row.ItemID, Manual: row.ManualID, Started: row.StartedAt.Time(), Indexers: lines,
		Finished: row.FinishedAt.Time(), Candidates: int(row.Candidates), GrabID: row.GrabID,
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
	case indexers.IsLimited, indexers.IsFailing, indexers.Late, indexers.LimitReached, indexers.NoCategories, indexers.Removed,
		indexers.RSSOnly:
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

// Scope is the part of a Work a search covers. The zero Scope is the
// whole Work.
type Scope struct{ part *works.Within }

// Within is the Items within part: the live Items under its levels and
// each Item it names.
func Within(part works.Within) Scope { return Scope{part: &part} }

func (s Scope) items(t works.Tree) ([]works.Item, error) {
	if s.part == nil {
		return t.Items, nil
	}
	return t.ItemsWithin(*s.part)
}

// SearchWork records a search run for each Item of work within scope with
// a wanted cell, each with the Indexers it asks and skips and a query
// reserved on each it asks, and returns their ids.
// Each run reserves its own queries, so an Indexer's search limit skips it
// for the runs past the limit. Run searches them. An id scope names that
// work doesn't have is a *works.NotInWorkError.
func (gr *Grabs) SearchWork(ctx context.Context, _ identity.Admin, work works.WorkID, scope Scope) ([]SearchID, error) {
	var ids []SearchID
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		tree, err := gr.cfg.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return err
		}
		items, err := scope.items(tree)
		if err != nil {
			return err
		}
		root, err := gr.searchable(ctx, tx, tree)
		if err != nil {
			return err
		}
		if err := gr.cfg.Roots.Reachable(ctx, tx, root); err != nil {
			return err
		}
		open, err := gr.openCells(ctx, tx, work)
		if err != nil {
			return err
		}
		started := storage.Now()
		for _, it := range wantedOf(items, open) {
			id, err := gr.plan(ctx, tx, tree, it, started)
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

// wantedOf is each of items that has a cell in open, in items' order.
func wantedOf(items []works.Item, open []files.Cell) []works.ItemID {
	wanted := make(map[works.ItemID]bool, len(open))
	for _, c := range open {
		wanted[c.Item.ID] = true
	}
	var out []works.ItemID
	for _, it := range items {
		if wanted[it.ID] {
			out = append(out, it.ID)
		}
	}
	return out
}

// plan records a search run of item with the lines Plan read and
// reserved.
func (gr *Grabs) plan(ctx context.Context, tx *storage.Tx[file], tree works.Tree, item works.ItemID, started storage.Time) (SearchID, error) {
	q := db.New(tx)
	id := SearchID{UUID: storage.NewUUID()}
	if err := q.InsertSearch(ctx, db.InsertSearchParams{ID: id, WorkID: tree.ID, ItemID: item, StartedAt: started}); err != nil {
		return SearchID{}, err
	}
	lines, err := gr.cfg.Indexers.Plan(ctx, tx, tree.ContentType, indexers.AutoSearch)
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

// searchable is why tree can't be searched, nil when it can, and its
// root: its Content Type can't cover it, no Indexer exists at all, or the
// Work is rootless.
func (gr *Grabs) searchable(ctx context.Context, r storage.Reader[file], tree works.Tree) (roots.Root, error) {
	if len(tree.Items) > 0 {
		terms, err := gr.cfg.Terms[tree.ContentType].SearchTerms(ctx, r, tree.Items[0].ID)
		if err != nil {
			return roots.Root{}, err
		}
		if terms.Covers == nil {
			return roots.Root{}, ErrNotSearchable
		}
	}
	all, err := indexers.All(ctx, r)
	if err != nil {
		return roots.Root{}, err
	}
	if len(all) == 0 {
		return roots.Root{}, ErrNoIndexer
	}
	return roots.OfWork(ctx, r, tree.ID)
}

// openCells is work's wanted cells, in tree and rank order: the cells a
// search may fill.
func (gr *Grabs) openCells(ctx context.Context, r storage.Reader[file], work works.WorkID) ([]files.Cell, error) {
	cells, err := gr.cfg.Files.Cells(ctx, r, work)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(cells, func(c files.Cell) bool { return !c.Wanted() }), nil
}

// search asks every Indexer run planned to ask, assesses every Release
// they list, then finishes the run in one Write with the Grab its
// decision took. A search never waits on an Indexer: each one that
// couldn't answer is its line's outcome, and the Admin searches again.
func (gr *Grabs) search(ctx context.Context, run Search) result {
	var s indexers.Search
	var asked []db.GrabsSearchIndexer
	var j judging
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		tree, err := gr.cfg.Works.TreeIn(ctx, r, run.WorkID)
		if err != nil {
			return err
		}
		s.Type = tree.ContentType
		if s.Terms, err = gr.cfg.Terms[tree.ContentType].SearchTerms(ctx, r, run.ItemID); err != nil {
			return err
		}
		if j, err = gr.judgingOf(ctx, r, tree.ContentType, run, s.Terms.Expect); err != nil {
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
	answers, err := gr.cfg.Indexers.Search(ctx, s, nil)
	if ctx.Err() != nil {
		return halted
	}
	if err != nil {
		return gr.searchWaits(ctx, run, err)
	}
	var releases []indexers.Release
	for _, a := range answers {
		releases = append(releases, a.Releases...)
	}
	as := assessAll(j, dedupe(listedOf(releases)))
	var grab *Grab
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		grab, err = gr.finish(ctx, tx, run, asked, answers, as)
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
// over as, the Releases they listed, assessed. It stores every Release
// with its outcome, keeping what a rerun needs while the run's Grab
// fetches, and deletes the rows of the Item's older runs. It returns the
// Grab, nil when the run takes none.
func (gr *Grabs) finish(ctx context.Context, tx *storage.Tx[file], run Search, asked []db.GrabsSearchIndexer, answers []indexers.Answer, as []assessed) (*Grab, error) {
	q := db.New(tx)
	names := map[int]string{}
	for i, a := range answers {
		p := outcomeRow(a.Outcome)
		p.SearchID, p.Position = run.ID, asked[i].Position
		if _, err := q.AnswerSearchIndexer(ctx, p); err != nil {
			return nil, err
		}
		for range a.Releases {
			names[len(names)] = asked[i].IndexerName
		}
	}
	bl, err := blocksOf(ctx, tx, as)
	if err != nil {
		return nil, err
	}
	as = bl.gate(as)
	if err := sight(ctx, tx, as); err != nil {
		return nil, err
	}
	results := make([]formats.Result, len(as))
	for i, a := range as {
		results[i] = a.formats
	}
	if err := targets.AddTimeouts(ctx, tx, results); err != nil {
		return nil, err
	}
	d, c, err := gr.decision(ctx, tx, run, as)
	if err != nil {
		return nil, err
	}
	finished := storage.Now()
	grab, pick, err := gr.take(ctx, tx, run, d.picks(c), nil, finished)
	if err != nil {
		return nil, err
	}
	if err := gr.storeReleases(ctx, q, run.ID, outcomes(d, c, pick.at), names, grab != nil, nil); err != nil {
		return nil, err
	}
	if err := q.DeleteOlderReleasesOfItem(ctx, db.DeleteOlderReleasesOfItemParams{ItemID: run.ItemID, SearchID: run.ID}); err != nil {
		return nil, err
	}
	finish := db.FinishSearchParams{ID: run.ID, FinishedAt: finished, Outcome: new(NothingAcceptable), Candidates: int64(len(as))}
	if grab == nil {
		return nil, q.FinishSearch(ctx, finish)
	}
	notes, err := encodeReasons(pick.Notes)
	if err != nil {
		return nil, err
	}
	finish.Outcome, finish.GrabID, finish.Notes = new(Grabbed), grab.ID, &notes
	return grab, q.FinishSearch(ctx, finish)
}

// sight records a sighting of the cut of every listed Release whose name
// names its Work and the Blocklist doesn't hold, for each Item it covers,
// so a cut seen now is wanted now.
func sight(ctx context.Context, tx *storage.Tx[file], as []assessed) error {
	for _, a := range as {
		if !a.namesTheWork() || a.blocked() {
			continue
		}
		for _, item := range a.covers {
			if err := targets.See(ctx, tx, item, a.read.Facts.Cut, releaseKey(a.Release)); err != nil {
				return err
			}
		}
	}
	return nil
}

// decision judges as at run's Item's open cells: the Decision, and the
// cells it judged them at.
func (gr *Grabs) decision(ctx context.Context, r storage.Reader[file], run Search, as []assessed) (Decision, cells, error) {
	open, err := gr.openCells(ctx, r, run.WorkID)
	if err != nil {
		return Decision{}, cells{}, err
	}
	var c cells
	for _, cell := range open {
		if cell.Item.ID == run.ItemID {
			c.open = append(c.open, cell)
		}
	}
	return decide(c, as), c, nil
}

// take grabs what choose takes of picks with Indexers.Take, and returns
// the Grab and the pick at the position taken. With none taken the Grab is
// nil and the position -1: an Indexer that is stopped, at its grab limit
// or deleted would make the Grab wait for the Admin. It skips the
// positions gone, which an earlier Grab of the run took.
func (gr *Grabs) take(ctx context.Context, tx *storage.Tx[file], run Search, picks []Pick, gone map[int]bool, now storage.Time) (*Grab, Pick, error) {
	p, t, ok, err := choose(picks, gone, func(r indexers.Release) (indexers.Ticket, error) {
		return gr.cfg.Indexers.Take(ctx, tx, r)
	})
	if err != nil {
		return nil, Pick{}, err
	}
	if !ok {
		return nil, Pick{at: -1}, nil
	}
	g, err := gr.insertGrab(ctx, tx, newGrab{work: run.WorkID, item: run.ItemID, pick: p, ticket: t, source: FromSearch}, now)
	return &g, p, err
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

// newGrab is what insertGrab writes: a search's pick, or an Admin's from
// a Manual search with its record.
type newGrab struct {
	work      works.WorkID
	item      works.ItemID
	pick      Pick
	ticket    indexers.Ticket
	source    Source
	manual    ManualID
	by        *Actor
	overrides Overrides
	// sent is the client the Admin chose, nil for where Tracker routing
	// sends it.
	sent *downloadclients.Choice
}

// insertGrab inserts n's Grab, at fetching on its Indexer's home Node.
func (gr *Grabs) insertGrab(ctx context.Context, tx *storage.Tx[file], n newGrab, now storage.Time) (Grab, error) {
	q := db.New(tx)
	r, t := n.pick.Release, n.ticket
	g := Grab{
		ID: GrabID{UUID: storage.NewUUID()}, WorkID: n.work, ItemID: n.item, Items: []works.ItemID{n.item}, Edition: n.pick.Edition,
		Release: Release{Title: r.Title, Size: r.Size, InfoHash: r.InfoHash}, Ticket: t, Stage: Fetching, Created: now.Time(),
		Source: n.source, Manual: n.manual, By: n.by, Overrides: n.overrides, Sent: n.sent,
	}
	for _, target := range n.pick.Targets {
		g.Targets = append(g.Targets, Cell{Key: files.CellKey{Item: n.item, Edition: n.pick.Edition, Target: target}})
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
	overrides, err := json.Marshal(g.Overrides)
	if err != nil {
		return Grab{}, fmt.Errorf("grab %s: encode the overrides: %w", g.ID, err)
	}
	p := db.InsertGrabParams{
		ID: g.ID, WorkID: g.WorkID, ItemID: g.ItemID, Edition: g.Edition, Source: g.Source, ManualID: g.Manual, Overrides: string(overrides),
		IndexerID: t.Indexer, TrackerID: t.Tracker, NodeID: g.Node,
		ReleaseTitle: g.Release.Title, ReleaseSize: g.Release.Size, ReleaseLinkSealed: g.Release.link, ReleaseMagnet: orNil(g.Release.magnet),
		ReleaseInfoHash: hint, FetchQueryID: t.Query, CreatedAt: now,
		MinRatio: orNil(r.Floor.MinRatio), MinSeedSeconds: orNil(int64(r.Floor.MinSeed.Seconds())),
		DownloadFactor: new(r.Factors.Download), UploadFactor: new(r.Factors.Upload),
	}
	if b := g.By; b != nil {
		p.GrabbedByName = orNil(b.Name)
		if b.Account != nil {
			p.GrabbedBy = *b.Account
		}
	}
	if err := q.InsertGrab(ctx, p); err != nil {
		return Grab{}, err
	}
	if c := g.Sent; c != nil {
		s := db.SetSentParams{ID: g.ID, ClientID: c.Client, UpdatedAt: now}
		if c.Client == nil {
			s.BuiltInOn = &c.BuiltInOn
		}
		if err := q.SetSent(ctx, s); err != nil {
			return Grab{}, err
		}
	}
	if err := insertCells(ctx, q, g); err != nil {
		return Grab{}, err
	}
	return g, q.InsertStage(ctx, db.InsertStageParams{GrabID: g.ID, Stage: Fetching, EnteredAt: now})
}

// insertCells inserts g's Items and the cells it holds.
func insertCells(ctx context.Context, q *db.Queries, g Grab) error {
	for i, item := range g.Items {
		if err := q.InsertGrabItem(ctx, db.InsertGrabItemParams{GrabID: g.ID, ItemID: item, Position: int64(i)}); err != nil {
			return err
		}
	}
	for i, c := range g.Targets {
		p := db.InsertTargetParams{GrabID: g.ID, Position: int64(i), ItemID: c.Key.Item, TargetID: c.Key.Target, Forced: c.Forced}
		if err := q.InsertTarget(ctx, p); err != nil {
			return err
		}
	}
	return nil
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
		asked, err := q.ListSearchIndexersOfWork(ctx, work)
		if err != nil {
			return err
		}
		lines := map[SearchID][]SearchIndexer{}
		for _, row := range asked {
			o := outcomeOf(row)
			lines[row.SearchID] = append(lines[row.SearchID], SearchIndexer{
				Indexer: row.IndexerID, Name: row.IndexerName, Outcome: o, Line: gr.cfg.Indexers.Line(tree.ContentType, o),
			})
		}
		for _, row := range rows {
			s, err := searchOf(row, lines[row.ID])
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

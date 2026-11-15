package grabs

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Search is one search run: the Items of a search group searched on
// every Indexer it planned to ask.
type Search struct {
	ID     SearchID
	WorkID works.WorkID
	// Group is the search group the run searched, zero for a Manual run.
	Group indexers.GroupKey
	// Lane is the run's lane in words and Trigger what queued it, both ""
	// for a Manual run.
	Lane    string
	Trigger searches.Trigger
	// Items are the Items it searched.
	Items []works.ItemID
	// Manual is the Manual search the run lists Releases for, zero for an
	// automatic run.
	Manual  ManualID
	Started time.Time
	// Indexers are the Indexers the run asked or skipped, by priority.
	Indexers []SearchIndexer
	// Finished, Outcome and Candidates are set once the run finished.
	// Grabs are the Grabs it made, then those it joined, and Notes the
	// grabbed Release's: the decision's, or the rerun's once a dropped
	// Grab reran it.
	Finished   time.Time
	Outcome    Outcome
	Candidates int
	// Took counts the Grabs and joins at decide time, so it outlives a
	// pruned Grab.
	Took  int
	Grabs []GrabID
	Notes []Reason
}

// item is the one Item a Manual run or a press of one Item searches.
func (s Search) item() works.ItemID { return s.Items[0] }

// The press records its run as one of the now lane, with gone cells,
// under the trigger search.
const (
	pressLane    int64 = 10
	pressTrigger       = "search"
)

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

// runsOf is rows as runs, each with its Items, its Grabs and its lines
// in lines.
func runsOf(ctx context.Context, q *db.Queries, rows []db.GrabsSearch, lines map[SearchID][]SearchIndexer) ([]Search, error) {
	ids := make([]SearchID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	items, err := q.ListSearchItems(ctx, ids)
	if err != nil {
		return nil, err
	}
	itemsOf := map[SearchID][]works.ItemID{}
	for _, it := range items {
		itemsOf[it.SearchID] = append(itemsOf[it.SearchID], it.ItemID)
	}
	grabs, err := q.ListSearchGrabs(ctx, ids)
	if err != nil {
		return nil, err
	}
	grabsOf := map[SearchID][]GrabID{}
	for _, g := range grabs {
		grabsOf[g.SearchID] = append(grabsOf[g.SearchID], g.GrabID)
	}
	out := make([]Search, len(rows))
	for i, row := range rows {
		if out[i], err = searchOf(row, itemsOf[row.ID], grabsOf[row.ID], lines[row.ID]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func searchOf(row db.GrabsSearch, items []works.ItemID, grabs []GrabID, lines []SearchIndexer) (Search, error) {
	s := Search{
		ID: row.ID, WorkID: row.WorkID, Items: items, Manual: row.ManualID, Started: row.StartedAt.Time(), Indexers: lines,
		Finished: row.FinishedAt.Time(), Candidates: int(row.Candidates), Took: int(row.Grabs), Grabs: grabs,
	}
	if row.GroupKey != nil {
		s.Group = *row.GroupKey
	}
	if row.Lane != nil {
		s.Lane = pace.Lane(*row.Lane).String()
	}
	if row.Trigger != nil {
		s.Trigger = searches.Trigger(*row.Trigger)
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

// lineRow is line l of run, at position, planned at started. Plan
// reserves an asked line's query, so the line is dispatched as it is
// planned, with the default wait as its deadline.
func lineRow(run SearchID, position int, l indexers.Line, started storage.Time) db.InsertSearchIndexerParams {
	p := db.InsertSearchIndexerParams{SearchID: run, Position: int64(position), IndexerID: l.Indexer, IndexerName: l.Name, QueryID: l.Query}
	if !l.Query.IsZero() {
		p.DispatchedAt, p.DeadlineAt = started, storage.FromTime(started.Time().Add(indexers.SearchWait.Default))
	}
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

// Pressed is one run a press of SearchWork recorded: its search group and
// the label of the Item it searches.
type Pressed struct {
	Search SearchID
	Group  indexers.GroupKey
	Label  string
}

// SearchWork records a search run for each Item of work within scope with
// a wanted cell, each with the Indexers it asks and skips and a query
// reserved on each it asks, and returns them.
// Each run reserves its own queries, so an Indexer's search limit skips it
// for the runs past the limit. Run searches them. An id scope names that
// work doesn't have is a *works.NotInWorkError.
func (gr *Grabs) SearchWork(ctx context.Context, _ identity.Admin, work works.WorkID, scope Scope) ([]Pressed, error) {
	var runs []Pressed
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
			runs = append(runs, Pressed{Search: id, Group: pressGroup(it), Label: it.Label})
		}
		if len(runs) == 0 {
			return ErrNothingWanted
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, p := range runs {
		gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: p.Search.String(), WorkID: work.String()})
	}
	gr.wake()
	return runs, nil
}

// wantedOf is each of items that has a cell in open, in items' order.
func wantedOf(items []works.Item, open []files.Cell) []works.Item {
	wanted := make(map[works.ItemID]bool, len(open))
	for _, c := range open {
		wanted[c.Item.ID] = true
	}
	var out []works.Item
	for _, it := range items {
		if wanted[it.ID] {
			out = append(out, it)
		}
	}
	return out
}

// pressGroup is the search group a press of item alone records its run in:
// its level's, or its own when it has none.
func pressGroup(item works.Item) indexers.GroupKey {
	if item.LevelID.IsZero() {
		return indexers.ItemGroup(item.ID)
	}
	return indexers.LevelGroup(item.LevelID)
}

// plan records a press's search run of item, of tree, with the lines Plan
// read and reserved.
func (gr *Grabs) plan(ctx context.Context, tx *storage.Tx[file], tree works.Tree, item works.Item, started storage.Time) (SearchID, error) {
	q := db.New(tx)
	id := SearchID{UUID: storage.NewUUID()}
	group, lane, trigger := pressGroup(item), pressLane, pressTrigger
	p := db.InsertSearchParams{
		ID: id, WorkID: tree.ID, GroupKey: &group, Lane: &lane, Trigger: new(trigger), IncludeGone: true, StartedAt: started,
	}
	if err := q.InsertSearch(ctx, p); err != nil {
		return SearchID{}, err
	}
	if err := q.InsertSearchItem(ctx, db.InsertSearchItemParams{SearchID: id, ItemID: item.ID}); err != nil {
		return SearchID{}, err
	}
	lines, err := gr.cfg.Indexers.Plan(ctx, tx, tree.ContentType, indexers.AutoSearch)
	if err != nil {
		return SearchID{}, err
	}
	for i, l := range lines {
		if err := q.InsertSearchIndexer(ctx, lineRow(id, i, l, started)); err != nil {
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
		if s.Terms, err = gr.cfg.Terms[tree.ContentType].SearchTerms(ctx, r, run.item()); err != nil {
			return err
		}
		if j, err = gr.judgingOf(ctx, r, tree.ContentType, run.WorkID, run.item(), s.Terms.Expect); err != nil {
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
	var grabs []Grab
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		grabs, err = gr.finish(ctx, tx, run, asked, answers, as)
		return err
	})
	if err != nil {
		return gr.searchWaits(ctx, run, err)
	}
	gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: run.ID.String(), WorkID: run.WorkID.String()})
	for _, g := range grabs {
		gr.cfg.Indexers.Taken(ctx, g.Ticket)
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: g.ID.String(), WorkID: g.WorkID.String()})
	}
	return committed
}

// finish ends run in tx with each asked line's answer and decides once
// over as, the Releases they listed, assessed. It stores every Release
// with its outcome, keeping what a rerun needs while the run's Grabs
// fetch, records when its Items were first searched, and prunes the
// group's runs the bound lets go. It returns the Grabs the run made or
// joined.
func (gr *Grabs) finish(ctx context.Context, tx *storage.Tx[file], run Search, asked []db.GrabsSearchIndexer, answers []indexers.Answer, as []assessed) ([]Grab, error) {
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
	d, err := gr.decideIn(ctx, tx, runInput(run, as))
	if err != nil {
		return nil, err
	}
	if err := gr.storeReleases(ctx, q, run.ID, d.rows, names, len(d.grabs) > 0, nil); err != nil {
		return nil, err
	}
	finish := db.FinishSearchParams{ID: run.ID, FinishedAt: d.at, Outcome: new(NothingAcceptable), Candidates: int64(d.candidates)}
	if len(d.grabs) > 0 {
		notes, err := encodeReasons(d.notes)
		if err != nil {
			return nil, err
		}
		finish.Outcome, finish.Grabs, finish.Notes = new(Grabbed), int64(len(d.grabs)), &notes
	}
	if err := q.FinishSearch(ctx, finish); err != nil {
		return nil, err
	}
	for _, item := range run.Items {
		if err := q.InsertItemSearch(ctx, db.InsertItemSearchParams{ItemID: item, FirstSearchedAt: storage.FromTime(run.Started)}); err != nil {
			return nil, err
		}
	}
	return d.grabs, q.PruneGroup(ctx, db.PruneGroupParams{WorkID: run.WorkID, GroupKey: &run.Group})
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

// unfinishedOf is work's unfinished Grabs, read in r, oldest first.
func unfinishedOf(ctx context.Context, r storage.Reader[file], work works.WorkID) ([]Grab, error) {
	rows, err := db.New(r).ListOpenGrabsOfWork(ctx, work)
	if err != nil {
		return nil, err
	}
	return loadAll(ctx, r, rows)
}

// joinable reports whether a decision that picks g's Release adds its
// cells to g: while g fetches or downloads. Past downloading its Import
// has decided.
func (g Grab) joinable() bool { return g.Stage == Fetching || g.Stage == Downloading }

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
	work works.WorkID
	item works.ItemID
	// items is every Item the pick's Release covers, and cells the cells
	// it holds across them.
	items     []works.ItemID
	cells     []Cell
	pick      Pick
	ticket    indexers.Ticket
	source    Source
	manual    ManualID
	by        *Actor
	overrides Overrides
	// sent is the client the Admin chose, nil for where Tracker routing
	// sends it.
	sent *downloadclients.Choice
	// fills is the Items the fewer-Releases step chose it to fill, 0 when
	// another step did.
	fills int
}

// insertGrab inserts n's Grab, at fetching on its Indexer's home Node.
func (gr *Grabs) insertGrab(ctx context.Context, tx *storage.Tx[file], n newGrab, now storage.Time) (Grab, error) {
	q := db.New(tx)
	r, t := n.pick.Release, n.ticket
	g := Grab{
		ID: GrabID{UUID: storage.NewUUID()}, WorkID: n.work, ItemID: n.item, Items: n.items, Edition: n.pick.Edition,
		Release: Release{Title: r.Title, Size: r.Size, InfoHash: r.InfoHash}, Ticket: t, Stage: Fetching, Created: now.Time(),
		Source: n.source, Manual: n.manual, By: n.by, Overrides: n.overrides, Sent: n.sent, Targets: n.cells, ChosenFills: n.fills,
	}
	if len(g.Items) == 0 {
		g.Items = []works.ItemID{n.item}
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
		DownloadFactor: new(r.Factors.Download), UploadFactor: new(r.Factors.Upload), ChosenFills: orNil(int64(n.fills)),
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
		out, err = runsOf(ctx, q, rows, lines)
		return err
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

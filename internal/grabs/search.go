package grabs

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Search is one search run: one Item searched on the first Indexer.
type Search struct {
	ID     SearchID
	WorkID works.WorkID
	ItemID works.ItemID
	// Indexer is the Indexer the run asked: zero until it asked, or once
	// the Indexer is deleted.
	Indexer indexers.ID
	Started time.Time
	// Finished, Outcome, Candidates and Refused are set once the run
	// finished, and GrabID and Notes, the grabbed release's, when it
	// grabbed.
	Finished   time.Time
	Outcome    Outcome
	Candidates int
	Refused    []Refusal
	GrabID     GrabID
	Notes      []Reason
}

func searchOf(row db.GrabsSearch, refused []Refusal, asked []db.GrabsSearchIndexer) (Search, error) {
	s := Search{
		ID: row.ID, WorkID: row.WorkID, ItemID: row.ItemID, Started: row.StartedAt.Time(),
		Finished: row.FinishedAt.Time(), Candidates: int(row.Candidates), Refused: refused, GrabID: row.GrabID,
	}
	if row.Outcome != nil {
		s.Outcome = *row.Outcome
	}
	if len(asked) > 0 {
		s.Indexer = asked[0].IndexerID
		if s.Outcome == NothingAcceptable && asked[0].Outcome != nil {
			s.Outcome = runOutcome[*asked[0].Outcome]
		}
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

// runOutcome is how a run that found nothing reads after its Indexer's
// outcome.
var runOutcome = map[indexers.OutcomeKind]Outcome{
	indexers.Found: NothingAcceptable, indexers.NoReleases: NothingAcceptable, indexers.NoCategories: NothingAcceptable,
	indexers.IsStopped: SearchIndexerRefused, indexers.Removed: SearchIndexerUnavailable,
	indexers.IsLimited: SearchIndexerUnavailable, indexers.IsFailing: SearchIndexerUnavailable, indexers.Waiting: SearchIndexerUnavailable,
	indexers.Late: SearchIndexerUnavailable, indexers.LimitReached: SearchIndexerUnavailable,
}

// SearchWork records a search run for each Item of work with a wanted cell
// no unfinished Grab holds, and returns their ids. Run searches them on
// the first Indexer by name.
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
		q := db.New(tx)
		for _, it := range tree.Items {
			if !slices.ContainsFunc(open, func(c files.Cell) bool { return c.Item.ID == it.ID }) {
				continue
			}
			id := SearchID{UUID: storage.NewUUID()}
			if err := q.InsertSearch(ctx, db.InsertSearchParams{ID: id, WorkID: work, ItemID: it.ID, StartedAt: started}); err != nil {
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

// search runs one search on the first Indexer by name and finishes its run
// in one Write, with the Grab it picked. A search never waits on the
// Indexer: an Indexer that couldn't search finishes the run, and the Admin
// searches again.
func (gr *Grabs) search(ctx context.Context, run Search) result {
	var idx *indexers.Indexer
	var terms indexers.Terms
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		all, err := indexers.All(ctx, r)
		if err != nil {
			return err
		}
		if len(all) > 0 {
			idx = &all[0]
		}
		tree, err := gr.cfg.Works.TreeIn(ctx, r, run.WorkID)
		if err != nil {
			return err
		}
		terms, err = gr.cfg.Terms[tree.ContentType].SearchTerms(ctx, r, run.ItemID)
		return err
	})
	if err != nil {
		return gr.searchWaits(ctx, run, err)
	}
	var answer *indexers.Answer
	if idx != nil {
		a, err := gr.cfg.Indexers.Search(ctx, *idx, terms)
		if ctx.Err() != nil {
			return halted
		}
		if err != nil {
			return gr.searchWaits(ctx, run, err)
		}
		answer = &a
	}
	var grab *Grab
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		grab, err = gr.finish(ctx, tx, run, idx, terms, answer)
		return err
	})
	if err != nil {
		return gr.searchWaits(ctx, run, err)
	}
	gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: run.ID.String(), WorkID: run.WorkID.String()})
	if grab != nil {
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: grab.ID.String(), WorkID: grab.WorkID.String()})
	}
	return committed
}

// finish ends run in tx with idx's answer, and the Grab it picks from its
// Releases, nil when it picks none. idx and answer are nil when no
// Indexer was left to ask.
func (gr *Grabs) finish(ctx context.Context, tx *storage.Tx[file], run Search, idx *indexers.Indexer, terms indexers.Terms, answer *indexers.Answer) (*Grab, error) {
	q := db.New(tx)
	finish := db.FinishSearchParams{ID: run.ID, FinishedAt: storage.Now()}
	outcome := NothingAcceptable
	var grab *Grab
	if answer != nil {
		if err := insertAsked(ctx, q, run, *idx, answer.Outcome); err != nil {
			return nil, err
		}
		finish.Candidates = int64(len(answer.Releases))
	}
	if answer != nil && answer.Outcome.Kind == indexers.Found {
		pick, err := gr.pick(ctx, tx, run, terms, answer.Releases)
		if err != nil {
			return nil, err
		}
		if pick != nil {
			g, err := gr.insertGrab(ctx, q, run, *idx, *pick, finish.FinishedAt)
			if err != nil {
				return nil, err
			}
			notes, err := encodeReasons(pick.Notes)
			if err != nil {
				return nil, err
			}
			grab, finish.GrabID, finish.Notes, outcome = &g, g.ID, &notes, Grabbed
		}
	}
	finish.Outcome = &outcome
	return grab, q.FinishSearch(ctx, finish)
}

// insertAsked records idx's outcome in run, with its name at search time.
func insertAsked(ctx context.Context, q *db.Queries, run Search, idx indexers.Indexer, o indexers.Outcome) error {
	p := db.InsertSearchIndexerParams{
		SearchID: run.ID, Position: 0, IndexerID: idx.ID, IndexerName: idx.Name, Outcome: &o.Kind,
		UntilAt: storage.FromTime(o.Until), SinceAt: storage.FromTime(o.Since), NodeID: o.Node,
	}
	switch o.Kind {
	case indexers.Found, indexers.NoReleases:
		releases := int64(o.Releases)
		p.Releases = &releases
	case indexers.Waiting:
		p.Lapsed = &o.Lapsed
	case indexers.IsStopped:
		p.Cause, p.Text = &o.Cause, &o.Text
	case indexers.IsLimited, indexers.IsFailing, indexers.Late, indexers.LimitReached, indexers.NoCategories, indexers.Removed:
	}
	return q.InsertSearchIndexer(ctx, p)
}

// pick decides among releases for run's Item's open cells and stores the
// releases it refused. Every listed release of this Work records a
// sighting of its cut first, so a cut seen now is wanted now.
func (gr *Grabs) pick(ctx context.Context, tx *storage.Tx[file], run Search, terms indexers.Terms, releases []indexers.Release) (*Pick, error) {
	tree, err := gr.cfg.Works.TreeIn(ctx, tx, run.WorkID)
	if err != nil {
		return nil, err
	}
	known, err := gr.cfg.Targets.CutsIn(ctx, tx)
	if err != nil {
		return nil, err
	}
	timings, err := gr.cfg.Targets.TimingsIn(ctx, tx, tree.ContentType, []works.WorkID{run.WorkID})
	if err != nil {
		return nil, err
	}
	candidates := make([]candidate, len(releases))
	for i, r := range releases {
		candidates[i] = candidateOf(r, known.For(tree.ContentType), run.ItemID, timings[run.ItemID].Runtime)
		if wrongWork(r, terms) {
			continue
		}
		if err := targets.See(ctx, tx, run.ItemID, candidates[i].Facts.Cut, releaseKey(r)); err != nil {
			return nil, err
		}
	}
	open, err := gr.openCells(ctx, tx, run.WorkID)
	if err != nil {
		return nil, err
	}
	var item []files.Cell
	for _, c := range open {
		if c.Item.ID == run.ItemID {
			item = append(item, c)
		}
	}
	pick, refused := decide(item, terms, candidates)
	q := db.New(tx)
	for i, r := range refused {
		reasons, err := encodeReasons(r.Reasons)
		if err != nil {
			return nil, err
		}
		if err := q.InsertRefusal(ctx, db.InsertRefusalParams{SearchID: run.ID, Position: int64(i), Title: r.Title, Reasons: reasons}); err != nil {
			return nil, err
		}
	}
	return pick, nil
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

func (gr *Grabs) insertGrab(ctx context.Context, q *db.Queries, run Search, idx indexers.Indexer, pick Pick, now storage.Time) (Grab, error) {
	g := Grab{
		ID: GrabID{UUID: storage.NewUUID()}, WorkID: run.WorkID, ItemID: run.ItemID, Edition: pick.Edition, Targets: pick.Targets,
		Release: Release{Title: pick.Release.Title, Size: pick.Release.Size, Indexer: idx.ID, InfoHash: pick.Release.InfoHash},
		Node:    idx.Home, Stage: Fetching, Created: now.Time(),
	}
	if link := offered(pick.Release); link.IsMagnet() {
		g.Release.magnet = link.Reveal()
	} else {
		g.Release.link = gr.cfg.Sealer.Seal([]byte(link.Reveal()), linkColumn.Row(g.ID.String()))
	}
	var hint []byte
	if h := g.Release.InfoHash; h != nil {
		hint = h[:]
	}
	err := q.InsertGrab(ctx, db.InsertGrabParams{
		ID: g.ID, WorkID: g.WorkID, ItemID: g.ItemID, Edition: g.Edition, IndexerID: idx.ID, NodeID: g.Node,
		ReleaseTitle: g.Release.Title, ReleaseSize: g.Release.Size, ReleaseLinkSealed: g.Release.link, ReleaseMagnet: orNil(g.Release.magnet),
		ReleaseInfoHash: hint, CreatedAt: now,
	})
	if err != nil {
		return Grab{}, err
	}
	for i, t := range g.Targets {
		if err := q.InsertTarget(ctx, db.InsertTargetParams{GrabID: g.ID, Position: int64(i), TargetID: t}); err != nil {
			return Grab{}, err
		}
	}
	return g, q.InsertStage(ctx, db.InsertStageParams{GrabID: g.ID, Stage: Fetching, EnteredAt: now})
}

// offered is the link a Grab of r fetches: its .torrent link, else its
// magnet link, since a .torrent's info dictionary needs no peer.
func offered(r indexers.Release) searching.Link {
	if r.Link.IsZero() {
		return r.Magnet
	}
	return r.Link
}

// Searches lists work's search runs, newest first.
func (gr *Grabs) Searches(ctx context.Context, _ identity.Admin, work works.WorkID) ([]Search, error) {
	var out []Search
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		if _, err := gr.cfg.Works.TreeIn(ctx, r, work); err != nil {
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
		lines, err := q.ListSearchIndexersOfWork(ctx, work)
		if err != nil {
			return err
		}
		asked := map[SearchID][]db.GrabsSearchIndexer{}
		for _, l := range lines {
			asked[l.SearchID] = append(asked[l.SearchID], l)
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
			s, err := searchOf(row, refused[row.ID], asked[row.ID])
			if err != nil {
				return err
			}
			out = append(out, s)
		}
		return nil
	})
	return out, err
}

func orNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

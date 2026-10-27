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
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Search is one search run: one Item searched on one Indexer.
type Search struct {
	ID      SearchID
	WorkID  works.WorkID
	ItemID  works.ItemID
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

func searchOf(row db.GrabsSearch, refused []Refusal) (Search, error) {
	s := Search{
		ID: row.ID, WorkID: row.WorkID, ItemID: row.ItemID, Indexer: row.IndexerID, Started: row.StartedAt.Time(),
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

// SearchWork records a search run for each Item of work with a wanted cell
// no unfinished Grab holds, and returns their ids. Run searches them on
// the only Indexer.
func (gr *Grabs) SearchWork(ctx context.Context, _ identity.Admin, work works.WorkID) ([]SearchID, error) {
	var ids []SearchID
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		tree, err := gr.cfg.Works.TreeIn(ctx, tx, work)
		if err != nil {
			return err
		}
		indexer, err := gr.searchable(ctx, tx, tree)
		if err != nil {
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
			if err := q.InsertSearch(ctx, db.InsertSearchParams{ID: id, WorkID: work, ItemID: it.ID, IndexerID: indexer, StartedAt: started}); err != nil {
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

// searchable is the Indexer tree's searches run on, or why tree can't be
// searched: a rootless Work, or one whose root's Node is offline or
// written off, doesn't search.
func (gr *Grabs) searchable(ctx context.Context, r storage.Reader[file], tree works.Tree) (indexers.ID, error) {
	if len(tree.Items) > 0 {
		terms, err := gr.cfg.Terms[tree.ContentType].SearchTerms(ctx, r, tree.Items[0].ID)
		if err != nil {
			return indexers.ID{}, err
		}
		if terms.Mode == "" && terms.Query == "" && len(terms.IDs) == 0 {
			return indexers.ID{}, ErrNotSearchable
		}
	}
	all, err := indexers.All(ctx, r)
	if err != nil {
		return indexers.ID{}, err
	}
	if len(all) == 0 {
		return indexers.ID{}, ErrNoIndexer
	}
	root, err := roots.OfWork(ctx, r, tree.ID)
	if err != nil {
		return indexers.ID{}, err
	}
	if err := gr.cfg.Roots.Reachable(ctx, r, root); err != nil {
		return indexers.ID{}, err
	}
	return all[0].ID, nil
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

// search runs one search and finishes its run in one Write, with the Grab
// it picked. A search never waits on the Indexer: an unavailable or
// refusing Indexer finishes the run, and the Admin searches again.
func (gr *Grabs) search(ctx context.Context, run Search) result {
	var idx indexers.Indexer
	var terms indexers.Terms
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if idx, err = indexers.Get(ctx, r, run.Indexer); err != nil {
			return err
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
	releases, err := gr.cfg.Indexers.Search(ctx, idx, terms)
	if ctx.Err() != nil {
		return halted
	}
	outcome := Grabbed
	if _, refused := errors.AsType[*searching.Stopped](err); refused {
		outcome = SearchIndexerRefused
	} else if err != nil {
		outcome = SearchIndexerUnavailable
	}
	var grab *Grab
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		grab, err = gr.finish(ctx, tx, run, idx, terms, releases, outcome)
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

// finish ends run in tx with the Grab it picks from releases, nil when it
// picks none.
func (gr *Grabs) finish(ctx context.Context, tx *storage.Tx[file], run Search, idx indexers.Indexer, terms indexers.Terms, releases []indexers.Release, outcome Outcome) (*Grab, error) {
	q := db.New(tx)
	finish := db.FinishSearchParams{ID: run.ID, FinishedAt: storage.Now(), Candidates: int64(len(releases))}
	var grab *Grab
	if outcome == Grabbed {
		pick, err := gr.pick(ctx, tx, run, terms, releases)
		if err != nil {
			return nil, err
		}
		if pick == nil {
			outcome = NothingAcceptable
		} else {
			g, err := insertGrab(ctx, q, run, idx, *pick, finish.FinishedAt)
			if err != nil {
				return nil, err
			}
			notes, err := encodeReasons(pick.Notes)
			if err != nil {
				return nil, err
			}
			grab, finish.GrabID, finish.Notes = &g, g.ID, &notes
		}
	}
	finish.Outcome = &outcome
	return grab, q.FinishSearch(ctx, finish)
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

func insertGrab(ctx context.Context, q *db.Queries, run Search, idx indexers.Indexer, pick Pick, now storage.Time) (Grab, error) {
	g := Grab{
		ID: GrabID{UUID: storage.NewUUID()}, WorkID: run.WorkID, ItemID: run.ItemID, Edition: pick.Edition, Targets: pick.Targets,
		Release: Release{Title: pick.Release.Title, Size: pick.Release.Size, Indexer: idx.ID, Link: offered(pick.Release), InfoHash: pick.Release.InfoHash},
		Node:    idx.HomeNode, Stage: Fetching, Created: now.Time(),
	}
	var hint []byte
	if h := g.Release.InfoHash; h != nil {
		hint = h[:]
	}
	err := q.InsertGrab(ctx, db.InsertGrabParams{
		ID: g.ID, WorkID: g.WorkID, ItemID: g.ItemID, Edition: g.Edition, IndexerID: idx.ID, NodeID: g.Node,
		ReleaseTitle: g.Release.Title, ReleaseSize: g.Release.Size, ReleaseLink: g.Release.Link.Reveal(), ReleaseInfoHash: hint, CreatedAt: now,
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
		refused := map[SearchID][]Refusal{}
		for _, row := range refusals {
			reasons, err := decodeReasons(row.Reasons)
			if err != nil {
				return fmt.Errorf("search %s: %w", row.SearchID, err)
			}
			refused[row.SearchID] = append(refused[row.SearchID], Refusal{Title: row.Title, Reasons: reasons})
		}
		for _, row := range rows {
			s, err := searchOf(row, refused[row.ID])
			if err != nil {
				return err
			}
			out = append(out, s)
		}
		return nil
	})
	return out, err
}

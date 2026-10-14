package grabs

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Search is one search run: one Item searched on one Indexer.
type Search struct {
	ID      SearchID
	WorkID  works.WorkID
	ItemID  works.ItemID
	Indexer indexers.ID
	Started time.Time
	// Finished, Outcome and Candidates are set once the run finished, and
	// GrabID when it grabbed.
	Finished   time.Time
	Outcome    Outcome
	Candidates int
	GrabID     GrabID
}

func searchOf(row db.GrabsSearch) Search {
	s := Search{
		ID: row.ID, WorkID: row.WorkID, ItemID: row.ItemID, Indexer: row.IndexerID, Started: row.StartedAt.Time(),
		Finished: row.FinishedAt.Time(), Candidates: int(row.Candidates), GrabID: row.GrabID,
	}
	if row.Outcome != nil {
		s.Outcome = *row.Outcome
	}
	return s
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
// searched.
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
	if _, err := roots.Of(ctx, r, tree.ContentType); err != nil {
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
	items, err := gr.cfg.Indexers.Search(ctx, idx, terms)
	if ctx.Err() != nil {
		return halted
	}
	outcome := Grabbed
	if _, refused := errors.AsType[*indexers.RefusedError](err); refused {
		outcome = SearchIndexerRefused
	} else if err != nil {
		outcome = SearchIndexerUnavailable
	}
	var grab *Grab
	err = gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		grab, err = gr.finish(ctx, tx, run, idx, terms, items, outcome)
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

// finish ends run in tx with the Grab it picks from items, nil when it
// picks none.
func (gr *Grabs) finish(ctx context.Context, tx *storage.Tx[file], run Search, idx indexers.Indexer, terms indexers.Terms, items []indexers.Item, outcome Outcome) (*Grab, error) {
	q := db.New(tx)
	finish := db.FinishSearchParams{ID: run.ID, FinishedAt: storage.Now(), Candidates: int64(len(items))}
	var grab *Grab
	if outcome == Grabbed {
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
		pick, rejected := decide(item, terms, items)
		for _, r := range rejected {
			gr.cfg.Log.Debug("a release was rejected", "search", run.ID.String(), "title", r.Title, "reason", string(r.Reason))
		}
		if pick == nil {
			outcome = NothingAcceptable
		} else {
			g, err := insertGrab(ctx, q, run, idx, *pick, finish.FinishedAt)
			if err != nil {
				return nil, err
			}
			grab, finish.GrabID = &g, g.ID
		}
	}
	finish.Outcome = &outcome
	return grab, q.FinishSearch(ctx, finish)
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
		ID: GrabID{UUID: storage.NewUUID()}, WorkID: run.WorkID, ItemID: run.ItemID, Edition: files.Standard, Targets: pick.Targets,
		Release: Release{Title: pick.Item.Title, Size: pick.Item.Size, Indexer: idx.ID, Link: pick.Item.Link},
		Node:    idx.HomeNode, Stage: Fetching, Created: now.Time(),
	}
	var hint []byte
	if h, err := bencode.ParseInfoHash(pick.Item.InfoHash); err == nil {
		g.Release.InfoHash, hint = &h, h[:]
	}
	err := q.InsertGrab(ctx, db.InsertGrabParams{
		ID: g.ID, WorkID: g.WorkID, ItemID: g.ItemID, Edition: g.Edition, IndexerID: idx.ID, NodeID: g.Node,
		ReleaseTitle: g.Release.Title, ReleaseSize: g.Release.Size, ReleaseLink: g.Release.Link, ReleaseInfoHash: hint, CreatedAt: now,
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

// Searches lists work's search runs, newest first.
func (gr *Grabs) Searches(ctx context.Context, _ identity.Admin, work works.WorkID) ([]Search, error) {
	var out []Search
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		if _, err := gr.cfg.Works.TreeIn(ctx, r, work); err != nil {
			return err
		}
		rows, err := db.New(r).ListSearchesOfWork(ctx, work)
		if err != nil {
			return err
		}
		for _, row := range rows {
			out = append(out, searchOf(row))
		}
		return nil
	})
	return out, err
}

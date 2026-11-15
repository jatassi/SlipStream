package grabs

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
)

// heardRun is what a run's lines heard: their Releases in the order they
// landed, each one's Indexer name by that order, and whether any asked
// Indexer answered.
type heardRun struct {
	releases []indexers.Release
	names    map[int]string
	answered bool
}

// settle decides run once every line answered, was skipped or went late:
// it assesses what the lines heard, and in one fenced Write decides, keeps
// every Release with its outcome and finishes the run.
func (gr *Grabs) settle(ctx context.Context, run Search) result {
	var h heardRun
	var j judging
	err := gr.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		if h, err = gr.linesHeard(ctx, r, run.ID); err != nil {
			return err
		}
		tree, err := gr.cfg.Works.TreeIn(ctx, r, run.WorkID)
		if err != nil {
			return err
		}
		expect, err := gr.cfg.Terms[tree.ContentType].Expect(ctx, r, run.WorkID)
		if err != nil {
			return err
		}
		j, err = gr.judgingOf(ctx, r, tree.ContentType, run.WorkID, run.item(), expect)
		return err
	})
	if err != nil {
		return gr.searchWaits(ctx, run, err)
	}
	grab, err := gr.decideRun(ctx, run, h, assessAll(j, dedupe(listedOf(h.releases))))
	switch {
	case errors.Is(err, errRunGone):
		return committed
	case errors.Is(err, ErrReconciling):
		return waited
	case err != nil:
		return gr.searchWaits(ctx, run, err)
	}
	gr.cfg.SearchChanged.Publish(ctx, SearchChanged{SearchID: run.ID.String(), WorkID: run.WorkID.String()})
	gr.cfg.QueueChanged.Publish(ctx, QueueChanged{WorkIDs: []string{run.WorkID.String()}})
	if grab != nil {
		gr.cfg.Indexers.Taken(ctx, grab.Ticket)
		gr.cfg.Changed.Publish(ctx, Changed{GrabID: grab.ID.String(), WorkID: grab.WorkID.String()})
	}
	gr.wake()
	return committed
}

// decideRun is run's one fenced deciding Write over as, what h heard
// assessed: the decision, every Release with its outcome, and the run's
// finish. It answers the Grab it took, and errRunGone once run is gone or
// finished.
func (gr *Grabs) decideRun(ctx context.Context, run Search, h heardRun, as []assessed) (*Grab, error) {
	var grab *Grab
	err := gr.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		if err := gr.fence(ctx, tx); err != nil {
			return err
		}
		q := db.New(tx)
		row, err := q.GetSearch(ctx, run.ID)
		if errors.Is(err, storage.ErrNoRows) || err == nil && !row.FinishedAt.IsZero() {
			return errRunGone
		}
		if err != nil {
			return err
		}
		if err := q.DeleteHeard(ctx, run.ID); err != nil {
			return err
		}
		d, err := gr.decideIn(ctx, tx, runInput(run, as))
		if err != nil {
			return err
		}
		if err := gr.storeReleases(ctx, q, run.ID, d.rows, h.names, d.grab != nil, nil); err != nil {
			return err
		}
		grab = d.grab
		return finishRun(ctx, q, run, d, finishedAs(d, h))
	})
	return grab, err
}

// linesHeard is what run's lines heard, read in r.
func (gr *Grabs) linesHeard(ctx context.Context, r storage.Reader[file], run SearchID) (heardRun, error) {
	q := db.New(r)
	h := heardRun{names: map[int]string{}}
	lines, err := q.ListSearchIndexersOfSearch(ctx, run)
	if err != nil {
		return heardRun{}, err
	}
	for _, l := range lines {
		if o := l.Outcome; o != nil && (*o == indexers.Found || *o == indexers.NoReleases) {
			h.answered = true
		}
	}
	rows, err := q.ListHeard(ctx, run)
	if err != nil {
		return heardRun{}, err
	}
	for i, row := range rows {
		rel, err := gr.kept(row)
		if err != nil {
			return heardRun{}, err
		}
		h.releases = append(h.releases, rel)
		h.names[i] = row.IndexerName
	}
	return h, nil
}

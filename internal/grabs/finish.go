package grabs

import (
	"context"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// finishedAs is how a run that decided d over what h heard ended: grabbed
// when it took a Grab, no_indexer when no asked Indexer answered,
// nothing_found when they listed no Release, else nothing_acceptable.
func finishedAs(d decisionOut, h heardRun) Outcome {
	switch {
	case d.grab != nil:
		return Grabbed
	case !h.answered:
		return NoIndexer
	case len(h.releases) == 0:
		return NothingFound
	}
	return NothingAcceptable
}

// finishRun ends run in q's Write with outcome and what d decided, records
// when its Items were first searched, frees its group's due row for the
// next run, and prunes the group's runs the bound lets go.
func finishRun(ctx context.Context, q *db.Queries, run Search, d decisionOut, outcome Outcome) error {
	f := db.FinishSearchParams{ID: run.ID, FinishedAt: d.at, Outcome: &outcome, Candidates: int64(d.candidates)}
	if d.grab != nil {
		notes, err := encodeReasons(d.notes)
		if err != nil {
			return err
		}
		f.Grabs, f.Notes = 1, &notes
	}
	if err := q.FinishSearch(ctx, f); err != nil {
		return err
	}
	for _, item := range run.Items {
		if err := q.InsertItemSearch(ctx, db.InsertItemSearchParams{ItemID: item, FirstSearchedAt: storage.FromTime(run.Started)}); err != nil {
			return err
		}
	}
	if err := q.EndDueSearch(ctx, db.EndDueSearchParams{ChangedAt: d.at, SearchID: run.ID}); err != nil {
		return err
	}
	return q.PruneGroup(ctx, db.PruneGroupParams{WorkID: run.WorkID, GroupKey: &run.Group})
}

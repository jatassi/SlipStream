package grabs

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/storage"
)

// finishedAs is how a run that decided d over what h heard ended: grabbed
// when it made or joined a Grab, waiting when a cell waits for one of its
// Releases, no_indexer when no asked Indexer answered, nothing_found when
// they listed no Release, else nothing_acceptable.
func finishedAs(d decisionOut, h heardRun) Outcome {
	switch {
	case len(d.grabs) > 0:
		return Grabbed
	case len(d.waits) > 0:
		return Waiting
	case !h.answered:
		return NoIndexer
	case len(h.releases) == 0:
		return NothingFound
	}
	return NothingAcceptable
}

// resultOf is what a run that decided d over what h heard tells its
// group's pace: a Release its cells wait for is one it took.
func resultOf(d decisionOut, h heardRun) pace.Result {
	switch {
	case len(d.grabs) > 0, len(d.waits) > 0:
		return pace.Took
	case h.answered:
		return pace.Answered
	}
	return pace.Unheard
}

// finishRun ends run in tx with what d decided over what h heard, records
// when its Items were first searched, paces its group's due row under
// Backlog search's interval and prunes the group's runs the bound lets go.
func (gr *Grabs) finishRun(ctx context.Context, tx *storage.Tx[file], run Search, d decisionOut, h heardRun, interval time.Duration) error {
	q := db.New(tx)
	outcome := finishedAs(d, h)
	f := db.FinishSearchParams{ID: run.ID, FinishedAt: d.at, Outcome: &outcome, Candidates: int64(d.candidates)}
	if len(d.grabs) > 0 {
		notes, err := encodeReasons(d.notes)
		if err != nil {
			return err
		}
		f.Grabs, f.Notes = int64(len(d.grabs)), &notes
	}
	if err := q.FinishSearch(ctx, f); err != nil {
		return err
	}
	for _, item := range run.Items {
		if err := q.InsertItemSearch(ctx, db.InsertItemSearchParams{ItemID: item, FirstSearchedAt: storage.FromTime(run.Started)}); err != nil {
			return err
		}
	}
	if err := gr.paceDue(ctx, tx, run, resultOf(d, h), d.at.Time(), interval); err != nil {
		return err
	}
	return q.PruneGroup(ctx, db.PruneGroupParams{WorkID: run.WorkID, GroupKey: &run.Group})
}

// paceDue frees run's due row for the next run, due a gap after at, the
// gap set by what the run told the pace: the interval after it took, or
// while one of its Items is inside the full-pace window, double after its
// Indexers answered with nothing to take, the same when none answered.
func (gr *Grabs) paceDue(ctx context.Context, tx *storage.Tx[file], run Search, r pace.Result, at time.Time, interval time.Duration) error {
	q := db.New(tx)
	due, err := q.GetDue(ctx, db.GetDueParams{WorkID: run.WorkID, GroupKey: run.Group})
	if errors.Is(err, storage.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	s, err := q.GetSettings(ctx)
	if err != nil {
		return err
	}
	cells, err := gr.cfg.Files.Cells(ctx, tx, run.WorkID)
	if err != nil {
		return err
	}
	fresh := freshIn(within(cells, run.Items), at.Add(-FullPace.Value(s.FullPaceSeconds)))
	var capped time.Duration
	if c, ok := gr.cfg.GapCap(ctx, tx, run.WorkID, run.Items); ok {
		capped = c
	}
	prev := time.Duration(due.GapSeconds) * time.Second
	gap := pace.Gap(prev, interval, max(LongestGap.Value(s.LongestGapSeconds), interval), r, fresh, capped)
	return q.PaceDue(ctx, db.PaceDueParams{
		GapSeconds: int64(gap / time.Second), DueAt: storage.FromTime(at.Add(gap)), ChangedAt: storage.FromTime(at), SearchID: run.ID,
	})
}

// freshIn reports whether a Paced cell of cells was released after since.
func freshIn(cells []files.Cell, since time.Time) bool {
	for _, c := range cells {
		if from, ok := c.Timing.From(); ok && c.Paced() && from.After(since) {
			return true
		}
	}
	return false
}

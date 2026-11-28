package conversions

import (
	"context"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/conversions/internal/order"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// ask is one Work's nudges, merged: why the drain converges it, when its
// download wait counts from, and the backfill it queues into.
type ask struct {
	Work     works.WorkID
	Why      files.Arrival
	Since    time.Time
	Backfill BackfillID
}

// explicit is an Admin's ask for the Work by name: a backfill's Convert
// or a remake, which clears the Work's declines.
func (a ask) explicit() bool { return a.Why == files.ArrivalRemake || !a.Backfill.IsZero() }

// laneRules is the lane of each want, the first rule that holds.
var laneRules = []struct {
	Holds func(ask, Want) bool
	Lane  Lane
}{
	{func(a ask, _ Want) bool {
		return a.explicit() || a.Why == files.ArrivalAdopted || a.Why == files.ArrivalProfile
	}, LaneBackfill},
	{func(_ ask, w Want) bool { return w.Upgrade }, LaneUpgrade},
	{func(ask, Want) bool { return true }, LaneNew},
}

func laneOf(a ask, w Want) Lane {
	for _, r := range laneRules {
		if r.Holds(a, w) {
			return r.Lane
		}
	}
	panic("conversions: no lane rule holds")
}

// notBefore is when a want may start: a "Downloaded, or made from" cell
// waits its grace for a download first.
func notBefore(a ask, w Want) time.Time {
	if w.Sourcing == targets.DownloadThenDerive {
		return a.Since.Add(w.Grace)
	}
	return a.Since
}

// enqueue queues w at the tail of its lane.
func (c *Conversions) enqueue(ctx context.Context, q *db.Queries, a ask, w Want, work works.Summary, target string) (ID, error) {
	lane := laneOf(a, w)
	tail, err := q.TailOfLane(ctx, string(lane))
	if err != nil {
		return ID{}, fmt.Errorf("read the lane's tail: %w", err)
	}
	id := ID{UUID: storage.NewUUID()}
	err = q.InsertConversion(ctx, db.InsertConversionParams{
		ID: id, WorkID: w.Work, Type: work.ContentType, Items: jsonOf(nonNil(w.Items)), Edition: w.Edition,
		SourceFileID: w.Source, TargetID: w.Target, Version: int64(w.Version), Lane: string(lane),
		Position: string(order.After(order.Key(tail))), BackfillID: a.Backfill, NotBefore: storage.FromTime(notBefore(a, w)),
		Snapshot: jsonOf(Snapshot{Work: work.Label, Target: target, SourceName: w.SourceName}),
		QueuedAt: storage.FromTime(c.cfg.Clock()),
	})
	if err != nil {
		return ID{}, fmt.Errorf("queue the Conversion: %w", err)
	}
	if a.Backfill.IsZero() {
		return id, nil
	}
	return id, q.AddToBackfill(ctx, db.AddToBackfillParams{N: 1, ID: a.Backfill})
}

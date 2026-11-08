package grabs

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/grabs/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/trackers/seeding"
)

// TrackerFacts answers trackers.Grabs from the Grabs.
type TrackerFacts struct{}

// Stated is each Tracker's most recent Grab whose Release stated a
// minimum ratio or seeding time.
func (TrackerFacts) Stated(ctx context.Context, r storage.Reader[file]) (map[indexers.TrackerID]trackers.Stated, error) {
	rows, err := db.New(r).ListStatedFloors(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[indexers.TrackerID]trackers.Stated, len(rows))
	for _, row := range rows {
		out[row.TrackerID] = trackers.Stated{Rule: seeding.Stated(row.MinRatio, seconds(row.MinSeedSeconds)), At: row.CreatedAt.Time()}
	}
	return out, nil
}

// Unfinished is how many Grabs naming id are neither imported nor failed.
func (TrackerFacts) Unfinished(ctx context.Context, r storage.Reader[file], id indexers.TrackerID) (int, error) {
	n, err := db.New(r).CountUnfinishedOfTracker(ctx, id)
	return int(n), err
}

// Forget sets tracker_id NULL on every Grab naming id.
func (TrackerFacts) Forget(ctx context.Context, tx *storage.Tx[file], id indexers.TrackerID) error {
	return db.New(tx).ForgetTracker(ctx, id)
}

// Grabbed is how many Grabs of each Tracker were made since since.
func (TrackerFacts) Grabbed(ctx context.Context, r storage.Reader[file], since time.Time) (map[indexers.TrackerID]int, error) {
	rows, err := db.New(r).CountGrabsSince(ctx, storage.FromTime(since))
	if err != nil {
		return nil, err
	}
	out := make(map[indexers.TrackerID]int, len(rows))
	for _, row := range rows {
		out[row.TrackerID] = int(row.Grabs)
	}
	return out, nil
}

// seconds is a stated seeding time, nil for none.
func seconds(s *int64) *time.Duration {
	if s == nil {
		return nil
	}
	d := time.Duration(*s) * time.Second
	return &d
}

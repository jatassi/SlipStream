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
		var seed *time.Duration
		if row.MinSeedSeconds != nil {
			d := time.Duration(*row.MinSeedSeconds) * time.Second
			seed = &d
		}
		out[row.TrackerID] = trackers.Stated{Rule: seeding.Stated(row.MinRatio, seed), At: row.CreatedAt.Time()}
	}
	return out, nil
}

package indexers

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// Privacy is whether a Tracker admits anyone.
type Privacy string

const (
	Private     Privacy = "private"
	SemiPrivate Privacy = "semi-private"
	Public      Privacy = "public"
	Unknown     Privacy = "unknown"
)

// Tracker is a site that Indexers reach.
type Tracker struct {
	ID      TrackerID
	Name    string
	Privacy Privacy
	// Indexers are the Indexers that reach it, by name.
	Indexers []ID
}

// Trackers is every Tracker by name.
func (x *Indexers) Trackers(ctx context.Context, _ identity.Admin) ([]Tracker, error) {
	var out []Tracker
	err := x.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		q := db.New(r)
		rows, err := q.ListTrackers(ctx)
		if err != nil {
			return err
		}
		reach, err := reachOf(ctx, r)
		if err != nil {
			return err
		}
		out = make([]Tracker, len(rows))
		for i, row := range rows {
			out[i] = Tracker{ID: row.ID, Name: row.Name, Privacy: Privacy(row.Privacy), Indexers: reach.indexers[row.ID]}
		}
		return nil
	})
	return out, err
}

// reach is indexers_reach both ways.
type reach struct {
	indexers map[TrackerID][]ID
	trackers map[ID][]TrackerID
}

func reachOf(ctx context.Context, r storage.Reader[file]) (reach, error) {
	rows, err := db.New(r).ListReach(ctx)
	if err != nil {
		return reach{}, err
	}
	out := reach{indexers: map[TrackerID][]ID{}, trackers: map[ID][]TrackerID{}}
	for _, row := range rows {
		out.indexers[row.TrackerID] = append(out.indexers[row.TrackerID], row.IndexerID)
		out.trackers[row.IndexerID] = append(out.trackers[row.IndexerID], row.TrackerID)
	}
	return out, nil
}

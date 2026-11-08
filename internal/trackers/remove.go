package trackers

import (
	"context"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/trackers/internal/db"
)

// InUseError is what blocks the removal of the Tracker named Name: the
// Indexers that reach it, its live Torrents and its Grabs neither imported
// nor failed.
type InUseError struct {
	Name                      string
	Indexers, Torrents, Grabs int
}

func (e *InUseError) Error() string { return "the Tracker is in use" }

// Remove deletes Tracker id in one Write. It fails with ErrNoTracker, or
// with *InUseError while any Indexer reaches it, any live Torrent names it
// or any Grab neither imported nor failed names it. Otherwise its Grabs
// and removed Torrents keep their rows with no Tracker, and its rule,
// goal, routes, windows and goal runs are deleted with it.
func (t *Trackers) Remove(ctx context.Context, _ identity.Admin, id indexers.TrackerID) error {
	err := t.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		s, err := summaryIn(ctx, tx, id)
		if err != nil {
			return err
		}
		live, removed, err := downloadclients.OfTracker(ctx, tx, id)
		if err != nil {
			return err
		}
		unfinished, err := t.cfg.Grabs.Unfinished(ctx, tx, id)
		if err != nil {
			return err
		}
		if len(s.Indexers) > 0 || live > 0 || unfinished > 0 {
			return &InUseError{Name: s.Name, Indexers: len(s.Indexers), Torrents: live, Grabs: unfinished}
		}
		if err := t.cfg.Grabs.Forget(ctx, tx, id); err != nil {
			return err
		}
		if err := downloadclients.ForgetTracker(ctx, tx, id); err != nil {
			return err
		}
		if err := db.New(tx).DeleteGoalRuns(ctx, removed); err != nil {
			return err
		}
		return indexers.DeleteTracker(ctx, tx, id)
	})
	if err != nil {
		return err
	}
	t.cfg.TrackersChanged.Publish(ctx, indexers.TrackerChanged{TrackerID: id.String()})
	return nil
}

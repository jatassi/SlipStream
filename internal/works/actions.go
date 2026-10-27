package works

import (
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works/internal/db"
)

// MonitorWorks monitors or unmonitors every Item of each of ws in one
// Write, and publishes one works.changed naming the Works whose Items it
// changed. known holds each of ws a Work has, changed each whose Items
// moved.
func (w *Works) MonitorWorks(ctx context.Context, _ identity.Admin, ws []WorkID, on bool) (known, changed map[WorkID]bool, err error) {
	var present, moved []WorkID
	err = w.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		var err error
		if present, err = presentIn(ctx, q, ws); err != nil {
			return err
		}
		if moved, err = q.ListWorksToMonitor(ctx, db.ListWorksToMonitorParams{Monitored: on, WorkIds: present}); err != nil || len(moved) == 0 {
			return err
		}
		return q.MonitorWorks(ctx, db.MonitorWorksParams{Monitored: on, WorkIds: moved})
	})
	if err != nil {
		return nil, nil, err
	}
	w.Touched(ctx, moved)
	return set(present), set(moved), nil
}

// RefreshEach refreshes each of ws now, up to parallel at once, each as
// Refresh does in its own Write, and publishes one works.changed naming
// every Work that reads differently after. A Work already gone upstream
// is ErrGoneUpstream without a call to its source. Each of ws answers
// nil once it refreshed, changed or not, else its error.
func (w *Works) RefreshEach(ctx context.Context, _ identity.Admin, ws []WorkID, parallel int) map[WorkID]error {
	out := make(map[WorkID]error, len(ws))
	var (
		mu      sync.Mutex
		changed []WorkID
		wg      sync.WaitGroup
	)
	slots := make(chan struct{}, parallel)
	for _, id := range ws {
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			moved, err := w.refreshOnce(ctx, id)
			mu.Lock()
			defer mu.Unlock()
			out[id] = err
			if moved {
				changed = append(changed, id)
			}
		})
	}
	wg.Wait()
	slices.SortFunc(changed, func(a, b WorkID) int { return a.Compare(b.UUID) })
	w.Touched(ctx, changed)
	return out
}

func (w *Works) refreshOnce(ctx context.Context, id WorkID) (bool, error) {
	row, err := db.New(w.cfg.Library).GetWork(ctx, id)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return false, ErrNoWork
	case err != nil:
		return false, err
	case !row.GoneUpstreamAt.IsZero():
		return false, ErrGoneUpstream
	}
	return w.refresh(ctx, id, role.NodeID{})
}

// Touched publishes one works.changed naming ws, none when ws is empty:
// for a composer whose owner's event names no Work, such as a root
// change.
func (w *Works) Touched(ctx context.Context, ws []WorkID) {
	if len(ws) > 0 {
		w.cfg.Changed.Publish(ctx, ChangedOf(ws...))
	}
}

// SummariesOf reads the summary of each of ws a Work has, inside a
// caller's read.
func SummariesOf(ctx context.Context, r storage.Reader[file], ws []WorkID) (map[WorkID]Summary, error) {
	rows, err := db.New(r).ListWorksOf(ctx, ws)
	if err != nil {
		return nil, err
	}
	out := make(map[WorkID]Summary, len(rows))
	for _, row := range rows {
		out[row.ID] = Summary(row)
	}
	return out, nil
}

// presentIn is each of ws a Work has, in id order.
func presentIn(ctx context.Context, q *db.Queries, ws []WorkID) ([]WorkID, error) {
	rows, err := q.ListWorksOf(ctx, ws)
	ids := make([]WorkID, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return ids, err
}

func set(ids []WorkID) map[WorkID]bool {
	out := make(map[WorkID]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out
}

package imports

import (
	"context"
	"sync"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
)

// Run reconciles the unfinished Imports and starts the first scan of each
// root until ctx ends. At start and after each roots.changed or
// nodes.changed, every root Ready at a place no comparison of it has been
// at starts its unit, once per place while
// this process runs. A new root's first Ready report and a finished
// re-home are both a new place, and a crash before the comparison
// finished leaves the place uncompared, so this one rule starts all three.
func (im *Imports) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	defer wg.Wait()
	wg.Go(func() { im.reconcile(ctx) })
	woken := im.cfg.Hub.Watch(ctx, roots.ChangedEvent, nodes.ChangedEvent)
	asked := map[roots.ID]int64{}
	for {
		if err := im.firstScans(ctx, asked); err != nil && ctx.Err() == nil {
			im.cfg.Log.Warn("start the first library scans", "err", err)
		}
		select {
		case <-woken:
		case <-ctx.Done():
			return nil
		}
	}
}

func (im *Imports) firstScans(ctx context.Context, asked map[roots.ID]int64) error {
	ready, err := im.cfg.Roots.Ready(ctx)
	if err != nil {
		return err
	}
	compared, err := db.New(im.cfg.Library).ListCompared(ctx)
	if err != nil {
		return err
	}
	at := make(map[roots.ID]int64, len(compared))
	for _, c := range compared {
		at[c.RootID] = c.Place
	}
	var units []tasks.UnitKey
	var places []roots.Root
	for _, r := range ready {
		if at[r.ID] == r.Place || asked[r.ID] == r.Place {
			continue
		}
		units = append(units, unitKey(r.ID))
		places = append(places, r)
	}
	if len(units) == 0 {
		return nil
	}
	if _, err := im.cfg.Start(ctx, tasks.Start{Task: TaskID, Units: units, Trigger: tasks.ByEvent}); err != nil {
		return err
	}
	for _, r := range places {
		asked[r.ID] = r.Place
	}
	return nil
}

// Scan starts a run with root's unit, or adds it to the open run: Scan
// now. A unit already running runs once more after it. roots.ErrNotFound
// when no root has that id.
func (im *Imports) Scan(ctx context.Context, by identity.Admin, root roots.ID) (tasks.RunID, error) {
	if err := im.exists(ctx, root); err != nil {
		return 0, err
	}
	return im.start(ctx, by, root)
}

// ScanAll starts a run with every root's unit, or adds them to the open
// run: S54 (Import engine) calls it after carrying v1's Works.
func (im *Imports) ScanAll(ctx context.Context) (tasks.RunID, error) {
	st, err := im.cfg.Start(ctx, tasks.Start{Task: TaskID, Trigger: tasks.ByEvent})
	return st.Run, err
}

// MarkGone confirms root's stopped comparison and starts its unit, which
// marks up to the confirmed count gone: a worse vanish stops again.
// ErrNotStopped when root's last comparison didn't stop, roots.ErrNotFound
// when no root has that id.
func (im *Imports) MarkGone(ctx context.Context, by identity.Admin, root roots.ID) (tasks.RunID, error) {
	if err := im.exists(ctx, root); err != nil {
		return 0, err
	}
	if err := im.confirmGone(ctx, root); err != nil {
		return 0, err
	}
	return im.start(ctx, by, root)
}

func (im *Imports) start(ctx context.Context, by identity.Admin, root roots.ID) (tasks.RunID, error) {
	var account storage.UUID
	if a, ok := by.Account(); ok {
		account = a.Profile().ID.UUID
	}
	st, err := im.cfg.Start(ctx, tasks.Start{Task: TaskID, Units: []tasks.UnitKey{unitKey(root)}, Trigger: tasks.ByRunNow, By: account})
	return st.Run, err
}

func (im *Imports) exists(ctx context.Context, root roots.ID) error {
	return im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		_, err := roots.Get(ctx, r, root)
		return err
	})
}

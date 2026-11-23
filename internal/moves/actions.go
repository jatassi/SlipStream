package moves

import (
	"context"

	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// act changes unit id in one Write with fn, then has the runner pass.
func (m *Moves) act(ctx context.Context, id UnitID, fn func(ctx context.Context, tx *storage.Tx[file], u *Unit) error) error {
	var u Unit
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		if u, err = load(ctx, tx, id); err != nil {
			return err
		}
		if err := fn(ctx, tx, &u); err != nil {
			return err
		}
		return save(ctx, tx, u)
	})
	if err != nil {
		return err
	}
	m.touchTorrents(u)
	m.cfg.Changed.Publish(ctx, changedOf(u))
	m.Poke()
	return nil
}

// Cancel stops unit id before its switch; its undo removes what it placed.
// A unit past its switch is ErrUnitSwitched, one already ended nothing.
func (m *Moves) Cancel(ctx context.Context, id UnitID) error {
	now := m.now()
	return m.act(ctx, id, func(ctx context.Context, tx *storage.Tx[file], u *Unit) error {
		if p := u.Stage.Phase(); p == PhaseCleanup || p == PhaseDone {
			return ErrUnitSwitched
		}
		return m.cancelIn(ctx, tx, u, now)
	})
}

// CancelBatch cancels every unit of batch before its switch, answering
// how many it canceled and how many had already switched.
func (m *Moves) CancelBatch(ctx context.Context, batch BatchID) (canceled, switched int, err error) {
	now := m.now()
	var us []Unit
	err = m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if _, err := q.GetBatch(ctx, batch); err != nil {
			return notFound(err)
		}
		counts, err := q.CountBatchPhases(ctx, batch)
		if err != nil {
			return err
		}
		k := switchedIn(counts)
		switched = k.works + k.torrents
		rows, err := q.ListOpenUnitsOfBatch(ctx, batch)
		if err != nil {
			return err
		}
		if us, err = unitsOf(ctx, q, rows); err != nil {
			return err
		}
		for i := range us {
			if err := m.cancelIn(ctx, tx, &us[i], now); err != nil {
				return err
			}
			if err := save(ctx, tx, us[i]); err != nil {
				return err
			}
		}
		return q.CancelBatch(ctx, db.CancelBatchParams{At: storage.FromTime(now), ID: batch})
	})
	if err != nil || len(us) == 0 {
		return 0, switched, err
	}
	m.touchTorrents(us...)
	m.cfg.Changed.Publish(ctx, changedOf(us...))
	m.Poke()
	return len(us), switched, nil
}

// switchedIn is how many units of each kind counts has switched.
func switchedIn(counts []db.CountBatchPhasesRow) kinds {
	var k kinds
	for _, c := range counts {
		switch {
		case Phase(c.Phase) != PhaseCleanup && Phase(c.Phase) != PhaseDone:
		case c.Kind == "torrent":
			k.torrents += int(c.N)
		default:
			k.works += int(c.N)
		}
	}
	return k
}

// Retry queues a failed unit again once its undo finished.
func (m *Moves) Retry(ctx context.Context, id UnitID) error {
	return m.act(ctx, id, func(ctx context.Context, tx *storage.Tx[file], u *Unit) error {
		s, ok := u.Stage.(Failed)
		if !ok || s.Undone == nil {
			return ErrUnitNotFailed
		}
		return m.retryIn(ctx, tx, u, s)
	})
}

// SwitchNow has a unit waiting for playback or a Media Server switch
// anyway. Any other unit is ErrSwitchNotWaiting.
func (m *Moves) SwitchNow(ctx context.Context, id UnitID) error {
	now := m.now()
	return m.act(ctx, id, func(_ context.Context, _ *storage.Tx[file], u *Unit) error {
		s, ok := u.Stage.(Switching)
		if !ok || s.Wait != WaitPlaying && s.Wait != WaitMediaServers {
			return ErrSwitchNotWaiting
		}
		s.SwitchNow = &now
		u.Stage, u.Due = s, &now
		return nil
	})
}

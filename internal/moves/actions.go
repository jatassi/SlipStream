package moves

import (
	"context"

	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// act changes unit id in one Write with fn, then has the runner pass.
func (m *Moves) act(ctx context.Context, id UnitID, fn func(u *Unit) error) error {
	var u Unit
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		var err error
		if u, err = load(ctx, tx, id); err != nil {
			return err
		}
		if err := fn(&u); err != nil {
			return err
		}
		return save(ctx, tx, u)
	})
	if err != nil {
		return err
	}
	m.cfg.Changed.Publish(ctx, changedOf(u))
	m.Poke()
	return nil
}

// Cancel stops unit id before its switch; its undo removes what it placed.
// A unit past its switch is ErrUnitSwitched, one already ended nothing.
func (m *Moves) Cancel(ctx context.Context, id UnitID) error {
	now := m.now()
	return m.act(ctx, id, func(u *Unit) error {
		switch {
		case u.beforeSwitch():
			u.Stage, u.Due = Canceled{At: u.Stage.Phase()}, &now
		case u.Stage.Phase() == PhaseCleanup || u.Stage.Phase() == PhaseDone:
			return ErrUnitSwitched
		}
		return nil
	})
}

// CancelBatch cancels every unit of batch before its switch.
func (m *Moves) CancelBatch(ctx context.Context, batch BatchID) error {
	now := m.now()
	var us []Unit
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		if _, err := q.GetBatch(ctx, batch); err != nil {
			return notFound(err)
		}
		rows, err := q.ListOpenUnitsOfBatch(ctx, batch)
		if err != nil {
			return err
		}
		all, err := unitsOf(ctx, q, rows)
		if err != nil {
			return err
		}
		us = us[:0]
		for _, u := range all {
			if !u.beforeSwitch() {
				continue
			}
			u.Stage, u.Due = Canceled{At: u.Stage.Phase()}, &now
			if err := save(ctx, tx, u); err != nil {
				return err
			}
			us = append(us, u)
		}
		return q.CancelBatch(ctx, db.CancelBatchParams{At: storage.FromTime(now), ID: batch})
	})
	if err != nil || len(us) == 0 {
		return err
	}
	m.cfg.Changed.Publish(ctx, changedOf(us...))
	m.Poke()
	return nil
}

// Retry queues a failed unit again once its undo finished.
func (m *Moves) Retry(ctx context.Context, id UnitID) error {
	return m.act(ctx, id, func(u *Unit) error {
		s, ok := u.Stage.(Failed)
		if !ok || s.Undone == nil {
			return ErrUnitNotFailed
		}
		u.Stage, u.Due = Queued{Wait: WaitPairBusy, Since: m.now()}, nil
		return nil
	})
}

// SwitchNow has a unit waiting for playback or a Media Server switch
// anyway. Any other unit is ErrSwitchNotWaiting.
func (m *Moves) SwitchNow(ctx context.Context, id UnitID) error {
	now := m.now()
	return m.act(ctx, id, func(u *Unit) error {
		s, ok := u.Stage.(Switching)
		if !ok || s.Wait != WaitPlaying && s.Wait != WaitMediaServers {
			return ErrSwitchNotWaiting
		}
		s.SwitchNow = &now
		u.Stage, u.Due = s, &now
		return nil
	})
}

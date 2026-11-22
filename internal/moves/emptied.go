package moves

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/moves/internal/db"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// emptied is a source root of a batch that deletes the roots it empties,
// and the root most of its Works went to.
type emptied struct {
	From, To roots.ID
}

// deleteEmptied deletes the source roots of each batch that asked for it
// once every unit has ended, then finishes the batch. A root a Media
// Server still carries a Move from waits, and pending is true while one
// does. A root that still has files stays, and the batch notes it.
func (m *Moves) deleteEmptied(ctx context.Context) (pending bool, err error) {
	var batches []BatchID
	err = m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		var err error
		batches, err = db.New(r).ListEmptyingBatches(ctx)
		return err
	})
	if err != nil {
		return false, err
	}
	for _, b := range batches {
		waits, err := m.emptyBatch(ctx, b)
		if err != nil {
			return false, err
		}
		pending = pending || waits
	}
	return pending, nil
}

func (m *Moves) emptyBatch(ctx context.Context, batch BatchID) (waits bool, err error) {
	var rs []emptied
	err = m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := db.New(r).ListBatchRoots(ctx, batch)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if len(rs) > 0 && rs[len(rs)-1].From == row.FromRootID {
				continue
			}
			rs = append(rs, emptied{From: row.FromRootID, To: row.ToRootID})
		}
		for _, e := range rs {
			n, err := mediaservers.OpenMovesFrom(ctx, r, e.From)
			if err != nil {
				return err
			}
			waits = waits || n > 0
		}
		return nil
	})
	if err != nil || waits {
		return waits, err
	}
	kept := []roots.ID{}
	for _, e := range rs {
		ok, err := m.deleteRoot(ctx, e)
		if err != nil {
			return false, err
		}
		if !ok {
			kept = append(kept, e.From)
		}
	}
	notes, err := json.Marshal(kept)
	if err != nil {
		return false, err
	}
	err = m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).FinishBatchKeeping(ctx, db.FinishBatchKeepingParams{At: storage.FromTime(m.now()), Kept: string(notes), ID: batch})
	})
	if err == nil {
		m.cfg.Changed.Publish(ctx, Changed{BatchIDs: []string{batch.String()}, UnitIDs: []string{}, WorkIDs: []string{}})
	}
	return false, err
}

// deleteRoot deletes e.From with its Works going to e.To, first making
// e.To the default when e.From is. ok is false when the root still holds
// files or a Grab or a Move uses it. A root already gone is deleted.
func (m *Moves) deleteRoot(ctx context.Context, e emptied) (ok bool, err error) {
	var from roots.Root
	err = m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		from, err = roots.Get(ctx, r, e.From)
		return err
	})
	if errors.Is(err, roots.ErrNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if from.Default {
		if err := m.cfg.Roots.MakeDefault(ctx, identity.Admin{}, e.To); err != nil {
			return false, err
		}
	}
	err = m.cfg.Roots.Delete(ctx, identity.Admin{}, e.From, &e.To)
	if _, inUse := errors.AsType[*roots.InUseError](err); inUse {
		m.cfg.Log.Info("kept a root the Move didn't empty", "root", e.From.String(), "err", err.Error())
		return false, nil
	}
	if errors.Is(err, roots.ErrNotFound) {
		return true, nil
	}
	return err == nil, err
}

package conversions

import (
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/conversions/internal/order"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// MoveUp puts a queued Conversion at the head of Moved up.
func (c *Conversions) MoveUp(ctx context.Context, id ID) (ListRow, error) {
	return c.act(ctx, id, func(ctx context.Context, _ *storage.Tx[storage.Library], q *db.Queries, r Row) error {
		if r.Stage != StageQueued {
			return ErrNotWaiting
		}
		head, err := q.HeadOfLane(ctx, string(LaneMovedUp))
		if err != nil {
			return err
		}
		return q.MoveConversion(ctx, db.MoveConversionParams{ID: id, Lane: string(LaneMovedUp), Position: string(order.Before(order.Key(head)))})
	})
}

// MoveAfter puts a queued Conversion right after after in its lane, or at
// the head of Moved up when after is nil.
func (c *Conversions) MoveAfter(ctx context.Context, id ID, after *ID) (ListRow, error) {
	if after == nil {
		return c.MoveUp(ctx, id)
	}
	return c.act(ctx, id, func(ctx context.Context, _ *storage.Tx[storage.Library], q *db.Queries, r Row) error {
		if r.Stage != StageQueued {
			return ErrNotWaiting
		}
		if *after == id {
			return nil
		}
		at, err := readRow(ctx, q, *after)
		if err != nil {
			return err
		}
		if at.FinishedAt != nil {
			return ErrFinished
		}
		next, err := q.NextInLane(ctx, db.NextInLaneParams{Lane: string(at.Lane), Position: string(at.Position)})
		if err != nil {
			return err
		}
		pos := order.After(at.Position)
		if next != "" {
			pos = order.Between(at.Position, order.Key(next))
		}
		return q.MoveConversion(ctx, db.MoveConversionParams{ID: id, Lane: string(at.Lane), Position: string(pos)})
	})
}

// Retry queues a failed Conversion at the head of its lane with a fresh
// budget.
func (c *Conversions) Retry(ctx context.Context, id ID) (ListRow, error) {
	return c.act(ctx, id, func(ctx context.Context, tx *storage.Tx[storage.Library], _ *db.Queries, r Row) error {
		if r.Stage != StageFailed {
			return ErrNotFailed
		}
		return c.advance(ctx, tx, id, Event{On: Retry})
	})
}

// Cancel cancels an unfinished Conversion and declines its pair.
func (c *Conversions) Cancel(ctx context.Context, id ID) (ListRow, error) {
	return c.act(ctx, id, func(ctx context.Context, tx *storage.Tx[storage.Library], _ *db.Queries, _ Row) error {
		return c.advance(ctx, tx, id, Event{On: Canceled})
	})
}

// CancelGroup cancels every unfinished Conversion of a backfill's or a
// season's group, answering how many.
func (c *Conversions) CancelGroup(ctx context.Context, id GroupID) (int, error) {
	var ids []ID
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if ids, err = c.groupMembers(ctx, tx, id); err != nil {
			return err
		}
		for _, id := range ids {
			if err := c.advance(ctx, tx, id, Event{On: Canceled}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	c.acted(ctx, ids, nil)
	return len(ids), nil
}

// act runs do on Conversion id in one Write and answers the row as it
// stands after.
func (c *Conversions) act(ctx context.Context, id ID, do func(context.Context, *storage.Tx[storage.Library], *db.Queries, Row) error) (ListRow, error) {
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		r, err := readRow(ctx, q, id)
		if err != nil {
			return err
		}
		if r.FinishedAt != nil {
			return ErrFinished
		}
		return do(ctx, tx, q, r)
	})
	if err != nil {
		return ListRow{}, err
	}
	out, err := c.row(ctx, id)
	if err != nil {
		return ListRow{}, err
	}
	c.acted(ctx, []ID{id}, []works.WorkID{out.Row.Work})
	return out, nil
}

// acted publishes the change to ids of ws and wakes the reconciler and
// Transfers.
func (c *Conversions) acted(ctx context.Context, ids []ID, ws []works.WorkID) {
	if len(ids) == 0 {
		return
	}
	ch := Changed{ConversionIDs: make([]string, len(ids)), WorkIDs: make([]string, len(ws)), GroupIDs: []string{}}
	for i, id := range ids {
		ch.ConversionIDs[i] = id.String()
	}
	for i, w := range ws {
		ch.WorkIDs[i] = w.String()
	}
	c.cfg.Changed.Publish(ctx, ch)
	c.cfg.Transfers.Touch()
	c.cfg.Poke.Wake()
}

func readRow(ctx context.Context, q *db.Queries, id ID) (Row, error) {
	raw, err := q.GetConversion(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return Row{}, ErrNoConversion
	}
	if err != nil {
		return Row{}, fmt.Errorf("read the Conversion: %w", err)
	}
	return rowOf(raw)
}

package conversions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
)

// checks probes each checking output on its root's Node, outside any
// Write, and applies each verdict in a Write of its own. A probe that
// fails is tried again after Timings.Landing, at retry.
func (c *Conversions) checks(ctx context.Context) (moved []ID, retry time.Time, err error) {
	var rows []Row
	err = c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		raws, err := db.New(r).ListChecking(ctx)
		for _, raw := range raws {
			row, err := rowOf(raw)
			if err != nil {
				return err
			}
			rows = append(rows, row)
		}
		return err
	})
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("read the outputs to check: %w", err)
	}
	now := c.cfg.Clock()
	for _, row := range rows {
		c.rec.mu.Lock()
		at, waits := c.rec.retry[row.ID]
		c.rec.mu.Unlock()
		if waits && at.After(now) {
			retry = earliest(retry, at)
			continue
		}
		checked, err := c.check(ctx, row)
		if err != nil {
			return moved, retry, err
		}
		c.rec.mu.Lock()
		if checked {
			delete(c.rec.retry, row.ID)
		} else {
			c.rec.retry[row.ID] = now.Add(c.cfg.Timings.Landing)
			retry = earliest(retry, c.rec.retry[row.ID])
		}
		c.rec.mu.Unlock()
		if checked {
			moved = append(moved, row.ID)
		}
	}
	return moved, retry, nil
}

func earliest(a, b time.Time) time.Time {
	if a.IsZero() || b.Before(a) {
		return b
	}
	return a
}

// check probes row's output and applies its verdict, false when the
// probe couldn't run, such as while the root's Node is away.
func (c *Conversions) check(ctx context.Context, row Row) (bool, error) {
	var src files.Located
	var t targets.Target
	var rel storing.Rel
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		m, err := c.moveOf(ctx, tx, row.ID)
		if err != nil {
			return err
		}
		if src, err = m.source(ctx); err != nil {
			return err
		}
		made, err := m.target(ctx)
		if err != nil {
			return err
		}
		goals, err := goalsOf(ctx, tx)
		if err != nil {
			return err
		}
		t = goals[*row.Target]
		if src.Node == *row.Node {
			rel = holdingRel(row.ID, row.Attempt, made.Settings.Container)
			return nil
		}
		id, err := m.outputTransfer(ctx, 0)
		rel = transfers.StagedRel(id, 0)
		return err
	})
	if err != nil {
		return false, err
	}
	out, err := c.cfg.Probes.ProbeIn(ctx, src.Node, src.Root.UUID, rel)
	var unreadable *media.UnreadableError
	switch {
	case errors.As(err, &unreadable):
	case err != nil:
		c.cfg.Log.Info("check a made file later", "conversion", row.ID, "err", err)
		return false, nil
	}
	code := CodeStreams
	if unreadable == nil {
		code = verdict(src, t, out, c.cfg.Run.LengthTolerance)
	}
	err = c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return c.advance(ctx, tx, row.ID, Event{
			On: Checked, Verdict: Verdict{Code: code, Class: faults[code].Class}, Probe: &out, Basis: Basis{Stage: row.Stage, Attempt: row.Attempt},
		})
	})
	return true, err
}

// verdict is the root-side check of out, made from src for t: a length
// within tol and the streams the settings promise are a run fault's to
// miss, and a file t refuses is a source fault. "" passes.
func verdict(src files.Located, t targets.Target, out media.Record, tol converting.Tolerance) Code {
	s := t.Conversion.Settings
	diff := out.Duration - src.Media.Duration
	if diff < 0 {
		diff = -diff
	}
	switch {
	case diff > tol.Of(src.Media.Duration):
		return CodeShortOutput
	case out.Codec == "",
		!encode.CopiesVideo(src.Media, s) && out.Codec != string(s.Codec),
		len(src.Media.Audio) > 0 && len(out.Audio) == 0:
		return CodeStreams
	}
	made := encode.Rewrite(src.Read, out, encode.CopiesVideo(src.Media, s))
	if !targets.Judge(t, targets.Scored{Facts: made.Facts}, nil).Meets() {
		return CodeRefused
	}
	return ""
}

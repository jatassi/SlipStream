package imports

import (
	"context"
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/imports/internal/db"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
)

// probeFile reads f's media record where its bytes are, in its Node's probe
// slots, and keeps it: its quality and cut too. A timeout or an away Node
// holds f for the backoff. An ffprobe error holds it once, and the second
// fails it unreadable.
func (im *Imports) probeFile(ctx context.Context, imp Import, f File) error {
	im.hold(ctx, imp, f, AtProbe)
	var limit int64
	err := im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		set, err := db.New(r).GetSettings(ctx)
		limit = Probes.Value(set.Probes)
		return err
	})
	if err != nil {
		return err
	}
	if !im.probes.take(ctx, imp.Node, int(limit)) {
		return fmt.Errorf("wait for a probe slot: %w", ctx.Err())
	}
	rec, err := im.cfg.Probes.Probe(ctx, imp.Node, f.sourceOf(imp))
	im.probes.give(imp.Node)
	_, err = im.write(ctx, imp.ID, func(ctx context.Context, tx *storage.Tx[file], cur *Import, _ *after) error {
		g := fileAt(cur, f.Position)
		if g.Outcome != Pending || g.Probe != nil {
			return nil
		}
		unreadable, isUnreadable := errors.AsType[*media.UnreadableError](err)
		switch {
		case err == nil:
			return im.keepProbe(ctx, tx, cur, g, rec)
		case isUnreadable && g.Hold != nil && g.Hold.Code == ProbeError:
			g.Outcome, g.Reason, g.Hold = FailedOut, &Reason{Code: Unreadable, Line: unreadable.Line}, nil
		case isUnreadable:
			g.wait(ProbeError)
		case errors.Is(err, media.ErrProbeTimeout):
			g.wait(ProbeTimeout)
		case away(err):
			g.wait(NodeAway)
		default:
			return err
		}
		return nil
	})
	return err
}

// keepProbe keeps rec as g's probe, with the quality and cut its path
// reads as, a folder file's by its release name, which the Admin may
// edit. Its Import's Edition, when it has one, is its cut.
func (im *Imports) keepProbe(ctx context.Context, tx *storage.Tx[file], imp *Import, g *File, rec media.Record) error {
	cuts, err := im.cfg.Targets.CutsIn(ctx, tx)
	if err != nil {
		return err
	}
	g.Probe, g.Hold = &rec, nil
	if imp.Source == FromFolder {
		return im.readName(ctx, tx, imp.ContentType, imp.Edition, g)
	}
	read, quality := scanning.QualityOf(g.Path, im.cfg.Facets[imp.ContentType], cuts.For(imp.ContentType), rec.Resolution())
	g.Quality, g.Cut = quality, read.Facts.Cut
	if imp.Edition != nil {
		g.Cut = *imp.Edition
	}
	return nil
}

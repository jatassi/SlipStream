package conversions

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Report is imports' word on a conversion Import: placed as File, or ended,
// Declined when the Admin declined it.
type Report struct {
	Placed   bool
	File     files.ID
	Declined bool
}

// Followed is what Follow changed, published after its Write commits.
type Followed struct {
	Conversion ID
	Work       works.WorkID
}

// Follow takes a landing Conversion to done or canceled in imports' Write.
// An Import the Admin declined also declines the pair.
func (c *Conversions) Follow(ctx context.Context, tx *storage.Tx[storage.Library], id ID, r Report) (Followed, error) {
	ev := Event{On: Ended, Why: ImportEnded}
	switch {
	case r.Placed:
		ev = Event{On: Landed, File: r.File}
	case r.Declined:
		ev.Why = ImportDeclined
	}
	if err := c.advance(ctx, tx, id, ev); err != nil {
		return Followed{}, err
	}
	m, err := c.moveOf(ctx, tx, id)
	if err != nil {
		return Followed{}, err
	}
	return Followed{Conversion: id, Work: m.row.Work}, nil
}

// Followed publishes what Follow changed, after its Write commits, and
// wakes the reconciler for the pair it freed.
func (c *Conversions) Followed(ctx context.Context, fs ...Followed) {
	if len(fs) == 0 {
		return
	}
	ch := Changed{ConversionIDs: []string{}, WorkIDs: []string{}, GroupIDs: []string{}}
	for _, f := range fs {
		ch.ConversionIDs = append(ch.ConversionIDs, f.Conversion.String())
		ch.WorkIDs = append(ch.WorkIDs, f.Work.String())
	}
	c.cfg.Changed.Publish(ctx, ch)
	c.cfg.Poke.Wake()
}

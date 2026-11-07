package library

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var errPreview = errors.New("library: a preview rolls back")

// MonitoringOutcome is one Work's result: its Outcome, what moved (or
// would), and on a preview what each preset would monitor of it.
type MonitoringOutcome struct {
	Outcome
	Delta  works.Delta
	Counts works.PresetCounts
}

// SetMonitoring applies c to each of ws in one Write, all or none;
// preview rolls it back. A Work that moves nothing is skipped as already;
// an unknown one fails. One works.changed names every Work that moved.
// Searches nothing.
func (l *Library) SetMonitoring(ctx context.Context, _ identity.Admin, ws []works.WorkID, c works.Choice, preview bool) ([]MonitoringOutcome, error) {
	ids := distinct(ws)
	var (
		deltas map[works.WorkID]works.Delta
		counts map[works.WorkID]works.PresetCounts
	)
	err := l.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if deltas, counts, err = l.monitoringIn(ctx, tx, works.Only(ids), c, preview); err != nil {
			return err
		}
		if preview {
			return errPreview
		}
		return nil
	})
	if err != nil && !errors.Is(err, errPreview) {
		return nil, err
	}
	out := make([]MonitoringOutcome, len(ids))
	var moved []works.WorkID
	for i, id := range ids {
		d, ok := deltas[id]
		o := MonitoringOutcome{Outcome: done(id), Delta: d, Counts: counts[id]}
		switch {
		case !ok:
			o.Outcome = failed(id, works.ErrNoWork)
		case d.Zero():
			o.Outcome = skipped(id, ErrAlready)
		default:
			moved = append(moved, id)
		}
		out[i] = o
	}
	if !preview {
		l.cfg.Works.Touched(ctx, moved)
	}
	return out, nil
}

// monitoringIn applies c to each of ws in tx, and on a preview counts
// what each preset would monitor of each first.
func (l *Library) monitoringIn(ctx context.Context, tx *storage.Tx[storage.Library], ws works.Among[works.WorkID], c works.Choice, preview bool) (deltas map[works.WorkID]works.Delta, counts map[works.WorkID]works.PresetCounts, err error) {
	if !preview {
		deltas, err = l.applyIn(ctx, tx, ws, c)
		return deltas, nil, err
	}
	facts, err := l.cfg.Files.ItemFacts(ctx, tx, ws)
	if err != nil {
		return nil, nil, err
	}
	trees, err := works.OutlinesIn(ctx, tx, ws)
	if err != nil {
		return nil, nil, err
	}
	counts = make(map[works.WorkID]works.PresetCounts, len(trees))
	for _, t := range trees {
		counts[t.ID] = works.CountPresets(t, facts[t.ID])
	}
	deltas, err = l.cfg.Works.ApplyPreset(ctx, tx, ws, c, facts)
	return deltas, counts, err
}

// applyIn applies c to each of ws in tx, reading facts only for a preset
// that needs them.
func (l *Library) applyIn(ctx context.Context, tx *storage.Tx[storage.Library], ws works.Among[works.WorkID], c works.Choice) (map[works.WorkID]works.Delta, error) {
	var facts map[works.WorkID]works.Facts
	if c.Preset.NeedsFacts() {
		var err error
		if facts, err = l.cfg.Files.ItemFacts(ctx, tx, ws); err != nil {
			return nil, err
		}
	}
	return l.cfg.Works.ApplyPreset(ctx, tx, ws, c, facts)
}

// SetMonitored turns monitoring on or off for the levels and Items part
// names of Work id, each level standing for every live Item under it,
// leaving the Work's rule. A *works.NotInWorkError answers an id of
// another Work. Searches nothing.
func (l *Library) SetMonitored(ctx context.Context, by identity.Admin, id works.WorkID, part works.Within, on bool) error {
	_, err := l.cfg.Works.SetMonitored(ctx, by, id, part, on)
	return err
}

// SetOptions changes Work id's options; preview reports what the change
// would withdraw and writes nothing.
func (l *Library) SetOptions(ctx context.Context, by identity.Admin, id works.WorkID, o works.Optioning, preview bool) (works.OptionsChange, error) {
	return l.cfg.Works.SetOptions(ctx, by, id, o, preview)
}

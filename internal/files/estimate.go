package files

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/files/internal/db"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Estimate is the disk one Target of a Profile costs: the MiB per minute
// the files filling the Target average, over the runtime of the Profile's
// Works.
type Estimate struct {
	MiBPerMinute float64
	Bytes        int64
}

// Estimates is each of p's Targets' Estimate, absent for a Target no file
// of known runtime fills yet. An Item whose runtime is unknown adds no
// minutes, to a rate or to the Works' runtime. Runtimes are read in one
// batch per Content Type.
func (f *Files) Estimates(ctx context.Context, _ identity.Admin, p targets.Profile) (map[targets.TargetID]Estimate, error) {
	ids := make([]targets.TargetID, len(p.Entries))
	for i, e := range p.Entries {
		ids[i] = e.Target
	}
	var out map[targets.TargetID]Estimate
	err := f.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		ws, err := targets.WorksOf(ctx, r, p.ID)
		if err != nil {
			return err
		}
		fills, err := db.New(r).ListFillsOfTargets(ctx, ids)
		if err != nil {
			return err
		}
		runtime, err := f.runtimes(ctx, r, p.ContentType, ws, fills)
		if err != nil {
			return err
		}
		minutes, err := runtimeOf(ctx, r, ws, runtime)
		if err != nil {
			return err
		}
		out = estimates(fills, runtime, minutes)
		return nil
	})
	return out, err
}

// runtimes reads the known runtime of the Items of ws, of Content Type
// typ and of the Works the fills fill, one batch per Content Type.
func (f *Files) runtimes(ctx context.Context, r storage.Reader[file], typ works.TypeID, ws []works.WorkID, fills []db.ListFillsOfTargetsRow) (map[works.ItemID]time.Duration, error) {
	all, err := works.Summaries(ctx, r)
	if err != nil {
		return nil, err
	}
	typeOf := make(map[works.WorkID]works.TypeID, len(all))
	for _, w := range all {
		typeOf[w.ID] = w.ContentType
	}
	byType := map[works.TypeID][]works.WorkID{typ: ws}
	listed := map[works.WorkID]bool{}
	for _, w := range ws {
		listed[w] = true
	}
	for _, fill := range fills {
		if t := typeOf[fill.WorkID]; !listed[fill.WorkID] {
			listed[fill.WorkID] = true
			byType[t] = append(byType[t], fill.WorkID)
		}
	}
	out := map[works.ItemID]time.Duration{}
	for _, t := range slices.Sorted(maps.Keys(byType)) {
		timings, err := f.cfg.Targets.TimingsIn(ctx, r, t, byType[t])
		if err != nil {
			return nil, err
		}
		for item, timing := range timings {
			if d, known := timing.Runtime.Get(); known && d > 0 {
				out[item] = d
			}
		}
	}
	return out, nil
}

// runtimeOf is the minutes of ws's Items whose runtime is known.
func runtimeOf(ctx context.Context, r storage.Reader[file], ws []works.WorkID, runtime map[works.ItemID]time.Duration) (float64, error) {
	if len(ws) == 0 {
		return 0, nil
	}
	items, err := works.ItemsOf(ctx, r, ws)
	if err != nil {
		return 0, err
	}
	var minutes float64
	for _, of := range items {
		for _, it := range of {
			minutes += runtime[it.ID].Minutes()
		}
	}
	return minutes, nil
}

func estimates(fills []db.ListFillsOfTargetsRow, runtime map[works.ItemID]time.Duration, minutes float64) map[targets.TargetID]Estimate {
	type sum struct{ mib, minutes float64 }
	sums := map[targets.TargetID]sum{}
	for _, fill := range fills {
		if d, ok := runtime[fill.ItemID]; ok {
			s := sums[fill.TargetID]
			sums[fill.TargetID] = sum{s.mib + float64(fill.Size)/(1<<20), s.minutes + d.Minutes()}
		}
	}
	out := make(map[targets.TargetID]Estimate, len(sums))
	for id, s := range sums {
		rate := s.mib / s.minutes
		out[id] = Estimate{MiBPerMinute: rate, Bytes: int64(rate * minutes * (1 << 20))}
	}
	return out
}

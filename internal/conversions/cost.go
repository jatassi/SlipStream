package conversions

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/conversions/internal/plan"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Cost is what making made would take: each pair's bytes, and its
// seconds as the forecast runs it after the queue in the backfill lane.
// A Work none of whose pairs any Node can convert today is Unconvertible.
func (s Seam) Cost(ctx context.Context, tx *storage.Tx[storage.Library], made []files.MadePair) (files.MadeCost, error) {
	c := s.Conversions()
	plans, err := c.forecastPairs(ctx, tx, made)
	if err != nil {
		return files.MadeCost{}, err
	}
	var out files.MadeCost
	at := map[works.WorkID]int{}
	convertible := map[works.WorkID]bool{}
	for i, p := range made {
		k, ok := at[p.Work]
		if !ok {
			k, at[p.Work] = len(out.Rows), len(out.Rows)
			out.Rows = append(out.Rows, files.MadeRow{Work: p.Work})
		}
		pp := plans[i]
		out.Rows[k].Files++
		out.Rows[k].Bytes += pp.bytes
		out.Rows[k].Seconds += pp.seconds
		out.Bytes += pp.bytes
		out.Seconds += pp.seconds
		convertible[p.Work] = convertible[p.Work] || !pp.unconvertible
		if pp.node != "" && !slices.Contains(out.Nodes, pp.node) {
			out.Nodes = append(out.Nodes, pp.node)
		}
	}
	summaries, err := works.SummariesOf(ctx, tx, slices.Collect(maps.Keys(at)))
	if err != nil {
		return files.MadeCost{}, fmt.Errorf("read the Works: %w", err)
	}
	for i, r := range out.Rows {
		out.Rows[i].Added = summaries[r.Work].AddedAt.Time()
		if !convertible[r.Work] {
			out.Unconvertible = append(out.Unconvertible, r.Work)
		}
	}
	out.Works = len(out.Rows)
	return out, nil
}

// pairPlan is one pair's estimate.
type pairPlan struct {
	bytes, seconds int64
	node           string
	unconvertible  bool
}

// forecastPairs runs the forecast to the horizon over the queue with
// pairs appended in order, and answers each pair's estimate.
func (c *Conversions) forecastPairs(ctx context.Context, tx *storage.Tx[storage.Library], pairs []files.MadePair) ([]pairPlan, error) {
	s, err := c.snapshot(ctx, tx, c.roster())
	if err != nil {
		return nil, err
	}
	out := make([]pairPlan, len(pairs))
	ids := make([]storage.UUID, len(pairs))
	rooms := map[roots.ID]int64{}
	for i, p := range pairs {
		ids[i] = storage.NewUUID()
		src, err := c.cfg.Files.Located(ctx, tx, p.Source)
		switch {
		case errors.Is(err, files.ErrNoFile):
			continue
		case err != nil:
			return nil, fmt.Errorf("read a source file: %w", err)
		}
		s.srcs[src.ID] = src
		row := Row{ID: ID{UUID: ids[i]}, Stage: StageQueued, SourceFile: &p.Source, Target: &p.Target, NotBefore: s.Now}
		j, err := c.jobOf(ctx, tx, s, row, rooms)
		if err != nil {
			return nil, err
		}
		out[i].bytes = j.OutBytes
		s.Jobs = append(s.Jobs, j)
	}
	_, f := plan.Run(s.Snapshot, s.Now.Add(c.cfg.Timings.Horizon))
	for i, id := range ids {
		p, ok := f[id]
		out[i].unconvertible = !ok || p.Wait.Code == plan.NoNode || p.Wait.Code == plan.NoSource
		if !ok {
			continue
		}
		if p.Start != nil && p.Finish != nil {
			out[i].seconds = int64(p.Finish.Sub(*p.Start).Seconds())
		}
		if p.Node != nil {
			out[i].node = s.names[*p.Node]
		}
	}
	return out, nil
}

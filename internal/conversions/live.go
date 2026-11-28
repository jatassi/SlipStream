package conversions

import (
	"context"
	"math"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// sample publishes conversions.live for the runs on nodes, while an Admin
// listens: each run's share done, speed and time left, and each Node's
// running count.
func (c *Conversions) sample(ctx context.Context, nodes map[role.NodeID]bool) {
	if len(nodes) == 0 || !c.cfg.Live.Listening() {
		return
	}
	out := LiveSample{Rows: []LiveRow{}, Nodes: []LiveNode{}}
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		for node := range nodes {
			if err := c.sampleOn(ctx, r, node, &out); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.cfg.Log.Warn("sample the running Conversions", "err", err)
		return
	}
	c.cfg.Live.Publish(ctx, out)
}

// sampleOn adds node's runs and running count to out.
func (c *Conversions) sampleOn(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID, out *LiveSample) error {
	raws, err := db.New(r).ListOnNode(ctx, node)
	if err != nil {
		return err
	}
	lengths := map[ID]time.Duration{}
	for _, raw := range raws {
		if raw.Stage != string(StageRunning) && raw.Stage != string(StagePaused) {
			continue
		}
		src, err := c.cfg.Files.Located(ctx, r, raw.SourceFileID)
		if err != nil {
			return err
		}
		lengths[raw.ID] = src.Media.Duration
	}
	out.Nodes = append(out.Nodes, LiveNode{ID: node.String(), Running: len(lengths)})
	f, _ := c.cfg.Control().Facts(node)
	for id, p := range factsOf(f, len(lengths), lengths).Progress {
		out.Rows = append(out.Rows, liveRow(id, p, lengths[id]))
	}
	return nil
}

func liveRow(id ID, p RunProgress, length time.Duration) LiveRow {
	row := LiveRow{ID: id.String(), Speed: p.Speed, Paused: p.Paused}
	if length > 0 {
		row.Percent = math.Min(100, 100*float64(p.Out)/float64(length))
	}
	if p.Speed > 0 {
		row.SecondsLeft = int64(max(0, (length-p.Out).Seconds()/p.Speed))
	}
	return row
}

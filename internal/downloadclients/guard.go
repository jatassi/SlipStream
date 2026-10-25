package downloadclients

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/health"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// Volume is a Node's downloads folder volume as its client last sampled
// it.
type Volume struct {
	Node role.NodeID
	// Mount is as the Node names it, as Storage names the same volume, so
	// Health judges the two as one.
	Mount     string
	Name      string
	Total     int64
	Free      int64
	SampledAt time.Time
}

// Guard is the disk guard holding new downloads on a Node: its downloads
// volume is below S18 (Health)'s error level for it.
type Guard struct {
	Volume     Volume
	ErrorBelow health.Resolved
}

// Words is what the guard on node says, as Health and the client's card
// both say it: a sentence, then the hint that explains it.
func (g Guard) Words(node string) (sentence, hint string) {
	return "New downloads on " + node + " wait for disk space.",
		fmt.Sprintf("%s has %s left, below its error level of %s. Free up space there, or move the downloads folder to a volume with more room. Downloads already running continue.",
			g.Volume.Name, bytesWords(g.Volume.Free), bytesWords(g.ErrorBelow.Bytes))
}

func bytesWords(n int64) string { return setting.Amount{N: n, Unit: setting.BytesUnit}.Words() }

// guardOn is the disk guard on v's Node at l, false once v is not below
// its error level: the guard lifts on the first sample above it, as Health
// judges a volume.
func guardOn(v Volume, l health.Levels) (Guard, bool) {
	lv := l.For(v.Node, v.Mount, v.Total)
	return Guard{Volume: v, ErrorBelow: lv.Error}, v.Total > 0 && lv.Judge(v.Free) == health.Fail
}

// guardsIn is the disk guard on every Node whose recorded downloads volume
// is below its error level.
func guardsIn(ctx context.Context, rd storage.Reader[file]) (map[role.NodeID]Guard, error) {
	vols, err := volumesIn(ctx, rd)
	if err != nil {
		return nil, err
	}
	levels, err := health.LevelsIn(ctx, rd)
	if err != nil {
		return nil, err
	}
	out := map[role.NodeID]Guard{}
	for _, v := range vols {
		if g, on := guardOn(v, levels); on {
			out[v.Node] = g
		}
	}
	return out, nil
}

func volumesIn(ctx context.Context, rd storage.Reader[file]) ([]Volume, error) {
	rows, err := db.New(rd).ListVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("read the downloads volumes: %w", err)
	}
	out := make([]Volume, len(rows))
	for i, r := range rows {
		out[i] = Volume{Node: r.NodeID, Mount: r.Mount, Name: r.Name, Total: r.Total, Free: r.Free, SampledAt: r.SampledAt.Time()}
	}
	return out, nil
}

// disk is what Clients last told the Nodes of the disk guard, and when it
// last recorded a sample.
type disk struct {
	mu sync.Mutex
	// guarded is each Node the guard held when its intent was last touched
	// for it.
	guarded map[role.NodeID]bool
	// recorded is closed and replaced at each recorded sample.
	recorded chan struct{}
}

// record keeps the downloads volume f carries when it is newer than the
// one recorded.
func (c *Clients) record(ctx context.Context, node role.NodeID, f *dlv1.Facts) error {
	v := f.GetDownloads()
	if v == nil {
		return nil
	}
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return db.New(tx).PutVolume(ctx, db.PutVolumeParams{
			NodeID: node, Mount: v.GetMount(), Name: v.GetName(), Total: v.GetTotal(), Free: v.GetFree(),
			SampledAt: storage.FromTime(time.UnixMilli(v.GetSampledAtUnixMs())),
		})
	})
	if err != nil {
		return fmt.Errorf("record node %s's downloads volume: %w", node, err)
	}
	c.disk.mu.Lock()
	defer c.disk.mu.Unlock()
	close(c.disk.recorded)
	c.disk.recorded = make(chan struct{})
	return nil
}

// guard touches the intent of each of nodes whose disk guard moved since
// Clients last told it, and answers those.
func (c *Clients) guard(ctx context.Context, nodes []role.NodeID) ([]role.NodeID, error) {
	var guards map[role.NodeID]Guard
	err := c.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		guards, err = guardsIn(ctx, rd)
		return err
	})
	if err != nil {
		return nil, err
	}
	var moved []role.NodeID
	c.disk.mu.Lock()
	for _, node := range nodes {
		_, on := guards[node]
		if on != c.disk.guarded[node] {
			c.disk.guarded[node] = on
			moved = append(moved, node)
		}
	}
	c.disk.mu.Unlock()
	if len(moved) > 0 {
		c.cfg.Touch(moved...)
	}
	return moved, nil
}

// levelsChanged follows the storage levels: a guard they move goes to its
// Node and its card.
func (c *Clients) levelsChanged(ctx context.Context) {
	moved, err := c.guard(ctx, c.cfg.Roster.Serving(role.Downloading))
	if err != nil {
		c.cfg.Log.Error("judge the downloads volumes at the new storage levels", "err", err.Error())
		return
	}
	if len(moved) > 0 {
		c.cardsChanged(moved...)
	}
}

// CadenceChanged tells every downloading Node's intent that Storage's
// cadence, at which it samples its downloads volume, changed.
func (c *Clients) CadenceChanged() { c.cfg.Touch(c.cfg.Roster.Serving(role.Downloading)...) }

// sampleVolume has node's client sample its downloads volume and returns
// once the sample is recorded.
func (c *Clients) sampleVolume(ctx context.Context, node role.NodeID) error {
	if !c.cfg.Roster.InStep(node) {
		return fmt.Errorf("%w: node %s is out of step", health.ErrUnreachable, node)
	}
	res, err := c.cfg.Remote(node).Sample(ctx, &dlv1.SampleRequest{})
	if err != nil {
		if _, notSent := rpc.NotSent(err); notSent || rpc.Lost(err) {
			return fmt.Errorf("%w: %w", health.ErrUnreachable, err)
		}
		return fmt.Errorf("sample node %s's downloads volume: %w", node, err)
	}
	at := time.UnixMilli(res.GetSampledAtUnixMs())
	for {
		c.disk.mu.Lock()
		recorded := c.disk.recorded
		c.disk.mu.Unlock()
		var vols []Volume
		err := c.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
			var err error
			vols, err = volumesIn(ctx, rd)
			return err
		})
		if err != nil {
			return err
		}
		for _, v := range vols {
			if v.Node == node && !v.SampledAt.Before(at) {
				return nil
			}
		}
		select {
		case <-recorded:
		case <-ctx.Done():
			return fmt.Errorf("await node %s's downloads volume: %w", node, ctx.Err())
		}
	}
}

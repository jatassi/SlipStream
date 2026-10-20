package downloadclients

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// LiveTick is what moved in the latest live sample.
type LiveTick struct {
	At        time.Time `json:"at"`
	Rows      []LiveRow `json:"rows" nullable:"false" doc:"The Torrents whose rates, progress or peers moved"`
	Downloads Aggregate `json:"downloads"`
}

type LiveRow struct {
	ID      string `json:"id" format:"uuid"`
	DownBps int64  `json:"downBps" doc:"Payload bytes a second it downloads"`
	UpBps   int64  `json:"upBps" doc:"Payload bytes a second it uploads"`
	Done    int64  `json:"done" doc:"Bytes downloaded"`
	Wanted  int64  `json:"wanted" doc:"Bytes its selected files hold"`
	Peers   int    `json:"peers"`
	Seeds   int    `json:"seeds"`
}

// Aggregate is every download at once: the Torrents of the Downloading
// tab.
type Aggregate struct {
	Done      int64 `json:"done" doc:"Bytes the downloads hold, live while sampled"`
	Wanted    int64 `json:"wanted" doc:"Bytes the downloads want"`
	AllPaused bool  `json:"allPaused" doc:"There are downloads and every one is paused"`
}

var LiveEvent = live.AdminEvent[LiveTick]("torrents.live")

type LivePublisher interface {
	Publish(ctx context.Context, v LiveTick)
	// Listening reports whether an Admin's live stream is open.
	Listening() bool
}

// LiveConfig is how Clients watches the Nodes' clients while an Admin
// listens.
type LiveConfig struct {
	// Watched is the Nodes to watch: in step and serving Downloading.
	Watched func() []role.NodeID
	Event   LivePublisher
	// Sample is how often the Nodes sample and torrents.live goes out.
	Sample time.Duration
}

// Sample is one Torrent as its Node last sampled it.
type Sample struct {
	DownBps, UpBps int64
	Done, Wanted   int64
	Peers, Seeds   int
}

// watching is what Clients knows of the watched Nodes. It exists only
// while an Admin listens.
type watching struct {
	mu    sync.Mutex
	nodes map[role.NodeID]*watched
	// moved is each Torrent sampled since the last torrents.live, and
	// shown the aggregate it carried.
	moved map[ID]Sample
	shown Aggregate
}

type watched struct {
	stop     context.CancelFunc
	down, up int64
	torrents map[ID]Sample
	sampled  bool
}

// Run parts the Torrents of each Node the Admin wrote Downloading off,
// at its start and at each change of the roster. It watches every Node
// the config names while an Admin listens, and publishes torrents.live
// once each sample with the rows that moved. Nothing is watched, and no
// Node samples, while no Admin listens.
func (c *Clients) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	defer wg.Wait()
	defer c.unwatch()
	tick := time.NewTicker(c.cfg.Live.Sample)
	defer tick.Stop()
	roster := c.cfg.Roster.Changed()
	c.settleWriteOffs(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-roster:
			roster = c.cfg.Roster.Changed()
			c.settleWriteOffs(ctx)
			continue
		case <-tick.C:
		}
		if !c.cfg.Live.Event.Listening() {
			c.unwatch()
			continue
		}
		c.follow(ctx, &wg)
		c.publishLive(ctx)
	}
}

// follow watches each Node to watch and stops watching the rest.
func (c *Clients) follow(ctx context.Context, wg *sync.WaitGroup) {
	want := c.cfg.Live.Watched()
	w := &c.watching
	w.mu.Lock()
	defer w.mu.Unlock()
	for node, n := range w.nodes {
		if !slices.Contains(want, node) {
			n.stop()
			delete(w.nodes, node)
		}
	}
	for _, node := range want {
		if w.nodes[node] != nil {
			continue
		}
		nctx, stop := context.WithCancel(ctx)
		n := &watched{stop: stop, torrents: map[ID]Sample{}}
		w.nodes[node] = n
		wg.Go(func() { c.watch(nctx, node, n) })
	}
}

func (c *Clients) unwatch() {
	w := &c.watching
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, n := range w.nodes {
		n.stop()
	}
	clear(w.nodes)
	clear(w.moved)
	w.shown = Aggregate{}
}

// watch calls node's Watch back to back until ctx ends, waiting a sample
// after a failed call.
func (c *Clients) watch(ctx context.Context, node role.NodeID, n *watched) {
	remote := c.cfg.Remote(node)
	var after uint64
	for {
		res, err := remote.Watch(ctx, &dlv1.WatchRequest{After: after})
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			c.cfg.Log.Debug("watch a Node's torrent client", "node", node.String(), "err", err.Error())
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.cfg.Live.Sample):
			}
			continue
		}
		c.fold(node, n, after, res)
		after = res.GetSample()
	}
}

// fold takes in one of a Node's samples. One that doesn't follow sample
// after replaces what the Node's earlier samples said.
func (c *Clients) fold(node role.NodeID, n *watched, after uint64, res *dlv1.WatchResponse) {
	w := &c.watching
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.nodes[node] != n {
		return
	}
	if res.GetSample() != after+1 {
		clear(n.torrents)
	}
	n.down, n.up, n.sampled = res.GetDownBps(), res.GetUpBps(), true
	for _, row := range res.GetRows() {
		id, err := idOf(row.GetId())
		if err != nil {
			continue
		}
		s := Sample{
			DownBps: row.GetDownBps(), UpBps: row.GetUpBps(), Done: row.GetDone(), Wanted: row.GetWanted(),
			Peers: int(row.GetPeers()), Seeds: int(row.GetSeeds()),
		}
		n.torrents[id] = s
		w.moved[id] = s
	}
	for _, gone := range res.GetGone() {
		if id, err := idOf(gone); err == nil {
			delete(n.torrents, id)
			delete(w.moved, id)
		}
	}
}

func (c *Clients) publishLive(ctx context.Context) {
	agg, err := c.Downloads(ctx)
	if err != nil {
		if ctx.Err() == nil {
			c.cfg.Log.Error("add up the downloads", "err", err.Error())
		}
		return
	}
	w := &c.watching
	w.mu.Lock()
	tick := LiveTick{At: time.Now(), Rows: []LiveRow{}, Downloads: agg}
	for _, id := range slices.SortedFunc(maps.Keys(w.moved), func(a, b ID) int { return a.Compare(b.UUID) }) {
		s := w.moved[id]
		tick.Rows = append(tick.Rows, LiveRow{
			ID: id.String(), DownBps: s.DownBps, UpBps: s.UpBps, Done: s.Done, Wanted: s.Wanted, Peers: s.Peers, Seeds: s.Seeds,
		})
	}
	publish := len(tick.Rows) > 0 || agg != w.shown
	clear(w.moved)
	w.shown = agg
	w.mu.Unlock()
	if publish {
		c.cfg.Live.Event.Publish(ctx, tick)
	}
}

// Downloads adds up the downloads, each at its live sample while it has
// one, else as its Node last reported it.
func (c *Clients) Downloads(ctx context.Context) (Aggregate, error) {
	rows, err := db.New(c.cfg.Library).ListDownloads(ctx)
	if err != nil {
		return Aggregate{}, err
	}
	agg, downloads, paused := Aggregate{}, 0, 0
	for _, row := range rows {
		rep, err := reportOf(row.Report)
		if err != nil {
			return Aggregate{}, err
		}
		f := Facts{Wanted: Wanted(row.Wanted), Report: rep}
		s := StateOf(f)
		if TabsOf(f, s)&TabDownloading == 0 {
			continue
		}
		downloads++
		if s == StatePaused {
			paused++
		}
		done := rep.done(row.Size)
		if l, ok := c.Sample(row.ID); ok {
			done = l.Done
		}
		agg.Done += done
		agg.Wanted += row.Size
	}
	agg.AllPaused = downloads > 0 && paused == downloads
	return agg, nil
}

// Sample is Torrent id's latest live sample, false while it has none.
func (c *Clients) Sample(id ID) (Sample, bool) {
	w := &c.watching
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, n := range w.nodes {
		if s, ok := n.torrents[id]; ok {
			return s, true
		}
	}
	return Sample{}, false
}

// Live is node's client as its samples show it, false until its first
// sample while an Admin listens.
func (c *Clients) Live(node role.NodeID) (nodes.LiveClient, bool) {
	w := &c.watching
	w.mu.Lock()
	defer w.mu.Unlock()
	n := w.nodes[node]
	if n == nil || !n.sampled {
		return nodes.LiveClient{}, false
	}
	out := nodes.LiveClient{DownBps: n.down, UpBps: n.up, Torrents: make(map[storage.UUID]nodes.LiveTorrent, len(n.torrents))}
	for id, s := range n.torrents {
		out.Torrents[id.UUID] = nodes.LiveTorrent{Done: s.Done, Wanted: s.Wanted, Peers: s.Peers}
	}
	return out, true
}

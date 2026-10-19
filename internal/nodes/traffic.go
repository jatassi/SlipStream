package nodes

import (
	"context"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Traffic is one sample of the bytes moving between and through the Nodes.
type Traffic struct {
	At    time.Time     `json:"at"`
	Links []LinkTraffic `json:"links" nullable:"false" doc:"Every live link, between the Controller and a Worker"`
	Nodes []NodeTraffic `json:"nodes" nullable:"false" doc:"Every Node whose torrent client reports, in id order"`
}

type LinkTraffic struct {
	A     string `json:"a" format:"uuid" doc:"The Controller"`
	B     string `json:"b" format:"uuid" doc:"The Worker"`
	ABBps int64  `json:"abBps" doc:"Bytes a second from a to b, TLS records included"`
	BABps int64  `json:"baBps" doc:"Bytes a second from b to a, TLS records included"`
	RTTMs int64  `json:"rttMs" doc:"The link's round trip; 0 until its first ping is answered"`
}

type NodeTraffic struct {
	ID       string           `json:"id" format:"uuid"`
	DownBps  int64            `json:"downBps" doc:"Payload bytes a second its torrents download"`
	UpBps    int64            `json:"upBps" doc:"Payload bytes a second its torrents upload"`
	Torrents []TorrentTraffic `json:"torrents" nullable:"false" doc:"Each Grab whose torrent its client keeps"`
}

type TorrentTraffic struct {
	GrabID string `json:"grabId" format:"uuid"`
	Done   int64  `json:"done"`
	Peers  int    `json:"peers"`
}

func (t Traffic) moving() bool {
	for _, l := range t.Links {
		if l.ABBps > 0 || l.BABps > 0 {
			return true
		}
	}
	for _, n := range t.Nodes {
		if n.DownBps > 0 || n.UpBps > 0 {
			return true
		}
	}
	return false
}

var TrafficEvent = live.AdminEvent[Traffic]("nodes.traffic")

// TrafficPublisher is TrafficEvent bound to the hub.
type TrafficPublisher interface {
	Publish(ctx context.Context, v Traffic)
	// Listening reports whether an Admin's live stream is open.
	Listening() bool
}

// Downloads is each Node's torrent client as the Downloading Role's
// control half last heard from it.
type Downloads interface {
	Facts(node ID) (*dlv1.Facts, bool)
}

// Grabbed is a Grab whose torrent a Node's client keeps.
type Grabbed struct {
	Grab     storage.UUID
	InfoHash bencode.InfoHash
	Title    string
}

// meter turns the link's byte counters and each client's torrent counters
// into rates. Link counters are read at each sample; a client's counters
// arrive with its facts, so its rate spans its last two facts and reads
// zero once no facts came for quiet.
type meter struct {
	quiet time.Duration

	mu      sync.Mutex
	links   map[ID]sampled
	clients map[ID]*client
	// kept is each Node's Grabs by the hash of the torrent they share.
	kept map[ID]map[bencode.InfoHash][]storage.UUID
	// moving: the last sample published showed bytes moving, so the next
	// is published even when nothing moves, and the canvas comes to rest.
	moving bool
}

type sampled struct {
	at             time.Time
	sent, received int64
}

type client struct {
	at       time.Time
	progress map[bencode.InfoHash]progress
	down, up int64
}

type progress struct{ done, uploaded int64 }

func newMeter(quiet time.Duration) *meter {
	return &meter{quiet: quiet, links: map[ID]sampled{}, clients: map[ID]*client{}, kept: map[ID]map[bencode.InfoHash][]storage.UUID{}}
}

// link is the rate each way of a Worker's live session since the last
// sample, zero for its first or a new session.
func (m *meter) link(id ID, now time.Time, st link.Stats) (sent, received int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	was, ok := m.links[id]
	m.links[id] = sampled{at: now, sent: st.Sent, received: st.Received}
	if !ok || st.Sent < was.sent || st.Received < was.received || !now.After(was.at) {
		return 0, 0
	}
	d := now.Sub(was.at)
	return perSecond(st.Sent-was.sent, d), perSecond(st.Received-was.received, d)
}

func (m *meter) linkDown(id ID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.links, id)
}

func (m *meter) arrive(node ID, at time.Time, f *dlv1.Facts) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := m.clients[node]
	if c == nil {
		c = &client{}
		m.clients[node] = c
	}
	next := make(map[bencode.InfoHash]progress, len(f.GetTorrents()))
	var down, up int64
	for _, t := range f.GetTorrents() {
		h := bencode.InfoHash(t.GetInfoHash())
		p := progress{done: t.GetDone(), uploaded: t.GetUploaded()}
		next[h] = p
		if was, ok := c.progress[h]; ok {
			down += max(0, p.done-was.done)
			up += max(0, p.uploaded-was.uploaded)
		}
	}
	if !c.at.IsZero() && at.After(c.at) {
		c.down, c.up = perSecond(down, at.Sub(c.at)), perSecond(up, at.Sub(c.at))
	}
	c.at, c.progress = at, next
}

func (m *meter) rates(node ID, now time.Time) (down, up int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := m.clients[node]
	if c == nil || now.Sub(c.at) > m.quiet {
		return 0, 0
	}
	return c.down, c.up
}

// forget drops what the meter holds of a Node whose client no longer
// reports.
func (m *meter) forget(node ID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, node)
	delete(m.kept, node)
}

func perSecond(bytes int64, d time.Duration) int64 { return int64(float64(bytes) / d.Seconds()) }

// factsIn notes the time Downloading facts arrived from node, so its
// client's rate spans the facts' own interval, and passes every arrival on.
func (s *Service) factsIn(node ID, scope string) {
	if scope == string(role.Downloading) {
		if f, ok := s.cfg.Downloads.Facts(node); ok {
			s.meter.arrive(node, time.Now(), f)
		}
	}
	s.cfg.Facts(node, scope)
}

// sample reads every link's counters, and while an Admin listens, every
// client's rates, and publishes them while bytes move and once more after
// they stop.
func (s *Service) sample(ctx context.Context) {
	now := time.Now()
	t := Traffic{At: now, Links: []LinkTraffic{}, Nodes: []NodeTraffic{}}
	board := s.Board()
	for _, v := range board.nodes {
		if v.Controller {
			continue
		}
		key, ok := s.Key(v.ID)
		if !ok {
			continue
		}
		st, up := s.cfg.Link.Stats(key)
		if !up {
			s.meter.linkDown(v.ID)
			continue
		}
		sent, received := s.meter.link(v.ID, now, st)
		t.Links = append(t.Links, LinkTraffic{A: s.self.String(), B: v.ID.String(), ABBps: sent, BABps: received, RTTMs: st.RTT.Milliseconds()})
	}
	listening := s.cfg.Traffic.Listening()
	for _, v := range board.nodes {
		f, ok := s.cfg.Downloads.Facts(v.ID)
		if !ok {
			s.meter.forget(v.ID)
			continue
		}
		if !listening {
			continue
		}
		down, up := s.meter.rates(v.ID, now)
		t.Nodes = append(t.Nodes, NodeTraffic{ID: v.ID.String(), DownBps: down, UpBps: up, Torrents: s.torrentTraffic(ctx, v.ID, f)})
	}
	moving := listening && t.moving()
	s.meter.mu.Lock()
	publish := moving || listening && s.meter.moving
	s.meter.moving = moving
	s.meter.mu.Unlock()
	if publish {
		s.cfg.Traffic.Publish(ctx, t)
	}
}

// torrentTraffic is each Grab whose torrent node's client reports in f.
// It reads the Grabs again when f names a torrent it hasn't seen.
func (s *Service) torrentTraffic(ctx context.Context, node ID, f *dlv1.Facts) []TorrentTraffic {
	s.meter.mu.Lock()
	kept := s.meter.kept[node]
	s.meter.mu.Unlock()
	for _, t := range f.GetTorrents() {
		if _, ok := kept[bencode.InfoHash(t.GetInfoHash())]; !ok {
			kept = s.readKept(ctx, node)
			break
		}
	}
	out := []TorrentTraffic{}
	for _, t := range f.GetTorrents() {
		for _, grab := range kept[bencode.InfoHash(t.GetInfoHash())] {
			out = append(out, TorrentTraffic{GrabID: grab.String(), Done: t.GetDone(), Peers: int(t.GetPeers())})
		}
	}
	return out
}

func (s *Service) readKept(ctx context.Context, node ID) map[bencode.InfoHash][]storage.UUID {
	grabbed, err := s.cfg.Kept(ctx, s.cfg.Library, node)
	if err != nil {
		if ctx.Err() == nil {
			s.cfg.Log.Error("read the Grabs a Node's client keeps", "node", node.String(), "err", err.Error())
		}
		return nil
	}
	kept := map[bencode.InfoHash][]storage.UUID{}
	for _, g := range grabbed {
		kept[g.InfoHash] = append(kept[g.InfoHash], g.Grab)
	}
	s.meter.mu.Lock()
	s.meter.kept[node] = kept
	s.meter.mu.Unlock()
	return kept
}

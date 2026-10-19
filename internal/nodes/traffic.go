package nodes

import (
	"context"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
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

// Grabbed is a Grab whose torrent a Node's client keeps, with the
// torrent's progress as the Node last reported it.
type Grabbed struct {
	Grab         storage.UUID
	InfoHash     bencode.InfoHash
	Title        string
	Done, Wanted int64
}

// meter turns the link's byte counters, read at each sample, into rates.
type meter struct {
	mu    sync.Mutex
	links map[ID]sampled
	// moving: the last sample published showed bytes moving, so the next
	// is published even when nothing moves, and the canvas comes to rest.
	moving bool
}

type sampled struct {
	at             time.Time
	sent, received int64
}

func newMeter() *meter { return &meter{links: map[ID]sampled{}} }

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

func perSecond(bytes int64, d time.Duration) int64 { return int64(float64(bytes) / d.Seconds()) }

// sample reads every link's counters and, while an Admin listens,
// publishes them while bytes move and once more after they stop.
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
	moving := listening && t.moving()
	s.meter.mu.Lock()
	publish := moving || listening && s.meter.moving
	s.meter.moving = moving
	s.meter.mu.Unlock()
	if publish {
		s.cfg.Traffic.Publish(ctx, t)
	}
}

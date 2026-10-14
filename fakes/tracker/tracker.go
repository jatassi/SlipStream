// Package tracker is fake-tracker: a private HTTPS BitTorrent tracker that
// knows the world's one passkey and the release catalog's torrents, answers
// compact peer lists, and records every announce for tests and verify,
// which read them from GET /records.
package tracker

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
)

// Passkey is the one passkey the tracker accepts in a world with Scenario s.
func Passkey(s scenario.Scenario) string { return releases.Passkey(s) }

// DefaultInterval is the announce interval a Scenario that sets none gets.
const DefaultInterval = 30 * time.Second

const (
	FailurePasskey = "passkey not recognized" //nolint:gosec // a failure reason, not a credential
	FailureTorrent = "torrent not registered with this tracker"
)

var Fake = fake.Spec{
	Name:      "tracker",
	TLS:       true,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
}

// Announce is one announce as the tracker received it. Source is the IP
// address it came from, and Failure the failure reason it answered, empty
// when it answered peers.
type Announce struct {
	Passkey    string `json:"passkey"`
	InfoHash   string `json:"infoHash"`
	PeerID     string `json:"peerId"`
	Source     string `json:"source"`
	Port       uint16 `json:"port"`
	Event      string `json:"event"`
	Uploaded   int64  `json:"uploaded"`
	Downloaded int64  `json:"downloaded"`
	Left       int64  `json:"left"`
	Failure    string `json:"failure"`
}

type peer struct {
	id   string
	addr netip.AddrPort
}

type server struct {
	passkey  string
	torrents map[bencode.InfoHash]bool
	interval time.Duration

	mu      sync.Mutex
	swarms  map[bencode.InfoHash][]peer
	records []Announce
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	srv := &server{
		torrents: map[bencode.InfoHash]bool{},
		interval: s.Tracker.Interval,
		swarms:   map[bencode.InfoHash][]peer{},
		records:  []Announce{},
	}
	if srv.interval == 0 {
		srv.interval = DefaultInterval
	}
	if env.Releases != nil {
		srv.passkey = env.Releases.Passkey
		for _, r := range env.Releases.Releases {
			srv.torrents[r.InfoHash] = true
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /announce/{passkey}", srv.announce)
	mux.HandleFunc("GET /records", srv.list)
	return fake.HTTP(mux), nil
}

func (s *server) announce(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	source, _, _ := net.SplitHostPort(r.RemoteAddr)
	a := Announce{
		Passkey: r.PathValue("passkey"),
		PeerID:  hex.EncodeToString([]byte(q.Get("peer_id"))),
		Source:  source,
		Event:   q.Get("event"),
	}
	if port, err := strconv.ParseUint(q.Get("port"), 10, 16); err == nil {
		a.Port = uint16(port)
	}
	a.Uploaded, _ = strconv.ParseInt(q.Get("uploaded"), 10, 64)
	a.Downloaded, _ = strconv.ParseInt(q.Get("downloaded"), 10, 64)
	a.Left, _ = strconv.ParseInt(q.Get("left"), 10, 64)
	var hash bencode.InfoHash
	if raw := q.Get("info_hash"); len(raw) == len(hash) {
		copy(hash[:], raw)
		a.InfoHash = hash.String()
	}

	s.mu.Lock()
	switch {
	case a.Passkey != s.passkey:
		a.Failure = FailurePasskey
	case !s.torrents[hash]:
		a.Failure = FailureTorrent
	}
	var reply map[string]any
	if a.Failure != "" {
		reply = map[string]any{"failure reason": a.Failure}
	} else {
		reply = map[string]any{
			"interval":     int64(s.interval / time.Second),
			"min interval": int64(s.interval / time.Second),
			"peers":        s.join(hash, a),
		}
	}
	s.records = append(s.records, a)
	s.mu.Unlock()

	body, err := bencode.Encode(reply)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(body)
}

// join updates the announcer's place in the swarm and returns every other
// peer, compact, in peer id order. Called with s.mu held.
func (s *server) join(hash bencode.InfoHash, a Announce) []byte {
	swarm := slices.DeleteFunc(s.swarms[hash], func(p peer) bool { return p.id == a.PeerID })
	var compact []byte
	for _, p := range swarm {
		if ip := p.addr.Addr(); ip.Is4() {
			compact = binary.BigEndian.AppendUint16(append(compact, ip.AsSlice()...), p.addr.Port())
		}
	}
	if ip, err := netip.ParseAddr(a.Source); err == nil && a.Event != "stopped" {
		swarm = append(swarm, peer{a.PeerID, netip.AddrPortFrom(ip.Unmap(), a.Port)})
		slices.SortFunc(swarm, func(x, y peer) int { return strings.Compare(x.id, y.id) })
	}
	s.swarms[hash] = swarm
	return compact
}

func (s *server) list(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	body, err := json.Marshal(s.records)
	s.mu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

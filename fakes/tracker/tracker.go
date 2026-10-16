// Package tracker is fake-tracker: a private HTTPS BitTorrent tracker that
// knows the world's one passkey and the release catalog's torrents. It
// answers a leecher with the seeder front of its own origin, and records
// every announce as its announces capture.
package tracker

import (
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
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

// Announces is the capture of every announce, in tracker.announce records
// whose node is the announcing origin.
var Announces = fake.CaptureSpec{Name: "announces", Kind: "tracker.announce"}

var Fake = fake.Spec{
	Name:      "tracker",
	TLS:       fake.FrontTLS,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("private trackers get fixtures only, #26 Q12"),
	Captures:  []fake.CaptureSpec{Announces},
}

// SeedersTarget is the fault target of the seeder fronts of release.
func SeedersTarget(release string) fault.Target { return fault.Target("seeders/" + release) }

// Announce is one announce as the tracker received it, and Failure the
// failure reason it answered, empty when it answered peers.
type Announce struct {
	Passkey    string `json:"passkey"`
	InfoHash   string `json:"infoHash"`
	PeerID     string `json:"peerId"`
	Port       uint16 `json:"port"`
	Event      string `json:"event"`
	Uploaded   int64  `json:"uploaded"`
	Downloaded int64  `json:"downloaded"`
	Left       int64  `json:"left"`
	Failure    string `json:"failure"`
}

type server struct {
	passkey   string
	torrents  map[bencode.InfoHash]string
	interval  time.Duration
	fronts    fake.Fronts
	announces *fake.Capture
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	announces, err := env.Capture(Announces.Name)
	if err != nil {
		return nil, err
	}
	srv := &server{
		torrents:  map[bencode.InfoHash]string{},
		interval:  s.Tracker.Interval,
		fronts:    env.Fronts,
		announces: announces,
	}
	if srv.interval == 0 {
		srv.interval = DefaultInterval
	}
	if env.Releases != nil {
		srv.passkey = env.Releases.Passkey
		for _, r := range env.Releases.Releases {
			srv.torrents[r.InfoHash] = r.Name
		}
	}
	return fake.HTTP(env, fake.Routes{
		"announce": {Pattern: "GET /announce/{passkey}", Handler: srv.announce},
	}), nil
}

func (s *server) announce(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	a := Announce{
		Passkey: r.PathValue("passkey"),
		PeerID:  hex.EncodeToString([]byte(q.Get("peer_id"))),
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
	release, known := s.torrents[hash]
	switch {
	case a.Passkey != s.passkey:
		a.Failure = FailurePasskey
	case !known:
		a.Failure = FailureTorrent
	}
	var reply map[string]any
	if a.Failure != "" {
		reply = map[string]any{"failure reason": a.Failure}
	} else {
		reply = map[string]any{
			"interval":     int64(s.interval / time.Second),
			"min interval": int64(s.interval / time.Second),
			"peers":        s.peers(fake.OriginOf(r.Context()), release, a),
		}
	}
	s.announces.Record(r.Context(), a)

	body, err := bencode.Encode(reply)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(body)
}

// peers is the compact peer list for a: a leecher's own origin's seeder
// front for the release, so every swarm byte crosses a front, and nothing
// for a seeder, so no seeder dials a Node.
func (s *server) peers(origin, release string, a Announce) []byte {
	if a.Left == 0 || a.Event == "stopped" {
		return []byte{}
	}
	at, ok := s.fronts.Find(SeedersTarget(release), origin)
	if !ok {
		return []byte{}
	}
	ip := at.AddrPort.Addr().As4()
	return binary.BigEndian.AppendUint16(ip[:], at.AddrPort.Port())
}

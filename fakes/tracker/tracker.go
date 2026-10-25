// Package tracker is fake-tracker: a private HTTPS BitTorrent tracker that
// knows the world's one passkey and the release catalog's torrents. It
// answers a leecher with the seeder and bad peer fronts of its own origin,
// and records every announce as its announces capture.
package tracker

import (
	"cmp"
	"encoding/binary"
	"encoding/hex"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/torrent/unit3d"
)

// Passkey is the one passkey the tracker accepts in a world with Scenario s.
func Passkey(s scenario.Scenario) string { return releases.Passkey(s) }

// DefaultInterval is the announce interval a Scenario that sets none gets.
const DefaultInterval = 30 * time.Second

// DefaultAllowPeerIDs are the peer ID prefixes gazelle mode accepts when
// the Scenario lists none: libtorrent's.
var DefaultAllowPeerIDs = []string{"-LT"}

const (
	FailurePasskey      = "passkey not recognized" //nolint:gosec // a failure reason, not a credential
	FailureTorrent      = "torrent not registered with this tracker"
	FailureUnregistered = "Unregistered torrent"
	FailurePeerID       = "Your client is not on the whitelist"
)

// The rules a rejected announce's record names, beside unit3d's.
const (
	RulePasskey      = "passkey"
	RuleTorrent      = "unknown-torrent"
	RuleUnregistered = "unregistered"
	RulePeerID       = "gazelle-peer-id"
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

// BadPeerTarget is the fault target of the bad peer fronts of release.
func BadPeerTarget(release string) fault.Target { return fault.Target("badpeer/" + release) }

// Announce is one announce as the tracker received it. Accepted is whether
// it answered peers; when not, Rule is the rule the announce broke and
// Failure the failure reason it answered.
type Announce struct {
	Passkey    string `json:"passkey"`
	InfoHash   string `json:"infoHash"`
	PeerID     string `json:"peerId"`
	Port       uint16 `json:"port"`
	Event      string `json:"event"`
	Uploaded   int64  `json:"uploaded"`
	Downloaded int64  `json:"downloaded"`
	Left       int64  `json:"left"`
	Accepted   bool   `json:"accepted"`
	Rule       string `json:"rule"`
	Failure    string `json:"failure"`
}

type torrent struct {
	release      string
	unregistered bool
}

type server struct {
	passkey             string
	torrents            map[bencode.InfoHash]torrent
	interval            time.Duration
	minInterval         time.Duration
	failWithoutInterval bool
	// allowPeerIDs is nil outside gazelle mode, and windows outside unit3d.
	allowPeerIDs []string
	windows      *unit3d.Windows
	fronts       fake.Fronts
	announces    *fake.Capture
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	announces, err := env.Capture(Announces.Name)
	if err != nil {
		return nil, err
	}
	srv := &server{
		torrents:            map[bencode.InfoHash]torrent{},
		interval:            cmp.Or(s.Tracker.Interval, DefaultInterval),
		failWithoutInterval: s.Tracker.FailWithoutInterval,
		fronts:              env.Fronts,
		announces:           announces,
	}
	srv.minInterval = cmp.Or(s.Tracker.MinInterval, srv.interval)
	switch s.Tracker.Mode {
	case "", scenario.Plain:
	case scenario.Gazelle:
		srv.allowPeerIDs = s.Tracker.AllowPeerIDs
		if len(srv.allowPeerIDs) == 0 {
			srv.allowPeerIDs = DefaultAllowPeerIDs
		}
	case scenario.UNIT3D:
		srv.windows = unit3d.NewWindows(srv.minInterval, env.Streams.Stream("unit3d-windows"))
	}
	if env.Releases != nil {
		srv.passkey = env.Releases.Passkey
		for _, r := range env.Releases.Releases {
			srv.torrents[r.InfoHash] = torrent{release: r.Name, unregistered: r.Unregistered}
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
	a.Rule, a.Failure = s.judge(a, hash, q.Get("peer_id"))
	a.Accepted = a.Rule == ""
	reply := map[string]any{
		"interval":     int64(s.interval / time.Second),
		"min interval": int64(s.minInterval / time.Second),
	}
	switch {
	case a.Accepted:
		reply["peers"] = s.peers(fake.OriginOf(r.Context()), s.torrents[hash].release, a)
	case s.failWithoutInterval:
		reply = map[string]any{"failure reason": a.Failure}
	default:
		reply["failure reason"] = a.Failure
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

// judge is the rule a breaks and the failure reason the tracker answers,
// both empty when it accepts a.
func (s *server) judge(a Announce, hash bencode.InfoHash, peerID string) (rule, failure string) {
	t, known := s.torrents[hash]
	switch {
	case a.Passkey != s.passkey:
		return RulePasskey, FailurePasskey
	case !known:
		return RuleTorrent, FailureTorrent
	case t.unregistered:
		return RuleUnregistered, FailureUnregistered
	case s.allowPeerIDs != nil && !slices.ContainsFunc(s.allowPeerIDs, func(p string) bool { return strings.HasPrefix(peerID, p) }):
		return RulePeerID, FailurePeerID
	case s.windows != nil:
		var id [20]byte
		copy(id[:], peerID)
		broke, reason := s.windows.Admit(a.Passkey, hash, id, a.Event, time.Now())
		return string(broke), reason
	}
	return "", ""
}

// peers is the compact peer list for a: a leecher's own origin's seeder
// front and bad peer front for the release, so every swarm byte crosses a
// front, and nothing for a seeder, so no seeder dials a Node.
func (s *server) peers(origin, release string, a Announce) []byte {
	peers := []byte{}
	if a.Left == 0 || a.Event == "stopped" {
		return peers
	}
	for _, target := range []fault.Target{SeedersTarget(release), BadPeerTarget(release)} {
		if at, ok := s.fronts.Find(target, origin); ok {
			ip := at.AddrPort.Addr().As4()
			peers = binary.BigEndian.AppendUint16(append(peers, ip[:]...), at.AddrPort.Port())
		}
	}
	return peers
}

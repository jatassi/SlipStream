// Package tracker is fake-tracker: a private HTTPS BitTorrent tracker
// with one account per fixture-site site and one for fixture-seeders,
// each a passkey, that knows the release catalog's torrents. It answers a
// leecher with the seeder and bad peer fronts of its own origin, records
// every announce as its announces capture, and keeps each site account's
// history of each torrent as UNIT3D does, recorded as its seeding capture.
package tracker

import (
	"cmp"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/torrent/unit3d"
)

// Passkey is the passkey of account, a fixture-site site's label or
// releases.Seeders, in a world with Scenario s.
func Passkey(s scenario.Scenario, account string) string { return releases.Passkey(s, account) }

// DefaultInterval is the announce interval a Scenario that sets none gets.
const DefaultInterval = 30 * time.Second

// UNIT3D's own prewarn and grace, and its ghost peer flush, for a
// Scenario that sets none.
const (
	DefaultPrewarn       = 24 * time.Hour
	DefaultGrace         = 72 * time.Hour
	DefaultInactiveAfter = 2 * time.Hour
)

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

// Seeding is the capture of each site account's History of each torrent,
// a tracker.seeding record whenever it changes, whose node is the origin
// of the account's last announce of the torrent. fixture-seeders' account
// keeps none.
var Seeding = fake.CaptureSpec{Name: "seeding", Kind: "tracker.seeding"}

// Unregister answers a release's announces with "Unregistered torrent"
// from now on, as for a torrent the tracker deleted while it seeds:
// {"release": "<name>"}.
var Unregister = fake.CueSpec{Name: "tracker-unregister"}

// sharedSwarm is the key of fake-tracker's swarm in its run's Shared.
const sharedSwarm = "tracker"

var Fake = fake.Spec{
	Name:      "tracker",
	TLS:       fake.FrontTLS,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("private trackers get fixtures only, #26 Q12"),
	Captures:  []fake.CaptureSpec{Announces, Seeding},
	Cues:      []fake.CueSpec{Unregister},
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
	release string
	size    int64
	// listed is how many seeders the feeds list, which the swarm counts
	// add to the peers the tracker saw.
	listed       int64
	unregistered bool
}

type server struct {
	// accounts are the account of each passkey: a site's label, or
	// releases.Seeders.
	accounts            map[string]string
	tracker             scenario.Tracker
	interval            time.Duration
	minInterval         time.Duration
	failWithoutInterval bool
	// allowPeerIDs is nil outside gazelle mode, and windows outside unit3d.
	allowPeerIDs []string
	windows      *unit3d.Windows
	fronts       fake.Fronts
	announces    *fake.Capture
	seeding      *fake.Capture
	hitrun       hitrunTimings

	mu        sync.Mutex
	torrents  map[bencode.InfoHash]*torrent
	peers     map[peerKey]*peer
	completed map[bencode.InfoHash]int64
	histories map[historyKey]*history
	// closed stops the histories' timers once the server stops serving.
	closed bool
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	announces, err := env.Capture(Announces.Name)
	if err != nil {
		return nil, err
	}
	seeding, err := env.Capture(Seeding.Name)
	if err != nil {
		return nil, err
	}
	srv := &server{
		accounts:            map[string]string{releases.Passkey(s, releases.Seeders): releases.Seeders},
		tracker:             s.Tracker,
		interval:            cmp.Or(s.Tracker.Interval, DefaultInterval),
		failWithoutInterval: s.Tracker.FailWithoutInterval,
		fronts:              env.Fronts,
		announces:           announces,
		seeding:             seeding,
		hitrun: hitrunTimings{
			prewarn:       cmp.Or(s.Tracker.Prewarn, DefaultPrewarn),
			grace:         cmp.Or(s.Tracker.Grace, DefaultGrace),
			inactiveAfter: cmp.Or(s.Tracker.InactiveAfter, DefaultInactiveAfter),
		},
		torrents:  map[bencode.InfoHash]*torrent{},
		peers:     map[peerKey]*peer{},
		completed: map[bencode.InfoHash]int64{},
		histories: map[historyKey]*history{},
	}
	for _, labels := range scenario.FixtureSites {
		for _, label := range labels {
			srv.accounts[releases.Passkey(s, label)] = label
		}
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
		for _, r := range env.Releases.Releases {
			srv.torrents[r.InfoHash] = &torrent{release: r.Name, size: r.Size, listed: int64(r.Seeders), unregistered: r.Unregistered}
		}
	}
	if err := fake.OnCue(env, Unregister.Name, srv.unregister); err != nil {
		return nil, err
	}
	env.Shared.Put(sharedSwarm, srv)
	return serving{Server: fake.HTTP(env, fake.Routes{
		"announce": {Pattern: "GET /announce/{passkey}", Handler: srv.announce},
	}), srv: srv}, nil
}

// serving stops the histories' timers once its Server stops.
type serving struct {
	fake.Server
	srv *server
}

func (s serving) Serve(ctx context.Context, l net.Listener) error {
	defer s.srv.close()
	return s.Server.Serve(ctx, l)
}

// UnregisterArgs names the release the Unregister cue unregisters.
type UnregisterArgs struct {
	Release string `json:"release"`
}

type wasUnregistered struct {
	Was bool `json:"was"`
}

func (s *server) unregister(_ context.Context, a UnregisterArgs) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.torrents {
		if t.release == a.Release {
			was := t.unregistered
			t.unregistered = true
			return wasUnregistered{was}, nil
		}
	}
	return nil, fmt.Errorf("the tracker has no release %q", a.Release)
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
	origin := fake.OriginOf(r.Context())
	now := time.Now()
	reply := map[string]any{
		"interval":     int64(s.interval / time.Second),
		"min interval": int64(s.minInterval / time.Second),
	}
	s.mu.Lock()
	a.Rule, a.Failure = s.judge(a, hash, q.Get("peer_id"), now)
	a.Accepted = a.Rule == ""
	s.announces.Record(r.Context(), a)
	switch {
	case a.Accepted:
		s.accept(a, hash, origin, now)
		reply["complete"], reply["incomplete"] = s.swarm(hash, now)
		reply["downloaded"] = s.completed[hash]
		reply["peers"] = s.peerFronts(origin, s.torrents[hash].release, a)
	case s.failWithoutInterval:
		reply = map[string]any{"failure reason": a.Failure}
	default:
		reply["failure reason"] = a.Failure
	}
	s.mu.Unlock()

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
func (s *server) judge(a Announce, hash bencode.InfoHash, peerID string, now time.Time) (rule, failure string) {
	if _, account := s.accounts[a.Passkey]; !account {
		return RulePasskey, FailurePasskey
	}
	if rule, failure := s.judgeTorrent(hash); rule != "" {
		return rule, failure
	}
	switch {
	case s.allowPeerIDs != nil && !slices.ContainsFunc(s.allowPeerIDs, func(p string) bool { return strings.HasPrefix(peerID, p) }):
		return RulePeerID, FailurePeerID
	case s.windows != nil:
		var id [20]byte
		copy(id[:], peerID)
		broke, reason := s.windows.Admit(a.Passkey, hash, id, a.Event, now)
		return string(broke), reason
	}
	return "", ""
}

// judgeTorrent is the rule an announce of hash breaks whatever account
// sends it, and the failure reason the tracker answers.
func (s *server) judgeTorrent(hash bencode.InfoHash) (rule, failure string) {
	t, known := s.torrents[hash]
	switch {
	case !known:
		return RuleTorrent, FailureTorrent
	case t.unregistered:
		return RuleUnregistered, FailureUnregistered
	}
	return "", ""
}

// peerFronts is the compact peer list for a: a leecher's own origin's
// seeder front and bad peer front for the release, so every swarm byte
// crosses a front, and nothing for a seeder, so no seeder dials a Node.
func (s *server) peerFronts(origin, release string, a Announce) []byte {
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

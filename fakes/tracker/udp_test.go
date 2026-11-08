package tracker_test

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
)

// udpTracker is fake-tracker serving HTTPS announces at http and BEP 15
// ones at the seedbox's socket udp, recording into runDir.
type udpTracker struct {
	http   string
	udp    netip.AddrPort
	runDir string
}

func serveUDP(t *testing.T, s scenario.Scenario) udpTracker {
	t.Helper()
	w := udpTracker{runDir: t.TempDir()}
	log, err := events.Create(w.runDir, func(events.Record) []events.Record { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = log.Close() })
	plan := fault.NewPlan(nil, nil)
	env := fake.NewEnv(s, tracker.Fake, slog.New(slog.DiscardHandler), plan, log)
	env.Releases, env.Fronts = catalog(), fronts
	server, err := tracker.Fake.New(s, env)
	if err != nil {
		t.Fatal(err)
	}
	udpEnv := fake.NewEnv(s, tracker.UDP, slog.New(slog.DiscardHandler), plan, log)
	udpEnv.Releases, udpEnv.Fronts, udpEnv.Shared = env.Releases, env.Fronts, env.Shared
	udp, err := tracker.UDP.NewUDP(s, udpEnv)
	if err != nil {
		t.Fatal(err)
	}
	l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	socket, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:0")))
	if err != nil {
		t.Fatal(err)
	}
	w.http, w.udp = "http://"+l.Addr().String(), netip.MustParseAddrPort(socket.LocalAddr().String())
	ctx, cancel := context.WithCancel(context.WithoutCancel(t.Context()))
	done := make(chan error, 2)
	go func() { done <- server.Serve(ctx, l) }()
	go func() { done <- udp.ServeUDP(ctx, map[string]*net.UDPConn{"seedbox": socket}) }()
	t.Cleanup(func() {
		cancel()
		for range 2 {
			if err := <-done; err != nil {
				t.Error(err)
			}
		}
	})
	return w
}

// client is a BEP 15 client on its own loopback socket.
type client struct {
	t    *testing.T
	conn *net.UDPConn
	at   netip.AddrPort
	tx   uint32
}

func dial(t *testing.T, at netip.AddrPort) *client {
	t.Helper()
	conn, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:0")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return &client{t: t, conn: conn, at: at}
}

// ask sends a request on connection conn of action with body, and returns the reply's action and what follows its
// transaction ID.
func (c *client) ask(conn uint64, action uint32, body []byte) (replied uint32, reply []byte) {
	c.t.Helper()
	c.tx++
	req := binary.BigEndian.AppendUint32(binary.BigEndian.AppendUint32(binary.BigEndian.AppendUint64(nil, conn), action), c.tx)
	if _, err := c.conn.WriteToUDPAddrPort(append(req, body...), c.at); err != nil {
		c.t.Fatal(err)
	}
	_ = c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 2048)
	n, _, err := c.conn.ReadFromUDPAddrPort(buf)
	if err != nil {
		c.t.Fatalf("action %d got no reply: %v", action, err)
	}
	if n < 8 || binary.BigEndian.Uint32(buf[4:]) != c.tx {
		c.t.Fatalf("action %d got %x, want a reply to transaction %d", action, buf[:n], c.tx)
	}
	return binary.BigEndian.Uint32(buf), buf[8:n]
}

func (c *client) connect() uint64 {
	c.t.Helper()
	action, body := c.ask(0x41727101980, 0, nil)
	if action != 0 || len(body) != 8 {
		c.t.Fatalf("connect got action %d %x, want a connection ID", action, body)
	}
	return binary.BigEndian.Uint64(body)
}

// udpEvent is BEP 15's number for event.
var udpEvent = map[string]uint32{"": 0, "completed": 1, "started": 2, "stopped": 3}

// announce sends peer's announce of hash with left bytes to go, and
// returns the reply's action and body.
func (c *client) announce(conn uint64, hash bencode.InfoHash, peer, event string, port uint16, left int64) (action uint32, reply []byte) {
	c.t.Helper()
	body := slices.Concat(hash[:], []byte(peer))
	for _, v := range []uint64{0, uint64(left), 0} { //nolint:gosec // a test's byte counts
		body = binary.BigEndian.AppendUint64(body, v)
	}
	for _, v := range []uint32{udpEvent[event], 0, 7, 0xffffffff} {
		body = binary.BigEndian.AppendUint32(body, v)
	}
	body = binary.BigEndian.AppendUint16(body, port)
	body = append(body, 2, 9)
	return c.ask(conn, 1, append(body, "/announce"...))
}

// swarmReply is an accepted announce's reply: the interval, leechers and
// seeders, and the peers.
type swarmReply struct {
	interval, leechers, seeders uint32
	peers                       string
}

// swarm is the reply to an announce that is accepted.
func (c *client) swarm(conn uint64, hash bencode.InfoHash, peer, event string, port uint16, left int64) swarmReply {
	c.t.Helper()
	action, body := c.announce(conn, hash, peer, event, port, left)
	if action != 1 || len(body) < 12 {
		c.t.Fatalf("announce got action %d %q, want an announce reply", action, body)
	}
	return swarmReply{binary.BigEndian.Uint32(body), binary.BigEndian.Uint32(body[4:]), binary.BigEndian.Uint32(body[8:]), string(body[12:])}
}

func (w udpTracker) announceHTTPS(t *testing.T, hash bencode.InfoHash, peer, event string, left int64) {
	t.Helper()
	q := url.Values{"info_hash": {string(hash[:])}, "peer_id": {peer}, "port": {"6881"}, "left": {strconv.FormatInt(left, 10)}, "event": {event}}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, w.http+"/announce/"+passkey+"?"+q.Encode(), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, res.Body)
}

// TestTheUDPAnnounceAnswersFromTheTrackersSwarm connects and announces
// over BEP 15: a leecher is answered with the interval, the swarm
// fake-tracker's HTTPS announces also count, and its own origin's seeder
// and bad peer fronts, and a seeder with no peers.
func TestTheUDPAnnounceAnswersFromTheTrackersSwarm(t *testing.T) {
	w := serveUDP(t, scenario.Scenario{Seed: "tracker", Tracker: scenario.Tracker{Interval: 40 * time.Second}})
	w.announceHTTPS(t, orchard, seeder, "started", 0)
	c := dial(t, w.udp)
	conn := c.connect()

	fronts := "\x7f\x00\x00\x01\x17\x72\x7f\x00\x00\x01\x17\x73"
	if got, want := c.swarm(conn, orchard, leecher, "started", 6882, size), (swarmReply{40, 1, 1, fronts}); got != want {
		t.Errorf("the leecher's started = %+v, want %+v", got, want)
	}
	if got, want := c.swarm(conn, orchard, leecher, "completed", 6882, 0), (swarmReply{40, 0, 2, ""}); got != want {
		t.Errorf("the leecher's completed = %+v, want %+v", got, want)
	}

	var heard []string
	as, err := captured[tracker.Announce](w.runDir, tracker.UDPAnnounces.Kind)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range as {
		heard = append(heard, a.InfoHash+" "+a.Event+" "+strconv.FormatBool(a.Accepted))
	}
	if want := []string{orchard.String() + " started true", orchard.String() + " completed true"}; !slices.Equal(heard, want) {
		t.Errorf("the UDP announces capture holds %q, want %q", heard, want)
	}
}

// TestTheUDPAnnounceRefuses answers an error to an announce of a torrent
// it doesn't know or unregistered.
func TestTheUDPAnnounceRefuses(t *testing.T) {
	w := serveUDP(t, scenario.Scenario{Seed: "tracker"})
	c := dial(t, w.udp)
	conn := c.connect()
	for _, tc := range []struct {
		what string
		hash bencode.InfoHash
		want string
	}{
		{"a torrent it doesn't know", decoy, tracker.FailureTorrent},
		{"an unregistered torrent", ghost, tracker.FailureUnregistered},
	} {
		if action, body := c.announce(conn, tc.hash, leecher, "started", 6882, size); action != 3 || string(body) != tc.want {
			t.Errorf("an announce on %s = action %d %q, want error %q", tc.what, action, body, tc.want)
		}
	}
}

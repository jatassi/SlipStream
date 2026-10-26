package dht_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/netip"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/dht"
	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/internal/bencode"
)

var (
	orchard = bencode.InfoHash{1}
	harbor  = bencode.InfoHash{2}
	meadow  = bencode.InfoHash{3}
)

// catalog holds orchard, public and seeded, harbor, private and seeded,
// and meadow, public with no seeder.
var catalog = &releases.Catalog{Releases: []releases.Release{
	{Release: scenario.Release{Name: "orchard", Public: true, Seeded: true}, InfoHash: orchard},
	{Release: scenario.Release{Name: "harbor", Seeded: true}, InfoHash: harbor},
	{Release: scenario.Release{Name: "meadow", Public: true}, InfoHash: meadow},
}}

var fronts = fake.Fronts{
	{Target: tracker.SeedersTarget("orchard"), Node: "home", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:6001")}},
	{Target: tracker.SeedersTarget("orchard"), Node: "seedbox", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:6002")}},
	{Target: tracker.SeedersTarget("harbor"), Node: "home", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:6003")}},
}

type sink struct {
	mu   sync.Mutex
	recs []events.Record
}

func (s *sink) Append(r events.Record) (events.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recs = append(s.recs, r)
	return r, nil
}

func (s *sink) records() []events.Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]events.Record{}, s.recs...)
}

// serve runs fake-dht on a socket for home and one for seedbox until the
// test ends.
func serve(t *testing.T, records *sink) map[string]netip.AddrPort {
	t.Helper()
	s := scenario.Scenario{Seed: "dht"}
	env := fake.NewEnv(s, dht.Fake, slog.New(slog.DiscardHandler), fault.NewPlan(nil, nil), records)
	env.Releases, env.Fronts = catalog, fronts
	srv, err := dht.Fake.NewUDP(s, env)
	if err != nil {
		t.Fatal(err)
	}
	sockets, at := map[string]*net.UDPConn{}, map[string]netip.AddrPort{}
	for _, origin := range []string{"home", "seedbox"} {
		conn, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:0")))
		if err != nil {
			t.Fatal(err)
		}
		sockets[origin], at[origin] = conn, netip.MustParseAddrPort(conn.LocalAddr().String())
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- srv.ServeUDP(ctx, sockets) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	return at
}

func getPeers(t *testing.T, conn *net.UDPConn, at netip.AddrPort, h bencode.InfoHash) any {
	t.Helper()
	q, err := bencode.Encode(map[string]any{"t": "aa", "y": "q", "q": "get_peers", "a": map[string]any{"id": "abcdefghij0123456789", "info_hash": string(h[:])}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.WriteToUDPAddrPort(q, at); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 2048)
	n, _, err := conn.ReadFromUDPAddrPort(buf)
	if err != nil {
		t.Fatalf("get_peers got no response: %v", err)
	}
	v, err := bencode.Decode(buf[:n])
	m, ok := v.(map[string]any)
	if err != nil || !ok {
		t.Fatalf("get_peers got %q", buf[:n])
	}
	r, _ := m["r"].(map[string]any)
	return r["values"]
}

func TestGetPeersAnswersTheOriginsSeederFrontOfAPublicSeededRelease(t *testing.T) {
	records := &sink{}
	at := serve(t, records)
	conn, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:0")))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	for _, tc := range []struct {
		origin string
		hash   bencode.InfoHash
		want   any
	}{
		{"home", orchard, []any{"\x7f\x00\x00\x01\x17\x71"}},
		{"seedbox", orchard, []any{"\x7f\x00\x00\x01\x17\x72"}},
		{"home", harbor, nil},
		{"home", meadow, nil},
	} {
		if got := getPeers(t, conn, at[tc.origin], tc.hash); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("get_peers %s through %s's socket = %q, want %q", tc.hash, tc.origin, got, tc.want)
		}
	}
	from := conn.LocalAddr().String()
	var got []string
	for _, r := range records.records() {
		var q dht.Query
		if err := json.Unmarshal(r.Data, &q); err != nil || r.Kind != dht.Lookups.Kind || q.Method != "get_peers" || q.From != from {
			t.Fatalf("recorded %s %s, %v; want a dht.query of a get_peers from %s", r.Kind, r.Data, err, from)
		}
		got = append(got, r.Node+" "+q.InfoHash)
	}
	want := []string{"home " + orchard.String(), "seedbox " + orchard.String(), "home " + harbor.String(), "home " + meadow.String()}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("recorded %q, want %q", got, want)
	}
}

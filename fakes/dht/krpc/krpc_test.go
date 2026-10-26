package krpc_test

import (
	"net"
	"net/netip"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/dht/krpc"
	"github.com/jatassi/slipstream/internal/bencode"
)

var (
	seeded = bencode.InfoHash{1}
	lonely = bencode.InfoHash{2}
	front  = map[string]netip.AddrPort{
		"home": netip.MustParseAddrPort("127.0.0.2:41000"),
		"away": netip.MustParseAddrPort("[::1]:42000"),
	}
)

type world struct {
	srv     *krpc.Server
	mu      sync.Mutex
	queries []krpc.Query
}

func start(t *testing.T) *world {
	t.Helper()
	w := &world{}
	w.srv = krpc.New(krpc.Config{
		Peers: func(origin string, h bencode.InfoHash) []netip.AddrPort {
			if h != seeded {
				return nil
			}
			return []netip.AddrPort{front[origin]}
		},
		Record: func(q krpc.Query) {
			w.mu.Lock()
			defer w.mu.Unlock()
			w.queries = append(w.queries, q)
		},
	})
	t.Cleanup(func() {
		if err := w.srv.Close(); err != nil {
			t.Error(err)
		}
	})
	return w
}

func (w *world) recorded() []krpc.Query {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.queries)
}

func ask(t *testing.T, conn *net.UDPConn, addr netip.AddrPort, method string, args map[string]any) map[string]any {
	t.Helper()
	args["id"] = "abcdefghij0123456789"
	out, err := bencode.Encode(map[string]any{"t": "aa", "y": "q", "q": method, "a": args})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.WriteToUDPAddrPort(out, addr); err != nil {
		t.Fatal(err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 2048)
	n, _, err := conn.ReadFromUDPAddrPort(buf)
	if err != nil {
		t.Fatalf("%s got no response: %v", method, err)
	}
	v, err := bencode.Decode(buf[:n])
	m, ok := v.(map[string]any)
	if err != nil || !ok || m["t"] != "aa" {
		t.Fatalf("%s got %q, want a KRPC message with the query's transaction ID", method, buf[:n])
	}
	return m
}

func client(t *testing.T) *net.UDPConn {
	t.Helper()
	conn, err := net.ListenUDP("udp", net.UDPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:0")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func listen(t *testing.T, w *world, origin string) netip.AddrPort {
	t.Helper()
	addr, err := w.srv.Listen(origin, netip.MustParseAddrPort("127.0.0.1:0"))
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

func TestEachOriginsSocketAnswersGetPeersWithItsOwnPeers(t *testing.T) {
	w := start(t)
	home, away := listen(t, w, "home"), listen(t, w, "away")
	c := client(t)
	for _, at := range []struct {
		addr netip.AddrPort
		want string
	}{
		{home, "\x7f\x00\x00\x02\xa0\x28"},
		{away, "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\xa4\x10"},
	} {
		r, _ := ask(t, c, at.addr, "get_peers", map[string]any{"info_hash": string(seeded[:])})["r"].(map[string]any)
		if got := r["values"]; !reflect.DeepEqual(got, []any{at.want}) || r["token"] == "" {
			t.Errorf("get_peers at %s answered %q, want values [%q] and a token", at.addr, r, at.want)
		}
	}
	r, _ := ask(t, c, home, "get_peers", map[string]any{"info_hash": string(lonely[:])})["r"].(map[string]any)
	if _, ok := r["values"]; ok || r["nodes"] != "" {
		t.Errorf("get_peers for a hash nobody seeds answered %q, want no values and no nodes", r)
	}
	from := netip.MustParseAddrPort(c.LocalAddr().String())
	want := []krpc.Query{
		{Origin: "home", Method: "get_peers", From: from, InfoHash: seeded},
		{Origin: "away", Method: "get_peers", From: from, InfoHash: seeded},
		{Origin: "home", Method: "get_peers", From: from, InfoHash: lonely},
	}
	if got := w.recorded(); !reflect.DeepEqual(got, want) {
		t.Errorf("recorded %+v, want %+v", got, want)
	}
}

func TestPingFindNodeAndAnnouncePeerAreAnsweredAndRecorded(t *testing.T) {
	w := start(t)
	home := listen(t, w, "home")
	c := client(t)
	ping := ask(t, c, home, "ping", map[string]any{})
	id, _ := ping["r"].(map[string]any)["id"].(string)
	if ping["y"] != "r" || len(id) != 20 {
		t.Fatalf("ping answered %q, want a response with a 20-byte node ID", ping)
	}
	if r, _ := ask(t, c, home, "find_node", map[string]any{"target": id})["r"].(map[string]any); r["nodes"] != "" || r["id"] != id {
		t.Errorf("find_node answered %q, want the same node ID and no nodes", r)
	}
	ask(t, c, home, "announce_peer", map[string]any{"info_hash": string(seeded[:]), "port": 6881, "token": "x"})
	ask(t, c, home, "announce_peer", map[string]any{"info_hash": string(lonely[:]), "port": 1, "implied_port": 1, "token": "x"})
	if e := ask(t, c, home, "sample_infohashes", map[string]any{"target": id}); e["y"] != "e" || !reflect.DeepEqual(e["e"], []any{int64(204), "Method Unknown"}) {
		t.Errorf("an unknown method answered %q, want error 204", e)
	}
	from := netip.MustParseAddrPort(c.LocalAddr().String())
	want := []krpc.Query{
		{Origin: "home", Method: "ping", From: from},
		{Origin: "home", Method: "find_node", From: from},
		{Origin: "home", Method: "announce_peer", From: from, InfoHash: seeded, Port: 6881},
		{Origin: "home", Method: "announce_peer", From: from, InfoHash: lonely, Port: from.Port()},
		{Origin: "home", Method: "sample_infohashes", From: from},
	}
	if got := w.recorded(); !reflect.DeepEqual(got, want) {
		t.Errorf("recorded %+v, want %+v", got, want)
	}
}

package outside_test

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strings"
	"testing"

	"golang.org/x/net/proxy"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/gateway"
	"github.com/jatassi/slipstream/fakes/outside"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

var nodes = []string{"atlas", "den", "seedbox", "pi", "nas", "box"}

func world() scenario.Scenario {
	site := map[string]string{"atlas": "home", "den": "home", "seedbox": "ams", "pi": "cell", "nas": "open6", "box": "open6"}
	s := scenario.Scenario{Name: "outside", Network: &scenario.Network{
		Sites: []scenario.Site{
			{Name: "home", WAN: "public", Address: "203.0.113.9", Mapping: []string{"nat-pmp"}, Prefix6: "2001:db8:1::/64", Forwards: []scenario.Forward{{Node: "atlas", Service: "http"}}},
			{Name: "ams", WAN: "none", Address: "192.0.2.44", Prefix6: "2001:db8:2::/64"},
			{Name: "cell", WAN: "cgnat", Address: "203.0.113.50", Gateway: "100.64.12.7", Forwards: []scenario.Forward{{Node: "pi", Service: "http"}}},
			{Name: "open6", WAN: "public", Address: "203.0.113.70", Prefix6: "2001:db8:7::/64", Firewall6: "open"},
		},
		Proxies: []scenario.Proxy{{Name: "the-vps", Address: "198.51.100.8", Routes: []scenario.ProxyRoute{{Host: "plex.proxy.fake.localhost", Node: "atlas", Service: "plex"}}}},
	}}
	for _, n := range nodes {
		s.Nodes = append(s.Nodes, scenario.Node{Name: n, Setup: scenario.SetupJoin, Site: site[n]})
	}
	s.Nodes[0].Setup = scenario.SetupController
	s.Nodes[4].Firewall = []scenario.FirewallRule{{Service: "http", Family: "ipv6", Policy: "blocks"}}
	s.Nodes[5].Bind = "local"
	return s
}

// echo is a server that answers what it is and the X-Forwarded-For it saw.
func echo(t *testing.T, what string) netip.AddrPort {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, what+" xff="+r.Header.Get("X-Forwarded-For")) //nolint:gosec // G705: a test server answering a test client
	}))
	t.Cleanup(srv.Close)
	return netip.MustParseAddrPort(srv.Listener.Addr().String())
}

type served struct {
	dialer, gateway faketest.Served
	http, plex      map[string]netip.AddrPort
}

func serve(t *testing.T) served {
	t.Helper()
	s := world()
	srv := served{http: map[string]netip.AddrPort{}, plex: map[string]netip.AddrPort{}}
	var fronts fake.Fronts
	for _, n := range nodes {
		srv.http[n], srv.plex[n] = echo(t, "http@"+n), echo(t, "plex@"+n)
		fronts = append(fronts, fake.Front{Target: "plex", Node: n, Address: fake.Address{AddrPort: srv.plex[n]}})
	}
	for _, n := range append(slices.Clone(nodes), events.World) {
		fronts = append(fronts, fake.Front{Target: outside.ProxyTarget, Node: n, Address: fake.Address{AddrPort: echo(t, "proxy@"+n)}})
	}
	listeners := faketest.WithListeners(func(name string) (node.Listeners, error) { return node.Listeners{HTTP: srv.http[name]}, nil })
	shared, origins := faketest.WithShared(fake.NewShared()), faketest.WithOrigins(append(slices.Clone(nodes), events.World)...)
	srv.gateway = faketest.UDP(t, gateway.Fake, s, origins, shared, listeners, faketest.WithFronts(fronts))
	srv.dialer = faketest.InMemory(t, outside.Fake, s, origins, shared, listeners, faketest.WithFronts(fronts))
	return srv
}

// via dials the outside dialer's front for origin.
type via struct {
	ctx    context.Context
	srv    faketest.Served
	origin string
}

func (v via) Dial(_, _ string) (net.Conn, error) { return v.srv.Dial(v.ctx, v.origin) }

// get asks the outside dialer, as origin, to connect to addr, and answers
// the body of a GET there or the SOCKS5 refusal.
func (srv served) get(t *testing.T, origin, addr string) string {
	t.Helper()
	d, err := proxy.SOCKS5("tcp", "outside", nil, via{ctx: t.Context(), srv: srv.dialer, origin: origin})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		for _, refusal := range []string{"host unreachable", "connection refused", "address type not supported"} {
			if strings.Contains(err.Error(), refusal) {
				return refusal
			}
		}
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: plex.proxy.fake.localhost\r\nX-Forwarded-For: 10.9.9.9\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	res, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestEachConnectionTakesItsSitesPath(t *testing.T) {
	srv := serve(t)
	at := func(addr, svc, n string) string {
		port := srv.http[n].Port()
		if svc == "plex" {
			port = srv.plex[n].Port()
		}
		return netip.AddrPortFrom(netip.MustParseAddr(addr), port).String()
	}
	for _, c := range []struct {
		name, origin, to, want string
	}{
		{"a forward lets HTTP in, with the source in X-Forwarded-For", "seedbox", at("203.0.113.9", "http", "atlas"), "http@atlas xff=192.0.2.44"},
		{"a port with no forward refuses", "seedbox", at("203.0.113.9", "plex", "atlas"), "connection refused"},
		{"hairpin NAT turns a same-site connection back in", "den", at("203.0.113.9", "http", "atlas"), "http@atlas xff=203.0.113.9"},
		{"a CGNAT site has no inbound path", "seedbox", at("203.0.113.50", "http", "pi"), "host unreachable"},
		{"a closed IPv6 firewall refuses", "seedbox", at("2001:db8:1::20", "http", "atlas"), "connection refused"},
		{"an open IPv6 firewall lets a Media Server in", "seedbox", at("2001:db8:7::24", "plex", "nas"), "plex@nas xff=10.9.9.9"},
		{"a Node's firewall refuses its family", "seedbox", at("2001:db8:7::24", "http", "nas"), "connection refused"},
		{"a local bind refuses", "seedbox", at("2001:db8:7::25", "http", "box"), "connection refused"},
		{"an address no site holds is unreachable", "seedbox", at("203.0.113.200", "http", "atlas"), "host unreachable"},
		{"a proxy's address goes to fake-proxy's front for the origin", "seedbox", "198.51.100.8:443", "proxy@seedbox xff=10.9.9.9"},
		{"a name is refused, since a tester resolves it", "seedbox", "plex.home.localhost:80", "address type not supported"},
	} {
		if got := srv.get(t, c.origin, c.to); got != c.want {
			t.Errorf("%s: %s connecting to %s = %q, want %q", c.name, c.origin, c.to, got, c.want)
		}
	}
	var hairpin []string
	for _, r := range srv.dialer.Records(t) {
		var rec outside.Connect
		if err := json.Unmarshal(r.Data, &rec); err != nil {
			t.Fatal(err)
		}
		if rec.Hairpin {
			hairpin = append(hairpin, r.Node+"->"+rec.Node)
		}
	}
	if !slices.Equal(hairpin, []string{"den->atlas"}) {
		t.Errorf("hairpin records = %v, want [den->atlas]", hairpin)
	}
}

func TestAGrantedMappingLetsATesterIn(t *testing.T) {
	srv := serve(t)
	to := netip.AddrPortFrom(netip.MustParseAddr("203.0.113.9"), 40000).String()
	if got := srv.get(t, "seedbox", to); got != "connection refused" {
		t.Fatalf("before the mapping, seedbox connecting to %s = %q, want connection refused", to, got)
	}
	conn, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(srv.gateway.Sockets["atlas"]))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	req := binary.BigEndian.AppendUint32(binary.BigEndian.AppendUint16(binary.BigEndian.AppendUint16([]byte{0, 2, 0, 0}, srv.plex["atlas"].Port()), 40000), 3600)
	if _, err := conn.Write(req); err != nil {
		t.Fatal(err)
	}
	res := make([]byte, 16)
	if _, err := conn.Read(res); err != nil || binary.BigEndian.Uint16(res[2:]) != 0 {
		t.Fatalf("NAT-PMP map answered %x, %v", res, err)
	}
	if got := srv.get(t, "seedbox", to); got != "plex@atlas xff=10.9.9.9" {
		t.Errorf("after the mapping, seedbox connecting to %s = %q, want plex@atlas", to, got)
	}
}

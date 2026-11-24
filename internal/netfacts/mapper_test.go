package netfacts_test

import (
	"context"
	"log/slog"
	"net/netip"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	fakegateway "github.com/jatassi/slipstream/fakes/gateway"
	"github.com/jatassi/slipstream/fakes/scenario"
	fakestun "github.com/jatassi/slipstream/fakes/stun"
	"github.com/jatassi/slipstream/internal/netfacts"
	"github.com/jatassi/slipstream/internal/netfacts/gateway"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// routers serves a router per origin, each on its own site speaking every
// protocol, with fake-stun so a pass learns its way out.
type routers struct {
	udp, igd, stun faketest.Served
}

func newRouters(t *testing.T, origins []string) routers {
	t.Helper()
	s := scenario.Scenario{Name: "mapping", Network: &scenario.Network{}}
	var fronts fake.Fronts
	for i, o := range origins {
		s.Network.Sites = append(s.Network.Sites, scenario.Site{
			Name: o, WAN: "public", Address: netip.AddrFrom4([4]byte{203, 0, 113, byte(10 + i)}).String(), Mapping: []string{"pcp", "nat-pmp", "upnp"},
		})
		s.Nodes = append(s.Nodes, scenario.Node{Name: o, Setup: scenario.SetupJoin, Site: o})
	}
	s.Nodes[0].Setup = scenario.SetupController
	for i, n := range append(slices.Clone(origins), events.World) {
		fronts = append(fronts, fake.Front{Target: "gateway-igd", Node: n, Address: fake.Address{AddrPort: netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(43000+i))}})
	}
	shared := fake.NewShared()
	opts := faketest.WithOrigins(append(slices.Clone(origins), events.World)...)
	return routers{
		udp:  faketest.UDP(t, fakegateway.Fake, s, opts, faketest.WithShared(shared), faketest.WithFronts(fronts)),
		igd:  faketest.InMemory(t, fakegateway.IGD, s, opts, faketest.WithShared(shared), faketest.WithFronts(fronts)),
		stun: faketest.UDP(t, fakestun.Fake, s, opts, faketest.WithShared(shared)),
	}
}

func (r routers) client(t *testing.T, origin string) *gateway.Client {
	return &gateway.Client{Packets: loopback{}, HTTP: r.igd.From(t, origin), Timings: gateway.Timings{Exchange: 200 * time.Millisecond, SSDPWait: 200 * time.Millisecond}}
}

func (r routers) netfacts(t *testing.T, origin string, local *storage.DB[storage.Local], node role.NodeID) *netfacts.Netfacts {
	return netfacts.New(netfacts.Config{
		Local: local, Node: node, Env: env(), Timings: timings(), HTTP: r.igd.From(t, origin), Log: slog.New(slog.DiscardHandler),
		Endpoints: netfacts.Endpoints{
			STUN:    endpoint.Optional{endpoint.MustParseHostPort(r.stun.Sockets[origin].String())},
			Gateway: endpoint.Optional{endpoint.MustParseHostPort(r.udp.Sockets[origin].String())},
		},
	})
}

// forwards is the router's forwards to port, as it lists them.
func (r routers) forwards(t *testing.T, origin string, port uint16) int {
	t.Helper()
	c := r.client(t, origin)
	gw, err := c.Toward(t.Context(), r.udp.Sockets[origin], true)
	if err != nil {
		t.Fatal(err)
	}
	g, err := c.Read(t.Context(), gw, nil)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, f := range g.Forwards {
		if f.External == port {
			n++
		}
	}
	return n
}

// seed is what a crash left behind before a Node starts: its row's state
// (empty for none), the protocol, and whether the router granted it.
type seed struct {
	row    string
	method netmodel.MapProtocol
	router bool
}

const mappedPort = 32400

// plant leaves s on origin's router and in local, as node.
func (r routers) plant(t *testing.T, origin string, local *storage.DB[storage.Local], node role.NodeID, s seed) {
	t.Helper()
	ask := gateway.Ask{Method: s.method, External: mappedPort, Internal: mappedPort, Lifetime: time.Hour, Nonce: [12]byte{7}, Description: "SlipStream " + node.String()}
	if s.router {
		gw, err := r.client(t, origin).Toward(t.Context(), r.udp.Sockets[origin], true)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := r.client(t, origin).Map(t.Context(), gw, ask); err != nil {
			t.Fatal(err)
		}
	}
	if s.row == "" {
		return
	}
	method := map[netmodel.MapProtocol]string{netmodel.MapPCP: "pcp", netmodel.MapNATPMP: "nat-pmp", netmodel.MapUPnP: "upnp"}[s.method]
	var nonce []byte
	if s.method == netmodel.MapPCP {
		nonce = ask.Nonce[:]
	}
	var granted, expires any
	if s.row == "granted" {
		granted, expires = time.Now().UnixMicro(), time.Now().Add(time.Hour).UnixMicro()
	}
	err := local.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO netfacts_mapping (port, protocol, method, gateway, external_port, nonce, asked_at, granted_at, expires_at)
			VALUES (?, 'tcp', ?, ?, ?, ?, ?, ?, ?)`, mappedPort, method, r.udp.Sockets[origin].String(), mappedPort, nonce, time.Now().UnixMicro(), granted, expires)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// settle watches n until it published once, mapping the port when want.
func (r routers) settle(t *testing.T, n *netfacts.Netfacts, origin string, want bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	h := newHolder()
	done := make(chan error, 1)
	go func() { done <- n.Watch(ctx, h) }()
	intent := &netv1.Intent{Watch: true, IntervalMs: time.Hour.Milliseconds(), Stun: &netv1.StunList{Servers: []string{r.stun.Sockets[origin].String()}}}
	if want {
		intent.Map = []*netv1.MapWant{{External: mappedPort, Internal: mappedPort}}
	}
	h.intents <- intent
	awaitFacts(t, h)
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// The mapper converges from every point a crash can leave it at: a row
// asking with or without the router's grant, a granted row no longer
// wanted, and a UPnP entry of this Node's with no row. Each takes one
// watch, and Release after a restart deletes every mapping.
func TestTheMapperConvergesFromEachCrashPoint(t *testing.T) {
	cases := []struct {
		name    string
		seed    seed
		want    bool
		restart bool
	}{
		{name: "asking, granted, wanted", seed: seed{row: "asking", method: netmodel.MapPCP, router: true}, want: true},
		{name: "asking, never granted, wanted", seed: seed{row: "asking", method: netmodel.MapNATPMP}, want: true},
		{name: "nothing yet, wanted", want: true},
		{name: "asking, granted, no longer wanted", seed: seed{row: "asking", method: netmodel.MapPCP, router: true}},
		{name: "granted, no longer wanted", seed: seed{row: "granted", method: netmodel.MapUPnP, router: true}},
		{name: "a UPnP entry with no row", seed: seed{method: netmodel.MapUPnP, router: true}},
		{name: "granted, released after a restart", seed: seed{row: "granted", method: netmodel.MapNATPMP, router: true}, restart: true},
	}
	origins := make([]string, len(cases))
	for i := range cases {
		origins[i] = string(rune('a'+i)) + "-node"
	}
	r := newRouters(t, origins)
	for i, c := range cases {
		origin, node, local := origins[i], role.NodeID{UUID: storage.NewUUID()}, storagetest.Local(t, netfacts.Schema())
		r.plant(t, origin, local, node, c.seed)
		n := r.netfacts(t, origin, local, node)
		if c.restart {
			if err := n.Release(t.Context()); err != nil {
				t.Fatalf("%s: release: %v", c.name, err)
			}
		} else {
			r.settle(t, n, origin, c.want)
		}
		f, err := n.Facts(t.Context(), nil, 0, true)
		if err != nil {
			t.Fatal(err)
		}
		held := slices.ContainsFunc(f.Mappings, func(m netmodel.Mapping) bool { return m.External == mappedPort && m.Refused == netmodel.RefusedNone })
		if got := r.forwards(t, origin, mappedPort); got != map[bool]int{true: 1, false: 0}[c.want] || held != c.want || !c.want && len(f.Mappings) != 0 {
			t.Errorf("%s: the router lists %d forwards to the port and the Node holds %+v; want mapped %t", c.name, got, f.Mappings, c.want)
		}
	}
}

// The mapping table holds only the shapes the mapper writes.
func TestTheMappingTableRefusesEachMisshapenRow(t *testing.T) {
	local := storagetest.Local(t, netfacts.Schema())
	for _, c := range []struct {
		name string
		cols string
		ok   bool
	}{
		{"an asking PCP row", `32400, 'tcp', 'pcp', 'r', 32400, x'000000000000000000000000', 1, NULL, NULL, NULL`, true},
		{"a permanent UPnP grant", `8484, 'tcp', 'upnp', 'r', 443, NULL, 1, 2, NULL, NULL`, true},
		{"a refused NAT-PMP row", `9000, 'tcp', 'nat-pmp', 'r', 9000, NULL, 1, NULL, NULL, 'not-authorized'`, true},
		{"port 0", `0, 'tcp', 'upnp', 'r', 1, NULL, 1, NULL, NULL, NULL`, false},
		{"UDP", `1, 'udp', 'upnp', 'r', 1, NULL, 1, NULL, NULL, NULL`, false},
		{"an unknown method", `2, 'tcp', 'igd', 'r', 1, NULL, 1, NULL, NULL, NULL`, false},
		{"external port 70000", `3, 'tcp', 'upnp', 'r', 70000, NULL, 1, NULL, NULL, NULL`, false},
		{"a short nonce", `4, 'tcp', 'pcp', 'r', 4, x'00', 1, NULL, NULL, NULL`, false},
		{"PCP with no nonce", `5, 'tcp', 'pcp', 'r', 5, NULL, 1, NULL, NULL, NULL`, false},
		{"an expiry with no grant", `6, 'tcp', 'upnp', 'r', 6, NULL, 1, NULL, 3, NULL`, false},
		{"a grant and a refusal", `7, 'tcp', 'upnp', 'r', 7, NULL, 1, 2, 3, 'refused'`, false},
		{"an unknown refusal", `8, 'tcp', 'upnp', 'r', 8, NULL, 1, NULL, NULL, 'busy'`, false},
	} {
		err := local.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
			_, err := tx.ExecContext(ctx, `INSERT INTO netfacts_mapping (port, protocol, method, gateway, external_port, nonce, asked_at, granted_at, expires_at, refused) VALUES (`+c.cols+`)`)
			return err
		})
		if (err == nil) != c.ok {
			t.Errorf("%s: inserted %v, want ok %t", c.name, err, c.ok)
		}
	}
}

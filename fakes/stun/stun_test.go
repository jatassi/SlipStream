package stun_test

import (
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/pion/stun/v4"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	fakestun "github.com/jatassi/slipstream/fakes/stun"
)

func sites() scenario.Scenario {
	return scenario.Scenario{
		Name: "stun",
		Nodes: []scenario.Node{
			{Name: "atlas", Setup: scenario.SetupController, Site: "home"},
			{Name: "seedbox", Setup: scenario.SetupJoin, Site: "ams"},
			{Name: "den", Setup: scenario.SetupJoin, Site: "cell"},
		},
		Network: &scenario.Network{Sites: []scenario.Site{
			{Name: "home", WAN: "public", Address: "203.0.113.9"},
			{Name: "ams", WAN: "none", Address: "192.0.2.44", STUN: "varying"},
			{Name: "cell", WAN: "cgnat", Address: "203.0.113.50", Gateway: "100.64.12.7", STUN: "timeout"},
		}},
	}
}

func TestEachServerAnswersTheAddressTheSiteShows(t *testing.T) {
	s := sites()
	shared := fake.NewShared()
	origins := faketest.WithOrigins("atlas", "seedbox", "den", events.World)
	first := faketest.UDP(t, fakestun.Fake, s, origins, faketest.WithShared(shared))
	second := faketest.UDP(t, fakestun.Second, s, origins, faketest.WithShared(shared))
	for _, c := range []struct {
		origin string
		second bool
		want   string
	}{
		{"atlas", false, "203.0.113.9:50000"},
		{"atlas", true, "203.0.113.9:50000"},
		{"seedbox", false, "192.0.2.44:50001"},
		{"seedbox", true, "192.0.2.44:51001"},
		{"den", false, ""},
		{events.World, false, "198.51.100.254:50000"},
	} {
		at := first.Sockets[c.origin]
		if c.second {
			at = second.Sockets[c.origin]
		}
		if got := bind(t, at); got != c.want {
			t.Errorf("%s asking server two=%t got %q, want %q", c.origin, c.second, got, c.want)
		}
	}
	if got := len(first.Records(t)); got != 6 {
		t.Errorf("stun-requests holds %d records, want one per request, 6", got)
	}
}

// bind sends one Binding request to at and answers the mapped address,
// empty when no answer comes.
func bind(t *testing.T, at netip.AddrPort) string {
	t.Helper()
	conn, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(at))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	req := stun.MustBuild(stun.TransactionID, stun.BindingRequest, stun.Fingerprint)
	if _, err := conn.Write(req.Raw); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		return ""
	}
	res := &stun.Message{Raw: buf[:n]}
	var mapped stun.XORMappedAddress
	if err := res.Decode(); err != nil || res.TransactionID != req.TransactionID {
		t.Fatalf("answer %x is no response to the request", buf[:n])
	}
	if err := mapped.GetFrom(res); err != nil {
		t.Fatal(err)
	}
	return mapped.String()
}

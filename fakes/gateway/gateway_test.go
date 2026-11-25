package gateway_test

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/gateway"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/netfacts/script"
)

var nodes = []string{"atlas", "den", "nas", "pi", "box", "seedbox"}

func world() scenario.Scenario {
	site := map[string]string{"atlas": "home", "den": "cell", "nas": "double", "pi": "permanent", "box": "other", "seedbox": "vps"}
	s := scenario.Scenario{Name: "gateway", Network: &scenario.Network{Sites: []scenario.Site{
		{Name: "home", WAN: "public", Address: "203.0.113.9", Mapping: []string{"pcp", "nat-pmp", "upnp"}, Prefix6: "2001:db8:1::/64", Forwards: []scenario.Forward{{Node: "atlas", Service: "plex", Listed: true}}},
		{Name: "cell", WAN: "cgnat", Address: "203.0.113.50", Gateway: "100.64.12.7", Mapping: []string{"pcp", "nat-pmp"}, Refuse: "refuse"},
		{Name: "double", WAN: "private", Address: "203.0.113.60", Gateway: "192.168.0.5", Mapping: []string{"upnp"}, Refuse: "606"},
		{Name: "permanent", WAN: "public", Address: "203.0.113.70", Mapping: []string{"nat-pmp", "upnp"}, Refuse: "725"},
		{Name: "other", WAN: "empty", Address: "203.0.113.80", Mapping: []string{"pcp", "nat-pmp", "upnp"}, Refuse: "other-port"},
		{Name: "vps", WAN: "none", Address: "192.0.2.44"},
	}}}
	for _, n := range nodes {
		s.Nodes = append(s.Nodes, scenario.Node{Name: n, Setup: scenario.SetupJoin, Site: site[n]})
	}
	s.Nodes[0].Setup = scenario.SetupController
	return s
}

// served is fake-gateway on a socket per Node, and its IGD reached as
// each Node reaches it, whose fronts are the ones SSDP names.
type served struct {
	udp, igd faketest.Served
}

func serve(t *testing.T) served {
	t.Helper()
	s := world()
	var fronts fake.Fronts
	for i, n := range append(slices.Clone(nodes), events.World) {
		fronts = append(fronts,
			fake.Front{Target: "plex", Node: n, Address: fake.Address{AddrPort: netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(32400+i))}},
			fake.Front{Target: "gateway-igd", Node: n, Address: fake.Address{AddrPort: netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(41000+i))}})
	}
	shared := fake.NewShared()
	origins := faketest.WithOrigins(append(slices.Clone(nodes), events.World)...)
	udp := faketest.UDP(t, gateway.Fake, s, origins, faketest.WithShared(shared), faketest.WithFronts(fronts))
	igd := faketest.InMemory(t, gateway.IGD, s, origins, faketest.WithShared(shared), faketest.WithFronts(fronts))
	return served{udp: udp, igd: igd}
}

func TestEachRouterPlaysItsSite(t *testing.T) {
	srv := serve(t)
	address := []byte{0, 0}
	mapTCP := func(internal, external uint16, lifetime uint32) []byte {
		p := binary.BigEndian.AppendUint16(binary.BigEndian.AppendUint16([]byte{0, 2, 0, 0}, internal), external)
		return binary.BigEndian.AppendUint32(p, lifetime)
	}
	announce := pcpRequest(0, 0, nil)
	for _, c := range []struct {
		name, origin string
		send         []byte
		want         string
		write        bool
	}{
		{"NAT-PMP reads a public WAN", "atlas", address, "nat-pmp result=0 wan=203.0.113.9", false},
		{"NAT-PMP reads a WAN in 100.64/10", "den", address, "nat-pmp result=0 wan=100.64.12.7", false},
		{"NAT-PMP reads an empty WAN", "box", address, "nat-pmp result=3 wan=0.0.0.0", false},
		{"NAT-PMP is silent on a router without it", "nas", address, "silent", false},
		{"no router answers nothing", "seedbox", address, "silent", false},
		{"the world's socket answers as a public router", events.World, address, "nat-pmp result=0 wan=198.51.100.254", false},
		{"NAT-PMP grants a mapping", "atlas", mapTCP(8484, 8484, 3600), "nat-pmp result=0 internal=8484 external=8484", true},
		{"NAT-PMP refuses on a refusing router", "den", mapTCP(8484, 8484, 3600), "nat-pmp result=2 internal=8484 external=0", true},
		{"NAT-PMP grants another port", "box", mapTCP(8484, 8484, 3600), "nat-pmp result=0 internal=8484 external=8485", true},
		{"PCP announces", "atlas", announce, "pcp result=0", false},
		{"PCP maps on the WAN", "atlas", pcpRequest(1, 3600, pcpMap(32400)), "pcp result=0 external=203.0.113.9:32400", true},
		{"PCP refuses on a refusing router", "den", pcpRequest(1, 3600, pcpMap(32400)), "pcp result=2 external=[::]:32400", true},
		{"PCP to a NAT-PMP router answers unsupported version", "pi", announce, "nat-pmp result=1", false},
		{"PCP to a router with neither is silent", "nas", announce, "silent", false},
		{"PCP grants another port", "box", pcpRequest(1, 3600, pcpMap(32400)), "pcp result=0 external=0.0.0.0:32401", true},
		{"SSDP names the origin's IGD", "atlas", search, "ssdp location=http://127.0.0.1:41000/rootDesc.xml", false},
		{"SSDP is silent on a router without UPnP", "den", search, "silent", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			before := len(srv.udp.Records(t))
			if got := exchange(t, srv.udp.Sockets[c.origin], c.send); got != c.want {
				t.Fatalf("got %s, want %s", got, c.want)
			}
			recs := srv.udp.Records(t)[before:]
			if c.want == "silent" || strings.HasPrefix(c.want, "nat-pmp result=1") {
				return
			}
			var req gateway.Request
			if len(recs) != 1 || json.Unmarshal(recs[0].Data, &req) != nil || recs[0].Node != c.origin || req.Write != c.write {
				t.Fatalf("records %v, want one from %s with write %t", recs, c.origin, c.write)
			}
		})
	}
}

var search = []byte("M-SEARCH * HTTP/1.1\r\nHOST: 239.255.255.250:1900\r\nMAN: \"ssdp:discover\"\r\nMX: 1\r\nST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\n\r\n")

func pcpRequest(op byte, lifetime uint32, data []byte) []byte {
	p := binary.BigEndian.AppendUint32([]byte{2, op, 0, 0}, lifetime)
	return append(append(p, netip.MustParseAddr("::ffff:192.168.1.20").AsSlice()...), data...)
}

func pcpMap(port uint16) []byte {
	p := append(make([]byte, 12), 6, 0, 0, 0)
	p = binary.BigEndian.AppendUint16(binary.BigEndian.AppendUint16(p, port), port)
	return append(p, make([]byte, 16)...)
}

// exchange sends p to at and describes the answer, silent when none comes.
func exchange(t *testing.T, at netip.AddrPort, p []byte) string {
	t.Helper()
	conn, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(at))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.Write(p); err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil {
		return "silent"
	}
	r := buf[:n]
	switch {
	case strings.HasPrefix(string(r), "HTTP/1.1 200"):
		return "ssdp location=" + regexp.MustCompile(`LOCATION: (\S+)`).FindStringSubmatch(string(r))[1]
	case r[0] == 0 && n == 12:
		return fmt.Sprintf("nat-pmp result=%d wan=%s", r[3], netip.AddrFrom4([4]byte(r[8:12])))
	case r[0] == 0 && n == 16:
		return fmt.Sprintf("nat-pmp result=%d internal=%d external=%d", r[3], binary.BigEndian.Uint16(r[8:]), binary.BigEndian.Uint16(r[10:]))
	case r[0] == 0:
		return fmt.Sprintf("nat-pmp result=%d", r[3])
	case n >= 60:
		return fmt.Sprintf("pcp result=%d external=%s", r[3], netip.AddrPortFrom(netip.AddrFrom16([16]byte(r[44:60])).Unmap(), binary.BigEndian.Uint16(r[42:])))
	}
	return fmt.Sprintf("pcp result=%d", r[3])
}

func TestTheIGDAnswersEachSiteOverBothServices(t *testing.T) {
	srv := serve(t)
	for _, c := range []struct {
		name, origin, action, args, want string
	}{
		{"the WAN", "atlas", "GetExternalIPAddress", "", "NewExternalIPAddress=203.0.113.9"},
		{"an empty WAN", "box", "GetExternalIPAddress", "", "NewExternalIPAddress="},
		{"the listed forward", "atlas", "GetGenericPortMappingEntry", "<NewPortMappingIndex>0</NewPortMappingIndex>",
			"NewRemoteHost= NewExternalPort=32400 NewProtocol=TCP NewInternalPort=32400 NewInternalClient=192.168.1.10 NewEnabled=1 NewPortMappingDescription=plex NewLeaseDuration=0"},
		{"past the last entry", "atlas", "GetGenericPortMappingEntry", "<NewPortMappingIndex>1</NewPortMappingIndex>", "error 713"},
		{"a port with no entry", "atlas", "GetSpecificPortMappingEntry", "<NewRemoteHost></NewRemoteHost><NewExternalPort>8484</NewExternalPort><NewProtocol>TCP</NewProtocol>", "error 714"},
		{"a refused mapping", "nas", "AddPortMapping", add(3600), "error 606"},
		{"a lease on a permanent-only router", "pi", "AddPortMapping", add(3600), "error 725"},
		{"a permanent lease", "pi", "AddPortMapping", add(0), ""},
		{"a port another mapping holds", "box", "AddPortMapping", add(3600), "error 718"},
		{"another port", "box", "AddAnyPortMapping", add(3600), "NewReservedPort=8485"},
		{"a delete of nothing", "atlas", "DeletePortMapping", "<NewRemoteHost></NewRemoteHost><NewExternalPort>8484</NewExternalPort><NewProtocol>TCP</NewProtocol>", "error 714"},
		{"a granted mapping", "atlas", "AddPortMapping", add(3600), ""},
		{"the granted mapping listed", "atlas", "GetSpecificPortMappingEntry", "<NewRemoteHost></NewRemoteHost><NewExternalPort>8484</NewExternalPort><NewProtocol>TCP</NewProtocol>",
			"NewInternalPort=8484 NewInternalClient=192.168.1.10 NewEnabled=1 NewPortMappingDescription=SlipStream atlas NewLeaseDuration=3600"},
		{"its delete", "atlas", "DeletePortMapping", "<NewRemoteHost></NewRemoteHost><NewExternalPort>8484</NewExternalPort><NewProtocol>TCP</NewProtocol>", ""},
	} {
		for _, service := range []string{"IPConn", "PPPConn"} {
			listing := strings.HasPrefix(c.action, "GetGeneric") || strings.HasPrefix(c.action, "GetSpecific")
			if service == "PPPConn" && !listing {
				continue
			}
			if got := soap(t, srv.igd.From(t, c.origin), service, c.action, c.args); got != c.want {
				t.Errorf("%s: %s %s from %s = %q, want %q", c.name, service, c.action, c.origin, got, c.want)
			}
		}
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://gateway-igd/rootDesc.xml", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := srv.igd.From(t, "den").Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("a router without UPnP serves its description with %d, want 404", res.StatusCode)
	}
}

// add is AddPortMapping's arguments for port 8484 with lease.
func add(lease int) string {
	return fmt.Sprintf("<NewRemoteHost></NewRemoteHost><NewExternalPort>8484</NewExternalPort><NewProtocol>TCP</NewProtocol><NewInternalPort>8484</NewInternalPort>"+
		"<NewInternalClient>192.168.1.10</NewInternalClient><NewEnabled>1</NewEnabled><NewPortMappingDescription>SlipStream atlas</NewPortMappingDescription><NewLeaseDuration>%d</NewLeaseDuration>", lease)
}

var (
	outArg = regexp.MustCompile(`<(New\w+)>([^<]*)</New\w+>`)
	code   = regexp.MustCompile(`<errorCode>(\d+)</errorCode>`)
)

// soap posts action to the IGD's service and describes the answer: its out
// arguments in order, or the UPnP error code.
func soap(t *testing.T, client *http.Client, service, action, args string) string {
	t.Helper()
	urn := map[string]string{"IPConn": "urn:schemas-upnp-org:service:WANIPConnection:1", "PPPConn": "urn:schemas-upnp-org:service:WANPPPConnection:1"}[service]
	body := `<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><u:` + action + ` xmlns:u="` + urn + `">` + args + `</u:` + action + `></s:Body></s:Envelope>`
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://gateway-igd/ctl/"+service, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("SOAPAction", `"`+urn+"#"+action+`"`)
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	out, _ := io.ReadAll(res.Body)
	if m := code.FindSubmatch(out); m != nil {
		return "error " + string(m[1])
	}
	var parts []string
	for _, m := range outArg.FindAllSubmatch(out, -1) {
		parts = append(parts, string(m[1])+"="+string(m[2]))
	}
	return strings.Join(parts, " ")
}

func TestTheNetworkCueRewritesTheScriptsItChanges(t *testing.T) {
	srv := serve(t)
	read := func(node string) script.File {
		f, err := script.Read(gateway.ScriptPath(srv.udp.Dir, node))
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	if f := read("atlas"); f.Egress != "eth0" || f.Gateway != netip.MustParseAddr("192.168.1.1") || len(f.Links) != len(nodes)-1 {
		t.Fatalf("atlas's script = %+v, want eth0 by 192.168.1.1 with a link per other Node", f)
	}
	if f := read("seedbox"); f.Gateway.IsValid() || f.Interfaces[0].Addresses[0].Prefix != netip.MustParsePrefix("192.0.2.44/24") {
		t.Fatalf("seedbox's script = %+v, want its site's address and no router", f)
	}
	for _, c := range []struct {
		change map[string]string
		nodes  string
		check  func(script.File) bool
	}{
		{map[string]string{"site": "home", "prefix6": "2001:db8:2::/64"}, "[atlas]", func(f script.File) bool {
			return f.Interfaces[0].Addresses[1].Prefix == netip.MustParsePrefix("2001:db8:2::20/64")
		}},
		{map[string]string{"site": "home", "prefix6": "none"}, "[atlas]", func(f script.File) bool { return len(f.Interfaces[0].Addresses) == 1 }},
		{map[string]string{"node": "atlas", "service": "plex", "family": "ipv6", "firewall": "blocks"}, "[atlas]", func(f script.File) bool {
			return slices.Equal(f.Firewall, []script.Rule{{Port: 32400, Family: script.IPv6, Policy: script.Blocks}})
		}},
		{map[string]string{"node": "atlas", "bind": "local"}, "[atlas]", func(f script.File) bool { return f.Bind == script.BindLocal }},
		{map[string]string{"site": "home", "wan": "203.0.113.10"}, "[]", func(script.File) bool { return true }},
	} {
		got := srv.udp.Cue(t, gateway.Network.Name, c.change)
		nodes, _ := got.(map[string][]string)
		if fmt.Sprint(nodes["nodes"]) != c.nodes || !c.check(read("atlas")) {
			t.Errorf("cue %v changed %v and left atlas's script %+v", c.change, got, read("atlas"))
		}
	}
	if got := exchange(t, srv.udp.Sockets["atlas"], []byte{0, 0}); got != "nat-pmp result=0 wan=203.0.113.10" {
		t.Errorf("after a WAN change the router reads %s, want the new WAN", got)
	}
}

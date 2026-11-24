package host

import (
	"net"
	"net/netip"
	"os"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// listen opens an IPv4 socket. A wildcard "tcp" listener would be Go's
// dual-stack IPv6 one.
func listen(t *testing.T, at string) net.Listener {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp4", at)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln
}

func portOf(ln net.Listener) uint16 { return netip.MustParseAddrPort(ln.Addr().String()).Port() }

func TestWindowsSocketsNameFamiliesAndOwner(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	wild, loop, closed := listen(t, "0.0.0.0:0"), listen(t, "127.0.0.1:0"), listen(t, "127.0.0.1:0")
	_ = closed.Close()
	got, err := Sockets(t.Context(), []uint16{portOf(wild), portOf(loop), portOf(closed)})
	if err != nil {
		t.Fatal(err)
	}
	if l := got[0]; l.V4 != netmodel.Yes || l.Loopback || !strings.EqualFold(l.Program.Name, exe) {
		t.Errorf("the wildcard listener reads %+v, want IPv4 owned by %s", l, exe)
	}
	if l := got[1]; !l.Loopback || !strings.EqualFold(l.Program.Name, exe) {
		t.Errorf("the loopback listener reads %+v", l)
	}
	if l := got[2]; l.V4 != netmodel.No || l.V6 != netmodel.No || l.Program.Name != "" {
		t.Errorf("a port nothing listens on reads %+v", l)
	}
}

func TestWindowsLoopbackLeavesByAnInterfaceWithNoGateway(t *testing.T) {
	iface, gw, err := Route(t.Context(), netip.MustParseAddr("127.0.0.1"))
	if err != nil || iface == "" || gw.IsValid() {
		t.Fatalf("Route = %q, %v, %v", iface, gw, err)
	}
	ifaces, err := Interfaces(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range ifaces {
		if i.Name == iface && i.Tunnel {
			t.Errorf("%s reads as a tunnel", iface)
		}
	}
}

func TestWindowsFirewallReadsWithoutElevation(t *testing.T) {
	port := portOf(listen(t, "0.0.0.0:0"))
	asks := []netmodel.Rule{{Port: port, Family: netmodel.IPv4}, {Port: port, Family: netmodel.IPv6}}
	got, err := Firewall(t.Context(), "", asks)
	if err != nil || len(got) != len(asks) {
		t.Fatalf("Firewall = %+v, %v", got, err)
	}
	for _, r := range got {
		if r.Tool != netmodel.WindowsDefender || r.Because == "" {
			t.Errorf("port %d on %v reads %+v, want an answer from INetFwPolicy2", r.Port, r.Family, r)
		}
	}
}

package main

import (
	"fmt"
	"net"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/node"
)

// acceptsIPv4OnUnspecifiedIPv6 is whether a tcp listener on [::], made as
// the Node makes its link listener, takes connections to 127.0.0.1. Go
// clears IPV6_V6ONLY on it, and opens an IPv4 socket on a host with no
// IPv6.
func acceptsIPv4OnUnspecifiedIPv6(t *testing.T) bool {
	t.Helper()
	ln, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "[::]:0")
	if err != nil {
		t.Fatalf("listen on [::]: %v", err)
	}
	defer func() { _ = ln.Close() }()
	d := net.Dialer{Timeout: 5 * time.Second}
	c, err := d.DialContext(t.Context(), "tcp4", netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(ln.Addr().(*net.TCPAddr).Port)).String()) //nolint:gosec // a port number
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// TestALinkOnTheUnspecifiedIPv6AddressServesIPv4 runs home's link on
// [::] and joins box through 127.0.0.1: the join succeeds where the OS
// is dual-stack and finds no Controller where it is not.
func TestALinkOnTheUnspecifiedIPv6AddressServesIPv4(t *testing.T) {
	t.Parallel()
	dual := acceptsIPv4OnUnspecifiedIPv6(t)
	t.Logf("this OS accepts IPv4 on [::]: %v", dual)
	h := startHomeWith(t, []string{
		"SLIPSTREAM_TORRENT_LISTEN=127.0.0.2:0",
		"SLIPSTREAM_HTTP_LISTEN=127.0.0.2:0",
		"SLIPSTREAM_LINK_LISTEN=[::]:0",
		"SLIPSTREAM_NAME=home",
	})
	bound := h.ready.Listeners.Link
	t.Logf("home's link listener reports %s", bound)
	if !bound.Addr().IsUnspecified() {
		t.Fatalf("home's link is on %s, want an unspecified address", bound)
	}
	minted, err := joincode.Parse(h.mint(t))
	if err != nil {
		t.Fatal(err)
	}
	key, err := link.OpenKey(h.dir)
	if err != nil {
		t.Fatal(err)
	}
	public := key.Public()
	viaIPv4 := netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), bound.Port())
	code, err := joincode.New(public[:], minted.Secret(), []netip.AddrPort{viaIPv4})
	if err != nil {
		t.Fatal(err)
	}

	res := slipstream(t, childEnv(boxEnv...), "join", "--data-dir", dataDir(t), code.String())
	if !dual {
		if res.code != node.ExitFailed || !strings.HasPrefix(res.stderr, "join failed: the Controller is unreachable: ") {
			t.Fatalf("join through %s: exit %d, stderr %q; want exit 1, the Controller unreachable", viaIPv4, res.code, res.stderr)
		}
		return
	}
	_, id := h.nodes(t)
	if want := fmt.Sprintf("joined home as box (node %s); link up to %s\n", id, viaIPv4); res.code != node.ExitOK || res.stdout != want {
		t.Fatalf("join through %s: exit %d, stdout %q, stderr %q; want exit 0, stdout %q", viaIPv4, res.code, res.stdout, res.stderr, want)
	}
}

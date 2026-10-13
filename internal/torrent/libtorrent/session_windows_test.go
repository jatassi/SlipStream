package libtorrent_test

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"syscall"
	"testing"

	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

const soExclusiveAddrUse = ^4

// Windows hands out TCP ports in sequence, and refuses a UDP bind to
// 127.0.0.1 on a port another socket holds on 0.0.0.0 with
// SO_EXCLUSIVEADDRUSE (WSAEACCES).
func TestEngineListensWhenTheNextTCPPortsAreTakenForUDP(t *testing.T) {
	var lc net.ListenConfig
	l, err := lc.Listen(t.Context(), "tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cursor := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	held := 0
	for port := cursor + 1; port <= min(cursor+2048, 65535); port++ {
		if holdUDP(t, port) {
			held++
		}
	}
	if held < 1024 {
		t.Fatalf("held UDP on %d ports after TCP port %d, want at least 1024", held, cursor)
	}

	for range 8 {
		e, err := libtorrent.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.1:0")})
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		_, err = e.ListenAddr()
		if cerr := e.Close(); cerr != nil && !errors.Is(cerr, engine.ErrClosed) {
			t.Errorf("close: %v", cerr)
		}
		if err != nil {
			t.Fatalf("listen addr: %v", err)
		}
	}
}

func holdUDP(t *testing.T, port int) bool {
	t.Helper()
	lc := net.ListenConfig{Control: func(_, _ string, c syscall.RawConn) error {
		var serr error
		if err := c.Control(func(fd uintptr) {
			serr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, soExclusiveAddrUse, 1)
		}); err != nil {
			return fmt.Errorf("control: %w", err)
		}
		if serr != nil {
			return fmt.Errorf("set SO_EXCLUSIVEADDRUSE: %w", serr)
		}
		return nil
	}}
	c, err := lc.ListenPacket(t.Context(), "udp4", (&net.UDPAddr{IP: net.IPv4zero, Port: port}).String())
	if err != nil {
		return false
	}
	t.Cleanup(func() { _ = c.Close() })
	return true
}

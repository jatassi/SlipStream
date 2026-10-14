package fault_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/netip"
	"os"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
)

var (
	home    = netip.MustParseAddr("127.0.0.2")
	seedbox = netip.MustParseAddr("127.0.0.3")
)

// pair dials seedbox:8485 from home and returns both ends.
func pair(t *testing.T, n *fault.Network) (client, server net.Conn) {
	t.Helper()
	l, err := n.Host(seedbox).Listen("tcp", "127.0.0.3:8485")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	client, err = n.Host(home).DialContext(t.Context(), "tcp", "127.0.0.3:8485")
	if err != nil {
		t.Fatal(err)
	}
	server, err = l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })
	return client, server
}

type read struct {
	data string
	err  error
}

// readAsync reads once from c, as a protocol's receive loop does.
func readAsync(c net.Conn) <-chan read {
	out := make(chan read, 1)
	go func() {
		b := make([]byte, 64)
		n, err := c.Read(b)
		out <- read{string(b[:n]), err}
	}()
	return out
}

func write(t *testing.T, c net.Conn, s string) {
	t.Helper()
	if n, err := c.Write([]byte(s)); n != len(s) || err != nil {
		t.Fatalf("Write(%q) = %d, %v", s, n, err)
	}
}

// set plays f on every connection p covers, as the simulation tier's
// one-call flip.
func set(t *testing.T, p *fault.Plan, f fault.Fault) {
	t.Helper()
	if _, err := p.Set(fault.Rule{Fault: f}); err != nil {
		t.Fatal(err)
	}
}

func TestStallDropsWritesAndHoldsReadsUntilCleared(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := fault.NewNetwork()
		client, server := pair(t, n)
		atServer, atClient := readAsync(server), readAsync(client)
		set(t, n.Path(home, seedbox), fault.Stall{})
		write(t, server, "lost")
		write(t, client, "held")
		synctest.Wait()
		select {
		case r := <-atServer:
			t.Fatalf("a read under Stall returned: %+v", r)
		case r := <-atClient:
			t.Fatalf("a write under Stall arrived: %+v", r)
		default:
		}
		n.Path(seedbox, home).Clear("")
		if r := <-atServer; r != (read{"held", nil}) {
			t.Fatalf("after the clear the server read %+v, want held", r)
		}
		write(t, server, "after")
		if r := <-atClient; r != (read{"after", nil}) {
			t.Fatalf("after the clear the client read %+v, want after", r)
		}
	})
}

func TestResetKillsBothEndsAndLaterDials(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := fault.NewNetwork()
		client, server := pair(t, n)
		got := readAsync(server)
		set(t, n.Path(home, seedbox), fault.Reset{})
		if r := <-got; !errors.Is(r.err, fault.ErrReset) {
			t.Fatalf("server read %+v, want ErrReset", r)
		}
		if _, err := client.Write([]byte("x")); !errors.Is(err, fault.ErrReset) {
			t.Fatalf("client write: %v, want ErrReset", err)
		}
		again, err := n.Host(home).DialContext(t.Context(), "tcp", "127.0.0.3:8485")
		if err != nil {
			t.Fatalf("dial under Reset: %v", err)
		}
		t.Cleanup(func() { _ = again.Close() })
		if _, err := again.Read(make([]byte, 1)); !errors.Is(err, fault.ErrReset) {
			t.Fatalf("a connection dialed under Reset read %v, want ErrReset", err)
		}
		n.Path(home, seedbox).Clear("")
		if _, err := client.Write([]byte("x")); !errors.Is(err, fault.ErrReset) {
			t.Fatalf("a reset conn wrote after the clear: %v", err)
		}
	})
}

func TestRefuseRefusesDialsUntilCleared(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := fault.NewNetwork()
		client, server := pair(t, n)
		set(t, n.Path(home, seedbox), fault.Refuse{})
		_, err := n.Host(home).DialContext(t.Context(), "tcp", "127.0.0.3:8485")
		if err == nil || err.Error() != "dial tcp 127.0.0.3:8485: connection refused" {
			t.Fatalf("dial under Refuse: %v", err)
		}
		got := readAsync(server)
		write(t, client, "open")
		if r := <-got; r != (read{"open", nil}) {
			t.Fatalf("a connection open before Refuse read %+v", r)
		}
		n.Path(home, seedbox).Clear("")
		c, err := n.Host(home).DialContext(t.Context(), "tcp", "127.0.0.3:8485")
		if err != nil {
			t.Fatalf("dial after the clear: %v", err)
		}
		_ = c.Close()
	})
}

func TestCutPassesExactlyNBytes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := fault.NewNetwork()
		client, server := pair(t, n)
		set(t, n.Path(home, seedbox), fault.Cut(5))
		got := make(chan string, 1)
		go func() {
			b, _ := io.ReadAll(client)
			got <- string(b)
		}()
		write(t, server, "hel")
		if n, err := server.Write([]byte("lo world")); n != 2 || !errors.Is(err, fault.ErrCut) {
			t.Fatalf("Write = %d, %v; want 2, ErrCut", n, err)
		}
		if s := <-got; s != "hello" {
			t.Fatalf("delivered %q, want hello", s)
		}
		if _, err := server.Write([]byte("x")); !errors.Is(err, fault.ErrCut) {
			t.Fatalf("write after the cut: %v", err)
		}
	})
}

func TestLatencyDelaysEachWayOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := fault.NewNetwork()
		client, server := pair(t, n)
		set(t, n.Path(home, seedbox), fault.Latency(100*time.Millisecond))
		start := time.Now()
		got := readAsync(server)
		write(t, client, "ping")
		if r := <-got; r != (read{"ping", nil}) {
			t.Fatalf("read %+v", r)
		}
		if d := time.Since(start); d != 100*time.Millisecond {
			t.Fatalf("delivered after %v, want 100ms", d)
		}
	})
}

func TestDialWithNoListenerRefusesAtOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		_, err := fault.NewNetwork().Host(home).DialContext(t.Context(), "tcp", "127.0.0.3:9")
		if !errors.Is(err, fault.ErrRefused) || err.Error() != "dial tcp 127.0.0.3:9: connection refused" {
			t.Fatalf("dial: %v", err)
		}
		if d := time.Since(start); d != 0 {
			t.Fatalf("refused after %v", d)
		}
	})
}

func TestStalledDialReturnsTheContextError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := fault.NewNetwork()
		l, err := n.Host(seedbox).Listen("tcp", "127.0.0.3:8485")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = l.Close() })
		set(t, n.Path(home, seedbox), fault.Stall{})
		ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
		defer cancel()
		start := time.Now()
		_, err = n.Host(home).DialContext(ctx, "tcp", "127.0.0.3:8485")
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("dial: %v", err)
		}
		if d := time.Since(start); d != 3*time.Second {
			t.Fatalf("failed after %v, want 3s", d)
		}
	})
}

func TestEachFamilyHasItsOwnPath(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := fault.NewNetwork()
		home6, seedbox6 := netip.MustParseAddr("fd00::2"), netip.MustParseAddr("fd00::3")
		l, err := n.Host(seedbox6, seedbox).Listen("tcp", "[::]:8485")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = l.Close() })
		if got := l.Addr().String(); got != "[::]:8485" {
			t.Fatalf("listener at %s", got)
		}
		set(t, n.Path(home6, seedbox6), fault.Stall{})
		dialer := n.Host(home6, home)
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		if _, err := dialer.DialContext(ctx, "tcp", "[fd00::3]:8485"); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("IPv6 dial: %v", err)
		}
		c, err := dialer.DialContext(t.Context(), "tcp", "127.0.0.3:8485")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = c.Close() })
		s, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
		got := [4]string{c.LocalAddr().String(), c.RemoteAddr().String(), s.LocalAddr().String(), s.RemoteAddr().String()}
		want := [4]string{"127.0.0.2:49152", "127.0.0.3:8485", "127.0.0.3:8485", "127.0.0.2:49152"}
		if got != want {
			t.Fatalf("addresses %v, want %v", got, want)
		}
	})
}

func TestListenOnPortZeroPicksAPort(t *testing.T) {
	n := fault.NewNetwork()
	h := n.Host(seedbox)
	l, err := h.Listen("tcp", "127.0.0.3:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	if got := l.Addr().String(); got != "127.0.0.3:49152" {
		t.Fatalf("listener at %s", got)
	}
	if _, err := h.Listen("tcp", "127.0.0.3:49152"); err == nil || err.Error() != "listen tcp 127.0.0.3:49152: address already in use" {
		t.Fatalf("second listen: %v", err)
	}
	if _, err := h.Listen("tcp", "127.0.0.2:80"); err == nil || err.Error() != "listen tcp 127.0.0.2:80: cannot assign requested address" {
		t.Fatalf("listen on another host's address: %v", err)
	}
}

func TestUnixPathsBelongToOneHost(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := fault.NewNetwork()
		box := n.Host(seedbox)
		l, err := box.Listen("unix", "/data/control.sock")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = l.Close() })
		if _, err := n.Host(home).DialContext(t.Context(), "unix", "/data/control.sock"); !errors.Is(err, fault.ErrRefused) {
			t.Fatalf("dial from another host: %v", err)
		}
		c, err := box.DialContext(t.Context(), "unix", "/data/control.sock")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = c.Close() })
		s, err := l.Accept()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
		got := readAsync(s)
		write(t, c, "join")
		if r := <-got; r != (read{"join", nil}) {
			t.Fatalf("read %+v", r)
		}
	})
}

func TestReadDeadlineFiresOnFakeTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client, _ := pair(t, fault.NewNetwork())
		start := time.Now()
		if err := client.SetReadDeadline(start.Add(2 * time.Second)); err != nil {
			t.Fatal(err)
		}
		if r := <-readAsync(client); !errors.Is(r.err, os.ErrDeadlineExceeded) {
			t.Fatalf("read %+v", r)
		}
		if d := time.Since(start); d != 2*time.Second {
			t.Fatalf("deadline fired after %v, want 2s", d)
		}
	})
}

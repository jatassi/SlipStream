package mux_test

import (
	"bytes"
	"io"
	"net/netip"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link/internal/mux"
)

var (
	home    = netip.MustParseAddr("127.0.0.2")
	seedbox = netip.MustParseAddr("127.0.0.3")
	config  = mux.Config{Keepalive: 25 * time.Second, WriteTimeout: 10 * time.Second, StreamWindow: 16 << 20}
)

// sessions joins home to seedbox over n with one session each side.
func sessions(t *testing.T, n *fault.Network) (client, server *mux.Session) {
	t.Helper()
	l, err := n.Host(seedbox).Listen("tcp", "127.0.0.3:8485")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()
	c, err := n.Host(home).DialContext(t.Context(), "tcp", "127.0.0.3:8485")
	if err != nil {
		t.Fatal(err)
	}
	s, err := l.Accept()
	if err != nil {
		t.Fatal(err)
	}
	if client, err = mux.Client(c, config); err != nil {
		t.Fatal(err)
	}
	if server, err = mux.Server(s, config); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })
	return client, server
}

// echo is run in two bubbles, so a library that shares timers or channels
// between sessions through a package variable fails the second one.
func echo(t *testing.T) {
	client, server := sessions(t, fault.NewNetwork())
	go func() {
		st, err := server.Accept()
		if err != nil {
			return
		}
		_, _ = io.Copy(st, st)
		_ = st.Close()
	}()
	st, err := client.Open(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	// Four times the initial window, so the echo needs window updates.
	sent := bytes.Repeat([]byte("slipstream"), 100<<10)
	go func() { _, _ = st.Write(sent) }()
	got := make([]byte, len(sent))
	if _, err := io.ReadFull(st, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, sent) {
		t.Fatal("the echo differs from what was sent")
	}
}

func TestEchoInABubble(t *testing.T)       { synctest.Test(t, echo) }
func TestEchoInASecondBubble(t *testing.T) { synctest.Test(t, echo) }

func TestKeepaliveEndsAStalledSession(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := fault.NewNetwork()
		client, server := sessions(t, n)
		synctest.Wait()
		start := time.Now()
		if err := n.Path(home, seedbox).Set(fault.Rule{Fault: fault.Stall{}}); err != nil {
			t.Fatal(err)
		}
		<-client.Done()
		if got := time.Since(start); got != 35*time.Second {
			t.Fatalf("the stalled session ended after %v, want keepalive + write timeout = 35s", got)
		}
		<-server.Done()
	})
}

func TestStreamWindowUnderTheInitialWindowIsRefused(t *testing.T) {
	err := mux.Config{Keepalive: time.Second, WriteTimeout: time.Second, StreamWindow: 1 << 10}.Check()
	if err == nil || err.Error() != "stream_window 1024 is under 262144 bytes" {
		t.Fatalf("Check() = %v", err)
	}
}

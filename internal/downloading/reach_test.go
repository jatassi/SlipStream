package downloading_test

import (
	"net/netip"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/downloading"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

func TestPublicPeerRule(t *testing.T) {
	for _, tc := range []struct {
		addr     string
		proves   bool
		loopback bool
	}{
		{"203.0.113.9", true, true},
		{"2606:4700::1111", true, true},
		{"::ffff:203.0.113.9", true, true},
		{"100.128.0.1", true, true},
		{"127.0.0.1", false, true},
		{"127.0.0.3", false, true},
		{"::1", false, true},
		{"::ffff:127.0.0.1", false, true},
		{"10.0.0.5", false, false},
		{"172.16.4.4", false, false},
		{"192.168.1.20", false, false},
		{"::ffff:192.168.1.20", false, false},
		{"fd00::1", false, false},
		{"169.254.10.10", false, false},
		{"fe80::1", false, false},
		{"100.64.0.1", false, false},
		{"100.127.255.254", false, false},
		{"0.0.0.0", false, false},
		{"224.0.0.1", false, false},
	} {
		a := netip.MustParseAddr(tc.addr)
		if got := downloading.PublicPeer(false)(a); got != tc.proves {
			t.Errorf("a peer at %s proves the port: %v, want %v", a, got, tc.proves)
		}
		if got := downloading.PublicPeer(true)(a); got != tc.loopback {
			t.Errorf("with loopback counting, a peer at %s proves the port: %v, want %v", a, got, tc.loopback)
		}
	}
}

// facts is the client's facts once its next facts tick has passed.
func (c *client) facts() *dlv1.Facts {
	time.Sleep(timings.Facts)
	synctest.Wait()
	return c.h.lastFacts()
}

// running starts a client holding one running torrent.
func running(t *testing.T) (*node, *client) {
	t.Helper()
	n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
	f := newFile(t, "Orchard.mkv")
	n.world.Payload(f.hash, f.payload)
	id := n.hold(f)
	c := n.start()
	c.send(live(f.started(id, 1)))
	return n, c
}

func TestAPeerFromTheInternetProvesTheListenPort(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n, c := running(t)
		if !c.engine.running().WatchIncoming {
			t.Fatal("the client does not watch incoming peers while its port is unproven")
		}
		n.world.Incoming(listen, netip.MustParseAddr("192.168.1.20"))
		synctest.Wait()
		if f := c.facts(); f.GetIncomingAtUnixMs() != 0 {
			t.Fatalf("a peer on the LAN proved the port: facts %v", f)
		}

		proved := time.Now()
		n.world.Incoming(listen, netip.MustParseAddr("203.0.113.9"))
		synctest.Wait()
		if f := c.facts(); f.GetIncomingAtUnixMs() != proved.UnixMilli() {
			t.Fatalf("facts after a peer from the internet connected in = %v, want incoming at %d", f, proved.UnixMilli())
		}
		if c.engine.running().WatchIncoming {
			t.Error("the client still watches incoming peers once one proved the port")
		}
	})
}

func TestAMovedListenerIsUnprovenUntilAPeerConnectsInAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n, c := running(t)
		public := netip.MustParseAddr("203.0.113.9")
		n.world.Incoming(listen, public)
		synctest.Wait()
		if f := c.facts(); f.GetIncomingAtUnixMs() == 0 {
			t.Fatalf("a peer from the internet did not prove the port: facts %v", f)
		}

		moved := netip.MustParseAddrPort("192.0.2.7:6881")
		n.world.Move(listen, moved)
		if f := c.facts(); f.GetBound() != moved.String() || f.GetIncomingAtUnixMs() != 0 {
			t.Fatalf("facts once the listener moved = %v, want bound to %s and unproven", f, moved)
		}
		if !c.engine.running().WatchIncoming {
			t.Fatal("the client does not watch incoming peers on its new address")
		}

		proved := time.Now()
		n.world.Incoming(moved, public)
		synctest.Wait()
		if f := c.facts(); f.GetIncomingAtUnixMs() != proved.UnixMilli() {
			t.Fatalf("facts after a peer connected in to the new address = %v, want incoming at %d", f, proved.UnixMilli())
		}
	})
}

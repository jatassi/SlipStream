package link_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/netip"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/internal/wire"
	"github.com/jatassi/slipstream/internal/link/linktest"
)

// echo serves each lane by sending back what it reads, and reports each
// lane's peer.
func echo(peers chan<- link.Key) func(context.Context, link.Key, net.Conn) {
	return func(_ context.Context, peer link.Key, c net.Conn) {
		peers <- peer
		_, _ = io.Copy(c, c)
	}
}

func roundTrip(t *testing.T, c net.Conn, msg []byte) {
	t.Helper()
	if _, err := c.Write(msg); err != nil {
		t.Fatalf("write on the lane: %v", err)
	}
	got := make([]byte, len(msg))
	if _, err := io.ReadFull(c, got); err != nil || !bytes.Equal(got, msg) {
		t.Fatalf("the lane sent back %d bytes of %d: %v", len(got), len(msg), err)
	}
}

// remoteAlert is the TLS alert the peer sent, "" for none.
func remoteAlert(err error) string {
	var op *net.OpError
	if errors.As(err, &op) && op.Op == "remote error" {
		return op.Err.Error()
	}
	return ""
}

func expectNoLane(t *testing.T, peers chan link.Key) {
	t.Helper()
	synctest.Wait()
	select {
	case peer := <-peers:
		t.Fatalf("a lane from %s reached the handler", peer.Fingerprint())
	default:
	}
}

// The Worker dials its lane with no addresses, so it reaches the
// Controller at the address it keeps its session at.
func TestALaneAndASessionShareTheListener(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls, peers := make(chan link.Key, 1), make(chan link.Key, 1)
		p := linktest.New(t, linktest.Side{Lockstep: lockstep(calls), Lanes: echo(peers)}, linktest.Side{})
		p.Controller.Trust.AllowLane(p.Worker.Key)
		p.Connect(t)
		changes := p.Controller.Subscribe(t.Context())

		lane, err := p.Worker.DialLane(t.Context(), p.Controller.Key, link.Addresses{})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = lane.Close() }()
		roundTrip(t, lane, []byte("chunk"))
		if peer := <-peers; peer != p.Worker.Key {
			t.Fatalf("the lane handler saw %s, want the Worker", peer.Fingerprint())
		}
		if seq, err := syncOnce(t.Context(), p.Worker.Link, p.Controller.Key); seq != 1 || err != nil {
			t.Fatalf("a call beside the lane = %d, %v", seq, err)
		}
		<-calls
		synctest.Wait()
		select {
		case c := <-changes:
			t.Fatalf("the lane changed the Controller's sessions: %+v", c)
		default:
		}
	})
}

// A Controller admits an unknown key to Pair, and a Worker refuses it,
// but neither lets it open a lane. TLS 1.3 ends the dialer's handshake
// before the peer checks the dialer's key, so the refusal can arrive on
// the lane's first read.
func TestAKeyWithNoTransferCantOpenALane(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		peers := make(chan link.Key, 2)
		p := linktest.New(t, linktest.Side{Lanes: echo(peers)}, linktest.Side{Lanes: echo(peers)})
		p.Controller.Trust.AllowLane(p.Worker.Key)
		p.Worker.Trust.AllowLane(p.Controller.Key)
		stranger := linktest.Start(t, p.Net, "stranger", []netip.Addr{linktest.OtherWorkerAddr}, linktest.Side{})
		p.Controller.Trust.Set(stranger.Key, link.Pairing{})

		for _, n := range []*linktest.Node{p.Controller, p.Worker} {
			lane, err := stranger.DialLane(t.Context(), n.Key, n.Target().Addresses)
			if err == nil {
				_, err = lane.Read(make([]byte, 1))
				_ = lane.Close()
			}
			if remoteAlert(err) == "" {
				t.Fatalf("a stranger's lane to %s ended with %v, want a TLS alert from the peer", n.Key.Fingerprint(), err)
			}
		}
		expectNoLane(t, peers)

		if _, err := stranger.UnwelcomedClient(t.Context(), p.Controller.Target()); err != nil {
			t.Fatalf("the Controller refused the stranger's session, which its Trust admits to Pair: %v", err)
		}
	})
}

func TestALaneFromAnotherBuildIsRefusedByALPN(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		peers := make(chan link.Key, 1)
		p := linktest.New(t, linktest.Side{Lanes: echo(peers)}, linktest.Side{Version: "0.0.1-test"})
		p.Controller.Trust.AllowLane(p.Worker.Key)
		_, err := p.Worker.DialLane(t.Context(), p.Controller.Key, p.Controller.Target().Addresses)
		if alert := remoteAlert(err); alert != "tls: no application protocol" {
			t.Fatalf("a lane from another build = %v, want the no_application_protocol alert", err)
		}
		expectNoLane(t, peers)
	})
}

// tlsServer accepts one connection at 127.0.0.2:8485 and runs a TLS
// handshake on it with this build's server config, offering protos. It
// returns the server's key and the handshake's result.
func tlsServer(t *testing.T, n *fault.Network, protos []string) (key link.Key, handshakes <-chan error) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := wire.Certificate(priv)
	if err != nil {
		t.Fatal(err)
	}
	cfg := wire.ServerConfig(cert, wire.LaneALPN("0.0.0-test"), func(ed25519.PublicKey, string) error { return nil })
	cfg.NextProtos = protos
	ln, err := n.Host(linktest.ControllerAddr).Listen("tcp", "127.0.0.2:8485")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	done := make(chan error, 1)
	go func() {
		c, err := ln.Accept()
		if err == nil {
			defer func() { _ = c.Close() }()
			err = tls.Server(c, cfg).HandshakeContext(t.Context())
		}
		done <- err
	}()
	return link.Key(pub), done
}

// A server from before lanes negotiates no protocol, so it would take a
// lane as a Node link session. The same server offering this build's lane
// protocol takes the lane.
func TestALaneDialRefusesAServerFromBeforeLanes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		protos []string
		takes  bool
	}{
		{"before lanes", nil, false},
		{"with lanes", []string{wire.LaneALPN("0.0.0-test")}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				n := fault.NewNetwork()
				worker := linktest.Start(t, n, "worker", []netip.Addr{linktest.WorkerAddr}, linktest.Side{})
				key, handshakes := tlsServer(t, n, tc.protos)
				c, dialErr := worker.DialLane(t.Context(), key, addresses(t, "127.0.0.2:8485"))
				if dialErr == nil {
					_ = c.Close()
				}
				served := <-handshakes
				if took := dialErr == nil && served == nil; took != tc.takes {
					t.Fatalf("the lane dial = %v and the server's handshake = %v, want the lane taken %v", dialErr, served, tc.takes)
				}
			})
		})
	}
}

func TestLaneBytesNeverGoBackwards(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		peers := make(chan link.Key, 1)
		p := linktest.New(t, linktest.Side{Lanes: echo(peers)}, linktest.Side{})
		p.Controller.Trust.AllowLane(p.Worker.Key)
		if got := p.Worker.LaneBytes(p.Controller.Key); got != (link.Stats{}) {
			t.Fatalf("lane bytes before any lane = %+v", got)
		}
		const size = 64 << 10
		var last link.Stats
		for i := range 2 {
			lane, err := p.Worker.DialLane(t.Context(), p.Controller.Key, p.Controller.Target().Addresses)
			if err != nil {
				t.Fatal(err)
			}
			roundTrip(t, lane, make([]byte, size))
			<-peers
			open := p.Worker.LaneBytes(p.Controller.Key)
			if open.Sent < last.Sent+size || open.Received < last.Received+size {
				t.Fatalf("with lane %d open the Worker counts %+v, want %d bytes each way past %+v", i, open, size, last)
			}
			_ = lane.Close()
			synctest.Wait()
			closed := p.Worker.LaneBytes(p.Controller.Key)
			if closed.Sent < open.Sent || closed.Received < open.Received {
				t.Fatalf("closing lane %d took the Worker's count from %+v to %+v", i, open, closed)
			}
			if at := p.Controller.LaneBytes(p.Worker.Key); at.Sent != closed.Received || at.Received != closed.Sent {
				t.Fatalf("the Controller counts %+v and the Worker %+v, want each side's sent to be the other's received", at, closed)
			}
			last = closed
		}
	})
}

// Two Workers admit each other only to lanes. With the first Worker's
// dials refused, the second Worker still opens a lane to it.
func TestWorkersOpenALaneTheWayTheNetworkAllows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		peers := [2]chan link.Key{make(chan link.Key, 1), make(chan link.Key, 1)}
		p := linktest.NewTrio(t, linktest.Side{}, [2]linktest.Side{{Lanes: echo(peers[0])}, {Lanes: echo(peers[1])}})
		a, b := p.Workers[0], p.Workers[1]
		a.Trust.AllowLane(b.Key)
		b.Trust.AllowLane(a.Key)
		p.Connect(t)
		if _, err := p.Net.Path(linktest.WorkerAddr, linktest.OtherWorkerAddr).Set(fault.Rule{ID: "out", Origin: linktest.WorkerAddr.String(), Fault: fault.Refuse{}}); err != nil {
			t.Fatal(err)
		}

		if _, err := a.DialLane(t.Context(), b.Key, b.Target().Addresses); !errors.Is(err, fault.ErrRefused) {
			t.Fatalf("a lane dialed out of the refusing Worker = %v, want refused", err)
		}
		expectNoLane(t, peers[1])
		lane, err := b.DialLane(t.Context(), a.Key, a.Target().Addresses)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = lane.Close() }()
		roundTrip(t, lane, []byte("chunk"))
		if peer := <-peers[0]; peer != b.Key {
			t.Fatalf("the Worker's lane handler saw %s, want the other Worker", peer.Fingerprint())
		}
	})
}

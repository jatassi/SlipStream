// Package linktest runs Links on an in-memory fault.Network with a static
// Trust, for link and rpc tests.
package linktest

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/netip"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
)

// Port is where every test Node listens.
const Port = 8485

// Side configures one Node. The zero Side runs version "0.0.0-test",
// answers lockstep calls 404, and answers Pair NOT_A_CONTROLLER.
type Side struct {
	Version  string
	Lockstep http.Handler
	Pair     link.PairFunc
	Timings  link.Timings
	Log      *slog.Logger
	// Net wraps the Node's Host for its dials, as a recorder does.
	Net func(*fault.Host) link.Network
}

// Node is one running Link.
type Node struct {
	*link.Link
	Key   link.Key
	Host  *fault.Host
	Trust *Trust
	// Listening is every address the Node accepts sessions on.
	Listening link.Addresses
}

// Target is how a peer keeps a session to n.
func (n *Node) Target() link.Target {
	return link.Target{Key: n.Key, Addresses: n.Listening}
}

// Start runs a Link named name on net at addrs, listening on Port at each.
// Its Trust refuses every key until the test sets one.
func Start(t testing.TB, net *fault.Network, name string, addrs []netip.Addr, side Side) *Node {
	t.Helper()
	if side.Version == "" {
		side.Version = "0.0.0-test"
	}
	if side.Lockstep == nil {
		side.Lockstep = http.NotFoundHandler()
	}
	if side.Pair == nil {
		side.Pair = notAController
	}
	if side.Timings == (link.Timings{}) {
		side.Timings = link.DefaultTimings()
	}
	if side.Log == nil {
		side.Log = slog.New(slog.DiscardHandler)
	}
	seed := sha256.Sum256([]byte("linktest " + name))
	key := link.NewPrivateKey(seed)
	n := &Node{Key: key.Public(), Host: net.Host(addrs...), Trust: &Trust{}}
	var dialer link.Network = n.Host
	if side.Net != nil {
		dialer = side.Net(n.Host)
	}
	l, err := link.New(link.Config{
		Key:      key,
		Node:     link.NodeInfo{Version: side.Version, OS: "linux", Arch: "amd64", InstallKind: linkv1.InstallKind_INSTALL_KIND_DEV},
		Timings:  side.Timings,
		Settings: link.DefaultSettings(),
		Net:      dialer,
		Rand:     rand.New(&lockedSource{src: rand.NewChaCha8(seed)}), //nolint:gosec // seeded on purpose so tests replay
		Trust:    n.Trust,
		Lockstep: side.Lockstep,
		Pair:     side.Pair,
		Log:      side.Log,
	})
	if err != nil {
		t.Fatal(err)
	}
	n.Link = l
	ctx, cancel := context.WithCancel(context.Background())
	var served sync.WaitGroup
	var listening []netip.AddrPort
	for _, a := range addrs {
		ap := netip.AddrPortFrom(a, Port)
		ln, err := n.Host.Listen("tcp", ap.String())
		if err != nil {
			t.Fatal(err)
		}
		listening = append(listening, ap)
		served.Go(func() {
			if err := l.Serve(ctx, ln); err != nil {
				t.Errorf("serve: %v", err)
			}
		})
	}
	if n.Listening, err = link.ParseAddresses(listening); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		served.Wait()
		_ = l.Close()
	})
	return n
}

// Pair is a Controller on 127.0.0.2 and a Worker on 127.0.0.3, each
// pinning the other.
type Pair struct {
	Net                *fault.Network
	Controller, Worker *Node
}

var (
	ControllerAddr = netip.MustParseAddr("127.0.0.2")
	WorkerAddr     = netip.MustParseAddr("127.0.0.3")
)

func New(t testing.TB, controller, worker Side) *Pair {
	t.Helper()
	net := fault.NewNetwork()
	p := &Pair{
		Net:        net,
		Controller: Start(t, net, "controller", []netip.Addr{ControllerAddr}, controller),
		Worker:     Start(t, net, "worker", []netip.Addr{WorkerAddr}, worker),
	}
	p.Controller.Trust.Set(p.Worker.Key, link.Pinned{})
	p.Worker.Trust.Set(p.Controller.Key, link.Pinned{})
	return p
}

// Connect has the Worker keep the Controller and returns once both sides
// see the session live.
func (p *Pair) Connect(t testing.TB) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	atController, atWorker := p.Controller.Subscribe(ctx), p.Worker.Subscribe(ctx)
	p.Worker.Keep(p.Controller.Target())
	AwaitLive(t, atWorker, p.Controller.Key)
	AwaitLive(t, atController, p.Worker.Key)
}

// AwaitLive reads changes until key is Live and returns that state.
func AwaitLive(t testing.TB, changes <-chan link.Change, key link.Key) link.Live {
	t.Helper()
	for c := range changes {
		if live, ok := c.State.(link.Live); ok && c.Key == key {
			return live
		}
	}
	t.Fatal("the link closed before the peer was live")
	return link.Live{}
}

// Trust admits each key as set, and refuses the rest.
type Trust struct {
	mu sync.Mutex
	m  map[link.Key]link.Admission
}

func (t *Trust) Set(key link.Key, a link.Admission) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.m == nil {
		t.m = map[link.Key]link.Admission{}
	}
	t.m[key] = a
}

func (t *Trust) Admit(key link.Key) link.Admission {
	t.mu.Lock()
	defer t.mu.Unlock()
	if a, ok := t.m[key]; ok {
		return a
	}
	return link.Refused{}
}

func notAController(context.Context, link.Key, *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	return &linkv1.PairResponse{Outcome: &linkv1.PairResponse_Refused{
		Refused: &linkv1.Refused{Reason: linkv1.RefusedReason_REFUSED_REASON_NOT_A_CONTROLLER},
	}}, nil
}

type lockedSource struct {
	mu  sync.Mutex
	src *rand.ChaCha8
}

func (l *lockedSource) Uint64() uint64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.src.Uint64()
}

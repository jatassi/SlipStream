// Package linktest runs Links on an in-memory fault.Network with a static
// Trust, for link and rpc tests.
package linktest

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
)

const Port = 8485

// Timings and StreamWindow are the link's shipped defaults, spelled out:
// only the Node's composition root reads its defaults.
var Timings = link.Timings{
	Handshake:     10 * time.Second,
	Keepalive:     25 * time.Second,
	WriteTimeout:  10 * time.Second,
	RedialMin:     time.Second,
	RedialMax:     time.Minute,
	FallbackDelay: 300 * time.Millisecond,
	JoinDial:      20 * time.Second,
	AwaitDial:     10 * time.Minute,
}

const StreamWindow = 16 << 20

// Side configures one Node. The zero Side runs version "0.0.0-test",
// answers lockstep calls 404, and answers Pair NOT_A_CONTROLLER.
type Side struct {
	Version  string
	Lockstep http.Handler
	Pair     link.PairFunc
	// Trust replaces the Node's static Trust for its sessions.
	Trust     link.Trust
	Timings   link.Timings
	Advertise []netip.AddrPort
	Log       *slog.Logger
	// Net wraps the Node's Host for its dials, as a recorder does.
	Net func(*fault.Host) link.Network
	// Streams serves stream kinds beside the link's own.
	Streams map[link.Kind]func(peer link.Key, s link.Stream)
	// Lanes serves each lane from a key the Node's Trust admits with
	// AllowLane. With none the Node refuses every lane.
	Lanes func(ctx context.Context, peer link.Key, c net.Conn)
}

type Node struct {
	*link.Link
	Key   link.Key
	Host  *fault.Host
	Trust *Trust

	listening link.Addresses
}

func (n *Node) Target() link.Target {
	return link.Target{Key: n.Key, Addresses: n.listening}
}

// Start runs a Link named name on nw at addrs, listening on Port at each.
// Its Trust refuses every key until the test sets one.
func Start(t testing.TB, nw *fault.Network, name string, addrs []netip.Addr, side Side) *Node {
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
		side.Timings = Timings
	}
	if side.Log == nil {
		side.Log = slog.New(slog.DiscardHandler)
	}
	seed := sha256.Sum256([]byte("linktest " + name))
	key := link.NewPrivateKey(seed)
	n := &Node{Key: key.Public(), Host: nw.Host(addrs...), Trust: &Trust{}}
	var trust link.Trust = n.Trust
	if side.Trust != nil {
		trust = side.Trust
	}
	var dialer link.Network = n.Host
	if side.Net != nil {
		dialer = side.Net(n.Host)
	}
	l, err := link.New(link.Config{
		Key:      key,
		Node:     link.NodeInfo{Version: side.Version, OS: "linux", Arch: "amd64", InstallKind: linkv1.InstallKind_INSTALL_KIND_DEV},
		Timings:  side.Timings,
		Settings: link.Settings{Advertise: side.Advertise, StreamWindow: StreamWindow},
		Net:      dialer,
		Rand:     rand.New(&splitmix{x: binary.BigEndian.Uint64(seed[:])}), //nolint:gosec // seeded on purpose so tests replay
		Trust:    trust,
		Lockstep: side.Lockstep,
		Pair:     side.Pair,
		Log:      side.Log,
	})
	if err != nil {
		t.Fatal(err)
	}
	for kind, h := range side.Streams {
		l.HandleStream(kind, h)
	}
	if side.Lanes != nil {
		l.HandleLanes(lanes{n.Trust, side.Lanes})
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
		serve := l.Serve(ln)
		served.Go(func() {
			if err := serve(ctx); err != nil {
				t.Errorf("serve: %v", err)
			}
		})
	}
	if n.listening, err = link.ParseAddresses(listening); err != nil {
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
	nw := fault.NewNetwork()
	p := &Pair{
		Net:        nw,
		Controller: Start(t, nw, "controller", []netip.Addr{ControllerAddr}, controller),
		Worker:     Start(t, nw, "worker", []netip.Addr{WorkerAddr}, worker),
	}
	pin(p.Controller, p.Worker)
	return p
}

// Connect has the Worker keep the Controller and returns once both sides
// see the session live.
func (p *Pair) Connect(t testing.TB) {
	t.Helper()
	connect(t, p.Controller, p.Worker)
}

// Trio is a Controller on 127.0.0.2 and Workers on 127.0.0.3 and
// 127.0.0.4. Each Worker and the Controller pin each other. The Workers
// admit each other to nothing until a test calls AllowLane, as two Workers
// of one Controller admit each other only to the lanes of their Transfers.
type Trio struct {
	Net        *fault.Network
	Controller *Node
	Workers    [2]*Node
}

var OtherWorkerAddr = netip.MustParseAddr("127.0.0.4")

func NewTrio(t testing.TB, controller Side, workers [2]Side) *Trio {
	t.Helper()
	nw := fault.NewNetwork()
	p := &Trio{
		Net:        nw,
		Controller: Start(t, nw, "controller", []netip.Addr{ControllerAddr}, controller),
		Workers: [2]*Node{
			Start(t, nw, "worker", []netip.Addr{WorkerAddr}, workers[0]),
			Start(t, nw, "other worker", []netip.Addr{OtherWorkerAddr}, workers[1]),
		},
	}
	for _, w := range p.Workers {
		pin(p.Controller, w)
	}
	return p
}

// Connect has each Worker keep the Controller and returns once every
// session is live on both sides.
func (p *Trio) Connect(t testing.TB) {
	t.Helper()
	for _, w := range p.Workers {
		connect(t, p.Controller, w)
	}
}

func pin(controller, worker *Node) {
	controller.Trust.Set(worker.Key, link.Pinned{})
	worker.Trust.Set(controller.Key, link.Pinned{})
}

func connect(t testing.TB, controller, worker *Node) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	atController, atWorker := controller.Subscribe(ctx), worker.Subscribe(ctx)
	worker.Keep(controller.Target())
	AwaitLive(t, atWorker, controller.Key)
	AwaitLive(t, atController, worker.Key)
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

// Trust admits each key as set, and refuses the rest. It admits a lane
// only from a key given to AllowLane, whatever Set gave it.
type Trust struct {
	mu    sync.Mutex
	m     map[link.Key]link.Admission
	lanes map[link.Key]bool
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

func (t *Trust) AllowLane(key link.Key) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.lanes == nil {
		t.lanes = map[link.Key]bool{}
	}
	t.lanes[key] = true
}

func (t *Trust) AdmitLane(key link.Key) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lanes[key]
}

// lanes admits lanes by its Trust and serves them with serve.
type lanes struct {
	trust *Trust
	serve func(ctx context.Context, peer link.Key, c net.Conn)
}

func (l lanes) Admit(key link.Key) bool { return l.trust.AdmitLane(key) }

func (l lanes) Serve(ctx context.Context, key link.Key, c net.Conn) { l.serve(ctx, key, c) }

func notAController(context.Context, link.Key, *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	return &linkv1.PairResponse{Outcome: &linkv1.PairResponse_Refused{
		Refused: &linkv1.Refused{Reason: linkv1.RefusedReason_REFUSED_REASON_NOT_A_CONTROLLER},
	}}, nil
}

// splitmix is SplitMix64, a replayable stream for a test Node, which has
// no Node to hand it one. It is safe for concurrent draws.
type splitmix struct {
	mu sync.Mutex
	x  uint64
}

func (s *splitmix) Uint64() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.x += 0x9e3779b97f4a7c15
	z := s.x
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

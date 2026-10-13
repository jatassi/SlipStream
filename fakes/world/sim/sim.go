// Package sim is the simulation tier: a Scenario's Nodes run whole, each
// through node.Start, on one in-memory fault.Network inside the caller's
// testing/synctest bubble, so partitions, re-dials and restarts take fake
// time. Node i is at 127.0.0.(i+2) as in the process world, and the world
// sets it up with the same config.yaml and setup steps.
package sim

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/internal/setup"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/role/roletest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// Version is every Node's build unless an Option or a restart says
// otherwise.
const Version = "0.0.0-sim"

const logLinesOnFailure = 50

type options struct {
	version string
	probe   bool
}

type Option func(*options)

// Build runs every Node as build version.
func Build(version string) Option { return func(o *options) { o.version = version } }

// WithProbe serves and holds roletest's Probe on every Node, assigned to
// every Node beyond its Roles.
func WithProbe() Option { return func(o *options) { o.probe = true } }

// World is a launched Scenario. The test's cleanup stops every Node.
type World struct {
	t        *testing.T
	scenario scenario.Scenario
	net      *fault.Network
	client   *http.Client
	opts     options
	nodes    []*Node
	claims   setup.Claims
}

// Node is one Node of a World.
type Node struct {
	w    *World
	spec scenario.Node
	addr netip.Addr
	host *fault.Host
	dir  node.DataDir

	running *node.Node

	mu      sync.Mutex
	half    node.Half
	probe   *roletest.Probe
	control *roletest.Control
}

// Launch starts every Node of s inside the caller's synctest bubble and
// sets them up as the process world does: it returns once every Node is
// ready and claimed, joined and holding its Roles.
func Launch(t *testing.T, s scenario.Scenario, opts ...Option) *World {
	t.Helper()
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	net := fault.NewNetwork()
	w := &World{
		t:        t,
		scenario: s,
		net:      net,
		client:   &http.Client{Transport: &http.Transport{DialContext: net.Host(netip.MustParseAddr("127.0.0.1")).DialContext, DisableKeepAlives: true}},
		opts:     options{version: Version},
	}
	for _, o := range opts {
		o(&w.opts)
	}
	t.Cleanup(w.stop)
	listen := listenKeys(t)
	for i, spec := range s.Nodes {
		n := &Node{w: w, spec: spec, addr: setup.Address(i)}
		n.host = net.Host(n.addr)
		n.dir = dataDir(t, s, i, listen)
		w.nodes = append(w.nodes, n)
		n.Start()
	}
	ready := make([]node.Ready, len(w.nodes))
	for i, n := range w.nodes {
		ready[i] = n.Ready()
	}
	var err error
	if w.claims, err = setup.Steps(t.Context(), w.client, s, ready); err != nil {
		t.Fatal(err)
	}
	return w
}

// listenKeys reads the listen keys from the configuration a Node resolves,
// as the process world probes the binary.
func listenKeys(t *testing.T) []string {
	t.Helper()
	probe, err := node.ParseDataDir(shortTempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := node.Resolve(probe, nil)
	if err != nil {
		t.Fatal(err)
	}
	return setup.ListenKeys(resolved)
}

func dataDir(t *testing.T, s scenario.Scenario, i int, listen []string) node.DataDir {
	t.Helper()
	dir, err := node.ParseDataDir(shortTempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	config, err := setup.Config(s, i, listen, node.Defaults().Endpoints)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir.Path(), "config.yaml"), config, 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func (w *World) stop() {
	for _, n := range slices.Backward(w.nodes) {
		if n.running != nil {
			n.Stop()
		}
	}
	if w.t.Failed() {
		for _, n := range w.nodes {
			n.logTail()
		}
	}
}

// Manifest is the world as the process world's manifest describes it,
// read from each Node's readiness now, so a restarted Node's new
// listeners show.
func (w *World) Manifest() world.Manifest {
	w.t.Helper()
	m := world.Manifest{
		Scenario: world.ScenarioRef{Name: w.scenario.Name, SHA256: w.scenario.Hash(), Seed: w.scenario.Seed},
		Binary:   world.Binary{Version: w.opts.version},
		Nodes:    []world.Node{},
		Fakes:    []world.Fake{},
		APIKey:   w.claims.APIKey,
	}
	if a := w.scenario.Admin; a != nil {
		m.Admin = &world.Admin{Email: a.Email, DisplayName: a.DisplayName, PIN: a.PIN}
	}
	for i, n := range w.nodes {
		entry, err := setup.Entry(setup.Ran{Spec: n.spec, Address: n.addr, Ready: n.Ready(), DataDir: n.dir.Path(), Log: n.logFile()}, w.claims.Members[i])
		if err != nil {
			w.t.Fatal(err)
		}
		m.Nodes = append(m.Nodes, entry)
	}
	return m
}

// API calls the world's Nodes as they run now.
func (w *World) API() *worldtest.API {
	return worldtest.NewAPI(w.t.Context(), w.t, w.client, w.Manifest())
}

// Node is the Node named name.
func (w *World) Node(name string) *Node {
	w.t.Helper()
	i := slices.IndexFunc(w.nodes, func(n *Node) bool { return n.spec.Name == name })
	if i < 0 {
		w.t.Fatalf("the world has no node %s", name)
	}
	return w.nodes[i]
}

// Path is the Switch of the path between the Nodes named a and b.
func (w *World) Path(a, b string) *fault.Switch {
	return w.net.Path(w.Node(a).addr, w.Node(b).addr)
}

// Client calls the world's Nodes from 127.0.0.1 on its network.
func (w *World) Client() *http.Client { return w.client }

// Start starts the Node on its data dir, after edit changes its Env.
func (n *Node) Start(edit ...func(*node.Env)) {
	t := n.w.t
	t.Helper()
	env := n.env()
	for _, e := range edit {
		e(&env)
	}
	running, err := node.Start(t.Context(), env)
	if err != nil {
		t.Fatalf("start node %s: %v", n.spec.Name, err)
	}
	n.running = running
}

// Join runs slipstream join on the stopped Node's data dir, as an
// operator pairs a Node with no service running.
func (n *Node) Join(ctx context.Context, code joincode.Code) (pairing.Joined, error) {
	return node.Join(ctx, n.env(), code)
}

// JoinOp sends the join op to the running Node over its control socket,
// as slipstream join does while a Node holds the data dir.
func (n *Node) JoinOp(ctx context.Context, code joincode.Code) (pairing.JoinReply, error) {
	return node.Call(ctx, n.host, n.dir, pairing.JoinOp, pairing.JoinRequest{Code: code.String()})
}

func (n *Node) env() node.Env {
	return node.Env{
		DataDir: n.dir,
		Environ: []string{},
		Stderr:  io.Discard,
		Version: n.w.opts.version,
		Net:     n.host,
		Engine:  enginetest.Open,
		OpenLog: node.OpenFileLog,
		Test:    n.test(),
	}
}

// test is what the world adds to the Node: a view of each half, and the
// Probe when the world runs it. Each start serves a fresh Probe control.
func (n *Node) test() node.Test {
	t := node.Test{Started: func(h node.Half) {
		n.mu.Lock()
		defer n.mu.Unlock()
		n.half = h
	}}
	if !n.w.opts.probe {
		return t
	}
	control := roletest.Serve()
	n.mu.Lock()
	n.control = control
	n.mu.Unlock()
	t.Library = []storage.Schema[storage.Library]{roletest.Schema()}
	t.Served = []role.Served{control}
	t.Always = []role.Name{roletest.Name}
	t.Held = func(local *storage.DB[storage.Local]) []role.Held {
		p := roletest.NewProbe(local)
		n.mu.Lock()
		defer n.mu.Unlock()
		n.probe = p
		return []role.Held{p.Held()}
	}
	return t
}

// Stop stops the Node and waits until it has, failing the test unless it
// stopped cleanly.
func (n *Node) Stop() {
	t := n.w.t
	t.Helper()
	n.running.Stop()
	<-n.running.Done()
	if err := n.running.Err(); err != nil {
		t.Errorf("node %s stopped with %v", n.spec.Name, err)
	}
	n.running = nil
}

func (n *Node) Ready() node.Ready { return n.running.Ready() }

func (n *Node) DataDir() node.DataDir { return n.dir }

// Half is the Node's half as it last started.
func (n *Node) Half() node.Half {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.half
}

// Probe is the Probe the Node's holder half last started with.
func (n *Node) Probe() *roletest.Probe {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.probe
}

// Control is the Probe's control half on a Controller.
func (n *Node) Control() *roletest.Control {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.control
}

func (n *Node) logFile() string { return filepath.Join(n.dir.LogsDir(), "slipstream.log") }

// Log is every record in the Node's log so far, one line each.
func (n *Node) Log() []string {
	t := n.w.t
	t.Helper()
	data, err := os.ReadFile(n.logFile())
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(string(bytes.TrimSuffix(data, []byte("\n"))), "\n")
}

func (n *Node) logTail() {
	lines := n.Log()
	lines = lines[max(0, len(lines)-logLinesOnFailure):]
	n.w.t.Logf("last %d lines of node %s's log:\n%s", len(lines), n.spec.Name, strings.Join(lines, "\n"))
}

// shortTempDir keeps a data dir's control socket path under macOS's 103-byte
// limit, which t.TempDir's path can exceed.
func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ss") //nolint:usetesting // t.TempDir's path is too long for the socket limit on macOS
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

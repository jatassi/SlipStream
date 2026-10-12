// Package world launches a Fake world: every fake in the world's list, and
// every Node of a Scenario on its own loopback address, set up as the
// Scenario says. Launch is the only launch path; fakeworld's supervisor and
// the end-to-end tier both call it.
package world

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/ca"
	"github.com/jatassi/slipstream/fakes/internal/nodeenv"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/internal/channel"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/fakes/world/internal/proc"
	"github.com/jatassi/slipstream/internal/node"
)

type Options struct {
	// Binary is the slipstream binary under test. The run copies it into
	// its run dir and runs the copy.
	Binary   string
	Scenario scenario.Scenario
	// RunDir keeps the run's evidence. It must not exist.
	RunDir string
	// RunDirCreated says the caller already made RunDir with CreateRunDir,
	// as fakeworld start does before it spawns its supervisor there.
	RunDirCreated bool
}

// Run is a launched world. It lives until Stop.
type Run struct {
	scenario   scenario.Scenario
	id         string
	runDir     string
	scratchDir string
	socket     string
	// life outlives the launch context: the run lasts until Stop.
	life    context.Context
	log     *slog.Logger
	logFile *os.File

	binary    Binary
	ca        *ca.CA
	caFile    string
	fakes     []*fakeRun
	endpoints node.EndpointsBlock
	nodes     []*nodeRun
	used      map[string]bool
	apiKey    string
	channel   *channel.Server
	manifest  Manifest

	stopOnce sync.Once
	report   StopReport
}

// nodeRun is one Node through the launch and the run.
type nodeRun struct {
	spec    scenario.Node
	addr    netip.Addr
	dataDir node.DataDir
	env     []string
	log     string
	logFile *os.File
	proc    *proc.Process
	ready   node.Ready
}

type fakeRun struct {
	spec   fake.Spec
	at     fake.Address
	cancel context.CancelFunc
	done   chan error
}

// unwind is a failed launch's undo steps, run newest first.
type unwind []func()

func (u *unwind) push(f func()) { *u = append(*u, f) }

func (u *unwind) run() {
	for _, f := range slices.Backward(*u) {
		f()
	}
}

// CreateRunDir makes a run dir and its bin/, refusing one that exists.
func CreateRunDir(dir string) error {
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return fmt.Errorf("run dir: %w", err)
	}
	if err := os.Mkdir(dir, 0o755); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("run dir %s already exists", dir)
		}
		return fmt.Errorf("run dir: %w", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "bin"), 0o755); err != nil {
		return fmt.Errorf("run dir: %w", err)
	}
	return nil
}

// Launch starts the world opts describes and returns once it is ready and
// its manifest is written. ctx bounds only the launch. A failed launch
// leaves no process running and writes no manifest; world.log and the
// Node logs stay as evidence.
func Launch(ctx context.Context, opts Options) (*Run, error) {
	return launch(ctx, opts, fakes)
}

func launch(ctx context.Context, opts Options, fakes []fake.Spec) (_ *Run, err error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	if err := opts.Scenario.Validate(); err != nil {
		return nil, err
	}
	runDir, err := filepath.Abs(opts.RunDir)
	if err != nil {
		return nil, fmt.Errorf("run dir: %w", err)
	}
	if !opts.RunDirCreated {
		if err := CreateRunDir(runDir); err != nil {
			return nil, err
		}
	}
	r := &Run{scenario: opts.Scenario, id: newRunID(), runDir: runDir, life: context.WithoutCancel(ctx), used: map[string]bool{}}
	scratchRoot := filepath.Join(os.TempDir(), "ssfw")
	r.scratchDir = filepath.Join(scratchRoot, r.id)
	r.socket = filepath.Join(scratchRoot, r.id+".sock")
	if r.log, r.logFile, err = openWorldLog(runDir); err != nil {
		return nil, err
	}
	r.log.Info(launchRecord, "run_id", r.id, "pid", os.Getpid(), "control", r.socket, "scratch_dir", r.scratchDir, "scenario", opts.Scenario.Name)

	var undo unwind
	defer func() {
		if err != nil {
			r.log.Error("launch failed", "err", err.Error())
			undo.run()
			_ = r.logFile.Close()
		}
	}()
	if err := os.WriteFile(filepath.Join(runDir, "scenario.yaml"), opts.Scenario.YAML(), 0o644); err != nil {
		return nil, fmt.Errorf("write scenario.yaml: %w", err)
	}
	if err := os.MkdirAll(scratchRoot, 0o700); err != nil {
		return nil, fmt.Errorf("scratch dir: %w", err)
	}
	if err := os.Mkdir(r.scratchDir, 0o700); err != nil {
		return nil, fmt.Errorf("scratch dir: %w", err)
	}
	undo.push(r.removeScratch)

	steps := []func(context.Context, *unwind) error{
		func(context.Context, *unwind) error { return r.copyBinary(opts.Binary) },
		r.checkAddresses,
		r.mintCA,
		func(ctx context.Context, undo *unwind) error { return r.startFakes(ctx, undo, fakes) },
		r.prepareNodes,
		r.startNodes,
		r.checkListeners,
		r.claim,
		r.publish,
	}
	for _, step := range steps {
		if err := step(ctx, &undo); err != nil {
			return nil, err
		}
	}
	r.log.Info("ready", "manifest", filepath.Join(runDir, manifestFile))
	return r, nil
}

// newRunID is 8 lower-case Crockford base32 characters.
func newRunID() string {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	return base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").WithPadding(base32.NoPadding).EncodeToString(b)
}

func (r *Run) Manifest() Manifest { return r.manifest }

func (r *Run) removeScratch() {
	if err := os.RemoveAll(r.scratchDir); err != nil {
		r.log.Error("remove scratch dir", "err", err.Error())
	}
}

func (r *Run) copyBinary(source string) error {
	src, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("binary: %w", err)
	}
	dst := filepath.Join(r.runDir, "bin", "slipstream"+filepath.Ext(src))
	sum, err := copyExecutable(src, dst)
	if err != nil {
		return err
	}
	r.binary = Binary{Source: src, Path: dst, SHA256: sum}
	r.log.Info("binary copied", "source", src, "path", dst, "sha256", sum)
	return nil
}

// copyExecutable copies src to dst and returns its hex SHA-256. It holds
// syscall.ForkLock while dst is open for writing: a child forked in that
// window would inherit the descriptor and make exec of dst fail with "text
// file busy" until it execs.
func copyExecutable(src, dst string) (string, error) {
	in, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("binary: %w", err)
	}
	defer func() { _ = in.Close() }()
	syscall.ForkLock.RLock()
	defer syscall.ForkLock.RUnlock()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755) //nolint:gosec // the run's copy of the binary must be executable
	if err != nil {
		return "", fmt.Errorf("binary: %w", err)
	}
	h := sha256.New()
	_, cerr := io.Copy(io.MultiWriter(out, h), in)
	if err := firstErr(cerr, out.Close()); err != nil {
		return "", fmt.Errorf("binary %s: %w", dst, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (r *Run) checkAddresses(ctx context.Context, _ *unwind) error {
	for i, n := range r.scenario.Nodes {
		addr, err := usableAddress(ctx, i)
		if err != nil {
			return err
		}
		r.nodes = append(r.nodes, &nodeRun{spec: n, addr: addr})
	}
	return nil
}

func (r *Run) mintCA(context.Context, *unwind) error {
	authority, err := ca.New(r.id)
	if err != nil {
		return err
	}
	r.ca = authority
	r.caFile = filepath.Join(r.runDir, "ca.pem")
	if err := os.WriteFile(r.caFile, authority.PEM(), 0o644); err != nil {
		return fmt.Errorf("write ca.pem: %w", err)
	}
	return nil
}

func (r *Run) startFakes(ctx context.Context, undo *unwind, specs []fake.Spec) error {
	for _, spec := range specs {
		f, err := r.startFake(ctx, spec)
		if err != nil {
			return err
		}
		r.fakes = append(r.fakes, f)
		undo.push(func() {
			if err := f.stop(); err != nil {
				r.log.Error("fake stopped", "fake", spec.Name, "err", err.Error())
			}
		})
	}
	r.endpoints = node.Defaults().Endpoints
	for _, f := range r.fakes {
		f.spec.Endpoints(&r.endpoints, f.at)
	}
	return nil
}

func (r *Run) startFake(ctx context.Context, spec fake.Spec) (*fakeRun, error) {
	server, err := spec.New(r.scenario, fake.NewEnv(r.scenario, spec.Name, r.log))
	if err != nil {
		return nil, fmt.Errorf("fake %s: %w", spec.Name, err)
	}
	l, err := new(net.ListenConfig).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("fake %s: %w", spec.Name, err)
	}
	at := fake.Address{AddrPort: netip.MustParseAddrPort(l.Addr().String()), TLS: spec.TLS}
	if spec.TLS {
		leaf, err := r.ca.Leaf("127.0.0.1", spec.Name+".fake.localhost")
		if err != nil {
			_ = l.Close()
			return nil, fmt.Errorf("fake %s: %w", spec.Name, err)
		}
		l = tls.NewListener(l, &tls.Config{Certificates: []tls.Certificate{leaf}, MinVersion: tls.VersionTLS12})
	}
	fctx, cancel := context.WithCancel(r.life)
	f := &fakeRun{spec: spec, at: at, cancel: cancel, done: make(chan error, 1)}
	go func() { f.done <- server.Serve(fctx, l) }()
	r.log.Info("fake started", "fake", spec.Name, "url", at.URL(""))
	return f, nil
}

func (f *fakeRun) stop() error {
	f.cancel()
	if err := <-f.done; err != nil {
		return fmt.Errorf("fake %s: %w", f.spec.Name, err)
	}
	return nil
}

func (r *Run) prepareNodes(ctx context.Context, _ *unwind) error {
	env := nodeenv.Environ(r.caFile)
	probe, err := r.resolve(ctx, "", filepath.Join(r.scratchDir, ".probe"), env)
	if err != nil {
		return err
	}
	listen := listenKeys(probe)
	for i, n := range r.nodes {
		name := n.spec.Name
		n.dataDir, err = node.ParseDataDir(filepath.Join(r.scratchDir, name))
		if err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		if err := os.Mkdir(n.dataDir.Path(), 0o700); err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		config, err := nodeConfig(r.scenario, i, listen, r.endpoints)
		if err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(n.dataDir.Path(), "config.yaml"), config, 0o600); err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		n.env = env
		resolved, err := r.resolve(ctx, name, n.dataDir.Path(), env)
		if err != nil {
			return err
		}
		if err := r.admit(name, resolved, nodeenv.World(r.caFile)); err != nil {
			return err
		}
	}
	return nil
}

// resolve runs slipstream config on dir, through S02's loader on exactly
// the file the Node will read.
func (r *Run) resolve(ctx context.Context, name, dir string, env []string) (node.Resolved, error) {
	res, err := proc.Run(ctx, r.binary.Path, []string{"config", "--data-dir", dir}, env)
	if err != nil {
		return node.Resolved{}, err
	}
	prefix := "node " + name + ": "
	if name == "" {
		prefix = "probe the binary's config: "
	}
	if res.Code == node.ExitUsage {
		return node.Resolved{}, errors.New(prefix + strings.TrimPrefix(strings.TrimSpace(string(res.Stderr)), "slipstream config: "))
	}
	if res.Code != node.ExitOK {
		return node.Resolved{}, fmt.Errorf("%sslipstream config exited with code %d: %s", prefix, res.Code, strings.TrimSpace(string(res.Stderr)))
	}
	var resolved node.Resolved
	if err := json.Unmarshal(res.Stdout, &resolved); err != nil {
		return node.Resolved{}, fmt.Errorf("%sslipstream config: %w", prefix, err)
	}
	return resolved, nil
}

// admit refuses a Node whose census names a real host or whose settings
// differ from production in a way no divergence names.
func (r *Run) admit(name string, resolved node.Resolved, worldEnv []string) error {
	if len(resolved.Census) > 0 {
		values := map[string]string{}
		for _, k := range resolved.Keys {
			values[k.Key] = k.Value
		}
		entries := make([]string, len(resolved.Census))
		for i, key := range resolved.Census {
			entries[i] = key + " (" + values[key] + ")"
		}
		return fmt.Errorf("node %s: endpoints still at real hosts: %s; add its fake to fakes/world", name, strings.Join(entries, ", "))
	}
	used, uncovered := diverge(settings(overrides(resolved.Keys), worldEnv))
	if len(uncovered) > 0 {
		lines := make([]string, len(uncovered))
		for i, s := range uncovered {
			lines[i] = "node " + name + " sets " + s + ", which the divergences list in fakes/world does not name"
		}
		return errors.New(strings.Join(lines, "\n"))
	}
	for id := range used {
		r.used[id] = true
	}
	return nil
}

// overrides are the keys whose value differs from the default and whose
// source is not random, the rule the readiness record uses.
func overrides(keys []node.Key) []node.Key {
	var out []node.Key
	for _, k := range keys {
		if k.Value != k.Default && k.Source != "random" {
			out = append(out, k)
		}
	}
	return out
}

func (r *Run) startNodes(ctx context.Context, undo *unwind) error {
	if err := os.Mkdir(filepath.Join(r.runDir, "nodes"), 0o755); err != nil {
		return fmt.Errorf("node logs: %w", err)
	}
	for _, n := range r.nodes {
		if err := r.startNode(ctx, undo, n); err != nil {
			return err
		}
	}
	return nil
}

func (r *Run) startNode(ctx context.Context, undo *unwind, n *nodeRun) error {
	name := n.spec.Name
	n.log = filepath.Join(r.runDir, "nodes", name+".log")
	var err error
	if n.logFile, err = os.Create(n.log); err != nil {
		return fmt.Errorf("node %s: %w", name, err)
	}
	ready := make(chan node.Ready, 1)
	found := false
	watch := func(line []byte) {
		if found {
			return
		}
		if rec, err := node.ParseReady(line); err == nil {
			found = true
			ready <- rec
		}
	}
	n.proc, err = proc.Start(ctx, r.binary.Path, []string{"run", "--data-dir", n.dataDir.Path()}, n.env, n.logFile, watch)
	if err != nil {
		_ = n.logFile.Close()
		return fmt.Errorf("node %s: %w", name, err)
	}
	r.log.Info(spawnRecord, "node", name, "pid", n.proc.PID, "data_dir", n.dataDir.Path())
	undo.push(func() { r.stopNode(n, false) })
	go func() {
		<-n.proc.Exited()
		r.log.Info("node exited", "node", name, "pid", n.proc.PID, "code", n.proc.ExitCode())
	}()
	select {
	case n.ready = <-ready:
		r.log.Info("node ready", "node", name, "pid", n.ready.PID)
		return nil
	case <-n.proc.Exited():
		return fmt.Errorf("node %s exited with code %d before it was ready\n%s", name, n.proc.ExitCode(), strings.Join(n.proc.Tail(), "\n"))
	case <-ctx.Done():
		return fmt.Errorf("node %s was not ready: %w", name, ctx.Err())
	}
}

func (r *Run) checkListeners(context.Context, *unwind) error {
	for _, n := range r.nodes {
		listeners, err := listenerMap(n.ready.Listeners)
		if err != nil {
			return fmt.Errorf("node %s: %w", n.spec.Name, err)
		}
		for _, name := range slices.Sorted(maps.Keys(listeners)) {
			if err := n.checkListener(name, listeners[name]); err != nil {
				return err
			}
		}
	}
	return nil
}

// listenerMap reads the readiness record's listeners as a JSON object, so
// a listener a later spec adds needs no world edit.
func listenerMap(l node.Listeners) (map[string]string, error) {
	data, err := json.Marshal(l)
	if err != nil {
		return nil, fmt.Errorf("listeners: %w", err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("listeners: %w", err)
	}
	return m, nil
}

func (n *nodeRun) checkListener(name, value string) error {
	if name == "control" {
		if rel, err := filepath.Rel(n.dataDir.Path(), value); err != nil || !filepath.IsLocal(rel) {
			return fmt.Errorf("node %s listener control is at %s, outside its data dir %s", n.spec.Name, value, n.dataDir)
		}
		return nil
	}
	if ap, err := netip.ParseAddrPort(value); err != nil || ap.Addr() != n.addr {
		return fmt.Errorf("node %s listener %s is on %s, not on its address %s", n.spec.Name, name, value, n.addr)
	}
	return nil
}

func (r *Run) claim(ctx context.Context, _ *unwind) error {
	admin := r.scenario.Admin
	for _, n := range r.nodes {
		if n.spec.Setup != scenario.SetupController {
			continue
		}
		key, err := nodeapi.ClaimController(ctx, n.url(), n.ready.SetupCode, nodeapi.Admin{Email: admin.Email, DisplayName: admin.DisplayName, PIN: admin.PIN})
		if err != nil {
			return fmt.Errorf("node %s: %w", n.spec.Name, err)
		}
		r.apiKey = key
		r.log.Info("node claimed", "node", n.spec.Name)
	}
	return nil
}

func (n *nodeRun) url() string { return "http://" + n.ready.Listeners.HTTP.String() }

func (r *Run) publish(ctx context.Context, undo *unwind) error {
	var err error
	if r.channel, err = channel.Serve(ctx, r.socket, r.stopAnswer); err != nil {
		return err
	}
	undo.push(r.channel.Close)
	r.manifest, err = r.buildManifest()
	if err != nil {
		return err
	}
	return writeAtomically(filepath.Join(r.runDir, manifestFile), marshal(r.manifest))
}

func (r *Run) buildManifest() (Manifest, error) {
	m := Manifest{
		Schema:      manifestSchema,
		RunID:       r.id,
		RunDir:      r.runDir,
		ScratchDir:  r.scratchDir,
		Scenario:    ScenarioRef{Name: r.scenario.Name, SHA256: r.scenario.Hash(), Seed: r.scenario.Seed},
		Binary:      r.binary,
		Supervisor:  Supervisor{PID: os.Getpid(), Log: filepath.Join(r.runDir, worldLogFile), Control: r.socket},
		CA:          CAFile{File: r.caFile},
		Divergences: usedIDs(r.used),
		Nodes:       []Node{},
		Fakes:       []Fake{},
		APIKey:      r.apiKey,
	}
	m.Binary.Version = r.nodes[0].ready.Version
	if a := r.scenario.Admin; a != nil {
		m.Admin = &Admin{Email: a.Email, DisplayName: a.DisplayName, PIN: a.PIN}
	}
	for _, f := range r.fakes {
		m.Fakes = append(m.Fakes, Fake{Name: f.spec.Name, URL: f.at.URL("")})
	}
	for _, n := range r.nodes {
		listeners, err := listenerMap(n.ready.Listeners)
		if err != nil {
			return Manifest{}, err
		}
		port := n.ready.Listeners.HTTP.Port()
		host := n.spec.Name + ".localhost"
		m.Nodes = append(m.Nodes, Node{
			Name:       n.spec.Name,
			Setup:      n.spec.Setup,
			Claim:      claimAfter[n.spec.Setup],
			Address:    n.addr.String(),
			Host:       host,
			URL:        n.url(),
			BrowserURL: fmt.Sprintf("http://%s:%d", host, port),
			Listeners:  listeners,
			DataDir:    n.dataDir.Path(),
			PID:        n.ready.PID,
			Seed:       n.ready.Seed,
			Log:        n.log,
			Overrides:  append([]node.Key{}, n.ready.Overrides...),
		})
	}
	return m, nil
}

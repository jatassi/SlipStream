// Package world launches a Fake world: every fake in the world's list, and
// every Node of a Scenario on its own loopback address, set up as the
// Scenario says. Launch is the only launch path; fakeworld's supervisor and
// the end-to-end tier both call it.
package world

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/ca"
	"github.com/jatassi/slipstream/fakes/internal/front"
	"github.com/jatassi/slipstream/fakes/internal/nodeenv"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/internal/seeders"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world/internal/control"
	"github.com/jatassi/slipstream/fakes/world/internal/lifecycle"
	"github.com/jatassi/slipstream/fakes/world/internal/proc"
	"github.com/jatassi/slipstream/fakes/world/internal/setup"
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
	Origin        Origin
}

// Run is a launched world. It lives until Stop.
type Run struct {
	scenario   scenario.Scenario
	id         string
	runDir     string
	scratchDir string
	socket     string
	life       context.Context
	log        *slog.Logger
	logFile    *os.File
	events     *events.Log

	binary    Binary
	ca        *ca.CA
	caFile    string
	probe     node.Resolved
	fakes     []*fakeRun
	faults    *fault.Plan
	fronts    *front.Set
	listeners []*front.Front
	table     fake.Fronts
	releases  *releases.Catalog
	seeders   *seeders.Seeders
	nodes     []*nodeRun
	used      map[string]bool
	claims    setup.Claims
	control   *control.Server

	// manifestMu guards the manifest and whether manifest.json holds it.
	manifestMu sync.Mutex
	manifest   Manifest
	published  bool

	// lastStep and openSteps change only under the event log's lock.
	lastStep  int
	openSteps map[int]bool

	stopOnce sync.Once
	report   StopReport
}

type nodeRun struct {
	spec    scenario.Node
	addr    netip.Addr
	dataDir node.DataDir
	env     []string
	log     string
	life    *lifecycle.Node
}

// fakeRun is a fake the world hosts: the address of its world front, and
// the inbox each of its fronts pushes connections into.
type fakeRun struct {
	spec   fake.Spec
	env    fake.Env
	at     fake.Address
	inbox  *front.Inbox
	cancel context.CancelFunc
	done   chan error
}

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
// leaves no process running and writes no manifest; the event log, ending
// in a run.stopped that names the failure, world.log and the Node logs
// stay as evidence.
func Launch(ctx context.Context, opts Options) (*Run, error) {
	return launch(ctx, opts, fakes)
}

func launch(ctx context.Context, opts Options, fakes []fake.Spec) (_ *Run, err error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w", err)
	}
	if err := checkOptions(opts, fakes); err != nil {
		return nil, err
	}
	addrs, err := CheckAddresses(ctx, len(opts.Scenario.Nodes))
	if err != nil {
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
	r := &Run{scenario: opts.Scenario, id: newRunID(), runDir: runDir, life: context.WithoutCancel(ctx), used: map[string]bool{}, openSteps: map[int]bool{}}
	for i, n := range opts.Scenario.Nodes {
		r.nodes = append(r.nodes, &nodeRun{spec: n, addr: addrs[i]})
	}
	scratchRoot := filepath.Join(os.TempDir(), "ssfw")
	r.scratchDir = filepath.Join(scratchRoot, r.id)
	r.socket = filepath.Join(scratchRoot, r.id+".sock")
	if r.log, r.logFile, err = openWorldLog(runDir); err != nil {
		return nil, err
	}
	if r.events, err = events.Create(runDir, r.observe); err != nil {
		_ = r.logFile.Close()
		return nil, err
	}
	r.record(events.RunLaunched, events.World, launched{RunID: r.id, PID: os.Getpid(), Control: r.socket, ScratchDir: r.scratchDir, Scenario: opts.Scenario.Name, Origin: opts.Origin})

	var undo unwind
	defer func() {
		if err != nil {
			undo.run()
			r.record(events.RunStopped, events.World, stopped{Error: err.Error()})
			_ = r.events.Close()
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
		r.mintCA,
		r.probeBinary,
		func(ctx context.Context, undo *unwind) error { return r.bindFronts(ctx, undo, fakes) },
		r.buildReleases,
		r.serveFakes,
		r.setFaults,
		r.startSeeders,
		r.prepareNodes,
		r.startNodes,
		r.checkListeners,
		r.pinPorts,
		r.setUp,
		r.seedLibrary,
		r.publish,
	}
	for _, step := range steps {
		if err := step(ctx, &undo); err != nil {
			return nil, err
		}
	}
	r.record(events.RunReady, events.World, struct {
		Manifest string `json:"manifest"`
	}{manifestFile})
	return r, nil
}

func checkOptions(opts Options, fakes []fake.Spec) error {
	if err := opts.Scenario.Validate(); err != nil {
		return err
	}
	if opts.Origin.Kind != FromSteps && opts.Origin.Kind != FromTest {
		return fmt.Errorf("launch %s: the options name no origin", opts.Scenario.Name)
	}
	for _, spec := range fakes {
		if err := spec.Check(); err != nil {
			return err
		}
	}
	return nil
}

func newRunID() string {
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	return base32.NewEncoding("0123456789abcdefghjkmnpqrstvwxyz").WithPadding(base32.NoPadding).EncodeToString(b)
}

func (r *Run) Manifest() Manifest {
	r.manifestMu.Lock()
	defer r.manifestMu.Unlock()
	return r.manifest
}

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
	sum, err := CopyExecutable(src, dst)
	if err != nil {
		return err
	}
	r.binary = Binary{Source: src, Path: dst, SHA256: sum}
	r.log.Info("binary copied", "source", src, "path", dst, "sha256", sum)
	for _, tool := range []string{"ffmpeg", "ffprobe"} {
		name := tool + filepath.Ext(src)
		if _, err := CopyExecutable(filepath.Join(filepath.Dir(src), name), filepath.Join(r.runDir, "bin", name)); err != nil {
			return fmt.Errorf("%s beside the binary: %w", tool, err)
		}
	}
	return nil
}

// CopyExecutable copies src to dst and returns its hex SHA-256. It holds
// syscall.ForkLock while dst is open for writing: a child forked in that
// window would inherit the descriptor and make exec of dst fail with "text
// file busy" until it execs.
func CopyExecutable(src, dst string) (string, error) {
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

func (r *Run) probeBinary(ctx context.Context, _ *unwind) error {
	var err error
	if r.probe, err = r.resolve(ctx, "", filepath.Join(r.scratchDir, ".probe"), nodeenv.Environ(r.caFile)); err != nil {
		return err
	}
	for _, id := range r.scenario.ContentTypes() {
		if !slices.Contains(r.probe.ContentTypes, id) {
			return fmt.Errorf("scenario %s needs Content Type %s, which %s does not register; launch it with worldtest.LaunchE2E", r.scenario.Name, id, filepath.Base(r.binary.Source))
		}
	}
	return nil
}

func (r *Run) buildReleases(ctx context.Context, _ *unwind) error {
	cache, err := releases.CacheDir()
	if err != nil {
		return err
	}
	cfg := releases.Config{
		FFmpeg: filepath.Join(r.runDir, "bin", "ffmpeg"+filepath.Ext(r.binary.Path)),
		Cache:  cache,
		Dir:    filepath.Join(r.scratchDir, "releases"),
	}
	if len(r.scenario.Releases) > 0 {
		i := slices.IndexFunc(r.fakes, func(f *fakeRun) bool { return f.spec.Name == tracker.Fake.Name })
		if i < 0 {
			return fmt.Errorf("scenario %s lists releases, and the world runs no fake-tracker", r.scenario.Name)
		}
		cfg.Tracker = r.fakes[i].at.AddrPort
	}
	if r.releases, err = releases.Build(ctx, r.scenario, cfg); err != nil {
		return err
	}
	for _, rel := range r.releases.Releases {
		r.log.Info("release built", "release", rel.Name, "info_hash", rel.InfoHash.String(), "size", rel.Size)
	}
	return nil
}

func (r *Run) serveFakes(_ context.Context, undo *unwind) error {
	for _, f := range r.fakes {
		f.env = fake.NewEnv(r.scenario, f.spec, r.log, r.faults, r.events)
		f.env.Releases, f.env.Fronts, f.env.Files = r.releases, r.table, r.runDir
		server, err := f.spec.New(r.scenario, f.env)
		if err != nil {
			return fmt.Errorf("fake %s: %w", f.spec.Name, err)
		}
		if err := r.serveFronts(f); err != nil {
			return err
		}
		fctx, cancel := context.WithCancel(r.life)
		f.cancel, f.done = cancel, make(chan error, 1)
		go func() { f.done <- server.Serve(fctx, f.inbox) }()
		r.log.Info("fake started", "fake", f.spec.Name, "url", f.at.URL(""))
		undo.push(func() {
			if err := f.stop(); err != nil {
				r.log.Error("fake stopped", "fake", f.spec.Name, "err", err.Error())
			}
		})
	}
	return nil
}

// startSeeders seeds each seeded release once fake-tracker serves, and
// returns once the tracker heard each seeder start.
func (r *Run) startSeeders(ctx context.Context, undo *unwind) error {
	var err error
	if r.seeders, err = seeders.Start(ctx, r.releases, r.caFile); err != nil {
		return err
	}
	undo.push(func() {
		if err := r.seeders.Close(); err != nil {
			r.log.Error("seeders stopped", "err", err.Error())
		}
	})
	r.serveSeederFronts()
	return nil
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
	listen := setup.ListenKeys(r.probe)
	for i, n := range r.nodes {
		name := n.spec.Name
		var err error
		n.dataDir, err = node.ParseDataDir(filepath.Join(r.scratchDir, name))
		if err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		if err := os.Mkdir(n.dataDir.Path(), 0o700); err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		endpoints, err := r.endpoints(name)
		if err != nil {
			return err
		}
		config, err := setup.Config(r.scenario, i, listen, r.linkFront(name).Addr(), endpoints)
		if err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(n.dataDir.Path(), "config.yaml"), config, 0o600); err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		n.env = env
		n.log = filepath.Join(r.runDir, "nodes", name+".log")
		n.life = lifecycle.New(lifecycle.Config{Name: name, Binary: r.binary.Path, DataDir: n.dataDir, Env: env, Log: n.log, Record: r.recordFor, Exited: func() { r.nodeExited(i) }})
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
	r.serveLinkFronts()
	for _, n := range r.nodes {
		if err := r.startNode(ctx, undo, n); err != nil {
			return err
		}
	}
	return nil
}

func (r *Run) startNode(ctx context.Context, undo *unwind, n *nodeRun) error {
	undo.push(func() { n.life.End(r.life) })
	return n.life.Start(ctx)
}

func (r *Run) checkListeners(context.Context, *unwind) error {
	for _, n := range r.nodes {
		listeners, err := setup.Listeners(n.life.Ready().Listeners)
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

// pinPorts rewrites each Node's config.yaml with every listen key at the
// port it bound, so a Node started again returns on its first address and
// ports, then runs slipstream config on it again as the check.
func (r *Run) pinPorts(ctx context.Context, _ *unwind) error {
	for _, n := range r.nodes {
		name := n.spec.Name
		path := filepath.Join(n.dataDir.Path(), "config.yaml")
		config, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		listeners, err := setup.Listeners(n.life.Ready().Listeners)
		if err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		if config, err = setup.Pin(config, listeners); err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		if err := os.WriteFile(path, config, 0o600); err != nil {
			return fmt.Errorf("node %s: %w", name, err)
		}
		resolved, err := r.resolve(ctx, name, n.dataDir.Path(), n.env)
		if err != nil {
			return err
		}
		if err := r.admit(name, resolved, nodeenv.World(r.caFile)); err != nil {
			return err
		}
	}
	return nil
}

// setUp runs the setup steps over the OS network.
func (r *Run) setUp(ctx context.Context, _ *unwind) error {
	var err error
	r.claims, err = setup.Steps(ctx, http.DefaultClient, r.scenario, r.ready())
	if err != nil {
		return err
	}
	r.log.Info("nodes set up")
	return nil
}

// seedLibrary creates the Scenario's library on the Controller, its roots'
// folders under the media dir and its Indexers at the site fronts of the
// Node that holds Searching, which searches from them.
func (r *Run) seedLibrary(ctx context.Context, _ *unwind) error {
	if r.scenario.Library == nil {
		return nil
	}
	sites := map[string]fake.Address{}
	for _, f := range r.fakes {
		sites[f.spec.Name], _ = r.table.Find(fault.Target(f.spec.Name), r.scenario.Searching())
	}
	if err := setup.Library(ctx, http.DefaultClient, r.scenario, r.ready(), r.claims.APIKey, r.mediaDir(), sites); err != nil {
		return err
	}
	r.log.Info("library seeded")
	return nil
}

func (r *Run) ready() []node.Ready {
	ready := make([]node.Ready, len(r.nodes))
	for i, n := range r.nodes {
		ready[i] = n.life.Ready()
	}
	return ready
}

func (r *Run) mediaDir() string { return filepath.Join(r.scratchDir, "media") }

func (r *Run) publish(ctx context.Context, undo *unwind) error {
	var err error
	if r.control, err = control.Serve(ctx, r.socket, controlHandler(r)); err != nil {
		return err
	}
	undo.push(r.control.Close)
	r.manifestMu.Lock()
	defer r.manifestMu.Unlock()
	r.manifest, err = r.buildManifest()
	if err != nil {
		return err
	}
	if err := writeAtomically(filepath.Join(r.runDir, manifestFile), marshal(r.manifest)); err != nil {
		return err
	}
	r.published = true
	return nil
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
		Events:      events.File,
		Divergences: usedIDs(r.used),
		Nodes:       []Node{},
		Fakes:       []Fake{},
		Fronts:      []Front{},
		Links:       map[string]string{},
		Releases:    []Release{},
		APIKey:      r.claims.APIKey,
	}
	if r.scenario.Library != nil {
		m.MediaDir = r.mediaDir()
	}
	m.Binary.Version = r.nodes[0].life.Ready().Version
	if a := r.scenario.Admin; a != nil {
		m.Admin = &Admin{Email: a.Email, DisplayName: a.DisplayName, PIN: a.PIN}
	}
	for _, f := range r.fakes {
		m.Fakes = append(m.Fakes, Fake{Name: f.spec.Name, URL: f.at.URL(""), Captures: append([]fake.CaptureSpec{}, f.spec.Captures...), Cues: append([]fake.CueSpec{}, f.spec.Cues...)})
	}
	for _, f := range r.table {
		entry := Front{Target: string(f.Target), Node: f.Node, Address: f.Address.AddrPort.String()}
		if slices.ContainsFunc(r.fakes, func(fr *fakeRun) bool { return fault.Target(fr.spec.Name) == f.Target }) {
			entry.URL = f.Address.URL("")
		}
		m.Fronts = append(m.Fronts, entry)
	}
	for _, n := range r.nodes {
		m.Links[n.spec.Name] = r.linkFront(n.spec.Name).Addr().String()
	}
	for _, rel := range r.releases.Releases {
		m.Releases = append(m.Releases, Release{Name: rel.Name, InfoHash: rel.InfoHash.String(), Size: rel.Size, BLAKE3: hex.EncodeToString(rel.BLAKE3[:])})
	}
	for i, n := range r.nodes {
		entry, err := setup.Entry(setup.Ran{Spec: n.spec, Address: n.addr, State: n.life.State(), Ready: n.life.Ready(), DataDir: n.dataDir.Path(), Log: n.log}, r.claims.Members[i])
		if err != nil {
			return Manifest{}, err
		}
		m.Nodes = append(m.Nodes, entry)
	}
	return m, nil
}

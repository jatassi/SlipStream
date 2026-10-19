package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	// The Maintenance window names its zone where the OS has no zone
	// database: Windows and a container with TZ set.
	_ "time/tzdata"

	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/node/internal/config"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/internal/host"
	"github.com/jatassi/slipstream/internal/node/internal/lock"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/serving"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

// Env is what a Node runs on. A process passes the OS network, libtorrent
// and a rotating log; the simulation tier passes an in-memory network, an
// in-memory engine and a plain log file, which a testing/synctest bubble
// can hold.
type Env struct {
	DataDir DataDir
	// Install is the kind the marker beside the executable states.
	Install install.Kind
	// Mode is the subcommand running the Node, which status reports.
	Mode    Mode
	Environ []string
	Stderr  io.Writer
	Version string
	// Net serves every listener, the control socket's included, and every
	// link dial.
	Net    link.Network
	Engine func(engine.Config) (engine.Engine, error)
	// OpenLog opens the log in dir with s, mirrored to stderr.
	OpenLog func(dir string, stderr io.Writer, s logs.Settings) (*logs.Log, error)
	Types   contenttype.Set
	// Media probes video files: the bundled ffprobe for a process, which
	// Run finds beside the executable.
	Media media.Prober
	// Disk opens every filesystem call the Storage Role makes, with data
	// the folder no root may overlap: the OS's own for a process.
	Disk func(data string) (storing.Disk, error)
	// Restart starts the process that follows this one once it has stopped
	// and released the data dir: an exec in place on Unix, a new process on
	// Windows. Nil for a Node that cannot restart itself.
	Restart func() error
	// RestartOf is the PID this process follows after a restart, which
	// may still hold the data dir; 0 otherwise.
	RestartOf int
	// Zone is where the Maintenance window's wall-clock times are kept;
	// nil is the process's own, time.Local.
	Zone *time.Location
	// Test adds test-only Roles; a process leaves it zero.
	Test Test
}

// ProcessEnv is this process's Env for the subcommand cmd. The install
// kind comes from the marker beside the symlink-resolved executable, and
// the data dir from dataDirFlag, else SLIPSTREAM_DATA_DIR, else the kind's
// default.
func ProcessEnv(cmd, dataDirFlag string) (Env, error) {
	environ, follows, err := restartOf(os.Environ())
	if err != nil {
		return Env{}, &UsageError{Err: err}
	}
	layout, err := processLayout(environ)
	if err != nil {
		return Env{}, err
	}
	kind, err := install.Read(layout)
	if err != nil {
		return Env{}, &UsageError{Err: err}
	}
	raw := dataDirFlag
	if raw == "" {
		if i := slices.IndexFunc(environ, func(kv string) bool { return strings.HasPrefix(kv, config.DataDirVar+"=") }); i >= 0 {
			raw = strings.TrimPrefix(environ[i], config.DataDirVar+"=")
		}
	}
	if raw == "" {
		if raw, err = kind.DefaultDataDir(layout); errors.Is(err, install.ErrNoDefault) {
			return Env{}, &UsageError{Err: fmt.Errorf("no data dir: pass --data-dir or set %s", config.DataDirVar)}
		}
		if err != nil {
			return Env{}, &UsageError{Err: fmt.Errorf("no default data dir for a %s install: %w", kind, err)}
		}
	}
	dir, err := ParseDataDir(raw)
	if err != nil {
		return Env{}, &UsageError{Err: err}
	}
	if err := checkHost(dir, kind, cmd); err != nil {
		return Env{}, &UsageError{Err: err}
	}
	version := "dev"
	if s, ok := buildinfo.Current(); ok {
		version = s.Version
	}
	return Env{
		DataDir:   dir,
		Install:   kind,
		Environ:   environ,
		Stderr:    os.Stderr,
		Version:   version,
		Net:       osNetwork{},
		Engine:    func(c engine.Config) (engine.Engine, error) { return libtorrent.Open(c) },
		OpenLog:   logs.Open,
		Disk:      OSDisk,
		Restart:   processRestart(layout.Executable, os.Args, environ, dir.LogsDir(), os.Stderr),
		RestartOf: follows,
	}, nil
}

func processLayout(environ []string) (install.Layout, error) {
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
	}
	if err != nil {
		return install.Layout{}, fmt.Errorf("find the executable: %w", err)
	}
	return install.Layout{GOOS: runtime.GOOS, Executable: exe, Environ: environ}, nil
}

func checkHost(dir DataDir, kind install.Kind, cmd string) error {
	owner, err := host.ForeignOwner(dir.Path())
	if err != nil {
		return err
	}
	if owner != nil {
		hint := fmt.Sprintf("sudo -u %s slipstream %s ...", owner.Name, cmd)
		if kind == install.Docker {
			hint = fmt.Sprintf("docker exec -u %d <container> slipstream %s ...", owner.UID, cmd)
		}
		return fmt.Errorf("data dir %s belongs to %s; run: %s", dir, owner.Name, hint)
	}
	remote, err := host.OnNetworkShare(dir.Path())
	if err != nil {
		return err
	}
	if remote {
		return fmt.Errorf("data dir %s is on a network share; SQLite's WAL needs a local disk", dir)
	}
	return nil
}

// Node is one running SlipStream process's worth of state. Several can run in
// one Go process; they share nothing.
type Node struct {
	env      Env
	started  time.Time
	logs     *logs.Log
	rootLog  *slog.Logger
	log      *slog.Logger
	stack    *Stack
	deadline time.Duration
	ready    Ready

	server  *control.Server
	isReady atomic.Bool

	// Set during start, before the Node is ready. cfgMu guards the Node
	// settings table's keys in cfg, which setConfig changes later.
	cfgMu     sync.Mutex
	cfg       Config
	key       link.PrivateKey
	info      link.NodeInfo
	hub       *live.Hub
	gate      *identity.Gate
	link      *link.Link
	linkBound netip.AddrPort
	// served is the Controller's end of the Serve streams Serving Workers
	// open.
	served    *serving.Listener
	pairing   *pairing.Pairing
	lockstep  *http.ServeMux
	roleRand  *rand.Rand
	grabsRand *rand.Rand
	filesRand *rand.Rand
	tasksRand *rand.Rand
	// systemRand draws the Controller's waits on an offline Node's log.
	systemRand *rand.Rand
	// host is what NodeSystemService serves for this Node, and hostMu
	// orders its reports to the holder half.
	host   *nodeHost
	hostMu sync.Mutex
	// torrent is the Node's torrent engine, and downloads its built-in
	// client.
	torrent   engine.Engine
	downloads *downloading.Client
	store     *storing.Store
	// claimed is where library.db and a half push their stops, so they
	// stop after HTTP and the link, whenever they start.
	claimed *Stack

	libMu sync.Mutex
	lib   atomic.Pointer[library]
	half  atomic.Pointer[half]
	// claimMu serializes claims, so a Controller claim and a Join never
	// interleave.
	claimMu sync.Mutex

	stopOnce sync.Once
	stopping chan struct{}
	// restart is whether the stop is a restart's, set once with stopping.
	restart bool
	// drained closes once the Stack has unwound, released once the lock is
	// released, and done once every control connection has finished.
	drained  chan struct{}
	released chan struct{}
	done     chan struct{}
	err      error
}

// Start runs the whole startup and returns once the Node is ready. It
// installs no signal handler and pins no thread. Canceling ctx stops the
// Node, before or after it is ready.
func Start(ctx context.Context, env Env) (*Node, error) {
	n := &Node{
		env:      env,
		started:  time.Now(),
		deadline: Defaults(env.Types, env.Install).Timings.Node.StopDeadline,
		stopping: make(chan struct{}),
		drained:  make(chan struct{}),
		released: make(chan struct{}),
		done:     make(chan struct{}),
	}
	if err := n.start(n.canceledAfterStopping(ctx)); err != nil {
		n.Stop()
		n.finish(err)
		return nil, err
	}
	go n.wait()
	return n, nil
}

func (n *Node) canceledAfterStopping(ctx context.Context) context.Context {
	stepCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	go func() {
		select {
		case <-ctx.Done():
			n.Stop()
		case <-n.stopping:
		}
		cancel()
	}()
	return stepCtx
}

func (n *Node) Ready() Ready { return n.ready }

// Done is closed once the Node has fully stopped.
func (n *Node) Done() <-chan struct{} { return n.done }

// Err is why the Node stopped: nil for a clean stop. It is valid once Done is closed.
func (n *Node) Err() error { return n.err }

// Stop asks the Node to stop. It is idempotent and safe from any goroutine.
func (n *Node) Stop() { n.stop(false) }

// start runs the startup order. Errors before the log opens go to stderr as
// plain text, later ones to the log.
func (n *Node) start(ctx context.Context) error {
	lk, err := n.acquireLock()
	if held, ok := errors.AsType[*lock.HeldError](err); ok {
		err = &InUseError{Dir: n.env.DataDir, PID: held.PID}
	}
	if err != nil {
		_, _ = fmt.Fprintln(n.env.Stderr, err)
		return err
	}
	log, err := n.env.OpenLog(n.env.DataDir.LogsDir(), n.env.Stderr, Defaults(n.env.Types, n.env.Install).Log)
	if err != nil {
		_, _ = fmt.Fprintln(n.env.Stderr, err)
		return errors.Join(err, lk.Release())
	}
	n.logs = log
	n.rootLog = log.Logger
	n.log = n.rootLog.With("subsystem", "node")
	n.stack = newStack(n.log)
	n.stack.OnStop("lock", func(context.Context) error {
		if err := lk.Release(); err != nil {
			return err
		}
		close(n.released)
		return nil
	})
	n.stack.OnStop("log", func(context.Context) error {
		n.log.Info("stopped")
		return log.Close()
	})
	if err := n.startLogged(ctx); err != nil {
		if !errors.Is(err, ErrStopped) {
			n.log.Error("start failed", "err", err.Error())
		}
		return errors.Join(err, n.stack.unwind(n.deadline))
	}
	return nil
}

func (n *Node) startLogged(ctx context.Context) error {
	if err := logs.RemoveArchives(n.env.DataDir.RunDir()); err != nil {
		return err
	}
	cfg, keys, err := LoadConfig(n.env.DataDir, n.env.Environ, n.env.Types, n.env.Install)
	if err != nil {
		return err
	}
	if err := n.logs.Apply(cfg.Log); err != nil {
		return err
	}
	n.deadline = cfg.Timings.Node.StopDeadline
	if changed := changedTimings(keys); len(changed) > 0 {
		n.log.Warn("timings differ from their defaults", "keys", changed)
	}
	var l Listeners
	if n.server, err = control.Listen(n.env.Net.Listen, n.env.DataDir.ControlSocket(), n.dispatch); err != nil {
		return err
	}
	n.stack.OnStop("control", func(context.Context) error { return n.server.Close() })
	l.Control = n.env.DataDir.ControlSocket()
	if err := n.checkpoint(); err != nil {
		return err
	}
	claim, setupCode, err := n.startSubsystems(ctx, cfg, &l)
	if err != nil {
		return err
	}
	if err := n.checkpoint(); err != nil {
		return err
	}
	if unbound := l.unbound(); len(unbound) > 0 {
		return fmt.Errorf("listeners without a bound address: %s", strings.Join(unbound, ", "))
	}
	n.ready = Ready{
		PID:       os.Getpid(),
		DataDir:   n.env.DataDir.Path(),
		Version:   n.env.Version,
		Seed:      cfg.Seed,
		Claim:     claim,
		NodeKey:   n.key.Public().Fingerprint(),
		Listeners: l,
		Overrides: overrides(keys),
		SetupCode: setupCode,
	}
	if setupCode != "" {
		n.log.Info("setup page", "url", setupURL(l.HTTP, setupCode))
	}
	if n.env.Install.SelfInstalled() {
		for _, msg := range host.BelowFloor(ctx) {
			n.log.Warn("host below the supported floor", "reason", msg)
		}
	}
	n.isReady.Store(true)
	n.log.Info("ready", "ready", n.ready)
	return nil
}

// startSubsystems is the start order. Each subsystem pushes its stop onto the
// Stack, so they stop in reverse. HTTP starts last, once every other
// listener is bound, since getSystem reports them all. It returns the
// claim and, while the Node is Unclaimed, the Setup code.
func (n *Node) startSubsystems(ctx context.Context, cfg Config, l *Listeners) (system.Claim, string, error) {
	n.cfg = cfg
	key, err := openKey(n.env.DataDir)
	if err != nil {
		return 0, "", err
	}
	n.key = key
	streams := NewStreams(cfg.Seed)
	n.roleRand = streams.Stream("role")
	n.hub = live.NewHub(surface.Events)
	n.grabsRand = streams.Stream("grabs")
	n.filesRand = streams.Stream("files")
	n.systemRand = streams.Stream("system")
	n.host = &nodeHost{n: n}
	n.tasksRand = streams.Stream("tasks")
	if err := n.startTorrent(cfg, l, streams.Stream("torrent")); err != nil {
		return 0, "", err
	}
	if err := n.startStorage(cfg); err != nil {
		return 0, "", err
	}
	// The claimed half stops before the torrent engine closes under it.
	n.claimed = n.stack.Sub("claimed")
	if _, err := os.Stat(n.env.DataDir.LibraryDB()); err == nil {
		if _, err := n.openLibrary(ctx); err != nil {
			return 0, "", n.stoppedOr(err)
		}
	}
	claim, err := n.claim(ctx)
	if err != nil {
		return 0, "", err
	}
	var setupCode string
	if n.gate, setupCode, err = identity.NewGate(ctx, n.claim, n.rootLog.With("subsystem", "identity")); err != nil {
		return 0, "", err
	}
	if err := n.startLink(cfg, streams, l); err != nil {
		return 0, "", err
	}
	n.pairing, err = pairing.Open(pairing.Config{
		DataDir:   n.env.DataDir.Path(),
		Key:       n.key.Public(),
		Link:      n.link,
		Name:      cfg.Name,
		Claims:    surfaceNode{n},
		AwaitDial: cfg.Timings.Link.AwaitDial,
		Removed:   n.removed,
		Log:       n.rootLog.With("subsystem", "pairing"),
	})
	if err != nil {
		return 0, "", refusedRecord(err)
	}
	switch claim {
	case system.Controller:
		err = n.startController(ctx)
	case system.Worker:
		// A join.code that re-pairs starts the half once its Claim ends.
		if !n.pairing.Repairs(cfg.Join.Code) {
			err = n.startWorker(ctx)
		}
	case system.Unclaimed:
		if r, ok := n.pairing.Removal(); ok {
			n.log.Warn(r.Sentence())
		}
	}
	if err != nil {
		return 0, "", n.stoppedOr(err)
	}
	if code := cfg.Join.Code; !code.IsZero() {
		n.stack.Go("join-code", func(ctx context.Context) error {
			n.joinAtStart(ctx, code)
			return nil
		})
	}
	return claim, setupCode, n.startHTTP(ctx, cfg, l, surface.Services{Node: surfaceNode{n}, Hub: n.hub, Live: cfg.Timings.Live}, n.gate)
}

// startTorrent opens the torrent engine and the Node's built-in client,
// which reads its events from now on, claimed or not.
func (n *Node) startTorrent(cfg Config, l *Listeners, ports *rand.Rand) error {
	torrent, err := n.env.Engine(engine.Config{ListenInterface: cfg.Torrent.Listen, OutgoingUTP: cfg.Torrent.OutgoingUTP, Rand: ports})
	if err != nil {
		return fmt.Errorf("open torrent engine on %s: %w", cfg.Torrent.Listen, err)
	}
	n.torrent = torrent
	n.downloads = downloading.NewClient(downloading.Config{
		Engine:  torrent,
		Dir:     n.env.DataDir.Resolve(cfg.Torrent.Downloads),
		Timings: cfg.Timings.Downloading,
		Log:     n.rootLog.With("subsystem", "downloading"),
	})
	n.stack.Go("torrent events", n.downloads.Drain)
	n.stack.OnStop("torrent", func(context.Context) error { return torrent.Close() })
	if l.Torrent, err = torrent.ListenAddr(); err != nil {
		return fmt.Errorf("torrent listen address: %w", err)
	}
	return nil
}

func OSDisk(data string) (storing.Disk, error) { return storing.NewOSDisk(data) }

// startStorage opens the Node's Storage over its Disk, claimed or not.
func (n *Node) startStorage(cfg Config) error {
	disk, err := n.env.Disk(n.env.DataDir.Path())
	if err != nil {
		return fmt.Errorf("open the disk: %w", err)
	}
	n.store = storing.NewStore(storing.Config{Disk: disk, Payloads: n.downloads, Downloads: n.downloads.Folders, Timings: cfg.Timings.Storing})
	return nil
}

// checkpoint ends the startup early once the Node is asked to stop or fails.
func (n *Node) checkpoint() error {
	select {
	case <-n.stack.Failed():
		return n.stack.Err()
	case <-n.stopping:
		return ErrStopped
	default:
		return nil
	}
}

func (n *Node) wait() {
	select {
	case <-n.stopping:
	case <-n.stack.Failed():
		n.Stop()
	}
	n.finish(errors.Join(n.stack.Err(), n.stack.unwind(n.deadline)))
}

// finish lets held stop requests see the outcome before Done closes, so a
// process that exits on Done has already answered them. Then a restart's
// successor starts.
func (n *Node) finish(err error) {
	close(n.drained)
	if n.server != nil {
		n.server.Wait()
	}
	n.err = n.handoff(err)
	close(n.done)
}

func changedTimings(keys []Key) []Key {
	var changed []Key
	for _, k := range overrides(keys) {
		if strings.HasPrefix(k.Key, "timings.") {
			changed = append(changed, k)
		}
	}
	return changed
}

var ErrInterrupted = errors.New("interrupted again while stopping")

// bundledProber is the ffprobe the release ships beside the executable,
// with ffmpeg beside it too (ADR 0015).
func bundledProber() (media.FFprobe, error) {
	exe, err := os.Executable()
	if err != nil {
		return media.FFprobe{}, fmt.Errorf("find the executable: %w", err)
	}
	beside := func(tool string) string { return filepath.Join(filepath.Dir(exe), tool+filepath.Ext(exe)) }
	for _, tool := range []string{"ffprobe", "ffmpeg"} {
		if info, err := os.Stat(beside(tool)); err != nil || !info.Mode().IsRegular() {
			return media.FFprobe{}, fmt.Errorf("%s not found beside %s", tool, exe)
		}
	}
	return media.FFprobe{Path: beside("ffprobe")}, nil
}

// Run is Start plus signals: SIGINT or SIGTERM stops the Node, and a second
// one returns ErrInterrupted at once. Losing the reader of stderr does not
// stop it.
func Run(ctx context.Context, env Env) error {
	return serve(ctx, env, func(*Node, <-chan struct{}) {})
}

func serve(ctx context.Context, env Env, front func(n *Node, exit <-chan struct{})) error {
	prober, err := bundledProber()
	if err != nil {
		_, _ = fmt.Fprintln(env.Stderr, err)
		return err
	}
	env.Media = prober
	// Without a Notify for SIGPIPE, Go kills the process on a write to a
	// broken pipe on stderr. With one, the write fails and the log file
	// still gets the record.
	signal.Notify(make(chan os.Signal, 1), syscall.SIGPIPE)
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigs)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-sigs:
			cancel()
		case <-ctx.Done():
		}
	}()
	n, err := Start(ctx, env)
	if err != nil {
		return err
	}
	exit := make(chan struct{})
	go func() {
		defer close(exit)
		select {
		case <-n.Done():
			err = n.Err()
			return
		case <-ctx.Done():
		}
		select {
		case <-n.Done():
			err = n.Err()
		case <-sigs:
			n.log.Error("signaled again while stopping, exiting at once")
			err = ErrInterrupted
		}
	}()
	front(n, exit)
	<-exit
	return err
}

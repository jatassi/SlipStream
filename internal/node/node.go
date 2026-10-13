package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/node/internal/config"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/internal/lock"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/pairing"
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
	Environ []string
	Stderr  io.Writer
	Version string
	// Net serves every listener, the control socket's included, and every
	// link dial.
	Net    link.Network
	Engine func(engine.Config) (engine.Engine, error)
	// OpenLog opens the log in dir, mirrored to stderr.
	OpenLog func(dir string, stderr io.Writer) (*logs.Log, error)
	// Test adds test-only Roles; a process leaves it zero.
	Test Test
}

// ProcessEnv is this process's Env, with the data dir from dataDirFlag or
// else SLIPSTREAM_DATA_DIR.
func ProcessEnv(dataDirFlag string) (Env, error) {
	environ := os.Environ()
	raw := dataDirFlag
	if raw == "" {
		if i := slices.IndexFunc(environ, func(kv string) bool { return strings.HasPrefix(kv, config.DataDirVar+"=") }); i >= 0 {
			raw = strings.TrimPrefix(environ[i], config.DataDirVar+"=")
		}
	}
	if raw == "" {
		return Env{}, &UsageError{Err: fmt.Errorf("no data dir: pass --data-dir or set %s", config.DataDirVar)}
	}
	dir, err := ParseDataDir(raw)
	if err != nil {
		return Env{}, &UsageError{Err: err}
	}
	version := "dev"
	if s, ok := buildinfo.Current(); ok {
		version = s.Version
	}
	return Env{
		DataDir: dir,
		Environ: environ,
		Stderr:  os.Stderr,
		Version: version,
		Net:     osNetwork{},
		Engine:  func(c engine.Config) (engine.Engine, error) { return libtorrent.Open(c) },
		OpenLog: logs.Open,
	}, nil
}

// Node is one running SlipStream process's worth of state. Several can run in
// one Go process; they share nothing.
type Node struct {
	env      Env
	started  time.Time
	rootLog  *slog.Logger
	log      *slog.Logger
	stack    *Stack
	deadline time.Duration
	ready    Ready

	server  *control.Server
	isReady atomic.Bool

	// Set during start, before the Node is ready.
	cfg      Config
	key      link.PrivateKey
	info     link.NodeInfo
	hub      *live.Hub
	gate     *identity.Gate
	link     *link.Link
	pairing  *pairing.Pairing
	lockstep *http.ServeMux
	roleRand *rand.Rand
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
		deadline: Defaults().Timings.Node.StopDeadline,
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
func (n *Node) Stop() { n.stopOnce.Do(func() { close(n.stopping) }) }

// start runs the startup order. Errors before the log opens go to stderr as
// plain text, later ones to the log.
func (n *Node) start(ctx context.Context) error {
	lk, err := lock.Acquire(n.env.DataDir.RunDir())
	if held, ok := errors.AsType[*lock.HeldError](err); ok {
		err = &InUseError{Dir: n.env.DataDir, PID: held.PID}
	}
	if err != nil {
		_, _ = fmt.Fprintln(n.env.Stderr, err)
		return err
	}
	log, err := n.env.OpenLog(n.env.DataDir.LogsDir(), n.env.Stderr)
	if err != nil {
		_, _ = fmt.Fprintln(n.env.Stderr, err)
		return errors.Join(err, lk.Release())
	}
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
	cfg, keys, err := LoadConfig(n.env.DataDir, n.env.Environ)
	if err != nil {
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
	n.log.Info("ready", "ready", n.ready)
	n.isReady.Store(true)
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
	torrent, err := n.env.Engine(engine.Config{ListenInterface: cfg.Torrent.Listen})
	if err != nil {
		return 0, "", fmt.Errorf("open torrent engine on %s: %w", cfg.Torrent.Listen, err)
	}
	n.stack.OnStop("torrent", func(context.Context) error { return torrent.Close() })
	if l.Torrent, err = torrent.ListenAddr(); err != nil {
		return 0, "", fmt.Errorf("torrent listen address: %w", err)
	}
	if err := n.startLink(cfg, streams, l); err != nil {
		return 0, "", err
	}
	n.pairing, err = pairing.Open(pairing.Config{
		DataDir: n.env.DataDir.Path(),
		Link:    n.link,
		Name:    cfg.Name,
		Claims:  surfaceNode{n},
		Log:     n.rootLog.With("subsystem", "pairing"),
	})
	if _, malformed := errors.AsType[*pairing.RecordError](err); malformed {
		err = &RecordError{Err: err}
	}
	if err != nil {
		return 0, "", err
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
// process that exits on Done has already answered them.
func (n *Node) finish(err error) {
	n.err = err
	close(n.drained)
	if n.server != nil {
		n.server.Wait()
	}
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

// Run is Start plus signals: SIGINT or SIGTERM stops the Node, and a second
// one returns ErrInterrupted at once. Losing the reader of stderr does not
// stop it.
func Run(ctx context.Context, env Env) error {
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
	select {
	case <-n.Done():
		return n.Err()
	case <-ctx.Done():
	}
	select {
	case <-n.Done():
		return n.Err()
	case <-sigs:
		n.log.Error("signaled again while stopping, exiting at once")
		return ErrInterrupted
	}
}

package node

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/node/internal/config"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/internal/lock"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/libtorrent"
)

// Env is what the process supplies to a Node.
type Env struct {
	DataDir DataDir
	Environ []string
	Stderr  io.Writer
	Version string
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
	return Env{DataDir: dir, Environ: environ, Stderr: os.Stderr, Version: version}, nil
}

// Node is one running SlipStream process's worth of state. Several can run in
// one Go process; they share nothing.
type Node struct {
	env      Env
	log      *slog.Logger
	stack    *Stack
	deadline time.Duration
	ready    Ready

	server  *control.Server
	isReady atomic.Bool

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
		deadline: DefaultTimings().StopDeadline,
		stopping: make(chan struct{}),
		drained:  make(chan struct{}),
		released: make(chan struct{}),
		done:     make(chan struct{}),
	}
	go func() {
		select {
		case <-ctx.Done():
			n.Stop()
		case <-n.stopping:
		}
	}()
	if err := n.start(); err != nil {
		n.Stop()
		n.finish(err)
		return nil, err
	}
	go n.wait()
	return n, nil
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
func (n *Node) start() error {
	lk, err := lock.Acquire(n.env.DataDir.RunDir())
	if held, ok := errors.AsType[*lock.HeldError](err); ok {
		err = &InUseError{Dir: n.env.DataDir, PID: held.PID}
	}
	if err != nil {
		_, _ = fmt.Fprintln(n.env.Stderr, err)
		return err
	}
	log, err := logs.Open(n.env.DataDir.LogsDir(), n.env.Stderr)
	if err != nil {
		_, _ = fmt.Fprintln(n.env.Stderr, err)
		return errors.Join(err, lk.Release())
	}
	n.log = log.With("subsystem", "node")
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
	if err := n.startLogged(); err != nil {
		if !errors.Is(err, ErrStopped) {
			n.log.Error("start failed", "err", err.Error())
		}
		return errors.Join(err, n.stack.unwind(n.deadline))
	}
	return nil
}

func (n *Node) startLogged() error {
	cfg, keys, err := LoadConfig(n.env.DataDir, n.env.Environ)
	if err != nil {
		return err
	}
	n.deadline = cfg.Timings.Node.StopDeadline
	if changed := changedTimings(keys); len(changed) > 0 {
		n.log.Warn("timings differ from their defaults", "keys", changed)
	}
	var l Listeners
	if n.server, err = control.Listen(n.env.DataDir.ControlSocket(), n.dispatch); err != nil {
		return err
	}
	n.stack.OnStop("control", func(context.Context) error { return n.server.Close() })
	l.Control = n.env.DataDir.ControlSocket()
	if err := n.checkpoint(); err != nil {
		return err
	}
	if err := n.startSubsystems(cfg, &l); err != nil {
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
		Listeners: l,
		Overrides: overrides(keys),
	}
	n.log.Info("ready", "ready", n.ready)
	n.isReady.Store(true)
	return nil
}

// startSubsystems is the start order. Each subsystem pushes its stop onto the
// Stack, so they stop in reverse.
func (n *Node) startSubsystems(cfg Config, l *Listeners) error {
	torrent, err := libtorrent.Open(engine.Config{ListenInterface: cfg.Torrent.Listen})
	if err != nil {
		return fmt.Errorf("open torrent engine on %s: %w", cfg.Torrent.Listen, err)
	}
	n.stack.OnStop("torrent", func(context.Context) error { return torrent.Close() })
	if l.Torrent, err = torrent.ListenAddr(); err != nil {
		return fmt.Errorf("torrent listen address: %w", err)
	}
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

// ErrInterrupted is a second signal during a stop.
var ErrInterrupted = errors.New("interrupted again while stopping")

// Run is Start plus signals: SIGINT or SIGTERM stops the Node, and a second
// one returns ErrInterrupted at once.
func Run(ctx context.Context, env Env) error {
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

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

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/buildinfo"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/converting/clip"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/fvad"
	"github.com/jatassi/slipstream/internal/media/speech"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/netfacts"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/node/internal/config"
	"github.com/jatassi/slipstream/internal/node/internal/control"
	"github.com/jatassi/slipstream/internal/node/internal/host"
	"github.com/jatassi/slipstream/internal/node/internal/lock"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/node/surface"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifying"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/ports/all"
	"github.com/jatassi/slipstream/internal/ports/port"
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
	// Speech builds Speech Maps: the bundled ffmpeg and libfvad for a
	// process.
	Speech speech.Mapper
	// Converting is how this Node runs ffmpeg and proves what it can
	// convert, given its scratch folder, ffmpeg path, prober and timings:
	// converting.Real over the bundled tools for a process.
	Converting func(cfg converting.RealConfig) (converting.Converter, converting.Proofs)
	// Scripts runs the Notifying Role's Custom Scripts: a ProcessRunner for
	// a process.
	Scripts notifying.Runner
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
	// Network reads this machine's network for netfacts: the OS's readers
	// for a process.
	Network netfacts.Env
	// Restore is the restore the Docker variables ask for, nil for none.
	Restore *EnvRestore
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
	environ, restore, err := envRestore(environ)
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
		Scripts:   notifying.ProcessRunner{Environ: environ, TempDir: os.TempDir()},
		Disk:      OSDisk,
		Network:   netfacts.OS(osPackets{}),
		Restart:   processRestart(layout.Executable, os.Args, environ, dir.LogsDir(), os.Stderr),
		RestartOf: follows,
		Restore:   restore,
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
		return fmt.Errorf("data dir %s belongs to %s. Run: %s", dir, owner.Name, hint)
	}
	remote, err := host.OnNetworkShare(dir.Path())
	if err != nil {
		return err
	}
	if remote {
		return fmt.Errorf("data dir %s is on a network share, and SQLite's WAL needs a local disk", dir)
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

	server *control.Server
	// phaseOf is the Phase, which status reads before ready.
	phaseOf atomic.Uint32

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
	served      *serving.Listener
	pairing     *pairing.Pairing
	lockstep    *http.ServeMux
	roleRand    *rand.Rand
	grabsRand   *rand.Rand
	importsRand *rand.Rand
	filesRand   *rand.Rand
	tasksRand   *rand.Rand
	// metadataRand draws the metadata call layer's retry jitter.
	metadataRand *rand.Rand
	// subtitlingRand draws the subtitles call layer's retry jitter.
	subtitlingRand *rand.Rand
	// subtitlesRand draws the Controller's subtitles removal backoffs.
	subtitlesRand *rand.Rand
	// cacheRand draws the Controller's metadata cache generation.
	cacheRand *rand.Rand
	// systemRand draws the Controller's waits on an offline Node's log.
	systemRand *rand.Rand
	// notificationsRand draws the jitter of a failing connection's retries.
	notificationsRand *rand.Rand
	// notifications sends History's entries, on a Controller.
	notifications *notifications.Service
	// history is the log of what happened to the library, on a Controller.
	history *history.Service
	// host is what NodeSystemService serves for this Node, and hostMu
	// orders its reports to the holder half.
	host   *nodeHost
	hostMu sync.Mutex
	// torrent is the Node's torrent engine, downloads its built-in client,
	// and downloading the Role that holds it beside each external client.
	torrent     engine.Engine
	downloads   *downloading.Client
	downloading *downloading.Node
	// mediaLink is the running half's Media Server link holder.
	mediaLink *mediaserverlink.Link
	// converter runs this Node's ffmpeg, and proofs are its test runs.
	converter converting.Converter
	proofs    converting.Proofs
	// network is the running half's network facts, and web the web
	// listener they report as SlipStream's own, once HTTP binds.
	network *netfacts.Netfacts
	web     atomic.Pointer[netmodel.Listener]
	// ports is the one port Registry, which the Controller lists and the
	// Node runs.
	ports port.Registry
	// store is the Node's Storage over disk, which Transfers read and
	// write through too.
	store *storing.Store
	disk  storing.Disk
	// backups takes every Backup, the hook's included.
	backups *backups.Engine
	// claimed is where library.db and a half push their stops, so they
	// stop after HTTP and the link, whenever they start. Its goroutines
	// halt as the Node starts stopping, so a call they have in flight ends
	// canceled rather than cut off by the link closing under it.
	claimed *Stack

	libMu sync.Mutex
	lib   atomic.Pointer[libraryDB]
	half  atomic.Pointer[half]
	// claimMu serializes claims, so a Controller claim and a Join never
	// interleave.
	claimMu sync.Mutex
	// restoreMu guards pending, the first-run page's restore.
	restoreMu sync.Mutex
	pending   *pendingRestore
	// wiped is the Join Code a Wipe and join finished at this start joins
	// with.
	wiped joincode.Code

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
		ports:    all.Registry(),
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
		if hint, ok := hookHint(err); ok {
			_, _ = fmt.Fprintln(n.env.Stderr, hint)
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
	n.setPhase(PhaseReady)
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
	n.importsRand = streams.Stream("imports")
	n.filesRand = streams.Stream("files")
	n.systemRand = streams.Stream("system")
	n.host = &nodeHost{n: n}
	n.tasksRand = streams.Stream("tasks")
	n.metadataRand = streams.Stream("metadata")
	n.subtitlingRand = streams.Stream("subtitling")
	n.subtitlesRand = streams.Stream("subtitles")
	n.cacheRand = streams.Stream("metadata-cache")
	n.notificationsRand = streams.Stream("notifications")
	if n.disk, err = n.env.Disk(n.env.DataDir.Path()); err != nil {
		return 0, "", fmt.Errorf("open the disk: %w", err)
	}
	if err := n.startTorrent(cfg, l, streams.Stream("torrent"), streams.Stream("download-clients")); err != nil {
		return 0, "", err
	}
	n.startStorage(cfg)
	n.startConverting(cfg)
	// The claimed half stops before the torrent engine closes under it.
	n.claimed = n.stack.Sub("claimed")
	if err := n.restoreFromEnv(ctx); err != nil {
		return 0, "", err
	}
	if err := n.resumeLifecycle(ctx); err != nil {
		return 0, "", err
	}
	if n.backups, err = backupsEngine(n.env.DataDir, cfg, n.disk, n.key.Public(), n.env.Version, n.librarySchemas(), n.rootLog.With("subsystem", "backups")); err != nil {
		return 0, "", err
	}
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
		DataDir:          n.env.DataDir.Path(),
		Key:              n.key.Public(),
		Link:             n.link,
		Name:             cfg.Name,
		Claims:           surfaceNode{n},
		AwaitDial:        cfg.Timings.Link.AwaitDial,
		Removed:          n.removed,
		UnreachableAfter: cfg.Timings.Pairing.UnreachableAfter,
		Local:            localPage{n},
		Log:              n.rootLog.With("subsystem", "pairing"),
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
	if err := n.joinAtStartFrom(cfg.Join.Code); err != nil {
		return 0, "", err
	}
	return claim, setupCode, n.startHTTP(ctx, cfg, l, surface.Services{Node: surfaceNode{n}, Hub: n.hub, Live: cfg.Timings.Live}, n.gate)
}

// startTorrent opens the torrent engine and the Node's built-in client,
// which reads its events from now on, claimed or not.
func (n *Node) startTorrent(cfg Config, l *Listeners, torrentRand, clientsRand *rand.Rand) error {
	torrent, err := n.env.Engine(engine.Config{
		ListenInterface:   cfg.Torrent.Listen,
		AnnounceFloor:     cfg.Timings.Downloading.AnnounceFloor,
		RouteCheck:        cfg.Timings.Downloading.RouteCheck,
		DHTRouters:        cfg.Endpoints.Torrent.DHTRouters,
		Rand:              torrentRand,
		PeersShareAddress: cfg.Torrent.LoopbackPeers,
	})
	if err != nil {
		return fmt.Errorf("open torrent engine on %s: %w", cfg.Torrent.Listen, err)
	}
	n.torrent = torrent
	n.downloads = downloading.NewClient(downloading.Config{
		Engine:     torrent,
		Dir:        n.env.DataDir.Resolve(cfg.Torrent.Downloads),
		Listen:     cfg.Torrent.Listen,
		Timings:    cfg.Timings.Downloading,
		RetryMin:   cfg.Timings.Role.RestartMin,
		RetryMax:   cfg.Timings.Role.RestartMax,
		Rand:       torrentRand,
		ProvesPort: downloading.PublicPeer(cfg.Torrent.LoopbackPeers),
		Volume:     func(ctx context.Context, path string) (storing.Volume, error) { return n.store.Volume(ctx, path) },
		Log:        n.rootLog.With("subsystem", "downloading"),
	})
	n.downloading = downloading.NewNode(n.downloads, downloading.NodeConfig{
		HTTP:    httpClient(n.stack, n.env.Net.DialContext),
		Timings: cfg.Timings.Downloading,
		Folder: func(ctx context.Context, path string, write bool) (storing.CheckResult, error) {
			return n.store.JudgeClientFolder(ctx, path, write)
		},
		Holds: func(path string) bool { return n.store.Holds(context.Background(), path) },
		Disk:  n.disk,
		Rand:  clientsRand,
		Log:   n.rootLog.With("subsystem", "download clients"),
	})
	n.stack.Go("torrent client", n.downloads.Loop)
	n.stack.OnStop("torrent", func(context.Context) error { return torrent.Close() })
	// An engine whose port was taken runs without a listener, and the
	// engine-starts probe says so.
	if l.Torrent, err = torrent.ListenAddr(); err != nil && !errors.Is(err, engine.ErrNotListening) {
		return fmt.Errorf("torrent listen address: %w", err)
	}
	return nil
}

func OSDisk(data string) (storing.Disk, error) { return storing.NewOSDisk(data) }

// startStorage opens the Node's Storage over its Disk, claimed or not.
func (n *Node) startStorage(cfg Config) {
	scratch, backupsAt := n.env.DataDir.Resolve(cfg.Converting.Scratch), n.env.DataDir.Resolve(cfg.Backups.Folder)
	n.store = storing.NewStore(storing.Config{
		Disk: n.disk, Payloads: n.downloading, Downloads: n.downloading.Folders, Torrents: n.downloading.Torrents,
		Scratch: func() string { return scratch }, Backups: func() string { return backupsAt }, Timings: cfg.Timings.Storing,
		Prober: n.env.Media, ProbeTimeout: cfg.Timings.Media.Probe, PathLimit: cfg.Storage.PathLimit,
		RemoveRetry: cfg.Timings.Moves.RemoveRetry, RemoveBound: cfg.Timings.Moves.RemoveBound,
	})
}

// startConverting opens the Node's ffmpeg and runs its base test run,
// claimed or not, so Add a Node can suggest Converting.
func (n *Node) startConverting(cfg Config) {
	n.converter, n.proofs = n.env.Converting(converting.RealConfig{
		Scratch: n.scratch, Override: n.ffmpegPath, Media: n.env.Media, Timings: cfg.Timings.Converting,
	})
	n.stack.Go("ffmpeg test run", func(ctx context.Context) error { return n.proofs.Warm(ctx, n.reportHost) })
}

// scratch is the Node's scratch folder now, clean.
func (n *Node) scratch() string {
	n.cfgMu.Lock()
	defer n.cfgMu.Unlock()
	return filepath.Clean(n.env.DataDir.Resolve(n.cfg.Converting.Scratch))
}

// ffmpegPath is the ffmpeg the Admin set in place of the bundled one, ""
// for the bundled one.
func (n *Node) ffmpegPath() string {
	n.cfgMu.Lock()
	defer n.cfgMu.Unlock()
	return n.cfg.Converting.FFmpeg
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
	n.claimed.Halt()
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

// bundledTools are the ffprobe, ffmpeg and dovi_tool the release ships
// beside the executable (ADR 0015), run with environ.
func bundledTools(environ []string) (clip.Tools, error) {
	exe, err := os.Executable()
	if err != nil {
		return clip.Tools{}, fmt.Errorf("find the executable: %w", err)
	}
	beside := func(tool string) string { return filepath.Join(filepath.Dir(exe), tool+filepath.Ext(exe)) }
	for _, tool := range []string{"ffprobe", "ffmpeg", "dovi_tool"} {
		if info, err := os.Stat(beside(tool)); err != nil || !info.Mode().IsRegular() {
			return clip.Tools{}, fmt.Errorf("%s not found beside %s", tool, exe)
		}
	}
	return clip.Tools{FFmpeg: beside("ffmpeg"), FFprobe: beside("ffprobe"), DoviTool: beside("dovi_tool"), Environ: environ}, nil
}

// Run is Start plus signals: SIGINT or SIGTERM stops the Node, and a second
// one returns ErrInterrupted at once. Losing the reader of stderr does not
// stop it.
func Run(ctx context.Context, env Env) error {
	return serve(ctx, env, func(*Node, <-chan struct{}) {})
}

func serve(ctx context.Context, env Env, front func(n *Node, exit <-chan struct{})) error {
	tools, err := bundledTools(env.Environ)
	if err != nil {
		_, _ = fmt.Fprintln(env.Stderr, err)
		return err
	}
	env.Media = media.FFprobe{Path: tools.FFprobe}
	env.Speech = fvad.Mapper{Decoder: media.Decoder{FFmpeg: tools.FFmpeg, FFprobe: tools.FFprobe}}
	env.Converting = func(cfg converting.RealConfig) (converting.Converter, converting.Proofs) {
		cfg.Tools = tools
		return converting.Real(cfg)
	}
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

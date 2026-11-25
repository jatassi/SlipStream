package node

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/calendar"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history"
	"github.com/jatassi/slipstream/internal/home"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/library"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/servingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/trackers"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

type libraryDB struct {
	db     *storage.DB[storage.Library]
	keys   *secret.Keys
	sealer *secret.Sealer
	minter artwork.Minter
	mail   *mail.Mail
	outbox *mail.Outbox
	ids    *identity.Service
}

func (l *libraryDB) close(ctx context.Context) error {
	return errors.Join(l.outbox.Close(ctx), l.db.Close())
}

type libraryFiles struct {
	dir     DataDir
	schemas []storage.Schema[storage.Library]
	// hook is the Backup before a migration, nil for a library.db no Node
	// has started on.
	hook    storage.PreMigrationHook
	timings TimingsBlock
	hub     *live.Hub
	log     *slog.Logger
	test    Test
}

// openLibraryFiles is where a running Node and `slipstream admin reset`
// offline both open library.db.
func openLibraryFiles(ctx context.Context, f libraryFiles) (*libraryDB, error) {
	if err := f.test.seedLibrary(f.dir.LibraryDB(), f.schemas); err != nil {
		return nil, err
	}
	db, err := storage.OpenLibrary(ctx, f.dir.LibraryDB(), f.schemas, f.hook, f.timings.Storage, f.log.With("subsystem", "storage"))
	if err != nil {
		return nil, err
	}
	keys, err := secret.Open(ctx, db, f.dir.LibraryKey())
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	sealer := keys.Sealer()
	smtp := mail.New(mail.Config{Library: db, Sealer: sealer, Timings: f.timings.Mail})
	lib := &libraryDB{db: db, keys: keys, sealer: sealer, minter: artwork.NewMinter(keys.ArtworkToken()), mail: smtp, outbox: mail.NewOutbox(smtp, f.log.With("subsystem", "mail"))}
	lib.ids, err = identity.New(ctx, identity.Config{Library: db, Pepper: keys.PINPepper(), Sealer: sealer, Hub: f.hub, Mail: smtp, Outbox: lib.outbox})
	if err != nil {
		return nil, errors.Join(err, lib.close(ctx))
	}
	return lib, nil
}

func (n *Node) librarySchemas() []storage.Schema[storage.Library] {
	return append(LibrarySchemas(n.env.Types), n.env.Test.Library...)
}

// openLibrary opens library.db and what opens with it once. A Node opens
// it at start only if library.db exists, and otherwise when it is claimed
// as a Controller.
func (n *Node) openLibrary(ctx context.Context) (*libraryDB, error) {
	return n.openLibraryWith(ctx, backups.BeforeMigrating(n.backups, n.backingUp))
}

// openLibraryWith is openLibrary with hook before a migration: none for a
// restore's finish, whose library.db isn't live until its Write.
func (n *Node) openLibraryWith(ctx context.Context, hook storage.PreMigrationHook) (*libraryDB, error) {
	n.libMu.Lock()
	defer n.libMu.Unlock()
	if lib := n.lib.Load(); lib != nil {
		return lib, nil
	}
	lib, err := openLibraryFiles(ctx, libraryFiles{
		dir:     n.env.DataDir,
		schemas: n.librarySchemas(),
		hook:    hook,
		timings: n.cfg.Timings,
		hub:     n.hub,
		log:     n.rootLog,
		test:    n.env.Test,
	})
	if err != nil {
		return nil, err
	}
	n.claimed.OnStop("library.db", lib.close)
	n.lib.Store(lib)
	return lib, nil
}

// claim is the Node's one definition of its claim, read from its files on
// every call: the Setup gate, the status endpoint and every Account check
// read it.
func (n *Node) claim(ctx context.Context) (system.Claim, error) {
	worker, err := pairing.IsWorker(n.env.DataDir.Path())
	switch {
	case err != nil:
		return 0, refusedRecord(fmt.Errorf("read the claim: %w", err))
	case worker:
		return system.Worker, nil
	}
	lib := n.lib.Load()
	if lib == nil {
		return system.Unclaimed, nil
	}
	admin, err := lib.ids.HasAdmin(ctx)
	if err != nil || !admin {
		return system.Unclaimed, err
	}
	return system.Controller, nil
}

// surfaceNode is the Node as the API reaches it.
type surfaceNode struct{ n *Node }

func (s surfaceNode) Claim(ctx context.Context) (system.Claim, error) { return s.n.claim(ctx) }

func (s surfaceNode) Worker() (servingapi.Facts, bool) {
	h := s.n.half.Load()
	if h == nil || h.serving == nil {
		return servingapi.Facts{}, false
	}
	rec, _ := s.n.pairing.Record()
	f := servingapi.Facts{Name: h.holding.Name(), Fingerprint: s.n.key.Public().Fingerprint(), Controller: rec.ControllerName, Seeding: s.n.downloads.Seeding()}
	f.Since, _ = h.serving.Unreachable()
	for _, r := range role.Names() {
		if r != role.Serving && h.holding.Holds(r) {
			f.Holding = append(f.Holding, r)
		}
	}
	return f, true
}

func (s surfaceNode) Identity() (*identity.Service, bool) {
	lib := s.n.lib.Load()
	if lib == nil {
		return nil, false
	}
	return lib.ids, true
}

// ClaimController creates and migrates library.db and library.key, runs
// claim, which inserts the Admin, and starts the Controller half. Opening
// and the half outlive the request that claimed. It returns once the
// Controller's own holder half holds the Roles its new row assigns.
func (s surfaceNode) Mail() (*mail.Mail, bool) {
	lib := s.n.lib.Load()
	if lib == nil {
		return nil, false
	}
	return lib.mail, true
}

func (s surfaceNode) ClaimController(ctx context.Context, claim func(*identity.Service) error) error {
	s.n.claimMu.Lock()
	defer s.n.claimMu.Unlock()
	if s.n.pairing.Paired() {
		return identity.ErrAlreadyClaimed
	}
	if s.n.pairing.Awaiting() {
		return identity.ErrJoinPending
	}
	if err := s.n.lifecycleIdle(); err != nil {
		return err
	}
	lib, err := s.n.openLibrary(context.WithoutCancel(ctx))
	if err != nil {
		return err
	}
	if err := claim(lib.ids); err != nil {
		return err
	}
	if err := s.n.startController(context.WithoutCancel(ctx)); err != nil {
		s.n.stack.fail(fmt.Errorf("start the Controller half: %w", err))
		return err
	}
	s.n.log.Info("claimed", "claim", "controller")
	side := s.n.half.Load().nodes
	return side.AwaitDelivery(ctx, side.Self())
}

func (s surfaceNode) Nodes() (*nodes.Service, bool) {
	h := s.n.half.Load()
	if h == nil || h.nodes == nil {
		return nil, false
	}
	return h.nodes, true
}

func (s surfaceNode) Reconciling() (*reconciling.Service, bool) {
	h := s.n.half.Load()
	if h == nil || h.reconciling == nil {
		return nil, false
	}
	return h.reconciling, true
}

// refusal is the reconciling problem's detail, which only a Controller
// writes.
func (s surfaceNode) refusal(ctx context.Context) (string, error) {
	return s.n.half.Load().reconciling.Refusal(ctx)
}

func (s surfaceNode) Systems() (*nodesystem.Nodes, bool) {
	h := s.n.half.Load()
	if h == nil || h.systems == nil {
		return nil, false
	}
	return h.systems, true
}

func (s surfaceNode) Works() (*works.Works, bool) {
	h := s.n.half.Load()
	if h == nil || h.works == nil {
		return nil, false
	}
	return h.works, true
}

func (s surfaceNode) Minter() (artwork.Minter, bool) {
	h, lib := s.n.half.Load(), s.n.lib.Load()
	if h == nil || h.works == nil || lib == nil {
		return artwork.Minter{}, false
	}
	return lib.minter, true
}

func (s surfaceNode) Metadata() (*metadata.Metadata, bool) {
	h := s.n.half.Load()
	if h == nil || h.metadata == nil {
		return nil, false
	}
	return h.metadata, true
}

func (s surfaceNode) Indexers() (*indexers.Indexers, bool) {
	h := s.n.half.Load()
	if h == nil || h.indexers == nil {
		return nil, false
	}
	return h.indexers, true
}

func (s surfaceNode) Trackers() (*trackers.Trackers, bool) {
	h := s.n.half.Load()
	if h == nil || h.trackers == nil {
		return nil, false
	}
	return h.trackers, true
}

func (s surfaceNode) Targets() (*targets.Targets, bool) {
	h := s.n.half.Load()
	if h == nil || h.targets == nil {
		return nil, false
	}
	return h.targets, true
}

func (s surfaceNode) Roots() (*roots.Roots, bool) {
	h := s.n.half.Load()
	if h == nil || h.roots == nil {
		return nil, false
	}
	return h.roots, true
}

func (s surfaceNode) Files() (*files.Files, bool) {
	h := s.n.half.Load()
	if h == nil || h.files == nil {
		return nil, false
	}
	return h.files, true
}

func (s surfaceNode) Naming() (*naming.Naming, bool) {
	h := s.n.half.Load()
	if h == nil || h.naming == nil {
		return nil, false
	}
	return h.naming, true
}

func (s surfaceNode) Imports() (*imports.Imports, bool) {
	h := s.n.half.Load()
	if h == nil || h.imports == nil {
		return nil, false
	}
	return h.imports, true
}

func (s surfaceNode) Library() (*library.Library, bool) {
	h := s.n.half.Load()
	if h == nil || h.library == nil {
		return nil, false
	}
	return h.library, true
}

func (s surfaceNode) Grabs() (*grabs.Grabs, bool) {
	h := s.n.half.Load()
	if h == nil || h.grabs == nil {
		return nil, false
	}
	return h.grabs, true
}

func (s surfaceNode) DownloadClients() (*downloadclients.Clients, bool) {
	h := s.n.half.Load()
	if h == nil || h.clients == nil {
		return nil, false
	}
	return h.clients, true
}

func (s surfaceNode) Backups() (*backups.Backups, bool) {
	h := s.n.half.Load()
	if h == nil || h.backups == nil {
		return nil, false
	}
	return h.backups, true
}

func (s surfaceNode) Transfers() (*transfers.Transfers, bool) {
	h := s.n.half.Load()
	if h == nil || h.transfers == nil {
		return nil, false
	}
	return h.transfers, true
}

func (s surfaceNode) Moves() (*moves.Moves, bool) {
	h := s.n.half.Load()
	if h == nil || h.moves == nil {
		return nil, false
	}
	return h.moves, true
}

func (s surfaceNode) MediaServers() (mediaservers.Service, bool) {
	h := s.n.half.Load()
	if h == nil || h.mediaServers == nil {
		return nil, false
	}
	return h.mediaServers, true
}

func (s surfaceNode) Reachability() (*reachability.Reachability, bool) {
	h := s.n.half.Load()
	if h == nil || h.reachability == nil {
		return nil, false
	}
	return h.reachability, true
}

func (s surfaceNode) Notifications() (*notifications.Service, bool) {
	h := s.n.half.Load()
	if h == nil || h.notifications == nil {
		return nil, false
	}
	return h.notifications, true
}

func (s surfaceNode) Calendar() (*calendar.Calendar, bool) {
	h := s.n.half.Load()
	if h == nil || h.calendar == nil {
		return nil, false
	}
	return h.calendar, true
}

func (s surfaceNode) Home() (*home.Service, bool) {
	h := s.n.half.Load()
	if h == nil || h.home == nil {
		return nil, false
	}
	return h.home, true
}

func (s surfaceNode) Subtitles() (*subtitles.Subtitles, bool) {
	h := s.n.half.Load()
	if h == nil || h.subtitles == nil {
		return nil, false
	}
	return h.subtitles, true
}

func (s surfaceNode) History() (*history.Service, bool) {
	h := s.n.half.Load()
	if h == nil || h.history == nil {
		return nil, false
	}
	return h.history, true
}

// OpenIn runs on a Controller only, as the Work page it fills does.
func (s surfaceNode) OpenIn(ctx context.Context, work works.WorkID) ([]mediaservers.OpenIn, error) {
	return s.n.half.Load().mediaServers.OpenIn(ctx, work)
}

func (s surfaceNode) Tasks() (*tasks.Scheduler, bool) {
	h := s.n.half.Load()
	if h == nil || h.tasks == nil {
		return nil, false
	}
	return h.tasks, true
}

func (s surfaceNode) Health() (*health.Health, bool) {
	h := s.n.half.Load()
	if h == nil || h.health == nil {
		return nil, false
	}
	return h.health, true
}

func (s surfaceNode) FirstRun() (*firstrun.FirstRun, bool) {
	h := s.n.half.Load()
	if h == nil || h.firstRun == nil {
		return nil, false
	}
	return h.firstRun, true
}

var errControllerNotRunning = errors.New("the Controller isn't running")

// FirstRunOpen reads first setup, which only a running Controller has. A
// claim writes its Admin before the Controller half starts, so a sign-in
// can land in between.
func (s surfaceNode) FirstRunOpen(ctx context.Context) (bool, error) {
	f, ok := s.FirstRun()
	if !ok {
		return false, errControllerNotRunning
	}
	return f.IsOpen(ctx)
}

// Join is claimWorker's Join. The HTTP server's graceful stop lets the
// answer finish before a restart onto a re-paired Controller ends it.
func (s surfaceNode) Join(ctx context.Context, code joincode.Code, awaiting func(pairing.Pending)) (pairing.Joined, error) {
	joined, repaired, err := s.n.join(ctx, code, awaiting)
	if repaired {
		s.n.repaired(context.WithoutCancel(ctx))
	}
	return joined, err
}

// Pending holds a joined claim at awaiting-dial until the Worker half
// runs, so a page that opens the Worker page on joined finds getServing
// answering.
func (s surfaceNode) Pending() (pairing.Pending, bool) {
	p, ok := s.n.pairing.Pending()
	if ok && p.State == pairing.JoinedState && s.n.half.Load() == nil {
		p.State, p.Joined = pairing.AwaitingDial, nil
	}
	return p, ok
}

func (s surfaceNode) Removal() (pairing.Removal, bool) { return s.n.pairing.Removal() }

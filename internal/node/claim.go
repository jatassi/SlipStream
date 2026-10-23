package node

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/servingapi"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

type library struct {
	db     *storage.DB[storage.Library]
	sealer *secret.Sealer
	mail   *mail.Mail
	outbox *mail.Outbox
	ids    *identity.Service
}

func (l *library) close(ctx context.Context) error {
	return errors.Join(l.outbox.Close(ctx), l.db.Close())
}

type libraryFiles struct {
	dir     DataDir
	schemas []storage.Schema[storage.Library]
	timings TimingsBlock
	hub     *live.Hub
	log     *slog.Logger
}

// openLibraryFiles is where a running Node and `slipstream admin reset`
// offline both open library.db.
func openLibraryFiles(ctx context.Context, f libraryFiles) (*library, error) {
	db, err := storage.OpenLibrary(ctx, f.dir.LibraryDB(), f.schemas, preMigrationHook, f.timings.Storage, f.log.With("subsystem", "storage"))
	if err != nil {
		return nil, err
	}
	keys, err := secret.Open(ctx, db, f.dir.LibraryKey())
	if err != nil {
		return nil, errors.Join(err, db.Close())
	}
	sealer := keys.Sealer()
	smtp := mail.New(mail.Config{Library: db, Sealer: sealer, Timings: f.timings.Mail})
	lib := &library{db: db, sealer: sealer, mail: smtp, outbox: mail.NewOutbox(smtp, f.log.With("subsystem", "mail"))}
	lib.ids, err = identity.New(ctx, identity.Config{Library: db, Pepper: keys.PINPepper(), Sealer: sealer, Hub: f.hub, Mail: smtp, Outbox: lib.outbox})
	if err != nil {
		return nil, errors.Join(err, lib.close(ctx))
	}
	return lib, nil
}

// openLibrary opens library.db and what opens with it once. A Node opens
// it at start only if library.db exists, and otherwise when it is claimed
// as a Controller.
func (n *Node) openLibrary(ctx context.Context) (*library, error) {
	n.libMu.Lock()
	defer n.libMu.Unlock()
	if lib := n.lib.Load(); lib != nil {
		return lib, nil
	}
	lib, err := openLibraryFiles(ctx, libraryFiles{
		dir:     n.env.DataDir,
		schemas: append(LibrarySchemas(n.env.Types), n.env.Test.Library...),
		timings: n.cfg.Timings,
		hub:     n.hub,
		log:     n.rootLog,
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

func (s surfaceNode) Indexers() (*indexers.Indexers, bool) {
	h := s.n.half.Load()
	if h == nil || h.indexers == nil {
		return nil, false
	}
	return h.indexers, true
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

func (s surfaceNode) Grabs() (*grabs.Grabs, bool) {
	h := s.n.half.Load()
	if h == nil || h.grabs == nil {
		return nil, false
	}
	return h.grabs, true
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

func (s surfaceNode) Join(ctx context.Context, code joincode.Code, awaiting func(pairing.Pending)) (pairing.Joined, error) {
	return s.n.join(ctx, code, awaiting)
}

func (s surfaceNode) Pending() (pairing.Pending, bool) { return s.n.pairing.Pending() }

func (s surfaceNode) Removal() (pairing.Removal, bool) { return s.n.pairing.Removal() }

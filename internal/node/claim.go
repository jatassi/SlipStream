package node

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// library is library.db and what opens with it: the sealer for its
// credentials and the identity service.
type library struct {
	db     *storage.DB[storage.Library]
	sealer *secret.Sealer
	ids    *identity.Service
}

// openLibrary opens library.db, library.key and the identity service once,
// creating the files. A Node opens them at start only if library.db
// exists, and otherwise when it is claimed as a Controller.
func (n *Node) openLibrary(ctx context.Context) (*library, error) {
	n.libMu.Lock()
	defer n.libMu.Unlock()
	if lib := n.lib.Load(); lib != nil {
		return lib, nil
	}
	db, err := storage.OpenLibrary(ctx, n.env.DataDir.LibraryDB(), append(LibrarySchemas(n.env.Types), n.env.Test.Library...), preMigrationHook, n.cfg.Timings.Storage, n.rootLog.With("subsystem", "storage"))
	if err != nil {
		return nil, err
	}
	n.claimed.OnStop("library.db", func(context.Context) error { return db.Close() })
	keys, err := secret.Open(ctx, db, n.env.DataDir.LibraryKey())
	if err != nil {
		return nil, err
	}
	ids, err := identity.New(ctx, identity.Config{Library: db, Pepper: keys.PINPepper(), Hub: n.hub})
	if err != nil {
		return nil, err
	}
	lib := &library{db: db, sealer: keys.Sealer(), ids: ids}
	n.lib.Store(lib)
	return lib, nil
}

// claim is the Node's one definition of its claim, read from its files on
// every call: the Setup gate, the status endpoint and every Account check
// read it.
func (n *Node) claim(ctx context.Context) (system.Claim, error) {
	_, err := os.Stat(filepath.Join(n.env.DataDir.Path(), pairing.RecordFile))
	switch {
	case err == nil:
		return system.Worker, nil
	case !errors.Is(err, fs.ErrNotExist):
		return 0, fmt.Errorf("read the claim: %w", err)
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
func (s surfaceNode) ClaimController(ctx context.Context, claim func(*identity.Service) error) error {
	s.n.claimMu.Lock()
	defer s.n.claimMu.Unlock()
	if s.n.pairing.Paired() {
		return identity.ErrAlreadyClaimed
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

func (s surfaceNode) Join(ctx context.Context, code joincode.Code) (pairing.Joined, error) {
	return s.n.join(ctx, code)
}

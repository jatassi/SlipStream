package node

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

func librarySchemas() []storage.Schema[storage.Library] {
	return []storage.Schema[storage.Library]{
		secret.Schema(),
		identity.Schema(),
	}
}

func localSchemas() []storage.Schema[storage.Local] {
	return []storage.Schema[storage.Local]{}
}

func preMigrationHook(context.Context, *storage.Unmigrated, storage.Plan) error {
	return nil
}

// openStorage opens both files and hands library.db on to the subsystems
// that start after it.
func (n *Node) openStorage(ctx context.Context, cfg Config) (*storage.DB[storage.Library], error) {
	log := n.rootLog.With("subsystem", "storage")
	lib, err := storage.OpenLibrary(ctx, n.env.DataDir.LibraryDB(), librarySchemas(), preMigrationHook, cfg.Timings.Storage, log)
	if err != nil {
		return nil, n.stoppedOr(err)
	}
	n.stack.OnStop("library.db", func(context.Context) error { return lib.Close() })
	local, err := storage.OpenLocal(ctx, n.env.DataDir.LocalDB(), localSchemas(), cfg.Timings.Storage, log)
	if err != nil {
		return nil, n.stoppedOr(err)
	}
	n.stack.OnStop("local.db", func(context.Context) error { return local.Close() })
	return lib, nil
}

func (n *Node) stoppedOr(err error) error {
	if stop := n.checkpoint(); stop != nil {
		return stop
	}
	return err
}

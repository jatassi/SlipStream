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

func (n *Node) openStorage(ctx context.Context, cfg Config) error {
	log := n.rootLog.With("subsystem", "storage")
	lib, err := storage.OpenLibrary(ctx, n.env.DataDir.LibraryDB(), librarySchemas(), preMigrationHook, cfg.Timings.Storage, log)
	if err != nil {
		return n.stoppedOr(err)
	}
	n.stack.OnStop("library.db", func(context.Context) error { return lib.Close() })
	local, err := storage.OpenLocal(ctx, n.env.DataDir.LocalDB(), localSchemas(), cfg.Timings.Storage, log)
	if err != nil {
		return n.stoppedOr(err)
	}
	n.stack.OnStop("local.db", func(context.Context) error { return local.Close() })
	return nil
}

func (n *Node) stoppedOr(err error) error {
	if stop := n.checkpoint(); stop != nil {
		return stop
	}
	return err
}

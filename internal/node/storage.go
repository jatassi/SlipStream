package node

import (
	"context"

	"github.com/jatassi/slipstream/internal/storage"
)

// librarySchemas is library.db's migration order. A core owner adds one
// line; a set may reference only sets above it. Content Types' schemas are
// appended after every core owner once Content Types exist (ADR 0001).
// TestSchemaListsPassTheAudit audits this list.
func librarySchemas() []storage.Schema[storage.Library] {
	return []storage.Schema[storage.Library]{}
}

// localSchemas is local.db's migration order, audited the same way.
func localSchemas() []storage.Schema[storage.Local] {
	return []storage.Schema[storage.Local]{}
}

// backupBeforeMigrate is the pre-migration Backup seam. Backups and restore
// replaces this body with the encrypted Backup and keeps the call site.
func backupBeforeMigrate(context.Context, *storage.Unmigrated, storage.Plan) error {
	return nil
}

// openStorage opens library.db before local.db, so local.db closes first.
func (n *Node) openStorage(ctx context.Context, cfg Config) error {
	log := n.base.With("subsystem", "storage")
	lib, err := storage.OpenLibrary(ctx, n.env.DataDir.LibraryDB(), librarySchemas(), backupBeforeMigrate, cfg.Timings.Storage, log)
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

// stoppedOr reports a stop that interrupted a step as ErrStopped (exit 0),
// not as the step's context.Canceled (exit 1).
func (n *Node) stoppedOr(err error) error {
	if stop := n.checkpoint(); stop != nil {
		return stop
	}
	return err
}

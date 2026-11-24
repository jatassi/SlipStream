package node

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/lifecycle"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

// writeRestored opens the swapped library.db with no Backup before its
// migrations and runs the restore's Write, then records the restore in
// the Backups index. An automatic rollback only opens: nothing but
// migrations wrote between its Backup and the rollback.
func (n *Node) writeRestored(ctx context.Context, op *lifecycle.Restore) error {
	lib, err := n.openLibraryWith(ctx, nil)
	if err != nil {
		return err
	}
	if op.Cause == lifecycle.AutomaticRollback {
		return nil
	}
	if err := restoredWrite(ctx, lib.db, n.env.DataDir, n.key.Public(), op); err != nil {
		return err
	}
	index, err := backups.OpenIndex(filepath.Join(n.env.DataDir.Path(), backups.IndexFile), n.cfg.Timings.Backups.WindowsRetry)
	if err != nil {
		return err
	}
	return backups.Restored(index, n.env.Version, op.Typed, time.Now())
}

// restoredWrite is the restore's Write. It runs once: the first thing it
// reads is the restore id reconciling_state records, and a library that
// holds op's id was already written. A different id is a Backup taken
// while an earlier Reconciling ran, which Begin replaces.
func restoredWrite(ctx context.Context, db *storage.DB[storage.Library], dir DataDir, self link.Key, op *lifecycle.Restore) error {
	pre := filepath.Join(dir.Path(), storage.LibraryFile+lifecycle.Aside)
	steps := []func(context.Context, *storage.Tx[storage.Library]) error{
		func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return settleNodes(ctx, tx, self, op)
		},
		func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return reconciling.Begin(ctx, tx, reconciling.BeginArgs{
				ID: op.ID, Cause: reconcilingCause[op.Cause], Backup: op.Backup.Name, TakenAt: op.Backup.TakenAt, NewMachine: op.NewMachine,
			})
		},
		func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return carryCredentials(ctx, tx, op, pre)
		},
		identity.DeleteSessions, health.DeleteItems, roots.DeleteVolumes, downloadclients.DeleteVolumes,
		grabs.CancelOpenRuns, files.DropQueuedMoves, moves.Restored,
		func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return nodes.DeleteSelfAddresses(ctx, tx, self)
		},
		func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return notifications.GiveUpBefore(ctx, tx, op.Backup.TakenAt)
		},
	}
	err := db.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		done, err := reconciling.WroteRestore(ctx, tx, op.ID)
		if err != nil || done {
			return err
		}
		for _, step := range steps {
			if err := step(ctx, tx); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("write the restored library: %w", err)
	}
	return nil
}

var reconcilingCause = map[lifecycle.Cause]reconciling.Cause{
	lifecycle.ByRestore:      reconciling.Restore,
	lifecycle.ManualRollback: reconciling.ManualRollback,
}

// settleNodes gives the Controller row this Node's key on a new machine,
// removes the Nodes the Admin unticked and, in place, the ones removed
// after the Backup was taken, so none of them waits for Holdings, and
// clears the fence.
func settleNodes(ctx context.Context, tx *storage.Tx[storage.Library], self link.Key, op *lifecycle.Restore) error {
	if op.NewMachine {
		old, err := link.KeyFromBytes(op.Backup.ControllerKey)
		if err != nil {
			return fmt.Errorf("the Backup's Controller key: %w", err)
		}
		if _, err := nodes.Rekey(ctx, tx, self, old, op.Superseded[0].Proof); err != nil {
			return err
		}
	}
	for _, s := range op.Remove {
		u, err := storage.ParseUUID(s)
		if err != nil {
			return fmt.Errorf("a Node to remove: %w", err)
		}
		if err := nodes.RemoveIn(ctx, tx, nodes.ID{UUID: u}); err != nil {
			return err
		}
	}
	for _, b := range op.RemovedKeys {
		k, err := link.KeyFromBytes(b)
		if err != nil {
			return fmt.Errorf("a removed Node's key: %w", err)
		}
		if err := nodes.RemoveKeyIn(ctx, tx, k); err != nil {
			return err
		}
	}
	return nodes.ClearFence(ctx, tx)
}

// carryCredentials keeps, in place, the Accounts, PINs, passkeys and API
// key of the library the restore replaced, which the swap moved aside to
// pre. A data dir that held no library has none to keep. A manual
// rollback replaces the API key, and a new machine keeps the Backup's.
func carryCredentials(ctx context.Context, tx *storage.Tx[storage.Library], op *lifecycle.Restore, pre string) error {
	switch {
	case op.NewMachine:
		return nil
	case op.Cause == lifecycle.ManualRollback:
		return identity.ReplaceAPIKey(ctx, tx)
	}
	switch _, err := os.Stat(pre); {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("find the library the restore replaced: %w", err)
	}
	return storage.CarryFrom(ctx, tx, pre, identity.Carried)
}

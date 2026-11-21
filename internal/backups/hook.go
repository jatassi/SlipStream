package backups

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/jatassi/slipstream/internal/backups/archive"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

// HookError is the hook's failure: nothing migrated, so the previous
// binary still opens library.db. node.ExitCode maps it to BackupFailed
// (7). Space is set when the cause is room, so the CLI adds its hint
// without reading the words.
type HookError struct {
	Space *SpaceError
	Err   error
}

func (e *HookError) Error() string { return "back up library.db: " + e.Err.Error() }

func (e *HookError) Unwrap() error { return e.Err }

// BeforeMigrating is the storage.PreMigrationHook every open of a live
// library.db runs. storage calls it only when the file holds an applied
// migration and a migration is pending. It verifies library.key against
// the secret set's check row, so a foreign key refuses with
// LibraryRefused (5) before anything is written, then takes an update
// Backup and lists it before it returns, so a crash after the migration
// leaves it for a rollback. phase is true while the Backup runs.
func BeforeMigrating(e *Engine, phase func(backingUp bool)) storage.PreMigrationHook {
	return func(ctx context.Context, u *storage.Unmigrated, _ storage.Plan) error {
		check, ok, err := u.SecretCheck(ctx)
		if err != nil {
			return err
		}
		keys, err := secret.Verify(filepath.Join(e.cfg.Dir, secret.KeyFile), check, ok)
		if err != nil {
			return err
		}
		if _, err := e.Vault(keys); err != nil {
			return &HookError{Err: err}
		}
		phase(true)
		defer phase(false)
		if _, err := e.take(ctx, u, request{Trigger: archive.Update}); err != nil {
			space, _ := errors.AsType[*SpaceError](err)
			return &HookError{Space: space, Err: err}
		}
		return nil
	}
}

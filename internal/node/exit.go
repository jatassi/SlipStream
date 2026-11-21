package node

import (
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/backups"
	"github.com/jatassi/slipstream/internal/exitcode"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/storage"
)

const (
	ExitOK             = exitcode.OK
	ExitFailed         = exitcode.Failed
	ExitUsage          = exitcode.Usage
	ExitInUse          = exitcode.InUse
	ExitLibraryRefused = exitcode.LibraryRefused
	ExitRecordRefused  = exitcode.RecordRefused
	ExitBackupFailed   = exitcode.BackupFailed
)

// UsageError is a bad command line or a missing or invalid data dir.
type UsageError struct{ Err error }

func (e *UsageError) Error() string { return e.Err.Error() }

func (e *UsageError) Unwrap() error { return e.Err }

type ConfigError struct{ Err error }

func (e *ConfigError) Error() string { return e.Err.Error() }

func (e *ConfigError) Unwrap() error { return e.Err }

// InUseError is a data dir another process holds. PID is 0 while the holder
// is still starting and has not written its PID file.
type InUseError struct {
	Dir DataDir
	PID int
}

func (e *InUseError) Error() string {
	if e.PID == 0 {
		return fmt.Sprintf("data dir %s is in use by a SlipStream that is still starting", e.Dir)
	}
	return fmt.Sprintf("data dir %s is in use by SlipStream PID %d", e.Dir, e.PID)
}

// RecordError is a node.key or pairing.json the Node cannot use.
type RecordError struct{ Err error }

func (e *RecordError) Error() string { return e.Err.Error() }

func (e *RecordError) Unwrap() error { return e.Err }

// refusedRecord is err as a RecordError when pairing.json is one this
// binary can't use.
func refusedRecord(err error) error {
	if _, malformed := errors.AsType[*pairing.RecordError](err); malformed {
		return &RecordError{Err: err}
	}
	return err
}

var ErrStopped = errors.New("node stopped before it was ready")

func ExitCode(err error) int {
	var usage *UsageError
	var cfg *ConfigError
	var inUse *InUseError
	var refused *storage.RefusedError
	var record *RecordError
	var hook *backups.HookError
	switch {
	case err == nil, errors.Is(err, ErrStopped):
		return ExitOK
	case errors.As(err, &usage), errors.As(err, &cfg):
		return ExitUsage
	case errors.As(err, &inUse):
		return ExitInUse
	case errors.As(err, &hook):
		return ExitBackupFailed
	case errors.As(err, &refused):
		return ExitLibraryRefused
	case errors.As(err, &record):
		return ExitRecordRefused
	default:
		return ExitFailed
	}
}

// hookHint is what to do after the Backup before a migration failed,
// chosen by the error's type, never its words.
func hookHint(err error) (string, bool) {
	hook, ok := errors.AsType[*backups.HookError](err)
	switch {
	case !ok:
		return "", false
	case hook.Space != nil:
		return "Free space in " + hook.Space.Folder + ", or start the previous version of SlipStream.", true
	default:
		return "Start the previous version of SlipStream, then check the log for the cause.", true
	}
}

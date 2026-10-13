package node

import (
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/storage"
)

// Exit codes are a contract with the tray launcher, the service manager and
// the health trial. 2 is left to Go's runtime, which exits 2 on an
// unrecovered panic.
const (
	ExitOK     = 0
	ExitFailed = 1
	ExitUsage  = 3
	ExitInUse  = 4
	// ExitLibraryRefused is a library.db newer than this binary, holding a
	// schema set it lacks, or unreadable.
	ExitLibraryRefused = 5
	// ExitRecordRefused is a node key or pairing record refused: an
	// unreadable or malformed node.key or pairing.json.
	ExitRecordRefused = 6
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
		return fmt.Sprintf("data dir %s is in use by SlipStream PID unknown (starting)", e.Dir)
	}
	return fmt.Sprintf("data dir %s is in use by SlipStream PID %d", e.Dir, e.PID)
}

// RecordError is a node.key or pairing.json the Node cannot use.
type RecordError struct{ Err error }

func (e *RecordError) Error() string { return e.Err.Error() }

func (e *RecordError) Unwrap() error { return e.Err }

var ErrStopped = errors.New("node stopped before it was ready")

func ExitCode(err error) int {
	var usage *UsageError
	var cfg *ConfigError
	var inUse *InUseError
	var refused *storage.RefusedError
	var record *RecordError
	switch {
	case err == nil, errors.Is(err, ErrStopped):
		return ExitOK
	case errors.As(err, &usage), errors.As(err, &cfg):
		return ExitUsage
	case errors.As(err, &inUse):
		return ExitInUse
	case errors.As(err, &refused):
		return ExitLibraryRefused
	case errors.As(err, &record):
		return ExitRecordRefused
	default:
		return ExitFailed
	}
}

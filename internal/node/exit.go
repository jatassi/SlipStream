package node

import (
	"errors"
	"fmt"
)

// Exit codes are a contract with the tray launcher, the service manager and
// the health trial. 2 is left to Go's runtime, which exits 2 on an
// unrecovered panic.
const (
	ExitOK     = 0
	ExitFailed = 1
	ExitUsage  = 3
	ExitInUse  = 4
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

var ErrStopped = errors.New("node stopped before it was ready")

func ExitCode(err error) int {
	var usage *UsageError
	var cfg *ConfigError
	var inUse *InUseError
	switch {
	case err == nil, errors.Is(err, ErrStopped):
		return ExitOK
	case errors.As(err, &usage), errors.As(err, &cfg):
		return ExitUsage
	case errors.As(err, &inUse):
		return ExitInUse
	default:
		return ExitFailed
	}
}

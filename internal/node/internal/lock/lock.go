// Package lock holds a Node's data dir: a try-lock on run/node.lock, which
// the OS releases however the process dies, and the holder's PID in
// run/node.pid. The PID has its own unlocked file because Windows locks are
// mandatory, so a refused process could not read bytes inside the locked file.
package lock

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofrs/flock"
)

type Lock struct {
	file *flock.Flock
	pid  string
}

// HeldError is a lock another process holds. PID is 0 when the holder has
// not written its PID file yet.
type HeldError struct{ PID int }

func (e *HeldError) Error() string {
	return fmt.Sprintf("lock held by PID %d", e.PID)
}

// Acquire creates runDir with mode 0700 if it is missing, takes the lock and
// writes this process's PID.
func Acquire(runDir string) (*Lock, error) {
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		return nil, fmt.Errorf("create run dir: %w", err)
	}
	l := &Lock{file: flock.New(filepath.Join(runDir, "node.lock")), pid: filepath.Join(runDir, "node.pid")}
	ok, err := l.file.TryLock()
	if err != nil {
		return nil, fmt.Errorf("lock %s: %w", l.file.Path(), err)
	}
	if !ok {
		return nil, &HeldError{PID: readPID(runDir)}
	}
	if err := writePID(runDir, l.pid); err != nil {
		return nil, errors.Join(err, l.file.Unlock())
	}
	return l, nil
}

// Release removes the PID file, then unlocks.
func (l *Lock) Release() error {
	var errs []error
	if err := os.Remove(l.pid); err != nil && !errors.Is(err, fs.ErrNotExist) {
		errs = append(errs, fmt.Errorf("remove PID file: %w", err))
	}
	if err := l.file.Unlock(); err != nil {
		errs = append(errs, fmt.Errorf("unlock %s: %w", l.file.Path(), err))
	}
	return errors.Join(errs...)
}

// writePID writes by temp file and rename, so a reader never sees a partial PID.
func writePID(runDir, path string) error {
	tmp, err := os.CreateTemp(runDir, "node.pid.*")
	if err != nil {
		return fmt.Errorf("write PID file: %w", err)
	}
	_, err = tmp.WriteString(strconv.Itoa(os.Getpid()) + "\n")
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return errors.Join(fmt.Errorf("write PID file: %w", err), os.Remove(tmp.Name()))
	}
	return nil
}

func readPID(runDir string) int {
	b, err := fs.ReadFile(os.DirFS(runDir), "node.pid")
	if errors.Is(err, fs.ErrNotExist) {
		return 0
	}
	pid, perr := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || perr != nil {
		return 0
	}
	return pid
}

// Package logs builds one Node's logger: JSON records fanned out to stderr,
// to a size-rotated file and to the viewer's ring, at the level, rotation
// and ring size its log section sets.
package logs

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Log struct {
	*slog.Logger
	dir   string
	level *slog.LevelVar
	file  sink
	ring  *ring
}

// sink is the log file.
type sink interface {
	io.WriteCloser
	// rotate takes s's rotation for the records after it.
	rotate(s Settings) error
}

const File = "slipstream.log"

// Open opens the log in dir, rotated as s says, creating dir with mode
// 0700 if it is missing.
func Open(dir string, stderr io.Writer, s Settings) (*Log, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create logs dir: %w", err)
	}
	r := &rotating{path: filepath.Join(dir, File)}
	if err := r.rotate(s); err != nil {
		return nil, err
	}
	return newLog(dir, stderr, r, s), nil
}

// OpenFile opens one log file in dir that never rotates. Rotation runs a
// goroutine that a testing/synctest bubble never sees end.
func OpenFile(dir string, stderr io.Writer, s Settings) (*Log, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create logs dir: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(dir, File), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // dir is the Node's own logs dir
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	return newLog(dir, stderr, plain{f}, s), nil
}

func newLog(dir string, stderr io.Writer, file sink, s Settings) *Log {
	level := new(slog.LevelVar)
	level.Set(levels[s.Level])
	opts := &slog.HandlerOptions{Level: level}
	r := newRing(s.ViewerLines)
	h := slog.NewMultiHandler(slog.NewJSONHandler(stderr, opts), slog.NewJSONHandler(file, opts), newRingHandler(r, level))
	return &Log{Logger: slog.New(h), dir: dir, level: level, file: file, ring: r}
}

// Apply sets the level, the rotation and the ring's size. An unchanged
// rotation keeps the open file, and a smaller ring drops the oldest
// records.
func (l *Log) Apply(s Settings) error {
	l.level.Set(levels[s.Level])
	l.ring.resize(s.ViewerLines)
	return l.file.rotate(s)
}

func (l *Log) Close() error {
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("close log file: %w", err)
	}
	return nil
}

type plain struct{ *os.File }

func (plain) rotate(Settings) error { return nil }

type rotation struct {
	maxSizeMB, maxFiles, maxAgeDays int
	compress                        bool
}

// rotating is a lumberjack file. A change of rotation replaces the whole
// lumberjack.Logger under mu, since its rotation goroutine reads the fields
// without a lock. Each replaced Logger leaves that goroutine behind.
type rotating struct {
	path string

	mu  sync.Mutex
	set rotation
	lj  *lumberjack.Logger
}

func (r *rotating) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, err := r.lj.Write(p)
	if err != nil {
		return n, fmt.Errorf("write log file: %w", err)
	}
	return n, nil
}

func (r *rotating) rotate(s Settings) error {
	set := rotation{maxSizeMB: s.MaxSizeMB, maxFiles: s.MaxFiles, maxAgeDays: s.MaxAgeDays, compress: s.Compress}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.lj != nil && set == r.set {
		return nil
	}
	old := r.lj
	r.set = set
	r.lj = &lumberjack.Logger{Filename: r.path, MaxSize: set.maxSizeMB, MaxBackups: set.maxFiles, MaxAge: set.maxAgeDays, Compress: set.compress}
	if old == nil {
		return nil
	}
	if err := old.Close(); err != nil {
		return fmt.Errorf("close log file: %w", err)
	}
	return nil
}

func (r *rotating) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lj.Close() //nolint:wrapcheck // Log.Close wraps it
}

// Package logs builds one Node's logger: JSON records fanned out to stderr
// and to a size-rotated file, at a level the Node owns.
package logs

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Log struct {
	*slog.Logger
	Level *slog.LevelVar
	file  io.Closer
}

// Open creates dir with mode 0700 if it is missing. Rotation matches v1:
// 10 MB, 5 backups, 30 days, compressed.
func Open(dir string, stderr io.Writer) (*Log, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create logs dir: %w", err)
	}
	return New(stderr, &lumberjack.Logger{
		Filename:   filepath.Join(dir, File),
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}), nil
}

const File = "slipstream.log"

// OpenFile opens one log file in dir that never rotates. Rotation runs a
// goroutine that a testing/synctest bubble never sees end.
func OpenFile(dir string, stderr io.Writer) (*Log, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create logs dir: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(dir, File), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	return New(stderr, f), nil
}

// New fans records out to stderr and file, and Close closes file.
func New(stderr io.Writer, file io.WriteCloser) *Log {
	level := new(slog.LevelVar)
	opts := &slog.HandlerOptions{Level: level}
	h := slog.NewMultiHandler(slog.NewJSONHandler(stderr, opts), slog.NewJSONHandler(file, opts))
	return &Log{Logger: slog.New(h), Level: level, file: file}
}

func (l *Log) Close() error {
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("close log file: %w", err)
	}
	return nil
}

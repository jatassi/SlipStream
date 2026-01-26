package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps zerolog for application logging.
type Logger struct {
	zerolog.Logger
	rotator *lumberjack.Logger
}

// Config holds logger configuration.
type Config struct {
	Level      string
	Format     string // "console" or "json"
	Path       string // directory for log files
	MaxSizeMB  int    // max size in MB before rotation (default: 10)
	MaxBackups int    // max number of old log files to keep (default: 5)
	MaxAgeDays int    // max age in days to keep old files (default: 30)
	Compress   bool   // compress rotated files (default: true)
}

// IsDevBuild returns true if running via "go run" (development mode).
// This is detected by checking if the executable path contains "go-build",
// which is where Go compiles temporary binaries during "go run".
func IsDevBuild() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return strings.Contains(exe, "go-build")
}

// New creates a new logger instance.
// When running via "go run" (dev build), automatically uses debug level
// unless a more verbose level (trace) is explicitly configured.
func New(cfg Config) *Logger {
	var consoleOutput io.Writer

	if cfg.Format == "json" {
		consoleOutput = os.Stdout
	} else {
		consoleOutput = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	level := parseLevel(cfg.Level)

	// Auto-enable debug logging for dev builds (go run)
	if IsDevBuild() && level > zerolog.DebugLevel {
		level = zerolog.DebugLevel
	}

	var output io.Writer = consoleOutput
	var rotator *lumberjack.Logger

	if cfg.Path != "" {
		if err := os.MkdirAll(cfg.Path, 0755); err == nil {
			logPath := filepath.Join(cfg.Path, "slipstream.log")

			maxSize := cfg.MaxSizeMB
			if maxSize <= 0 {
				maxSize = 10
			}
			maxBackups := cfg.MaxBackups
			if maxBackups <= 0 {
				maxBackups = 5
			}
			maxAge := cfg.MaxAgeDays
			if maxAge <= 0 {
				maxAge = 30
			}
			compress := cfg.Compress

			rotator = &lumberjack.Logger{
				Filename:   logPath,
				MaxSize:    maxSize,
				MaxBackups: maxBackups,
				MaxAge:     maxAge,
				Compress:   compress,
				LocalTime:  true,
			}

			output = io.MultiWriter(consoleOutput, rotator)
		}
	}

	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &Logger{Logger: logger, rotator: rotator}
}

// Close closes the log file if one is open.
func (l *Logger) Close() error {
	if l.rotator != nil {
		return l.rotator.Close()
	}
	return nil
}

// parseLevel converts string level to zerolog.Level
func parseLevel(level string) zerolog.Level {
	switch level {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// With returns a new logger with additional context fields.
func (l *Logger) With() zerolog.Context {
	return l.Logger.With()
}

// WithComponent returns a new logger with component field.
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.Logger.With().Str("component", component).Logger(),
	}
}

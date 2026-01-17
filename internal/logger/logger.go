package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog for application logging.
type Logger struct {
	zerolog.Logger
}

// Config holds logger configuration.
type Config struct {
	Level  string
	Format string // "console" or "json"
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
	var output io.Writer

	if cfg.Format == "json" {
		output = os.Stdout
	} else {
		// Console format with colors
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	level := parseLevel(cfg.Level)

	// Auto-enable debug logging for dev builds (go run)
	if IsDevBuild() && level > zerolog.DebugLevel {
		level = zerolog.DebugLevel
	}

	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &Logger{Logger: logger}
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

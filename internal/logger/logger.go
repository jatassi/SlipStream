package logger

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog for application logging.
type Logger struct {
	zerolog.Logger
	logFile *os.File
}

// Config holds logger configuration.
type Config struct {
	Level  string
	Format string // "console" or "json"
	Path   string // directory for log files
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
	var logFile *os.File

	if cfg.Path != "" {
		if err := os.MkdirAll(cfg.Path, 0755); err == nil {
			logPath := filepath.Join(cfg.Path, "slipstream.log")
			f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				logFile = f
				output = io.MultiWriter(consoleOutput, f)
			}
		}
	}

	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &Logger{Logger: logger, logFile: logFile}
}

// Close closes the log file if one is open.
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
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

package logger

import (
	"fmt"
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
	rotator     *lumberjack.Logger
	broadcaster *LogBroadcaster
}

// Config holds logger configuration.
type Config struct {
	Level           string
	Format          string // "console" or "json"
	Path            string // directory for log files
	MaxSizeMB       int    // max size in MB before rotation (default: 10)
	MaxBackups      int    // max number of old log files to keep (default: 5)
	MaxAgeDays      int    // max age in days to keep old files (default: 30)
	Compress        bool   // compress rotated files (default: true)
	EnableStreaming bool   // enable log streaming with ring buffer
	BufferSize      int    // ring buffer size for recent logs (default: 1000)
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
func New(cfg *Config) *Logger {
	consoleOutput := newConsoleOutput(cfg.Format)
	level := effectiveLevel(cfg.Level)

	output := consoleOutput
	var rotator *lumberjack.Logger
	var logBroadcaster *LogBroadcaster

	if cfg.Path != "" {
		rotator, output = setupFileLogging(cfg, consoleOutput)
	}

	if cfg.EnableStreaming {
		logBroadcaster = NewLogBroadcaster(nil, cfg.BufferSize)
		output = io.MultiWriter(output, logBroadcaster)
	}

	logger := zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &Logger{Logger: logger, rotator: rotator, broadcaster: logBroadcaster}
}

func newConsoleOutput(format string) io.Writer {
	if format == "json" {
		return os.Stdout
	}
	return zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
}

func effectiveLevel(levelStr string) zerolog.Level {
	level := parseLevel(levelStr)
	if IsDevBuild() && level > zerolog.DebugLevel {
		level = zerolog.DebugLevel
	}
	return level
}

func setupFileLogging(cfg *Config, consoleOutput io.Writer) (*lumberjack.Logger, io.Writer) {
	if err := os.MkdirAll(cfg.Path, 0o750); err != nil {
		bootstrapLogError("Failed to create log directory", cfg.Path, err)
		return nil, consoleOutput
	}

	rotator := &lumberjack.Logger{
		Filename:   filepath.Join(cfg.Path, "slipstream.log"),
		MaxSize:    positiveOrDefault(cfg.MaxSizeMB, 10),
		MaxBackups: positiveOrDefault(cfg.MaxBackups, 5),
		MaxAge:     positiveOrDefault(cfg.MaxAgeDays, 30),
		Compress:   cfg.Compress,
		LocalTime:  true,
	}

	fileWriter := zerolog.ConsoleWriter{
		Out:        rotator,
		TimeFormat: time.RFC3339,
		NoColor:    true,
	}

	if isValidStdout() {
		return rotator, io.MultiWriter(consoleOutput, fileWriter)
	}
	return rotator, fileWriter
}

func positiveOrDefault(val, defaultVal int) int {
	if val <= 0 {
		return defaultVal
	}
	return val
}

// Close closes the log file if one is open.
func (l *Logger) Close() error {
	if l.rotator != nil {
		return l.rotator.Close()
	}
	return nil
}

// GetRecentLogs returns buffered log entries from the broadcaster.
func (l *Logger) GetRecentLogs() []LogEntry {
	if l.broadcaster == nil {
		return nil
	}
	return l.broadcaster.GetRecentLogs()
}

// GetLogFilePath returns the path to the current log file, if any.
func (l *Logger) GetLogFilePath() string {
	if l.rotator == nil {
		return ""
	}
	return l.rotator.Filename
}

// SetBroadcastHub sets the hub for broadcasting log entries via WebSocket.
// This should be called after the hub is created to enable real-time streaming.
func (l *Logger) SetBroadcastHub(hub Broadcaster) {
	if l.broadcaster != nil {
		l.broadcaster.SetHub(hub)
	}
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

// bootstrapLogError writes an error to a temporary log file for early diagnostics
func bootstrapLogError(msg, path string, err error) {
	// Try to write to a fallback location for diagnostics
	var logPath string
	if home, homeErr := os.UserHomeDir(); homeErr == nil {
		logPath = filepath.Join(home, "slipstream_bootstrap_error.log")
	} else {
		logPath = "slipstream_bootstrap_error.log"
	}
	f, openErr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if openErr != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "[%s] %s: path=%s error=%v\n", time.Now().Format(time.RFC3339), msg, path, err)
}

// isValidStdout checks if stdout is a valid file descriptor
// On Windows GUI apps (non-console), stdout may not be valid
func isValidStdout() bool {
	// Try to get file info for stdout
	_, err := os.Stdout.Stat()
	return err == nil
}

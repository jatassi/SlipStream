package logs

import (
	"fmt"
	"log/slog"
)

// Settings is the log section: the level and the file's rotation.
type Settings struct {
	Level     Level `yaml:"level"`
	MaxSizeMB int   `yaml:"max_size_mb"`
	// MaxFiles of 0 keeps every rotated file.
	MaxFiles int `yaml:"max_files"`
	// MaxAgeDays of 0 keeps rotated files of any age.
	MaxAgeDays  int  `yaml:"max_age_days"`
	Compress    bool `yaml:"compress"`
	ViewerLines int  `yaml:"viewer_lines"`
}

func DefaultSettings() Settings {
	return Settings{Level: Info, MaxSizeMB: 10, MaxFiles: 5, MaxAgeDays: 30, Compress: true, ViewerLines: 2000}
}

// Level is the least severe record the log keeps.
type Level string

const (
	Debug Level = "debug"
	Info  Level = "info"
	Warn  Level = "warn"
	Error Level = "error"
)

var levels = map[Level]slog.Level{Debug: slog.LevelDebug, Info: slog.LevelInfo, Warn: slog.LevelWarn, Error: slog.LevelError}

func (l Level) MarshalText() ([]byte, error) { return []byte(l), nil }

func (l *Level) UnmarshalText(b []byte) error {
	if _, ok := levels[Level(b)]; !ok {
		return fmt.Errorf("%q is not debug, info, warn or error", b)
	}
	*l = Level(b)
	return nil
}

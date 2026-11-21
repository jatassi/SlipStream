package backups

import (
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/backups/archive"
)

// Settings is the Controller's Backups section of config.yaml. The hook
// reads it before library.db opens, so it can't be a stored setting.
type Settings struct {
	// Folder is where the Controller keeps its Backups. A relative folder
	// is in the data dir.
	Folder string `yaml:"folder"`
}

func DefaultSettings() Settings { return Settings{Folder: "backups"} }

// Timings is the archive's fixed encoding and the Windows retry.
type Timings struct {
	// Scrypt is the age work factor every new Backup is encrypted with.
	Scrypt int `yaml:"scrypt"`
	// ZstdWindow is the compression window in bytes, at most what the zstd
	// CLI opens by default.
	ZstdWindow int `yaml:"zstd_window"`
	// ZstdMemory bounds the decoder's memory in bytes.
	ZstdMemory int64 `yaml:"zstd_memory"`
	// WindowsRetry is how long a rename or delete on Windows retries a
	// sharing violation: an antivirus scan holds a file it has just seen.
	WindowsRetry time.Duration `yaml:"windows_retry"`
}

func DefaultTimings() Timings {
	return Timings{Scrypt: 18, ZstdWindow: 8 << 20, ZstdMemory: 64 << 20, WindowsRetry: 2 * time.Second}
}

func (t Timings) Check(fail func(key string, err error)) {
	if t.Scrypt < 15 || t.Scrypt > 22 {
		fail("timings.backups.scrypt", fmt.Errorf("%d isn't between 15 and 22", t.Scrypt))
	}
	if t.ZstdWindow < 1<<20 || t.ZstdWindow > 128<<20 {
		fail("timings.backups.zstd_window", fmt.Errorf("%d isn't between 1 MiB and 128 MiB", t.ZstdWindow))
	}
	if t.ZstdMemory < 16<<20 || t.ZstdMemory > 512<<20 {
		fail("timings.backups.zstd_memory", fmt.Errorf("%d isn't between 16 MiB and 512 MiB", t.ZstdMemory))
	}
	if t.WindowsRetry < 0 || t.WindowsRetry > 30*time.Second {
		fail("timings.backups.windows_retry", fmt.Errorf("%s isn't between 0s and 30s", t.WindowsRetry))
	}
}

func (t Timings) params() archive.Params {
	return archive.Params{ScryptLog: t.Scrypt, ZstdWindow: t.ZstdWindow}
}

func (t Timings) limits(library int64) archive.Limits {
	return archive.Limits{Library: library, ZstdWindow: t.ZstdWindow, ZstdMemory: t.ZstdMemory}
}

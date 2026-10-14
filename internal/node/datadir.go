package node

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

// maxSocketPath is the smallest sun_path limit across the targets, macOS's,
// so a data dir valid on Linux is valid everywhere.
const maxSocketPath = 103

// DataDir is a Node's home: absolute and clean, made only by ParseDataDir.
type DataDir struct{ path string }

func ParseDataDir(s string) (DataDir, error) {
	if s == "" {
		return DataDir{}, errors.New("data dir is empty")
	}
	abs, err := filepath.Abs(s)
	if err != nil {
		return DataDir{}, fmt.Errorf("data dir %s: %w", s, err)
	}
	d := DataDir{abs}
	if n := len(d.ControlSocket()); n > maxSocketPath {
		return DataDir{}, fmt.Errorf("data dir %s: control socket path %s is %d bytes, over the %d-byte limit", abs, d.ControlSocket(), n, maxSocketPath)
	}
	return d, nil
}

func (d DataDir) Path() string { return d.path }

func (d DataDir) String() string { return d.path }

func (d DataDir) RunDir() string { return filepath.Join(d.path, "run") }

func (d DataDir) ControlSocket() string { return filepath.Join(d.RunDir(), "control.sock") }

func (d DataDir) LogsDir() string { return filepath.Join(d.path, "logs") }

func (d DataDir) LibraryDB() string { return filepath.Join(d.path, storage.LibraryFile) }

func (d DataDir) LibraryKey() string { return filepath.Join(d.path, secret.KeyFile) }

func (d DataDir) LocalDB() string { return filepath.Join(d.path, storage.LocalFile) }

// Resolve is p against the data dir, unless p is absolute.
func (d DataDir) Resolve(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(d.path, p)
}

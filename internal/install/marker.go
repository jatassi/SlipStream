package install

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// MarkerName is the marker file: one token, optionally followed by one
// newline, beside the binary it describes.
const MarkerName = "install-kind"

// MarkerPath is where l's executable keeps its marker: beside it, or in
// Contents/Resources for a macOS bundle, so the bundle signature covers it.
func MarkerPath(l Layout) string {
	dir := l.dir(l.Executable)
	if l.GOOS == "darwin" && l.base(dir) == "MacOS" && l.base(l.dir(dir)) == "Contents" {
		return l.join(l.dir(dir), "Resources", MarkerName)
	}
	return l.join(dir, MarkerName)
}

// Read is the Kind l's marker states, or Dev when there is none. A marker
// it cannot read, an unknown token or another OS's Kind is an error naming
// the marker, since a damaged install is reported, never guessed around.
func Read(l Layout) (Kind, error) {
	p := MarkerPath(l)
	b, err := os.ReadFile(p) //nolint:gosec // G304: the marker sits beside the running binary
	if errors.Is(err, fs.ErrNotExist) {
		return Dev, nil
	}
	if err != nil {
		return Dev, fmt.Errorf("install marker %s: %w", p, err)
	}
	k, ok := parseToken(strings.TrimSuffix(string(b), "\n"))
	if !ok {
		return Dev, fmt.Errorf("install marker %s holds %q, not an install kind", p, b)
	}
	if k.GOOS() != "" && k.GOOS() != l.GOOS {
		return Dev, fmt.Errorf("install marker %s holds %q, a %s install kind, on %s", p, b, k.GOOS(), l.GOOS)
	}
	return k, nil
}

// Write places k's marker at path, which MarkerPath names for the binary
// it describes.
func Write(path string, k Kind) error {
	if err := os.WriteFile(path, []byte(k.String()+"\n"), 0o644); err != nil { //nolint:gosec // G306: the marker is world-readable like the binary beside it
		return fmt.Errorf("write install marker: %w", err)
	}
	return nil
}

// Package buildinfo holds the build stamp the release tool links into the binary.
package buildinfo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
)

// Stamp names exactly what a release binary was built from.
type Stamp struct {
	Version      string `json:"version"`
	Commit       string `json:"commit"`
	Fingerprint  string `json:"fingerprint"`
	Target       string `json:"target"`
	NativeBundle string `json:"nativeBundle"`
	Libtorrent   string `json:"libtorrent"`
}

const magic = "slipstream-stamp:"

// ErrUnstamped reports a binary built without the release tool's stamp.
var ErrUnstamped = errors.New("binary carries no build stamp")

var stamp string

// Encode returns the value for -X github.com/jatassi/slipstream/internal/buildinfo.stamp.
// The go command splits -ldflags at whitespace, so no field may contain any.
func Encode(s Stamp) (string, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("encode the stamp: %w", err)
	}
	if strings.ContainsFunc(string(payload), unicode.IsSpace) {
		return "", fmt.Errorf("stamp %s contains whitespace", payload)
	}
	return magic + string(payload), nil
}

// Current returns the running binary's stamp, or false for an unstamped dev build.
func Current() (Stamp, bool) {
	if stamp == "" {
		return Stamp{}, false
	}
	var s Stamp
	if err := json.Unmarshal([]byte(strings.TrimPrefix(stamp, magic)), &s); err != nil {
		panic("buildinfo: malformed stamp: " + err.Error())
	}
	return s, true
}

// InstallKind is how this binary was installed. Every build reports DEV
// until #181's install-kind marker exists.
func InstallKind() linkv1.InstallKind { return linkv1.InstallKind_INSTALL_KIND_DEV }

// ReadFile reads the stamp from a binary file without running it.
func ReadFile(path string) (Stamp, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: reading the binary the caller names is this function's job
	if err != nil {
		return Stamp{}, fmt.Errorf("read the stamp: %w", err)
	}
	for {
		i := bytes.Index(data, []byte(magic))
		if i < 0 {
			return Stamp{}, ErrUnstamped
		}
		data = data[i+len(magic):]
		var s Stamp
		if json.NewDecoder(bytes.NewReader(data)).Decode(&s) == nil {
			return s, nil
		}
	}
}

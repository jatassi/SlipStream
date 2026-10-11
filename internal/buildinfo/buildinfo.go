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
		return "", err
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

// ReadFile reads the stamp from a binary file without running it.
func ReadFile(path string) (Stamp, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Stamp{}, err
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

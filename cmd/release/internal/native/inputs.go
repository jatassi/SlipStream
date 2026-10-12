// Package native holds the native bundle's inputs, its layer digests, the
// lock that pins published layers, the per-machine layer cache and the cgo
// environment a layer provides.
package native

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
)

// Gate says which CI runs build a target: every PR, or only main, nightly,
// release and PRs that touch native code.
type Gate string

const (
	GatePR     Gate = "pr"
	GateNative Gate = "native"
)

type Target struct {
	Name      string `json:"name"`
	GOOS      string `json:"goos"`
	GOARCH    string `json:"goarch"`
	Runner    string `json:"runner"`
	Toolchain string `json:"toolchain"`
	MinOS     string `json:"min_os"`
	Gate      Gate   `json:"gate"`
	// Layers are the bundle layers published and pinned for the target.
	Layers []Kind `json:"layers"`
	// OpenSSLTarget is the target name OpenSSL's Configure takes.
	OpenSSLTarget string `json:"openssl_target"`
	// SystemLibs are linker arguments after the static libraries.
	SystemLibs []string `json:"system_libs"`
}

type Component struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
	License string `json:"license"`
}

type Inputs struct {
	Targets    []Target    `json:"targets"`
	Components []Component `json:"components"`
}

func LoadInputs(path string) (Inputs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Inputs{}, err
	}
	return ParseInputs(data)
}

func ParseInputs(data []byte) (Inputs, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var in Inputs
	if err := dec.Decode(&in); err != nil {
		return Inputs{}, fmt.Errorf("native inputs: %w", err)
	}
	for _, t := range in.Targets {
		if t.Gate != GatePR && t.Gate != GateNative {
			return Inputs{}, fmt.Errorf("native inputs: target %s: gate %q is neither %q nor %q", t.Name, t.Gate, GatePR, GateNative)
		}
		for _, k := range t.Layers {
			if _, ok := layerDigests[k]; !ok {
				return Inputs{}, fmt.Errorf("native inputs: target %s: unknown layer kind %q", t.Name, k)
			}
		}
	}
	for _, c := range in.Components {
		if b, err := hex.DecodeString(c.SHA256); err != nil || len(b) != 32 {
			return Inputs{}, fmt.Errorf("native inputs: component %s: sha256 %q is not 64 hex digits", c.Name, c.SHA256)
		}
	}
	return in, nil
}

func (in Inputs) Target(name string) (Target, error) {
	for _, t := range in.Targets {
		if t.Name == name {
			return t, nil
		}
	}
	return Target{}, fmt.Errorf("unknown target %q", name)
}

func (in Inputs) Component(name string) (Component, error) {
	for _, c := range in.Components {
		if c.Name == name {
			return c, nil
		}
	}
	return Component{}, fmt.Errorf("native inputs list no component %q", name)
}

func HostTargetName() string { return runtime.GOOS + "-" + runtime.GOARCH }

// Toolchain is the C and C++ compiler a layer is built with and that cgo
// uses against it.
type Toolchain struct {
	CC, CXX string
}

// toolchains maps a target's toolchain name to its compilers. host-gcc is
// the host's glibc gcc, used until the musl toolchain layer exists.
var toolchains = map[string]Toolchain{
	"host-gcc": {CC: "gcc", CXX: "g++"},
}

func (t Target) Compilers() (Toolchain, error) {
	tc, ok := toolchains[t.Toolchain]
	if !ok {
		return Toolchain{}, fmt.Errorf("target %s: toolchain %q is not available yet", t.Name, t.Toolchain)
	}
	return tc, nil
}

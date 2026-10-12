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
	"slices"
)

// Gate says which CI runs build a target: every PR, or only main, nightly,
// release and PRs that touch native code.
type Gate string

const (
	GatePR     Gate = "pr"
	GateNative Gate = "native"
)

type Target struct {
	Name   string `json:"name"`
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	Runner string `json:"runner"`
	// Toolchain names the entry of Inputs.Toolchains that compiles the
	// target's libs layer and its cgo packages.
	Toolchain string `json:"toolchain"`
	// Triple is the target triple the toolchain compiles for.
	Triple string `json:"triple,omitempty"`
	MinOS  string `json:"min_os"`
	Gate   Gate   `json:"gate"`
	// GoTags are build tags every build against the target's bundle needs.
	GoTags []string `json:"go_tags,omitempty"`
	// Env is the environment every C and C++ compile for the target runs
	// in, both in the libs layer build and in cgo.
	Env map[string]string `json:"env,omitempty"`
	// OpenSSLTarget is the target name OpenSSL's Configure takes.
	OpenSSLTarget string `json:"openssl_target"`
	// SystemLibs are linker arguments after the static libraries.
	SystemLibs []string `json:"system_libs"`
}

// Source is a pinned source archive a recipe builds from.
type Source struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

// Component is a source the libs layer builds and the binary links.
type Component struct {
	Source
	License string `json:"license"`
}

// Runtime is a library a toolchain links statically into every binary of
// the targets that use it, shipped from one of the toolchain's sources.
type Runtime struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	License string `json:"license"`
}

// Toolchain is the C and C++ compiler a target builds with. A toolchain
// with sources is a layer its recipe, native/recipe/toolchain/<name>,
// builds or mirrors from them. A toolchain with an Xcode version is the
// runner's own Xcode at that version and has no layer.
type Toolchain struct {
	Name     string    `json:"name"`
	Sources  []Source  `json:"sources,omitempty"`
	Runtimes []Runtime `json:"runtimes,omitempty"`
	Xcode    string    `json:"xcode,omitempty"`
}

func (tc Toolchain) Layered() bool { return tc.Xcode == "" }

type Inputs struct {
	Targets    []Target    `json:"targets"`
	Toolchains []Toolchain `json:"toolchains"`
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
		if _, err := in.Toolchain(t); err != nil {
			return Inputs{}, fmt.Errorf("native inputs: %w", err)
		}
	}
	for _, tc := range in.Toolchains {
		if tc.Layered() == (len(tc.Sources) == 0) {
			return Inputs{}, fmt.Errorf("native inputs: toolchain %s: lists either sources or an Xcode version, not both or neither", tc.Name)
		}
	}
	var sources []Source
	for _, c := range in.Components {
		sources = append(sources, c.Source)
	}
	for _, tc := range in.Toolchains {
		sources = append(sources, tc.Sources...)
		for _, r := range tc.Runtimes {
			if !slices.ContainsFunc(tc.Sources, func(s Source) bool { return s.Name == r.Source }) {
				return Inputs{}, fmt.Errorf("native inputs: toolchain %s: runtime %s names no source %q", tc.Name, r.Name, r.Source)
			}
		}
	}
	for _, s := range sources {
		if b, err := hex.DecodeString(s.SHA256); err != nil || len(b) != 32 {
			return Inputs{}, fmt.Errorf("native inputs: source %s: sha256 %q is not 64 hex digits", s.Name, s.SHA256)
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

func (in Inputs) Toolchain(t Target) (Toolchain, error) {
	for _, tc := range in.Toolchains {
		if tc.Name == t.Toolchain {
			return tc, nil
		}
	}
	return Toolchain{}, fmt.Errorf("target %s: toolchain %q is not listed", t.Name, t.Toolchain)
}

func (tc Toolchain) Source(name string) Source {
	i := slices.IndexFunc(tc.Sources, func(s Source) bool { return s.Name == name })
	return tc.Sources[i]
}

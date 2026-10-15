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

// Tool is a prebuilt program the release ships beside the binary and runs
// as a separate process. Each target fetches its own published archive of
// it, which holds Programs at its root.
type Tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	License string `json:"license"`
	// Source is where the tool's source for Version is published, for the
	// notices.
	Source   string   `json:"source"`
	Programs []string `json:"programs"`
	// Archives maps each target name to the URL of its archive.
	Archives map[string]string `json:"archives"`
	// SourceArchive is the published source of Version, which a release
	// ships beside its packages to meet a copyleft license's source offer.
	SourceArchive *Download `json:"source_archive,omitempty"`
	// Debian is the tool as Debian packages, which the Docker image
	// installs in place of the archive.
	Debian *DebianTool `json:"debian,omitempty"`
}

// DebianTool is a tool's Debian package for each Linux target, which
// installs Programs into Dir.
type DebianTool struct {
	Dir      string              `json:"dir"`
	Packages map[string]Download `json:"packages"`
}

// Download is a file pinned by its content.
type Download struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// Packager is a program the release runs to build a package, never ships.
type Packager struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Target  string `json:"target"`
	Download
}

type Inputs struct {
	Targets    []Target    `json:"targets"`
	Toolchains []Toolchain `json:"toolchains"`
	Components []Component `json:"components"`
	Tools      []Tool      `json:"tools"`
	Packagers  []Packager  `json:"packagers"`
}

func LoadInputs(path string) (Inputs, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Inputs{}, fmt.Errorf("read the native inputs: %w", err)
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
	if err := checkDigests(sources); err != nil {
		return Inputs{}, err
	}
	if err := in.checkTools(); err != nil {
		return Inputs{}, err
	}
	if err := in.checkDownloads(); err != nil {
		return Inputs{}, err
	}
	return in, nil
}

func (in Inputs) checkDownloads() error {
	var downloads []Source
	for _, tool := range in.Tools {
		if tool.SourceArchive != nil {
			downloads = append(downloads, Source{Name: tool.Name + " source", URL: tool.SourceArchive.URL, SHA256: tool.SourceArchive.SHA256})
		}
		if tool.Debian == nil {
			continue
		}
		for target, d := range tool.Debian.Packages {
			if _, err := in.Target(target); err != nil {
				return fmt.Errorf("native inputs: tool %s: Debian package for %w", tool.Name, err)
			}
			downloads = append(downloads, Source{Name: tool.Name + " Debian package for " + target, URL: d.URL, SHA256: d.SHA256})
		}
	}
	for _, p := range in.Packagers {
		if _, err := in.Target(p.Target); err != nil {
			return fmt.Errorf("native inputs: packager %s: %w", p.Name, err)
		}
		downloads = append(downloads, Source{Name: p.Name, URL: p.URL, SHA256: p.SHA256})
	}
	return checkDigests(downloads)
}

func (in Inputs) Tool(name string) (Tool, error) {
	for _, t := range in.Tools {
		if t.Name == name {
			return t, nil
		}
	}
	return Tool{}, fmt.Errorf("native inputs list no tool %q", name)
}

func (in Inputs) Packager(name, target string) (Packager, error) {
	for _, p := range in.Packagers {
		if p.Name == name && p.Target == target {
			return p, nil
		}
	}
	return Packager{}, fmt.Errorf("native inputs list no packager %s for %s", name, target)
}

func (in Inputs) checkTools() error {
	for _, tool := range in.Tools {
		for _, t := range in.Targets {
			if _, ok := tool.Archives[t.Name]; !ok {
				return fmt.Errorf("native inputs: tool %s: no archive for target %s", tool.Name, t.Name)
			}
		}
		for name := range tool.Archives {
			if _, err := in.Target(name); err != nil {
				return fmt.Errorf("native inputs: tool %s: archive for %w", tool.Name, err)
			}
		}
	}
	return nil
}

func checkDigests(sources []Source) error {
	for _, s := range sources {
		if b, err := hex.DecodeString(s.SHA256); err != nil || len(b) != 32 {
			return fmt.Errorf("native inputs: source %s: sha256 %q is not 64 hex digits", s.Name, s.SHA256)
		}
	}
	return nil
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

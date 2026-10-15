package nativetouch

import (
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func file(path, content string) Side { return Side{path, []byte(content)} }

func modified(path, before, after string) Change {
	return Change{file(path, before), file(path, after)}
}

const goMod = "module github.com/jatassi/slipstream\n\ngo 1.27.2\n"

func TestMatches(t *testing.T) {
	tests := []struct {
		name    string
		changes []Change
		want    []Match
	}{
		{
			name:    "docs only",
			changes: []Change{modified("docs/release.md", "a", "b"), {New: file("README.md", "x")}},
		},
		{
			name:    "deleted shim source",
			changes: []Change{{Old: file("internal/torrent/libtorrent/shim.cpp", "int x;")}},
			want: []Match{
				{"native-root", "internal/torrent/libtorrent/shim.cpp"},
				{"c-family", "internal/torrent/libtorrent/shim.cpp"},
			},
		},
		{
			name:    "rename out of the native roots",
			changes: []Change{{Old: file("native/recipe/boost.cmake", "x"), New: file("docs/boost.cmake", "x")}},
			want:    []Match{{"native-root", "native/recipe/boost.cmake"}},
		},
		{
			name:    "modified release tool file matches once",
			changes: []Change{modified("cmd/release/plan.go", "package main\n", "package main\n\n")},
			want:    []Match{{"native-root", "cmd/release/plan.go"}},
		},
		{
			name:    "workflow change",
			changes: []Change{modified(".github/workflows/ci.yml", "a", "b")},
			want:    []Match{{"native-root", ".github/workflows/ci.yml"}},
		},
		{
			name: "data dir lock and control socket packages",
			changes: []Change{
				modified("internal/node/internal/lock/lock.go", "package lock\n", "package lock\n\n"),
				{New: file("internal/node/internal/control/control.go", "package control\n")},
				{New: file("internal/node/internal/config/config.go", "package config\n")},
				{New: file("internal/node/node.go", "package node\n")},
			},
			want: []Match{
				{"native-root", "internal/node/internal/lock/lock.go"},
				{"native-root", "internal/node/internal/control/control.go"},
			},
		},
		{
			name: "tray and trampoline",
			changes: []Change{
				modified("internal/node/internal/desktop/tray.go", "package desktop\n", "package desktop\n\n"),
				modified("internal/node/app.go", "package node\n", "package node\n\n"),
				{New: file("cmd/slipstream-app/main.go", "package main\n")},
				{New: file("internal/node/status.go", "package node\n")},
			},
			want: []Match{
				{"native-root", "internal/node/internal/desktop/tray.go"},
				{"native-root", "internal/node/app.go"},
				{"native-root", "cmd/slipstream-app/main.go"},
			},
		},
		{
			name:    "header outside the native roots",
			changes: []Change{{New: file("web/vendor/thing.h", "")}},
			want:    []Match{{"c-family", "web/vendor/thing.h"}},
		},
		{
			name: "os and arch filename suffixes",
			changes: []Change{
				{New: file("internal/media/probe_windows.go", "package media\n")},
				{Old: file("internal/media/probe_darwin_arm64_test.go", "package media\n")},
				modified("internal/media/probe_amd64.go", "package media\n", "package media\n\n"),
			},
			want: []Match{
				{"go-os-arch-suffix", "internal/media/probe_windows.go"},
				{"go-os-arch-suffix", "internal/media/probe_darwin_arm64_test.go"},
				{"go-os-arch-suffix", "internal/media/probe_amd64.go"},
			},
		},
		{
			name: "names that are not suffixes",
			changes: []Change{
				{New: file("internal/media/windows.go", "package media\n")},
				{New: file("internal/media/probe_windows.txt", "")},
				{New: file("internal/media/windows_probe.go", "package media\n")},
			},
		},
		{
			name: "build constraint naming an os",
			changes: []Change{
				{New: file("internal/media/probe.go", "// Copyright\n\n//go:build darwin && cgo\n\npackage media\n")},
				modified("internal/media/open.go", "//go:build !unix\n\npackage media\n", "package media\n"),
				{New: file("internal/media/wide.go", "//go:build integration || arm64\n\npackage media\n")},
			},
			want: []Match{
				{"go-os-arch-constraint", "internal/media/probe.go"},
				{"go-os-arch-constraint", "internal/media/open.go"},
				{"go-os-arch-constraint", "internal/media/wide.go"},
			},
		},
		{
			name: "build constraints naming no os or arch, or after the package clause",
			changes: []Change{
				{New: file("internal/media/slow.go", "//go:build integration\n\npackage media\n")},
				{New: file("internal/media/late.go", "package media\n\n//go:build linux\n")},
				{New: file("internal/media/notes.md", "//go:build linux\n")},
			},
		},
		{
			name: "go.mod dependency bump",
			changes: []Change{modified("go.mod", goMod,
				goMod+"\nrequire golang.org/x/mod v0.30.0 // indirect\n")},
		},
		{
			name:    "go.mod go directive bump",
			changes: []Change{modified("go.mod", goMod, strings.Replace(goMod, "1.27.2", "1.27.3", 1))},
			want:    []Match{{"go-directive", "go.mod"}},
		},
		{
			name:    "go.mod toolchain directive added",
			changes: []Change{modified("go.mod", goMod, goMod+"\ntoolchain go1.27.3\n")},
			want:    []Match{{"go-directive", "go.mod"}},
		},
		{
			name:    "nested go.mod is not the module's",
			changes: []Change{modified("tools/go.mod", goMod, strings.Replace(goMod, "1.27.2", "1.28.0", 1))},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Matches(tt.changes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Matches = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKnownPortsCoverDistList(t *testing.T) {
	out, err := exec.CommandContext(t.Context(), "go", "tool", "dist", "list").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, port := range strings.Fields(string(out)) {
		goos, goarch, _ := strings.Cut(port, "/")
		if !knownOS[goos] || !knownArch[goarch] {
			t.Errorf("go tool dist list has %s, which knownOS and knownArch lack", port)
		}
	}
}

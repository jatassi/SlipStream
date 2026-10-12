package native

import (
	"maps"
	"slices"
	"testing"
	"testing/fstest"
)

const fixtureInputs = `{
	"targets": [
		{"name": "linux-amd64", "goos": "linux", "goarch": "amd64", "runner": "ubuntu-24.04", "toolchain": "musl-cross",
		 "triple": "x86_64-linux-musl", "min_os": "linux 3.2", "gate": "pr",
		 "go_tags": ["netgo"], "openssl_target": "linux-x86_64", "system_libs": ["-static", "-lstdc++"]},
		{"name": "darwin-arm64", "goos": "darwin", "goarch": "arm64", "runner": "macos-15", "toolchain": "xcode",
		 "min_os": "macos 13.0", "gate": "native", "env": {"MACOSX_DEPLOYMENT_TARGET": "13.0"},
		 "openssl_target": "darwin64-arm64-cc", "system_libs": ["-lc++"]}
	],
	"toolchains": [
		{"name": "musl-cross",
		 "sources": [{"name": "musl", "version": "1.2.6", "url": "https://example.test/musl.tar.gz",
		              "sha256": "d585fd3b613c66151fc3249e8ed44f77020cb5e6c1e635a616d3f9f82460512a"}],
		 "runtimes": [{"name": "musl", "source": "musl", "license": "MIT"}]},
		{"name": "xcode", "xcode": "16.4"}
	],
	"components": [
		{"name": "libtorrent-rasterbar", "version": "2.0.15", "url": "https://example.test/lt.tar.gz",
		 "sha256": "5e2e79129823b7ea48721164c32b5aaf83d3fd733b5502100f6705b29f27bb02", "license": "BSD-3-Clause"}
	]
}`

func fixture(t *testing.T) (Inputs, fstest.MapFS) {
	t.Helper()
	in, err := ParseInputs([]byte(fixtureInputs))
	if err != nil {
		t.Fatal(err)
	}
	recipe := fstest.MapFS{
		"toolchain/musl-cross/toolchain.cmake": {Data: []byte("execute_process()\n")},
		"libs/CMakeLists.txt":                  {Data: []byte("project(x)\n")},
		"libs/metadata.cmake":                  {Data: []byte("file(WRITE x)\n")},
	}
	return in, recipe
}

// digests maps each layer kind of every target to its digest.
func digests(t *testing.T, in Inputs, recipe fstest.MapFS) map[LayerID]string {
	t.Helper()
	ids, err := Want(in, recipe)
	if err != nil {
		t.Fatal(err)
	}
	got := map[LayerID]string{}
	for _, id := range ids {
		got[LayerID{Kind: id.Kind, Target: id.Target}] = id.Digest
	}
	return got
}

var (
	linuxToolchain = LayerID{Kind: KindToolchain, Target: "linux-amd64"}
	linuxLibs      = LayerID{Kind: KindLibs, Target: "linux-amd64"}
	darwinLibs     = LayerID{Kind: KindLibs, Target: "darwin-arm64"}
)

func TestDigestsAreStable(t *testing.T) {
	in, recipe := fixture(t)
	got := digests(t, in, recipe)
	want := map[LayerID]string{
		linuxToolchain: "c871a31bd840d2719733f3d367edc4c1807ebcd6ba999b630294eb6bab409d39",
		linuxLibs:      "5198b18496aea3325f343b78e62f606383025afa4445da5ef2a971e1b13ae398",
		darwinLibs:     "8c0bac6bc6c590541977390e9181d8c0cd634f805c78e61568ea806710f41548",
	}
	if !maps.Equal(got, want) {
		t.Errorf("digests = %v, want %v", got, want)
	}
}

func TestDigestsCoverTheirInputs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		edit    func(*Inputs, fstest.MapFS)
		changes []LayerID
	}{
		{"toolchain recipe", func(_ *Inputs, r fstest.MapFS) {
			r["toolchain/musl-cross/toolchain.cmake"] = &fstest.MapFile{Data: []byte("execute_process(x)\n")}
		}, []LayerID{linuxToolchain, linuxLibs}},
		{"another toolchain's recipe", func(_ *Inputs, r fstest.MapFS) {
			r["toolchain/llvm-mingw-ucrt/toolchain.cmake"] = &fstest.MapFile{Data: []byte("file(DOWNLOAD)\n")}
		}, nil},
		{"toolchain source", func(in *Inputs, _ fstest.MapFS) { in.Toolchains[0].Sources[0].Version = "1.2.7" }, []LayerID{linuxToolchain, linuxLibs}},
		{"triple", func(in *Inputs, _ fstest.MapFS) { in.Targets[0].Triple = "x86_64-linux-musl2" }, []LayerID{linuxToolchain, linuxLibs}},
		{"runtime license", func(in *Inputs, _ fstest.MapFS) { in.Toolchains[0].Runtimes[0].License = "0BSD" }, nil},
		{"xcode version", func(in *Inputs, _ fstest.MapFS) { in.Toolchains[1].Xcode = "26.3" }, []LayerID{darwinLibs}},
		{"target env", func(in *Inputs, _ fstest.MapFS) { in.Targets[1].Env["MACOSX_DEPLOYMENT_TARGET"] = "14.0" }, []LayerID{darwinLibs}},
		{"libs recipe content", func(_ *Inputs, r fstest.MapFS) {
			r["libs/metadata.cmake"] = &fstest.MapFile{Data: []byte("file(WRITE y)\n")}
		}, []LayerID{linuxLibs, darwinLibs}},
		{"libs recipe file added", func(_ *Inputs, r fstest.MapFS) { r["libs/patches/fix.patch"] = &fstest.MapFile{Data: []byte("x")} }, []LayerID{linuxLibs, darwinLibs}},
		{"libs recipe file renamed", func(_ *Inputs, r fstest.MapFS) {
			r["libs/meta.cmake"] = r["libs/metadata.cmake"]
			delete(r, "libs/metadata.cmake")
		}, []LayerID{linuxLibs, darwinLibs}},
		{"component version", func(in *Inputs, _ fstest.MapFS) { in.Components[0].Version = "2.0.16" }, []LayerID{linuxLibs, darwinLibs}},
		{"component sha256", func(in *Inputs, _ fstest.MapFS) {
			in.Components[0].SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
		}, []LayerID{linuxLibs, darwinLibs}},
		{"own target row", func(in *Inputs, _ fstest.MapFS) { in.Targets[0].SystemLibs = nil }, []LayerID{linuxLibs}},
		{"another target row", func(in *Inputs, _ fstest.MapFS) { in.Targets[1].OpenSSLTarget = "darwin64-x86_64-cc" }, []LayerID{darwinLibs}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in, recipe := fixture(t)
			before := digests(t, in, recipe)
			tc.edit(&in, recipe)
			after := digests(t, in, recipe)
			for id := range before {
				changed, want := before[id] != after[id], slices.Contains(tc.changes, id)
				if changed != want {
					t.Errorf("%s %s digest changed = %v, want %v", id.Target, id.Kind, changed, want)
				}
			}
		})
	}
}

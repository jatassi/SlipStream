package native

import (
	"slices"
	"testing"
	"testing/fstest"
)

const fixtureInputs = `{
	"targets": [
		{"name": "linux-amd64", "goos": "linux", "goarch": "amd64", "runner": "ubuntu-24.04", "toolchain": "musl-cross",
		 "triple": "x86_64-linux-musl", "min_os": "linux 3.2", "gate": "pr", "layers": ["toolchain", "libs"],
		 "go_tags": ["netgo"], "openssl_target": "linux-x86_64", "system_libs": ["-static", "-lstdc++"]},
		{"name": "darwin-arm64", "goos": "darwin", "goarch": "arm64", "runner": "macos-15", "toolchain": "xcode",
		 "min_os": "macos 13.0", "gate": "native", "openssl_target": "darwin64-arm64-cc", "system_libs": ["-lc++"]}
	],
	"toolchains": [
		{"name": "musl-cross",
		 "sources": [{"name": "musl", "version": "1.2.6", "url": "https://example.test/musl.tar.gz",
		              "sha256": "d585fd3b613c66151fc3249e8ed44f77020cb5e6c1e635a616d3f9f82460512a"}],
		 "runtimes": [{"name": "musl", "source": "musl", "license": "MIT"}]}
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
		"toolchain/toolchain.cmake": {Data: []byte("execute_process()\n")},
		"libs/CMakeLists.txt":       {Data: []byte("project(x)\n")},
		"libs/metadata.cmake":       {Data: []byte("file(WRITE x)\n")},
	}
	return in, recipe
}

func digests(t *testing.T, in Inputs, recipe fstest.MapFS) map[Kind]string {
	t.Helper()
	tg, err := in.Target("linux-amd64")
	if err != nil {
		t.Fatal(err)
	}
	got := map[Kind]string{}
	for k, digest := range layerDigests {
		if got[k], err = digest(in, tg, recipe); err != nil {
			t.Fatal(err)
		}
	}
	return got
}

func TestDigestsAreStable(t *testing.T) {
	in, recipe := fixture(t)
	got := digests(t, in, recipe)
	want := map[Kind]string{
		KindToolchain: "c871a31bd840d2719733f3d367edc4c1807ebcd6ba999b630294eb6bab409d39",
		KindLibs:      "1c74e51413566aee8c4891038cc0675c4b45cff18264b40319bd460bfb5a15a7",
	}
	for k := range want {
		if got[k] != want[k] {
			t.Errorf("%s digest = %s, want %s", k, got[k], want[k])
		}
	}
}

func TestDigestsCoverTheirInputs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		edit    func(*Inputs, fstest.MapFS)
		changes []Kind
	}{
		{"toolchain recipe", func(_ *Inputs, r fstest.MapFS) {
			r["toolchain/toolchain.cmake"] = &fstest.MapFile{Data: []byte("execute_process(x)\n")}
		}, []Kind{KindToolchain, KindLibs}},
		{"toolchain source", func(in *Inputs, _ fstest.MapFS) { in.Toolchains[0].Sources[0].Version = "1.2.7" }, []Kind{KindToolchain, KindLibs}},
		{"triple", func(in *Inputs, _ fstest.MapFS) { in.Targets[0].Triple = "x86_64-linux-musl2" }, []Kind{KindToolchain, KindLibs}},
		{"runtime license", func(in *Inputs, _ fstest.MapFS) { in.Toolchains[0].Runtimes[0].License = "0BSD" }, nil},
		{"libs recipe content", func(_ *Inputs, r fstest.MapFS) {
			r["libs/metadata.cmake"] = &fstest.MapFile{Data: []byte("file(WRITE y)\n")}
		}, []Kind{KindLibs}},
		{"libs recipe file added", func(_ *Inputs, r fstest.MapFS) { r["libs/patches/fix.patch"] = &fstest.MapFile{Data: []byte("x")} }, []Kind{KindLibs}},
		{"libs recipe file renamed", func(_ *Inputs, r fstest.MapFS) {
			r["libs/meta.cmake"] = r["libs/metadata.cmake"]
			delete(r, "libs/metadata.cmake")
		}, []Kind{KindLibs}},
		{"component version", func(in *Inputs, _ fstest.MapFS) { in.Components[0].Version = "2.0.16" }, []Kind{KindLibs}},
		{"component sha256", func(in *Inputs, _ fstest.MapFS) {
			in.Components[0].SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
		}, []Kind{KindLibs}},
		{"own target row", func(in *Inputs, _ fstest.MapFS) { in.Targets[0].SystemLibs = nil }, []Kind{KindLibs}},
		{"another target row", func(in *Inputs, _ fstest.MapFS) { in.Targets[1].OpenSSLTarget = "darwin64-x86_64-cc" }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in, recipe := fixture(t)
			before := digests(t, in, recipe)
			tc.edit(&in, recipe)
			after := digests(t, in, recipe)
			for k := range layerDigests {
				changed, want := before[k] != after[k], slices.Contains(tc.changes, k)
				if changed != want {
					t.Errorf("%s digest changed = %v, want %v", k, changed, want)
				}
			}
		})
	}
}

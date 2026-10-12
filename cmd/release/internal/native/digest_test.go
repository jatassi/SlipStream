package native

import (
	"testing"
	"testing/fstest"
)

const fixtureInputs = `{
	"targets": [
		{"name": "linux-amd64", "goos": "linux", "goarch": "amd64", "runner": "ubuntu-24.04", "toolchain": "host-gcc",
		 "min_os": "linux 3.2", "gate": "pr", "layers": ["libs"], "openssl_target": "linux-x86_64", "system_libs": ["-lstdc++", "-lm"]},
		{"name": "darwin-arm64", "goos": "darwin", "goarch": "arm64", "runner": "macos-15", "toolchain": "xcode",
		 "min_os": "macos 13.0", "gate": "native", "openssl_target": "darwin64-arm64-cc", "system_libs": ["-lc++"]}
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
		"CMakeLists.txt": {Data: []byte("project(x)\n")},
		"metadata.cmake": {Data: []byte("file(WRITE x)\n")},
	}
	return in, recipe
}

func digest(t *testing.T, in Inputs, target string, recipe fstest.MapFS) string {
	t.Helper()
	tg, err := in.Target(target)
	if err != nil {
		t.Fatal(err)
	}
	d, err := LibsDigest(in, tg, recipe)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestLibsDigestIsStable(t *testing.T) {
	in, recipe := fixture(t)
	const want = "6c4509948055f209c3ec2d01b7f0d552d687946d1852efc1d8569178a41efe9c"
	if got := digest(t, in, "linux-amd64", recipe); got != want {
		t.Fatalf("digest = %s, want %s", got, want)
	}
}

func TestLibsDigestCoversItsInputs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		edit    func(*Inputs, fstest.MapFS)
		changes bool
	}{
		{"recipe content", func(_ *Inputs, r fstest.MapFS) {
			r["metadata.cmake"] = &fstest.MapFile{Data: []byte("file(WRITE y)\n")}
		}, true},
		{"recipe file added", func(_ *Inputs, r fstest.MapFS) { r["patches/fix.patch"] = &fstest.MapFile{Data: []byte("x")} }, true},
		{"recipe file renamed", func(_ *Inputs, r fstest.MapFS) {
			r["meta.cmake"] = r["metadata.cmake"]
			delete(r, "metadata.cmake")
		}, true},
		{"component version", func(in *Inputs, _ fstest.MapFS) { in.Components[0].Version = "2.0.16" }, true},
		{"component sha256", func(in *Inputs, _ fstest.MapFS) {
			in.Components[0].SHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
		}, true},
		{"own target row", func(in *Inputs, _ fstest.MapFS) { in.Targets[0].Toolchain = "musl-cross" }, true},
		{"another target row", func(in *Inputs, _ fstest.MapFS) { in.Targets[1].OpenSSLTarget = "darwin64-x86_64-cc" }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in, recipe := fixture(t)
			before := digest(t, in, "linux-amd64", recipe)
			tc.edit(&in, recipe)
			after := digest(t, in, "linux-amd64", recipe)
			if (before != after) != tc.changes {
				t.Fatalf("digest changed = %v, want %v", before != after, tc.changes)
			}
		})
	}
}

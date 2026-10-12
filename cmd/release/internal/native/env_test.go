package native

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestCgoEnvFromMetadata(t *testing.T) {
	layer := filepath.Join(t.TempDir(), "layer root")
	if err := os.MkdirAll(layer, 0o755); err != nil {
		t.Fatal(err)
	}
	metadata := `{
		"include_dirs": ["include"],
		"defines": ["TORRENT_NO_DEPRECATE", "WIN32_LEAN_AND_MEAN"],
		"static_libs": ["lib/libtorrent-rasterbar.a", "lib/libcrypto.a"],
		"system_libs": ["-lstdc++", "-lm"]
	}`
	if err := os.WriteFile(filepath.Join(layer, MetadataFile), []byte(metadata), 0o644); err != nil {
		t.Fatal(err)
	}
	md, err := ReadMetadata(layer)
	if err != nil {
		t.Fatal(err)
	}

	target := Target{GoTags: []string{"netgo", "osusergo"}, Env: map[string]string{"MACOSX_DEPLOYMENT_TARGET": "13.0", "CMAKE_OSX_ARCHITECTURES": "arm64"}}
	got := CgoEnv(layer, md, Compilers{CC: "gcc", CXX: "g++"}, target)

	cflags := "-O2 -g -DSLIPSTREAM_NATIVE_BUNDLE '-I" + filepath.Join(layer, "include") + "'"
	want := []EnvVar{
		{"CGO_ENABLED", "1"},
		{"GOFLAGS", "-tags=netgo,osusergo"},
		{"CC", "gcc"},
		{"CXX", "g++"},
		{"CGO_CFLAGS", cflags},
		{"CGO_CXXFLAGS", cflags + " -DTORRENT_NO_DEPRECATE -DWIN32_LEAN_AND_MEAN"},
		{"CGO_LDFLAGS", "'" + filepath.Join(layer, "lib", "libtorrent-rasterbar.a") + "' '" +
			filepath.Join(layer, "lib", "libcrypto.a") + "' -lstdc++ -lm"},
		{"CMAKE_OSX_ARCHITECTURES", "arm64"},
		{"MACOSX_DEPLOYMENT_TARGET", "13.0"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("CgoEnv =\n%q\nwant\n%q", got, want)
	}
}

func TestExportLinesEvalInShell(t *testing.T) {
	vars := []EnvVar{{"A", "-O2 '-I/x y'"}, {"B", `it's "quoted"`}}
	if got, want := ExportLines(vars), "export A='-O2 '\\''-I/x y'\\'''\nexport B='it'\\''s \"quoted\"'\n"; got != want {
		t.Fatalf("ExportLines =\n%s\nwant\n%s", got, want)
	}
	out, err := exec.Command("sh", "-c", ExportLines(vars)+`printf '%s|%s' "$A" "$B"`).Output()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), `-O2 '-I/x y'|it's "quoted"`; got != want {
		t.Fatalf("shell read back %q, want %q", got, want)
	}
}

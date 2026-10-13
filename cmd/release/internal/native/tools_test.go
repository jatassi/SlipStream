package native

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ulikunitz/xz"
)

func tarXZ(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	xw, err := xz.NewWriter(&buf)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(xw)
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(files[name])), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(files[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := xw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func zipped(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func serveArchives(t *testing.T, archives map[string][]byte) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, ok := archives[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

var (
	linuxTarget   = Target{Name: "linux-amd64", GOOS: "linux"}
	windowsTarget = Target{Name: "windows-amd64", GOOS: "windows"}
)

func TestFetchToolExtractsEachArchiveKind(t *testing.T) {
	linux := tarXZ(t, map[string]string{"ffmpeg": "linux ffmpeg", "ffprobe": "linux ffprobe"})
	windows := zipped(t, map[string]string{"ffmpeg.exe": "windows ffmpeg", "ffprobe.exe": "windows ffprobe"})
	base := serveArchives(t, map[string][]byte{"/linux.tar.xz": linux, "/windows.zip": windows})
	cache := t.TempDir()

	for _, tc := range []struct {
		target  Target
		archive string
		data    []byte
		want    map[string]string
	}{
		{linuxTarget, "/linux.tar.xz", linux, map[string]string{"ffmpeg": "linux ffmpeg", "ffprobe": "linux ffprobe"}},
		{windowsTarget, "/windows.zip", windows, map[string]string{"ffmpeg.exe": "windows ffmpeg", "ffprobe.exe": "windows ffprobe"}},
	} {
		p := PinnedTool{ToolArchive{"ffmpeg", tc.target.Name, base + tc.archive}, sha(tc.data)}
		dir, err := FetchTool(context.Background(), cache, p, tc.target, []string{"ffmpeg", "ffprobe"})
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(cache, "tool", tc.target.Name, sha(tc.data)); dir != want {
			t.Fatalf("FetchTool dir = %s, want %s", dir, want)
		}
		for name, content := range tc.want {
			got, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil || string(got) != content {
				t.Errorf("%s/%s = %q, %v; want %q", tc.target.Name, name, got, err, content)
			}
		}
	}
	info, err := os.Stat(filepath.Join(cache, "tool", "linux-amd64", sha(linux), "ffprobe"))
	if err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Errorf("ffprobe mode = %v, %v; want executable", info.Mode(), err)
	}
}

func TestFetchToolRefusesAWrongArchive(t *testing.T) {
	archive := tarXZ(t, map[string]string{"ffmpeg": "ffmpeg"})
	base := serveArchives(t, map[string][]byte{"/ffmpeg.tar.xz": archive})
	cache := t.TempDir()
	url := base + "/ffmpeg.tar.xz"
	wrong := strings.Repeat("0", 64)

	for _, tc := range []struct {
		pinned PinnedTool
		want   string
	}{
		{PinnedTool{ToolArchive{"ffmpeg", "linux-amd64", url}, wrong},
			url + " has SHA-256 " + sha(archive) + ", native/lock.json pins " + wrong},
		{PinnedTool{ToolArchive{"ffmpeg", "linux-amd64", url}, sha(archive)},
			url + " holds no ffprobe at its root"},
	} {
		_, err := FetchTool(context.Background(), cache, tc.pinned, linuxTarget, []string{"ffmpeg", "ffprobe"})
		if err == nil || err.Error() != tc.want {
			t.Errorf("FetchTool err = %v, want %s", err, tc.want)
		}
	}
	entries, err := os.ReadDir(filepath.Join(cache, "tool", "linux-amd64"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("cache holds %v (%v) after refused fetches, want nothing", entries, err)
	}
}

func TestPinToolsHashesEachArchive(t *testing.T) {
	linux := tarXZ(t, map[string]string{"ffmpeg": "ffmpeg"})
	base := serveArchives(t, map[string][]byte{"/linux.tar.xz": linux})
	archive := ToolArchive{"ffmpeg", "linux-amd64", base + "/linux.tar.xz"}

	got, err := PinTools(context.Background(), []ToolArchive{archive})
	if err != nil {
		t.Fatal(err)
	}
	if want := []PinnedTool{{archive, sha(linux)}}; !slices.Equal(got, want) {
		t.Fatalf("PinTools = %+v, want %+v", got, want)
	}
}

func TestToolArchivesListEveryToolForEveryTarget(t *testing.T) {
	in, _ := fixture(t)
	want := []ToolArchive{
		{"ffmpeg", "linux-amd64", "https://example.test/ffmpeg-linux.tar.xz"},
		{"ffmpeg", "darwin-arm64", "https://example.test/ffmpeg-mac.tar.xz"},
	}
	if got := in.ToolArchives(); !slices.Equal(got, want) {
		t.Fatalf("ToolArchives = %+v, want %+v", got, want)
	}
}

package native

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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
	writeTar(t, xw, files)
	if err := xw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func tarGZ(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	writeTar(t, gw, files)
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeTar(t *testing.T, w io.Writer, files map[string]string) {
	t.Helper()
	tw := tar.NewWriter(w)
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
	gz := tarGZ(t, map[string]string{"./dovi_tool": "linux dovi_tool"})
	base := serveArchives(t, map[string][]byte{"/linux.tar.xz": linux, "/windows.zip": windows, "/dovi.tar.gz": gz})
	cache := t.TempDir()

	for _, tc := range []struct {
		target   Target
		archive  string
		data     []byte
		programs []string
		want     map[string]string
	}{
		{linuxTarget, "/linux.tar.xz", linux, []string{"ffmpeg", "ffprobe"}, map[string]string{"ffmpeg": "linux ffmpeg", "ffprobe": "linux ffprobe"}},
		{windowsTarget, "/windows.zip", windows, []string{"ffmpeg", "ffprobe"}, map[string]string{"ffmpeg.exe": "windows ffmpeg", "ffprobe.exe": "windows ffprobe"}},
		{linuxTarget, "/dovi.tar.gz", gz, []string{"dovi_tool"}, map[string]string{"dovi_tool": "linux dovi_tool"}},
	} {
		p := PinnedTool{ToolArchive{"ffmpeg", tc.target.Name, base + tc.archive}, sha(tc.data)}
		dir, err := FetchTool(context.Background(), cache, p, tc.target, tc.programs)
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
	// Windows has no execute bit: os.Stat reports 0666 for every writable file.
	if runtime.GOOS == "windows" {
		return
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

func TestFetchFileChecksItsDigestAndCaches(t *testing.T) {
	served := map[string][]byte{"/runtime-x86_64": []byte("runtime")}
	base := serveArchives(t, served)
	cache := t.TempDir()
	d := Download{URL: base + "/runtime-x86_64", SHA256: sha([]byte("runtime"))}
	file, err := FetchFile(context.Background(), cache, d)
	if want := filepath.Join(cache, "file", d.SHA256, "runtime-x86_64"); err != nil || file != want {
		t.Fatalf("FetchFile = %s, %v; want %s", file, err, want)
	}
	delete(served, "/runtime-x86_64")
	if again, err := FetchFile(context.Background(), cache, d); err != nil || again != file {
		t.Fatalf("cached FetchFile = %s, %v; want %s", again, err, file)
	}
	wrong := Download{URL: d.URL + "-new", SHA256: d.SHA256}
	served["/runtime-x86_64-new"] = []byte("other")
	if _, err := FetchFile(context.Background(), t.TempDir(), wrong); err == nil || err.Error() != wrong.URL+" has SHA-256 "+sha([]byte("other"))+", native/inputs.json pins "+d.SHA256 {
		t.Fatalf("FetchFile of a changed file: %v", err)
	}
}

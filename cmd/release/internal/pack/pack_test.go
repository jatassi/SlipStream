package pack

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
	"github.com/jatassi/slipstream/internal/install"
)

func fakeRepo(t *testing.T, target native.Target) Input {
	t.Helper()
	root, dist := t.TempDir(), t.TempDir()
	write := func(path, content string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "LICENSE"), "GPL-3.0 text\n")
	write(filepath.Join(root, filepath.FromSlash(NoticesPath)), "notices\n")
	in := Input{
		Root: root, Dist: dist, Out: t.TempDir(), Version: "1.2.0", Target: target,
		Inputs: native.Inputs{Tools: []native.Tool{{Name: "jellyfin-ffmpeg", Programs: []string{"ffmpeg", "ffprobe"}}, {Name: "dovi_tool", Programs: []string{"dovi_tool"}}}},
		Log:    io.Discard,
	}
	for _, p := range in.programs() {
		write(filepath.Join(dist, p), p+" program")
	}
	return in
}

type entry struct {
	name    string
	mode    os.FileMode
	content string
}

func TestTarballHoldsOneFolderOfTheProgramsMarkerAndLicense(t *testing.T) {
	in := fakeRepo(t, native.Target{Name: "linux-arm64", GOOS: "linux", GOARCH: "arm64"})
	files, err := buildTarball(t.Context(), in, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"slipstream-1.2.0-linux-arm64.tar.gz"}; !slices.Equal(files, want) {
		t.Fatalf("tarball files = %v, want %v", files, want)
	}
	f, err := os.Open(filepath.Join(in.Out, files[0]))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var got []entry
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		content, _ := io.ReadAll(tr)
		got = append(got, entry{hdr.Name, hdr.FileInfo().Mode().Perm(), string(content)})
	}
	want := []entry{
		{"slipstream-1.2.0/", 0o755, ""},
		{"slipstream-1.2.0/LICENSE", 0o644, "GPL-3.0 text\n"},
		{"slipstream-1.2.0/NOTICES.txt", 0o644, "notices\n"},
		{"slipstream-1.2.0/dovi_tool", 0o755, "dovi_tool program"},
		{"slipstream-1.2.0/ffmpeg", 0o755, "ffmpeg program"},
		{"slipstream-1.2.0/ffprobe", 0o755, "ffprobe program"},
		{"slipstream-1.2.0/install-kind", 0o644, "tarball\n"},
		{"slipstream-1.2.0/slipstream", 0o755, "slipstream program"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tarball entries:\n%v\nwant:\n%v", got, want)
	}
}

func TestZIPHoldsTheTrampolineAndAReadme(t *testing.T) {
	in := fakeRepo(t, windowsInput.Target)
	files, err := buildZIP(t.Context(), in, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"SlipStream-1.2.0-windows-amd64.zip"}; !slices.Equal(files, want) {
		t.Fatalf("zip files = %v, want %v", files, want)
	}
	zr, err := zip.OpenReader(filepath.Join(in.Out, files[0]))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = zr.Close() }()
	got := map[string]string{}
	for _, f := range zr.File {
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, _ := io.ReadAll(r)
		_ = r.Close()
		got[f.Name] = string(content)
	}
	var names []string
	for name := range got {
		names = append(names, name)
	}
	slices.Sort(names)
	wantNames := []string{
		"SlipStream/", "SlipStream/LICENSE", "SlipStream/NOTICES.txt", "SlipStream/README.txt", "SlipStream/dovi_tool.exe",
		"SlipStream/ffmpeg.exe", "SlipStream/ffprobe.exe", "SlipStream/install-kind", "SlipStream/slipstream-app.exe", "SlipStream/slipstream.exe",
	}
	if !slices.Equal(names, wantNames) {
		t.Fatalf("zip entries = %v, want %v", names, wantNames)
	}
	if got["SlipStream/install-kind"] != "zip\n" {
		t.Errorf("marker = %q, want zip and a newline", got["SlipStream/install-kind"])
	}
	readme := got["SlipStream/README.txt"]
	if !strings.HasPrefix(readme, "SlipStream 1.2.0\r\n\r\nOpen slipstream-app.exe to start SlipStream.") {
		t.Errorf("README.txt starts %q", readme)
	}
}

func TestAppMarkerSitsInResources(t *testing.T) {
	in := fakeRepo(t, native.Target{Name: "darwin-arm64", GOOS: "darwin"})
	macOS := filepath.Join(t.TempDir(), "SlipStream.app", "Contents", "MacOS")
	if err := in.stagePrograms(macOS, install.App); err != nil {
		t.Fatal(err)
	}
	marker, err := os.ReadFile(filepath.Join(macOS, "..", "Resources", "install-kind"))
	if err != nil || string(marker) != "app\n" {
		t.Fatalf("Contents/Resources/install-kind = %q, %v; want app", marker, err)
	}
	if _, err := os.Stat(filepath.Join(macOS, "slipstream-app")); err != nil {
		t.Fatalf("the trampoline is not in Contents/MacOS: %v", err)
	}
}

func TestSystemPackageConfig(t *testing.T) {
	in := fakeRepo(t, native.Target{Name: "linux-arm64", GOOS: "linux", GOARCH: "arm64"})
	in.Version = "1.2.0-beta.1"
	got := systemPackageConfig(in, rpmPackage, "/stage/root", map[string]string{"postinstall": "/stage/postinstall.sh"})
	data, err := got.encode()
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"name":"slipstream","arch":"aarch64","platform":"linux","version":"1.2.0-beta.1","version_schema":"semver","release":"1",` +
		`"maintainer":"jatassi <38116120+jatassi@users.noreply.github.com>",` +
		`"description":"SlipStream manages a movie and TV library across your machines.","vendor":"SlipStream",` +
		`"homepage":"https://github.com/jatassi/SlipStream","license":"GPL-3.0","depends":["glibc >= 2.28"],"contents":[` +
		`{"src":"/stage/root/lib/slipstream","dst":"/usr/lib/slipstream/slipstream"},` +
		`{"src":"/stage/root/lib/ffmpeg","dst":"/usr/lib/slipstream/ffmpeg"},` +
		`{"src":"/stage/root/lib/ffprobe","dst":"/usr/lib/slipstream/ffprobe"},` +
		`{"src":"/stage/root/lib/dovi_tool","dst":"/usr/lib/slipstream/dovi_tool"},` +
		`{"src":"/stage/root/lib/install-kind","dst":"/usr/lib/slipstream/install-kind"},` +
		`{"src":"/usr/lib/slipstream/slipstream","dst":"/usr/bin/slipstream","type":"symlink"},` +
		`{"src":"/stage/root/slipstream.service","dst":"/usr/lib/systemd/system/slipstream.service"},` +
		`{"src":"/stage/root/sysusers.conf","dst":"/usr/lib/sysusers.d/slipstream.conf"},` +
		`{"src":"/stage/root/doc/LICENSE","dst":"/usr/share/doc/slipstream/LICENSE"},` +
		`{"src":"/stage/root/doc/NOTICES.txt","dst":"/usr/share/doc/slipstream/NOTICES.txt"}],` +
		`"scripts":{"postinstall":"/stage/postinstall.sh"}}` + "\n"
	if data != want {
		t.Fatalf("nfpm config:\n%s\nwant:\n%s", data, want)
	}
}

func TestVersions(t *testing.T) {
	for _, tc := range []struct {
		described string
		want      Version
		numeric   string
		tags      []string
	}{
		{"v1.2.0", "1.2.0", "1.2.0", []string{"1.2.0", "1.2", "1", "latest"}},
		{"v1.2.0-beta.1", "1.2.0-beta.1", "1.2.0", []string{"beta"}},
		{"v1.2.0-3-gabc1234", "1.2.0-3-gabc1234", "1.2.0", []string{"beta"}},
		{"abc1234", "0.0.0-gabc1234", "0.0.0", []string{"beta"}},
	} {
		v := Described(tc.described)
		if v != tc.want || v.Numeric() != tc.numeric || !slices.Equal(v.ImageTags(), tc.tags) {
			t.Errorf("Described(%q) = %q, numeric %q, tags %v; want %q, %q, %v", tc.described, v, v.Numeric(), v.ImageTags(), tc.want, tc.numeric, tc.tags)
		}
	}
	for _, tag := range []string{"1.2.0", "v1.2", "v1.2.0+meta", "v1.2.0-"} {
		if _, err := ParseTag(tag); err == nil {
			t.Errorf("ParseTag(%q) accepted a tag that is not vX.Y.Z", tag)
		}
	}
	if v, err := ParseTag("v2.0.1-rc.2"); err != nil || v != "2.0.1-rc.2" || !v.Prerelease() {
		t.Errorf("ParseTag(v2.0.1-rc.2) = %q, %v", v, err)
	}
}

func TestRTFEscapesBracesAndBreaksLines(t *testing.T) {
	got := rtf("GNU {GPL}\nC:\\path\n")
	want := "{\\rtf1\\ansi\\deff0{\\fonttbl{\\f0 Consolas;}}\\f0\\fs18\nGNU \\{GPL\\}\\par\nC:\\\\path\\par\n}\n"
	if got != want {
		t.Fatalf("rtf = %q, want %q", got, want)
	}
}

func TestICNSHoldsThreePNGs(t *testing.T) {
	data, err := icns()
	if err != nil {
		t.Fatal(err)
	}
	if string(data[:4]) != "icns" || int(binary.BigEndian.Uint32(data[4:8])) != len(data) {
		t.Fatalf("icns header %q %d, want icns and the file length %d", data[:4], binary.BigEndian.Uint32(data[4:8]), len(data))
	}
	var kinds []string
	for rest := data[8:]; len(rest) > 0; {
		size := binary.BigEndian.Uint32(rest[4:8])
		if !bytes.HasPrefix(rest[8:], []byte("\x89PNG")) {
			t.Errorf("entry %s holds no PNG", rest[:4])
		}
		kinds = append(kinds, string(rest[:4]))
		rest = rest[size:]
	}
	if want := []string{"ic08", "ic09", "ic10"}; !slices.Equal(kinds, want) {
		t.Fatalf("icns entries = %v, want %v", kinds, want)
	}
}

func TestWixVersionMatchesTheDotnetToolPin(t *testing.T) {
	data, err := os.ReadFile("../../../../.config/dotnet-tools.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Tools map[string]struct {
			Version string `json:"version"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if got := manifest.Tools["wix"].Version; got != wixVersion {
		t.Fatalf(".config/dotnet-tools.json pins wix %s, and the UI extension takes %s", got, wixVersion)
	}
}

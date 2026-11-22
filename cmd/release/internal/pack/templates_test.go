package pack

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

var update = flag.Bool("update", false, "rewrite the golden files under testdata from the templates")

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s differs from %s:\n%s", name, path, got)
	}
}

var windowsInput = Input{
	Version: "1.2.0-beta.1",
	Target:  native.Target{Name: "windows-amd64", GOOS: "windows"},
	Inputs:  native.Inputs{Tools: []native.Tool{{Programs: []string{"ffmpeg", "ffprobe"}}, {Programs: []string{"dovi_tool"}}}},
}

func TestTemplatesMatchTheirGoldens(t *testing.T) {
	for _, p := range []systemPackage{debPackage, rpmPackage} {
		for script, text := range maintainerScripts(p) {
			golden(t, p.kind.String()+"-"+script+".sh", text)
		}
	}
	golden(t, "sysusers.conf", sysusers())

	desktop := desktopData{installNames, desktopArgs}
	golden(t, "AppRun", render("AppRun", desktop))
	golden(t, "slipstream.desktop", render("slipstream.desktop", desktop))

	docker := dockerData{installNames, "1.2.0", imageDir}
	golden(t, "Dockerfile", render("Dockerfile", docker))
	golden(t, "entrypoint", render("entrypoint", docker))

	golden(t, "Info.plist", infoPlist(Input{Version: "1.2.0-beta.1", Target: native.Target{Name: "darwin-arm64", GOOS: "darwin", MinOS: "macos 13.0"}}))

	files := append(windowsInput.programs(), "install-kind", "LICENSE", "NOTICES.txt")
	stage := filepath.FromSlash("/stage")
	golden(t, "SlipStream.wxs", strings.ReplaceAll(wixSource(windowsInput, stage, files), stage+string(filepath.Separator), "/stage/"))
}

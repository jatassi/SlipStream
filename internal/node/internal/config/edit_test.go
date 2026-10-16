package config_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jatassi/slipstream/internal/node/internal/config"
)

func text(s string) *string { return &s }

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func accept(fs.FS) error { return nil }

const commented = `# media-server's own settings
name: media-server # the box under the stairs

http:
  # 9000 is taken by Portainer
  listen: "[::]:8484"
log:
  max_files: 3
  level: info # quiet by default
`

// yaml.v3 keeps comments and order but not blank lines.
func TestEditKeepsCommentsAndOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", commented)
	err := config.Edit(dir, []config.Change{
		{Key: "log.level", Text: text("debug")},
		{Key: "http.listen", Text: text("[::]:9001")},
		{Key: "log.compress", Text: text("false")},
		{Key: "timings.node.stop_deadline", Text: text("5s")},
	}, accept)
	if err != nil {
		t.Fatal(err)
	}
	want := `# media-server's own settings
name: media-server # the box under the stairs
http:
  # 9000 is taken by Portainer
  listen: "[::]:9001"
log:
  max_files: 3
  level: debug # quiet by default
  compress: false
timings:
  node:
    stop_deadline: 5s
`
	if got := readFile(t, filepath.Join(dir, "config.yaml")); got != want {
		t.Fatalf("config.yaml =\n%s\nwant\n%s", got, want)
	}
}

func TestEditCreatesTheFileAndQuotesWhatNeedsIt(t *testing.T) {
	dir := t.TempDir()
	if err := config.Edit(dir, []config.Change{{Key: "http.listen", Text: text("[::]:9000")}, {Key: "name", Text: text("media: server")}}, accept); err != nil {
		t.Fatal(err)
	}
	want := "http:\n  listen: '[::]:9000'\nname: 'media: server'\n"
	if got := readFile(t, filepath.Join(dir, "config.yaml")); got != want {
		t.Fatalf("config.yaml =\n%s\nwant\n%s", got, want)
	}
}

func TestUnsetRemovesTheKeyAndAnyMappingLeftEmpty(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", commented)
	err := config.Edit(dir, []config.Change{{Key: "http.listen"}, {Key: "log.level"}, {Key: "log.compress"}}, accept)
	if err != nil {
		t.Fatal(err)
	}
	want := `# media-server's own settings
name: media-server # the box under the stairs
log:
  max_files: 3
`
	if got := readFile(t, filepath.Join(dir, "config.yaml")); got != want {
		t.Fatalf("config.yaml =\n%s\nwant\n%s", got, want)
	}
}

func TestUnsettingTheLastKeyLeavesAnEmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "log:\n  level: debug\n")
	if err := config.Edit(dir, []config.Change{{Key: "log.level"}}, accept); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(dir, "config.yaml")); got != "" {
		t.Fatalf("config.yaml = %q, want it empty", got)
	}
}

func TestEditChecksTheEditedFileBesideDotEnv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "retries: 4\n")
	writeFile(t, dir, ".env", "SLIPSTREAM_NAME=from-dotenv\n")
	var seen tree
	var keys []config.Key
	err := config.Edit(dir, []config.Change{{Key: "retries", Text: text("9")}}, func(fsys fs.FS) error {
		seen = defaults()
		var err error
		keys, err = config.LoadFS(&seen, fsys, nil)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if seen.Retries != 9 || seen.Name != "from-dotenv" || keyOf(t, keys, "retries").Source != config.SourceYAML {
		t.Fatalf("check saw retries %d and name %q, want 9 and from-dotenv", seen.Retries, seen.Name)
	}
}

func TestARefusedEditLeavesTheFileAsItWas(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", commented)
	refused := errors.New("refused")
	err := config.Edit(dir, []config.Change{{Key: "log.level", Text: text("debug")}}, func(fsys fs.FS) error {
		b, err := fs.ReadFile(fsys, "config.yaml")
		if err != nil || string(b) == commented {
			t.Errorf("check read %q, %v; want the edited file", b, err)
		}
		return refused
	})
	if !errors.Is(err, refused) {
		t.Fatalf("err = %v, want the check's refusal", err)
	}
	if got := readFile(t, filepath.Join(dir, "config.yaml")); got != commented {
		t.Fatalf("config.yaml after a refusal =\n%s\nwant it unchanged", got)
	}
}

func TestEditRefusesASymlinkedFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating a symlink needs Developer Mode on Windows")
	}
	dir, elsewhere := t.TempDir(), t.TempDir()
	target := filepath.Join(elsewhere, "config.yaml")
	writeFile(t, elsewhere, "config.yaml", commented)
	if err := os.Symlink(target, filepath.Join(dir, "config.yaml")); err != nil {
		t.Fatal(err)
	}
	err := config.Edit(dir, []config.Change{{Key: "log.level", Text: text("debug")}}, accept)
	want := "config.yaml is not a regular file, so SlipStream can't rewrite it; edit it by hand"
	if !errors.Is(err, config.ErrNotRegular) || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
	if got := readFile(t, target); got != commented {
		t.Fatalf("the link's target =\n%s\nwant it unchanged", got)
	}
}

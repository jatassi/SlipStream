package rules_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/lint"
	_ "github.com/jatassi/slipstream/cmd/release/internal/lint/rules"
	"github.com/jatassi/slipstream/cmd/release/internal/native"
)

func fixtureRepo(t *testing.T, name string) *lint.Repo {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(root, os.DirFS(filepath.Join("testdata", name))); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return lint.Open(root)
}

func check(t *testing.T, rule string, repo *lint.Repo) []string {
	t.Helper()
	r, ok := lint.Lookup(rule)
	if !ok {
		t.Fatalf("rule %s is not registered", rule)
	}
	var out strings.Builder
	if err := lint.Run(&out, repo, []lint.Rule{r}); err == nil {
		t.Fatal("lint passed a fixture with findings")
	}
	return strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
}

func assertFindings(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("findings:\n%q\nwant:\n%q", got, want)
	}
}

func TestCgoConfined(t *testing.T) {
	repo := fixtureRepo(t, "cgo-confined")
	untracked := filepath.Join(repo.Root, "internal", "media", "untracked.go")
	if err := os.WriteFile(untracked, []byte("package media\n\nimport \"C\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	assertFindings(t, check(t, "cgo-confined", repo), []string{
		`internal/media/probe_windows.go:8: [cgo-confined] import "C" outside the declared native packages`,
		`internal/torrent/libtorrent/session/session.go:3: [cgo-confined] import "C" outside the declared native packages`,
	})
}

func TestReleaseImports(t *testing.T) {
	all := "linux/amd64, linux/arm64, windows/amd64, darwin/arm64"
	assertFindings(t, check(t, "release-imports", fixtureRepo(t, "release-imports")), []string{
		"cmd/slipstream/main.go:3: [release-imports] cmd/slipstream imports internal/app -> cmd/release/stamp on " + all,
		"cmd/slipstream/main.go:3: [release-imports] cmd/slipstream imports internal/app -> fakes/web on " + all,
		"cmd/slipstream/tmdb_windows.go:6: [release-imports] cmd/slipstream imports fakes/tmdb on windows/amd64",
	})
}

func TestParityOwners(t *testing.T) {
	assertFindings(t, check(t, "parity-owners", fixtureRepo(t, "parity-owners")), []string{
		"docs/parity-matrix.md:6: [parity-owners] P-002: expected one spec in Covered by, found 0",
		"docs/parity-matrix.md:7: [parity-owners] P-003: Status is open, expected planned or shipped",
		"docs/parity-matrix.md:7: [parity-owners] P-003: expected one spec in Covered by, found 2",
		"docs/parity-matrix.md:9: [parity-owners] P-005: Status is planned, expected -",
		"docs/parity-matrix.md:10: [parity-owners] P-006: unknown disposition open",
	})
}

func TestNativeLock(t *testing.T) {
	hint := "; " + native.PublishHint
	assertFindings(t, check(t, "native-lock", fixtureRepo(t, "native-lock")), []string{
		"native/lock.json:1: [native-lock] pins libs layer 0000000000000000000000000000000000000000000000000000000000000000 for linux-amd64, inputs and recipe build 33c32da85e817d553bca283ba63360455c21ad0be9c02f857261c75b2ecfe946" + hint,
		"native/lock.json:1: [native-lock] pins libs layer 2222222222222222222222222222222222222222222222222222222222222222 for windows-amd64, which native/inputs.json does not list" + hint,
		"native/lock.json:1: [native-lock] pins no libs layer for darwin-arm64, inputs and recipe build 13fef6af146855c74b4206aa97728129f7231c1f0793afb381abc1310259fc84" + hint,
		"native/lock.json:1: [native-lock] pins no toolchain layer for linux-amd64, inputs and recipe build c871a31bd840d2719733f3d367edc4c1807ebcd6ba999b630294eb6bab409d39" + hint,
	})
}

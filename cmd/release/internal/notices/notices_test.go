package notices_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/notices"
)

func fixtureRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS(filepath.Join("testdata", "repo"))); err != nil {
		t.Fatal(err)
	}
	return root
}

func edit(t *testing.T, root, rel, old, new string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q", rel, old)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), old, new, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertError(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("got no error, want:\n%s", want)
	}
	if err.Error() != want {
		t.Errorf("error:\n%s\nwant:\n%s", err, want)
	}
}

var update = flag.Bool("update", false, "regenerate testdata/repo/internal/notices/NOTICES.txt")

func TestCheckPassesOnCommittedNotices(t *testing.T) {
	if *update {
		if err := notices.Write(filepath.Join("testdata", "repo")); err != nil {
			t.Fatal(err)
		}
	}
	if err := notices.Check(fixtureRepo(t)); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateListsEveryShippedDependency(t *testing.T) {
	got, err := notices.Generate(fixtureRepo(t))
	if err != nil {
		t.Fatal(err)
	}
	var headings []string
	lines := strings.Split(string(got), "\n")
	for i, line := range lines {
		if line == strings.Repeat("=", 80) {
			headings = append(headings, strings.Join(lines[i+1:i+4], " | "))
		}
	}
	want := []string{
		"example.com/bsd v1.0.0 (Go module) | License: BSD-2-Clause | Targets: linux-amd64, windows-amd64",
		"example.com/odd v1.0.0 (Go module) | License: MIT | Targets: linux-amd64, windows-amd64",
		"example.com/winonly v0.3.0 (Go module) | License: MIT | Targets: windows-amd64",
		"go 1.27 (toolchain runtime) | License: BSD-3-Clause | Targets: linux-amd64, windows-amd64",
		"leftpad 1.0.0 (npm package) | License: ISC | Targets: linux-amd64, windows-amd64",
		"musl 1.2.5 (toolchain runtime) | License: MIT | Targets: linux-amd64",
		"tick 1.0.0 (npm package) | License: 0BSD | Targets: linux-amd64, windows-amd64",
		"tick 2.0.0 (npm package) | License: MIT | Targets: linux-amd64, windows-amd64",
		"zlib 1.3.1 (native library) | License: Zlib | Targets: linux-amd64, windows-amd64",
	}
	if strings.Join(headings, "\n") != strings.Join(want, "\n") {
		t.Errorf("entries:\n%s\nwant:\n%s", strings.Join(headings, "\n"), strings.Join(want, "\n"))
	}
}

func TestCheckFailsOnDrift(t *testing.T) {
	root := fixtureRepo(t)
	edit(t, root, "native/licenses/zlib.txt", "zlib License", "zlib license, amended")
	assertError(t, notices.Check(root), "internal/notices/NOTICES.txt is stale; run go run ./cmd/release notices generate")
}

func TestOverrideDoesNotCoverTheNextVersion(t *testing.T) {
	root := fixtureRepo(t)
	edit(t, root, "go.mod", "example.com/odd v1.0.0", "example.com/odd v1.0.1")
	assertError(t, notices.Check(root), strings.Join([]string{
		"override example.com/odd v1.0.0 in licenses.json matches no shipped dependency",
		"Go module example.com/odd v1.0.1: license NOASSERTION is not on the allow-list in licenses.json",
	}, "\n"))
}

func TestDeniedLicenseFailsWithItsReason(t *testing.T) {
	root := fixtureRepo(t)
	edit(t, root, "web/node_modules/tick/package.json", `"0BSD"`, `"GPL-2.0-only"`)
	assertError(t, notices.Check(root),
		"npm package tick 1.0.0: license GPL-2.0-only is denied: GPL-2.0-only cannot combine with GPL-3.0 (ADR 0010, ADR 0021)")
}

func TestModuleWithoutLicenseIsDenied(t *testing.T) {
	root := fixtureRepo(t)
	if err := os.Remove(filepath.Join(root, "third_party", "bsd", "LICENSE")); err != nil {
		t.Fatal(err)
	}
	assertError(t, notices.Check(root),
		"Go module example.com/bsd v1.0.0: license NONE is denied: code with no license grants no right to redistribute it (ADR 0010, ADR 0021)")
}

func TestStaleNodeModulesNamesTheFix(t *testing.T) {
	root := fixtureRepo(t)
	edit(t, root, "web/bun.lock", `"tick@1.0.0"`, `"tick@1.0.1"`)
	dir := filepath.Join(root, "web", "node_modules", "tick")
	assertError(t, notices.Check(root),
		dir+" holds tick@1.0.0 but web/bun.lock pins tick@1.0.1; run bun install --frozen-lockfile in web")
}

func TestClassify(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("testdata", "licenses", "*.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 14 {
		t.Fatalf("found %d license fixtures, want 14", len(files))
	}
	for _, file := range files {
		want := strings.TrimSuffix(filepath.Base(file), ".txt")
		text, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if got := notices.Classify(string(text)); got != want {
			t.Errorf("Classify(%s) = %s, want %s", file, got, want)
		}
	}
}

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/cmd/release/internal/nativetouch"
)

func TestGitChangesReadsBothSidesFromTheMergeBase(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	recipe := "add_library(boost INTERFACE)\nset(BOOST_VERSION 1.92.0)\n"

	git("init", "-q", "-b", "main")
	write("native/recipe/boost.cmake", recipe)
	write("internal/media/probe_windows.go", "package media\n")
	write("docs/release.md", "v1\n")
	git("add", ".")
	git("commit", "-q", "-m", "base")
	git("checkout", "-q", "-b", "change")
	git("mv", "native/recipe/boost.cmake", "docs/boost.cmake")
	git("rm", "-q", "internal/media/probe_windows.go")
	write("docs/release.md", "v2\n")
	write("web/app.ts", "export {}\n")
	git("add", ".")
	git("commit", "-q", "-m", "change")
	git("checkout", "-q", "main")
	write("docs/later.md", "on main after the branch\n")
	git("add", ".")
	git("commit", "-q", "-m", "main moves on")

	got, err := gitChanges(t.Context(), dir, "main", "change")
	if err != nil {
		t.Fatal(err)
	}
	side := func(path, content string) nativetouch.Side {
		return nativetouch.Side{Path: path, Content: []byte(content)}
	}
	want := []nativetouch.Change{
		{Old: side("native/recipe/boost.cmake", recipe), New: side("docs/boost.cmake", recipe)},
		{Old: side("docs/release.md", "v1\n"), New: side("docs/release.md", "v2\n")},
		{Old: side("internal/media/probe_windows.go", "package media\n")},
		{New: side("web/app.ts", "export {}\n")},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("gitChanges =\n%v\nwant\n%v", got, want)
	}
}

package worktree_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/worktree"
)

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func fingerprint(t *testing.T, dir string) string {
	t.Helper()
	f, err := worktree.Fingerprint(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func newRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "init", "-q", "-b", "main")
	run(t, dir, "config", "user.email", "test@example.com")
	run(t, dir, "config", "user.name", "test")
	run(t, dir, "config", "commit.gpgsign", "false")
	write(t, dir, ".gitignore", "/out/\n")
	write(t, dir, "README.md", "readme\n")
	write(t, dir, "src/main.go", "package main\n")
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-q", "-m", "initial")
	return dir
}

func TestFingerprintTracksEveryChangeAndReturnsToCleanAfterCommit(t *testing.T) {
	dir := newRepo(t)
	if got := fingerprint(t, dir); got != worktree.CleanFingerprint {
		t.Fatalf("fresh clone fingerprint = %q, want %q", got, worktree.CleanFingerprint)
	}

	seen := map[string]string{worktree.CleanFingerprint: "clean"}
	step := func(name string, change func()) {
		t.Helper()
		change()
		got := fingerprint(t, dir)
		if prev, dup := seen[got]; dup {
			t.Fatalf("after %s fingerprint %s repeats the one after %s", name, got, prev)
		}
		seen[got] = name
	}
	step("edit", func() { write(t, dir, "README.md", "edited\n") })
	step("second edit of the same file", func() { write(t, dir, "README.md", "edited again\n") })
	step("add untracked", func() { write(t, dir, "src/new/util.go", "package new\n") })
	step("delete tracked", func() {
		if err := os.Remove(filepath.Join(dir, "src", "main.go")); err != nil {
			t.Fatal(err)
		}
	})

	before := fingerprint(t, dir)
	write(t, dir, "out/app.bin", "ignored\n")
	if got := fingerprint(t, dir); got != before {
		t.Errorf("ignored file changed fingerprint from %s to %s", before, got)
	}

	run(t, dir, "add", "-A")
	if got := fingerprint(t, dir); got != before {
		t.Errorf("staging changed fingerprint from %s to %s", before, got)
	}
	run(t, dir, "commit", "-q", "-m", "changes")
	if got := fingerprint(t, dir); got != worktree.CleanFingerprint {
		t.Errorf("fingerprint after commit = %q, want %q", got, worktree.CleanFingerprint)
	}
}

func TestFingerprintIsTheSameForTheSameEdit(t *testing.T) {
	dir := newRepo(t)
	write(t, dir, "README.md", "edited\n")
	first := fingerprint(t, dir)
	write(t, dir, "README.md", "other\n")
	write(t, dir, "README.md", "edited\n")
	if got := fingerprint(t, dir); got != first {
		t.Errorf("fingerprint = %s, want %s for identical content", got, first)
	}
}

func TestStampDescribesHeadAndTag(t *testing.T) {
	dir := newRepo(t)
	run(t, dir, "tag", "v2.0.0")
	write(t, dir, "README.md", "edited\n")
	run(t, dir, "commit", "-q", "-am", "after tag")

	s, err := worktree.Stamp(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	newHead := run(t, dir, "rev-parse", "HEAD")
	wantVersion := "v2.0.0-1-g" + newHead[:7]
	if s.Version != wantVersion || s.Commit+"\n" != newHead || s.Fingerprint != worktree.CleanFingerprint {
		t.Errorf("Stamp = %+v, want version %s, commit %s, clean", s, wantVersion, newHead)
	}
}

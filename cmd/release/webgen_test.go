package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebGeneratedReportsStaleAndUncommittedFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	write("web/src/api/gen/query/identity/identity.ts", "export const a = 1;\n")
	write("web/src/routeTree.gen.ts", "export const routeTree = {};\n")
	write("web/src/app.tsx", "export const app = 1;\n")
	git("init", "-q")
	git("add", ".")
	git("commit", "-q", "-m", "generated")

	var out strings.Builder
	if err := webGenerated(t.Context(), dir, &out); err != nil || out.Len() > 0 {
		t.Fatalf("clean tree: err %v, output %q", err, out.String())
	}

	write("web/src/api/gen/query/identity/identity.ts", "export const a = 2;\n")
	write("web/src/api/gen/query/live/live.ts", "export const b = 1;\n")
	write("web/src/app.tsx", "export const app = 2;\n")
	err := webGenerated(t.Context(), dir, &out)
	if want := "stale: web/src/api/gen/query/identity/identity.ts, web/src/api/gen/query/live/live.ts"; err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %s", err, want)
	}
	const run = ": generated code is stale or uncommitted; run bun run gen in web and commit it\n"
	if want := "web/src/api/gen/query/identity/identity.ts" + run + "web/src/api/gen/query/live/live.ts" + run; out.String() != want {
		t.Errorf("output =\n%s\nwant\n%s", out.String(), want)
	}
}

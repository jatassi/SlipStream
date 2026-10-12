package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrationsFrozenReportsMergedMigrationsTheChangeRewrites(t *testing.T) {
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
	table := func(name string) string {
		return "-- +goose Up\nCREATE TABLE " + name + " (id INTEGER PRIMARY KEY, title TEXT NOT NULL) STRICT;\n"
	}

	git("init", "-q", "-b", "main")
	write("internal/library/schema/00001_library.sql", table("library"))
	write("internal/library/schema/00002_library_tag.sql", table("library_tag"))
	write("internal/library/schema/00003_library_genre.sql", table("library_genre"))
	write("internal/library/queries/library.sql", "-- name: GetLibrary :one\nSELECT * FROM library WHERE id = ?;\n")
	write("internal/library/testdata/schema/00001_fixture.sql", table("fixture"))
	git("add", ".")
	git("commit", "-q", "-m", "merged migrations")
	git("checkout", "-q", "-b", "change")
	write("internal/library/schema/00001_library.sql", table("library")+"CREATE INDEX library_title ON library (title);\n")
	git("rm", "-q", "internal/library/schema/00002_library_tag.sql")
	git("mv", "internal/library/schema/00003_library_genre.sql", "internal/library/schema/00003_library_genres.sql")
	write("internal/library/schema/00004_library_note.sql", table("library_note"))
	write("internal/library/queries/library.sql", "-- name: GetLibrary :one\nSELECT id FROM library WHERE id = ?;\n")
	write("internal/library/testdata/schema/00001_fixture.sql", table("fixture_edited"))
	git("add", ".")
	git("commit", "-q", "-m", "change")
	git("checkout", "-q", "main")
	write("internal/media/schema/00001_media.sql", table("media"))
	git("add", ".")
	git("commit", "-q", "-m", "main moves on")
	git("checkout", "-q", "change")

	var out strings.Builder
	err := migrationsFrozen(t.Context(), &out, dir, "main")
	want := "internal/library/schema/00001_library.sql:1: [migrations-frozen] edited after it merged into main; add a new migration instead\n" +
		"internal/library/schema/00002_library_tag.sql:1: [migrations-frozen] deleted after it merged into main; add a new migration instead\n" +
		"internal/library/schema/00003_library_genre.sql:1: [migrations-frozen] renamed after it merged into main; add a new migration instead\n"
	if out.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", out.String(), want)
	}
	if err == nil || err.Error() != "3 findings" {
		t.Errorf("err = %v, want 3 findings", err)
	}

	out.Reset()
	if err := migrationsFrozen(t.Context(), &out, dir, "change"); err != nil || out.String() != "" {
		t.Errorf("against its own head: err = %v, output %q, want neither", err, out.String())
	}

	out.Reset()
	err = migrationsFrozen(t.Context(), &out, dir, "origin/main")
	if err == nil || !strings.Contains(err.Error(), "git merge-base") {
		t.Errorf("missing base: err = %v, want the git merge-base failure", err)
	}
}

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSqlcDriftReportsAnOwnerWhoseQueriesChanged(t *testing.T) {
	root, err := repoRoot(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.CopyFS(filepath.Join(dir, "shelf"), os.DirFS(filepath.Join(root, "internal", "storage", "example", "shelf"))); err != nil {
		t.Fatal(err)
	}
	configs := []string{"shelf/sqlc.yaml"}

	var out bytes.Buffer
	if err := sqlcDrift(t.Context(), root, dir, configs, &out); err != nil || out.Len() > 0 {
		t.Fatalf("fresh copy: err %v, output %q", err, out.String())
	}

	query := filepath.Join(dir, "shelf", "queries", "shelf.sql")
	src, err := os.ReadFile(query)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(src), "SELECT count(*) FROM shelf;", "SELECT count(*) AS n FROM shelf;", 1)
	if edited == string(src) {
		t.Fatal("query to edit not found")
	}
	if err := os.WriteFile(query, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	err = sqlcDrift(t.Context(), root, dir, configs, &out)
	if err == nil || err.Error() != "stale: shelf" {
		t.Fatalf("err = %v, want stale: shelf", err)
	}
	first, rest, _ := strings.Cut(out.String(), "\n")
	if want := "shelf: generated code is stale; run go run ./cmd/release sqlc"; first != want {
		t.Errorf("first line = %q, want %q", first, want)
	}
	if !strings.Contains(rest, "+SELECT count(*) AS n FROM shelf") {
		t.Errorf("output lacks sqlc's diff:\n%s", rest)
	}
}

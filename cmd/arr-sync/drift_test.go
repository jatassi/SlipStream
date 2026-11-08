package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func openRepo(t *testing.T, dir string) *os.Root {
	t.Helper()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Close() })
	return root
}

func repoPins(t *testing.T) (*os.Root, pins) {
	t.Helper()
	repo := openRepo(t, "../..")
	p, err := readPins(repo)
	if err != nil {
		t.Fatal(err)
	}
	return repo, p
}

func TestGeneratedFilesMatchThePins(t *testing.T) {
	repo, p := repoPins(t)
	problems, err := drift(repo, p)
	if err != nil {
		t.Fatal(err)
	}
	for _, problem := range problems {
		t.Error(problem)
	}
}

func TestDriftFindsAPinBumpWithoutARerun(t *testing.T) {
	repo, p := repoPins(t)
	sonarr := p.Upstreams["sonarr"]
	sonarr.Commit = strings.Repeat("0", 40)
	p.Upstreams["sonarr"] = sonarr
	problems, err := drift(repo, p)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) == 0 {
		t.Fatal("drift found nothing after Sonarr's pin moved")
	}
}

func TestDriftFindsAHandEdit(t *testing.T) {
	repo, p := repoPins(t)
	const file = "internal/release/internal/upstream/zz_sonarr_quality_parser.go"
	src, err := repo.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	pin := p.Upstreams["sonarr"].Commit
	for name, tc := range map[string]struct {
		edit     func(string) string
		problems int
	}{
		"untouched":           {func(s string) string { return s }, 0},
		"commit in header":    {func(s string) string { return strings.Replace(s, pin, strings.Repeat("1", 40), 1) }, 1},
		"digest in header":    {func(s string) string { return strings.Replace(s, digestPrefix, digestPrefix+"0", 1) }, 1},
		"pattern in the body": {func(s string) string { return strings.Replace(s, `\b`, `\B`, 1) }, 1},
		"header removed":      {func(s string) string { _, body, _ := strings.Cut(s, "\n\n"); return body }, 1},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, filepath.FromSlash(file))
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.edit(string(src))), 0o600); err != nil {
				t.Fatal(err)
			}
			problems, err := drift(openRepo(t, dir), p)
			if err != nil {
				t.Fatal(err)
			}
			if len(problems) != tc.problems {
				t.Errorf("drift found %d problems, want %d: %q", len(problems), tc.problems, problems)
			}
		})
	}
}

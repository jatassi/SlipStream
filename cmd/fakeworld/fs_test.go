package main_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestFsStatAndMatch(t *testing.T) {
	t.Parallel()
	runDir, m := startWorld(t)
	t.Cleanup(func() { fakeworld(t, "stop", runDir) })
	rel := m.Releases[0]
	payload := filepath.Join(m.ScratchDir, "releases", rel.Name+".mkv")

	stat := fakeworld(t, "fs", "stat", runDir, payload)
	if want := `{"path":` + jsonText(t, payload) + `,"size":` + jsonText(t, rel.Size) + `,"links":1,"blake3":"` + rel.BLAKE3 + `"}` + "\n"; stat != (result{0, want, ""}) {
		t.Fatalf("fs stat = %+v, want %q", stat, want)
	}
	if got := fakeworld(t, "fs", "match", runDir, payload, "--release", rel.Name); got != stat {
		t.Fatalf("fs match of the payload = %+v, want %+v", got, stat)
	}

	other := filepath.Join(runDir, "other.mkv")
	if err := os.WriteFile(other, []byte("not the payload"), 0o600); err != nil {
		t.Fatal(err)
	}
	mismatch := fakeworld(t, "fs", "match", runDir, other, "--release", rel.Name)
	if want := "fakeworld fs match: " + other + " holds 15 bytes with BLAKE3 "; mismatch.code != 1 || !strings.HasPrefix(mismatch.stderr, want) || !strings.HasSuffix(mismatch.stderr, "not release "+rel.Name+"'s "+jsonText(t, rel.Size)+" bytes with BLAKE3 "+rel.BLAKE3+"\n") {
		t.Fatalf("fs match of another file = %+v, want exit 1 naming both", mismatch)
	}

	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	want := result{1, "", "fakeworld fs stat: " + outside + " is outside the run dir " + m.RunDir + " and the scratch dir " + m.ScratchDir + "\n"}
	if got := fakeworld(t, "fs", "stat", runDir, outside); got != want {
		t.Fatalf("fs stat outside the run = %+v, want %+v", got, want)
	}

	var starts []string
	for _, line := range journal(t, runDir) {
		if strings.HasPrefix(line, "step.start") {
			starts = append(starts, line)
		}
	}
	wantStarts := []string{
		`step.start {"n":1,"verb":"fs stat","args":` + jsonText(t, []string{payload}) + `}`,
		`step.start {"n":2,"verb":"fs match","args":` + jsonText(t, []string{payload, "--release", rel.Name}) + `}`,
		`step.start {"n":3,"verb":"fs match","args":` + jsonText(t, []string{other, "--release", rel.Name}) + `}`,
		`step.start {"n":4,"verb":"fs stat","args":` + jsonText(t, []string{outside}) + `}`,
	}
	if !slices.Equal(starts, wantStarts) {
		t.Fatalf("the journal starts\n%s\nwant\n%s", strings.Join(starts, "\n"), strings.Join(wantStarts, "\n"))
	}
}

func TestFsChangesTheMediaDir(t *testing.T) {
	t.Parallel()
	runDir, m := startWorld(t)
	t.Cleanup(func() { fakeworld(t, "stop", runDir) })
	media := func(rel string) string { return filepath.Join(m.MediaDir, filepath.FromSlash(rel)) }
	if err := os.MkdirAll(media("shelf"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(media("shelf/a.mkv"), []byte("a film"), 0o600); err != nil {
		t.Fatal(err)
	}
	made := func(rel string) result { return result{0, `{"path":` + jsonText(t, media(rel)) + "}\n", ""} }
	outside := filepath.Join(t.TempDir(), "outside.mkv")
	steps := []struct {
		args []string
		want result
	}{
		{[]string{"cp", "shelf/a.mkv", "shelf/b.mkv"}, made("shelf/b.mkv")},
		{[]string{"cp", "shelf", "copy"}, made("copy")},
		{[]string{"mv", "shelf/b.mkv", "Heat (1995)/Heat.mkv"}, made("Heat (1995)/Heat.mkv")},
		{[]string{"mv", media("shelf/a.mkv"), "copy/a.mkv"}, result{1, "", "fakeworld fs mv: " + media("copy/a.mkv") + " already exists\n"}},
		{[]string{"rm", "copy"}, made("copy")},
		{[]string{"rm", "copy"}, result{1, "", "fakeworld fs rm: lstat " + media("copy") + ": no such file or directory\n"}},
		{[]string{"rm", outside}, result{1, "", "fakeworld fs rm: " + outside + " is not inside the media dir " + m.MediaDir + "\n"}},
		{[]string{"cp", "../ca.pem", "ca.pem"}, result{1, "", "fakeworld fs cp: " + filepath.Join(filepath.Dir(m.MediaDir), "ca.pem") + " is not inside the media dir " + m.MediaDir + "\n"}},
		{[]string{"rm", "."}, result{1, "", "fakeworld fs rm: " + m.MediaDir + " is not inside the media dir " + m.MediaDir + "\n"}},
	}
	var wantStarts []string
	for i, s := range steps {
		if got := fakeworld(t, append([]string{"fs", s.args[0], runDir}, s.args[1:]...)...); got != s.want {
			t.Errorf("fs %v = %+v, want %+v", s.args, got, s.want)
		}
		wantStarts = append(wantStarts, `step.start {"n":`+jsonText(t, i+1)+`,"verb":"fs `+s.args[0]+`","args":`+jsonText(t, s.args[1:])+`}`)
	}

	var got []string
	err := filepath.WalkDir(m.MediaDir, func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case path == media("movies"):
			return fs.SkipDir
		case d.IsDir():
			return nil
		}
		rel, err := filepath.Rel(m.MediaDir, path)
		data, _ := os.ReadFile(path)
		got = append(got, filepath.ToSlash(rel)+" "+string(data))
		return err //nolint:wrapcheck // the test reports it as is
	})
	if want := []string{"Heat (1995)/Heat.mkv a film", "shelf/a.mkv a film"}; err != nil || !slices.Equal(got, want) {
		t.Errorf("the media dir holds %q, %v; want %q", got, err, want)
	}
	var starts []string
	for _, line := range journal(t, runDir) {
		if strings.HasPrefix(line, "step.start") {
			starts = append(starts, line)
		}
	}
	if !slices.Equal(starts, wantStarts) {
		t.Fatalf("the journal starts\n%s\nwant\n%s", strings.Join(starts, "\n"), strings.Join(wantStarts, "\n"))
	}
}

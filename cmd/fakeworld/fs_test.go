package main_test

import (
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

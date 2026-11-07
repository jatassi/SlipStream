package provenance

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/ports/port"
)

const parserCS = "src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBitsParser.cs"

// prowlarr is a Prowlarr repository with one commit holding files, and
// that commit.
func prowlarr(t *testing.T, files ...string) (tree *Tree, commit string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir := t.TempDir()
	for _, f := range files {
		write(t, dir, f, "namespace NzbDrone;\n")
	}
	git := func(args ...string) string {
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("add", ".")
	git("commit", "-q", "-m", "Fixture")
	return &Tree{Dir: dir}, git("rev-parse", "HEAD")
}

func write(t *testing.T, root, file, content string) {
	t.Helper()
	name := filepath.Join(root, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(name), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// repo is a SlipStream repository with an HDBits port ported at commit
// from the two files HDBits maps, and a skipped.json.
func repo(t *testing.T, commit string) string {
	t.Helper()
	root := t.TempDir()
	u := Unit{ID: "port:hdbits", Kind: Port, Dir: "internal/ports/hdbits", Manifest: Manifest{
		Commit: commit, Date: "2026-11-06",
		Files: map[string][]string{
			"hdbits.go": {"src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBits.cs"},
			"parser.go": {parserCS},
		},
	}}
	write(t, root, "internal/ports/hdbits/hdbits.go", "package hdbits\n")
	write(t, root, "internal/ports/hdbits/parser.go", "package hdbits\n")
	write(t, root, "internal/ports/hdbits/own.go", "package hdbits\n")
	if err := Write(root, u); err != nil {
		t.Fatal(err)
	}
	write(t, root, SkippedPath, string(Skipped{Commit: commit, Sites: []SkippedSite{
		{Name: "Cardigann", Path: "src/NzbDrone.Core/Indexers/Definitions/Cardigann/", Why: WhyGeneric, Reason: "It runs the catalog."},
	}}.Marshal()))
	return root
}

func messages(problems []Problem) []string {
	out := make([]string, len(problems))
	for i, p := range problems {
		out[i] = p.String()
	}
	return out
}

func checkOf(t *testing.T, root string, tree *Tree) []string {
	t.Helper()
	problems, err := Check(root, tree)
	if err != nil {
		t.Fatal(err)
	}
	return messages(problems)
}

var hdbitsFiles = []string{
	"src/NzbDrone.Core/Indexers/Definitions/HDBits/HDBits.cs",
	parserCS,
	"src/NzbDrone.Core/Indexers/Definitions/Cardigann/Cardigann.cs",
}

func TestAPortThatAgreesWithItsManifestAndProwlarrPasses(t *testing.T) {
	tree, commit := prowlarr(t, hdbitsFiles...)
	root := repo(t, commit)
	if got := checkOf(t, root, tree); len(got) > 0 {
		t.Fatalf("Check = %q, want no problems", got)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash("internal/ports/hdbits/"+ManifestName)))
	if err != nil {
		t.Fatal(err)
	}
	if u := port.ReadUpstream(raw); u.Commit != commit || len(u.Files) != 2 {
		t.Errorf("port.ReadUpstream = %+v, want the manifest Write wrote", u)
	}
}

func TestAPortWithNoManifestFails(t *testing.T) {
	_, commit := prowlarr(t, hdbitsFiles...)
	root := repo(t, commit)
	write(t, root, "internal/ports/torrentbytes/torrentbytes.go", "package torrentbytes\n")
	write(t, root, "internal/ports/internal/gazelle/gazelle.go", "package gazelle\n")
	want := []string{
		"internal/ports/internal/gazelle: has no upstream.json",
		"internal/ports/torrentbytes: has no upstream.json",
	}
	if got := checkOf(t, root, nil); !slices.Equal(got, want) {
		t.Errorf("Check = %q, want %q", got, want)
	}
}

func TestAFileMissingAtItsCommitFails(t *testing.T) {
	tree, commit := prowlarr(t, hdbitsFiles[0], hdbitsFiles[2])
	root := repo(t, commit)
	if got := checkOf(t, root, nil); len(got) > 0 {
		t.Fatalf("Check with no tree = %q, want no problems", got)
	}
	want := []string{"internal/ports/hdbits/upstream.json: " + parserCS + " isn't in Prowlarr at " + commit}
	if got := checkOf(t, root, tree); !slices.Equal(got, want) {
		t.Errorf("Check = %q, want %q", got, want)
	}
}

func TestAHeaderThatDisagreesWithTheManifestFails(t *testing.T) {
	_, commit := prowlarr(t, hdbitsFiles...)
	other := strings.Repeat("1", 40)
	for _, c := range []struct {
		name, file, content, want string
	}{
		{"another commit", "parser.go", Header(other, []string{parserCS}) + "\npackage hdbits\n",
			"internal/ports/hdbits/parser.go: its header names commit " + other + ", and upstream.json names " + commit},
		{"another file", "parser.go", Header(commit, []string{"src/HDBitsApi.cs"}) + "\npackage hdbits\n",
			"internal/ports/hdbits/parser.go: its header names src/HDBitsApi.cs, and upstream.json maps it to " + parserCS},
		{"no header", "parser.go", "package hdbits\n",
			"internal/ports/hdbits/parser.go: has no Prowlarr header, though upstream.json maps it"},
		{"a header on a file the manifest doesn't map", "own.go", Header(commit, []string{parserCS}) + "\npackage hdbits\n",
			"internal/ports/hdbits/own.go: has a Prowlarr header, though upstream.json maps no files to it"},
		{"a header off the template", "parser.go", "// Ported from Prowlarr's " + parserCS + "\n// at main.\n\npackage hdbits\n",
			"internal/ports/hdbits/parser.go: the header isn't the template provenance.Header writes"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := repo(t, commit)
			write(t, root, "internal/ports/hdbits/"+c.file, c.content)
			if got := checkOf(t, root, nil); !slices.Equal(got, []string{c.want}) {
				t.Errorf("Check = %q, want %q", got, c.want)
			}
		})
	}
}

func TestBumpMovesTheManifestAndEveryHeader(t *testing.T) {
	tree, commit := prowlarr(t, hdbitsFiles...)
	root := repo(t, commit)
	write(t, root, "internal/ports/hdbits/parser.go", "package hdbits\n\nfunc parse() {}\n")
	next := strings.Repeat("2", 40)
	if err := Bump(root, "internal/ports/hdbits", next, "2026-12-01"); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash("internal/ports/hdbits/parser.go")))
	if err != nil {
		t.Fatal(err)
	}
	if want := Header(next, []string{parserCS}) + "\npackage hdbits\n\nfunc parse() {}\n"; string(src) != want {
		t.Errorf("parser.go after Bump =\n%s\nwant\n%s", src, want)
	}
	if got := checkOf(t, root, nil); len(got) > 0 {
		t.Errorf("Check after Bump = %q, want no problems", got)
	}
	if got := checkOf(t, root, tree); len(got) != 1 || !strings.Contains(got[0], "commit "+next+" isn't in the Prowlarr clone") {
		t.Errorf("Check against a clone without the commit = %q", got)
	}
}

func TestTheRepositorysProvenanceAgreesWithItsCode(t *testing.T) {
	problems, err := Check(filepath.Join("..", "..", ".."), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range problems {
		t.Error(p)
	}
}

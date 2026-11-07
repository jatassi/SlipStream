package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/ports/provenance"
)

var update = flag.Bool("update", false, "rewrite testdata/report.md and testdata/report.json")

const defs = definitionsDir

// fixture is a Prowlarr repository whose history plays every change the
// report lists, built commit by commit with fixed dates so its hashes
// never change.
type fixture struct {
	t    *testing.T
	dir  string
	day  int
	base string
}

func (f *fixture) git(args ...string) string {
	f.t.Helper()
	cmd := exec.CommandContext(f.t.Context(), "git", append([]string{"-C", f.dir, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.test", "-c", "commit.gpgsign=false", "-c", "init.defaultBranch=develop"}, args...)...)
	date := fmt.Sprintf("2026-01-%02dT12:00:00Z", f.day)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commit writes files ("" deletes one) and commits them as subject.
func (f *fixture) commit(subject string, files map[string]string) string {
	f.t.Helper()
	for name, content := range files {
		p := filepath.Join(f.dir, filepath.FromSlash(name))
		if content == "" {
			if err := os.RemoveAll(p); err != nil {
				f.t.Fatal(err)
			}
			continue
		}
		write(f.t, p, content)
	}
	f.day++
	f.git("add", "-A")
	f.git("commit", "-q", "-m", subject)
	return f.git("rev-parse", "HEAD")
}

func write(t *testing.T, name, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func site(attribute, name, parent, categories string) string {
	return "namespace NzbDrone.Core.Indexers.Definitions;\n\n" + attribute +
		"public class " + name + " : " + parent + "\n{\n    private IndexerCapabilities SetCapabilities()\n    {\n" + categories + "    }\n}\n"
}

func mapping(category string) string {
	return "        caps.Categories.AddCategoryMapping(1, NewznabStandardCategory." + category + ", \"x\");\n"
}

func blocklist(ids ...string) string {
	return "public class IndexerDefinitionUpdateService\n{\n    private readonly List<string> _definitionBlocklist = new()\n    {\n        \"" +
		strings.Join(ids, "\",\n        \"") + "\"\n    };\n}\n"
}

// history builds the fixture's Prowlarr repository and the SlipStream
// repository ported from its first commit.
func history(t *testing.T) (prowlarrDir, repo string) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	f := &fixture{t: t, dir: t.TempDir(), day: 1}
	f.git("init", "-q")
	f.base = f.commit("Initial", map[string]string{
		"src/NzbDrone.Core/Indexers/TorrentIndexerBase.cs": "public abstract class TorrentIndexerBase<TSettings> : HttpIndexerBase<TSettings>\n{\n}\n",
		defs + "HDBits/HDBits.cs":                          site("", "HDBits", "TorrentIndexerBase<HDBitsSettings>", mapping("MoviesHD")),
		defs + "HDBits/HDBitsParser.cs":                    "public class HDBitsParser : IParseIndexerResponse\n{\n    // freeleech is 0\n}\n",
		defs + "Gazelle/GazelleBase.cs":                    "public abstract class GazelleBase<TSettings> : TorrentIndexerBase<TSettings>\n{\n    // page size 50\n}\n",
		defs + "AlphaRatio.cs":                             site("", "AlphaRatio", "GazelleBase<AlphaRatioSettings>", mapping("TV")+mapping("Movies")),
		defs + "FunFile.cs":                                site("", "FunFile", "TorrentIndexerBase<UserPassTorrentBaseSettings>", mapping("TVSD")),
		defs + "ZonaQ.cs":                                  site("", "ZonaQ", "TorrentIndexerBase<UserPassTorrentBaseSettings>", mapping("MoviesForeign")),
		defs + "Rarbg/Rarbg.cs":                            site("", "Rarbg", "TorrentIndexerBase<RarbgSettings>", mapping("Movies")),
		defs + "Rarbg/RarbgParser.cs":                      "public class RarbgParser\n{\n}\n",
		defs + "Cardigann/Cardigann.cs":                    site("", "Cardigann", "TorrentIndexerBase<CardigannSettings>", ""),
		defs + "Unit3d/Unit3dBase.cs":                      "public abstract class Unit3dBase : TorrentIndexerBase<Unit3dSettings>\n{\n}\n",
		defs + "Newznab/Newznab.cs":                        "public class Newznab : UsenetIndexerBase<NewznabSettings>\n{\n}\n",
		hiddenListFile:                                     blocklist("hdbits", "beyond-hd"),
	})
	f.commit("Fixed: (HDBits) Parse freeleech from the API", map[string]string{
		defs + "HDBits/HDBitsParser.cs": "public class HDBitsParser : IParseIndexerResponse\n{\n    // freeleech is \"yes\"\n}\n",
	})
	f.commit("New: (Indexer) DesiGaane", map[string]string{
		defs + "DesiGaane.cs": site("", "DesiGaane", "GazelleBase<GazelleSettings>", mapping("Audio")+mapping("AudioLossless")+mapping("MoviesForeign")),
	})
	f.commit("Fixed: (ZonaQ) Obsolete per site policy", map[string]string{
		defs + "ZonaQ.cs": site("[Obsolete(\"Site does not allow automation\")]\n", "ZonaQ", "TorrentIndexerBase<UserPassTorrentBaseSettings>", mapping("MoviesForeign")),
	})
	f.commit("Remove Rarbg", map[string]string{defs + "Rarbg": ""})
	f.commit("Fixed: (FunFile) Moved to Cardigann", map[string]string{
		defs + "FunFile.cs": site("[Obsolete(\"Moved to YML for Cardigann\")]\n", "FunFile", "TorrentIndexerBase<UserPassTorrentBaseSettings>", mapping("TVSD")),
	})
	f.commit("Hide the FunFile definition", map[string]string{hiddenListFile: blocklist("hdbits", "beyond-hd", "funfile")})
	f.commit("Fixed: (Gazelle) Page size 100 | for every site", map[string]string{
		defs + "Gazelle/GazelleBase.cs": "public abstract class GazelleBase<TSettings> : TorrentIndexerBase<TSettings>\n{\n    // page size 100\n}\n",
	})

	repo = t.TempDir()
	units := []provenance.Unit{
		{Dir: "internal/ports/hdbits", Manifest: provenance.Manifest{Files: map[string][]string{
			"hdbits.go": {defs + "HDBits/HDBits.cs"}, "parser.go": {defs + "HDBits/HDBitsParser.cs"},
		}}},
		{Dir: "internal/ports/alpharatio", Manifest: provenance.Manifest{Family: "gazelle", Files: map[string][]string{"alpharatio.go": {defs + "AlphaRatio.cs"}}}},
		{Dir: "internal/ports/funfile", Manifest: provenance.Manifest{Files: map[string][]string{"funfile.go": {defs + "FunFile.cs"}}}},
		{Dir: "internal/ports/zonaq", Manifest: provenance.Manifest{Files: map[string][]string{"zonaq.go": {defs + "ZonaQ.cs"}}}},
		{Dir: "internal/ports/rarbg", Manifest: provenance.Manifest{Files: map[string][]string{
			"rarbg.go": {defs + "Rarbg/Rarbg.cs", defs + "Rarbg/RarbgParser.cs"},
		}}},
		{Dir: "internal/ports/internal/gazelle", Manifest: provenance.Manifest{Files: map[string][]string{"gazelle.go": {defs + "Gazelle/GazelleBase.cs"}}}},
		{Dir: "internal/ports/port", Manifest: provenance.Manifest{Files: map[string][]string{"port.go": {"src/NzbDrone.Core/Indexers/TorrentIndexerBase.cs"}}}},
	}
	for _, u := range units {
		u.Commit, u.Date = f.base, "2026-01-02"
		for gofile := range u.Files {
			write(t, filepath.Join(repo, filepath.FromSlash(u.Dir), gofile), "package "+filepath.Base(u.Dir)+"\n")
		}
		if err := provenance.Write(repo, u); err != nil {
			t.Fatal(err)
		}
	}
	err := provenance.WriteSkipped(repo, provenance.Skipped{Commit: f.base, Sites: []provenance.SkippedSite{
		{Name: "Cardigann", Path: defs + "Cardigann/", Why: provenance.WhyGeneric, Reason: "It runs the catalog."},
		{Name: "Unit3dBase", Path: defs + "Unit3d/", Why: provenance.WhyUnused, Reason: "No live site builds on it."},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return f.dir, repo
}

func runSync(t *testing.T, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	if err := run(t.Context(), args, &out); err != nil {
		t.Fatalf("prowlarr-sync %v: %v", args, err)
	}
	return out.String()
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		write(t, path, got)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Errorf("the report differs from %s; run go test ./cmd/prowlarr-sync -update and read the diff\ngot:\n%s", path, got)
	}
}

func TestTheReportListsEveryUpstreamChangeTheSameWayTwice(t *testing.T) {
	prowlarr, repo := history(t)
	for _, format := range []struct{ name, flag string }{{"report.md", ""}, {"report.json", "--json"}} {
		args := []string{"report", "--repo", repo, "--prowlarr", prowlarr}
		if format.flag != "" {
			args = append(args, format.flag)
		}
		first := runSync(t, args...)
		golden(t, format.name, first)
		if again := runSync(t, args...); again != first {
			t.Errorf("a second %s differs from the first", format.name)
		}
	}
}

func TestBumpBringsAUnitUpToDate(t *testing.T) {
	prowlarr, repo := history(t)
	runSync(t, "bump", "--repo", repo, "--prowlarr", prowlarr, "--unit", "family:gazelle", "--to", "develop")
	runSync(t, "bump", "--repo", repo, "--prowlarr", prowlarr, "--unit", "hidden-list", "--to", "develop")
	report := runSync(t, "report", "--repo", repo, "--prowlarr", prowlarr)
	for _, gone := range []string{"### family:gazelle@", "### hidden-list@"} {
		if strings.Contains(report, gone) {
			t.Errorf("the report after bump still lists %s", gone)
		}
	}
	if !strings.Contains(report, "- family:gazelle at ") {
		t.Errorf("the report after bump doesn't list family:gazelle as up to date:\n%s", report)
	}
	if got := runSync(t, "check", "--repo", repo, "--prowlarr", prowlarr); !strings.Contains(got, "matches its code") {
		t.Errorf("check after bump = %q", got)
	}
}

func TestNewWritesAPortThatPassesTheCheck(t *testing.T) {
	prowlarr, repo := history(t)
	runSync(t, "new", "--repo", repo, "--prowlarr", prowlarr, "--site", "DesiGaane", "--family", "gazelle")
	src, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash("internal/ports/desigaane/desigaane.go")))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(src), "// Ported from Prowlarr's "+defs+"DesiGaane.cs\n") || !strings.HasSuffix(string(src), "\npackage desigaane\n") {
		t.Errorf("desigaane.go =\n%s", src)
	}
	for _, f := range []string{"internal/ports/all/desigaane.go", "fakes/site/ports/desigaane/fixture.yaml"} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(f))); err != nil {
			t.Errorf("new didn't write %s: %v", f, err)
		}
	}
	report := runSync(t, "report", "--repo", repo, "--prowlarr", prowlarr)
	if strings.Contains(report, "### new:DesiGaane@") {
		t.Error("the report still lists DesiGaane as new once it has a port")
	}
	if got := runSync(t, "check", "--repo", repo, "--prowlarr", prowlarr); !strings.Contains(got, "matches its code") {
		t.Errorf("check after new = %q", got)
	}
}

func TestGoFileNamesFollowTheCSharpFiles(t *testing.T) {
	for _, c := range []struct{ file, want string }{
		{defs + "HDBits/HDBits.cs", "hdbits.go"},
		{defs + "HDBits/HDBitsParser.cs", "parser.go"},
		{defs + "HDBits/HDBitsRequestGenerator.cs", "requestgenerator.go"},
		{defs + "MTeamTp.cs", "mteamtp.go"},
	} {
		site := strings.TrimSuffix(filepath.Base(c.file), ".cs")
		if strings.HasPrefix(c.file, defs+"HDBits/") {
			site = "HDBits"
		}
		if got := goFile(site, strings.ToLower(site), c.file); got != c.want {
			t.Errorf("goFile(%s) = %s, want %s", c.file, got, c.want)
		}
	}
}

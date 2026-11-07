package dom

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

var (
	prowlarr = flag.String("prowlarr", "", "a Prowlarr checkout to read the sites' selectors from")
	update   = flag.Bool("update", false, "rewrite testdata/upstream-selectors.json from -prowlarr")
)

// siteSources are the C# files of the 50 ported sites and their family
// bases, under src/NzbDrone.Core/Indexers/Definitions.
var siteSources = []string{
	"AlphaRatio.cs", "Anidex.cs", "Anidub.cs", "AnimeBytes.cs", "AnimeZ.cs", "AvistaZ.cs", "BakaBT.cs",
	"BeyondHD.cs", "BitHDTV.cs", "BrokenStones.cs", "CinemaZ.cs", "DICMusic.cs", "DesiGaane.cs",
	"ExoticaZ.cs", "GazelleGames.cs", "GreatPosterWall.cs", "HDTorrents.cs", "IPTorrents.cs",
	"ImmortalSeed.cs", "Knaben.cs", "Libble.cs", "MTeamTp.cs", "MyAnonamouse.cs", "Nebulance.cs",
	"NorBits.cs", "Orpheus.cs", "PixelHD.cs", "PreToMe.cs", "PrivateHD.cs", "Redacted.cs", "RetroFlix.cs",
	"RevolutionTT.cs", "RuTracker.cs", "SceneHD.cs", "SecretCinema.cs", "Shazbat.cs", "SpeedApp.cs",
	"SpeedCD.cs", "SubsPlease.cs", "Toloka.cs", "TorrentBytes.cs", "TorrentDay.cs", "TorrentSyndikat.cs",
	"TorrentsCSV.cs", "Uniotaku.cs", "XSpeeds.cs",
	"Avistaz", "BroadcastheNet", "FileList", "Gazelle", "HDBits", "PassThePopcorn", "SpeedApp",
}

// upstreamSelector is one CSS selector a site's C# code passes to
// QuerySelector, QuerySelectorAll or Matches, or keeps in a constant for
// them. An interpolated hole such as {groupId} reads as 1.
type upstreamSelector struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Selector string `json:"selector"`
}

func TestUpstreamSelectorsCompile(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "upstream-selectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var sels []upstreamSelector
	if err := json.Unmarshal(raw, &sels); err != nil {
		t.Fatal(err)
	}
	if len(sels) == 0 {
		t.Fatal("testdata/upstream-selectors.json lists no selectors")
	}
	for _, s := range sels {
		if _, err := Compile(s.Selector); err != nil {
			t.Errorf("%s:%d: %v", s.File, s.Line, err)
		}
	}
}

func TestUpstreamSelectorsAreCurrent(t *testing.T) {
	if *prowlarr == "" {
		t.Skip("pass -prowlarr <checkout> to compare with Prowlarr's sites")
	}
	got := extractSelectors(t, *prowlarr)
	if *update {
		out, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join("testdata", "upstream-selectors.json"), append(out, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "upstream-selectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var want []upstreamSelector
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Error("testdata/upstream-selectors.json is out of date; rerun with -update")
	}
}

var (
	selectorCall  = regexp.MustCompile(`\b(?:QuerySelector|QuerySelectorAll|Matches)\(\s*([$@]{0,2}")`)
	selectorConst = regexp.MustCompile(`(?i)\b(?:const\s+string|var)\s+\w*selector\w*\s*=\s*([$@]{0,2}")`)
	hole          = regexp.MustCompile(`\{[^{}]*\}`)
)

func extractSelectors(t *testing.T, root string) []upstreamSelector {
	t.Helper()
	var out []upstreamSelector
	for _, f := range sourceFiles(t, root) {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		rel, err := filepath.Rel(root, f)
		if err != nil {
			t.Fatal(err)
		}
		sels, err := selectorsIn(filepath.ToSlash(rel), string(raw))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, sels...)
	}
	return out
}

func sourceFiles(t *testing.T, root string) []string {
	t.Helper()
	base := filepath.Join(root, "src", "NzbDrone.Core", "Indexers", "Definitions")
	var files []string
	for _, src := range siteSources {
		path := filepath.Join(base, src)
		if strings.HasSuffix(src, ".cs") {
			files = append(files, path)
			continue
		}
		nested, err := filepath.Glob(filepath.Join(path, "*.cs"))
		if err != nil || len(nested) == 0 {
			t.Fatalf("%s has no C# files", path)
		}
		files = append(files, nested...)
	}
	return files
}

func selectorsIn(file, src string) ([]upstreamSelector, error) {
	var out []upstreamSelector
	for i, line := range strings.Split(src, "\n") {
		for _, re := range []*regexp.Regexp{selectorCall, selectorConst} {
			for _, m := range re.FindAllStringSubmatchIndex(line, -1) {
				s, ok := csharpString(line[m[2]:])
				if !ok {
					return nil, fmt.Errorf("%s:%d: can't read the string literal in %q", file, i+1, line)
				}
				out = append(out, upstreamSelector{File: file, Line: i + 1, Selector: s})
			}
		}
	}
	return out, nil
}

// csharpString reads the C# string literal s starts with: regular,
// verbatim (@) or interpolated ($), with each hole read as 1.
func csharpString(s string) (string, bool) {
	prefix := s[:strings.IndexByte(s, '"')]
	verbatim, interpolated := strings.Contains(prefix, "@"), strings.Contains(prefix, "$")
	var b strings.Builder
	for i := len(prefix) + 1; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' && verbatim && i+1 < len(s) && s[i+1] == '"':
			b.WriteByte('"')
			i++
		case c == '"':
			out := b.String()
			if interpolated {
				out = hole.ReplaceAllString(strings.NewReplacer("{{", "\x00", "}}", "\x01").Replace(out), "1")
				out = strings.NewReplacer("\x00", "{", "\x01", "}").Replace(out)
			}
			return out, true
		case c == '\\' && !verbatim && i+1 < len(s):
			i++
			b.WriteByte(escape(s[i]))
		default:
			b.WriteByte(c)
		}
	}
	return "", false
}

// escape is the character a C# backslash escape stands for.
func escape(c byte) byte {
	switch c {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'r':
		return '\r'
	case '0':
		return 0
	}
	return c
}

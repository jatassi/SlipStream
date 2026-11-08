package release_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
)

func qualityKey(f release.Fact[release.Quality]) string {
	if q, ok := f.Get(); ok {
		return q.String()
	}
	return ""
}

var titleQualities = map[string]string{
	"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW":  "webdl-1080p",
	"The Quiet Orchard 2021 1080p WEBDL DDP5.1":     "webdl-1080p",
	"The.Quiet.Orchard.2021.2160p.WEB.H265":         "webdl-2160p",
	"The.Quiet.Orchard.2021.1080p.WEBRip.x264":      "webrip-1080p",
	"The Quiet Orchard (2021) [720p] [WEB-Rip]":     "webrip-720p",
	"The.Quiet.Orchard.2021.1080p.BluRay.x264":      "bluray-1080p",
	"The.Quiet.Orchard.2021.720p.Blu-Ray.DTS":       "bluray-720p",
	"The.Quiet.Orchard.2021.1080p.BDRip":            "bluray-1080p",
	"The.Quiet.Orchard.2021.1080p.BRRip":            "bluray-1080p",
	"The.Quiet.Orchard.2021.2160p.UHD.BluRay.REMUX": "remux-2160p",
	"The.Quiet.Orchard.2021.1080p.BDREMUX":          "remux-1080p",
	"The.Quiet.Orchard.2021.4K.HDR.BluRay":          "bluray-720p",
	"The.Quiet.Orchard.2021.4K.UHD.BluRay":          "bluray-2160p",
	"The.Quiet.Orchard.2021.720p.HDTV.x264":         "hdtv-720p",
	"The.Quiet.Orchard.S01E01.SDTV.XviD":            "sdtv",
	"The.Quiet.Orchard.2021.TVRip":                  "sdtv",
	"The.Quiet.Orchard.2021.DVDRip.XviD":            "dvd",
	"The.Quiet.Orchard.2021.576p.DVD":               "dvd",
	"The.Quiet.Orchard.2021.480p.WEBRip":            "webrip-480p",
	"The.Quiet.Orchard.2021.480p.WEB-DL":            "sdtv",
	"The.Quiet.Orchard.2021.480p.1080p.WEB-DL":      "sdtv",
	"The.Quiet.Orchard.2021.720p.Remux":             "hdtv-720p",

	"The.Quiet.Orchard.2021.CAM":                   "cam",
	"The.Quiet.Orchard.2021.HDCAM.x264":            "cam",
	"The.Quiet.Orchard.2021.TS.XviD":               "cam",
	"The.Quiet.Orchard.2021.720p.HDTS":             "cam",
	"The.Quiet.Orchard.2021.1080p.TELESYNC":        "cam",
	"The.Quiet.Orchard.2021.2160p.WEB-DL.TS":       "webdl-2160p",
	"The.Quiet.Orchard.2021.2160p.WEB-DL.TS.x265":  "cam",
	"The.Quiet.Orchard.2021.1080p.TELESYNC.BluRay": "cam",

	"Pirates.of.the.Orchard.2021.1080p.WEB-DL":             "webdl-1080p",
	"Lights.Out.2021.720p.BluRay":                          "bluray-720p",
	"Ghosts.2021.1080p":                                    "hdtv-1080p",
	"Its.Always.Orchard.S01E01.720p":                       "hdtv-720p",
	"Tsunami.Orchard.2021.1080p.WEBRip":                    "webrip-1080p",
	"T.S.Orchard.2021.1080p.BluRay":                        "bluray-1080p",
	"The.Orchard.Tapes.2021.2160p.WEB-DL":                  "webdl-2160p",
	"The.Cambridge.Orchard.2021.1080p.WEBRip":              "webrip-1080p",
	"Camera.Obscura.2021.720p":                             "hdtv-720p",
	"The.Quiet.Orchard.2021.1080p.WEB-DL.TrueHD.Atmos-TSG": "webdl-1080p",

	"The.Quiet.Orchard.2021.720p":     "hdtv-720p",
	"The.Quiet.Orchard.2021.1080p":    "hdtv-1080p",
	"The Quiet Orchard 2021 UHD":      "hdtv-2160p",
	"The.Quiet.Orchard.2021.576p":     "sdtv",
	"The.Quiet.Orchard.2021.480p":     "sdtv",
	"The.Quiet.Orchard.2021.WEB-DL":   "webdl-720p",
	"The.Quiet.Orchard.2021.WEBRip":   "webrip-480p",
	"The.Quiet.Orchard.2021.BluRay":   "bluray-720p",
	"The.Quiet.Orchard.2021.Remux":    "remux-1080p",
	"The.Quiet.Orchard.2021.HDTV":     "hdtv-720p",
	"The.Quiet.Orchard.2021":          "",
	"The.Quiet.Orchard.2021.1080i":    "hdtv-1080p",
	"The.Quiet.Orchard.2021.x264-FKW": "",
	"":                                "",
}

func TestTitleQuality(t *testing.T) {
	for title, want := range titleQualities {
		if got := qualityKey(release.Title(title, release.TitleOnly{}, release.BuiltIn()).Facts.Quality); got != want {
			t.Errorf("Title(%q) quality = %q, want %q", title, got, want)
		}
		if got := qualityKey(release.QualityIn(title)); got != want {
			t.Errorf("QualityIn(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestFileQualityTakesTheProbedResolution(t *testing.T) {
	for _, c := range []struct {
		name   string
		probed release.Resolution
		want   string
	}{
		{"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW", release.R720p, "webdl-720p"},
		{"The.Quiet.Orchard.2021.720p.BluRay", release.R2160p, "bluray-2160p"},
		{"The Quiet Orchard (2021) - 1080p.mkv", release.R1080p, "hdtv-1080p"},
		{"The.Quiet.Orchard.2021.Remux.mkv", release.R1080p, "remux-1080p"},
		{"The.Quiet.Orchard.2021.DVDRip.avi", release.R480p, "dvd"},
		{"The.Quiet.Orchard.2021.DVDRip.avi", release.R576p, "dvd"},
		{"The.Quiet.Orchard.2021.1080p.TELESYNC.mkv", release.R1080p, "cam"},
		{"Pirates.of.the.Orchard.2021.mkv", release.R480p, "sdtv"},
	} {
		if got := qualityKey(release.File(c.name, release.TitleOnly{}, release.BuiltIn()).Facts.WithProbe(release.Probed{Resolution: release.Known(c.probed)}).Quality); got != c.want {
			t.Errorf("File(%q, %s) quality = %q, want %q", c.name, c.probed, got, c.want)
		}
	}
}

func TestCut(t *testing.T) {
	cuts := append(release.BuiltIn(),
		release.Cut{Key: "superfan", Name: "Superfan", Aliases: []string{"Superfan", "Superfan Cut"}},
		release.Cut{Key: "black-and-chrome", Name: "Black and Chrome", Aliases: []string{"Black and Chrome", "Black & Chrome"}},
		release.Cut{Key: "extended-directors-cut", Name: "Extended Director's Cut", Aliases: []string{"Extended Director's Cut"}},
	)
	for title, want := range map[string]release.CutKey{
		"The.Quiet.Orchard.2021.1080p.WEB-DL":                       release.Standard,
		"The.Quiet.Orchard.2021.Directors.Cut.1080p.BluRay":         "directors-cut",
		"The Quiet Orchard 2021 Director's Cut 1080p":               "directors-cut",
		"The Quiet Orchard 2021 Director’s Cut 1080p":               "directors-cut",
		"The.Quiet.Orchard.2021.Director.s.Cut.1080p":               "directors-cut",
		"The.Quiet.Orchard.2021.DIRECTORS-CUT.1080p":                "directors-cut",
		"The.Quiet.Orchard.2021.Director.Cut.1080p":                 "directors-cut",
		"The Quiet Orchard (2021) {edition-Director's Cut} - 1080p": "directors-cut",
		"The.Quiet.Orchard.2021.Directors.1080p":                    release.Standard,
		"The.Quiet.Orchard.2021.Extended.1080p":                     "extended",
		"The.Quiet.Orchard.2021.EXTENDED.EDITION.1080p":             "extended",
		"The.Quiet.Orchard.2021.Extended.Directors.Cut.1080p":       "extended-directors-cut",
		"The.Quiet.Orchard.2021.Ultimate.Extended.Edition.1080p":    "extended",
		"The.Quiet.Orchard.2021.Theatrical.1080p":                   "theatrical",
		"The.Quiet.Orchard.2021.UNRATED.1080p":                      "unrated",
		"The.Quiet.Orchard.2021.Uncut.1080p":                        "uncut",
		"The.Quiet.Orchard.2021.IMAX.2160p":                         "imax",
		"The.Quiet.Orchard.2021.Remastered.1080p":                   "remastered",
		"The.Quiet.Orchard.2021.Special.Edition.1080p":              "special-edition",
		"The.Quiet.Orchard.2021.Special.1080p":                      release.Standard,
		"The.Quiet.Orchard.2021.Final.Cut.1080p":                    "final-cut",
		"The.Quiet.Orchard.2021.Ultimate.1080p":                     "ultimate",
		"The.Quiet.Orchard.2021.Superfan.1080p":                     "superfan",
		"The.Quiet.Orchard.2021.Black.and.Chrome.1080p":             "black-and-chrome",
		"The.Quiet.Orchard.2021.Black.&.Chrome.1080p":               "black-and-chrome",
		"The.Quiet.Orchard.2021.Standard.1080p":                     release.Standard,
		"The.Quiet.Orchard.2021.Unextended.1080p":                   release.Standard,
		"The.Quiet.Orchard.2021.Uncut.Remastered.1080p":             "remastered",
		"The.Quiet.Orchard.2021.Remastered.Theatrical.1080p":        "remastered",
		"The.Quiet.Orchard.2021.Theatrical.Remastered.1080p":        "theatrical",
	} {
		if got := release.Title(title, release.TitleOnly{}, cuts).Facts.Cut; got != want {
			t.Errorf("Title(%q) cut = %q, want %q", title, got, want)
		}
		if got := release.File(title, release.TitleOnly{}, cuts).Facts.Cut; got != want {
			t.Errorf("File(%q) cut = %q, want %q", title, got, want)
		}
	}
	if got := release.Title("The.Quiet.Orchard.2021.Extended.1080p", release.TitleOnly{}, nil).Facts.Cut; got != release.Standard {
		t.Errorf("Title with no cuts known = %q, want standard", got)
	}
}

func TestBuiltInCuts(t *testing.T) {
	var keys []release.CutKey
	for _, c := range release.BuiltIn() {
		keys = append(keys, c.Key)
		if c.Key != release.Standard && len(c.Aliases) == 0 {
			t.Errorf("built-in cut %s has no aliases", c.Key)
		}
	}
	want := []release.CutKey{"standard", "theatrical", "extended", "directors-cut", "unrated", "uncut", "imax", "remastered", "special-edition", "final-cut", "ultimate"}
	if len(keys) != len(want) {
		t.Fatalf("BuiltIn() = %v, want %v", keys, want)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("BuiltIn() = %v, want %v", keys, want)
		}
	}
}

func TestACustomCutsSlugAndPhraseAreTheWordsATitleReads(t *testing.T) {
	for name, want := range map[string]release.CutKey{
		"Black and Chrome":    "black-and-chrome",
		"Director's Superfan": "directors-superfan",
		"  Fan.Edit  ":        "fan-edit",
		"Édition Spéciale":    "édition-spéciale",
		"?!":                  "",
	} {
		if got := release.Slug(name); got != want {
			t.Errorf("Slug(%q) = %q, want %q", name, got, want)
		}
	}
	if !release.SamePhrase("Black & Chrome", "BLACK CHROME") || release.SamePhrase("Black & Chrome", "Black and Chrome") {
		t.Error("SamePhrase reads & as a separator between words, not as and")
	}
	if !release.SamePhrase("Fan's Cut", "FANS-CUT") || release.SamePhrase("Fan Cut", "Fan Cuts") {
		t.Error("SamePhrase must match the same words, case and marks aside, and nothing more")
	}
}

// titled reads every name as Title, then a year, then the tail.
type titled struct{ release.TitleOnly }

func (titled) ReadName(p release.Prepared) release.Name {
	title, tail, _ := strings.Cut(p.Title.String(), ".2019.")
	return release.Name{Titles: []string{title}, Year: release.Known(2019), Tail: tail}
}

func TestTheCutIsReadAfterTheTitle(t *testing.T) {
	r := release.Title("Uncut.Gems.2019.Directors.Cut.1080p.WEB-DL", titled{}, release.BuiltIn())
	if r.Facts.Cut != "directors-cut" || !slices.Equal(r.Name.Titles, []string{"Uncut.Gems"}) || r.Given != "Uncut.Gems.2019.Directors.Cut.1080p.WEB-DL" {
		t.Fatalf("Title read %+v, want the type's title and a Director's Cut", r)
	}
	if got := release.Title("Uncut.Gems.2019.1080p.WEB-DL", titled{}, release.BuiltIn()).Facts.Cut; got != release.Standard {
		t.Errorf("Uncut Gems reads as the %s cut, want standard: its title names no cut", got)
	}
	if got := release.Title("Uncut.Gems.2019.1080p.WEB-DL", release.TitleOnly{}, release.BuiltIn()).Facts.Cut; got != "uncut" {
		t.Errorf("with no title read, Uncut Gems reads as the %s cut, want uncut", got)
	}
}

func TestAReadStoresAsJSON(t *testing.T) {
	r := release.Title("The.Quiet.Orchard.2021.1080p.BDREMUX", release.TitleOnly{}, release.BuiltIn())
	r.Facts.Size = release.Known(int64(9_876_543_210))
	r.Facts.Flags = []release.Flag{release.Freeleech}
	r.Name.Span = release.One
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"Quality":"remux-1080p"`, `"Source":"bluray"`, `"Modifier":"remux"`, `"Resolution":"1080p"`, `"Group":null`, `"Span":"one"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("%s lacks %s", b, want)
		}
	}
	var back release.Read
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, r) {
		t.Fatalf("%s reads back as %+v, want %+v", b, back, r)
	}
}

func TestAProbeWinsOverTheName(t *testing.T) {
	f := release.Title("The.Quiet.Orchard.2021.2160p.HDTV", release.TitleOnly{}, nil).Facts
	if got := qualityKey(f.WithProbe(release.Probed{}).Quality); got != "hdtv-2160p" {
		t.Errorf("with nothing probed the quality is %q, want the name's hdtv-2160p", got)
	}
	probed := f.WithProbe(release.Probed{Resolution: release.Known(release.R480p), Video: release.Known[release.VideoCodec]("x265")})
	if v, _ := probed.Video.Get(); qualityKey(probed.Quality) != "sdtv" || v != "x265" {
		t.Errorf("probed at 480p in x265, the file reads as %q in %q, want sdtv in x265", qualityKey(probed.Quality), v)
	}
	french := release.Title("Orchard.2021.FRENCH.1080p.WEB-DL", release.TitleOnly{}, nil).Facts
	audio := french.WithProbe(release.Probed{Languages: release.Known([]release.Language{"de"})})
	if got, _ := audio.Languages.Get(); !reflect.DeepEqual(got, []release.Language{"de"}) {
		t.Errorf("a file whose audio tracks probe as German reads as %v, want [de]", got)
	}
}

func TestRevisionsCompareRealThenVersion(t *testing.T) {
	for _, c := range []struct {
		a, b release.Revision
		want int
	}{
		{release.Revision{}, release.Revision{Version: 1}, 0},
		{release.Revision{Version: 2}, release.Revision{Version: 1, Repack: true}, 1},
		{release.Revision{Version: 1, Real: 1}, release.Revision{Version: 3}, 1},
		{release.Revision{Version: 2, Repack: true}, release.Revision{Version: 2}, 0},
	} {
		if got := c.a.Compare(c.b); got != c.want {
			t.Errorf("%+v.Compare(%+v) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestEveryFlagParses(t *testing.T) {
	for _, f := range release.Flags() {
		if got, ok := release.ParseFlag(string(f)); !ok || got != f {
			t.Errorf("ParseFlag(%q) = %q, %v", f, got, ok)
		}
	}
	if _, ok := release.ParseFlag("g_freeleech"); ok {
		t.Error("ParseFlag read a name no Flag has")
	}
}

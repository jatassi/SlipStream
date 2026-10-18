package release_test

import (
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
	"The.Quiet.Orchard.2021.4K.HDR.BluRay":          "bluray-2160p",
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
	"The.Quiet.Orchard.2021.2160p.WEB-DL.TS":       "cam",
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
	"The.Quiet.Orchard.2021.1080i":    "",
	"The.Quiet.Orchard.2021.x264-FKW": "",
	"":                                "",
}

func TestTitleQuality(t *testing.T) {
	for title, want := range titleQualities {
		if got := qualityKey(release.Title(title, release.BuiltIn()).Quality); got != want {
			t.Errorf("Title(%q) quality = %q, want %q", title, got, want)
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
		if got := qualityKey(release.File(c.name, release.BuiltIn(), c.probed).Quality); got != c.want {
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
		if got := release.Title(title, cuts).Cut; got != want {
			t.Errorf("Title(%q) cut = %q, want %q", title, got, want)
		}
		if got := release.File(title, cuts, release.R1080p).Cut; got != want {
			t.Errorf("File(%q) cut = %q, want %q", title, got, want)
		}
	}
	if got := release.Title("The.Quiet.Orchard.2021.Extended.1080p", nil).Cut; got != release.Standard {
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

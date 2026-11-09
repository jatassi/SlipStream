package naming

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/works"
)

var filmTokens = Declaration{Tokens: []Token{
	{Name: "Movie Title", Kind: KindTitle, Scope: WorkScope, Example: "The Quiet Orchard"},
	{Name: "Year", Aliases: []string{"Release Year"}, Kind: KindYear, Scope: WorkScope, Example: "2021"},
	{Name: "ImdbId", Kind: KindID, Scope: WorkScope},
	{Name: "TmdbId", Kind: KindID, Scope: WorkScope},
}}

var showTokens = Declaration{Tokens: []Token{
	{Name: "Series Title", Kind: KindTitle, Scope: WorkScope},
	{Name: "Series Year", Kind: KindYear, Scope: WorkScope},
	{Name: "season", Kind: KindNumber, Scope: LevelScope, Seq: SeqOuter},
	{Name: "episode", Kind: KindNumber, Scope: ItemScope, Seq: SeqInner},
	{Name: "Episode Title", Kind: KindTitle, Scope: ItemScope},
	{Name: "Air Date", Kind: KindDate, Scope: ItemScope},
}}

func mustCatalog(t *testing.T, d Declaration) *catalog {
	t.Helper()
	c, err := newCatalog(d)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// folderName renders text as a folder of d's: rendered, then the segment
// rule.
func folderName(t *testing.T, d Declaration, text string, s Subject, f FileFacts) string {
	t.Helper()
	return segment(rendered(t, d, style{colon: ColonSmart}, text, s, f))
}

// fileName renders text as a file of d's, the segment rule applied with
// the extension on.
func fileName(t *testing.T, d Declaration, text string, s Subject, f FileFacts) string {
	t.Helper()
	return segment(rendered(t, d, style{colon: ColonSmart}, text, s, f) + ".mkv")
}

func rendered(t *testing.T, d Declaration, st style, text string, s Subject, f FileFacts) string {
	t.Helper()
	tmpl, err := Parse(text)
	if err != nil {
		t.Fatal(err)
	}
	b, err := bind(tmpl, mustCatalog(t, d), fileField)
	if err != nil {
		t.Fatal(err)
	}
	return st.render(b, s, f)
}

func work(s Set) Subject { return Subject{Work: s} }

func episode(work, level, item Set) Subject {
	return Subject{Work: work, Units: []Unit{{Level: level, Item: ItemValues{Set: item}}}}
}

var heat = Set{"Movie Title": Text("Heat"), "Year": Number(1995), "ImdbId": Text("tt0113277")}

func TestRenderGrammar(t *testing.T) {
	noYear := Set{"Movie Title": Text("Heat")}
	cutFacts := FileFacts{Target: "4K HDR", Cut: "Director's Cut", Facts: release.Facts{Group: release.Known("FLUX")}}
	plain := FileFacts{Target: "4K HDR"}
	for _, c := range []struct {
		text string
		s    Set
		f    FileFacts
		want string
	}{
		{"{Movie Title} {(Year)}", heat, plain, "Heat (1995)"},
		{"{Movie Title} {(Year)}", noYear, plain, "Heat"},
		{"{Movie Title} {(Year)} - {Edition Tag} {Target}", noYear, plain, "Heat - 4K HDR"},
		{"{Movie Title} {(Year)} - {Edition Tag} {Target}", heat, cutFacts, "Heat (1995) - {edition-Director's Cut} 4K HDR"},
		{"{Movie Title}{-Release Group}", heat, cutFacts, "Heat-FLUX"},
		{"{Movie Title}{-Release Group}", heat, plain, "Heat"},
		{"{Movie Title} {{Edition}}", heat, cutFacts, "Heat {Director's Cut}"},
		{"{Movie Title} {{Edition}}", heat, plain, "Heat"},
		{"{Movie Title} {edition-{Edition}}", heat, cutFacts, "Heat {edition-Director's Cut}"},
		{"{Movie Title} {edition-{Edition}} {Target}", heat, plain, "Heat 4K HDR"},
		{"{Movie Title} {imdb-{ImdbId}}", heat, plain, "Heat {imdb-tt0113277}"},
		{"{Movie Title} {imdb-{ImdbId}} - {Target}", noYear, plain, "Heat - 4K HDR"},
		{"{Movie Title} {tmdbid-{TmdbId}}", heat, plain, "Heat"},
		{"{Movie.Title}.{Year}", Set{"Movie Title": Text("The Quiet Orchard"), "Year": Number(2021)}, plain, "The.Quiet.Orchard.2021"},
		{"{Movie_Title}", Set{"Movie Title": Text("The Quiet Orchard")}, plain, "The_Quiet_Orchard"},
		{"{movie title} {TARGET}", Set{"Movie Title": Text("The Quiet Orchard")}, plain, "the quiet orchard 4K HDR"},
		{"{MOVIE TITLE} {target}", Set{"Movie Title": Text("The Quiet Orchard")}, plain, "THE QUIET ORCHARD 4k hdr"},
		{"{Release Year} {Movie TitleYear}", heat, plain, "1995 Heat (1995)"},
		{"{Movie Title} - {Edition} - {Quality Proper} - {Target}", heat, plain, "Heat - 4K HDR"},
		{"{Movie Title} {Edition}{Quality Proper}  {Target}", heat, plain, "Heat  4K HDR"},
		{"{Movie Title} - {Edition} {Target}", Set{"Movie Title": Text("Tick, Tick... Boom!")}, plain, "Tick, Tick... Boom! - 4K HDR"},
		{"{Movie Title} {Edition} - {Target}", Set{"Movie Title": Text("Tick, Tick... Boom!")}, plain, "Tick, Tick... Boom! - 4K HDR"},
		{"{Movie Title} {Edition}", Set{"Movie Title": Text("Tick, Tick... Boom!")}, plain, "Tick, Tick... Boom!"},
	} {
		if got := folderName(t, filmTokens, c.text, work(c.s), c.f); got != c.want {
			t.Errorf("%q = %q; want %q", c.text, got, c.want)
		}
	}
}

func TestRenderEpisodeTokens(t *testing.T) {
	swat := Set{"Series Title": Text("S.W.A.T."), "Series Year": Number(2017)}
	s := episode(swat, Set{"season": Number(1)}, Set{"episode": Number(3), "Episode Title": Text("Pilot"), "Air Date": Day(works.Date{Year: 2026, Month: 11, Day: 9})})
	f := FileFacts{Target: "1080p"}
	for text, want := range map[string]string{
		"{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Target}": "S.W.A.T. - S01E03 - Pilot 1080p",
		"{Series Title} - {Air-Date} - {Episode Title}":                         "S.W.A.T. - 2026-11-09 - Pilot",
		"{Series Title} - {Air Date}":                                           "S.W.A.T. - 2026 11 09",
		"{Series.Title}.{Air.Date}":                                             "S.W.A.T..2026.11.09",
		"{Series Title} {season}x{episode:000}":                                 "S.W.A.T. 1x003",
		"{Series Title}":                                                        "S.W.A.T",
	} {
		if got := folderName(t, showTokens, text, s, f); got != want {
			t.Errorf("%q = %q; want %q", text, got, want)
		}
	}
	if got, want := fileName(t, showTokens, "{Series Title}", s, f), "S.W.A.T..mkv"; got != want {
		t.Errorf("file {Series Title} = %q; want %q", got, want)
	}
}

func TestRenderCutsAtAWholeCharacter(t *testing.T) {
	title := "A東京の物語が始まる前の長い長い夜"
	s := episode(Set{"Series Title": Text("Tokyo")}, Set{"season": Number(1)}, Set{"episode": Number(1), "Episode Title": Text(title)})
	for text, want := range map[string]string{
		"{Episode Title:30}":  "A東京の物語が始ま....mkv",
		"{Episode Title:-30}": "...まる前の長い長い夜.mkv",
		"{Episode Title:60}":  title + ".mkv",
		"{Episode Title:3}":   "A.mkv",
	} {
		got := fileName(t, showTokens, text, s, FileFacts{})
		if got != want || !utf8.ValidString(got) {
			t.Errorf("%q = %q; want %q", text, got, want)
		}
	}
}

func TestRenderColonModes(t *testing.T) {
	for _, c := range []struct {
		colon        Colon
		custom       string
		wars, rezero string
	}{
		{ColonDelete, "", "Star Wars Episode IV", "ReZero"},
		{ColonDash, "", "Star Wars- Episode IV", "Re-Zero"},
		{ColonSpaceDash, "", "Star Wars - Episode IV", "Re -Zero"},
		{ColonSpaceDashSpace, "", "Star Wars - Episode IV", "Re - Zero"},
		{ColonSmart, "", "Star Wars - Episode IV", "Re-Zero"},
		{ColonCustom, "꞉", "Star Wars꞉ Episode IV", "Re꞉Zero"},
	} {
		for title, want := range map[string]string{"Star Wars: Episode IV": c.wars, "Re:Zero": c.rezero} {
			if got := segment(rendered(t, filmTokens, style{colon: c.colon, custom: c.custom}, "{Movie Title}", work(Set{"Movie Title": Text(title)}), FileFacts{})); got != want {
				t.Errorf("colon %d on %q = %q; want %q", c.colon, title, got, want)
			}
		}
	}
}

func TestRenderCrossOSCharacters(t *testing.T) {
	got := folderName(t, filmTokens, "{Movie Title}", work(Set{"Movie Title": Text("AC/DC\\Live <at> \"the\" Pit|Stop *now*?\x07")}), FileFacts{})
	if want := "AC+DC+Live at the PitStop -now-!"; got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestSegmentRule(t *testing.T) {
	for in, want := range map[string]string{
		"  Heat (1995)": "Heat (1995)",
		"Heat. . ":      "Heat",
		"CON":           "_CON",
		"con.Game":      "_con.Game",
		"Com1 .mkv":     "_Com1 .mkv",
		"LPT9":          "_LPT9",
		"Conan":         "Conan",
		"COM0":          "COM0",
		"Heat - 4K.mkv": "Heat - 4K.mkv",
	} {
		if got := segment(in); got != want {
			t.Errorf("segment(%q) = %q; want %q", in, got, want)
		}
	}
	if got := folderName(t, filmTokens, "{Movie.Title}", work(Set{"Movie Title": Text("Con Game")}), FileFacts{}); got != "_Con.Game" {
		t.Errorf("{Movie.Title} of Con Game = %q; want _Con.Game", got)
	}
}

func TestRenderCoreTokens(t *testing.T) {
	ja := release.Language("ja")
	f := FileFacts{
		Target:  "4K HDR",
		Cut:     "Extended",
		Release: "The.Quiet.Orchard.2021.2160p.BluRay.x265-FLUX",
		Source:  "the.quiet.orchard.2021.2160p.mkv",
		Formats: []string{"IMAX", "HDR10+", "Remaster"},
		Facts: release.Facts{
			Quality:   release.Known(release.Bluray2160p),
			Revision:  release.Revision{Version: 2, Real: 1, Repack: true},
			Group:     release.Known("FLUX"),
			HDR:       release.Known([]release.HDR{"DV", "HDR10"}),
			Video:     release.Known(release.VideoCodec("x265")),
			Audio:     release.Known([]release.AudioCodec{"DD+", "AAC"}),
			Channels:  release.Known([]release.Channels{"5.1"}),
			Languages: release.Known([]release.Language{release.Original, "en", "de"}),
			Coverage:  release.Coverage{Original: release.Known(ja)},
		},
		Media: media.Record{
			Video:     media.Video{BitDepth: 10},
			Subtitles: []media.Subtitle{{Language: "en"}, {Language: "nl"}, {Language: "en"}},
		},
	}
	for text, want := range map[string]string{
		"{Quality Title}":                      "Bluray-2160p",
		"{Quality Full}":                       "Bluray-2160p Proper REAL",
		"{Revision}":                           "REAL v2 REPACK",
		"{Release Group}":                      "FLUX",
		"{Custom Formats}":                     "IMAX HDR10+ Remaster",
		"{Custom Formats:-IMAX}":               "HDR10+ Remaster",
		"{Custom Formats:Remaster,IMAX}":       "IMAX Remaster",
		"{Custom Format:HDR10+}":               "HDR10+",
		"{Custom Format:DV}":                   "",
		"{Original Title}":                     "The.Quiet.Orchard.2021.2160p.BluRay.x265-FLUX",
		"{Original Filename}":                  "the.quiet.orchard.2021.2160p",
		"{MediaInfo Simple}":                   "x265 DDP",
		"{MediaInfo Full}":                     "x265 DDP[JA+EN+DE]",
		"{MediaInfo VideoCodec}":               "x265",
		"{MediaInfo VideoBitDepth}":            "10",
		"{MediaInfo VideoDynamicRange}":        "HDR",
		"{MediaInfo VideoDynamicRangeType}":    "DV HDR10",
		"{MediaInfo 3D}":                       "",
		"{MediaInfo AudioCodec}":               "DDP",
		"{MediaInfo AudioChannels}":            "5.1",
		"{MediaInfo AudioLanguages}":           "[JA+EN+DE]",
		"{MediaInfo AudioLanguages:EN+DE}":     "[EN+DE]",
		"{MediaInfo AudioLanguages:-EN}":       "[JA+DE]",
		"{MediaInfo AudioLanguages:DE+}":       "[DE+--]",
		"{MediaInfo AudioLanguages:JA}":        "",
		"{MediaInfo AudioLanguagesAll:JA}":     "[JA]",
		"{MediaInfo SubtitleLanguages}":        "[EN+NL]",
		"{MediaInfo SubtitleLanguagesAll:-NL}": "[EN]",
		"{Edition} {Target}":                   "Extended 4K HDR",
	} {
		if got := folderName(t, filmTokens, text, work(heat), f); got != want {
			t.Errorf("%q = %q; want %q", text, got, want)
		}
	}
	for _, text := range []string{"{Quality Full}", "{Revision}", "{Release Group}", "{MediaInfo Full}", "{MediaInfo AudioLanguages}", "{MediaInfo SubtitleLanguages}", "{MediaInfo VideoBitDepth}", "{Original Filename}", "{Edition Tag}"} {
		if got := folderName(t, filmTokens, text, work(heat), FileFacts{}); got != "" {
			t.Errorf("%q with no facts = %q; want nothing", text, got)
		}
	}
	only := FileFacts{Facts: release.Facts{Languages: release.Known([]release.Language{"ja"}), Coverage: release.Coverage{Original: release.Known(ja)}}}
	if got := folderName(t, filmTokens, "{MediaInfo AudioLanguages}", work(heat), only); got != "" {
		t.Errorf("AudioLanguages of the original language alone = %q; want nothing", got)
	}
}

func TestCollapseJoinsOnlyLiterals(t *testing.T) {
	for _, c := range []struct {
		ps   []piece
		want string
	}{
		{[]piece{{text: "A"}, {text: " ", literal: true}, {empty: true}, {text: " - ", literal: true}, {text: "B"}}, "A - B"},
		{[]piece{{text: "A"}, {text: " - ", literal: true}, {empty: true}, {text: " ", literal: true}, {empty: true}, {text: " - ", literal: true}, {text: "B"}}, "A - B"},
		{[]piece{{text: "A"}, {text: "-", literal: true}, {empty: true}, {text: ".", literal: true}, {text: "B"}}, "A-B"},
		{[]piece{{text: "S.W.A.T."}, {empty: true}, {text: ". ", literal: true}, {text: "B..."}}, "S.W.A.T.. B..."},
		{[]piece{{text: "A..."}, {text: " - ", literal: true}, {empty: true}, {text: " ", literal: true}}, "A..."},
		{[]piece{{text: "A"}, {text: " (", literal: true}, {empty: true}, {text: ")", literal: true}}, "A ()"},
	} {
		if got := collapse(c.ps); got != c.want {
			t.Errorf("collapse(%+v) = %q; want %q", c.ps, got, c.want)
		}
	}
}

func TestDerivedTitleValues(t *testing.T) {
	s := work(Set{"Movie Title": Text("The Rat Pack (A&E)"), "Year": Number(1999)})
	for text, want := range map[string]string{
		"{Movie CleanTitle}":               "The Rat Pack AandE",
		"{Movie TitleThe}":                 "Rat Pack, The (A&E)",
		"{Movie CleanTitleThe}":            "Rat Pack, The AandE",
		"{Movie TitleYear}":                "The Rat Pack (A&E) (1999)",
		"{Movie CleanTitleYear}":           "The Rat Pack AandE 1999",
		"{Movie TitleWithoutYear}":         "The Rat Pack (A&E)",
		"{Movie CleanTitleWithoutYear}":    "The Rat Pack AandE",
		"{Movie TitleTheYear}":             "Rat Pack, The (A&E) (1999)",
		"{Movie CleanTitleTheYear}":        "Rat Pack, The AandE 1999",
		"{Movie TitleTheWithoutYear}":      "Rat Pack, The (A&E)",
		"{Movie CleanTitleTheWithoutYear}": "Rat Pack, The AandE",
		"{Movie TitleFirstCharacter}":      "R",
	} {
		if got := folderName(t, filmTokens, text, s, FileFacts{}); got != want {
			t.Errorf("%q = %q; want %q", text, got, want)
		}
	}
	noYear := work(Set{"Movie Title": Text("The Flash (2010)")})
	for text, want := range map[string]string{
		"{Movie TitleYear}":         "The Flash (2010)",
		"{Movie TitleWithoutYear}":  "The Flash",
		"{Movie CleanTitleTheYear}": "Flash, The 2010",
		"{Movie TitleYear} - x":     "The Flash (2010) - x",
	} {
		if got := folderName(t, filmTokens, text, noYear, FileFacts{}); got != want {
			t.Errorf("%q with no year = %q; want %q", text, got, want)
		}
	}
	if got := folderName(t, filmTokens, "{Movie TitleThe}", work(Set{}), FileFacts{}); got != "" {
		t.Errorf("a derived title of no title = %q; want nothing", got)
	}
}

func TestCatalogDerivesYearFormsOnlyWithAYear(t *testing.T) {
	c := mustCatalog(t, showTokens)
	for _, name := range []string{"Series CleanTitleTheYear", "Series TitleWithoutYear", "Episode CleanTitle", "Episode TitleThe", "Episode CleanTitleThe", "Episode TitleFirstCharacter"} {
		if c.byKey[KeyOf(name)] == nil {
			t.Errorf("no {%s}", name)
		}
	}
	for _, name := range []string{"Episode TitleYear", "Episode CleanTitleWithoutYear", "Episode TitleTheYear"} {
		if c.byKey[KeyOf(name)] != nil {
			t.Errorf("{%s} exists with no year token in its scope", name)
		}
	}
	e := c.byKey[KeyOf("Series TitleThe")]
	if e.Scope != WorkScope || e.Group != GroupWork || !strings.HasPrefix(e.About, "{Series Title} ") {
		t.Errorf("{Series TitleThe} = %+v; want the base's scope and group and an About naming it", e.Token)
	}
	if got := mustCatalog(t, filmTokens).byKey[KeyOf("Movie CleanTitleTheYear")].Example; got != "Quiet Orchard, The 2021" {
		t.Errorf("{Movie CleanTitleTheYear} example = %q; want it derived from the base's", got)
	}
}

func TestCatalogRefusesCollisions(t *testing.T) {
	for _, c := range []struct {
		name   string
		tokens []Token
		want   string
	}{
		{"two names, one key", []Token{{Name: "Movie Title", Kind: KindTitle, Scope: WorkScope}, {Name: "movie.title", Kind: KindText, Scope: WorkScope}}, `tokens "Movie Title" and "movie.title" both match as "movietitle"`},
		{"an alias on a core token", []Token{{Name: "Label", Aliases: []string{"Quality-Title"}, Kind: KindText, Scope: WorkScope}}, `"Label" and "Quality Title" both match as "qualitytitle"`},
		{"a Seq on a title", []Token{{Name: "Part", Kind: KindTitle, Scope: ItemScope, Seq: SeqInner}}, `token "Part" has a sequence role`},
		{"a declared file token", []Token{{Name: "Disc", Kind: KindNumber, Scope: FileScope}}, `token "Disc" has the file scope`},
		{"two years in a scope", []Token{{Name: "Year", Kind: KindYear, Scope: WorkScope}, {Name: "Aired", Kind: KindYear, Scope: WorkScope}}, `tokens "Year" and "Aired" are both year tokens`},
		{"a derived name declared", []Token{{Name: "Movie Title", Kind: KindTitle, Scope: WorkScope}, {Name: "Movie CleanTitle", Kind: KindText, Scope: WorkScope}}, `token "Movie CleanTitle" is a title form "Movie Title" derives`},
		{"a derived name aliased", []Token{{Name: "Movie Title", Kind: KindTitle, Scope: WorkScope}, {Name: "Sort", Aliases: []string{"Movie TitleThe"}, Kind: KindText, Scope: WorkScope}}, `token "Movie TitleThe" is a title form "Movie Title" derives`},
	} {
		_, err := newCatalog(Declaration{Tokens: c.tokens})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: newCatalog = %v; want an error with %q", c.name, err, c.want)
		}
	}
}

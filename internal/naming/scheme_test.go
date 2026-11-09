package naming

import (
	"errors"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/works"
)

// filmDecl is shaped like a type with one Item per Work, anchored on its
// folder.
var filmDecl = Declaration{
	Tokens: []Token{
		{Name: "Movie Title", Kind: KindTitle, Scope: WorkScope, Example: "The Quiet Orchard"},
		{Name: "Year", Aliases: []string{"Release Year"}, Kind: KindYear, Scope: WorkScope, Example: "2021"},
		{Name: "ImdbId", Kind: KindID, Scope: WorkScope},
		{Name: "TmdbId", Kind: KindID, Scope: WorkScope},
	},
	Folder: "{Movie Title} {(Year)}",
	Files:  []Variant{{Key: "standard", Label: "File", File: "{Movie Title} {(Year)} - {Edition Tag} {Target}", Anchor: FolderAnchor()}},
	Samples: []Sample{
		{Label: "Heat", Work: Set{"Movie Title": Text("Heat"), "Year": Number(1995), "ImdbId": Text("tt0113277"), "TmdbId": Text("949")}},
		{Label: "A colon", Work: Set{"Movie Title": Text("Star Wars: Episode IV - A New Hope"), "Year": Number(1977)}},
		{Label: "A trailing dot", Work: Set{"Movie Title": Text("Monsters, Inc."), "Year": Number(2001)}},
		{Label: "An ampersand and an apostrophe", Work: Set{"Movie Title": Text("Harold & Kumar's Escape"), "Year": Number(2008)}},
		{Label: "CJK", Work: Set{"Movie Title": Text("千と千尋の神隠し"), "Year": Number(2001)}},
		{Label: "Nothing known", Missing: true, Work: Set{"Movie Title": Text("Heat")}},
		{Label: "A trailing dot and nothing known", Missing: true, Work: Set{"Movie Title": Text("Monsters, Inc.")}},
	},
}

// showDecl is shaped like a type with seasons of episodes, three file
// variants and an aside level.
var showDecl = Declaration{
	Tokens: []Token{
		{Name: "Series Title", Kind: KindTitle, Scope: WorkScope},
		{Name: "Series Year", Kind: KindYear, Scope: WorkScope},
		{Name: "TvdbId", Kind: KindID, Scope: WorkScope},
		{Name: "ImdbId", Kind: KindID, Scope: WorkScope},
		{Name: "season", Kind: KindNumber, Scope: LevelScope, Seq: SeqOuter},
		{Name: "episode", Kind: KindNumber, Scope: ItemScope, Seq: SeqInner},
		{Name: "absolute", Kind: KindNumber, Scope: ItemScope, Seq: SeqAlone},
		{Name: "Episode Title", Kind: KindTitle, Scope: ItemScope},
		{Name: "Air Date", Kind: KindDate, Scope: ItemScope},
	},
	Folder: "{Series Title} {(Series Year)}",
	Levels: []string{"Season {season:00}"},
	Aside:  "Specials",
	Files: []Variant{
		{Key: "standard", Label: "Episode file", File: "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Edition} {Target}", Anchor: BlockAnchor()},
		{Key: "absolute", Label: "Episode file, Absolute order", File: "{Series Title} - S{season:00}E{episode:00} - {absolute:000} - {Episode Title} {Edition} {Target}", Anchor: BlockAnchor()},
		{Key: "daily", Label: "Episode file, named by air date", File: "{Series Title} - {Air-Date} - {Episode Title} {Edition} {Target}", Anchor: TokenAnchor("Air Date")},
	},
	Samples: []Sample{
		{Label: "The Wire", Work: wire, Units: []Unit{aired(1, 1, "The Target")}},
		{Label: "Two episodes", Work: wire, Units: []Unit{aired(1, 1, "Pilot (1)"), aired(1, 2, "Pilot (2)")}},
		{Label: "Across a season", Work: wire, Units: []Unit{aired(1, 13, "Sentencing"), aired(2, 1, "Ebb Tide")}},
		{Label: "A special", Work: wire, Units: []Unit{{Level: Set{"season": Number(0)}, Item: ItemValues{Set: Set{"episode": Number(1), "Episode Title": Text("Inside the Wire")}}, Aside: true}}},
		{Label: "Awkward titles", Work: Set{"Series Title": Text("Marvel's Agents of S.H.I.E.L.D."), "Series Year": Number(2013)}, Units: []Unit{aired(1, 1, "Pilot: Part 1?")}},
		{Label: "A long title", Work: wire, Units: []Unit{aired(1, 1, strings.Repeat("A very long episode title ", 12))}},
		{Label: "Nothing known", Missing: true, Work: Set{"Series Title": Text("The Wire")}, Units: []Unit{{Level: Set{"season": Number(1)}, Item: ItemValues{Set: Set{"episode": Number(1)}}}}},
	},
}

var wire = Set{"Series Title": Text("The Wire"), "Series Year": Number(2002), "TvdbId": Text("79126")}

func aired(season, episode int64, title string) Unit {
	return Unit{
		Level: Set{"season": Number(season)},
		Item: ItemValues{Set: Set{
			"episode":       Number(episode),
			"absolute":      Number(13*(season-1) + episode),
			"Episode Title": Text(title),
			"Air Date":      Day(works.Date{Year: 2002, Month: 6, Day: 2 + int(episode)}),
		}},
	}
}

func mustScheme(t *testing.T, d Declaration, s Settings) *Scheme {
	t.Helper()
	sc, err := newScheme(d, s)
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

func ptr[T any](v T) *T { return &v }

func templates(kv ...string) map[string]*string {
	m := map[string]*string{}
	for ; len(kv) >= 2; kv = kv[2:] {
		m[kv[0]] = ptr(kv[1])
	}
	return m
}

const roomy = 1000

func mustName(t *testing.T, sc *Scheme, p Placing) string {
	t.Helper()
	if p.Budget == 0 {
		p.Budget = roomy
	}
	if p.Ext == "" {
		p.Ext = ".mkv"
	}
	named, err := sc.Name(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(named.Segments, "/"); got != named.Rel {
		t.Errorf("Segments %q join to %q; Rel is %q", named.Segments, got, named.Rel)
	}
	return named.Rel
}

func TestNameAnchorsAFileOnTheFolderItLandsIn(t *testing.T) {
	sc := mustScheme(t, filmDecl, Settings{})
	heat := Subject{Work: Set{"Movie Title": Text("Heat"), "Year": Number(1995)}}
	file := FileFacts{Target: "4K HDR"}
	for _, c := range []struct {
		name    string
		landing Landing
		sub     Subject
		f       FileFacts
		want    string
	}{
		{"rendered", Rendered(), heat, file, "Heat (1995)/Heat (1995) - 4K HDR.mkv"},
		{"a cut", Rendered(), heat, FileFacts{Target: "4K HDR", Cut: "Director's Cut"}, "Heat (1995)/Heat (1995) - {edition-Director's Cut} 4K HDR.mkv"},
		{"no year", Rendered(), Subject{Work: Set{"Movie Title": Text("Heat")}}, file, "Heat/Heat - 4K HDR.mkv"},
		{"a trailing dot", Rendered(), Subject{Work: Set{"Movie Title": Text("Monsters, Inc.")}}, file, "Monsters, Inc/Monsters, Inc - 4K HDR.mkv"},
		{"a found Work folder", InWork("Heat 1995"), heat, file, "Heat 1995/Heat 1995 - 4K HDR.mkv"},
		{"a found folder", InDir("Heat 1995"), heat, file, "Heat 1995/Heat 1995 - 4K HDR.mkv"},
		{"a found level folder", InDir("Heat 1995/Extras"), heat, file, "Heat 1995/Extras/Heat 1995 - 4K HDR.mkv"},
		{"the root", InDir("."), heat, file, "Heat (1995) - 4K HDR.mkv"},
	} {
		if got := mustName(t, sc, Placing{Subject: c.sub, Landing: c.landing, File: c.f}); got != c.want {
			t.Errorf("%s: Name = %q; want %q", c.name, got, c.want)
		}
	}
}

func TestNameWithRenamingOffKeepsTheName(t *testing.T) {
	sc := mustScheme(t, filmDecl, Settings{Rename: ptr(false)})
	got := mustName(t, sc, Placing{
		Subject: Subject{Work: Set{"Movie Title": Text("Heat"), "Year": Number(1995)}},
		File:    FileFacts{Target: "4K HDR", Keep: "heat.1995.2160p.what?.REMUX"},
		Ext:     ".MKV",
	})
	if want := "Heat (1995)/heat.1995.2160p.what!.REMUX.MKV"; got != want {
		t.Errorf("Name = %q; want %q", got, want)
	}
	if sc.Reads().Formats {
		t.Error("Reads().Formats with renaming off; the file name prints no format")
	}
}

func TestNameRendersEachUnitsVariantAndFolders(t *testing.T) {
	sc := mustScheme(t, showDecl, Settings{})
	daily := func(u Unit) Unit { u.Item.Variant = "daily"; return u }
	absolute := func(u Unit) Unit { u.Item.Variant = "absolute"; return u }
	undated := aired(1, 1, "The Target")
	delete(undated.Item.Set, "Air Date")
	unnumbered := aired(1, 1, "The Target")
	delete(unnumbered.Item.Set, "absolute")
	file := FileFacts{Target: "1080p"}
	for _, c := range []struct {
		name    string
		units   []Unit
		folders bool
		want    string
	}{
		{"standard", []Unit{aired(1, 1, "The Target")}, true, "The Wire (2002)/Season 01/The Wire - S01E01 - The Target 1080p.mkv"},
		{"no level folders", []Unit{aired(1, 1, "The Target")}, false, "The Wire (2002)/The Wire - S01E01 - The Target 1080p.mkv"},
		{"daily", []Unit{daily(aired(1, 1, "The Target"))}, true, "The Wire (2002)/Season 01/The Wire - 2002-06-03 - The Target 1080p.mkv"},
		{"daily with no air date", []Unit{daily(undated)}, true, "The Wire (2002)/Season 01/The Wire - S01E01 - The Target 1080p.mkv"},
		{"absolute", []Unit{absolute(aired(1, 1, "The Target"))}, true, "The Wire (2002)/Season 01/The Wire - S01E01 - 001 - The Target 1080p.mkv"},
		{"absolute with no number", []Unit{absolute(unnumbered)}, true, "The Wire (2002)/Season 01/The Wire - S01E01 - The Target 1080p.mkv"},
		{"two episodes", []Unit{aired(1, 1, "Pilot (1)"), aired(1, 2, "Pilot (2)")}, true, "The Wire (2002)/Season 01/The Wire - S01E01-E02 - Pilot 1080p.mkv"},
		{"across a season", []Unit{aired(1, 13, "Sentencing"), aired(2, 1, "Ebb Tide")}, true, "The Wire (2002)/Season 01/The Wire - S01E13 - S02E01 - Sentencing + Ebb Tide 1080p.mkv"},
		{"a special", []Unit{{Level: Set{"season": Number(0)}, Item: ItemValues{Set: Set{"episode": Number(1)}}, Aside: true}}, true, "The Wire (2002)/Specials/The Wire - S00E01 - 1080p.mkv"},
	} {
		got := mustName(t, sc, Placing{Subject: Subject{Work: wire, Units: c.units}, LevelFolders: c.folders, File: file})
		if got != c.want {
			t.Errorf("%s: Name = %q; want %q", c.name, got, c.want)
		}
	}
}

func TestNameShrinksOnlyItemTitles(t *testing.T) {
	sc := mustScheme(t, showDecl, Settings{Templates: templates("file.standard", "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {edition-{Edition}} {Target}")})
	title := strings.Repeat("Ünïcödé ", 40)
	p := Placing{
		Subject:      Subject{Work: wire, Units: []Unit{aired(1, 1, title)}},
		LevelFolders: true,
		File:         FileFacts{Target: "4K HDR (extra 1)", Cut: "Director's Cut"},
		Ext:          ".mkv",
		Budget:       120,
	}
	got := mustName(t, sc, p)
	if len(got) > p.Budget {
		t.Errorf("Name = %q, %d bytes; want at most %d", got, len(got), p.Budget)
	}
	head := "The Wire (2002)/Season 01/The Wire - S01E01 - Ünïcödé"
	tail := "... {edition-Director's Cut} 4K HDR (extra 1).mkv"
	if !strings.HasPrefix(got, head) || !strings.HasSuffix(got, tail) {
		t.Errorf("Name = %q; want %q, the title cut, then %q", got, head, tail)
	}

	p.Subject.Units[0] = aired(1, 1, strings.Repeat("x", 300))
	p.Budget = maxSegment + 100
	got = mustName(t, sc, p)
	if file := got[strings.LastIndex(got, "/")+1:]; len(file) > maxSegment || !strings.HasSuffix(file, "xxx... {edition-Director's Cut} 4K HDR (extra 1).mkv") {
		t.Errorf("file name %q is %d bytes; want the title cut to fit %d", file, len(file), maxSegment)
	}

	p.Budget = 70
	_, err := sc.Name(p)
	var long *TooLongError
	if !errors.As(err, &long) || long.Segment != "" || long.Limit != 70 {
		t.Fatalf("Name in 70 bytes = %v; want a *TooLongError for the path", err)
	}
	if want := "the path in the root is 91 bytes long and the limit there is 70 bytes"; err.Error() != want {
		t.Errorf("Error() = %q; want %q", err, want)
	}
}

func TestReadsFormatsOnlyWhenAFileTemplatePrintsThem(t *testing.T) {
	for text, want := range map[string]bool{
		"{Movie Title} {(Year)} - {Edition Tag} {Target}":                    false,
		"{Movie Title} {(Year)} - {Edition Tag} {Target} {[Custom Formats]}": true,
		"{Movie Title} {(Year)} - {Edition Tag} {Target} {Custom Format:DV}": true,
		"{Movie Title} {(Year)} - {Edition Tag} {Target} {MediaInfo Simple}": false,
	} {
		if got := mustScheme(t, filmDecl, Settings{Templates: templates("file.standard", text)}).Reads().Formats; got != want {
			t.Errorf("Reads().Formats of %q = %v; want %v", text, got, want)
		}
	}
}

func TestFolderNamesAWorkWithNoFiles(t *testing.T) {
	sc := mustScheme(t, filmDecl, Settings{})
	if got, err := sc.Folder(Set{"Movie Title": Text("AUX"), "Year": Number(2001)}); err != nil || got != "AUX (2001)" {
		t.Errorf("Folder = %q, %v; want AUX (2001)", got, err)
	}
	if got, err := sc.Folder(Set{"Movie Title": Text("con")}); err != nil || got != "_con" {
		t.Errorf("Folder = %q, %v; want _con", got, err)
	}
	if _, err := sc.Folder(Set{}); err == nil {
		t.Error("Folder of a Work with no title answered no error")
	}
}

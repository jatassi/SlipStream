package scanning_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

func walk(t *testing.T, path string) storing.Walked {
	t.Helper()
	return file{path, 1, "", monday}.walked(t)
}

func TestReadName(t *testing.T) {
	tmdb := func(v string) scanning.ID { return scanning.ID{Provider: scanning.TMDB, Value: v} }
	imdb := func(v string) scanning.ID { return scanning.ID{Provider: scanning.IMDb, Value: v} }
	tvdb := func(v string) scanning.ID { return scanning.ID{Provider: scanning.TVDB, Value: v} }
	tests := []struct {
		name string
		want scanning.Hint
	}{
		{"Heat (1995)", scanning.Hint{Title: "Heat", Year: 1995}},
		{"Trailer Park Boys (2001)", scanning.Hint{Title: "Trailer Park Boys", Year: 2001}},
		{"Trailer Park Boys", scanning.Hint{Title: "Trailer Park Boys"}},
		{"Proof (2005)", scanning.Hint{Title: "Proof", Year: 2005}},
		{"Scenes from a Marriage (1973)", scanning.Hint{Title: "Scenes from a Marriage", Year: 1973}},
		{"1917 (2019)", scanning.Hint{Title: "1917", Year: 2019}},
		{"2001 A Space Odyssey (1968)", scanning.Hint{Title: "2001 A Space Odyssey", Year: 1968}},
		{"2001 A Space Odyssey", scanning.Hint{Title: "2001 A Space Odyssey"}},
		{"Blade Runner 2049 (2017)", scanning.Hint{Title: "Blade Runner 2049", Year: 2017}},
		{"Mr. Robot (2015)", scanning.Hint{Title: "Mr. Robot", Year: 2015}},
		{"Heat 1995", scanning.Hint{Title: "Heat", Year: 1995}},
		{"The.Matrix.1999.1080p.BluRay.x264-GROUP", scanning.Hint{Title: "The Matrix", Year: 1999}},
		{"Brazil (1985) Bluray-1080p", scanning.Hint{Title: "Brazil", Year: 1985}},
		{"The Matrix (1999) {tmdb-603}", scanning.Hint{Title: "The Matrix", Year: 1999, IDs: []scanning.ID{tmdb("603")}}},
		{"The Matrix (1999) [tmdbid-0603]", scanning.Hint{Title: "The Matrix", Year: 1999, IDs: []scanning.ID{tmdb("603")}}},
		{"The Matrix (1999) {imdb-tt0133093}", scanning.Hint{Title: "The Matrix", Year: 1999, IDs: []scanning.ID{imdb("tt0133093")}}},
		{"The Matrix (1999) [imdbid-TT0133093] {tmdbid-603}", scanning.Hint{Title: "The Matrix", Year: 1999, IDs: []scanning.ID{imdb("tt0133093"), tmdb("603")}}},
		{"Severance (2022) {tvdb-371980}", scanning.Hint{Title: "Severance", Year: 2022, IDs: []scanning.ID{tvdb("371980")}}},
		{"Severance [tvdbid=371980]", scanning.Hint{Title: "Severance", IDs: []scanning.ID{tvdb("371980")}}},
		{"Blade Runner (1982) {edition-Final Cut}", scanning.Hint{Title: "Blade Runner", Year: 1982}},
		{"Brazil {imdb-603} {tmdb-tt1}", scanning.Hint{Title: "Brazil"}},
	}
	for _, tt := range tests {
		if got := scanning.ReadName(tt.name); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ReadName(%q) = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestFold(t *testing.T) {
	same := [][2]string{
		{"Amélie", "AMÉLIE"},
		{"Amélie", "Amélie"},
		{"Mission: Impossible", "Mission Impossible"},
		{"Schindler's List", "Schindlers List"},
		{"Se7en", "se7en"},
	}
	for _, p := range same {
		if scanning.Fold(p[0]) != scanning.Fold(p[1]) {
			t.Errorf("Fold(%q) = %q, Fold(%q) = %q; want the same", p[0], scanning.Fold(p[0]), p[1], scanning.Fold(p[1]))
		}
	}
	if scanning.Fold("Amélie") == scanning.Fold("Amelie") {
		t.Error("Fold drops accents")
	}
}

func TestGroupsByWorkFolder(t *testing.T) {
	ignored := scanning.Entry{ID: storage.NewUUID(), Ignored: true}
	pending := scanning.Entry{ID: storage.NewUUID()}
	p := scanning.Plan{
		Found: []storing.Walked{
			walk(t, "Heat (1995)/Heat.mkv"), walk(t, "Brazil.1985.1080p.mkv"), walk(t, "Heat (1995)/Heat 4K.mkv"),
			walk(t, "Changed (2001)/New.mkv"),
		},
		Listed: []storing.Walked{
			walk(t, "Ignored (2000)/x.mkv"), walk(t, "Changed (2001)/Old.mkv"), walk(t, "Halloween/Halloween.mkv"),
		},
	}
	got := scanning.Groups(p, map[string]scanning.Entry{"Ignored (2000)": ignored, "Changed (2001)": ignored, "Halloween": pending})
	var folders []string
	for _, g := range got {
		folders = append(folders, g.Folder)
	}
	if want := []string{"Brazil.1985.1080p.mkv", "Changed (2001)", "Halloween", "Heat (1995)"}; !slices.Equal(folders, want) {
		t.Fatalf("groups = %v, want %v", folders, want)
	}
	if g := got[0]; !g.Loose || g.Name() != "Brazil.1985.1080p" || g.Entry != nil {
		t.Errorf("the loose file's group = %+v, named %q; want loose, named without its extension", g, g.Name())
	}
	if g := got[1]; g.Entry == nil || g.Entry.ID != ignored.ID || len(g.Found) != 1 || len(g.Listed) != 1 {
		t.Errorf("an ignored folder with a new file = %+v, want its entry, its new file and its listed one", g)
	}
	if g := got[2]; g.Entry == nil || g.Entry.ID != pending.ID || len(g.Listed) != 1 {
		t.Errorf("a pending folder = %+v, want its entry and its listed file", g)
	}
	if g := got[3]; g.Loose || g.Name() != "Heat (1995)" || len(g.Found) != 2 {
		t.Errorf("Heat's group = %+v, want its two files under its folder", g)
	}
}

func TestShelfMatch(t *testing.T) {
	home := roots.ID{UUID: storage.NewUUID()}
	other := roots.ID{UUID: storage.NewUUID()}
	w := func() works.WorkID { return works.WorkID{UUID: storage.NewUUID()} }
	heat, heat4K, halloween78, halloween18, alien, kept := w(), w(), w(), w(), w(), w()
	shelved := []scanning.Shelved{
		{Work: heat, Root: home, Label: "Heat", Claim: scanning.Claim{Year: 1995, IDs: []scanning.ID{{Provider: scanning.TMDB, Value: "949"}}}},
		{Work: heat4K, Root: other, Label: "Heat 4K Edition", Titles: []string{"Heat 4K"}, Claim: scanning.Claim{Year: 1995, IDs: []scanning.ID{{Provider: scanning.TMDB, Value: "1949"}}}},
		{Work: halloween78, Root: home, Label: "Halloween", Claim: scanning.Claim{Year: 1978}},
		{Work: halloween18, Label: "Halloween", Claim: scanning.Claim{Year: 2018}},
		{Work: alien, Label: "Alien", Titles: []string{"Alien: Director's Cut"}, Claim: scanning.Claim{Year: 1979, IDs: []scanning.ID{{Provider: scanning.IMDb, Value: "tt0078748"}}}},
		{Work: kept, Root: home, Label: "Brazil", Claim: scanning.Claim{Year: 1985}},
	}
	recorded := []scanning.Recorded{
		{Work: kept, Path: rel(t, "Brazil Folder/Brazil.mkv")},
		{Work: heat, Path: rel(t, "Shared/a.mkv")},
		{Work: alien, Path: rel(t, "Shared/b.mkv")},
		{Work: halloween78, Path: rel(t, "Halloween.1978.mkv"), Gone: true},
	}
	s := scanning.NewShelf(home, shelved, recorded)
	tests := []struct {
		name      string
		folder    string
		hint      scanning.Hint
		work      works.WorkID
		rule      scanning.Rule
		elsewhere *roots.ID
	}{
		{"a recorded file's folder wins over every other rule", "Brazil Folder", scanning.ReadName("Heat (1995)"), kept, scanning.ByFolder, nil},
		{"a gone file still names its folder", "Halloween.1978.mkv", scanning.Hint{}, halloween78, scanning.ByFolder, nil},
		{"a folder two Works share names neither", "Shared", scanning.ReadName("Shared"), works.WorkID{}, 0, nil},
		{"an id tag a Work claims", "Whatever", scanning.ReadName("Whatever {imdb-tt0078748}"), alien, scanning.ByTag, nil},
		{"an id tag wins over the title", "Heat (1995) {tmdb-1949}", scanning.ReadName("Heat (1995) {tmdb-1949}"), heat4K, scanning.ByTag, &other},
		{"title and year, compared folded", "HEAT! (1995)", scanning.ReadName("HEAT! (1995)"), heat, scanning.ByTitle, nil},
		{"any stored title", "Alien - Director's Cut (1979)", scanning.ReadName("Alien Director's Cut (1979)"), alien, scanning.ByTitle, nil},
		{"title and year pick one of two same titles", "Halloween (2018)", scanning.ReadName("Halloween (2018)"), halloween18, scanning.ByTitle, nil},
		{"no year and two Works with the title match neither", "Halloween", scanning.ReadName("Halloween"), works.WorkID{}, 0, nil},
		{"no year and one Work with the title match it", "Alien", scanning.ReadName("Alien"), alien, scanning.ByTitle, nil},
		{"a year no Work has matches none", "Heat (1996)", scanning.ReadName("Heat (1996)"), works.WorkID{}, 0, nil},
		{"a Work in another root is elsewhere", "Heat 4K (1995)", scanning.ReadName("Heat 4K (1995)"), heat4K, scanning.ByTitle, &other},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.Match(tt.folder, tt.hint)
			if got.Work != tt.work || got.Rule != tt.rule || !reflect.DeepEqual(got.Elsewhere, tt.elsewhere) {
				t.Fatalf("Match(%q, %+v) = %+v, want work %v by rule %d, elsewhere %v", tt.folder, tt.hint, got, tt.work, tt.rule, tt.elsewhere)
			}
		})
	}

	brazil2 := w()
	s.Add(scanning.Shelved{Work: brazil2, Root: home, Label: "Brazil", Claim: scanning.Claim{Year: 2023}})
	if got := s.Match("Brazil (2023)", scanning.ReadName("Brazil (2023)")); got.Work != brazil2 {
		t.Fatalf("a Work added in the unit matches %+v, want it", got)
	}
}

func TestConfident(t *testing.T) {
	cs := []scanning.Candidate{
		{Key: "11", Titles: []string{"Halloween"}, Year: 1978},
		{Key: "22", Titles: []string{"Halloween"}, Year: 2018},
		{Key: "33", Titles: []string{"La Haine", "Hate"}, Year: 1995},
		{Key: "44", Titles: []string{"Heat"}, Year: 1995},
		{Key: "55", Titles: []string{"Heat"}, Year: 1995},
	}
	tests := []struct {
		name      string
		hint      scanning.Hint
		cs        []scanning.Candidate
		best      int
		confident bool
	}{
		{"a key the tag names", scanning.Hint{Title: "Anything", Key: "22"}, cs, 1, true},
		{"one title and year", scanning.Hint{Title: "Halloween", Year: 2018}, cs, 1, true},
		{"an original title counts", scanning.Hint{Title: "hate", Year: 1995}, cs, 2, true},
		{"two candidates with the title and year", scanning.Hint{Title: "Heat", Year: 1995}, cs, 0, false},
		{"no year is never confident", scanning.Hint{Title: "Halloween"}, cs[:1], 0, false},
		{"a year no candidate has", scanning.Hint{Title: "Halloween", Year: 2007}, cs, 0, false},
		{"a key no candidate has falls to title and year", scanning.Hint{Title: "Halloween", Year: 1978, Key: "99"}, cs, 0, true},
		{"no candidates", scanning.Hint{Title: "Brazil", Year: 1985}, nil, -1, false},
	}
	for _, tt := range tests {
		best, confident := scanning.Confident(tt.hint, tt.cs)
		if best != tt.best || confident != tt.confident {
			t.Errorf("%s: Confident = %d, %t; want %d, %t", tt.name, best, confident, tt.best, tt.confident)
		}
	}
}

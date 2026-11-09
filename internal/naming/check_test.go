package naming

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// checkBudget is a Windows root's 259 bytes less "D:\Media\Films\".
const checkBudget = 244

// problemsOf is what a save of s over d meets: templates that don't
// parse or bind, then the check.
func problemsOf(t *testing.T, d Declaration, s Settings) []Problem {
	t.Helper()
	sc, err := newScheme(d, s)
	var refused *RefusedError
	if errors.As(err, &refused) {
		return refused.Problems
	}
	if err != nil {
		t.Fatal(err)
	}
	return check(sc, checkBudget)
}

func TestTheDefaultsPassTheCheck(t *testing.T) {
	for _, d := range []Declaration{filmDecl, showDecl} {
		if got := problemsOf(t, d, Settings{}); got != nil {
			t.Errorf("the defaults of %s fail: %v", d.Folder, got)
		}
		if got := Conformance(d); got != nil {
			t.Errorf("Conformance of %s = %q", d.Folder, got)
		}
	}
}

const groupsWork = "because Jellyfin and Emby show a Work's files as one item only when every name does."

func TestTheCheckRefuses(t *testing.T) {
	for _, c := range []struct {
		name string
		d    Declaration
		s    Settings
		want []Problem
	}{
		{"no {Target}", filmDecl, Settings{Templates: templates("file.standard", "{Movie Title} {(Year)} - {Edition Tag}")},
			[]Problem{{"file.standard", "The file name has no {Target}. Add {Target} so each file of a Work gets its own name."}}},
		{"no Edition token", filmDecl, Settings{Templates: templates("file.standard", "{Movie Title} {(Year)} - {Target}")},
			[]Problem{{"file.standard", "The file name has no Edition token. Add {Edition Tag} or {Edition} so each cut gets its own name."}}},
		{"a tag before the dash", filmDecl, Settings{Templates: templates("file.standard", "{Movie Title} {(Year)} {edition-{Edition}} - {Target}")},
			[]Problem{{"file.standard", "{edition-{Edition}} comes before ' - '. Move it after ' - ', because it differs between files."}}},
		{"{Revision} before the dash", filmDecl, Settings{Templates: templates("file.standard", "{Movie Title} {(Year)} {Revision} - {Edition Tag} {Target}")},
			[]Problem{{"file.standard", "{Revision} comes before ' - '. Move it after ' - ', because it differs between files."}}},
		{"{Movie CleanTitle} under a {Movie Title} folder", filmDecl, Settings{Templates: templates("file.standard", "{Movie CleanTitle} {(Year)} - {Edition Tag} {Target}")},
			[]Problem{{"file.standard", "The file name starts with {Movie CleanTitle}. Start it with the folder name and ' - ', " + groupsWork}}},
		{"a year without brackets", filmDecl, Settings{Templates: templates("file.standard", "{Movie Title} {Year} - {Edition Tag} {Target}")},
			[]Problem{{"file.standard", "The file name starts with {Movie Title} {Year}. Start it with the folder name and ' - ', " + groupsWork}}},
		{"no dash after the folder", filmDecl, Settings{Templates: templates("file.standard", "{Movie Title} {(Year)} {Edition Tag} {Target}")},
			[]Problem{{"file.standard", "{Edition Tag} comes before ' - '. Move it after ' - ', because it differs between files."}}},
		{"an id between the folder and the dash", filmDecl, Settings{Templates: templates("file.standard", "{Movie Title} {(Year)} {imdb-{ImdbId}} - {Edition Tag} {Target}")},
			[]Problem{{"file.standard", "' - ' doesn't come right after the folder name. Put it there, " + groupsWork}}},
		{"{Quality Full} in a folder", filmDecl, Settings{Templates: templates("folder", "{Movie Title} {Quality Full}")},
			[]Problem{{"folder", "Folder names can't use {Quality Full}, because it changes with each upgrade. Remove it from the folder template."}}},
		{"a literal colon", filmDecl, Settings{Templates: templates("file.standard", "{Movie Title} {(Year)} - {Edition Tag}: {Target}")},
			[]Problem{{"file.standard", "The file name can't hold ':'. Remove it or use a token."}}},
		{"renaming on over a stored template without {Target}", filmDecl, Settings{Rename: ptr(true), Templates: templates("file.standard", "{Movie Title} {(Year)} - {Edition Tag}")},
			[]Problem{{"file.standard", "The file name has no {Target}. Add {Target} so each file of a Work gets its own name."}}},
		{"a folder empty for a sample", filmDecl, Settings{Templates: templates("folder", "{(Year)}", "file.standard", "{(Year)} - {Edition Tag} {Target}")},
			[]Problem{{"folder", `The folder name is empty for the sample "Nothing known". Use a token every Work has a value for, such as its title.`}}},
		{"an unknown token", filmDecl, Settings{Templates: templates("file.standard", "{Movie Titel} - {Target}")},
			[]Problem{{"file.standard", "{Movie Titel} isn't a token. Use a token from the token list, such as {Target}."}}},
		{"an unclosed brace", filmDecl, Settings{Templates: templates("folder", "{Movie Title")},
			[]Problem{{"folder", `{Movie Title has no closing brace. Add "}" to close it.`}}},
		{"a custom colon text with a colon", filmDecl, Settings{Colon: ptr(ColonCustom), ColonCustom: ptr(" : ")},
			[]Problem{{"colonCustom", "The colon text can't contain ':' because a name on Windows can't hold it. Remove it."}}},
		{"a custom colon text with a slash", filmDecl, Settings{Colon: ptr(ColonCustom), ColonCustom: ptr("/")},
			[]Problem{{"colonCustom", "The colon text can't contain '/' because a name on Windows can't hold it. Remove it."}}},
		{"an episode title before the dash", showDecl, Settings{Templates: templates("file.standard", "{Series Title} - S{season:00}E{episode:00} {Episode Title} - {Edition} {Target}")},
			[]Problem{{"file.standard", "{Episode Title} comes before ' - '. Move it after ' - ', because it differs between files."}}},
		{"a quality before the block", showDecl, Settings{Templates: templates("file.standard", "{Series Title} {Quality Title} - S{season:00}E{episode:00} - {Edition} {Target}")},
			[]Problem{{"file.standard", "{Quality Title} comes before ' - '. Move it after ' - ', because it differs between files."}}},
		{"no episode block", showDecl, Settings{Templates: templates("file.standard", "{Series Title} - {Episode Title} {Edition} {Target}")},
			[]Problem{{"file.standard", "The file name has no {season} and {episode}. Add them and then ' - ', because Plex, Jellyfin and Emby show an Item's files as one item only when every name does."}}},
		{"no air date", showDecl, Settings{Templates: templates("file.daily", "{Series Title} - {Episode Title} {Edition} {Target}")},
			[]Problem{{"file.daily", "The file name has no {Air Date}. Add it and then ' - ', because Plex, Jellyfin and Emby show an Item's files as one item only when every name does."}}},
		{"no dash after the air date", showDecl, Settings{Templates: templates("file.daily", "{Series Title} - {Air-Date}.{TvdbId} - {Edition} {Target}")},
			[]Problem{{"file.daily", "' - ' doesn't come right after {Air-Date}. Put it there, because Plex, Jellyfin and Emby show an Item's files as one item only when every name does."}}},
		{"an episode title in a level folder", showDecl, Settings{Templates: templates("levels.1", "Season {season} {Episode Title}")},
			[]Problem{{"levels.1", "{Episode Title} can't go in a level folder name because it can differ between the files in one level folder. Use it in the file template."}}},
	} {
		got := problemsOf(t, c.d, c.s)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: problems = %q; want %q", c.name, got, c.want)
		}
		for _, p := range got {
			if p.Error() != p.Sentence {
				t.Errorf("%s: Error() = %q; want the sentence %q", c.name, p.Error(), p.Sentence)
			}
		}
	}
}

func TestWithRenamingOffTheCheckJudgesOnlyParsingScopeAndCharacters(t *testing.T) {
	off := Settings{Rename: ptr(false), Templates: templates("file.standard", "{Revision} {Movie CleanTitle}")}
	if got := problemsOf(t, filmDecl, off); got != nil {
		t.Errorf("problems with renaming off = %q; want none", got)
	}
	off.Templates = templates("file.standard", "{Movie Title}?")
	want := []Problem{{"file.standard", `A template can't contain "?" because a name on Windows can't hold it. Remove the "?".`}}
	if got := problemsOf(t, filmDecl, off); !reflect.DeepEqual(got, want) {
		t.Errorf("problems with renaming off = %q; want %q", got, want)
	}
}

func TestTheCheckRefusesASampleThatDoesntFit(t *testing.T) {
	file := "{Series Title} - S{season:00}E{episode:00} - {Edition} {Target}" + strings.Repeat(" {Series TitleYear}", 13)
	got := problemsOf(t, showDecl, Settings{Templates: templates("file.standard", file)})
	want := []Problem{{"file.standard", `For the sample "The Wire" with a Standard file, the path in the root is 263 bytes long and the limit there is 244 bytes. Shorten the template.`}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("problems = %q; want %q", got, want)
	}
}

func TestConformanceNamesEachBrokenRule(t *testing.T) {
	broken := filmDecl
	broken.Files = []Variant{{Key: "standard", Label: "File", File: "{Movie Title} {(Year)} - {Edition Tag}", Anchor: FolderAnchor()}, {Label: "Other", File: "{Target}", Anchor: FolderAnchor()}}
	broken.Samples = []Sample{
		{Label: "Heat", Work: Set{"Movie Title": Number(1), "Episode": Text("x")}},
		{Label: "Picks", Work: Set{"Movie Title": Text("Heat")}, Units: []Unit{{Item: ItemValues{Variant: "daily"}}}},
	}
	want := []string{
		`file variant "Other" needs a key of its own`,
		`sample "Heat" has a value for "Episode", which isn't a token of its tier`,
		`sample "Heat" has a value for "Movie Title" of a kind the token doesn't hold`,
		`sample "Picks" picks file variant "daily", which the type doesn't declare`,
		"the default file.standard: The file name has no {Target}. Add {Target} so each file of a Work gets its own name.",
	}
	got := Conformance(broken)
	if len(got) < len(want) || !reflect.DeepEqual(got[:len(want)], want) {
		t.Errorf("Conformance = %q; want it to start %q", got, want)
	}

	clash := filmDecl
	clash.Tokens = append(append([]Token{}, filmDecl.Tokens...), Token{Name: "Movie.Title", Kind: KindText, Scope: WorkScope})
	if got := Conformance(clash); len(got) != 1 {
		t.Errorf("Conformance with two names of one key = %q; want one sentence", got)
	}
}

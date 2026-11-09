package naming

import (
	"errors"
	"reflect"
	"testing"
)

func TestBindRefuses(t *testing.T) {
	for _, c := range []struct {
		text  string
		field field
		pos   int
		token string
		want  string
	}{
		{"{Movie Title} {Nope}", fileField, 14, "{Nope}", "{Nope} isn't a token. Use a token from the token list, such as {Target}."},
		{"{Movie Title} {Target}", folderField, 14, "{Target}", "{Target} can't go in a Work folder name because it can differ between the Work's files. Use it in the file template."},
		{"Season {season:00}", folderField, 7, "{season:00}", "{season:00} can't go in a Work folder name because it can differ between the Work's files. Use it in the file template."},
		{"Season {season:00} {Episode Title}", levelField, 19, "{Episode Title}", "{Episode Title} can't go in a level folder name because it can differ between the files in one level folder. Use it in the file template."},
		{"{Movie Title:00}", fileField, 0, "{Movie Title:00}", `{Movie Title:00} pads a value that isn't a number. Remove the ":00". Only a number can be padded.`},
		{"{Year:000}", fileField, 0, "{Year:000}", `{Year:000} pads a value that isn't a number. Remove the ":000". Only a number can be padded.`},
		{"{season:30}", fileField, 0, "{season:30}", `{season:30} cuts a value that isn't text. Remove the ":30". Only a title or text can be cut.`},
		{"{Target:-12}", fileField, 0, "{Target:-12}", `{Target:-12} cuts a value that isn't text. Remove the ":-12". Only a title or text can be cut.`},
		{"{Movie Title:IMAX}", fileField, 0, "{Movie Title:IMAX}", `{Movie Title:IMAX} has a modifier this token doesn't take. Remove the ":IMAX".`},
		{"{MediaInfo AudioLanguages:E1}", fileField, 0, "{MediaInfo AudioLanguages:E1}", `{MediaInfo AudioLanguages:E1} has "E1", which isn't a list of language codes. Join language codes with "+", as in :EN+DE.`},
		{"{Custom Format:A,B}", fileField, 0, "{Custom Format:A,B}", "{Custom Format:A,B} doesn't name exactly one format. Name one format, as in {Custom Format:IMAX}."},
		{"{Custom Format:-A}", fileField, 0, "{Custom Format:-A}", "{Custom Format:-A} doesn't name exactly one format. Name one format, as in {Custom Format:IMAX}."},
		{"{Custom Formats:A,,B}", fileField, 0, "{Custom Formats:A,,B}", "{Custom Formats:A,,B} has an empty format name. Name a format between each pair of commas."},
		{"{edition-{Edition Tag}}", fileField, 0, "{edition-{Edition Tag}}", "{edition-{Edition Tag}} puts a tag inside a tag. Remove the outer tag."},
		{"Heat: {Target}", fileField, 4, "", `A template can't contain ":" outside a token because a name on Windows can't hold it. Remove the ":". The colon setting decides how a colon inside a title prints.`},
		{"{Movie Title} Why? {Target}", fileField, 17, "", `A template can't contain "?" because a name on Windows can't hold it. Remove the "?".`},
		{"{Movie Title} <{Target}>", fileField, 14, "", `A template can't contain "<" because a name on Windows can't hold it. Remove the "<".`},
		{"{Movie Title} *", fileField, 14, "", `A template can't contain "*" because a name on Windows can't hold it. Remove the "*".`},
		{"Ünï \"{Target}\"", fileField, 6, "", `A template can't contain '"' because a name on Windows can't hold it. Remove the '"'.`},
		{"{Movie Title}\t{Target}", fileField, 13, "", "A template can't contain the control character U+0009. Remove it."},
	} {
		tmpl, err := Parse(c.text)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.text, err)
		}
		_, err = bind(tmpl, mustCatalog(t, mixedTokens), c.field)
		var pe *ParseError
		if !errors.As(err, &pe) {
			t.Errorf("bind(%q) = %v; want a *ParseError", c.text, err)
			continue
		}
		if pe.Pos != c.pos || pe.Token != c.token || pe.Sentence != c.want {
			t.Errorf("bind(%q) = %d %q %q; want %d %q %q", c.text, pe.Pos, pe.Token, pe.Sentence, c.pos, c.token, c.want)
		}
	}
}

// mixedTokens has a token of every scope.
var mixedTokens = Declaration{Tokens: append(append([]Token{}, filmTokens.Tokens[:2]...), showTokens.Tokens[2:]...)}

func TestBindTakesEachFieldsScopes(t *testing.T) {
	for _, c := range []struct {
		text  string
		field field
	}{
		{"{Movie Title} {(Year)} {Movie CleanTitleTheYear}", folderField},
		{"Season {season:00} of {Movie TitleThe}", levelField},
		{"{Movie Title} - S{season:00}E{episode:00} - {Episode Title:30} {Air-Date} {Custom Formats:-A} {MediaInfo AudioLanguages:EN+DE+} {Custom Format:IMAX} {edition-{Edition}} {Target}", fileField},
	} {
		tmpl, err := Parse(c.text)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := bind(tmpl, mustCatalog(t, mixedTokens), c.field); err != nil {
			t.Errorf("bind(%q, %d): %v", c.text, c.field, err)
		}
	}
}

func TestBindMarksItemTitlesShrinkable(t *testing.T) {
	tmpl, err := Parse("{Movie Title} - S{season:00}E{episode:00} - {Episode Title} {Episode CleanTitleThe} {Edition} {Target}")
	if err != nil {
		t.Fatal(err)
	}
	b, err := bind(tmpl, mustCatalog(t, mixedTokens), fileField)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{6, 8}; !reflect.DeepEqual(b.shrinkable, want) {
		t.Errorf("shrinkable = %v; want %v", b.shrinkable, want)
	}
}

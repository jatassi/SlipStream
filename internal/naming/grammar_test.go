package naming

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func tok(text string, at int, name string, opts ...func(*ref)) *ref {
	r := &ref{text: text, at: at, name: name, key: KeyOf(name), casing: casingOf(name)}
	for _, o := range opts {
		o(r)
	}
	return r
}

func pre(s string) func(*ref)   { return func(r *ref) { r.prefix = s } }
func suf(s string) func(*ref)   { return func(r *ref) { r.suffix = s } }
func sep(c byte) func(*ref)     { return func(r *ref) { r.sep = c } }
func modOn(m mod) func(*ref)    { return func(r *ref) { r.mod = m } }
func tagOf(t tag) func(*ref)    { return func(r *ref) { r.tag = t } }
func cased(c casing) func(*ref) { return func(r *ref) { r.casing = c } }

func TestParse(t *testing.T) {
	for _, c := range []struct {
		text string
		want []part
	}{
		{"", nil},
		{"Season 1", []part{literal("Season 1")}},
		{"{Target}", []part{tok("{Target}", 0, "Target")}},
		{"{Movie Title} {(Year)}", []part{tok("{Movie Title}", 0, "Movie Title"), literal(" "), tok("{(Year)}", 14, "Year", pre("("), suf(")"))}},
		{"{[Quality Full]}", []part{tok("{[Quality Full]}", 0, "Quality Full", pre("["), suf("]"))}},
		{"{-Release Group}", []part{tok("{-Release Group}", 0, "Release Group", pre("-"))}},
		{"{ Mediainfo AudioChannels]}", []part{tok("{ Mediainfo AudioChannels]}", 0, "Mediainfo AudioChannels", pre(" "), suf("]"))}},
		{"{Mediainfo VideoCodec }", []part{tok("{Mediainfo VideoCodec }", 0, "Mediainfo VideoCodec", suf(" "))}},
		{"{Target -}", []part{tok("{Target -}", 0, "Target", suf(" -"))}},
		{"{{Edition}}", []part{tok("{{Edition}}", 0, "Edition", pre("{"), suf("}"))}},
		{"{{{Edition}}}", []part{tok("{{{Edition}}}", 0, "Edition", pre("{{"), suf("}}"))}},
		{"{({Edition})}", []part{tok("{({Edition})}", 0, "Edition", pre("({"), suf("})"))}},
		{"{edition-{Edition}}", []part{tok("{edition-{Edition}}", 0, "Edition", tagOf(editionTag))}},
		{"{imdb-{ImdbId}}", []part{tok("{imdb-{ImdbId}}", 0, "ImdbId", tagOf(imdbTag))}},
		{"{imdbid-{ImdbId}}", []part{tok("{imdbid-{ImdbId}}", 0, "ImdbId", tagOf(imdbidTag))}},
		{"{tmdb-{TmdbId}}", []part{tok("{tmdb-{TmdbId}}", 0, "TmdbId", tagOf(tmdbTag))}},
		{"{tmdbid-{TmdbId}}", []part{tok("{tmdbid-{TmdbId}}", 0, "TmdbId", tagOf(tmdbidTag))}},
		{"{tvdb-{TvdbId}}", []part{tok("{tvdb-{TvdbId}}", 0, "TvdbId", tagOf(tvdbTag))}},
		{"{tvdbid-{TvdbId}}", []part{tok("{tvdbid-{TvdbId}}", 0, "TvdbId", tagOf(tvdbidTag))}},
		{"x {imdb-{(ImdbId)}} y", []part{literal("x "), tok("{imdb-{(ImdbId)}}", 2, "ImdbId", tagOf(imdbTag), pre("("), suf(")")), literal(" y")}},
		{"[imdbid-{ImdbId}]", []part{literal("[imdbid-"), tok("{ImdbId}", 8, "ImdbId"), literal("]")}},
		{"{Movie.Title}", []part{tok("{Movie.Title}", 0, "Movie.Title", sep('.'))}},
		{"{Air-Date}", []part{tok("{Air-Date}", 0, "Air-Date", sep('-'))}},
		{"{Movie_Title}", []part{tok("{Movie_Title}", 0, "Movie_Title", sep('_'))}},
		{"{.Release.Year}", []part{tok("{.Release.Year}", 0, "Release.Year", pre("."), sep('.'))}},
		{"{MediaInfo VideoDynamic.Range}", []part{tok("{MediaInfo VideoDynamic.Range}", 0, "MediaInfo VideoDynamic.Range", sep('.'))}},
		{"{movie title}", []part{tok("{movie title}", 0, "movie title", cased(lower))}},
		{"{MOVIE TITLE}", []part{tok("{MOVIE TITLE}", 0, "MOVIE TITLE", cased(upper))}},
		{"{sErIES-tItLE}", []part{tok("{sErIES-tItLE}", 0, "sErIES-tItLE", sep('-'), cased(asIs))}},
		{"{season:00}", []part{tok("{season:00}", 0, "season", modOn(padding(2)))}},
		{"{absolute:000}", []part{tok("{absolute:000}", 0, "absolute", modOn(padding(3)))}},
		{"{Episode Title:30}", []part{tok("{Episode Title:30}", 0, "Episode Title", modOn(cut{bytes: 30}))}},
		{"{Episode Title:-30}", []part{tok("{Episode Title:-30}", 0, "Episode Title", modOn(cut{bytes: 30, fromEnd: true}))}},
		{"{Episode Title:007}", []part{tok("{Episode Title:007}", 0, "Episode Title", modOn(cut{bytes: 7}))}},
		{"{[Episode Title:30]}", []part{tok("{[Episode Title:30]}", 0, "Episode Title", pre("["), suf("]"), modOn(cut{bytes: 30}))}},
		{"{MediaInfo AudioLanguages:EN+DE}", []part{tok("{MediaInfo AudioLanguages:EN+DE}", 0, "MediaInfo AudioLanguages", modOn(words{text: "EN+DE"}))}},
		{"{MediaInfo AudioLanguages:-EN}", []part{tok("{MediaInfo AudioLanguages:-EN}", 0, "MediaInfo AudioLanguages", modOn(words{text: "EN", drop: true}))}},
		{"{MediaInfo AudioLanguages:DE+}", []part{tok("{MediaInfo AudioLanguages:DE+}", 0, "MediaInfo AudioLanguages", modOn(words{text: "DE+"}))}},
		{"{MediaInfo SubtitleLanguages:DE+NO.}", []part{tok("{MediaInfo SubtitleLanguages:DE+NO.}", 0, "MediaInfo SubtitleLanguages", modOn(words{text: "DE+NO"}), suf("."))}},
		{"{MediaInfo SubtitleLanguages:-EN-}End", []part{tok("{MediaInfo SubtitleLanguages:-EN-}", 0, "MediaInfo SubtitleLanguages", modOn(words{text: "EN", drop: true}), suf("-")), literal("End")}},
		{"{Custom Formats:A,B}", []part{tok("{Custom Formats:A,B}", 0, "Custom Formats", modOn(words{text: "A,B"}))}},
		{"{Custom Formats:-INTERNAL,NAME WITH SPACES}", []part{tok("{Custom Formats:-INTERNAL,NAME WITH SPACES}", 0, "Custom Formats", modOn(words{text: "INTERNAL,NAME WITH SPACES", drop: true}))}},
		{"{Custom Format:IMAX }", []part{tok("{Custom Format:IMAX }", 0, "Custom Format", modOn(words{text: "IMAX"}), suf(" "))}},
		{"S{season:00}E{episode:00}", []part{literal("S"), tok("{season:00}", 1, "season", modOn(padding(2))), literal("E"), tok("{episode:00}", 13, "episode", modOn(padding(2)))}},
		{"[{Mediainfo VideoCodec }{MediaInfo VideoBitDepth}bit]", []part{literal("["), tok("{Mediainfo VideoCodec }", 1, "Mediainfo VideoCodec", suf(" ")), tok("{MediaInfo VideoBitDepth}", 24, "MediaInfo VideoBitDepth"), literal("bit]")}},
		{"Heat: 1995 * ?", []part{literal("Heat: 1995 * ?")}},
	} {
		got, err := Parse(c.text)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.text, err)
			continue
		}
		if got.String() != c.text || !reflect.DeepEqual(got.parts, c.want) {
			t.Errorf("Parse(%q) = %s; want %s", c.text, show(got.parts), show(c.want))
		}
	}
}

func show(ps []part) string {
	var b strings.Builder
	for _, p := range ps {
		if r, ok := p.(*ref); ok {
			fmt.Fprintf(&b, "%+v ", *r)
			continue
		}
		fmt.Fprintf(&b, "%q ", p)
	}
	return b.String()
}

func TestParseRefuses(t *testing.T) {
	for _, c := range []struct {
		text, token string
		pos         int
		sentence    string
	}{
		{"{Target", "{Target", 0, `{Target has no closing brace. Add "}" to close it.`},
		{"x {Movie Title - {Target}", "{Movie Title -", 2, `{Movie Title - has no closing brace. Add "}" to close it.`},
		{"{{Edition}", "{{Edition}", 0, `{{Edition} has no closing brace. Add "}" to close it.`},
		{"{Edition-{Edition}}", "{Edition-", 0, `{Edition- has no closing brace. Add "}" to close it.`},
		{"Heat }", "", 5, `The "}" closes no token. Remove it, or make it a token's suffix, as in {{Edition}}.`},
		{"{Target}}", "", 8, `The "}" closes no token. Remove it, or make it a token's suffix, as in {{Edition}}.`},
		{"{}", "{}", 0, "{} names no token. Write a token name between the braces, such as {Target}."},
		{"{ }", "{ }", 0, "{ } names no token. Write a token name between the braces, such as {Target}."},
		{"{*Target}", "{*Target}", 1, `{*Target} has "*" where its name starts. Start the name with a letter or digit. Only a space, "-", ".", "_", "[", "(" or "{" can come before it.`},
		{"{Target|FR}", "{Target|FR}", 7, `{Target|FR} can't contain "|" there. After the name, a token can have a ":" modifier and then only a space, "-", ".", "_", ")", "]" or "}".`},
		{"{Target:30.5}", "{Target:30.5}", 11, `{Target:30.5} can't contain "5" there. After the name, a token can have a ":" modifier and then only a space, "-", ".", "_", ")", "]" or "}".`},
		{"{Movie..Title}", "{Movie..Title}", 6, `{Movie..Title} has ".." between two words of its name. Separate the words with one space, ".", "-" or "_".`},
		{"{Movie.Clean-Title}", "{Movie.Clean-Title}", 12, `{Movie.Clean-Title} separates the words of its name with both "." and "-". Use one of them.`},
		{"{Target:}", "{Target:}", 7, `{Target:} has nothing after ":". Add a modifier, such as :00 or :30, or remove the ":".`},
		{"{Custom Formats:-}", "{Custom Formats:-}", 15, `{Custom Formats:-} has nothing after ":". Add a modifier, such as :00 or :30, or remove the ":".`},
		{"{Episode Title:-0}", "{Episode Title:-0}", 15, "{Episode Title:-0} cuts the value to 0 bytes. Use a number from 1 to 255."},
		{"{Episode Title:256}", "{Episode Title:256}", 15, "{Episode Title:256} cuts to more than 255 bytes. Use a number from 1 to 255, the most a name can hold."},
		{"{season:" + strings.Repeat("0", 256) + "}", "{season:" + strings.Repeat("0", 256) + "}", 8, "{season:" + strings.Repeat("0", 256) + "} pads to more than 255 digits. Use 255 zeros or fewer."},
		{"{edition-{imdb-{ImdbId}}}", "{edition-{imdb-{ImdbId}", 0, "{edition-{imdb-{ImdbId} has a tag inside a tag. A tag holds one token, as in {edition-{Edition}}."},
		{"{edition-{Edition}", "{edition-{Edition}", 0, `{edition-{Edition} has no closing brace. Add "}" to close it.`},
		{"{edition-{Edition}x}", "{edition-{Edition}x}", 18, `{edition-{Edition}x} has "x" after its inner token. Close the tag with "}" right after it, as in {edition-{Edition}}.`},
		{"Season/{season}", "", 6, `A template can't contain "/" because it names one folder or file. Remove the "/".`},
		{`{Series Title}\x`, "", 14, `A template can't contain "\" because it names one folder or file. Remove the "\".`},
		{"{Café}", "{Café}", 4, `{Café} can't contain "é" there. After the name, a token can have a ":" modifier and then only a space, "-", ".", "_", ")", "]" or "}".`},
	} {
		_, err := Parse(c.text)
		var pe *ParseError
		if !errors.As(err, &pe) {
			t.Errorf("Parse(%q) = %v; want a *ParseError", c.text, err)
			continue
		}
		if pe.Pos != c.pos || pe.Token != c.token || pe.Sentence != c.sentence {
			t.Errorf("Parse(%q) = {%d %q %q}; want {%d %q %q}", c.text, pe.Pos, pe.Token, pe.Sentence, c.pos, c.token, c.sentence)
		}
	}
}

func TestParseErrorReadsAsItsSentence(t *testing.T) {
	_, err := Parse("Heat {Target")
	if got, want := err.Error(), `{Target has no closing brace. Add "}" to close it.`; got != want {
		t.Errorf("Error() = %q; want %q", got, want)
	}
}

func TestKeyOf(t *testing.T) {
	for name, want := range map[string]Key{
		"Movie CleanTitle":     "moviecleantitle",
		"Air-Date":             "airdate",
		".Release.Year":        "releaseyear",
		"MEDIA_info 3D":        "mediainfo3d",
		"Quality Full":         "qualityfull",
		"quality_full":         "qualityfull",
		"Custom Formats":       "customformats",
		"Mediainfo AudioCodec": "mediainfoaudiocodec",
	} {
		if got := KeyOf(name); got != want {
			t.Errorf("KeyOf(%q) = %q; want %q", name, got, want)
		}
	}
}

func TestWordsReadByKind(t *testing.T) {
	for _, c := range []struct {
		w    words
		want langs
		ok   bool
	}{
		{words{text: "EN+DE"}, langs{codes: []string{"EN", "DE"}}, true},
		{words{text: "EN", drop: true}, langs{codes: []string{"EN"}, drop: true}, true},
		{words{text: "DE+"}, langs{codes: []string{"DE"}, mark: true}, true},
		{words{text: "EN+NL", drop: true}, langs{codes: []string{"EN", "NL"}, drop: true}, true},
		{words{text: "EN++DE"}, langs{}, false},
		{words{text: "EN DE"}, langs{}, false},
		{words{text: "E1"}, langs{}, false},
		{words{text: "A,B"}, langs{}, false},
	} {
		if got, ok := c.w.langs(); ok != c.ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%+v.langs() = %+v, %t; want %+v, %t", c.w, got, ok, c.want, c.ok)
		}
	}
	for _, c := range []struct {
		w    words
		want formatFilter
		ok   bool
	}{
		{words{text: "A,B"}, formatFilter{names: []string{"A", "B"}}, true},
		{words{text: "INTERNAL,NAME WITH SPACES", drop: true}, formatFilter{names: []string{"INTERNAL", "NAME WITH SPACES"}, drop: true}, true},
		{words{text: "DV HDR10+"}, formatFilter{names: []string{"DV HDR10+"}}, true},
		{words{text: "A,,B"}, formatFilter{}, false},
	} {
		if got, ok := c.w.formats(); ok != c.ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%+v.formats() = %+v, %t; want %+v, %t", c.w, got, ok, c.want, c.ok)
		}
	}
	for _, c := range []struct {
		w    words
		want oneFormat
		ok   bool
	}{
		{words{text: "IMAX"}, "IMAX", true},
		{words{text: "NAME WITH SPACES"}, "NAME WITH SPACES", true},
		{words{text: "IMAX", drop: true}, "", false},
		{words{text: "A,B"}, "", false},
	} {
		if got, ok := c.w.one(); ok != c.ok || got != c.want {
			t.Errorf("%+v.one() = %q, %t; want %q, %t", c.w, got, ok, c.want, c.ok)
		}
	}
}

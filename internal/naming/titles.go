package naming

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// The title variants are Sonarr's FileNameBuilder title helpers
// (src/NzbDrone.Core/Organizer/FileNameBuilder.cs at the pinned commit),
// which Radarr shares.

// titlePrefix is Sonarr's TitlePrefixRegex: a leading article, the rest,
// and any trailing parenthesized groups the article goes before.
var titlePrefix = regexp.MustCompile(`(?i)^(The|An|A) (.*?)((?: *\([^)]+\))*)$`)

// yearEnd is Sonarr's YearRegex: a title ending with its year.
var yearEnd = regexp.MustCompile(`\(\d{4}\)$`)

// variant is one title form derived from a "<X> Title" token. year is 0
// when the Work has no year.
type variant struct {
	suffix string
	// year is true for a form only a scope with a year token has.
	year   bool
	kind   Kind
	about  string
	derive func(title string, year int64) string
}

// variants are Sonarr's set: {Title, CleanTitle} x {"", The} x {"", Year,
// WithoutYear}, plus TitleFirstCharacter.
var variants = []variant{
	{"CleanTitle", false, KindTitle, "%s with \"&\" as \"and\", and its accents, brackets and scene punctuation removed.", func(t string, _ int64) string { return cleanTitle(t) }},
	{"TitleThe", false, KindTitle, "%s with a leading \"The\", \"A\" or \"An\" moved to the end.", func(t string, _ int64) string { return titleThe(t) }},
	{"CleanTitleThe", false, KindTitle, "%s cleaned, with a leading \"The\", \"A\" or \"An\" moved to the end.", func(t string, _ int64) string { return cleanTitleThe(t) }},
	{"TitleYear", true, KindTitle, "%s with its year in brackets, unless it already ends with one.", titleYear},
	{"CleanTitleYear", true, KindTitle, "%s cleaned, with its year after it.", func(t string, y int64) string { return cleanTitle(titleYear(t, y)) }},
	{"TitleWithoutYear", true, KindTitle, "%s without a year in brackets at its end.", func(t string, _ int64) string { return titleWithoutYear(t) }},
	{"CleanTitleWithoutYear", true, KindTitle, "%s cleaned, without a year at its end.", func(t string, _ int64) string { return cleanTitle(titleWithoutYear(t)) }},
	{"TitleTheYear", true, KindTitle, "%s with a leading article moved to the end and its year in brackets.", func(t string, y int64) string { return titleYear(titleThe(t), y) }},
	{"CleanTitleTheYear", true, KindTitle, "%s cleaned, with a leading article moved to the end and its year after it.", cleanTitleTheYear},
	{"TitleTheWithoutYear", true, KindTitle, "%s with a leading article moved to the end and no year at its end.", func(t string, _ int64) string { return titleWithoutYear(titleThe(t)) }},
	{"CleanTitleTheWithoutYear", true, KindTitle, "%s cleaned, with a leading article moved to the end and no year at its end.", func(t string, _ int64) string { return cleanTitleThe(titleWithoutYear(t)) }},
	{"TitleFirstCharacter", false, KindText, "The first letter or digit of %s after its article, for folders by letter.", func(t string, _ int64) string { return titleFirstCharacter(titleThe(t)) }},
}

// cleanTitle is Sonarr's CleanTitle. Its spaces are folded to one, which
// Sonarr leaves to the cleanup over the whole name.
func cleanTitle(t string) string {
	t = strings.ReplaceAll(t, "&", "and")
	t = strings.ReplaceAll(t, "/", " ")
	rs := []rune(t)
	var b strings.Builder
	for i, r := range rs {
		if !sceneDropped(rs, i) {
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(spaces.ReplaceAllString(withoutDiacritics(b.String()), " "))
}

var spaces = regexp.MustCompile(`\s+`)

// sceneDropped is Sonarr's ScenifyRemoveChars at rs[i], whose lookarounds
// RE2 lacks: a lone punctuation mark between spaces; an apostrophe,
// backtick, colon, question mark or comma before a space, the end, or a
// contraction's tail and a space; and every bracket.
func sceneDropped(rs []rune, i int) bool {
	r := rs[i]
	space := func(j int) bool { return j >= 0 && j < len(rs) && unicode.IsSpace(rs[j]) }
	switch {
	case strings.ContainsRune("()[]{}", r):
		return true
	case strings.ContainsRune(`,<>/\;:'"|`+"`’~!?@%*-_=", r) && space(i-1) && space(i+1):
		return true
	case !strings.ContainsRune("'`’:?,", r):
		return false
	case i+1 == len(rs) || space(i+1):
		return true
	}
	rest := strings.ToLower(string(rs[i+1:]))
	for _, tail := range []string{"s", "m", "t", "ve", "ll", "d", "re"} {
		if after, ok := strings.CutPrefix(rest, tail); ok && after != "" {
			if r, _ := utf8.DecodeRuneInString(after); unicode.IsSpace(r) {
				return true
			}
		}
	}
	return false
}

// moreDiacritics are the letters the Diacritical package Sonarr uses maps
// that decompose to no base letter.
var moreDiacritics = strings.NewReplacer(
	"Æ", "AE", "æ", "ae", "Œ", "OE", "œ", "oe", "Ø", "O", "ø", "o", "ß", "ss",
	"Đ", "D", "đ", "d", "Ł", "L", "ł", "l", "Ð", "D", "ð", "d", "Þ", "TH", "þ", "th",
)

func withoutDiacritics(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return moreDiacritics.Replace(norm.NFC.String(b.String()))
}

func titleThe(t string) string { return titlePrefix.ReplaceAllString(t, "$2, $1$3") }

func cleanTitleThe(t string) string {
	m := titlePrefix.FindStringSubmatch(t)
	if m == nil {
		return cleanTitle(t)
	}
	rest := cleanTitle(m[3])
	if rest != "" {
		rest = " " + rest
	}
	return cleanTitle(m[2]) + ", " + m[1] + rest
}

// titleYear never adds a year the title already ends with.
func titleYear(t string, year int64) string {
	if year == 0 || yearEnd.MatchString(t) {
		return t
	}
	return t + " (" + strconv.FormatInt(year, 10) + ")"
}

func cleanTitleTheYear(t string, year int64) string {
	if year == 0 {
		return cleanTitleThe(t)
	}
	if m := yearEnd.FindStringIndex(t); m != nil {
		return cleanTitleThe(strings.TrimSpace(t[:m[0]])) + " " + t[m[0]+1:m[1]-1]
	}
	return cleanTitleThe(t) + " " + strconv.FormatInt(year, 10)
}

func titleWithoutYear(t string) string { return yearEnd.ReplaceAllString(t, "") }

// titleFirstCharacter is the first of t's first two characters that is a
// letter or digit, capitalized without its accent, or "_".
func titleFirstCharacter(t string) string {
	for i, r := range []rune(t) {
		if i == 2 {
			break
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			c, _ := utf8.DecodeRuneInString(withoutDiacritics(string(unicode.ToUpper(r))))
			return string(c)
		}
	}
	return "_"
}

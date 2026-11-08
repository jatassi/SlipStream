package movie

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"

	"golang.org/x/text/unicode/norm"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/upstream"
	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release"
)

var (
	titlePatterns        = compileAll(upstream.RadarrParserReportMovieTitleRegex)
	alternativeTitle     = compile(upstream.RadarrParserAlternativeTitleRegex)
	bracketedAlternative = compile(upstream.RadarrParserBracketedAlternativeTitleRegex)
	normalizeAlternative = compile(upstream.RadarrParserNormalizeAlternativeTitleRegex)
	requestInfo          = compile(upstream.RadarrParserRequestInfoRegex)
	normalizeTitle       = compile(upstream.RadarrParserNormalizeRegex)
	simpleReleaseTitle   = compile(upstream.RadarrParserSimpleReleaseTitleRegex)
)

// ReadName ports Radarr's ParseMovieTitle from its pattern loop: the
// first of ReportMovieTitleRegex to match p.Simple and give a title
// reads the titles and year. Masked and Tail are Radarr's
// simpleReleaseTitle with that title replaced by "A Movie", which Radarr
// reads the group, languages and edition from.
func (*Movie) ReadName(p release.Prepared) release.Name {
	for _, re := range titlePatterns {
		m, ok := re.find(p.Simple)
		if !ok {
			continue
		}
		n, ok := readMatch(m)
		if !ok {
			continue
		}
		masked, ok := mask(p.Release.String(), m)
		if !ok {
			return release.Name{}
		}
		n.Masked, n.Tail = masked, masked
		return n
	}
	return release.Name{}
}

// readMatch ports Radarr's ParseMovieMatchCollection.
func readMatch(m regex.Match) (release.Name, bool) {
	title, ok := m.Group("title")
	if !ok || title == "(" {
		return release.Name{}, false
	}
	name := normalizeAlternative.replace(strings.ReplaceAll(title, "_", " "), " AKA ")
	name = strings.Trim(requestInfo.replace(name, ""), " ")
	name = joinParts(strings.Split(name, "."))

	n := release.Name{Titles: []string{name}}
	if y, ok := m.Group("year"); ok {
		if year, ok := netInt(y); ok {
			n.Year = release.Known(year)
		}
	}
	for _, t := range alternativeTitle.split(bracketedAlternative.replace(name, "$1 AKA $2")) {
		if strings.TrimSpace(t) != "" && t != name {
			n.Titles = append(n.Titles, t)
		}
	}
	return n, true
}

// joinParts joins a dotted title's parts with spaces, keeping the dots
// of acronyms such as U.N.C.L.E. and of "Dr.".
func joinParts(parts []string) string {
	var b strings.Builder
	acronym := false
	for n, part := range parts {
		next := ""
		if len(parts) >= n+2 {
			next = parts[n+1]
		}
		_, partInt := netInt(part)
		_, nextInt := netInt(next)
		lower := strings.ToLower(part)
		switch {
		case units(part) == 1 && lower != "a" && !partInt &&
			(acronym || n < len(parts)-1) &&
			(acronym || units(next) != 1 || !nextInt):
			b.WriteString(part + ".")
			acronym = true
		case lower == "a" && (acronym || units(next) == 1), lower == "dr":
			b.WriteString(part + ".")
			acronym = true
		default:
			if acronym {
				b.WriteString(" ")
				acronym = false
			}
			b.WriteString(part + " ")
		}
	}
	return strings.Trim(b.String(), " ")
}

// mask is Radarr's simpleReleaseTitle with the matched title replaced.
// Radarr cuts at the title's place in the simple title, which a blanked
// pattern before it can shift past the end. Radarr then throws and
// reads no name, and so does mask.
func mask(releaseTitle string, m regex.Match) (string, bool) {
	simple := []rune(simpleReleaseTitle.replace(strings.Trim(releaseTitle, "-_"), ""))
	title, _ := m.Group("title")
	if strings.TrimSpace(title) == "" {
		return string(simple), true
	}
	at, length, _ := m.GroupSpan("title")
	if at+length > len(simple) {
		return "", false
	}
	placeholder := "A Movie"
	if strings.Contains(title, ".") {
		placeholder = "A.Movie"
	}
	return string(simple[:at]) + placeholder + string(simple[at+length:]), true
}

// CleanTitle ports Radarr's CleanMovieTitle.
func (*Movie) CleanTitle(title string) string {
	if strings.TrimSpace(title) == "" {
		return title
	}
	if _, err := strconv.ParseInt(strings.TrimSpace(title), 10, 64); err == nil {
		return title
	}
	return removeAccent(germanUmlauts.Replace(strings.ToLower(normalizeTitle.replace(title, ""))))
}

var germanUmlauts = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "Ä", "Ae", "Ö", "Oe", "Ü", "Ue", "ß", "ss")

func removeAccent(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(s) {
		if !unicode.Is(unicode.Mn, r) {
			b.WriteRune(r)
		}
	}
	return norm.NFC.String(b.String())
}

// netInt is .NET's int.TryParse: an int32, with surrounding spaces.
func netInt(s string) (int, bool) {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	return int(n), err == nil
}

// units is a string's .NET Length, in UTF-16 code units.
func units(s string) int { return len(utf16.Encode([]rune(s))) }

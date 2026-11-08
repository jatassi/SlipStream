package release

import (
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release/internal/upstream"
)

// upstreamLanguages is each of Sonarr's and Radarr's Language members by
// name, as the code of its first IsoLanguages row. Original has no row.
var upstreamLanguages = func() map[string]Language {
	m := map[string]Language{"Original": Original}
	for _, row := range slices.Concat(upstream.SonarrIsoLanguagesAll, upstream.RadarrIsoLanguagesAll) {
		if _, ok := m[row.Language]; ok {
			continue
		}
		l, err := ParseLanguage(row.Args[0])
		if err != nil {
			panic(err)
		}
		m[row.Language] = l
	}
	return m
}()

// UpstreamLanguage is the language Sonarr's or Radarr's Language member
// name is, and false for one with no ISO 639 code, such as Unknown.
func UpstreamLanguage(name string) (Language, bool) {
	l, ok := upstreamLanguages[name]
	return l, ok
}

var (
	cleanLanguageTitle    = compileReplaces(upstream.SonarrLanguageParserCleanShowTitleRegex)
	caseSensitiveLanguage = compile(upstream.SonarrLanguageParserCaseSensitiveLanguageRegex)
	languageTags          = compile(upstream.SonarrLanguageParserLanguageRegex)
	germanDual            = compile(upstream.SonarrLanguageParserGermanDualLanguageRegex)
	germanMulti           = compile(upstream.SonarrLanguageParserGermanMultiLanguageRegex)
)

// languageWords are Sonarr's language words, then those only Radarr
// knows, such as Persian, Telugu and Tagalog.
var languageWords = func() []upstream.Adds {
	words := slices.Clone(upstream.SonarrLanguageParserParseLanguagesContains)
	for _, a := range upstream.RadarrLanguageParserParseLanguagesContains {
		if !slices.ContainsFunc(words, func(b upstream.Adds) bool { return slices.Equal(a.Words, b.Words) }) {
			words = append(words, a)
		}
	}
	return words
}()

// readLanguages ports Sonarr's LanguageParser.ParseLanguages, reading
// Radarr's words too. A name that names no language reads as unknown.
func readLanguages(name string) Fact[[]Language] {
	t := regex.TextOf(name)
	for _, r := range cleanLanguageTitle {
		if r.match(t) {
			t = r.replace(t)
			break
		}
	}
	lower := strings.ToLower(t.String())
	var names []string
	for _, a := range languageWords {
		if slices.ContainsFunc(a.Words, func(w string) bool { return strings.Contains(lower, w) }) {
			names = append(names, a.Value)
		}
	}
	var matches []regex.Match
	if m, ok := caseSensitiveLanguage.find(t); ok {
		matches = append(matches, m)
	}
	for _, m := range append(matches, languageTags.findAll(t)...) {
		names = append(names, groupLanguages(m)...)
	}
	if len(names) == 0 {
		return Fact[[]Language]{}
	}
	if len(names) == 1 && names[0] == "German" {
		switch {
		case germanDual.match(t):
			names = append(names, "Original")
		case germanMulti.match(t):
			names = append(names, "Original", "English")
		}
	}
	var out []Language
	for _, n := range names {
		if l := upstreamLanguages[n]; !slices.Contains(out, l) {
			out = append(out, l)
		}
	}
	return Known(out)
}

// groupLanguages is the language of each of m's groups that matched. The
// generated table joins Sonarr's list for its case-sensitive pattern and
// its list for the other, so a group both list is tested once.
func groupLanguages(m regex.Match) []string {
	var tested, names []string
	for _, a := range upstream.SonarrLanguageParserRegexLanguageGroups {
		group := a.Words[0]
		if slices.Contains(tested, group) {
			continue
		}
		tested = append(tested, group)
		if _, ok := m.Group(group); ok {
			names = append(names, a.Value)
		}
	}
	return names
}

// withoutGroup is masked with the group replaced by "RlsGrp", as Radarr
// reads languages, so a group named like a language reads as none.
func withoutGroup(masked string, group Fact[string]) string {
	g, ok := group.Get()
	if !ok {
		return masked
	}
	return strings.ReplaceAll(masked, g, "RlsGrp")
}

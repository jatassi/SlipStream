package release

import (
	"slices"

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

// readLanguages is the languages tail names: unknown until LanguageParser
// is ported.
func readLanguages(string) Fact[[]Language] { return Fact[[]Language]{} }

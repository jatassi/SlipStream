package release

import (
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release/internal/upstream"
)

var (
	releaseGroup          = compile(upstream.RadarrReleaseGroupParserReleaseGroupRegex)
	invalidReleaseGroup   = compile(upstream.RadarrReleaseGroupParserInvalidReleaseGroupRegex)
	animeReleaseGroup     = compile(upstream.RadarrReleaseGroupParserAnimeReleaseGroupRegex)
	exceptionGroupExact   = compile(upstream.RadarrReleaseGroupParserExceptionReleaseGroupRegexExact)
	exceptionGroup        = compile(upstream.RadarrReleaseGroupParserExceptionReleaseGroupRegex)
	cleanReleaseGroup     = compileReplace(upstream.RadarrReleaseGroupParserCleanReleaseGroupRegex)
	groupWebsitePrefix    = compileReplace(upstream.RadarrParserCommonWebsitePrefixRegex)
	groupTorrentSuffix    = compileReplace(upstream.RadarrParserCommonCleanTorrentSuffixRegex)
	groupPreSubstitutions = compileReplaces(upstream.RadarrParserCommonPreSubstitutionRegex)
)

// readGroup ports Radarr's ReleaseGroupParser.ParseReleaseGroup over
// Sonarr's release title. Radarr's differs from Sonarr's on fewer
// fixture rows, and its exception lists hold the groups TRaSH's Radarr
// formats match by group (YTS, YIFY, Tigole, D-Z0N3), while TRaSH's
// Sonarr formats match Sonarr's exception groups by title.
func readGroup(p Prepared, s *scan) Fact[string] {
	t := regex.TextOf(withoutExtension(s, strings.TrimSpace(p.release.String())))
	for _, sub := range groupPreSubstitutions {
		if s.match(sub.fixed, t) {
			t = s.replace(sub, t)
			break
		}
	}
	t = s.replace(groupWebsitePrefix, t)
	t = s.replace(groupTorrentSuffix, t)
	if m, ok := s.find(animeReleaseGroup, t); ok {
		return Known(captured(m, "subgroup"))
	}
	t = s.replace(cleanReleaseGroup, t)
	for _, f := range []fixed{exceptionGroupExact, exceptionGroup} {
		if all := s.findAll(f, t); len(all) > 0 {
			return Known(captured(all[len(all)-1], "releasegroup"))
		}
	}
	all := s.findAll(releaseGroup, t)
	if len(all) == 0 {
		return Fact[string]{}
	}
	g := captured(all[len(all)-1], "releasegroup")
	if _, err := strconv.ParseInt(g, 10, 32); err == nil || s.match(invalidReleaseGroup, regex.TextOf(g)) {
		return Fact[string]{}
	}
	return Known(g)
}

func captured(m regex.Match, group string) string {
	v, _ := m.Group(group)
	return v
}

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

// readGroup ports Radarr's ReleaseGroupParser.ParseReleaseGroup.
// Radarr's differs from Sonarr's on fewer fixture rows, and its
// exception lists hold the groups TRaSH's Radarr formats match by group
// (YTS, YIFY, Tigole, D-Z0N3), while TRaSH's Sonarr formats match
// Sonarr's exception groups by title.
func readGroup(name string) Fact[string] {
	t := regex.TextOf(withoutExtension(strings.TrimSpace(name)))
	for _, sub := range groupPreSubstitutions {
		if sub.match(t) {
			t = sub.replace(t)
			break
		}
	}
	t = groupWebsitePrefix.replace(t)
	t = groupTorrentSuffix.replace(t)
	if m, ok := animeReleaseGroup.find(t); ok {
		return Known(captured(m, "subgroup"))
	}
	t = cleanReleaseGroup.replace(t)
	for _, f := range []fixed{exceptionGroupExact, exceptionGroup} {
		if all := f.findAll(t); len(all) > 0 {
			return Known(captured(all[len(all)-1], "releasegroup"))
		}
	}
	all := releaseGroup.findAll(t)
	if len(all) == 0 {
		return Fact[string]{}
	}
	g := captured(all[len(all)-1], "releasegroup")
	if _, err := strconv.ParseInt(g, 10, 32); err == nil || invalidReleaseGroup.match(regex.TextOf(g)) {
		return Fact[string]{}
	}
	return Known(g)
}

func captured(m regex.Match, group string) string {
	v, _ := m.Group(group)
	return v
}

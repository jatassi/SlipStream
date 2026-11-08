package release

import (
	"strconv"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release/internal/upstream"
)

var (
	reportImdbID = compile(upstream.RadarrParserReportImdbId)
	reportTmdbID = compile(upstream.RadarrParserReportTmdbId)
	// Neither upstream reads a TVDB tag from a name. SlipStream reads
	// Plex's {tvdb-81189} by Radarr's TMDB rule.
	reportTvdbID = compile(upstream.Pattern{Key: "SlipStream.ReportTvdbId", Text: `tvdb(id)?-(?<tvdbid>\d+)`, Options: upstream.IgnoreCase | upstream.Compiled})
)

var numberIDs = []struct {
	key string
	f   fixed
}{{"tmdbid", reportTmdbID}, {"tvdbid", reportTvdbID}}

// readIDs ports Radarr's ParseImdbId and ParseTmdbId over the name the
// group reads, Radarr's simpleReleaseTitle: the id tags it carries, by
// Torznab's names. An IMDb id is "tt" and 7 or 8 digits, and a TMDB or
// TVDB id a positive int32.
func readIDs(name string) map[string]string {
	t := regex.TextOf(name)
	ids := map[string]string{}
	if m, ok := reportImdbID.find(t); ok {
		if v, _ := m.Group("imdbid"); len(v) == 9 || len(v) == 10 {
			ids["imdbid"] = v
		}
	}
	for _, id := range numberIDs {
		m, ok := id.f.find(t)
		if !ok {
			continue
		}
		v, _ := m.Group(id.key)
		if n, err := strconv.ParseInt(v, 10, 32); err == nil && n != 0 {
			ids[id.key] = strconv.FormatInt(n, 10)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

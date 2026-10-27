package cardigann

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
)

// Settings are one Indexer's setting values by field name, opened, and the
// link it uses.
type Settings struct {
	// Values hold a select's option key, a checkbox's "true" or "false",
	// and text as typed. A setting missing here takes its default.
	Values map[string]string
	// BaseURL is the link the Admin chose. Empty, or one of the
	// definition's legacy links, means its first link.
	BaseURL string
}

// Query is one search.
type Query struct {
	// Mode is "search", "tv-search", "movie-search", "music-search" or
	// "book-search".
	Mode string
	Text string
	// IDs are keyed imdbid (tt form), tmdbid, tvdbid, rid, tvmazeid,
	// traktid and doubanid.
	IDs map[string]string
	// Season and Year are 0 when the search has none.
	Season     int
	Episode    string
	Year       int
	Categories []newznab.ID
}

// queryType is a mode's Torznab t value, Prowlarr's .Query.Type:
// <x>-search is t=x, but tv-search is t=tvsearch.
func queryType(mode string) (string, bool) {
	switch mode {
	case "search":
		return "search", true
	case "tv-search":
		return "tvsearch", true
	}
	_, known := modeParams[mode]
	t, _ := strings.CutSuffix(mode, "-search")
	return t, known
}

// queryIDVars are the .Query names each mode sets from Query.IDs, by key,
// besides the IMDb id's two forms.
var queryIDVars = map[string][]entry[string]{
	"movie-search": {{"tmdbid", "TMDBID"}, {"traktid", "TraktID"}, {"doubanid", "DoubanID"}},
	"tv-search": {
		{"tvdbid", "TVDBID"}, {"tmdbid", "TMDBID"}, {"rid", "TVRageID"},
		{"tvmazeid", "TVMazeID"}, {"traktid", "TraktID"}, {"doubanid", "DoubanID"},
	},
}

// queryVarNames are every .Query name Prowlarr sets before a search fills
// them (CardigannRequestGenerator.cs:150-193).
var queryVarNames = []string{
	".Query.Type", ".Query.Q", ".Query.Categories", ".Query.Limit", ".Query.Offset", ".Query.Extended", ".Query.APIKey", ".Query.Genre",
	".Query.Movie", ".Query.Year", ".Query.IMDBID", ".Query.IMDBIDShort", ".Query.TMDBID",
	".Query.Series", ".Query.Ep", ".Query.Season", ".Query.TVDBID", ".Query.TVRageID", ".Query.TVMazeID", ".Query.TraktID", ".Query.DoubanID", ".Query.Episode",
	".Query.Album", ".Query.Artist", ".Query.Label", ".Query.Track",
	".Query.Author", ".Query.Title", ".Query.Publisher",
}

// siteLink is ResolveSiteLink: the chosen link, or the first link when none
// is chosen or the chosen one is a legacy link.
func (d *Definition) siteLink(s Settings) string {
	if s.BaseURL == "" || slices.Contains(d.def.LegacyLinks, s.BaseURL) {
		return d.def.Links[0]
	}
	return s.BaseURL
}

// baseScope is GetBaseTemplateVariables (CardigannBase.cs:277-356): the
// site link, .True, .False, .Today.Year and a .Config entry per setting
// that holds a value. A select holds its option key and a checked checkbox
// the string ".True".
func (d *Definition) baseScope(s Settings, c Clock) scope {
	sc := scope{
		".Config.sitelink": textValue(d.siteLink(s)),
		".True":            textValue("True"),
		".False":           nullValue,
		".Today.Year":      textValue(strconv.Itoa(c.local().Year())),
	}
	for _, f := range d.def.Settings {
		v, set := s.Values[f.Name]
		if !set {
			v = f.Default
		}
		name := ".Config." + f.Name
		switch f.Type {
		case "text", "password", "select":
			sc[name] = textValue(v)
			if !set {
				sc[name] = optionalText(v)
			}
		case "checkbox":
			sc[name] = nullValue
			if checked, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil && checked {
				sc[name] = textValue(".True")
			}
		}
	}
	return sc
}

// optionalText is s, or null when s is empty, as a C# string that was
// never set.
func optionalText(s string) value {
	if s == "" {
		return nullValue
	}
	return textValue(s)
}

// searchScope is the base scope with Prowlarr's query variables for q
// (CardigannRequestGenerator.cs:48-193) and .Query.Keywords.
func (d *Definition) searchScope(s Settings, q Query, c Clock) (scope, error) {
	qtype, ok := queryType(q.Mode)
	if !ok {
		return nil, fmt.Errorf("the search mode %q isn't one SlipStream knows", q.Mode)
	}
	sc := d.baseScope(s, c)
	for _, n := range queryVarNames {
		sc[n] = nullValue
	}
	cats := make([]string, len(q.Categories))
	for i, id := range q.Categories {
		cats[i] = strconv.Itoa(int(id))
	}
	sc[".Query.Type"] = textValue(qtype)
	sc[".Query.Q"] = optionalText(q.Text)
	sc[".Query.Categories"] = listValue(cats)
	if ids, ok := queryIDVars[q.Mode]; ok {
		if q.Year != 0 {
			sc[".Query.Year"] = textValue(strconv.Itoa(q.Year))
		}
		for _, id := range ids {
			sc[".Query."+id.Value] = optionalText(q.IDs[id.Key])
		}
		full, short := imdbForms(q.IDs["imdbid"])
		sc[".Query.IMDBID"], sc[".Query.IMDBIDShort"] = full, short
	}
	if q.Mode == "tv-search" {
		sc[".Query.Ep"] = optionalText(q.Episode)
		if q.Season != 0 {
			sc[".Query.Season"] = textValue(strconv.Itoa(q.Season))
		}
		sc[".Query.Episode"] = optionalText(episodeSearchString(q.Season, q.Episode, c))
	}
	sc[".Query.Keywords"] = textValue(keywords(sc))
	return sc, nil
}

// imdbForms are IMDBID and IMDBIDShort: tt and seven digits, and the
// seven digits. Both are null for an id that isn't a number above 0.
func imdbForms(id string) (full, short value) {
	n, err := strconv.Atoi(strings.TrimPrefix(strings.TrimSpace(id), "tt"))
	if err != nil || n <= 0 {
		return nullValue, nullValue
	}
	return textValue(fmt.Sprintf("tt%07d", n)), textValue(fmt.Sprintf("%07d", n))
}

// keywords joins the query's text, series, movie and year, then the
// episode, skipping blanks (CardigannRequestGenerator.cs:1079-1096).
func keywords(sc scope) string {
	var tokens []string
	for _, k := range []string{".Query.Q", ".Query.Series", ".Query.Movie", ".Query.Year", ".Query.Episode"} {
		if v := sc[k]; !v.blank() {
			tokens = append(tokens, v.text)
		}
	}
	return strings.Join(tokens, " ")
}

// episodeSearchString is TvSearchCriteria.GetEpisodeSearchString: empty
// without a season, yyyy.MM.dd for a daily episode, else S01 or S01E02.
func episodeSearchString(season int, episode string, c Clock) string {
	if season == 0 {
		return ""
	}
	if t, ok := parseExact(fmt.Sprintf("%d %s", season, episode), "yyyy MM/dd", c); ok {
		return t.Format("2006.01.02")
	}
	if strings.TrimSpace(episode) == "" {
		return fmt.Sprintf("S%02d", season)
	}
	if n, err := coerceInt(episode); err == nil {
		return fmt.Sprintf("S%02dE%02d", season, n)
	}
	return fmt.Sprintf("S%02dE%s", season, episode)
}

package tvdb

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jatassi/slipstream/fakes/scenario"
)

// seasonType is one of the orders TVDB lists episodes in.
type seasonType struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	AlternateName string `json:"alternateName"`
}

var (
	aired    = seasonType{1, "Aired Order", "official", ""}
	dvd      = seasonType{2, "DVD Order", "dvd", ""}
	absolute = seasonType{3, "Absolute Order", "absolute", ""}
)

// typesOf are the orders series has: Aired always, DVD and Absolute when
// the Scenario numbers its episodes in them.
func typesOf(series scenario.TVDBSeries) []seasonType {
	types := []seasonType{aired}
	if len(series.DVD) > 0 {
		types = append(types, dvd)
	}
	if len(series.Absolute) > 0 {
		types = append(types, absolute)
	}
	return types
}

// placed is an episode at its place in one order.
type placed struct {
	scenario.TVDBEpisode
	season, number int
}

// episodesIn is every episode of series in order t, by season and number.
func episodesIn(series scenario.TVDBSeries, t seasonType) []placed {
	var out []placed
	for _, season := range series.Seasons {
		for _, e := range season.Episodes {
			switch t {
			case aired:
				out = append(out, placed{e, season.Number, e.Number})
			case dvd:
				if n, ok := series.DVD[e.ID]; ok {
					out = append(out, placed{e, n.Season, n.Episode})
				}
			case absolute:
				if n, ok := series.Absolute[e.ID]; ok {
					out = append(out, placed{e, 1, n})
				}
			}
		}
	}
	slices.SortStableFunc(out, func(a, b placed) int { return cmp.Or(cmp.Compare(a.season, b.season), cmp.Compare(a.number, b.number)) })
	return out
}

// seasonID is the id TVDB gives series' season number in order t.
func seasonID(series int64, t seasonType, number int) int64 {
	return series*1000 + int64(t.ID-1)*300 + int64(number)
}

// dates are series' first and last aired days and its next airing, from
// its Aired episodes outside the specials.
func dates(series scenario.TVDBSeries) (first, last, next string) {
	today := time.Now().UTC().Format(time.DateOnly)
	for _, e := range episodesIn(series, aired) {
		if e.season == 0 || e.Aired == "" {
			continue
		}
		if first == "" || e.Aired < first {
			first = e.Aired
		}
		if e.Aired <= today && e.Aired > last {
			last = e.Aired
		}
		if e.Aired > today && (next == "" || e.Aired < next) {
			next = e.Aired
		}
	}
	return first, last, next
}

func averageRuntime(series scenario.TVDBSeries) *int {
	sum, n := 0, 0
	for _, season := range series.Seasons {
		for _, e := range season.Episodes {
			if e.Runtime > 0 {
				sum, n = sum+e.Runtime, n+1
			}
		}
	}
	if n == 0 {
		return nil
	}
	avg := sum / n
	return &avg
}

// originalLanguage is the recorded one, else jpn for a series whose
// English name differs from its own, else eng.
func originalLanguage(series scenario.TVDBSeries) string {
	switch {
	case series.OriginalLanguage != "":
		return series.OriginalLanguage
	case series.English != "" && series.English != series.Name:
		return "jpn"
	default:
		return "eng"
	}
}

func englishName(series scenario.TVDBSeries) string { return cmp.Or(series.English, series.Name) }

// overview is the series' English overview, marked by each Change cue.
func overview(series scenario.TVDBSeries, rev int) string {
	s := cmp.Or(series.Overview, englishName(series)+" is a fictional series.")
	for n := range rev {
		s = fmt.Sprintf("%s Revision %d.", s, n+1)
	}
	return s
}

func statusName(series scenario.TVDBSeries) string {
	r := []rune(series.Status)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

var statusIDs = map[string]int{"continuing": 1, "ended": 2, "upcoming": 3}

var genreNames = []string{"Drama", "Comedy", "Mystery", "Science Fiction", "Documentary", "Animation", "Crime", "Fantasy"}

func genres(series scenario.TVDBSeries) []genreJSON {
	if len(series.Genres) > 0 {
		out := make([]genreJSON, len(series.Genres))
		for i, name := range series.Genres {
			out[i] = genreJSON{ID: int(hash("genre", name) % 1000), Name: name, Slug: slug(name)}
		}
		return out
	}
	first := int(hash("genres", strconv.FormatInt(series.ID, 10)) % uint64(len(genreNames))) //nolint:gosec // G115: the remainder fits the list
	second := (first + 1) % len(genreNames)
	return []genreJSON{genreOf(first), genreOf(second)}
}

func genreOf(i int) genreJSON {
	return genreJSON{ID: i + 1, Name: genreNames[i], Slug: slug(genreNames[i])}
}

func slug(name string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(name), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }), "-")
}

const lastUpdated = "2026-01-01 00:00:00"

type baseJSON struct {
	ID                   int64       `json:"id"`
	Name                 string      `json:"name"`
	Slug                 string      `json:"slug"`
	Image                string      `json:"image"`
	NameTranslations     []string    `json:"nameTranslations"`
	OverviewTranslations []string    `json:"overviewTranslations"`
	Aliases              []aliasJSON `json:"aliases"`
	FirstAired           string      `json:"firstAired"`
	LastAired            string      `json:"lastAired"`
	NextAired            string      `json:"nextAired"`
	Score                float64     `json:"score"`
	Status               statusJSON  `json:"status"`
	OriginalCountry      string      `json:"originalCountry"`
	OriginalLanguage     string      `json:"originalLanguage"`
	DefaultSeasonType    int         `json:"defaultSeasonType"`
	IsOrderRandomized    bool        `json:"isOrderRandomized"`
	LastUpdated          string      `json:"lastUpdated"`
	AverageRuntime       *int        `json:"averageRuntime"`
	Year                 string      `json:"year"`
}

type aliasJSON struct {
	Language string `json:"language"`
	Name     string `json:"name"`
}

type statusJSON struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	RecordType  string `json:"recordType"`
	KeepUpdated bool   `json:"keepUpdated"`
}

type extendedJSON struct {
	baseJSON
	Overview        string            `json:"overview"`
	OriginalNetwork *companyJSON      `json:"originalNetwork,omitempty"`
	LatestNetwork   *companyJSON      `json:"latestNetwork,omitempty"`
	Companies       []companyJSON     `json:"companies"`
	Genres          []genreJSON       `json:"genres"`
	RemoteIDs       []remoteIDJSON    `json:"remoteIds"`
	Seasons         []seasonJSON      `json:"seasons"`
	SeasonTypes     []seasonType      `json:"seasonTypes"`
	Translations    *translationsJSON `json:"translations,omitempty"`
}

type companyJSON struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	Slug               string `json:"slug"`
	Country            string `json:"country"`
	PrimaryCompanyType int    `json:"primaryCompanyType"`
}

type genreJSON struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type remoteIDJSON struct {
	ID         string `json:"id"`
	Type       int    `json:"type"`
	SourceName string `json:"sourceName"`
}

type seasonJSON struct {
	ID                   int64      `json:"id"`
	SeriesID             int64      `json:"seriesId"`
	Number               int        `json:"number"`
	Name                 string     `json:"name,omitempty"`
	Image                string     `json:"image"`
	ImageType            int        `json:"imageType"`
	Type                 seasonType `json:"type"`
	NameTranslations     []string   `json:"nameTranslations"`
	OverviewTranslations []string   `json:"overviewTranslations"`
	LastUpdated          string     `json:"lastUpdated"`
}

type translationsJSON struct {
	NameTranslations     []translationJSON `json:"nameTranslations"`
	OverviewTranslations []translationJSON `json:"overviewTranslations"`
	Alias                []string          `json:"alias"`
}

type translationJSON struct {
	Language  string `json:"language"`
	Name      string `json:"name,omitempty"`
	Overview  string `json:"overview,omitempty"`
	IsPrimary bool   `json:"isPrimary"`
}

func base(series scenario.TVDBSeries) baseJSON {
	first, last, next := dates(series)
	langs := []string{"eng"}
	if l := originalLanguage(series); l != "eng" {
		langs = append(langs, l)
	}
	b := baseJSON{
		ID: series.ID, Name: series.Name, Slug: slug(englishName(series)), Image: recorded(series.Image, "series", series.ID, "posters"),
		NameTranslations: langs, OverviewTranslations: []string{"eng"}, Aliases: []aliasJSON{},
		FirstAired: first, LastAired: last, NextAired: next,
		Status:          statusJSON{ID: statusIDs[series.Status], Name: statusName(series), RecordType: "series", KeepUpdated: series.Status != "ended"},
		OriginalCountry: cmp.Or(series.OriginalCountry, "usa"), OriginalLanguage: originalLanguage(series), DefaultSeasonType: aired.ID, LastUpdated: lastUpdated,
		AverageRuntime: averageRuntime(series),
	}
	if first != "" {
		b.Year = first[:4]
	}
	for _, a := range series.Aliases {
		b.Aliases = append(b.Aliases, aliasJSON{Language: "eng", Name: a})
	}
	return b
}

func network(series scenario.TVDBSeries) *companyJSON {
	if series.Network == "" {
		return nil
	}
	return &companyJSON{ID: int64(hash("network", series.Network) % 100_000), Name: series.Network, Slug: slug(series.Network), Country: "usa", PrimaryCompanyType: 1}
}

func (s *server) series(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		failure(w, http.StatusBadRequest, "InvalidSeriesID: "+r.PathValue("id")+" is not a series id")
		return
	}
	series, rev, ok := s.live(id)
	if !ok {
		failure(w, http.StatusNotFound, notFoundWords)
		return
	}
	out := extendedJSON{
		baseJSON: base(series), Overview: overview(series, rev), OriginalNetwork: network(series), LatestNetwork: network(series),
		Companies: []companyJSON{}, Genres: genres(series), RemoteIDs: []remoteIDJSON{}, Seasons: []seasonJSON{}, SeasonTypes: typesOf(series),
	}
	if n := out.OriginalNetwork; n != nil {
		out.Companies = append(out.Companies, *n)
	}
	if series.IMDb != "" {
		out.RemoteIDs = append(out.RemoteIDs, remoteIDJSON{ID: series.IMDb, Type: 2, SourceName: "IMDB"})
	}
	if series.TMDB != 0 {
		out.RemoteIDs = append(out.RemoteIDs, remoteIDJSON{ID: strconv.FormatInt(series.TMDB, 10), Type: 12, SourceName: "TheMovieDB.com"})
	}
	for _, t := range out.SeasonTypes {
		out.Seasons = append(out.Seasons, seasonsIn(series, t)...)
	}
	if r.URL.Query().Get("meta") == "translations" {
		out.Translations = translations(series, rev)
	}
	reply(w, http.StatusOK, envelope{Status: "success", Data: out})
}

// seasonsIn are the seasons of series in order t, each numbered once. An
// Aired season answers its recorded name and poster.
func seasonsIn(series scenario.TVDBSeries, t seasonType) []seasonJSON {
	var out []seasonJSON
	for _, e := range episodesIn(series, t) {
		if slices.ContainsFunc(out, func(s seasonJSON) bool { return s.Number == e.season }) {
			continue
		}
		var rec scenario.TVDBSeason
		if i := slices.IndexFunc(series.Seasons, func(s scenario.TVDBSeason) bool { return s.Number == e.season }); i >= 0 && t == aired {
			rec = series.Seasons[i]
		}
		sid := seasonID(series.ID, t, e.season)
		season := seasonJSON{
			ID: sid, SeriesID: series.ID, Number: e.season, Name: rec.Name, Image: recorded(rec.Image, "seasons", sid, "posters"), ImageType: 7, Type: t,
			NameTranslations: []string{}, OverviewTranslations: []string{}, LastUpdated: lastUpdated,
		}
		if e.season == 0 && season.Name == "" {
			season.Name = "Specials"
		}
		out = append(out, season)
	}
	return out
}

func translations(series scenario.TVDBSeries, rev int) *translationsJSON {
	out := &translationsJSON{
		NameTranslations:     []translationJSON{{Language: "eng", Name: englishName(series), IsPrimary: originalLanguage(series) == "eng"}},
		OverviewTranslations: []translationJSON{{Language: "eng", Overview: overview(series, rev), IsPrimary: true}},
		Alias:                append([]string{}, series.Aliases...),
	}
	if l := originalLanguage(series); l != "eng" {
		out.NameTranslations = append(out.NameTranslations, translationJSON{Language: l, Name: series.Name, IsPrimary: true})
	}
	return out
}

type episodeJSON struct {
	ID                   int64    `json:"id"`
	SeriesID             int64    `json:"seriesId"`
	Name                 string   `json:"name"`
	Aired                string   `json:"aired,omitempty"`
	Runtime              *int     `json:"runtime"`
	NameTranslations     []string `json:"nameTranslations"`
	Overview             string   `json:"overview"`
	OverviewTranslations []string `json:"overviewTranslations"`
	Image                string   `json:"image"`
	ImageType            int      `json:"imageType"`
	IsMovie              int      `json:"isMovie"`
	Number               int      `json:"number"`
	AbsoluteNumber       int      `json:"absoluteNumber"`
	SeasonNumber         int      `json:"seasonNumber"`
	LastUpdated          string   `json:"lastUpdated"`
	Year                 string   `json:"year,omitempty"`
}

// orderTypes are the season types an episodes path names, default being
// the series' default, Aired.
var orderTypes = map[string]seasonType{"default": aired, "official": aired, "dvd": dvd, "absolute": absolute}

func (s *server) episodes(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	t, known := orderTypes[r.PathValue("order")]
	if err != nil || !known {
		failure(w, http.StatusBadRequest, "InvalidSeasonType: "+r.PathValue("order")+" is not a season type")
		return
	}
	series, _, ok := s.live(id)
	if !ok {
		failure(w, http.StatusNotFound, notFoundWords)
		return
	}
	all := episodesIn(series, t)
	page := pageParam(r.URL.Query().Get("page"))
	episodes := []episodeJSON{}
	if from := page * episodesPage; from < len(all) {
		for _, e := range all[from:min(from+episodesPage, len(all))] {
			out := episodeJSON{
				ID: e.ID, SeriesID: series.ID, Name: e.Name, Aired: e.Aired, NameTranslations: []string{"eng"},
				Overview: cmp.Or(e.Overview, fmt.Sprintf("%s, an episode of %s.", e.Name, englishName(series))), OverviewTranslations: []string{"eng"},
				Image: recorded(e.Image, "episodes", e.ID, "screencap"), ImageType: 11, Number: e.number, AbsoluteNumber: series.Absolute[e.ID],
				SeasonNumber: e.season, LastUpdated: lastUpdated,
			}
			if e.Runtime > 0 {
				out.Runtime = &e.Runtime
			}
			if e.Aired != "" {
				out.Year = e.Aired[:4]
			}
			episodes = append(episodes, out)
		}
	}
	reply(w, http.StatusOK, envelope{Status: "success", Data: struct {
		Series   baseJSON      `json:"series"`
		Episodes []episodeJSON `json:"episodes"`
	}{base(series), episodes}, Links: pageLinks(r, page, len(all), episodesPage)})
}

type searchJSON struct {
	ObjectID        string            `json:"objectID"`
	Aliases         []string          `json:"aliases"`
	Companies       []string          `json:"companies"`
	Country         string            `json:"country"`
	ID              string            `json:"id"`
	ImageURL        string            `json:"image_url"`
	Name            string            `json:"name"`
	FirstAirTime    string            `json:"first_air_time,omitempty"`
	Overview        string            `json:"overview"`
	PrimaryLanguage string            `json:"primary_language"`
	Status          string            `json:"status"`
	Type            string            `json:"type"`
	TVDBID          string            `json:"tvdb_id"`
	Year            string            `json:"year,omitempty"`
	Slug            string            `json:"slug"`
	Overviews       map[string]string `json:"overviews"`
	Translations    map[string]string `json:"translations"`
	Network         string            `json:"network,omitempty"`
	RemoteIDs       []remoteIDJSON    `json:"remote_ids"`
	Thumbnail       string            `json:"thumbnail"`
}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query := strings.ToLower(cmp.Or(q.Get("query"), q.Get("q")))
	limit := cmp.Or(pageParam(q.Get("limit")), searchLimit)
	found := []searchJSON{}
	if query != "" && (q.Get("type") == "" || q.Get("type") == "series") {
		for listed := range s.catalog.All() {
			series, rev, ok := s.live(listed.ID)
			if !ok || !matches(series, query) {
				continue
			}
			b := base(series)
			id := strconv.FormatInt(series.ID, 10)
			hit := searchJSON{
				ObjectID: "series-" + id, Aliases: series.Aliases, Companies: []string{}, Country: b.OriginalCountry, ID: "series-" + id,
				ImageURL: b.Image, Name: series.Name, FirstAirTime: b.FirstAired, Overview: overview(series, rev), PrimaryLanguage: b.OriginalLanguage,
				Status: b.Status.Name, Type: "series", TVDBID: id, Year: b.Year, Slug: b.Slug,
				Overviews:    map[string]string{"eng": overview(series, rev)},
				Translations: map[string]string{"eng": englishName(series)},
				Network:      series.Network, RemoteIDs: []remoteIDJSON{}, Thumbnail: thumbnail(b.Image),
			}
			if hit.Aliases == nil {
				hit.Aliases = []string{}
			}
			if series.Network != "" {
				hit.Companies = append(hit.Companies, series.Network)
			}
			found = append(found, hit)
			if len(found) == limit {
				break
			}
		}
	}
	reply(w, http.StatusOK, envelope{Status: "success", Data: found, Links: pageLinks(r, 0, len(found), limit)})
}

func matches(series scenario.TVDBSeries, query string) bool {
	names := append([]string{series.Name, series.English}, series.Aliases...)
	return slices.ContainsFunc(names, func(n string) bool { return n != "" && strings.Contains(strings.ToLower(n), query) })
}

type updateJSON struct {
	EntityType string `json:"entityType"`
	Method     string `json:"method"`
	MethodInt  int    `json:"methodInt"`
	RecordID   int64  `json:"recordId"`
	TimeStamp  int64  `json:"timeStamp"`
	ExtraInfo  string `json:"extraInfo"`
	UserID     int    `json:"userId"`
}

// updated lists each series change since the since parameter, in order,
// one entry per change.
func (s *server) updated(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	since, err := strconv.ParseInt(q.Get("since"), 10, 64)
	if err != nil {
		failure(w, http.StatusBadRequest, "InvalidSince: since is not a Unix time")
		return
	}
	var all []updateJSON
	if t := q.Get("type"); t == "" || t == "series" {
		s.mu.Lock()
		for _, u := range s.updates {
			if u.at.Unix() < since {
				continue
			}
			e := updateJSON{EntityType: "series", Method: "update", MethodInt: 2, RecordID: u.id, TimeStamp: u.at.Unix()}
			if u.deleted {
				e.Method, e.MethodInt = "delete", 3
			}
			all = append(all, e)
		}
		s.mu.Unlock()
	}
	page := pageParam(q.Get("page"))
	out := []updateJSON{}
	if from := page * updatesPage; from < len(all) {
		out = all[from:min(from+updatesPage, len(all))]
	}
	reply(w, http.StatusOK, envelope{Status: "success", Data: out, Links: pageLinks(r, page, len(all), updatesPage)})
}

type languageJSON struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	NativeName string `json:"nativeName"`
	ShortCode  string `json:"shortCode"`
}

var languages = []languageJSON{
	{"deu", "German", "Deutsch", "de"},
	{"eng", "English", "English", "en"},
	{"fra", "French", "Français", "fr"},
	{"jpn", "Japanese", "日本語", "ja"},
	{"spa", "Spanish", "Español", "es"},
}

func (s *server) languages(w http.ResponseWriter, _ *http.Request) {
	reply(w, http.StatusOK, envelope{Status: "success", Data: languages})
}

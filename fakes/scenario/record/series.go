package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/scenario"
	tmdbapi "github.com/jatassi/slipstream/internal/providers/tmdb"
)

// widths is the width each TVDB image is scaled down to before it's
// stored, since TVDB serves only originals: big enough for the sizes the
// app asks for, small enough to check in.
var widths = map[string]int{"poster": 500, "season": 342, "still": 400}

// orders are the season types a recorded series keeps, Aired first.
var orders = []string{"official", "dvd", "absolute"}

type tvdbEnvelope[T any] struct {
	Data  T `json:"data"`
	Links struct {
		Next *string `json:"next"`
	} `json:"links"`
}

type tvdbSeriesJSON struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Aliases []struct {
		Language string `json:"language"`
		Name     string `json:"name"`
	} `json:"aliases"`
	Status struct {
		Name string `json:"name"`
	} `json:"status"`
	OriginalCountry  string      `json:"originalCountry"`
	OriginalLanguage string      `json:"originalLanguage"`
	OriginalNetwork  *tvdbNamed  `json:"originalNetwork"`
	LatestNetwork    *tvdbNamed  `json:"latestNetwork"`
	Genres           []tvdbNamed `json:"genres"`
	RemoteIDs        []struct {
		ID         string `json:"id"`
		SourceName string `json:"sourceName"`
	} `json:"remoteIds"`
	Seasons     []tvdbSeasonJSON `json:"seasons"`
	SeasonTypes []struct {
		Type string `json:"type"`
	} `json:"seasonTypes"`
	Translations struct {
		Names []struct {
			Language string `json:"language"`
			Name     string `json:"name"`
			IsAlias  bool   `json:"isAlias"`
		} `json:"nameTranslations"`
		Overviews []struct {
			Language string `json:"language"`
			Overview string `json:"overview"`
		} `json:"overviewTranslations"`
	} `json:"translations"`
}

type tvdbSeasonJSON struct {
	Number int    `json:"number"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	Type   struct {
		Type string `json:"type"`
	} `json:"type"`
}

type tvdbNamed struct {
	Name string `json:"name"`
}

type tvdbEpisodeJSON struct {
	ID           int64  `json:"id"`
	SeasonNumber int    `json:"seasonNumber"`
	Number       int    `json:"number"`
	Aired        string `json:"aired"`
	Runtime      *int   `json:"runtime"`
	Name         string `json:"name"`
	Overview     string `json:"overview"`
	Image        string `json:"image"`
}

// login is a TVDB token for the key and PIN.
func (c config) login(ctx context.Context) (string, error) {
	body, err := json.Marshal(struct { //nolint:gosec // G117: the login body carries the key by design
		Key string `json:"apikey"`
		Pin string `json:"pin,omitempty"`
	}{c.tvdbKey, c.tvdbPin})
	if err != nil {
		return "", fmt.Errorf("encode the TVDB login: %w", err)
	}
	at := c.tvdbAPI + "/login"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, at, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("POST %s: %w", at, err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.send(req, at)
	if err != nil {
		return "", err
	}
	var login tvdbEnvelope[struct {
		Token string `json:"token"`
	}]
	if err := json.Unmarshal(res, &login); err != nil || login.Data.Token == "" {
		return "", fmt.Errorf("read %s: no token in %s", at, res)
	}
	return login.Data.Token, nil
}

// show is TVDB's series id, TMDB's record of it and OMDb's ratings for
// it and its episodes, with its images saved in the store.
func (c config) show(ctx context.Context, token string, id int64) (scenario.TVDBSeries, scenario.TMDBSeries, scenario.OMDbTitle, error) {
	var s tvdbEnvelope[tvdbSeriesJSON]
	at := c.tvdbAPI + "/series/" + strconv.FormatInt(id, 10)
	if err := c.getJSON(ctx, at+"/extended", url.Values{"meta": {"translations"}, "short": {"true"}}, token, &s); err != nil {
		return scenario.TVDBSeries{}, scenario.TMDBSeries{}, scenario.OMDbTitle{}, err
	}
	episodes := map[string][]tvdbEpisodeJSON{}
	for _, t := range s.Data.SeasonTypes {
		if slices.Contains(orders, t.Type) {
			var err error
			if episodes[t.Type], err = c.episodes(ctx, token, at+"/episodes/"+t.Type+"/eng"); err != nil {
				return scenario.TVDBSeries{}, scenario.TMDBSeries{}, scenario.OMDbTitle{}, err
			}
		}
	}
	tv, err := tvdbRecord(s.Data, episodes)
	if err != nil {
		return scenario.TVDBSeries{}, scenario.TMDBSeries{}, scenario.OMDbTitle{}, err
	}
	show, err := c.tmdbSeries(ctx, tv)
	if err != nil {
		return scenario.TVDBSeries{}, scenario.TMDBSeries{}, scenario.OMDbTitle{}, err
	}
	files := append(c.tvdbFiles(tv), c.tmdbFiles(show.Images, show.Credits)...)
	for _, p := range show.Creators {
		files = append(files, c.tmdbFile(p.Profile, "profile"))
	}
	if n := show.Network; n != nil {
		files = append(files, c.tmdbFile(n.Logo, "logo"))
	}
	if err := c.download(ctx, files); err != nil {
		return scenario.TVDBSeries{}, scenario.TMDBSeries{}, scenario.OMDbTitle{}, err
	}
	title, err := c.omdbSeries(ctx, tv.IMDb)
	return tv, show, title, err
}

// episodes is every page of the episode list at at, in English.
func (c config) episodes(ctx context.Context, token, at string) ([]tvdbEpisodeJSON, error) {
	var out []tvdbEpisodeJSON
	q := url.Values{"page": {"0"}}
	for {
		var page tvdbEnvelope[struct {
			Episodes []tvdbEpisodeJSON `json:"episodes"`
		}]
		if err := c.getJSON(ctx, at, q, token, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Data.Episodes...)
		if page.Links.Next == nil || *page.Links.Next == "" {
			return out, nil
		}
		next, err := url.Parse(*page.Links.Next)
		if err != nil {
			return nil, fmt.Errorf("read %s: the next page %q isn't a URL", at, *page.Links.Next)
		}
		q = next.Query()
		next.RawQuery = ""
		at = next.String()
	}
}

// tvdbRecord is the Scenario's record of TVDB's series s, its episodes
// in each order, in English.
func tvdbRecord(s tvdbSeriesJSON, episodes map[string][]tvdbEpisodeJSON) (scenario.TVDBSeries, error) {
	out := scenario.TVDBSeries{
		ID: s.ID, Name: s.Name, Status: strings.ToLower(s.Status.Name), OriginalLanguage: s.OriginalLanguage, OriginalCountry: s.OriginalCountry,
		Image: tvdbPath(s.Image), Seasons: seasons(s.Seasons, episodes["official"]),
	}
	if !slices.Contains(scenario.TVDBStatuses, out.Status) {
		return scenario.TVDBSeries{}, fmt.Errorf("TVDB series %d has status %q", s.ID, s.Status.Name)
	}
	out.English, out.Overview = english(s)
	for _, a := range s.Aliases {
		if a.Language == "eng" {
			out.Aliases = append(out.Aliases, a.Name)
		}
	}
	if n := cmp.Or(s.OriginalNetwork, s.LatestNetwork); n != nil {
		out.Network = n.Name
	}
	for _, g := range s.Genres {
		out.Genres = append(out.Genres, g.Name)
	}
	for _, r := range s.RemoteIDs {
		switch r.SourceName {
		case "IMDB":
			out.IMDb = r.ID
		case "TheMovieDB.com":
			out.TMDB, _ = strconv.ParseInt(r.ID, 10, 64)
		}
	}
	if out.IMDb == "" || out.TMDB == 0 {
		return scenario.TVDBSeries{}, fmt.Errorf("TVDB series %d has IMDb id %q and TMDB id %d, and a recording needs both", s.ID, out.IMDb, out.TMDB)
	}
	out.DVD, out.Absolute = renumbered(episodes)
	return out, nil
}

// english is s's English name, when it isn't s's own, and its English
// overview.
func english(s tvdbSeriesJSON) (name, overview string) {
	for _, n := range s.Translations.Names {
		if n.Language == "eng" && !n.IsAlias && n.Name != s.Name {
			name = n.Name
		}
	}
	for _, o := range s.Translations.Overviews {
		if o.Language == "eng" {
			overview = o.Overview
		}
	}
	return name, overview
}

// seasons are the Aired seasons of the episodes aired, each named and
// pictured as TVDB's season record of it.
func seasons(records []tvdbSeasonJSON, aired []tvdbEpisodeJSON) []scenario.TVDBSeason {
	var out []scenario.TVDBSeason
	for _, e := range aired {
		i := slices.IndexFunc(out, func(season scenario.TVDBSeason) bool { return season.Number == e.SeasonNumber })
		if i < 0 {
			i = len(out)
			season := scenario.TVDBSeason{Number: e.SeasonNumber}
			if j := slices.IndexFunc(records, func(r tvdbSeasonJSON) bool { return r.Type.Type == "official" && r.Number == e.SeasonNumber }); j >= 0 {
				season.Name, season.Image = records[j].Name, tvdbPath(records[j].Image)
			}
			out = append(out, season)
		}
		out[i].Episodes = append(out[i].Episodes, scenario.TVDBEpisode{
			ID: e.ID, Number: e.Number, Name: e.Name, Aired: e.Aired, Runtime: deref(e.Runtime), Overview: e.Overview, Image: tvdbPath(e.Image),
		})
	}
	slices.SortStableFunc(out, func(a, b scenario.TVDBSeason) int { return cmp.Compare(a.Number, b.Number) })
	return out
}

// renumbered places each Aired episode in the DVD and Absolute orders,
// specials left out of Absolute. A nil map is an order the series lacks.
func renumbered(episodes map[string][]tvdbEpisodeJSON) (dvd map[int64]scenario.TVDBNumber, absolute map[int64]int) {
	aired := map[int64]bool{}
	for _, e := range episodes["official"] {
		aired[e.ID] = true
	}
	for _, e := range episodes["dvd"] {
		if aired[e.ID] {
			if dvd == nil {
				dvd = map[int64]scenario.TVDBNumber{}
			}
			dvd[e.ID] = scenario.TVDBNumber{Season: e.SeasonNumber, Episode: e.Number}
		}
	}
	for _, e := range episodes["absolute"] {
		if aired[e.ID] && e.SeasonNumber > 0 {
			if absolute == nil {
				absolute = map[int64]int{}
			}
			absolute[e.ID] = e.Number
		}
	}
	return dvd, absolute
}

func deref(n *int) int {
	if n == nil {
		return 0
	}
	return *n
}

// tvdbPath is the path of TVDB artwork URL raw when the store can hold
// it, else empty.
func tvdbPath(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || !scenario.TVDBArtworkFile.MatchString(u.Path) {
		return ""
	}
	return u.Path
}

// tvdbFiles are the posters and stills a TVDB record names, each scaled
// down to its kind's width.
func (c config) tvdbFiles(s scenario.TVDBSeries) []file {
	tvdbFile := func(p, kind string) file { return file{path: p, from: c.tvdbArtwork + p, width: widths[kind]} }
	out := []file{tvdbFile(s.Image, "poster")}
	for _, season := range s.Seasons {
		out = append(out, tvdbFile(season.Image, "season"))
		for _, e := range season.Episodes {
			out = append(out, tvdbFile(e.Image, "still"))
		}
	}
	return out
}

type tmdbSeriesJSON struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	OriginalName     string  `json:"original_name"`
	OriginalLanguage string  `json:"original_language"`
	Overview         string  `json:"overview"`
	FirstAirDate     string  `json:"first_air_date"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int64   `json:"vote_count"`
	CreatedBy        []struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		ProfilePath string `json:"profile_path"`
	} `json:"created_by"`
	Networks []struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		LogoPath string `json:"logo_path"`
	} `json:"networks"`
	Credits        tmdbapi.Credits `json:"credits"`
	ContentRatings struct {
		Results []tmdbRatingJSON `json:"results"`
	} `json:"content_ratings"`
	Images      tmdbapi.Images `json:"images"`
	Videos      tmdbapi.Videos `json:"videos"`
	ExternalIDs struct {
		TVDB *int64 `json:"tvdb_id"`
	} `json:"external_ids"`
}

type tmdbRatingJSON struct {
	Country string `json:"iso_3166_1"`
	Rating  string `json:"rating"`
}

// tmdbSeries is TMDB's record of the series TVDB recorded as tv.
func (c config) tmdbSeries(ctx context.Context, tv scenario.TVDBSeries) (scenario.TMDBSeries, error) {
	var j tmdbSeriesJSON
	q := url.Values{
		"api_key":                {c.tmdbKey},
		"language":               {"en-US"},
		"append_to_response":     {"credits,content_ratings,images,videos,external_ids"},
		"include_image_language": {"en,null"},
	}
	if err := c.getJSON(ctx, c.tmdbAPI+"/tv/"+strconv.FormatInt(tv.TMDB, 10), q, "", &j); err != nil {
		return scenario.TMDBSeries{}, err
	}
	if j.ExternalIDs.TVDB == nil || *j.ExternalIDs.TVDB != tv.ID {
		return scenario.TMDBSeries{}, fmt.Errorf("TVDB series %d names TMDB series %d, which doesn't name it back", tv.ID, tv.TMDB)
	}
	return tmdbSeriesRecord(j), nil
}

// tmdbSeriesRecord is the Scenario's record of TMDB's series j.
func tmdbSeriesRecord(j tmdbSeriesJSON) scenario.TMDBSeries {
	out := scenario.TMDBSeries{
		ID: j.ID, TVDB: *j.ExternalIDs.TVDB, Name: j.Name, OriginalName: j.OriginalName, OriginalLanguage: j.OriginalLanguage,
		Overview: j.Overview, FirstAired: j.FirstAirDate, Credits: scenario.TMDBCredits{Cast: cast(j.Credits.Cast)},
		Trailer: trailer(j.Videos.Results), Images: pickImages(j.Images), VoteAverage: j.VoteAverage, VoteCount: j.VoteCount,
	}
	for _, p := range j.CreatedBy {
		out.Creators = append(out.Creators, scenario.TMDBCast{ID: p.ID, Name: p.Name, Profile: stored(p.ProfilePath)})
	}
	if len(j.Networks) > 0 {
		n := j.Networks[0]
		out.Network = &scenario.TMDBNetwork{ID: n.ID, Name: n.Name, Logo: stored(n.LogoPath)}
	}
	for _, country := range countries {
		i := slices.IndexFunc(j.ContentRatings.Results, func(r tmdbRatingJSON) bool { return r.Country == country && r.Rating != "" })
		if i >= 0 {
			out.Certifications = append(out.Certifications, scenario.TMDBCertification{Country: country, Rating: j.ContentRatings.Results[i].Rating})
		}
	}
	return out
}

type omdbSeasonJSON struct {
	Response string `json:"Response"`
	Episodes []struct {
		Episode    string `json:"Episode"`
		IMDbID     string `json:"imdbID"`
		IMDbRating string `json:"imdbRating"`
	} `json:"Episodes"`
}

// omdbSeries is OMDb's ratings for the series IMDb numbers id, and for
// each episode of each season OMDb lists.
func (c config) omdbSeries(ctx context.Context, id string) (scenario.OMDbTitle, error) {
	out, seasons, err := c.omdb(ctx, id)
	if err != nil {
		return scenario.OMDbTitle{}, err
	}
	out.Type = "series"
	for n := 1; n <= seasons; n++ {
		var s omdbSeasonJSON
		q := url.Values{"i": {id}, "Season": {strconv.Itoa(n)}, "apikey": {c.omdbKey}}
		if err := c.getJSON(ctx, c.omdbAPI, q, "", &s); err != nil {
			return scenario.OMDbTitle{}, err
		}
		if s.Response != "True" {
			continue
		}
		season := scenario.OMDbSeason{Number: n}
		for _, e := range s.Episodes {
			number, err := strconv.Atoi(e.Episode)
			if err != nil || number < 1 || !scenario.IMDbID.MatchString(e.IMDbID) {
				continue
			}
			episode := scenario.OMDbEpisode{Episode: number, IMDbID: e.IMDbID}
			if v, err := strconv.ParseFloat(e.IMDbRating, 64); err == nil {
				episode.Rating = &v
			}
			season.Episodes = append(season.Episodes, episode)
		}
		out.Seasons = append(out.Seasons, season)
	}
	return out, nil
}

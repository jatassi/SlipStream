package tmdb

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/scenario"
)

func (s *server) seriesOf(id int64) (scenario.TMDBSeries, bool) {
	i, ok := slices.BinarySearchFunc(s.catalog.Series, id, func(t scenario.TMDBSeries, id int64) int { return cmp.Compare(t.ID, id) })
	if !ok {
		return scenario.TMDBSeries{}, false
	}
	return s.catalog.Series[i], true
}

// liveSeries is the series TMDB holds as id now, with its revision.
func (s *server) liveSeries(id int64) (scenario.TMDBSeries, int, bool) {
	t, ok := s.seriesOf(id)
	if !ok {
		return scenario.TMDBSeries{}, 0, false
	}
	rev, ok := s.revisionOf(&s.series, id)
	return t, rev, ok
}

func seriesArt(t scenario.TMDBSeries) art { return artOf("tv/"+strconv.FormatInt(t.ID, 10), t.Images) }

type seriesResultJSON struct {
	Adult            bool    `json:"adult"`
	BackdropPath     *string `json:"backdrop_path"`
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	OriginalLanguage string  `json:"original_language"`
	OriginalName     string  `json:"original_name"`
	Overview         string  `json:"overview"`
	PosterPath       *string `json:"poster_path"`
	MediaType        string  `json:"media_type,omitempty"`
	FirstAirDate     string  `json:"first_air_date"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int64   `json:"vote_count"`
}

func seriesResult(t scenario.TMDBSeries, rev int, lang string) seriesResultJSON {
	a := seriesArt(t)
	return seriesResultJSON{
		ID: t.ID, Name: t.Name, OriginalLanguage: seriesLanguage(t), OriginalName: seriesName(t), Overview: overview(t.Overview, rev),
		PosterPath: a.pathOf("poster", lang), BackdropPath: a.pathOf("backdrop", ""), FirstAirDate: t.FirstAired,
		VoteAverage: t.VoteAverage, VoteCount: t.VoteCount + int64(rev),
	}
}

func seriesName(t scenario.TMDBSeries) string { return cmp.Or(t.OriginalName, t.Name) }

func seriesLanguage(t scenario.TMDBSeries) string { return cmp.Or(t.OriginalLanguage, "en") }

type seriesPageJSON struct {
	Page         int                `json:"page"`
	Results      []seriesResultJSON `json:"results"`
	TotalPages   int                `json:"total_pages"`
	TotalResults int                `json:"total_results"`
}

func (s *server) searchSeries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query, lang := strings.ToLower(q.Get("query")), q.Get("language")
	var found []seriesResultJSON
	for _, t := range s.catalog.Series {
		rev, ok := s.revisionOf(&s.series, t.ID)
		names := []string{strings.ToLower(t.Name), strings.ToLower(t.OriginalName)}
		if !ok || query == "" || !slices.ContainsFunc(names, func(n string) bool { return n != "" && strings.Contains(n, query) }) {
			continue
		}
		found = append(found, seriesResult(t, rev, lang))
	}
	page := seriesPageJSON{Page: pageParam(q.Get("page")), Results: []seriesResultJSON{}, TotalResults: len(found)}
	page.TotalPages = (len(found) + pageSize - 1) / pageSize
	if from := (page.Page - 1) * pageSize; from < len(found) {
		page.Results = found[from:min(from+pageSize, len(found))]
	}
	reply(w, http.StatusOK, page)
}

type creatorJSON struct {
	ID           int64   `json:"id"`
	CreditID     string  `json:"credit_id"`
	Name         string  `json:"name"`
	OriginalName string  `json:"original_name"`
	ProfilePath  *string `json:"profile_path"`
}

type networkJSON struct {
	ID            int64   `json:"id"`
	LogoPath      *string `json:"logo_path"`
	Name          string  `json:"name"`
	OriginCountry string  `json:"origin_country"`
}

type contentRatingsJSON struct {
	Results []contentRatingJSON `json:"results"`
}

type contentRatingJSON struct {
	Descriptors []string `json:"descriptors"`
	Country     string   `json:"iso_3166_1"`
	Rating      string   `json:"rating"`
}

// seriesIDsJSON is a series' external ids. ID is set on its own route
// and left out of the appended block, as TMDB does.
type seriesIDsJSON struct {
	ID     int64   `json:"id,omitempty"`
	IMDbID *string `json:"imdb_id"`
	TVDBID *int64  `json:"tvdb_id"`
}

type seriesJSON struct {
	seriesResultJSON
	CreatedBy      []creatorJSON       `json:"created_by"`
	Genres         []genreJSON         `json:"genres"`
	Networks       []networkJSON       `json:"networks"`
	Credits        *creditsJSON        `json:"credits,omitempty"`
	ContentRatings *contentRatingsJSON `json:"content_ratings,omitempty"`
	Images         *imagesJSON         `json:"images,omitempty"`
	Videos         *videosJSON         `json:"videos,omitempty"`
	ExternalIDs    *seriesIDsJSON      `json:"external_ids,omitempty"`
}

// tv is what TMDB adds to a series, never its seasons or episodes, which
// SlipStream reads from TVDB.
func (s *server) tv(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	t, rev, ok := s.liveSeries(id)
	if err != nil || !ok {
		notFound(w)
		return
	}
	q := r.URL.Query()
	lang := q.Get("language")
	out := seriesJSON{seriesResultJSON: seriesResult(t, rev, lang), CreatedBy: []creatorJSON{}, Genres: []genreJSON{}, Networks: []networkJSON{}}
	for _, c := range t.Creators {
		out.CreatedBy = append(out.CreatedBy, creatorJSON{
			ID: c.ID, CreditID: fmt.Sprintf("%024x", hash("creator", strconv.FormatInt(t.ID, 10), strconv.FormatInt(c.ID, 10))),
			Name: c.Name, OriginalName: c.Name, ProfilePath: profile(c.ID, c.Profile),
		})
	}
	if n := t.Network; n != nil {
		logo := file("network", strconv.FormatInt(n.ID, 10), ".png")
		if n.Logo != "" {
			logo = &n.Logo
		}
		out.Networks = append(out.Networks, networkJSON{ID: n.ID, Name: n.Name, LogoPath: logo, OriginCountry: "US"})
	}
	imageLanguages := languagesParam(q.Get("include_image_language"), lang, true)
	videoLanguages := languagesParam(q.Get("include_video_language"), lang, false)
	for part := range strings.SplitSeq(q.Get("append_to_response"), ",") {
		switch part {
		case "credits":
			out.Credits = credits(t.Credits)
		case "content_ratings":
			out.ContentRatings = &contentRatingsJSON{Results: []contentRatingJSON{}}
			for _, c := range t.Certifications {
				out.ContentRatings.Results = append(out.ContentRatings.Results, contentRatingJSON{Descriptors: []string{}, Country: c.Country, Rating: c.Rating})
			}
		case "images":
			out.Images = seriesArt(t).set(imageLanguages)
		case "videos":
			out.Videos = videos(t.Trailer, t.FirstAired, videoLanguages)
		case "external_ids":
			ids := s.idsOf(t)
			out.ExternalIDs = &ids
		}
	}
	reply(w, http.StatusOK, out)
}

// idsOf names the series' TVDB id, and the IMDb id fake-tvdb lists for
// it; a series only TMDB lists names neither.
func (s *server) idsOf(t scenario.TMDBSeries) seriesIDsJSON {
	var ids seriesIDsJSON
	if t.TVDB != 0 {
		ids.TVDBID = &t.TVDB
	}
	if imdb := s.imdb[t.TVDB]; imdb != "" {
		ids.IMDbID = &imdb
	}
	return ids
}

func (s *server) externalIDs(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	t, _, ok := s.liveSeries(id)
	if err != nil || !ok {
		notFound(w)
		return
	}
	ids := s.idsOf(t)
	ids.ID = t.ID
	reply(w, http.StatusOK, ids)
}

type findJSON struct {
	MovieResults     []struct{}         `json:"movie_results"`
	PersonResults    []struct{}         `json:"person_results"`
	TVResults        []seriesResultJSON `json:"tv_results"`
	TVEpisodeResults []struct{}         `json:"tv_episode_results"`
	TVSeasonResults  []struct{}         `json:"tv_season_results"`
}

// find answers the series whose external ids name the TVDB id; any other
// external source finds nothing here.
func (s *server) find(w http.ResponseWriter, r *http.Request) {
	out := findJSON{MovieResults: []struct{}{}, PersonResults: []struct{}{}, TVResults: []seriesResultJSON{}, TVEpisodeResults: []struct{}{}, TVSeasonResults: []struct{}{}}
	q := r.URL.Query()
	tvdb, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err == nil && q.Get("external_source") == "tvdb_id" {
		for _, t := range s.catalog.Series {
			if rev, ok := s.revisionOf(&s.series, t.ID); ok && t.TVDB == tvdb {
				found := seriesResult(t, rev, q.Get("language"))
				found.MediaType = "tv"
				out.TVResults = append(out.TVResults, found)
			}
		}
	}
	reply(w, http.StatusOK, out)
}

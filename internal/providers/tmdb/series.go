package tmdb

import (
	"cmp"
	"context"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/metadata"
)

// SeriesFound is one series in a search. FirstAired is YYYY-MM-DD, or empty.
type SeriesFound struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	OriginalName string `json:"original_name"`
	Overview     string `json:"overview"`
	FirstAired   string `json:"first_air_date"`
	PosterPath   string `json:"poster_path"`
}

// SearchSeries reads the first page of a series search in ask's language.
// A year above 0 keeps the series first aired that year.
func (c *Client) SearchSeries(ctx context.Context, ask metadata.Ask, query string, year int) ([]SeriesFound, error) {
	q := url.Values{"query": {query}, "include_adult": {"false"}}
	if year > 0 {
		q.Set("first_air_date_year", strconv.Itoa(year))
	}
	language(q, ask)
	var p struct {
		Results []SeriesFound `json:"results"`
	}
	err := c.get(ctx, ask, metadata.Request{URL: c.url(q, "search", "tv"), Cache: true}, &p)
	return p.Results, err
}

// SeriesIDs are the ids other services know a series by; zero and empty
// when TMDB names none.
type SeriesIDs struct {
	TVDB int64
	IMDb string
}

type seriesIDsJSON struct {
	TVDB *int64  `json:"tvdb_id"`
	IMDb *string `json:"imdb_id"`
}

func (j seriesIDsJSON) parse() SeriesIDs {
	var ids SeriesIDs
	if j.TVDB != nil {
		ids.TVDB = *j.TVDB
	}
	if j.IMDb != nil {
		ids.IMDb = *j.IMDb
	}
	return ids
}

// SeriesExternalIDs reads the ids of the series TMDB numbers id.
func (c *Client) SeriesExternalIDs(ctx context.Context, ask metadata.Ask, id int64) (SeriesIDs, error) {
	var ids seriesIDsJSON
	err := c.get(ctx, ask, metadata.Request{URL: c.url(nil, "tv", strconv.FormatInt(id, 10), "external_ids"), Lookup: true, Cache: true}, &ids)
	return ids.parse(), err
}

// FindTVDB is the TMDB id of the series TVDB numbers tvdb, and false
// when TMDB lists none.
func (c *Client) FindTVDB(ctx context.Context, ask metadata.Ask, tvdb int64) (id int64, ok bool, err error) {
	var found struct {
		TV []struct {
			ID int64 `json:"id"`
		} `json:"tv_results"`
	}
	q := url.Values{"external_source": {"tvdb_id"}}
	if err := c.get(ctx, ask, metadata.Request{URL: c.url(q, "find", strconv.FormatInt(tvdb, 10)), Cache: true}, &found); err != nil {
		return 0, false, err
	}
	if len(found.TV) == 0 {
		return 0, false, nil
	}
	return found.TV[0].ID, true, nil
}

// Series is what TMDB adds to a series TVDB structures: its people, ratings
// by region, trailer and artwork. TMDB's seasons and episodes are never
// read. Each image is a file path, empty when TMDB has none.
type Series struct {
	ID int64
	SeriesIDs
	// Cast is in billing order.
	Cast     []Person
	Creators []Person
	// Certifications maps an ISO 3166-1 region to its content rating.
	Certifications map[string]string
	Trailer        *Video
	NetworkLogo    string
	Poster         string
	Backdrop       string
	// TitleTreatment is the series' title in its own lettering, which
	// TMDB calls a logo.
	TitleTreatment string
	VoteAverage    float64
	VoteCount      int64
}

// Person is one name a series credits. Role is a cast member's
// character, empty for a creator. Photo is a file path, or empty.
type Person struct {
	ID    int64
	Name  string
	Role  string
	Photo string
}

type seriesJSON struct {
	ID           int64         `json:"id"`
	PosterPath   string        `json:"poster_path"`
	BackdropPath string        `json:"backdrop_path"`
	VoteAverage  float64       `json:"vote_average"`
	VoteCount    int64         `json:"vote_count"`
	ExternalIDs  seriesIDsJSON `json:"external_ids"`
	CreatedBy    []struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		ProfilePath string `json:"profile_path"`
	} `json:"created_by"`
	Networks       []network `json:"networks"`
	Credits        Credits   `json:"credits"`
	ContentRatings struct {
		Results []struct {
			Country string `json:"iso_3166_1"`
			Rating  string `json:"rating"`
		} `json:"results"`
	} `json:"content_ratings"`
	Images Images `json:"images"`
	Videos Videos `json:"videos"`
}

type network struct {
	LogoPath string `json:"logo_path"`
}

const seriesAppended = "credits,content_ratings,images,videos,external_ids"

// Series reads what TMDB adds to the series it numbers id in one call, images
// and the trailer in ask's language, then English, then any.
func (c *Client) Series(ctx context.Context, ask metadata.Ask, id int64) (Series, error) {
	q := url.Values{"append_to_response": {seriesAppended}}
	language(q, ask)
	lang := languages(ask)
	q.Set("include_image_language", strings.Join(append(lang, "null"), ","))
	q.Set("include_video_language", strings.Join(lang, ","))
	var j seriesJSON
	if err := c.get(ctx, ask, metadata.Request{URL: c.url(q, "tv", strconv.FormatInt(id, 10)), Lookup: true, Cache: true}, &j); err != nil {
		return Series{}, err
	}
	return j.parse(ask.Language), nil
}

func (j seriesJSON) parse(language string) Series {
	rank := languageRank(language)
	textless := func(l string) int {
		if l == "" {
			return 0
		}
		return 1 + rank(l)
	}
	tv := Series{
		ID: j.ID, SeriesIDs: j.ExternalIDs.parse(), Certifications: map[string]string{},
		Trailer:        trailerOf(j.Videos.Results, rank),
		Poster:         cmp.Or(best(j.Images.Posters, rank), j.PosterPath),
		Backdrop:       cmp.Or(best(j.Images.Backdrops, textless), j.BackdropPath),
		TitleTreatment: best(j.Images.Logos, rank),
		VoteAverage:    j.VoteAverage, VoteCount: j.VoteCount,
	}
	cast := slices.SortedStableFunc(slices.Values(j.Credits.Cast), func(a, b Cast) int { return cmp.Compare(a.Order, b.Order) })
	for _, p := range cast {
		tv.Cast = append(tv.Cast, Person{ID: p.ID, Name: p.Name, Role: p.Character, Photo: p.ProfilePath})
	}
	for _, p := range j.CreatedBy {
		tv.Creators = append(tv.Creators, Person{ID: p.ID, Name: p.Name, Photo: p.ProfilePath})
	}
	for _, r := range j.ContentRatings.Results {
		if r.Rating != "" {
			tv.Certifications[r.Country] = r.Rating
		}
	}
	if i := slices.IndexFunc(j.Networks, func(n network) bool { return n.LogoPath != "" }); i >= 0 {
		tv.NetworkLogo = j.Networks[i].LogoPath
	}
	return tv
}

// trailerOf is the official trailer first, in the best ranked language,
// the earliest published first; nil when TMDB links none.
func trailerOf(vs []Video, rank func(string) int) *Video {
	unofficial := func(v Video) int {
		if v.Official {
			return 0
		}
		return 1
	}
	var first *Video
	for i, v := range vs {
		if v.Type != "Trailer" || v.Site == "" || v.Key == "" {
			continue
		}
		if first == nil || cmp.Or(cmp.Compare(unofficial(v), unofficial(*first)), cmp.Compare(rank(v.Language), rank(first.Language)),
			strings.Compare(v.PublishedAt, first.PublishedAt)) < 0 {
			first = &vs[i]
		}
	}
	return first
}

// languageRank orders TMDB's two-letter languages: the metadata
// language's, then English, then any other or none.
func languageRank(language string) func(string) int {
	iso, _, _ := strings.Cut(language, "-")
	order := []string{iso, "en"}
	return func(l string) int {
		if i := slices.Index(order, l); i >= 0 && l != "" {
			return i
		}
		return len(order)
	}
}

// best is the file of the best ranked image, the highest voted next.
func best(ims []Image, rank func(string) int) string {
	if len(ims) == 0 {
		return ""
	}
	return slices.MinFunc(ims, func(a, b Image) int {
		return cmp.Or(cmp.Compare(rank(a.Language), rank(b.Language)), cmp.Compare(b.VoteAverage, a.VoteAverage), cmp.Compare(b.VoteCount, a.VoteCount))
	}).FilePath
}

// SeriesChanges reads page (from 1) of the series changed from since to
// until, on TMDB's list for series, which is apart from its movies'.
func (c *Client) SeriesChanges(ctx context.Context, ask metadata.Ask, since, until time.Time, page int) (ChangesPage, error) {
	return c.changes(ctx, ask, "tv", since, until, page)
}

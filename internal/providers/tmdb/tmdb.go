// Package tmdb is the TMDB v3 client every Content Type that reads TMDB
// shares, so its hosts are core endpoints. Every call leaves through the
// Node's metadata Line for TMDB, which holds the key, the rate limit and
// the cache.
package tmdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

const ID metadata.ProviderID = "tmdb"

type Endpoints struct {
	API endpoint.URL `yaml:"api"`
	// Images is the image host. SlipStream never asks TMDB's configuration
	// for it.
	Images endpoint.URL `yaml:"images"`
	// KeyPage is where an Admin gets a key, opened in the browser only.
	KeyPage endpoint.URL `yaml:"key_page"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		API:     endpoint.MustParseURL("https://api.themoviedb.org/3"),
		Images:  endpoint.MustParseURL("https://image.tmdb.org/t/p"),
		KeyPage: endpoint.MustParseURL("https://www.themoviedb.org/settings/api"),
	}
}

// Timings are TMDB's protocol facts: one request every Interval per
// holder, up to Burst at once.
type Timings struct {
	Interval time.Duration `yaml:"interval"`
	Burst    int           `yaml:"burst"`
}

func DefaultTimings() Timings {
	return Timings{Interval: 50 * time.Millisecond, Burst: 20}
}

type Client struct {
	line      *metadata.Line
	endpoints Endpoints
	timings   Timings
}

func New(lines *metadata.Lines, e Endpoints, t Timings) *Client {
	c := &Client{endpoints: e, timings: t}
	c.line = lines.Line(c.Provider())
	return c
}

// Provider is TMDB as the call layer knows it: the key is the api_key
// query parameter, and Classify reads TMDB's error bodies. Its
// attribution is the credit TMDB's terms require, with the logo the web
// app bundles as "tmdb".
func (c *Client) Provider() metadata.Provider {
	return metadata.Provider{
		ID:          ID,
		Label:       "TMDB",
		Rate:        rate.Every(c.timings.Interval),
		Burst:       c.timings.Burst,
		Key:         func(q url.Values, key string) { q.Set("api_key", key) },
		Classify:    classify,
		KeyPage:     c.endpoints.KeyPage.String(),
		Attribution: &metadata.Attribution{Text: "This product uses the TMDB API but is not endorsed or certified by TMDB.", Logo: "tmdb"},
	}
}

// notFoundCode is the status_code of TMDB's body for a record it doesn't
// have.
const notFoundCode = 34

type statusBody struct {
	Code    int    `json:"status_code"`
	Message string `json:"status_message"`
}

// classify names TMDB's 401 in its own words and calls a 404 without
// TMDB's not-found body unreachable: it came from something in front of
// TMDB, not from TMDB.
func classify(status int, _ http.Header, body []byte, _ time.Time) (metadata.Outcome, bool) {
	var b statusBody
	_ = json.Unmarshal(body, &b)
	switch {
	case status == http.StatusUnauthorized:
		words := "refused the key"
		if b.Message != "" {
			words += ": " + b.Message
		}
		return metadata.Outcome{Kind: metadata.Refused, Words: words}, true
	case status == http.StatusNotFound && b.Code != notFoundCode:
		return metadata.Outcome{Kind: metadata.Unreachable, Words: "responded with 404 Not Found but no not-found body"}, true
	}
	return metadata.Outcome{}, false
}

// MovieResult is one movie in a search. ReleaseDate is the primary
// release date, YYYY-MM-DD, or empty.
type MovieResult struct {
	ID               int64  `json:"id"`
	Title            string `json:"title"`
	OriginalTitle    string `json:"original_title"`
	OriginalLanguage string `json:"original_language"`
	Overview         string `json:"overview"`
	ReleaseDate      string `json:"release_date"`
	PosterPath       string `json:"poster_path"`
}

// Page is one page of a search, 20 results a page.
type Page struct {
	Page         int           `json:"page"`
	TotalPages   int           `json:"total_pages"`
	TotalResults int           `json:"total_results"`
	Results      []MovieResult `json:"results"`
}

// SearchMovies reads page (from 1) of a search in ask's language, only
// movies whose primary release falls in year when year > 0.
func (c *Client) SearchMovies(ctx context.Context, ask metadata.Ask, query string, year, page int) (Page, error) {
	q := url.Values{"query": {query}, "page": {strconv.Itoa(page)}, "include_adult": {"false"}}
	language(q, ask)
	if year > 0 {
		q.Set("primary_release_year", strconv.Itoa(year))
	}
	var p Page
	err := c.get(ctx, ask, metadata.Request{URL: c.url(q, "search", "movie"), Cache: true}, &p)
	return p, err
}

// appended is every block a movie's one call carries.
const appended = "external_ids,release_dates,alternative_titles,credits,videos,images,translations"

// Movie reads the whole record of the movie TMDB numbers id in one call,
// texts in ask's language, images and videos in that language and English
// (images with no text too).
func (c *Client) Movie(ctx context.Context, ask metadata.Ask, id int64) (Movie, error) {
	q := url.Values{"append_to_response": {appended}}
	language(q, ask)
	lang := languages(ask)
	q.Set("include_image_language", strings.Join(append(lang, "null"), ","))
	q.Set("include_video_language", strings.Join(lang, ","))
	var m Movie
	err := c.get(ctx, ask, metadata.Request{URL: c.url(q, "movie", strconv.FormatInt(id, 10)), Lookup: true, Cache: true}, &m)
	return m, err
}

// ChangesWindow is how far back TMDB lists changes.
const ChangesWindow = 14 * 24 * time.Hour

// ChangesPage is one page of the movies TMDB changed in a window, 100 ids
// a page.
type ChangesPage struct {
	IDs        []int64
	Page       int
	TotalPages int
}

// MovieChanges reads page (from 1) of the movies changed from since to
// until, by day. A window longer than ChangesWindow is
// metadata.ErrNoChanges and sends nothing.
func (c *Client) MovieChanges(ctx context.Context, ask metadata.Ask, since, until time.Time, page int) (ChangesPage, error) {
	if until.Sub(since) > ChangesWindow {
		return ChangesPage{}, fmt.Errorf("TMDB lists %s of changes, asked from %s: %w", ChangesWindow, since.UTC().Format(time.DateOnly), metadata.ErrNoChanges)
	}
	q := url.Values{
		"start_date": {since.UTC().Format(time.DateOnly)},
		"end_date":   {until.UTC().Format(time.DateOnly)},
		"page":       {strconv.Itoa(page)},
	}
	var p struct {
		Page       int `json:"page"`
		TotalPages int `json:"total_pages"`
		Results    []struct {
			ID int64 `json:"id"`
		} `json:"results"`
	}
	if err := c.get(ctx, ask, metadata.Request{URL: c.url(q, "movie", "changes")}, &p); err != nil {
		return ChangesPage{}, err
	}
	out := ChangesPage{Page: p.Page, TotalPages: p.TotalPages, IDs: make([]int64, len(p.Results))}
	for i, r := range p.Results {
		out.IDs[i] = r.ID
	}
	return out, nil
}

// Test is the cheap call that says whether TMDB answers with ask's key.
func (c *Client) Test(ctx context.Context, ask metadata.Ask) error {
	var cfg json.RawMessage
	return c.get(ctx, ask, metadata.Request{URL: c.url(nil, "configuration")}, &cfg)
}

func (c *Client) get(ctx context.Context, ask metadata.Ask, req metadata.Request, v any) error {
	if err := c.line.Get(ctx, ask, req, v); err != nil {
		return fmt.Errorf("GET %s: %w", strings.TrimPrefix(req.URL.Path, c.endpoints.API.JoinPath().Path), err)
	}
	return nil
}

func (c *Client) url(q url.Values, elem ...string) *url.URL {
	u := c.endpoints.API.JoinPath(elem...)
	u.RawQuery = q.Encode()
	return u
}

func language(q url.Values, ask metadata.Ask) {
	if ask.Language != "" {
		q.Set("language", ask.Language)
	}
}

// languages is ask's language and English, as TMDB's two-letter codes.
func languages(ask metadata.Ask) []string {
	lang, _, _ := strings.Cut(ask.Language, "-")
	if lang == "" || lang == "en" {
		return []string{"en"}
	}
	return []string{lang, "en"}
}

// Package tvdb is SlipStream's client for TVDB API v4, written by hand
// over TVDB's published OpenAPI document. It parses every answer into the
// types below; no wire type leaves it. Every call leaves through the
// Node's metadata Line for TVDB, which logs in with the key and a PIN when
// one is set, and holds the rate limit and the cache.
package tvdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

const ID metadata.ProviderID = "tvdb"

type Endpoints struct {
	API endpoint.URL `yaml:"api"`
	// Artwork is the image host TVDB's artwork paths live on.
	Artwork endpoint.URL `yaml:"artwork"`
	// KeyPage is where an Admin gets a key, opened in the browser only.
	KeyPage endpoint.URL `yaml:"key_page"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		API:     endpoint.MustParseURL("https://api4.thetvdb.com/v4"),
		Artwork: endpoint.MustParseURL("https://artworks.thetvdb.com"),
		KeyPage: endpoint.MustParseURL("https://thetvdb.com/api-information"),
	}
}

// Timings are TVDB's protocol facts: one request every Interval per
// holder, up to Burst at once.
type Timings struct {
	Interval time.Duration `yaml:"interval"`
	Burst    int           `yaml:"burst"`
}

func DefaultTimings() Timings {
	return Timings{Interval: 100 * time.Millisecond, Burst: 10}
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

// Provider is TVDB as the call layer knows it: a login trades the key, and
// the PIN when one is set, for a bearer token.
func (c *Client) Provider() metadata.Provider {
	return metadata.Provider{
		ID:      ID,
		Label:   "TVDB",
		Rate:    rate.Every(c.timings.Interval),
		Burst:   c.timings.Burst,
		KeyPage: c.endpoints.KeyPage.String(),
		Auth: &metadata.Auth{
			Login:   c.login,
			Token:   token,
			Refused: refused,
			Pin:     true,
		},
	}
}

func (c *Client) login(key, pin string) (u *url.URL, body []byte) {
	body, err := json.Marshal(struct { //nolint:gosec // the login body carries the key by design
		APIKey string `json:"apikey"`
		Pin    string `json:"pin,omitempty"`
	}{key, pin})
	if err != nil {
		panic(err)
	}
	return c.endpoints.API.JoinPath("login"), body
}

// Test logs in and reads /languages, the cheapest authenticated call.
func (c *Client) Test(ctx context.Context, ask metadata.Ask) error {
	var languages envelope[[]struct {
		ID string `json:"id"`
	}]
	return c.get(ctx, ask, metadata.Request{URL: c.url(nil, "languages")}, &languages)
}

type (
	SeriesID  int64
	SeasonID  int64
	EpisodeID int64
)

// Order is one of the orders TVDB lists a series' episodes in.
type Order string

const (
	OrderAired    Order = "default"
	OrderDVD      Order = "dvd"
	OrderAbsolute Order = "absolute"
)

var orders = []Order{OrderAired, OrderDVD, OrderAbsolute}

// seasonType is the order's name in TVDB's paths and season types, which call
// the aired order "official".
func (o Order) seasonType() string {
	if o == OrderAired {
		return "official"
	}
	return string(o)
}

func orderOf(seasonType string) (Order, bool) {
	i := slices.IndexFunc(orders, func(o Order) bool { return o.seasonType() == seasonType })
	if i < 0 {
		return "", false
	}
	return orders[i], true
}

type Status string

const (
	Continuing Status = "continuing"
	Ended      Status = "ended"
	Upcoming   Status = "upcoming"
)

// Found is one search result.
type Found struct {
	ID SeriesID
	// Name is in the Ask's language when TVDB has it, else English.
	Name     string
	Original string
	Year     int
	Network  string
	Overview string
	// Poster is an artwork path.
	Poster  string
	Aliases []string
}

const searchLimit = 25

// Search is /search?type=series&query=, names and overviews in the Ask's
// language with English as fallback, at most 25 results.
func (c *Client) Search(ctx context.Context, ask metadata.Ask, query string) ([]Found, error) {
	q := url.Values{"query": {query}, "type": {"series"}, "limit": {strconv.Itoa(searchLimit)}}
	var page envelope[[]searchJSON]
	if err := c.get(ctx, ask, metadata.Request{URL: c.url(q, "search"), Cache: true}, &page); err != nil {
		return nil, err
	}
	lang := languageOf(ask.Language)
	found := make([]Found, 0, len(page.Data))
	for _, r := range page.Data {
		id, err := strconv.ParseInt(r.TVDBID, 10, 64)
		if err != nil {
			return nil, invalid("search", fmt.Errorf("tvdb_id %q is not a number", r.TVDBID))
		}
		year, _ := strconv.Atoi(r.Year)
		found = append(found, Found{
			ID: SeriesID(id), Name: translated(r.Translations, lang, r.Name), Original: r.Name, Year: year, Network: r.Network,
			Overview: translated(r.Overviews, lang, r.Overview), Poster: artworkPath(r.ImageURL), Aliases: r.Aliases,
		})
	}
	return found, nil
}

// translated is texts' entry in lang, else English, else own.
func translated(texts map[string]string, lang, own string) string {
	if t := texts[lang]; t != "" {
		return t
	}
	if t := texts[english]; t != "" {
		return t
	}
	return own
}

type Series struct {
	ID SeriesID
	// Name is in the Ask's language, else English.
	Name string
	// English is TVDB's English translation, else the series' own name,
	// for search terms and naming.
	English string
	// Original is the name in the series' original language.
	Original string
	// OriginalLanguage and Country are TVDB's three-letter codes.
	OriginalLanguage string
	Country          string
	Aliases          []string
	Status           Status
	Network          string
	// FirstAired, LastAired and NextAired are YYYY-MM-DD or empty.
	FirstAired     string
	LastAired      string
	NextAired      string
	AverageRuntime time.Duration
	Genres         []string
	// Overview is in the Ask's language, else English.
	Overview string
	// Seasons are every season in every order.
	Seasons []Season
	// Orders are the orders TVDB has for this series.
	Orders []Order
	IMDb   string
	// TMDB is 0 when TVDB's remote ids name none.
	TMDB int64
	// Poster is an artwork path, empty for none.
	Poster string
}

type Season struct {
	ID     SeasonID
	Number int
	Order  Order
	Name   string
	// Poster is an artwork path, empty for none.
	Poster string
}

// Series is /series/{id}/extended?meta=translations&short=true;
// works.ErrNotFound on 404.
func (c *Client) Series(ctx context.Context, ask metadata.Ask, id SeriesID) (Series, error) {
	q := url.Values{"meta": {"translations"}, "short": {"true"}}
	var rec envelope[seriesJSON]
	if err := c.get(ctx, ask, metadata.Request{URL: c.url(q, "series", strconv.FormatInt(int64(id), 10), "extended"), Lookup: true, Cache: true}, &rec); err != nil {
		return Series{}, err
	}
	return rec.Data.series(languageOf(ask.Language))
}

type Episode struct {
	ID     EpisodeID
	Season int
	Number int
	// Absolute is 0 when TVDB has none.
	Absolute int
	// Aired is YYYY-MM-DD or empty.
	Aired    string
	Runtime  time.Duration
	Name     string
	Overview string
	// Still is an artwork path, empty for none.
	Still string
}

// Episodes is /series/{id}/episodes/{order}/{language}, every page. A page
// that fails fails the call, so a partial answer is never returned.
func (c *Client) Episodes(ctx context.Context, ask metadata.Ask, id SeriesID, o Order) ([]Episode, error) {
	var out []Episode
	for page := 0; ; page++ {
		q := url.Values{"page": {strconv.Itoa(page)}}
		u := c.url(q, "series", strconv.FormatInt(int64(id), 10), "episodes", o.seasonType(), languageOf(ask.Language))
		var p envelope[struct {
			Episodes []episodeJSON `json:"episodes"`
		}]
		if err := c.get(ctx, ask, metadata.Request{URL: u, Lookup: true, Cache: true}, &p); err != nil {
			return nil, err
		}
		for _, e := range p.Data.Episodes {
			out = append(out, e.episode())
		}
		if p.Links == nil || p.Links.Next == nil || *p.Links.Next == "" || len(p.Data.Episodes) == 0 {
			return out, nil
		}
	}
}

type Updates struct {
	Series []SeriesID
	// Pages is how many pages the list holds.
	Pages int
}

// Updates is page (from 1) of /updates?since=&type=series: every series
// TVDB created, changed or deleted since then, once.
func (c *Client) Updates(ctx context.Context, ask metadata.Ask, since time.Time, page int) (Updates, error) {
	q := url.Values{"since": {strconv.FormatInt(since.Unix(), 10)}, "type": {"series"}, "page": {strconv.Itoa(page - 1)}}
	var p envelope[[]updateJSON]
	if err := c.get(ctx, ask, metadata.Request{URL: c.url(q, "updates")}, &p); err != nil {
		return Updates{}, err
	}
	var out Updates
	if l := p.Links; l != nil && l.PageSize > 0 {
		out.Pages = (l.TotalItems + l.PageSize - 1) / l.PageSize
	}
	for _, u := range p.Data {
		if id := SeriesID(u.RecordID); u.EntityType == "series" && !slices.Contains(out.Series, id) {
			out.Series = append(out.Series, id)
		}
	}
	return out, nil
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

const english = "eng"

// languageOf maps a BCP 47 tag such as fr-CA to TVDB's three-letter code
// such as fra, eng for a tag with no such code.
func languageOf(tag string) string {
	t, err := language.Parse(tag)
	if err != nil {
		return english
	}
	base, confidence := t.Base()
	if confidence == language.No {
		return english
	}
	code := base.ISO3()
	if len(code) != 3 {
		return english
	}
	return code
}

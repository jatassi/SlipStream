// Package tvdb is SlipStream's client for TVDB API v4, written by hand
// over TVDB's published OpenAPI document. It parses every answer into the
// types below; no wire type leaves it. Every call leaves through the
// Node's metadata Line for TVDB, which logs in with the key and a PIN when
// one is set, and holds the rate limit and the cache.
package tvdb

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"time"

	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

const ID metadata.ProviderID = "tvdb"

const label = "TVDB"

// ArtworkID is the source id of TVDB's image host, apart from the API's
// Line.
const ArtworkID artwork.SourceID = "tvdb-artwork"

var errNotImplemented = errors.New("not implemented")

type Endpoints struct {
	API     endpoint.URL `yaml:"api"`
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

// Timings are TVDB's protocol facts: Rate requests a second per holder, up
// to Burst at once.
type Timings struct {
	Rate  float64 `yaml:"rate"`
	Burst int     `yaml:"burst"`
}

func DefaultTimings() Timings {
	return Timings{Rate: 10, Burst: 5}
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
		Label:   label,
		Rate:    rate.Limit(c.timings.Rate),
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

func token([]byte) (string, time.Time, error) {
	return "", time.Time{}, errNotImplemented
}

func refused(int, []byte) metadata.Credential {
	return metadata.CredentialKey
}

// Test logs in and reads /languages, the cheapest authenticated call.
func (c *Client) Test(context.Context, metadata.Ask) error {
	return errNotImplemented
}

// Artwork is the image host TVDB's artwork paths live on, with originals
// and _t thumbnails.
func (c *Client) Artwork() artwork.Source {
	return artSource{c}
}

type artSource struct{ c *Client }

func (artSource) ID() artwork.SourceID { return ArtworkID }

func (artSource) URL(artwork.Shape, string, artwork.Size) (*url.URL, error) {
	return nil, errNotImplemented
}

func (a artSource) Rate() (perSecond float64, burst int) {
	return a.c.timings.Rate, a.c.timings.Burst
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

// Search is /search?type=series&query=, in the Ask's language with English
// as fallback, at most 25 results.
func (c *Client) Search(context.Context, metadata.Ask, string) ([]Found, error) {
	return nil, errNotImplemented
}

type Series struct {
	ID SeriesID
	// Name is in the Ask's language, else English.
	Name string
	// English is TVDB's English translation, for search terms and naming.
	English          string
	Original         string
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
	Overview       string
	// Seasons are every season in every order.
	Seasons []Season
	// Orders are the orders TVDB has for this series.
	Orders []Order
	IMDb   string
	// TMDB is 0 when TVDB's remote ids name none.
	TMDB    int64
	Artwork []Art
}

type Season struct {
	ID       SeasonID
	Number   int
	Order    Order
	Name     string
	Overview string
	Poster   string
}

type Art struct {
	// Kind is poster, background or clearlogo.
	Kind string
	Path string
	Lang string
}

// Series is /series/{id}/extended?meta=translations&short=true;
// works.ErrNotFound on 404.
func (c *Client) Series(context.Context, metadata.Ask, SeriesID) (Series, error) {
	return Series{}, errNotImplemented
}

type Episode struct {
	ID     EpisodeID
	Season int
	Number int
	// Absolute is 0 when TVDB has none.
	Absolute int
	Aired    string
	Runtime  time.Duration
	Name     string
	Overview string
	Still    string
	IMDb     string
}

// Episodes is /series/{id}/episodes/{order}/{language}, every page. A page
// that fails fails the call, so a partial answer is never returned.
func (c *Client) Episodes(context.Context, metadata.Ask, SeriesID, Order) ([]Episode, error) {
	return nil, errNotImplemented
}

type Updates struct {
	Series []SeriesID
	Pages  int
}

// Updates is /updates?since=&type=series, one page.
func (c *Client) Updates(context.Context, metadata.Ask, time.Time, int) (Updates, error) {
	return Updates{}, errNotImplemented
}

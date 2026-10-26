// Package omdb is the OMDb client: the IMDb, Rotten Tomatoes and Metacritic
// ratings and the awards line of a title, by its IMDb id. Every Content
// Type that reads ratings shares it, so its host is a core endpoint. Every
// call leaves through the Node's metadata Line for OMDb, which holds the
// key, the rate limit, the daily cap and the cache.
package omdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/works"
)

const ID metadata.ProviderID = "omdb"

const label = "OMDb"

type Endpoints struct {
	API endpoint.URL `yaml:"api"`
	// KeyPage is where an Admin gets a key, opened in the browser only.
	KeyPage endpoint.URL `yaml:"key_page"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		API:     endpoint.MustParseURL("https://www.omdbapi.com"),
		KeyPage: endpoint.MustParseURL("https://www.omdbapi.com/apikey.aspx"),
	}
}

// Timings are OMDb's protocol facts: one request every Interval per
// holder, up to Burst at once.
type Timings struct {
	Interval time.Duration `yaml:"interval"`
	Burst    int           `yaml:"burst"`
}

func DefaultTimings() Timings {
	return Timings{Interval: 500 * time.Millisecond, Burst: 1}
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

// Provider is OMDb as the call layer knows it: the key is the apikey query
// parameter, and Classify reads OMDb's error bodies.
func (c *Client) Provider() metadata.Provider {
	return metadata.Provider{
		ID:       ID,
		Label:    label,
		Rate:     rate.Every(c.timings.Interval),
		Burst:    c.timings.Burst,
		Key:      func(q url.Values, key string) { q.Set("apikey", key) },
		Classify: classify,
	}
}

// OMDb's error words, which it sends as {"Response":"False","Error":…}.
const (
	errCapped     = "Request limit reached!"
	errBadKey     = "Invalid API key!"
	errNoKey      = "No API key provided."
	errNotFound   = "Movie not found!"
	errIncorrect  = "Incorrect IMDb ID."
	responseFalse = "False"
)

type errorBody struct {
	Response string `json:"Response"`
	Error    string `json:"Error"`
}

// classify reads OMDb's error body, which carries what the status doesn't:
// a spent daily allowance, a bad key or an unknown id. The allowance comes
// back at the next UTC midnight.
func classify(status int, _ http.Header, body []byte, now time.Time) (metadata.Outcome, bool) {
	var b errorBody
	if json.Unmarshal(body, &b) != nil || b.Response != responseFalse {
		return metadata.Outcome{}, false
	}
	switch {
	case b.Error == errCapped:
		until := now.UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
		return metadata.Outcome{Kind: metadata.Capped, Until: until, Words: "is out of its daily allowance until " + until.Format(time.RFC3339)}, true
	case b.Error == errBadKey || b.Error == errNoKey || status == http.StatusUnauthorized:
		return metadata.Outcome{Kind: metadata.Refused, Words: "refused the key: " + b.Error}, true
	case b.Error == errNotFound || b.Error == errIncorrect:
		return metadata.Outcome{Kind: metadata.NotFound, Words: "has no such record"}, true
	}
	return metadata.Outcome{Kind: metadata.Unreachable, Words: "responded: " + b.Error}, true
}

// Title is what OMDb knows of one title: its ratings on the scales
// works.RatingSources names, each only when OMDb has it, and its awards
// line, empty when OMDb has none.
type Title struct {
	Ratings []works.Rating
	Awards  string
}

type titleJSON struct {
	Awards  string `json:"Awards"`
	Ratings []struct {
		Source string `json:"Source"`
		Value  string `json:"Value"`
	} `json:"Ratings"`
	IMDbVotes string `json:"imdbVotes"`
}

// sources maps OMDb's names for its rating sources to the core's.
var sources = map[string]works.RatingSource{
	"Internet Movie Database": "imdb",
	"Rotten Tomatoes":         "rottentomatoes",
	"Metacritic":              "metacritic",
}

// notAvailable is OMDb's value for a field it has nothing for.
const notAvailable = "N/A"

// Title reads the title IMDb numbers imdbID.
func (c *Client) Title(ctx context.Context, ask metadata.Ask, imdbID string) (Title, error) {
	var t titleJSON
	if err := c.get(ctx, ask, imdbID, true, &t); err != nil {
		return Title{}, err
	}
	out, err := t.parse()
	if err != nil {
		return Title{}, fmt.Errorf("GET %s: %w", imdbID, &works.SourceError{Source: label, Words: err.Error(), Err: works.ErrInvalid})
	}
	return out, nil
}

func (t titleJSON) parse() (Title, error) {
	var out Title
	if t.Awards != notAvailable {
		out.Awards = t.Awards
	}
	for _, r := range t.Ratings {
		source, ok := sources[r.Source]
		if !ok || r.Value == notAvailable {
			continue
		}
		value, err := scaled(r.Value, works.RatingSources[source].Scale)
		if err != nil {
			return Title{}, fmt.Errorf("sent a %s rating SlipStream can't read: %w", r.Source, err)
		}
		rating := works.Rating{Source: source, Value: value}
		if source == "imdb" && t.IMDbVotes != notAvailable {
			votes, err := strconv.ParseInt(strings.ReplaceAll(t.IMDbVotes, ",", ""), 10, 64)
			if err != nil || votes < 0 {
				return Title{}, fmt.Errorf("sent IMDb votes SlipStream can't read: %q", t.IMDbVotes)
			}
			rating.Votes = &votes
		}
		out.Ratings = append(out.Ratings, rating)
	}
	return out, nil
}

// scaled reads OMDb's "8.8/10", "81%" or "67/100" onto scale.
func scaled(v string, scale float64) (float64, error) {
	num, den, ok := strings.Cut(v, "/")
	if !ok {
		num, ok = strings.CutSuffix(v, "%")
		den = "100"
	}
	n, nerr := strconv.ParseFloat(num, 64)
	d, derr := strconv.ParseFloat(den, 64)
	if !ok || nerr != nil || derr != nil || d <= 0 || n < 0 || n > d {
		return 0, fmt.Errorf("%q is not a score out of a whole", v)
	}
	return n * scale / d, nil
}

// testID is the title Test looks up: Fight Club.
const testID = "tt0137523"

// Test is the cheap call that says whether OMDb answers with ask's key.
// It spends one call of the daily allowance, and an unknown id still
// counts as an answer: OMDb checks the key first.
func (c *Client) Test(ctx context.Context, ask metadata.Ask) error {
	var t json.RawMessage
	err := c.get(ctx, ask, testID, false, &t)
	if errors.Is(err, works.ErrNotFound) {
		return nil
	}
	return err
}

func (c *Client) get(ctx context.Context, ask metadata.Ask, imdbID string, cache bool, v any) error {
	u := c.endpoints.API.JoinPath("/")
	u.RawQuery = url.Values{"i": {imdbID}}.Encode()
	if err := c.line.Get(ctx, ask, metadata.Request{URL: u, Cache: cache}, v); err != nil {
		return fmt.Errorf("GET %s: %w", imdbID, err)
	}
	return nil
}

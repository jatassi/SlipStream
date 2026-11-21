// Package subsource is SubSource's v1 API, as the recorded run of
// 2026-11-17 (subtitles/online/run-2026-11-17.md) found it: an X-API-Key
// header; movies/search by IMDb id, one movieId per season, and by text
// when no id finds it; subtitles?movieId&language=<english name>; and
// subtitles/{id}/download, a zip. Every 200 carries three windows, a
// minute, an hour and a day, and one X-Ratelimit-Reset, the minute's end.
// seasonNumber and episodeNumber don't filter, so an episode search reads
// the season's list and keeps what its release names allow.
package subsource

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/subtitles/cues"
	"github.com/jatassi/slipstream/internal/subtitling"
)

const ID calls.ID = "subsource"

const label = "SubSource"

type Endpoints struct {
	API endpoint.URL `yaml:"api"`
	// KeyPage is where an Admin gets a key, opened in the browser only.
	KeyPage endpoint.URL `yaml:"key_page"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		API:     endpoint.MustParseURL("https://api.subsource.net"),
		KeyPage: endpoint.MustParseURL("https://subsource.net/profile"),
	}
}

// Timings are SubSource's protocol facts: one request every Interval per
// holder, up to Burst at once. Searches and downloads share its windows,
// whose hour allows one request every 2 s.
type Timings struct {
	Interval time.Duration `yaml:"interval"`
	Burst    int           `yaml:"burst"`
}

func DefaultTimings() Timings { return Timings{Interval: 2 * time.Second, Burst: 1} }

type Client struct {
	line      *calls.Line
	endpoints Endpoints
}

func New(lines *calls.Lines, e Endpoints, t Timings) *Client {
	c := &Client{endpoints: e}
	c.line = lines.Line(calls.Provider{
		ID:       ID,
		Label:    label,
		Rate:     rate.Every(t.Interval),
		Burst:    t.Burst,
		Hosts:    []string{e.API.JoinPath().Host},
		Header:   func(h http.Header, key string) { h.Set("X-API-Key", key) },
		Classify: classify,
		KeyPage:  e.KeyPage.String(),
	})
	return c
}

func (c *Client) Spec() subtitling.Spec {
	return subtitling.Spec{ID: ID, Label: label, Kinds: []subtitling.Kind{subtitling.Feature, subtitling.Episode}, Keyed: true, KeyPage: c.endpoints.KeyPage.String(), Position: 2}
}

var _ subtitling.Provider = (*Client)(nil)

// probe is the recorded run's film, Heat (1995), which a key check looks
// up: SubSource has no account route, so the check is one search and its
// allowance the search's windows.
const probe = "tt0113277"

// CheckKey looks up one title. SubSource names no account or plan.
func (c *Client) CheckKey(ctx context.Context, cr calls.Credentials) (subtitling.Account, error) {
	_, err := c.titles(ctx, cr, url.Values{"searchType": {"imdb"}, "imdb": {probe}})
	return subtitling.Account{}, err
}

type titlesJSON struct {
	Data []titleJSON `json:"data"`
}

type titleJSON struct {
	MovieID     int    `json:"movieId"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	ReleaseYear int    `json:"releaseYear"`
	Season      *int   `json:"season"`
}

func (c *Client) titles(ctx context.Context, cr calls.Credentials, v url.Values) ([]titleJSON, error) {
	u := c.endpoints.API.JoinPath("api", "v1", "movies", "search")
	u.RawQuery = v.Encode()
	a, err := c.line.Do(ctx, cr, calls.Request{URL: u})
	if err != nil {
		return nil, err
	}
	var found titlesJSON
	if err := json.Unmarshal(a.Body, &found); err != nil {
		return nil, fmt.Errorf("%s sent titles SlipStream can't read: %w", label, err)
	}
	return found.Data, nil
}

// title is the movieId of the subject: a film, or the series' season,
// each its own title. It looks up the IMDb id, then the title and year.
func (c *Client) title(ctx context.Context, cr calls.Credentials, s subtitling.Subject) (movieID int, ok bool, err error) {
	kind := "movie"
	v := url.Values{}
	if s.Kind == subtitling.Episode {
		kind = "tvseries"
		v.Set("season", strconv.Itoa(s.Season))
	}
	fits := func(t titleJSON) bool {
		return t.Type == kind && (kind == "movie" || t.Season != nil && *t.Season == s.Season)
	}
	if s.IMDb != "" {
		v.Set("searchType", "imdb")
		v.Set("imdb", s.IMDb)
		found, err := c.titles(ctx, cr, v)
		if i := slices.IndexFunc(found, fits); err != nil || i >= 0 {
			return idAt(found, i), i >= 0, err
		}
	}
	if s.Title == "" {
		return 0, false, nil
	}
	v.Del("imdb")
	v.Set("searchType", "text")
	v.Set("q", s.Title)
	found, err := c.titles(ctx, cr, v)
	i := slices.IndexFunc(found, func(t titleJSON) bool {
		return fits(t) && strings.EqualFold(t.Title, s.Title) && (s.Year == 0 || kind != "movie" || t.ReleaseYear == s.Year)
	})
	return idAt(found, i), i >= 0, err
}

func idAt(ts []titleJSON, i int) int {
	if i < 0 {
		return 0
	}
	return ts[i].MovieID
}

type subtitlesJSON struct {
	Data []subtitleJSON `json:"data"`
}

// subtitleJSON is one listed subtitle, as the recorded run's fields name
// it.
type subtitleJSON struct {
	SubtitleID      int      `json:"subtitleId"`
	Language        string   `json:"language"`
	ReleaseInfo     []string `json:"releaseInfo"`
	HearingImpaired bool     `json:"hearingImpaired"`
	ForeignParts    bool     `json:"foreignParts"`
	Downloads       int      `json:"downloads"`
	Rating          struct {
		Good  int `json:"good"`
		Total int `json:"total"`
	} `json:"rating"`
	Link string `json:"link"`
}

// listLimit is the most subtitles one list asks for, the largest page
// SubSource's clients ask for.
const listLimit = 100

// Search lists the subject's title in the wanted language. An episode's
// list is its whole season's, so it keeps the subtitles whose release
// names the episode, and the packs, which name none.
func (c *Client) Search(ctx context.Context, cr calls.Credentials, q subtitling.Query) ([]subtitling.Offer, error) {
	id, ok, err := c.title(ctx, cr, q.Subject)
	if err != nil || !ok {
		return nil, err
	}
	name := languageName(q.Language)
	u := c.endpoints.API.JoinPath("api", "v1", "subtitles")
	u.RawQuery = url.Values{"movieId": {strconv.Itoa(id)}, "language": {name}, "limit": {strconv.Itoa(listLimit)}}.Encode()
	a, err := c.line.Do(ctx, cr, calls.Request{URL: u})
	if err != nil {
		return nil, err
	}
	var found subtitlesJSON
	if err := json.Unmarshal(a.Body, &found); err != nil {
		return nil, fmt.Errorf("%s sent subtitles SlipStream can't read: %w", label, err)
	}
	var out []subtitling.Offer
	for _, s := range found.Data {
		if s.Language != name {
			continue
		}
		if o, ok := c.offer(s, q); ok {
			out = append(out, o)
		}
	}
	return out, nil
}

func (c *Client) offer(s subtitleJSON, q subtitling.Query) (subtitling.Offer, bool) {
	o := subtitling.Offer{
		Provider: ID, Key: strconv.Itoa(s.SubtitleID), Releases: s.ReleaseInfo, Language: q.Language,
		Forced: s.ForeignParts, HI: s.HearingImpaired, Downloads: s.Downloads,
		Page: c.endpoints.KeyPage.JoinPath().ResolveReference(&url.URL{Path: s.Link}).String(),
	}
	if s.Rating.Total > 0 {
		o.Rating = float64(s.Rating.Good) / float64(s.Rating.Total)
	}
	if q.Subject.Kind != subtitling.Episode {
		return o, true
	}
	o.Season = q.Subject.Season
	for _, r := range s.ReleaseInfo {
		if season, episodes := cues.Numbered(r); season == q.Subject.Season || season == 0 {
			for _, e := range episodes {
				if !slices.Contains(o.Episodes, e) {
					o.Episodes = append(o.Episodes, e)
				}
			}
		}
	}
	slices.Sort(o.Episodes)
	switch {
	case len(o.Episodes) == 0:
		o.Pack = true
	case len(q.Subject.Episodes) > 0 && !slices.Contains(o.Episodes, q.Subject.Episodes[0]):
		return subtitling.Offer{}, false
	default:
		o.Pack = len(o.Episodes) > 1
	}
	return o, true
}

// Download reads the subtitle's zip.
func (c *Client) Download(ctx context.Context, cr calls.Credentials, o subtitling.Offer) (subtitling.Download, error) {
	a, err := c.line.Do(ctx, cr, calls.Request{URL: c.endpoints.API.JoinPath("api", "v1", "subtitles", o.Key, "download"), Lookup: true})
	if err != nil {
		return subtitling.Download{}, err
	}
	name := o.Key + ".zip"
	if _, params, err := mime.ParseMediaType(a.Header.Get("Content-Disposition")); err == nil && params["filename"] != "" {
		name = params["filename"]
	}
	return subtitling.Download{Name: name, Bytes: a.Body}, nil
}

// names are the languages whose SubSource name isn't their English name.
var names = map[string]string{"fa": "farsi_persian", "zh": "chinese_bg_code"}

// languageName is c as SubSource's language parameter and field write
// it: its English name in lower case, words joined by underscores.
func languageName(c language.Code) string {
	if n, ok := names[c.String()]; ok {
		return n
	}
	return strings.ReplaceAll(strings.ToLower(c.Name()), " ", "_")
}

type errorJSON struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	RetryAfter int    `json:"retryAfter"`
}

// classify reads what the shared rules don't: the windows on an OK
// answer, the refusal's message, and which window a 429 spent. The day
// window names no reset, so a spent day pauses to the next UTC midnight;
// another waits retryAfter, else to the minute's reset.
func classify(a calls.Answer, now time.Time) (calls.Verdict, bool) {
	switch a.Status {
	case http.StatusOK:
		if w := dayWindow(a.Header, now); w != nil {
			return calls.Verdict{Outcome: calls.Outcome{Kind: calls.OK, Allowance: &calls.Allowance{Searches: w, Downloads: w}}}, true
		}
	case http.StatusUnauthorized:
		var e errorJSON
		if json.Unmarshal(a.Body, &e) == nil && e.Message != "" {
			return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Refused, Words: "refused the key: " + e.Message}}, true
		}
	case http.StatusTooManyRequests:
		var e errorJSON
		_ = json.Unmarshal(a.Body, &e)
		switch {
		case a.Header.Get("X-Ratelimit-Remaining-Day") == "0":
			return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Capped, Until: midnight(now), Words: "has used up today's requests"}}, true
		case e.RetryAfter > 0:
			return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Limited, Until: now.Add(time.Duration(e.RetryAfter) * time.Second), Words: "is rate limiting SlipStream"}}, true
		}
		if reset, err := time.Parse(time.RFC3339, a.Header.Get("X-Ratelimit-Reset")); err == nil {
			return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Limited, Until: reset, Words: "is rate limiting SlipStream"}}, true
		}
	}
	return calls.Verdict{}, false
}

// dayWindow is the day's requests an OK answer reports, which searches
// and downloads share.
func dayWindow(h http.Header, now time.Time) *calls.Window {
	limit, err := strconv.Atoi(h.Get("X-Ratelimit-Limit-Day"))
	if err != nil {
		return nil
	}
	remaining, _ := strconv.Atoi(h.Get("X-Ratelimit-Remaining-Day"))
	return &calls.Window{Used: limit - remaining, Limit: limit, Remaining: remaining, Resets: midnight(now), Exact: true}
}

func midnight(now time.Time) time.Time { return now.UTC().Truncate(24 * time.Hour).Add(24 * time.Hour) }

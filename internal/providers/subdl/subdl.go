// Package subdl is SubDL's v2 API, as the recorded run of 2026-11-17
// (subtitles/online/run-2026-11-17.md) found it: a Bearer key,
// /api/v2/me for the key check and allowance, /api/v2/subtitles/search
// with releases=1 and comment=1, and /api/v2/subtitles/{id}/download on
// the api host. It never follows or keeps a result's url or
// unpack_files[].url, which carry the key; the id comes from subtitlePage
// (/s/info/{id}).
package subdl

import (
	"cmp"
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
	"github.com/jatassi/slipstream/internal/subtitling"
)

const ID calls.ID = "subdl"

const label = "SubDL"

type Endpoints struct {
	API endpoint.URL `yaml:"api"`
	// KeyPage is where an Admin gets a key, opened in the browser only.
	KeyPage endpoint.URL `yaml:"key_page"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		API:     endpoint.MustParseURL("https://api.subdl.com"),
		KeyPage: endpoint.MustParseURL("https://subdl.com/panel/api"),
	}
}

// Timings are SubDL's protocol facts: one request every Interval per
// holder, up to Burst at once, and the schedule a service_busy answer is
// tried again on.
type Timings struct {
	Interval time.Duration  `yaml:"interval"`
	Burst    int            `yaml:"burst"`
	Busy     calls.Schedule `yaml:"busy"`
}

func DefaultTimings() Timings {
	return Timings{Interval: time.Second, Burst: 1, Busy: calls.Schedule{Every: 5 * time.Second, Tries: 5}}
}

type Client struct {
	line      *calls.Line
	endpoints Endpoints
	timings   Timings
}

func New(lines *calls.Lines, e Endpoints, t Timings) *Client {
	c := &Client{endpoints: e, timings: t}
	c.line = lines.Line(c.provider())
	return c
}

// provider is SubDL as the call layer knows it: the key is a Bearer
// header, the api host is the only host, and classify reads SubDL's
// signals.
func (c *Client) provider() calls.Provider {
	return calls.Provider{
		ID:       ID,
		Label:    label,
		Rate:     rate.Every(c.timings.Interval),
		Burst:    c.timings.Burst,
		Hosts:    []string{c.endpoints.API.JoinPath().Host},
		Header:   func(h http.Header, key string) { h.Set("Authorization", "Bearer "+key) },
		Classify: c.classify,
		KeyPage:  c.endpoints.KeyPage.String(),
	}
}

func (c *Client) Spec() subtitling.Spec {
	return subtitling.Spec{ID: ID, Label: label, Kinds: []subtitling.Kind{subtitling.Feature, subtitling.Episode}, Keyed: true, KeyPage: c.endpoints.KeyPage.String(), Position: 1}
}

var _ subtitling.Provider = (*Client)(nil)

type meJSON struct {
	User struct {
		Name     string  `json:"name"`
		Username *string `json:"username"`
	} `json:"user"`
	Plan struct {
		Name string `json:"name"`
	} `json:"plan"`
}

// CheckKey reads /me, which costs no search. Its allowance is in the
// call's outcome.
func (c *Client) CheckKey(ctx context.Context, cr calls.Credentials) (subtitling.Account, error) {
	a, err := c.line.Do(ctx, cr, calls.Request{URL: c.endpoints.API.JoinPath("api", "v2", "me")})
	if err != nil {
		return subtitling.Account{}, err
	}
	var me meJSON
	if err := json.Unmarshal(a.Body, &me); err != nil {
		return subtitling.Account{}, fmt.Errorf("%s sent an account SlipStream can't read: %w", label, err)
	}
	name := me.User.Name
	if me.User.Username != nil && *me.User.Username != "" {
		name = *me.User.Username
	}
	return subtitling.Account{Name: name, Plan: me.Plan.Name}, nil
}

type searchJSON struct {
	Subtitles []subtitleJSON `json:"subtitles"`
}

// subtitleJSON is one search result as SubDL sends it. Its url fields
// carry the key and are never read.
type subtitleJSON struct {
	ReleaseName  string       `json:"release_name"`
	Releases     []string     `json:"releases"`
	Language     string       `json:"language"`
	Author       string       `json:"author"`
	Season       int          `json:"season"`
	Episode      int          `json:"episode"`
	EpisodeFrom  int          `json:"episode_from"`
	EpisodeEnd   int          `json:"episode_end"`
	HI           bool         `json:"hi"`
	SubtitlePage string       `json:"subtitlePage"`
	UnpackFiles  []unpackJSON `json:"unpack_files"`
}

type unpackJSON struct {
	Episode int    `json:"episode"`
	Format  string `json:"format"`
}

// aiAuthor is the author SubDL gives a machine translation; it has no AI
// field.
const aiAuthor = "SUBDL AI"

// client is SubDL's documented value for a third-party client.
const client = "custom_integration"

// Search asks for one want by IMDb id, else TMDB id, and for an episode
// by its season and first episode.
func (c *Client) Search(ctx context.Context, cr calls.Credentials, q subtitling.Query) ([]subtitling.Offer, error) {
	u := c.endpoints.API.JoinPath("api", "v2", "subtitles", "search")
	v := url.Values{"languages": {code(q.Language)}, "releases": {"1"}, "comment": {"1"}, "client": {client}}
	s := q.Subject
	v.Set("type", "movie")
	if s.Kind == subtitling.Episode {
		v.Set("type", "tv")
		v.Set("season", strconv.Itoa(s.Season))
		if len(s.Episodes) > 0 {
			v.Set("episode", strconv.Itoa(s.Episodes[0]))
		}
	}
	switch {
	case s.IMDb != "":
		v.Set("imdb_id", s.IMDb)
	case s.TMDB != "":
		v.Set("tmdb_id", s.TMDB)
	default:
		return nil, nil
	}
	u.RawQuery = v.Encode()
	a, err := c.line.Do(ctx, cr, calls.Request{URL: u})
	if err != nil {
		return nil, err
	}
	var found searchJSON
	if err := json.Unmarshal(a.Body, &found); err != nil {
		return nil, fmt.Errorf("%s sent search results SlipStream can't read: %w", label, err)
	}
	var out []subtitling.Offer
	for _, item := range found.Subtitles {
		if o, ok := c.offer(item); ok {
			out = append(out, o)
		}
	}
	return out, nil
}

// offer reads one result. full_season is unreliable, so a pack is known
// by its members, which carry their episode.
func (c *Client) offer(s subtitleJSON) (subtitling.Offer, bool) {
	key, ok := pageID(s.SubtitlePage)
	lang, err := languageOf(s.Language)
	if !ok || err != nil {
		return subtitling.Offer{}, false
	}
	o := subtitling.Offer{
		Provider: ID, Key: key, Language: lang, HI: s.HI, AI: s.Author == aiAuthor, Season: s.Season,
		Page: c.endpoints.KeyPage.JoinPath().ResolveReference(&url.URL{Path: s.SubtitlePage}).String(),
	}
	for _, r := range append([]string{s.ReleaseName}, s.Releases...) {
		if r != "" && !slices.Contains(o.Releases, r) {
			o.Releases = append(o.Releases, r)
		}
	}
	images := 0
	for _, f := range s.UnpackFiles {
		if f.Episode > 0 && !slices.Contains(o.Episodes, f.Episode) {
			o.Episodes = append(o.Episodes, f.Episode)
		}
		if imageFormats[strings.ToLower(f.Format)] {
			images++
		}
	}
	slices.Sort(o.Episodes)
	o.Image = images > 0 && images == len(s.UnpackFiles)
	switch {
	case len(o.Episodes) > 1:
		o.Pack = true
	case s.EpisodeFrom > 0 && s.EpisodeEnd > s.EpisodeFrom:
		o.Episodes, o.Pack = nil, true
		for e := s.EpisodeFrom; e <= s.EpisodeEnd; e++ {
			o.Episodes = append(o.Episodes, e)
		}
	case len(o.Episodes) == 0 && s.Episode > 0:
		o.Episodes = []int{s.Episode}
	}
	return o, true
}

var imageFormats = map[string]bool{"sup": true, "sub": true, "idx": true}

// pageID is the id in a subtitlePage of /s/info/{id}.
func pageID(page string) (string, bool) {
	rest, ok := strings.CutPrefix(page, "/s/info/")
	id, _, _ := strings.Cut(rest, "/")
	return id, ok && id != ""
}

// Download reads the v2 route on the api host, which serves the zip with
// no redirect.
func (c *Client) Download(ctx context.Context, cr calls.Credentials, o subtitling.Offer) (subtitling.Download, error) {
	a, err := c.line.Do(ctx, cr, calls.Request{URL: c.endpoints.API.JoinPath("api", "v2", "subtitles", o.Key, "download"), Lookup: true})
	if err != nil {
		return subtitling.Download{}, err
	}
	name := o.Key + ".zip"
	if _, params, err := mime.ParseMediaType(a.Header.Get("Content-Disposition")); err == nil && params["filename"] != "" {
		name = params["filename"]
	}
	return subtitling.Download{Name: name, Bytes: a.Body}, nil
}

// code is a language as SubDL's languages parameter and language field
// write it: the upper-case ISO 639-1 code, and BR_PT.
func code(c language.Code) string {
	if c.String() == "pt-BR" {
		return "BR_PT"
	}
	return strings.ToUpper(c.Tag())
}

func languageOf(s string) (language.Code, error) {
	if s == "BR_PT" {
		return language.Parse("pt-BR")
	}
	return language.Parse(strings.ToLower(s))
}

type errorJSON struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// signal is one error answer: its status and v2 code, "" for any.
type signal struct {
	status int
	code   string
}

// reaction is what a signal does to a call.
type reaction uint8

const (
	// refused quotes the provider's message, which the UI shows.
	refused reaction = iota + 1
	// capped pauses until the allowance resets.
	capped
	// limited waits on Retry-After under the shared rules.
	limited
	// busy tries again on Timings.Busy.
	busy
	down
)

// signals is the v2 error table. The recorded run fixed 401 unauthorized;
// the 429 codes are documented, not recorded, and a later run corrects
// this table without reshaping the client.
var signals = map[signal]reaction{
	{http.StatusUnauthorized, "unauthorized"}:                   refused,
	{http.StatusTooManyRequests, "quota_exceeded"}:              capped,
	{http.StatusTooManyRequests, "daily_limit"}:                 capped,
	{http.StatusTooManyRequests, "api_download_limit_exceeded"}: capped,
	{http.StatusTooManyRequests, "rate_limit"}:                  limited,
	{http.StatusTooManyRequests, "service_busy"}:                busy,
	{http.StatusPaymentRequired, ""}:                            down,
	{http.StatusNotFound, ""}:                                   down,
}

// classify reads what the shared rules don't: the allowance on an OK
// answer, SubDL's error codes, and a 403 with no JSON, which is a passing
// challenge page and not a refused key.
func (c *Client) classify(a calls.Answer, now time.Time) (calls.Verdict, bool) {
	if a.Status == http.StatusOK {
		if allowance := allowanceOf(a); allowance != nil {
			return calls.Verdict{Outcome: calls.Outcome{Kind: calls.OK, Allowance: allowance}}, true
		}
		return calls.Verdict{}, false
	}
	var e errorJSON
	if json.Unmarshal(a.Body, &e) != nil || e.Error.Code == "" {
		switch {
		case a.Status == http.StatusForbidden:
			return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Unreachable, Words: "responded with a challenge page"}}, true
		case signals[signal{a.Status, ""}] == down:
			return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Unreachable, Words: "responded with " + http.StatusText(a.Status)}}, true
		}
		return calls.Verdict{}, false
	}
	r := cmp.Or(signals[signal{a.Status, e.Error.Code}], signals[signal{a.Status, ""}])
	switch r {
	case refused:
		return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Refused, Words: "refused the key: " + e.Error.Message}}, true
	case capped:
		return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Capped, Until: resetOf(a.Header, now), Words: "has used up its allowance until it resets"}}, true
	case busy:
		return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Unreachable, Words: "is busy"}, Retry: &c.timings.Busy}, true
	case down:
		return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Unreachable, Words: "responded with " + http.StatusText(a.Status)}}, true
	case limited:
	}
	return calls.Verdict{}, false
}

// resetOf is X-Ratelimit-Reset, an ISO 8601 UTC time, else the next UTC
// midnight, when the free plan's day resets.
func resetOf(h http.Header, now time.Time) time.Time {
	if t, err := time.Parse(time.RFC3339, h.Get("X-Ratelimit-Reset")); err == nil {
		return t
	}
	return now.UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
}

type usageJSON struct {
	Usage *struct {
		Search    windowJSON `json:"search"`
		Downloads windowJSON `json:"downloads"`
	} `json:"usage"`
}

type windowJSON struct {
	Used      int    `json:"used"`
	Limit     int    `json:"limit"`
	Remaining int    `json:"remaining"`
	ResetAt   string `json:"reset_at"`
}

// allowanceOf is what an OK answer says of the key's allowance: a search's
// X-Ratelimit-* headers, or /me's counters. Downloads count exactly; the
// search counters lag a few calls behind.
func allowanceOf(a calls.Answer) *calls.Allowance {
	if limit, err := strconv.Atoi(a.Header.Get("X-Ratelimit-Limit")); err == nil {
		remaining, _ := strconv.Atoi(a.Header.Get("X-Ratelimit-Remaining"))
		resets, _ := time.Parse(time.RFC3339, a.Header.Get("X-Ratelimit-Reset"))
		return &calls.Allowance{Searches: &calls.Window{Used: limit - remaining, Limit: limit, Remaining: remaining, Resets: resets}}
	}
	var u usageJSON
	if !strings.HasPrefix(a.Header.Get("Content-Type"), "application/json") || json.Unmarshal(a.Body, &u) != nil || u.Usage == nil {
		return nil
	}
	return &calls.Allowance{Searches: u.Usage.Search.window(false), Downloads: u.Usage.Downloads.window(true)}
}

func (w windowJSON) window(exact bool) *calls.Window {
	resets, _ := time.Parse(time.RFC3339, w.ResetAt)
	return &calls.Window{Used: w.Used, Limit: w.Limit, Remaining: w.Remaining, Resets: resets, Exact: exact}
}

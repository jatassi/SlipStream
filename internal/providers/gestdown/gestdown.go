// Package gestdown is Gestdown, Addic7ed's proxy: TV only and no key, as
// the recorded run of 2026-11-17 (subtitles/online/run-2026-11-17.md)
// found it. It looks a series up by TVDB id at /shows/external/tvdb/{id},
// lists an episode at /subtitles/get/{show}/{season}/{episode}/{lang},
// keeping only completed subtitles, and downloads /subtitles/download/{id}
// as a raw SRT, or a zip for a season pack. A 404 finds nothing, a 423
// is a show being refreshed, tried again on Timings.Refresh, and a
// download's 429 with a JSON error means its upstream accounts are out.
package gestdown

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/subtitling"
)

const ID calls.ID = "gestdown"

const label = "Gestdown"

type Endpoints struct {
	API endpoint.URL `yaml:"api"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{API: endpoint.MustParseURL("https://api.gestdown.info")}
}

// Timings are Gestdown's protocol facts: one request every Interval per
// holder, up to Burst at once, under its per-address bucket that every
// install shares; the schedule a 423 refresh is tried again on; and how
// long its upstream accounts stay out once a download says so.
type Timings struct {
	Interval time.Duration  `yaml:"interval"`
	Burst    int            `yaml:"burst"`
	Refresh  calls.Schedule `yaml:"refresh"`
	Upstream time.Duration  `yaml:"upstream"`
}

func DefaultTimings() Timings {
	return Timings{Interval: 2 * time.Second, Burst: 1, Refresh: calls.Schedule{Every: 30 * time.Second, Tries: 3}, Upstream: time.Hour}
}

type Client struct {
	line      *calls.Line
	endpoints Endpoints
	timings   Timings
}

func New(lines *calls.Lines, e Endpoints, t Timings) *Client {
	c := &Client{endpoints: e, timings: t}
	c.line = lines.Line(calls.Provider{
		ID:       ID,
		Label:    label,
		Rate:     rate.Every(t.Interval),
		Burst:    t.Burst,
		Hosts:    []string{e.API.JoinPath().Host},
		Classify: c.classify,
	})
	return c
}

func (c *Client) Spec() subtitling.Spec {
	return subtitling.Spec{ID: ID, Label: label, Kinds: []subtitling.Kind{subtitling.Episode}, Position: 3}
}

var _ subtitling.Provider = (*Client)(nil)

// CheckKey has nothing to check: Gestdown takes no key.
func (c *Client) CheckKey(context.Context, calls.Credentials) (subtitling.Account, error) {
	return subtitling.Account{}, nil
}

type showsJSON struct {
	Shows []struct {
		ID string `json:"id"`
	} `json:"shows"`
}

type episodeJSON struct {
	MatchingSubtitles []subtitleJSON `json:"matchingSubtitles"`
}

type subtitleJSON struct {
	SubtitleID      string `json:"subtitleId"`
	Version         string `json:"version"`
	Release         string `json:"release"`
	Completed       bool   `json:"completed"`
	HearingImpaired bool   `json:"hearingImpaired"`
	DownloadCount   int    `json:"downloadCount"`
}

// Search lists an episode, the file's first, by its series' TVDB id. A
// film, and a series Gestdown doesn't know, find nothing.
func (c *Client) Search(ctx context.Context, cr calls.Credentials, q subtitling.Query) ([]subtitling.Offer, error) {
	s := q.Subject
	if s.Kind != subtitling.Episode || s.TVDB == "" || len(s.Episodes) == 0 {
		return nil, nil
	}
	var shows showsJSON
	if found, err := c.get(ctx, cr, &shows, "shows", "external", "tvdb", s.TVDB); !found || err != nil || len(shows.Shows) == 0 {
		return nil, err
	}
	var ep episodeJSON
	if found, err := c.get(ctx, cr, &ep, "subtitles", "get", shows.Shows[0].ID, strconv.Itoa(s.Season), strconv.Itoa(s.Episodes[0]), q.Language.String()); !found || err != nil {
		return nil, err
	}
	var out []subtitling.Offer
	for _, m := range ep.MatchingSubtitles {
		if !m.Completed {
			continue
		}
		o := subtitling.Offer{
			Provider: ID, Key: m.SubtitleID, Language: q.Language, HI: m.HearingImpaired, Downloads: m.DownloadCount, Season: s.Season,
			Pack: strings.HasPrefix(m.SubtitleID, "sp_") && !strings.Contains(m.SubtitleID, "_ep_"),
		}
		if !o.Pack {
			o.Episodes = []int{s.Episodes[0]}
		}
		for _, r := range append([]string{m.Release}, strings.Split(m.Version, ",")...) {
			if r = strings.TrimSpace(r); r != "" && !slices.Contains(o.Releases, r) {
				o.Releases = append(o.Releases, r)
			}
		}
		out = append(out, o)
	}
	return out, nil
}

// get reads one lookup into v, and answers false when Gestdown has no
// such record.
func (c *Client) get(ctx context.Context, cr calls.Credentials, v any, path ...string) (bool, error) {
	a, err := c.line.Do(ctx, cr, calls.Request{URL: c.endpoints.API.JoinPath(path...), Lookup: true})
	if f, ok := errors.AsType[*calls.Failed](err); ok && f.Outcome.Kind == calls.NotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(a.Body, v); err != nil {
		return false, fmt.Errorf("%s sent a response SlipStream can't read: %w", label, err)
	}
	return true, nil
}

// Download reads a subtitle as Gestdown serves it: an SRT, or a pack's
// zip.
func (c *Client) Download(ctx context.Context, cr calls.Credentials, o subtitling.Offer) (subtitling.Download, error) {
	a, err := c.line.Do(ctx, cr, calls.Request{URL: c.endpoints.API.JoinPath("subtitles", "download", o.Key), Lookup: true})
	if err != nil {
		return subtitling.Download{}, err
	}
	name := o.Key + ".srt"
	if _, params, err := mime.ParseMediaType(a.Header.Get("Content-Disposition")); err == nil && params["filename"] != "" {
		name = params["filename"]
	}
	return subtitling.Download{Name: name, Bytes: a.Body}, nil
}

// classify reads Gestdown's own signals: a 423 refresh, tried again on
// its schedule, and a 429 whose JSON error says its upstream accounts are
// out. Its address bucket's 429 carries Retry-After, which the shared
// rules wait on.
func (c *Client) classify(a calls.Answer, now time.Time) (calls.Verdict, bool) {
	switch a.Status {
	case http.StatusLocked:
		return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Unreachable, Words: "is refreshing the show"}, Retry: &c.timings.Refresh}, true
	case http.StatusTooManyRequests:
		var e struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(a.Body, &e) == nil && e.Error != "" {
			return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Capped, Until: now.Add(c.timings.Upstream), Words: "has run out of downloads from Addic7ed"}}, true
		}
	}
	return calls.Verdict{}, false
}

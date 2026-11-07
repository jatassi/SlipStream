package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/internal/artwork"
	"github.com/jatassi/slipstream/fakes/internal/images"
	"github.com/jatassi/slipstream/fakes/scenario"
)

type config struct {
	scenario, store                    string
	tmdbAPI, tmdbImages, omdbAPI       string
	tvdbAPI, tvdbArtwork               string
	tmdbKey, omdbKey, tvdbKey, tvdbPin string
	movies, series                     []int64
	client                             *http.Client
}

// recording is what one run writes into the Scenario.
type recording struct {
	movies     []scenario.TMDBMovie
	tmdbSeries []scenario.TMDBSeries
	tvdbSeries []scenario.TVDBSeries
	titles     []scenario.OMDbTitle
}

func run(ctx context.Context, c config) error {
	var rec recording
	for _, id := range sortedOnce(c.movies) {
		m, title, err := c.movie(ctx, id)
		if err != nil {
			return err
		}
		rec.movies = append(rec.movies, m)
		rec.titles = append(rec.titles, title)
	}
	if len(c.series) > 0 {
		token, err := c.login(ctx)
		if err != nil {
			return err
		}
		for _, id := range sortedOnce(c.series) {
			tv, show, title, err := c.show(ctx, token, id)
			if err != nil {
				return err
			}
			rec.tvdbSeries = append(rec.tvdbSeries, tv)
			rec.tmdbSeries = append(rec.tmdbSeries, show)
			rec.titles = append(rec.titles, title)
		}
	}
	slices.SortFunc(rec.tmdbSeries, func(a, b scenario.TMDBSeries) int { return cmp.Compare(a.ID, b.ID) })
	return rewrite(c.scenario, rec)
}

func sortedOnce(ids []int64) []int64 {
	return slices.Compact(slices.Sorted(slices.Values(ids)))
}

// file is one image to save in the store: the path its host serves it at,
// the URL to download it from, and the width to scale it down to before
// it's saved, zero to save it as downloaded.
type file struct {
	path, from string
	width      int
}

// download saves each of files in the store, unless the store has it: a
// TMDB or TVDB file path never names other bytes.
func (c config) download(ctx context.Context, files []file) error {
	seen := map[string]bool{}
	for _, f := range files {
		if f.path == "" || seen[f.path] {
			continue
		}
		seen[f.path] = true
		if err := c.save(ctx, f); err != nil {
			return err
		}
	}
	return nil
}

func (c config) save(ctx context.Context, f file) error {
	name, ok := artwork.Name(f.path)
	if !ok {
		return fmt.Errorf("the artwork store can't hold %s", f.path)
	}
	dest := filepath.Join(c.store, filepath.FromSlash(name))
	if _, err := os.Stat(dest); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("check the store: %w", err)
	}
	body, err := c.get(ctx, f.from, nil, "")
	if err != nil {
		return err
	}
	if f.width > 0 {
		format, _ := images.FormatOf(f.path)
		if body, err = images.Scale(format, body, "w"+strconv.Itoa(f.width)); err != nil {
			return fmt.Errorf("scale %s: %w", f.path, err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("save %s: %w", f.path, err)
	}
	if err := os.WriteFile(dest, body, 0o644); err != nil {
		return fmt.Errorf("save %s: %w", f.path, err)
	}
	return nil
}

type omdbJSON struct {
	Response     string `json:"Response"`
	Error        string `json:"Error"`
	Awards       string `json:"Awards"`
	Metascore    string `json:"Metascore"`
	IMDbRating   string `json:"imdbRating"`
	IMDbVotes    string `json:"imdbVotes"`
	TotalSeasons string `json:"totalSeasons"`
	Ratings      []struct {
		Source string `json:"Source"`
		Value  string `json:"Value"`
	} `json:"Ratings"`
}

const notAvailable = "N/A"

// omdb is OMDb's ratings and awards for the title IMDb numbers id, and
// how many seasons OMDb lists for a series.
func (c config) omdb(ctx context.Context, id string) (scenario.OMDbTitle, int, error) {
	var o omdbJSON
	if err := c.getJSON(ctx, c.omdbAPI, url.Values{"i": {id}, "apikey": {c.omdbKey}}, "", &o); err != nil {
		return scenario.OMDbTitle{}, 0, err
	}
	if o.Response != "True" {
		return scenario.OMDbTitle{}, 0, fmt.Errorf("OMDb has no title %s: %s", id, o.Error)
	}
	out := scenario.OMDbTitle{IMDbID: id}
	if o.Awards != notAvailable {
		out.Awards = o.Awards
	}
	if v, err := strconv.ParseFloat(o.IMDbRating, 64); err == nil {
		out.IMDbRating = &v
	}
	if v, err := strconv.ParseInt(strings.ReplaceAll(o.IMDbVotes, ",", ""), 10, 64); err == nil {
		out.IMDbVotes = &v
	}
	if v, err := strconv.Atoi(o.Metascore); err == nil {
		out.Metacritic = &v
	}
	for _, r := range o.Ratings {
		if v, err := strconv.Atoi(strings.TrimSuffix(r.Value, "%")); r.Source == "Rotten Tomatoes" && err == nil {
			out.RottenTomatoes = &v
		}
	}
	seasons, _ := strconv.Atoi(o.TotalSeasons)
	return out, seasons, nil
}

func (c config) getJSON(ctx context.Context, at string, q url.Values, token string, v any) error {
	body, err := c.get(ctx, at, q, token)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("read %s: %w", at, err)
	}
	return nil
}

// get is the body at at with query q, sent with token as its bearer
// when there is one.
func (c config) get(ctx context.Context, at string, q url.Values, token string) ([]byte, error) {
	u := at
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", at, err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return c.send(req, at)
}

// send is the body of the response to req, which at names. Errors name
// at, never the query or the headers, which hold the keys.
func (c config) send(req *http.Request, at string) ([]byte, error) {
	res, err := c.client.Do(req)
	if err != nil {
		if ue, ok := errors.AsType[*url.Error](err); ok {
			err = ue.Err
		}
		return nil, fmt.Errorf("%s %s: %w", req.Method, at, err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", req.Method, at, err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s %s: %s %s", req.Method, at, res.Status, body)
	}
	return body, nil
}

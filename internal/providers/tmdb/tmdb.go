// Package tmdb is the TMDB v3 client every Content Type that reads TMDB
// shares, so its host is one core endpoint.
package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/works"
)

type Endpoints struct {
	API endpoint.URL `yaml:"api"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{API: endpoint.MustParseURL("https://api.themoviedb.org/3")}
}

type Client struct {
	http      *http.Client
	base      string
	userAgent string
}

func New(c *http.Client, e Endpoints, userAgent string) *Client {
	return &Client{http: c, base: e.API.String(), userAgent: userAgent}
}

// MovieResult is one movie in a search. ReleaseDate is YYYY-MM-DD, or
// empty when TMDB has none.
type MovieResult struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	ReleaseDate string `json:"release_date"`
}

type Movie struct {
	ID           int64        `json:"id"`
	Title        string       `json:"title"`
	ReleaseDate  string       `json:"release_date"`
	Runtime      int64        `json:"runtime"`
	ExternalIDs  ExternalIDs  `json:"external_ids"`
	ReleaseDates ReleaseDates `json:"release_dates"`
}

type ExternalIDs struct {
	IMDbID string `json:"imdb_id"`
}

type ReleaseDates struct {
	Results []CountryReleases `json:"results"`
}

type CountryReleases struct {
	Country  string    `json:"iso_3166_1"`
	Releases []Release `json:"release_dates"`
}

// Release is one dated release. Date is RFC 3339, Type TMDB's release
// type.
type Release struct {
	Date string `json:"release_date"`
	Type int    `json:"type"`
}

const (
	TheatricalLimited = 2
	Theatrical        = 3
	Digital           = 4
)

// Released is the earliest theatrical or digital date in any country, as
// YYYY-MM-DD, or empty when TMDB lists none.
func (m Movie) Released() string {
	var dates []string
	for _, c := range m.ReleaseDates.Results {
		for _, r := range c.Releases {
			if (r.Type == TheatricalLimited || r.Type == Theatrical || r.Type == Digital) && len(r.Date) >= len(time.DateOnly) {
				dates = append(dates, r.Date[:len(time.DateOnly)])
			}
		}
	}
	if len(dates) == 0 {
		return ""
	}
	return slices.Min(dates)
}

func (c *Client) SearchMovies(ctx context.Context, query string) ([]MovieResult, error) {
	var page struct {
		Results []MovieResult `json:"results"`
	}
	err := c.get(ctx, "/search/movie", url.Values{"query": {query}}, works.ErrUnavailable, &page)
	return page.Results, err
}

// Movie reads one movie with its external ids and release dates. id is
// TMDB's id as a decimal string.
func (c *Client) Movie(ctx context.Context, id string) (Movie, error) {
	var m Movie
	err := c.get(ctx, "/movie/"+url.PathEscape(id), url.Values{"append_to_response": {"external_ids,release_dates"}}, works.ErrNotFound, &m)
	return m, err
}

// get answers a 404 with notFound: only a lookup by id can tell that TMDB
// has no record.
func (c *Client) get(ctx context.Context, path string, q url.Values, notFound error, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path+"?"+q.Encode(), http.NoBody)
	if err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	res, err := c.http.Do(req)
	if err != nil {
		if ue, ok := errors.AsType[*url.Error](err); ok {
			err = ue.Err
		}
		return fmt.Errorf("GET %s: %w: %w", path, err, works.ErrUnavailable)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, res.Body)
	}
	switch {
	case res.StatusCode == http.StatusNotFound:
		return fmt.Errorf("GET %s: %s: %w", path, res.Status, notFound)
	case res.StatusCode >= http.StatusInternalServerError:
		return fmt.Errorf("GET %s: %s: %w", path, res.Status, works.ErrUnavailable)
	case res.StatusCode != http.StatusOK:
		return fmt.Errorf("GET %s: %s", path, res.Status)
	}
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		return fmt.Errorf("GET %s: decode: %w: %w", path, err, works.ErrUnavailable)
	}
	return nil
}

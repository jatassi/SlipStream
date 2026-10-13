package musicbrainz

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/works"
)

// pageSize is the most a browse returns per request.
const pageSize = 100

type Client struct {
	http      *http.Client
	base      string
	userAgent string
}

func New(c *http.Client, base endpoint.URL, userAgent string) *Client {
	return &Client{http: c, base: base.String(), userAgent: userAgent}
}

type Artist struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SortName       string `json:"sort-name"`
	Country        string `json:"country"`
	Disambiguation string `json:"disambiguation"`
}

// Album is a release group of primary type Album or EP.
type Album struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	PrimaryType  string `json:"primary-type"`
	FirstRelease string `json:"first-release-date"`
}

type Release struct {
	ID             string   `json:"id"`
	Date           string   `json:"date"`
	Disambiguation string   `json:"disambiguation"`
	Media          []Medium `json:"media"`
}

type Medium struct {
	Position int     `json:"position"`
	Tracks   []Track `json:"tracks"`
}

type Track struct {
	Position  int       `json:"position"`
	Title     string    `json:"title"`
	Length    *int64    `json:"length"`
	Recording Recording `json:"recording"`
}

type Recording struct {
	ID string `json:"id"`
}

func (c *Client) SearchArtists(ctx context.Context, query string) ([]Artist, error) {
	var page struct {
		Artists []Artist `json:"artists"`
	}
	err := c.get(ctx, "/artist", url.Values{"query": {query}, "limit": {"25"}}, works.ErrUnavailable, &page)
	return page.Artists, err
}

func (c *Client) Artist(ctx context.Context, mbid string) (Artist, error) {
	var a Artist
	err := c.get(ctx, "/artist/"+url.PathEscape(mbid), nil, works.ErrNotFound, &a)
	return a, err
}

func (c *Client) Albums(ctx context.Context, artist string) ([]Album, error) {
	albums, err := browse(ctx, c, "/release-group", url.Values{"artist": {artist}, "type": {"album|ep"}}, func(p *albumPage) ([]Album, int) {
		return p.ReleaseGroups, p.Count
	})
	if err != nil {
		return nil, err
	}
	slices.SortFunc(albums, func(a, b Album) int {
		return cmp.Or(compareDates(a.FirstRelease, b.FirstRelease), cmp.Compare(a.Title, b.Title), cmp.Compare(a.ID, b.ID))
	})
	return albums, nil
}

type albumPage struct {
	ReleaseGroups []Album `json:"release-groups"`
	Count         int     `json:"release-group-count"`
}

func (c *Client) Canonical(ctx context.Context, album string) (Release, bool, error) {
	releases, err := browse(ctx, c, "/release", url.Values{"release-group": {album}, "status": {"official"}, "inc": {"recordings media"}}, func(p *releasePage) ([]Release, int) {
		return p.Releases, p.Count
	})
	if err != nil {
		return Release{}, false, err
	}
	r, ok := canonical(releases)
	return r, ok, nil
}

type releasePage struct {
	Releases []Release `json:"releases"`
	Count    int       `json:"release-count"`
}

func canonical(releases []Release) (Release, bool) {
	if len(releases) == 0 {
		return Release{}, false
	}
	return slices.MinFunc(releases, func(a, b Release) int {
		return cmp.Or(trueFirst(a.Disambiguation == "", b.Disambiguation == ""), compareDates(a.Date, b.Date), cmp.Compare(a.ID, b.ID))
	}), true
}

// compareDates orders partial dates as text, which sorts them by time, and
// an empty date last.
func compareDates(a, b string) int {
	return cmp.Or(trueFirst(a != "", b != ""), cmp.Compare(a, b))
}

func trueFirst(a, b bool) int {
	switch {
	case a == b:
		return 0
	case a:
		return -1
	}
	return 1
}

func browse[P, T any](ctx context.Context, c *Client, path string, q url.Values, items func(*P) ([]T, int)) ([]T, error) {
	var out []T
	q.Set("limit", strconv.Itoa(pageSize))
	for {
		q.Set("offset", strconv.Itoa(len(out)))
		var page P
		if err := c.get(ctx, path, q, works.ErrUnavailable, &page); err != nil {
			return nil, err
		}
		got, total := items(&page)
		out = append(out, got...)
		if len(got) == 0 || len(out) >= total {
			return out, nil
		}
	}
}

// get answers a 404 with notFound: only a lookup by key can tell that the
// source has no record, while a 404 inside a browse means the source changed
// between two requests.
func (c *Client) get(ctx context.Context, path string, q url.Values, notFound error, v any) error {
	if q == nil {
		q = url.Values{}
	}
	q.Set("fmt", "json")
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

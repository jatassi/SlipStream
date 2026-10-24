package musicbrainz

import (
	"cmp"
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"

	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

// pageSize is the most a browse returns per request.
const pageSize = 100

type Client struct {
	line *metadata.Line
	base endpoint.URL
}

// New registers MusicBrainz's Line on lines. The fake it reads keeps no
// rate limit.
func New(lines *metadata.Lines, base endpoint.URL) *Client {
	return &Client{line: lines.Line(metadata.Provider{ID: "musicbrainz", Label: "MusicBrainz", Rate: rate.Inf, Burst: 1}), base: base}
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

func (c *Client) SearchArtists(ctx context.Context, ask metadata.Ask, query string) ([]Artist, error) {
	var page struct {
		Artists []Artist `json:"artists"`
	}
	err := c.get(ctx, ask, "/artist", url.Values{"query": {query}, "limit": {"25"}}, false, &page)
	return page.Artists, err
}

func (c *Client) Artist(ctx context.Context, ask metadata.Ask, mbid string) (Artist, error) {
	var a Artist
	err := c.get(ctx, ask, "/artist/"+url.PathEscape(mbid), url.Values{}, true, &a)
	return a, err
}

func (c *Client) Albums(ctx context.Context, ask metadata.Ask, artist string) ([]Album, error) {
	albums, err := browse(ctx, c, ask, "/release-group", url.Values{"artist": {artist}, "type": {"album|ep"}}, func(p *albumPage) ([]Album, int) {
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

func (c *Client) Canonical(ctx context.Context, ask metadata.Ask, album string) (Release, bool, error) {
	releases, err := browse(ctx, c, ask, "/release", url.Values{"release-group": {album}, "status": {"official"}, "inc": {"recordings media"}}, func(p *releasePage) ([]Release, int) {
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

func browse[P, T any](ctx context.Context, c *Client, ask metadata.Ask, path string, q url.Values, items func(*P) ([]T, int)) ([]T, error) {
	var out []T
	q.Set("limit", strconv.Itoa(pageSize))
	for {
		q.Set("offset", strconv.Itoa(len(out)))
		var page P
		if err := c.get(ctx, ask, path, q, false, &page); err != nil {
			return nil, err
		}
		got, total := items(&page)
		out = append(out, got...)
		if len(got) == 0 || len(out) >= total {
			return out, nil
		}
	}
}

// get decodes the answer at path into v. Only a lookup by key can tell
// that the source has no record: a 404 inside a browse means the source
// changed between two requests.
func (c *Client) get(ctx context.Context, ask metadata.Ask, path string, q url.Values, lookup bool, v any) error {
	q.Set("fmt", "json")
	u := c.base.JoinPath(path)
	u.RawQuery = q.Encode()
	if err := c.line.Get(ctx, ask, metadata.Request{URL: u, Lookup: lookup}, v); err != nil {
		return fmt.Errorf("GET %s: %w", path, err)
	}
	return nil
}

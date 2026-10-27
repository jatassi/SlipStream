package searching

import (
	"cmp"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/jatassi/slipstream/internal/searching/internal/torznab"
)

// pageSize is the page a Torznab search asks for: Prowlarr's.
const pageSize = 100

// torznabDriver reads a Torznab feed: one Prowlarr or Jackett indexer's
// api.
type torznabDriver struct {
	// results is the most Releases a search pages to.
	results int
}

// Search pages by offset until a short page, a page with nothing new, or
// results Releases.
func (d torznabDriver) Search(ctx context.Context, s *Site, q Query) ([]Release, error) {
	query := torznab.Query{Mode: q.Mode, IDs: q.IDs, Text: q.Text, Season: q.Season, Episode: q.Episode, Categories: s.Categories}
	var items []torznab.Item
	seen := map[string]bool{}
	for offset := 0; len(items) < d.results; offset += pageSize {
		page, err := d.feed(ctx, s, query.Page(offset, pageSize))
		if err != nil {
			return nil, err
		}
		fresh := 0
		for _, it := range page {
			if key := cmp.Or(it.GUID, it.Magnet, it.Title); !seen[key] {
				seen[key] = true
				items = append(items, it)
				fresh++
			}
		}
		if len(page) < pageSize || fresh == 0 {
			break
		}
	}
	return releases(items[:min(len(items), d.results)]), nil
}

// Recent is the feed's newest page.
func (d torznabDriver) Recent(ctx context.Context, s *Site) ([]Release, error) {
	items, err := d.feed(ctx, s, torznab.Query{Mode: "search", Categories: s.Categories}.Page(0, pageSize))
	if err != nil {
		return nil, err
	}
	return releases(items), nil
}

// Test reads the caps, then the newest page, and the sources it names.
func (d torznabDriver) Test(ctx context.Context, s *Site) (Report, error) {
	body, err := d.api(ctx, s, url.Values{"t": {"caps"}})
	if err != nil {
		return Report{}, err
	}
	caps, err := torznab.ParseCaps(body)
	if err != nil {
		return Report{}, &Failing{Err: err}
	}
	recent, err := d.Recent(ctx, s)
	if err != nil {
		return Report{}, err
	}
	r := Report{Capabilities: Capabilities{Modes: caps.Modes, Categories: caps.Categories}, Items: len(recent)}
	for _, rel := range recent {
		if rel.Source != nil && !slices.Contains(r.Sources, rel.Source.Name) {
			r.Sources = append(r.Sources, rel.Source.Name)
		}
	}
	return r, nil
}

// Fetch downloads link from wherever it is, without the feed's key: a
// link carries its own.
func (torznabDriver) Fetch(ctx context.Context, s *Site, link Link) (Got, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link.raw, http.NoBody)
	if err != nil {
		return nil, &Failing{Err: fmt.Errorf("build the request: %w", withoutURL(err))}
	}
	res, err := s.Do(ctx, req, Download)
	switch {
	case err != nil:
		return nil, err
	case !res.Magnet.IsZero():
		return Magnet{Link: res.Magnet}, nil
	case !succeeded(res.Status):
		return nil, &Failing{Err: fmt.Errorf("the Indexer responded with status %d", res.Status)}
	}
	return Torrent{Raw: res.Body}, nil
}

func (d torznabDriver) feed(ctx context.Context, s *Site, params url.Values) ([]torznab.Item, error) {
	body, err := d.api(ctx, s, params)
	if err != nil {
		return nil, err
	}
	items, err := torznab.ParseFeed(body)
	if err != nil {
		return nil, &Failing{Err: err}
	}
	return items, nil
}

// api sends params with the feed's key to its api path, the only request
// that carries the key. A Torznab error element is read whatever the
// status.
func (torznabDriver) api(ctx context.Context, s *Site, params url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, torznab.API(s.Address, s.APIKey, params).String(), http.NoBody)
	if err != nil {
		return nil, &Failing{Err: fmt.Errorf("build the request: %w", withoutURL(err))}
	}
	res, err := s.Do(ctx, req, Read)
	if err != nil {
		return nil, err
	}
	if e, ok := torznab.ErrorOf(res.Body); ok {
		return nil, refusal(e)
	}
	if !succeeded(res.Status) {
		return nil, &Failing{Err: fmt.Errorf("the Indexer responded with status %d", res.Status)}
	}
	return res.Body, nil
}

func succeeded(status int) bool {
	return status >= http.StatusOK && status < http.StatusMultipleChoices
}

// refusal is the outcome of a Torznab error element.
func refusal(e torznab.Error) error {
	switch {
	case e.Code == http.StatusTooManyRequests || e.Description == "Request limit reached":
		return &Limited{}
	case e.Code >= 100 && e.Code <= 199, e.Code == 910:
		return &Stopped{Cause: AccountRefused, Text: e.Description}
	case e.Code >= 200 && e.Code <= 203:
		return &Stopped{Cause: RequestRejected, Text: e.Description}
	}
	return &Failing{Err: fmt.Errorf("the Indexer responded with error %d: %s", e.Code, e.Description)}
}

func releases(items []torznab.Item) []Release {
	out := make([]Release, len(items))
	for i, it := range items {
		out[i] = Release{
			Title: it.Title, Size: it.Size, Published: it.Published, Seeders: it.Seeders, Leechers: it.Leechers, Grabs: it.Grabs,
			Link: LinkOf(it.Link), Magnet: LinkOf(it.Magnet), InfoHash: it.InfoHash, Categories: it.Categories, IDs: it.IDs,
			Factors: Factors{Download: it.DownloadFactor, Upload: it.UploadFactor},
			Floor:   Floor{MinRatio: it.MinRatio, MinSeed: it.MinSeed},
			Details: it.Details,
		}
		if src := it.Source; src != nil {
			out[i].Source = &Source{ID: src.ID, Name: src.Name, Privacy: src.Privacy, Origin: src.Origin}
		}
	}
	return out
}

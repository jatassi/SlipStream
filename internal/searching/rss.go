package searching

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jatassi/slipstream/internal/searching/internal/rss"
)

// RSSReading is how an RSS feed's items are read, found by its Test when
// it is added.
type RSSReading = rss.Reading

// RSSRefusal is why an RSS feed's Test found it can't be used.
type RSSRefusal = rss.Refusal

const (
	RSSNoItems      = rss.NoItems
	RSSNoTitle      = rss.NoTitle
	RSSNoLink       = rss.NoLink
	RSSRepeatedGUID = rss.RepeatedGUID
	RSSNoSize       = rss.NoSize
)

// rssDriver reads an RSS feed: one site's recent items, with no search.
type rssDriver struct{}

func (rssDriver) Search(context.Context, *Site, Query) ([]Release, error) {
	return nil, &Failing{Err: errors.New("an RSS feed has no search")}
}

func (d rssDriver) Recent(ctx context.Context, s *Site) ([]Release, error) {
	body, err := d.feed(ctx, s)
	if err != nil {
		return nil, err
	}
	entries, err := rss.Read(body, s.RSS)
	if err != nil {
		return nil, &Failing{Err: err}
	}
	out := make([]Release, len(entries))
	for i, e := range entries {
		out[i] = Release{
			Title: e.Title, Size: e.Size, Published: e.Published, Seeders: e.Seeders,
			Link: LinkOf(e.Link), Magnet: LinkOf(e.Magnet), InfoHash: e.InfoHash, Details: e.Details,
		}
	}
	return out, nil
}

// Test reads the feed and finds how its items are read. A feed SlipStream
// can't use passes with the refusal in its Report, since the site
// answered.
func (d rssDriver) Test(ctx context.Context, s *Site, _ *CaptchaAnswer) (Report, error) {
	body, err := d.feed(ctx, s)
	if err != nil {
		return Report{}, err
	}
	reading, items, err := rss.Check(body, s.RSS.AllowNoSize)
	if refusal, ok := errors.AsType[rss.Refusal](err); ok {
		return Report{Refusal: refusal}, nil
	}
	if err != nil {
		return Report{}, &Failing{Err: err}
	}
	return Report{Items: items, RSS: &reading}, nil
}

func (rssDriver) Captcha(context.Context, *Site) (Captcha, error) {
	return Captcha{}, &Failing{Err: errors.New("an RSS feed shows no captcha")}
}

func (rssDriver) Fetch(ctx context.Context, s *Site, link Link) (Got, error) {
	return fetch(ctx, s, link)
}

// feed is the feed's body. A feed that refuses the session's cookie stops
// the Indexer with cookie_expired.
func (rssDriver) feed(ctx context.Context, s *Site) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.Address.String(), http.NoBody)
	if err != nil {
		return nil, &Failing{Err: fmt.Errorf("build the request: %w", withoutURL(err))}
	}
	res, err := s.Do(ctx, req, Read)
	switch {
	case err != nil:
		return nil, err
	case res.Cookies && (res.Status == http.StatusUnauthorized || res.Status == http.StatusForbidden):
		return nil, &Stopped{Cause: CookieExpired}
	case !succeeded(res.Status):
		return nil, &Failing{Err: fmt.Errorf("the feed responded with status %d", res.Status)}
	}
	return res.Body, nil
}

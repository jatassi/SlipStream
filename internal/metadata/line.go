package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net/http"
	"net/url"
	"time"

	"github.com/jatassi/slipstream/internal/calls"
	"github.com/jatassi/slipstream/internal/works"
)

type Timings struct {
	// Attempts bounds the tries of one call, the first included.
	Attempts int `yaml:"attempts"`
	// Try bounds one request, its body included.
	Try time.Duration `yaml:"try"`
	// RetryMin and RetryMax bound the jittered backoff between tries.
	RetryMin time.Duration `yaml:"retry_min"`
	RetryMax time.Duration `yaml:"retry_max"`
	// Wait is how long one call may wait on a 429's Retry-After in all;
	// a longer one marks the provider limited until it ends.
	Wait time.Duration `yaml:"wait"`
	// CacheEntries bounds a Node's answer cache.
	CacheEntries int `yaml:"cache_entries"`
	// RetestMin and RetestMax bound the doubling wait before the Controller
	// tests a provider again on a holder that couldn't reach it.
	RetestMin time.Duration `yaml:"retest_min"`
	RetestMax time.Duration `yaml:"retest_max"`
	// ArtworkRetry is how long the Controller answers an image it failed
	// to fetch as missing before it tries again.
	ArtworkRetry time.Duration `yaml:"artwork_retry"`
	// ChangesPages bounds the pages of one changes list a holder reads; a
	// longer list counts as none, and the refresh tiers cover the gap.
	ChangesPages int `yaml:"changes_pages"`
}

func DefaultTimings() Timings {
	return Timings{
		Attempts: 3, Try: 30 * time.Second, RetryMin: time.Second, RetryMax: 8 * time.Second, Wait: 10 * time.Second, CacheEntries: 4096,
		RetestMin: time.Minute, RetestMax: time.Hour, ArtworkRetry: 5 * time.Minute, ChangesPages: 500,
	}
}

// calls is the call layer's share of t.
func (t Timings) calls() calls.Timings {
	return calls.Timings{Attempts: t.Attempts, Try: t.Try, RetryMin: t.RetryMin, RetryMax: t.RetryMax, Wait: t.Wait}
}

// Lines is one Node's metadata call layer: the shared call layer and the
// answer cache.
type Lines struct {
	calls *calls.Lines
	cache *Cache
}

func NewLines(c *http.Client, userAgent string, t Timings, r *rand.Rand) *Lines {
	return &Lines{calls: calls.NewLines(c, userAgent, t.calls(), r), cache: newCache(t.CacheEntries)}
}

// Line is p's way out of this Node. A provider has one Line, so a second
// registration of its id panics.
func (l *Lines) Line(p calls.Provider) *Line {
	return &Line{calls: l.calls.Line(p), cache: l.cache, id: p.ID, label: p.Label}
}

// Line is one provider's way out of one Node, through the answer cache.
type Line struct {
	calls *calls.Line
	cache *Cache
	id    calls.ID
	label string
}

type Request struct {
	// URL is what to GET, without the key: the Line adds it to what it
	// sends and never to the cache key.
	URL *url.URL
	// Lookup: a 404 means the source has no such record; otherwise a 404
	// is unreachable.
	Lookup bool
	// Cache: the answer may be served from and written to the Node's cache.
	Cache bool
}

type Image struct {
	Type string
	Body []byte
}

// Get sends GET req.URL and decodes a 200 into v. It records the call's
// outcome in ask's tally and returns a *works.SourceError on failure.
func (l *Line) Get(ctx context.Context, ask Ask, req Request, v any) error {
	key := cacheKey(l.id, req.URL)
	if req.Cache {
		l.cache.Sync(ask.generation)
		if body, ok := l.cache.Get(key, ask.cacheFor); ok && !ask.fresh {
			return l.decode(body, v)
		}
	}
	a, err := l.call(ctx, ask, req.URL, req.Lookup)
	if err != nil {
		return err
	}
	if err := l.decode(a.Body, v); err != nil {
		return err
	}
	if req.Cache {
		l.cache.Put(key, a.Body)
	}
	return nil
}

// Bytes fetches the image at u, uncached, under the same gate, bucket and
// retries as Get.
func (l *Line) Bytes(ctx context.Context, ask Ask, u *url.URL) (Image, error) {
	a, err := l.call(ctx, ask, u, true)
	if err != nil {
		return Image{}, err
	}
	return Image{Type: a.Header.Get("Content-Type"), Body: a.Body}, nil
}

func (l *Line) decode(body []byte, v any) error {
	if err := json.Unmarshal(body, v); err != nil {
		return &works.SourceError{Source: l.label, Words: "sent a response SlipStream can't read: " + err.Error(), Err: works.ErrInvalid}
	}
	return nil
}

var failures = map[calls.OutcomeKind]error{
	calls.NotFound:    works.ErrNotFound,
	calls.Refused:     works.ErrRefused,
	calls.Limited:     works.ErrUnavailable,
	calls.Capped:      works.ErrUnavailable,
	calls.Unreachable: works.ErrUnavailable,
	calls.PinRefused:  works.ErrPinRefused,
}

func (l *Line) call(ctx context.Context, ask Ask, u *url.URL, lookup bool) (calls.Answer, error) {
	a, err := l.calls.Do(ctx, ask.credentials(), calls.Request{URL: u, Lookup: lookup})
	if f, failed := errors.AsType[*calls.Failed](err); failed {
		o := f.Outcome
		return calls.Answer{}, &works.SourceError{Source: l.label, Until: o.Until, Words: o.Words, Err: failures[o.Kind]}
	}
	return a, err
}

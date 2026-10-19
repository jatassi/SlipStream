package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/works"
)

// MaxBody bounds every answer a Line reads. An image travels whole in one
// unary call, so this is also the largest image.
const MaxBody = 16 << 20

// Provider is one source's protocol facts, declared by its client package
// from its own timings. The core never names a provider.
type Provider struct {
	ID    ProviderID
	Label string
	// Rate and Burst are the token bucket per provider per holder.
	Rate  rate.Limit
	Burst int
	// Key puts the key on a request; nil for a keyless host.
	Key func(q url.Values, key string)
	// Classify reads an answer the shared rules don't settle, such as a 200
	// whose body says the daily allowance is spent. Nil leaves every answer
	// to the shared rules.
	Classify func(status int, h http.Header, body []byte, now time.Time) (Outcome, bool)
}

type Timings struct {
	// Attempts bounds the tries of one call, the first included.
	Attempts int `yaml:"attempts"`
	// Try bounds one try, its body included.
	Try time.Duration `yaml:"try"`
	// RetryMin and RetryMax bound the jittered backoff between tries.
	RetryMin time.Duration `yaml:"retry_min"`
	RetryMax time.Duration `yaml:"retry_max"`
	// Wait is how long one call may wait on a 429's Retry-After in all;
	// a longer one marks the provider limited until it ends.
	Wait time.Duration `yaml:"wait"`
	// CacheEntries bounds a Node's answer cache.
	CacheEntries int `yaml:"cache_entries"`
}

func DefaultTimings() Timings {
	return Timings{Attempts: 3, Try: 30 * time.Second, RetryMin: time.Second, RetryMax: 8 * time.Second, Wait: 10 * time.Second, CacheEntries: 4096}
}

// Lines is one Node's call layer: an HTTP client, the answer cache and a
// Line per provider.
type Lines struct {
	client    *http.Client
	userAgent string
	timings   Timings
	cache     *Cache

	mu    sync.Mutex
	rand  *rand.Rand
	lines map[ProviderID]*Line
}

func NewLines(c *http.Client, userAgent string, t Timings, r *rand.Rand) *Lines {
	return &Lines{client: c, userAgent: userAgent, timings: t, cache: newCache(t.CacheEntries), rand: r, lines: map[ProviderID]*Line{}}
}

// Line is p's way out of this Node. A provider has one Line, so a second
// registration of its id panics.
func (l *Lines) Line(p Provider) *Line {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.lines[p.ID]; ok {
		panic(fmt.Sprintf("metadata: provider %s already has a Line", p.ID))
	}
	line := &Line{lines: l, p: p, bucket: rate.NewLimiter(p.Rate, p.Burst)}
	l.lines[p.ID] = line
	return line
}

func (l *Lines) backoff(n int) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	return backoff.Delay(l.timings.RetryMin, l.timings.RetryMax, n, l.rand)
}

// Line is one provider's way out of one Node.
type Line struct {
	lines  *Lines
	p      Provider
	bucket *rate.Limiter

	mu sync.Mutex
	// gate is the last Limited or Capped outcome; no call is sent before
	// its Until.
	gate Outcome
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
	key := cacheKey(l.p.ID, req.URL)
	if req.Cache {
		l.lines.cache.Sync(ask.generation)
		if body, ok := l.lines.cache.Get(key, ask.cacheFor); ok && !ask.fresh {
			return l.decode(body, v)
		}
	}
	a, err := l.call(ctx, ask, req.URL, req.Lookup)
	if err != nil {
		return err
	}
	if err := l.decode(a.body, v); err != nil {
		return err
	}
	if req.Cache {
		l.lines.cache.Put(key, a.body)
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
	return Image{Type: a.header.Get("Content-Type"), Body: a.body}, nil
}

func (l *Line) decode(body []byte, v any) error {
	if err := json.Unmarshal(body, v); err != nil {
		return &works.SourceError{Source: l.p.Label, Words: "sent an answer SlipStream can't read: " + err.Error(), Err: works.ErrInvalid}
	}
	return nil
}

type answer struct {
	body   []byte
	header http.Header
}

var failures = map[OutcomeKind]error{
	NotFound:    works.ErrNotFound,
	Refused:     works.ErrRefused,
	Limited:     works.ErrUnavailable,
	Capped:      works.ErrUnavailable,
	Unreachable: works.ErrUnavailable,
}

// verdict is what one try settled: an outcome, the answer when it is OK,
// and whether another try may change it.
type verdict struct {
	Outcome
	answer
	retry bool
}

func (l *Line) call(ctx context.Context, ask Ask, u *url.URL, lookup bool) (answer, error) {
	v, err := l.tries(ctx, ask, u, lookup)
	if err != nil {
		return answer{}, fmt.Errorf("%s: %w", l.p.Label, err)
	}
	ask.tally.record(v.Outcome)
	if v.Kind == OK {
		return v.answer, nil
	}
	if v.Kind == Limited || v.Kind == Capped {
		l.close(v.Outcome)
	}
	return answer{}, &works.SourceError{Source: l.p.Label, Until: v.Until, Words: v.Words, Err: failures[v.Kind]}
}

// tries sends u until an answer settles the call or the tries run out. A
// 429 whose Retry-After ends inside the wait budget and ctx is waited out;
// a timeout, connection error or 5xx waits a jittered backoff step.
func (l *Line) tries(ctx context.Context, ask Ask, u *url.URL, lookup bool) (verdict, error) {
	t := l.lines.timings
	budget := time.Now().Add(t.Wait)
	if d, ok := ctx.Deadline(); ok && d.Before(budget) {
		budget = d
	}
	for try := 1; ; {
		if o, shut := l.shut(); shut {
			return verdict{Outcome: o}, nil
		}
		if err := l.bucket.Wait(ctx); err != nil {
			return verdict{}, fmt.Errorf("waiting on the rate limit: %w", err)
		}
		v := l.try(ctx, ask, u, lookup)
		if err := ctx.Err(); err != nil {
			return verdict{}, fmt.Errorf("gave up: %w", err)
		}
		if try >= t.Attempts {
			return v, nil
		}
		var wait time.Duration
		switch {
		case v.Kind == Limited && !v.Until.After(budget):
			wait = time.Until(v.Until)
		case v.retry:
			wait = l.lines.backoff(try - 1)
		default:
			return v, nil
		}
		try++
		if err := sleep(ctx, wait); err != nil {
			return verdict{}, err
		}
	}
}

func (l *Line) try(ctx context.Context, ask Ask, u *url.URL, lookup bool) verdict {
	ctx, cancel := context.WithTimeout(ctx, l.lines.timings.Try)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.keyed(ask, u).String(), http.NoBody)
	if err != nil {
		return verdict{Outcome: l.outcome(Unreachable, "can't be asked for "+u.Redacted())}
	}
	req.Header.Set("User-Agent", l.lines.userAgent)
	res, err := l.lines.client.Do(req)
	if err != nil {
		if ue, ok := errors.AsType[*url.Error](err); ok {
			err = ue.Err
		}
		return verdict{Outcome: l.outcome(Unreachable, "didn't answer: "+err.Error()), retry: true}
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, MaxBody+1))
	if err != nil {
		return verdict{Outcome: l.outcome(Unreachable, "broke off its answer: "+err.Error()), retry: true}
	}
	if len(body) > MaxBody {
		return verdict{Outcome: l.outcome(Unreachable, "answered more than 16 MiB")}
	}
	now := time.Now()
	if l.p.Classify != nil {
		if o, ok := l.p.Classify(res.StatusCode, res.Header, body, now); ok {
			o.Provider, o.At = l.p.ID, now
			return verdict{Outcome: o, answer: answer{body: body, header: res.Header}}
		}
	}
	switch {
	case res.StatusCode == http.StatusOK:
		return verdict{Outcome: l.outcome(OK, ""), answer: answer{body: body, header: res.Header}}
	case res.StatusCode == http.StatusTooManyRequests:
		until := now.Add(l.retryAfter(res.Header, now))
		o := l.outcome(Limited, "is rate limited until "+until.UTC().Format(time.RFC3339))
		o.Until = until
		return verdict{Outcome: o}
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return verdict{Outcome: l.outcome(Refused, "refused the key: "+res.Status)}
	case res.StatusCode == http.StatusNotFound && lookup:
		return verdict{Outcome: l.outcome(NotFound, "has no such record")}
	case res.StatusCode >= http.StatusInternalServerError:
		return verdict{Outcome: l.outcome(Unreachable, "answered "+res.Status), retry: true}
	}
	return verdict{Outcome: l.outcome(Unreachable, "answered "+res.Status)}
}

// retryAfter reads Retry-After as seconds or an HTTP date. Without one the
// provider waits a backoff step.
func (l *Line) retryAfter(h http.Header, now time.Time) time.Duration {
	v := h.Get("Retry-After")
	if s, err := strconv.ParseInt(v, 10, 64); err == nil && s >= 0 {
		return time.Duration(s) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(t.Sub(now), 0)
	}
	return l.lines.backoff(0)
}

func (l *Line) keyed(ask Ask, u *url.URL) *url.URL {
	key := ask.keys[l.p.ID]
	if key == "" || l.p.Key == nil {
		return u
	}
	q := u.Query()
	l.p.Key(q, key)
	k := *u
	k.RawQuery = q.Encode()
	return &k
}

func (l *Line) outcome(k OutcomeKind, words string) Outcome {
	return Outcome{Provider: l.p.ID, Kind: k, Words: words, At: time.Now()}
}

func (l *Line) shut() (Outcome, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if !now.Before(l.gate.Until) {
		return Outcome{}, false
	}
	o := l.gate
	o.At = now
	return o, true
}

func (l *Line) close(o Outcome) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if o.Until.After(l.gate.Until) {
		l.gate = o
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("waiting to try again: %w", ctx.Err())
	}
}

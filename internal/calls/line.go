package calls

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/httpretry"
)

// Lines is one Node's call layer: an HTTP client and a Line per provider.
type Lines struct {
	client    *http.Client
	userAgent string
	timings   Timings

	mu    sync.Mutex
	rand  *rand.Rand
	lines map[ID]*Line
}

func NewLines(c *http.Client, userAgent string, t Timings, r *rand.Rand) *Lines {
	return &Lines{client: c, userAgent: userAgent, timings: t, rand: r, lines: map[ID]*Line{}}
}

// Line is p's way out of this Node. A provider has one Line, so a second
// registration of its id panics.
func (l *Lines) Line(p Provider) *Line {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.lines[p.ID]; ok {
		panic(fmt.Sprintf("calls: provider %s already has a Line", p.ID))
	}
	line := &Line{lines: l, p: p, bucket: rate.NewLimiter(p.Rate, p.Burst), gates: map[string]Outcome{}, sessions: map[credential]*session{}}
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
	// gates is the last Limited or Capped outcome of each key, the empty
	// key included; no call with that key is sent before its Until. An
	// allowance is a key's, so another key goes on.
	gates map[string]Outcome
	// sessions is the login of each key and PIN a call has sent.
	sessions map[credential]*session
}

type credential struct{ key, pin string }

// session is one credential's bearer token. Its mutex holds each call
// while one logs in, so calls on an expired token log in once.
type session struct {
	mu      sync.Mutex
	token   string
	expires time.Time
}

// Request is one GET.
type Request struct {
	// URL is what to GET, without the key: the Line adds it only to what
	// it sends.
	URL *url.URL
	// Lookup: a 404 means the provider has no such record; otherwise a 404
	// is unreachable.
	Lookup bool
}

// Answer is one response: its status, headers and body, at most MaxBody.
type Answer struct {
	Status int
	Header http.Header
	Body   []byte
}

// Failed is a call that didn't end OK.
type Failed struct {
	// Label is the provider's name.
	Label   string
	Outcome Outcome
}

func (f *Failed) Error() string { return f.Label + " " + f.Outcome.Words }

// Do sends r under the gate, bucket and tries, records the call's outcome
// in c's tally and returns the OK answer. Any other end is a *Failed. The
// key never reaches a URL that an error prints.
func (l *Line) Do(ctx context.Context, c Credentials, r Request) (Answer, error) {
	v, err := l.tries(ctx, c, r)
	if err != nil {
		return Answer{}, fmt.Errorf("%s: %w", l.p.Label, err)
	}
	c.Tally.record(v.Outcome)
	if v.Kind == OK {
		return v.Answer, nil
	}
	if v.Kind == Limited || v.Kind == Capped {
		l.close(c, v.Outcome)
	}
	return Answer{}, &Failed{Label: l.p.Label, Outcome: v.Outcome}
}

// settled is what one try settled: an outcome, the answer when it is OK,
// and whether another try may change it.
type settled struct {
	Outcome
	Answer
	retry bool
}

// tries sends r until an answer settles the call or the tries run out. A
// 429 whose Retry-After ends inside the wait budget and ctx is waited out;
// a timeout, connection error or 5xx waits a jittered backoff step.
func (l *Line) tries(ctx context.Context, c Credentials, r Request) (settled, error) {
	t := l.lines.timings
	budget := time.Now().Add(t.Wait)
	if d, ok := ctx.Deadline(); ok && d.Before(budget) {
		budget = d
	}
	for try := 1; ; {
		if o, shut := l.shut(c); shut {
			return settled{Outcome: o}, nil
		}
		v, err := l.try(ctx, c, r)
		if err != nil {
			return settled{}, err
		}
		if err := ctx.Err(); err != nil {
			return settled{}, fmt.Errorf("gave up: %w", err)
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
			return settled{}, err
		}
	}
}

func (l *Line) try(ctx context.Context, c Credentials, r Request) (settled, error) {
	if l.p.Auth == nil {
		o := outgoing{method: http.MethodGet, url: l.keyed(c, r.URL)}
		if l.p.Header != nil {
			o.key = l.key(c)
		}
		return l.send(ctx, o, r.Lookup)
	}
	cred := credential{key: c.Keys[l.p.ID], pin: c.Pins[l.p.ID]}
	if cred.key == "" {
		return settled{Outcome: l.outcome(Refused, "can't be called without a key")}, nil
	}
	var stale string
	for {
		b, v, err := l.bearer(ctx, cred, stale)
		if err != nil || v.Kind != OK {
			return v, err
		}
		v, err = l.send(ctx, outgoing{method: http.MethodGet, url: r.URL, bearer: b.token}, r.Lookup)
		if err != nil || v.Status != http.StatusUnauthorized || b.minted || stale != "" {
			return v, err
		}
		stale = b.token
	}
}

type bearer struct {
	token string
	// minted is whether this call logged in for it.
	minted bool
}

// bearer is c's token, from a login when c has none, it has expired or it
// is stale, the token a 401 just refused. A settled that isn't OK is why
// the login failed.
func (l *Line) bearer(ctx context.Context, c credential, stale string) (bearer, settled, error) {
	l.mu.Lock()
	s, ok := l.sessions[c]
	if !ok {
		s = &session{}
		l.sessions[c] = s
	}
	l.mu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && s.token != stale && time.Now().Before(s.expires) {
		return bearer{token: s.token}, settled{Outcome: Outcome{Kind: OK}}, nil
	}
	s.token = ""
	u, body := l.p.Auth.Login(c.key, c.pin)
	v, err := l.send(ctx, outgoing{method: http.MethodPost, url: u, body: body}, false)
	switch {
	case err != nil:
		return bearer{}, settled{}, err
	case v.Kind == Refused && l.p.Auth.Refused(v.Status, v.Body) == CredentialPin:
		return bearer{}, settled{Outcome: l.outcome(PinRefused, "refused the PIN")}, nil
	case v.Kind != OK:
		return bearer{}, v, nil
	}
	token, expires, err := l.p.Auth.Token(v.Body)
	if err != nil {
		return bearer{}, settled{Outcome: l.outcome(Unreachable, "sent a login response SlipStream can't read: "+err.Error())}, nil
	}
	s.token, s.expires = token, expires
	return bearer{token: token, minted: true}, v, nil
}

type outgoing struct {
	method string
	url    *url.URL
	// body is sent as JSON when set.
	body   []byte
	bearer string
	// key goes in the provider's Header when set.
	key string
}

// send waits on the bucket, sends o and classifies the response.
func (l *Line) send(ctx context.Context, o outgoing, lookup bool) (settled, error) {
	if err := l.bucket.Wait(ctx); err != nil {
		return settled{}, fmt.Errorf("waiting on the rate limit: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, l.lines.timings.Try)
	defer cancel()
	var sent io.Reader = http.NoBody
	if o.body != nil {
		sent = bytes.NewReader(o.body)
	}
	req, err := http.NewRequestWithContext(ctx, o.method, o.url.String(), sent)
	if err != nil {
		return settled{Outcome: l.outcome(Unreachable, "has an address SlipStream can't send a request to: "+o.url.Redacted())}, nil
	}
	req.Header.Set("User-Agent", l.lines.userAgent)
	if o.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if o.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+o.bearer)
	}
	if o.key != "" {
		l.p.Header(req.Header, o.key)
	}
	res, err := l.lines.client.Do(req)
	if err != nil {
		if ue, ok := errors.AsType[*url.Error](err); ok {
			err = ue.Err
		}
		return settled{Outcome: l.outcome(Unreachable, "didn't respond: "+err.Error()), retry: true}, nil
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, MaxBody+1))
	if err != nil {
		return settled{Outcome: l.outcome(Unreachable, "ended its response early: "+err.Error()), retry: true}, nil
	}
	if len(body) > MaxBody {
		return settled{Outcome: l.outcome(Unreachable, "sent a response over 16 MiB")}, nil
	}
	return l.classify(Answer{Status: res.StatusCode, Header: res.Header, Body: body}, res.Status, lookup), nil
}

func (l *Line) classify(a Answer, status string, lookup bool) settled {
	now := time.Now()
	if l.p.Classify != nil {
		if o, ok := l.p.Classify(a.Status, a.Header, a.Body, now); ok {
			o.Provider, o.At = l.p.ID, now
			return settled{Outcome: o, Answer: a}
		}
	}
	v := settled{Answer: a}
	switch {
	case a.Status == http.StatusOK:
		v.Outcome = l.outcome(OK, "")
	case a.Status == http.StatusTooManyRequests:
		until := now.Add(l.retryAfter(a.Header, now))
		v.Outcome = l.outcome(Limited, "is rate limiting SlipStream")
		v.Until = until
	case a.Status == http.StatusUnauthorized || a.Status == http.StatusForbidden:
		v.Outcome = l.outcome(Refused, "refused the key: "+status)
	case a.Status == http.StatusNotFound && lookup:
		v.Outcome = l.outcome(NotFound, "has no such record")
	case a.Status >= http.StatusInternalServerError:
		v.Outcome, v.retry = l.outcome(Unreachable, "responded with "+status), true
	default:
		v.Outcome = l.outcome(Unreachable, "responded with "+status)
	}
	return v
}

// retryAfter is the wait Retry-After asks for. Without one the provider
// waits a backoff step.
func (l *Line) retryAfter(h http.Header, now time.Time) time.Duration {
	if d := httpretry.After(h.Get("Retry-After"), now); d > 0 {
		return d
	}
	return l.lines.backoff(0)
}

func (l *Line) keyed(c Credentials, u *url.URL) *url.URL {
	key := l.key(c)
	if key == "" || l.p.Key == nil {
		return u
	}
	q := u.Query()
	l.p.Key(q, key)
	k := *u
	k.RawQuery = q.Encode()
	return &k
}

// key is the key a call with c sends: none to a keyless host.
func (l *Line) key(c Credentials) string {
	if l.p.Key == nil && l.p.Header == nil && l.p.Auth == nil {
		return ""
	}
	return c.Keys[l.p.ID]
}

func (l *Line) outcome(k OutcomeKind, words string) Outcome {
	return Outcome{Provider: l.p.ID, Kind: k, Words: words, At: time.Now()}
}

func (l *Line) shut(c Credentials) (Outcome, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	key := l.key(c)
	o := l.gates[key]
	if !now.Before(o.Until) {
		delete(l.gates, key)
		return Outcome{}, false
	}
	o.At = now
	return o, true
}

func (l *Line) close(c Credentials, o Outcome) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if key := l.key(c); o.Until.After(l.gates[key].Until) {
		l.gates[key] = o
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

package searching

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/magnet"
)

// Site is one job's handle on one Indexer.
type Site struct {
	Indexer    string
	Revision   int64
	Address    *url.URL
	Settings   map[string]string
	APIKey     string
	Categories []int
	RSS        RSSReading
	// Port is the ported site a native Indexer runs.
	Port string
	// Code is the two-factor code the Admin typed for this Test, "" for
	// none.
	Code string
	// def is a catalog Indexer's definition, and delay the job's spacing
	// above the Node's floor.
	def   *cardigann.Definition
	delay time.Duration
	// solver is the Node's Challenge solver, nil for none.
	solver *Solver
	svc    *Service
	run    *run
	slot   *slot
}

// Purpose is what a request fetches, which sets its body cap.
type Purpose uint8

const (
	// Read is a page or a feed.
	Read Purpose = iota + 1
	// Download is a .torrent.
	Download
)

const readLimit = 8 << 20

func (p Purpose) limit() int64 {
	if p == Download {
		return bencode.MaxTorrent
	}
	return readLimit
}

// Response is a response Do hands the driver to read.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
	// URL is where the response came from, after any redirect.
	URL *url.URL
	// Magnet is the magnet link the request redirected to, zero otherwise.
	Magnet Link
	// Cookies is whether the request carried any of the session's cookies.
	Cookies bool
	// SetCookies are the cookies each redirect Do followed set, then the
	// response's own.
	SetCookies []*http.Cookie
}

// Send is how Do sends one request. The zero Send reads a page and
// follows redirects to any host.
type Send struct {
	Purpose Purpose
	// NoFollow returns a redirect as the response.
	NoFollow bool
	// Hosts are the Indexer's own hosts, as a URL's Host names them. When
	// set, a redirect to any other host is returned as the response, so a
	// request's headers and body never reach another host.
	Hosts []string
	// Rotates reports the cookies a response sets as a newly earned
	// session, which the Answer carries.
	Rotates bool
}

func (o Send) redirect(req *http.Request, via []*http.Request) error {
	if o.NoFollow || o.Hosts != nil && !slices.Contains(o.Hosts, strings.ToLower(req.URL.Host)) {
		return http.ErrUseLastResponse
	}
	return redirects(req, via)
}

// Do sends req as o says through this Node's transport with the Indexer's
// session on this Node: its cookies, each only to its own host, and its
// User-Agent. A cookie a response sets joins the session. It waits for
// the host's turn before the request and before each redirect it
// follows, caps the body, follows at most 10 redirects, stopping at one
// to a magnet link, and turns a browser challenge into *Stopped, a
// 429 into *Limited and a 5xx, a timeout or a connection failure into
// *Failing. It returns any other response for the driver to read.
// Credentials go only where the driver puts them, and neither its errors
// nor its log lines carry the URL.
func (s *Site) Do(ctx context.Context, req *http.Request, o Send) (*Response, error) {
	held := s.slot.held()
	j := newJar(nil)
	if held != nil {
		j = newJar(held.Cookies)
	}
	res, err := s.send(ctx, req, o, j)
	if cookies, changed := j.held(); changed && held != nil {
		s.slot.keep(held, cookies, o.Rotates)
	}
	return res, err
}

// Space keeps this job's requests to each host at least d apart, or the
// Node's floor when that is longer.
func (s *Site) Space(d time.Duration) { s.delay = max(s.delay, d) }

// Held is the Indexer's session on this Node, nil for none.
func (s *Site) Held() *Session { return s.slot.held() }

// send is Do with the cookies of j. A browser challenge goes to the
// Node's Challenge solver, and the request goes again once, with the
// clearance.
func (s *Site) send(ctx context.Context, req *http.Request, o Send, j *jar) (*Response, error) {
	res, err := s.sendOnce(ctx, req, o, j)
	if !errors.Is(err, errChallenged) {
		return res, err
	}
	if err := s.clear(ctx, req.URL); err != nil {
		return nil, err
	}
	again := req.Clone(ctx)
	if req.GetBody != nil {
		if again.Body, err = req.GetBody(); err != nil {
			return nil, &Failing{Err: fmt.Errorf("build the request again: %w", err)}
		}
	}
	res, err = s.sendOnce(ctx, again, o, j)
	if errors.Is(err, errChallenged) {
		return nil, &Stopped{Cause: ChallengeUnsolved, SolverTried: true}
	}
	return res, err
}

// sendOnce sends req once with the cookies of j and the User-Agent of
// the session, or the clearance this Node holds for its host, with that
// clearance's cookies. A browser challenge is errChallenged.
func (s *Site) sendOnce(ctx context.Context, req *http.Request, o Send, j *jar) (*Response, error) {
	if err := s.svc.spacer.wait(ctx, req.URL.Host, s.delay); err != nil {
		return nil, err
	}
	s.run.start()
	ctx, cancel := context.WithTimeout(ctx, s.svc.cfg.Timings.Request)
	defer cancel()
	cookies := len(j.Cookies(req.URL)) > 0
	if held := s.slot.held(); held != nil && held.UserAgent != "" {
		req.Header.Set("User-Agent", held.UserAgent)
	}
	if c, ok := s.svc.cleared.of(req.URL.Host); ok {
		j.keep(c.cookies)
		req.Header.Set("User-Agent", c.userAgent)
	}
	var hops []*http.Cookie
	client := *s.svc.client
	client.Jar = j
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if err := o.redirect(next, via); err != nil {
			return err
		}
		hops = append(hops, next.Response.Cookies()...)
		return s.svc.spacer.wait(next.Context(), next.URL.Host, s.delay)
	}
	p := o.Purpose
	sent := time.Now()
	res, err := client.Do(req.WithContext(ctx)) //nolint:gosec // G704: an Indexer's requests go to the address its Admin chose
	if err != nil {
		err = WithoutURL(err)
		s.svc.cfg.Log.Debug("A request to an Indexer failed", "indexer", s.Indexer, "method", req.Method, "host", req.URL.Host, "err", err.Error())
		return nil, &Failing{Err: fmt.Errorf("reach the Indexer: %w", err)}
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, p.limit()+1))
	if err != nil {
		return nil, &Failing{Err: fmt.Errorf("read the Indexer's response: %w", WithoutURL(err))}
	}
	s.svc.cfg.Log.Debug("An Indexer responded", "indexer", s.Indexer, "method", req.Method, "host", req.URL.Host,
		"status", res.StatusCode, "bytes", len(body), "took", time.Since(sent).String())
	switch {
	case int64(len(body)) > p.limit():
		return nil, &Failing{Err: fmt.Errorf("the Indexer's response is over %d bytes", p.limit())}
	case challenged(res, body):
		return nil, errChallenged
	case res.StatusCode == http.StatusTooManyRequests:
		return nil, &Limited{RetryAfter: retryAfter(res.Header.Get("Retry-After"), time.Now())}
	case res.StatusCode >= http.StatusInternalServerError:
		return nil, &Failing{Err: fmt.Errorf("the Indexer responded with status %d", res.StatusCode)}
	}
	out := &Response{
		Status: res.StatusCode, Header: res.Header, Body: body, URL: res.Request.URL, Cookies: cookies,
		SetCookies: append(hops, res.Cookies()...),
	}
	if to := res.Header.Get("Location"); isRedirect(res.StatusCode) && magnet.Is(to) {
		out.Magnet = LinkOf(to)
	}
	return out, nil
}

// redirects follows a redirect to any host, and stops at one to a magnet
// link, which Do returns.
func redirects(req *http.Request, via []*http.Request) error {
	if magnet.Is(req.URL.String()) {
		return http.ErrUseLastResponse
	}
	if len(via) >= 10 {
		return errors.New("the Indexer redirected 10 times")
	}
	return nil
}

func isRedirect(status int) bool {
	return status >= http.StatusMultipleChoices && status < http.StatusBadRequest
}

// retryAfter reads a Retry-After header of seconds or an HTTP date, zero
// when there is none.
func retryAfter(v string, now time.Time) time.Duration {
	if secs, err := strconv.Atoi(v); err == nil {
		return max(time.Duration(secs)*time.Second, 0)
	}
	if at, err := http.ParseTime(v); err == nil {
		return max(at.Sub(now), 0)
	}
	return 0
}

// WithoutURL drops the request URL from err, since it may carry a key.
func WithoutURL(err error) error {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		return ue.Err
	}
	return err
}

// spacer spaces the requests to each host by Floor, or a job's longer
// spacing, across every Indexer on this Node.
type spacer struct {
	floor time.Duration
	mu    sync.Mutex
	next  map[string]time.Time
}

// wait waits for host's turn and takes it, keeping the next request at
// least delay or the floor after it.
func (s *spacer) wait(ctx context.Context, host string, delay time.Duration) error {
	s.mu.Lock()
	at := time.Now()
	if next := s.next[host]; next.After(at) {
		at = next
	}
	s.next[host] = at.Add(max(s.floor, delay))
	s.mu.Unlock()
	t := time.NewTimer(time.Until(at))
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for the host's turn: %w", ctx.Err())
	}
}

// fetch downloads link from wherever it is, with the session's cookies
// for its host.
func fetch(ctx context.Context, s *Site, link Link) (Got, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link.raw, http.NoBody)
	if err != nil {
		return nil, &Failing{Err: fmt.Errorf("build the request: %w", WithoutURL(err))}
	}
	res, err := s.Do(ctx, req, Send{Purpose: Download})
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

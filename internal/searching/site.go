package searching

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
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
	svc        *Service
	run        *run
	slot       *slot
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
	// Magnet is the magnet link the request redirected to, zero otherwise.
	Magnet Link
}

// Do sends req through this Node's transport. It waits for the host's
// turn, caps the body, follows at most 10 redirects to any host, stopping
// at one to a magnet link, and turns a 429 into *Limited and a 5xx, a
// timeout or a connection failure into *Failing. It returns any other
// response for the driver to read. Credentials go only where the driver
// puts them, cookies only to their own hosts, and neither its errors nor
// its log lines carry the URL.
func (s *Site) Do(ctx context.Context, req *http.Request, p Purpose) (*Response, error) {
	if err := s.svc.spacer.wait(ctx, req.URL.Host); err != nil {
		return nil, err
	}
	s.run.start()
	ctx, cancel := context.WithTimeout(ctx, s.svc.cfg.Timings.Request)
	defer cancel()
	sent := time.Now()
	res, err := s.svc.client.Do(req.WithContext(ctx))
	if err != nil {
		err = withoutURL(err)
		s.svc.cfg.Log.Debug("A request to an Indexer failed", "indexer", s.Indexer, "method", req.Method, "host", req.URL.Host, "err", err.Error())
		return nil, &Failing{Err: fmt.Errorf("reach the Indexer: %w", err)}
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, p.limit()+1))
	if err != nil {
		return nil, &Failing{Err: fmt.Errorf("read the Indexer's response: %w", withoutURL(err))}
	}
	s.svc.cfg.Log.Debug("An Indexer responded", "indexer", s.Indexer, "method", req.Method, "host", req.URL.Host,
		"status", res.StatusCode, "bytes", len(body), "took", time.Since(sent).String())
	switch {
	case int64(len(body)) > p.limit():
		return nil, &Failing{Err: fmt.Errorf("the Indexer's response is over %d bytes", p.limit())}
	case res.StatusCode == http.StatusTooManyRequests:
		return nil, &Limited{RetryAfter: retryAfter(res.Header.Get("Retry-After"), time.Now())}
	case res.StatusCode >= http.StatusInternalServerError:
		return nil, &Failing{Err: fmt.Errorf("the Indexer responded with status %d", res.StatusCode)}
	}
	out := &Response{Status: res.StatusCode, Header: res.Header, Body: body}
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

// withoutURL drops the request URL from err, since it may carry a key.
func withoutURL(err error) error {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		return ue.Err
	}
	return err
}

// spacer spaces the requests to each host by Floor, across every Indexer
// on this Node.
type spacer struct {
	floor time.Duration
	mu    sync.Mutex
	next  map[string]time.Time
}

// wait waits for host's turn and takes it.
func (s *spacer) wait(ctx context.Context, host string) error {
	s.mu.Lock()
	at := time.Now()
	if next := s.next[host]; next.After(at) {
		at = next
	}
	s.next[host] = at.Add(s.floor)
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

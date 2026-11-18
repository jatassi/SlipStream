package calls_test

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/calls"
)

var timings = calls.Timings{Attempts: 3, Try: 30 * time.Second, RetryMin: time.Second, RetryMax: 8 * time.Second, Wait: 10 * time.Second}

const key = "secret-key"

var keys = calls.Credentials{Keys: map[calls.ID]string{"subs": key}}

// server is an HTTP server on an in-memory network, so it responds inside
// a synctest bubble, with every request it received and the fake time it
// arrived.
type server struct {
	base   *url.URL
	client *http.Client

	mu   sync.Mutex
	reqs []*http.Request
	at   []time.Time
}

func serve(t *testing.T, h func(n int, w http.ResponseWriter, r *http.Request)) *server {
	t.Helper()
	network := fault.NewNetwork()
	l, err := network.Host(netip.MustParseAddr("10.0.0.1")).Listen("tcp", "10.0.0.1:80")
	if err != nil {
		t.Fatal(err)
	}
	s := &server{base: &url.URL{Scheme: "http", Host: "10.0.0.1:80"}}
	srv := &http.Server{ReadHeaderTimeout: time.Minute, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.reqs, s.at = append(s.reqs, r), append(s.at, time.Now())
		n := len(s.reqs)
		s.mu.Unlock()
		h(n, w, r)
	})}
	go func() { _ = srv.Serve(l) }()
	tr := &http.Transport{DialContext: network.Host(netip.MustParseAddr("10.0.0.2")).DialContext}
	t.Cleanup(func() {
		tr.CloseIdleConnections()
		_ = srv.Close()
	})
	s.client = &http.Client{Transport: tr}
	return s
}

func (s *server) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.reqs)
}

func (s *server) lines(c *http.Client) *calls.Lines {
	return calls.NewLines(c, "SlipStream/test", timings, rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
}

func (s *server) get(path string) calls.Request {
	return calls.Request{URL: s.base.JoinPath(path)}
}

func failed(t *testing.T, err error) calls.Outcome {
	t.Helper()
	f, ok := errors.AsType[*calls.Failed](err)
	if !ok {
		t.Fatalf("err = %v, want a *calls.Failed", err)
	}
	return f.Outcome
}

func TestAHeaderKeyIsSentAndNeverReachesAURLOrAnError(t *testing.T) {
	subs := calls.Provider{
		ID: "subs", Label: "Subs", Rate: 20, Burst: 20,
		Header: func(h http.Header, key string) { h.Set("X-API-Key", key) },
	}
	synctest.Test(t, func(t *testing.T) {
		s := serve(t, func(_ int, w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
		if _, err := s.lines(s.client).Line(subs).Do(t.Context(), keys, s.get("/search")); err != nil {
			t.Fatal(err)
		}
		r := s.reqs[0]
		if r.Header.Get("X-API-Key") != key || strings.Contains(r.URL.String(), key) {
			t.Fatalf("sent %s with X-API-Key %q, want the key in the header alone", r.URL, r.Header.Get("X-API-Key"))
		}
	})
	t.Run("unreachable", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			down := &http.Client{Transport: &http.Transport{
				DialContext: func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("connection refused") },
			}}
			lines := calls.NewLines(down, "SlipStream/test", timings, rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
			_, err := lines.Line(subs).Do(t.Context(), keys, calls.Request{URL: &url.URL{Scheme: "http", Host: "subs.test", Path: "/search"}})
			if o := failed(t, err); o.Kind != calls.Unreachable || strings.Contains(err.Error(), key) || !strings.HasPrefix(o.Words, "didn't respond") {
				t.Fatalf("err = %q, want unreachable words without the key", err)
			}
		})
	})
}

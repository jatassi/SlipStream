package metadata_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/works"
)

var timings = metadata.Timings{Attempts: 3, Try: 30 * time.Second, RetryMin: time.Second, RetryMax: 8 * time.Second, Wait: 10 * time.Second, CacheEntries: 16}

func seeded() *rand.Rand {
	return rand.New(rand.NewPCG(1, 2)) //nolint:gosec // a fixed draw the test replays
}

var tmdb = metadata.Provider{
	ID: "tmdb", Label: "TMDB", Rate: 20, Burst: 20,
	Key: func(q url.Values, key string) { q.Set("api_key", key) },
}

// sent is every request a server received, in order, with the fake time
// it arrived.
type sent struct {
	mu   sync.Mutex
	reqs []*http.Request
	at   []time.Time
}

func (s *sent) add(r *http.Request) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reqs, s.at = append(s.reqs, r), append(s.at, time.Now())
	return len(s.reqs)
}

func (s *sent) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.reqs)
}

// serve runs h on an httptest server over an in-memory network, so it
// answers inside a synctest bubble, and returns Lines that reach it.
func serve(t *testing.T, h func(n int, w http.ResponseWriter, r *http.Request)) (*metadata.Lines, *url.URL, *sent) {
	t.Helper()
	var s sent
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h(s.add(r), w, r) }))
	_ = srv.Listener.Close()
	l := &pipeListener{conns: make(chan net.Conn), closed: make(chan struct{})}
	srv.Listener = l
	srv.Start()
	tr := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) { return l.dial(ctx) }}
	t.Cleanup(func() {
		tr.CloseIdleConnections()
		srv.Close()
	})
	base, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return metadata.NewLines(&http.Client{Transport: tr}, "SlipStream/test", timings, seeded()), base, &s
}

type pipeListener struct {
	conns  chan net.Conn
	closed chan struct{}
	once   sync.Once
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *pipeListener) Addr() net.Addr {
	return net.TCPAddrFromAddrPort(netip.MustParseAddrPort("127.0.0.1:80"))
}

func (l *pipeListener) dial(ctx context.Context) (net.Conn, error) {
	server, client := fault.Pipe(l.Addr(), l.Addr())
	select {
	case l.conns <- server:
		return client, nil
	case <-l.closed:
		return nil, net.ErrClosed
	case <-ctx.Done():
		return nil, fmt.Errorf("in-memory dial: %w", ctx.Err())
	}
}

func at(base *url.URL, path string, q url.Values) *url.URL {
	u := base.JoinPath(path)
	u.RawQuery = q.Encode()
	return u
}

func ok(w http.ResponseWriter, n int) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, `{"n":%d}`, n)
}

type body struct {
	N int `json:"n"`
}

func sourceError(t *testing.T, err, want error) *works.SourceError {
	t.Helper()
	se, isSource := errors.AsType[*works.SourceError](err)
	if !isSource || !errors.Is(err, want) {
		t.Fatalf("err = %v, want a SourceError wrapping %v", err, want)
	}
	return se
}

func TestLineHonoursRetryAfterInsideTheBudget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lines, base, s := serve(t, func(n int, w http.ResponseWriter, _ *http.Request) {
			if n == 1 {
				w.Header().Set("Retry-After", "4")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			ok(w, n)
		})
		ask, outcomes := metadata.Tallied(metadata.Ask{})
		var b body
		if err := lines.Line(tmdb).Get(t.Context(), ask, metadata.Request{URL: at(base, "/movie/1", nil)}, &b); err != nil || b.N != 2 {
			t.Fatalf("Get = %+v, %v; want the second answer", b, err)
		}
		if s.count() != 2 || s.at[1].Sub(s.at[0]) != 4*time.Second {
			t.Fatalf("sent at %v, want two requests 4s apart", s.at)
		}
		if o := outcomes(); len(o) != 1 || o[0].Kind != metadata.OK {
			t.Fatalf("outcomes = %+v, want one OK", o)
		}
	})
}

func TestLineMarksLimitedWhenRetryAfterOutlastsTheBudget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lines, base, s := serve(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
		})
		line := lines.Line(tmdb)
		until := time.Now().Add(time.Minute)
		ask, outcomes := metadata.Tallied(metadata.Ask{})
		err := line.Get(t.Context(), ask, metadata.Request{URL: at(base, "/movie/1", nil)}, &body{})
		if se := sourceError(t, err, works.ErrUnavailable); !se.Until.Equal(until) || se.Source != "TMDB" {
			t.Fatalf("err = %+v, want TMDB unavailable until %v", se, until)
		}
		if o := outcomes(); len(o) != 1 || o[0].Kind != metadata.Limited || !o[0].Until.Equal(until) || o[0].Provider != "tmdb" {
			t.Fatalf("outcomes = %+v, want one Limited until %v", o, until)
		}
		time.Sleep(30 * time.Second)
		err = line.Get(t.Context(), metadata.Ask{}, metadata.Request{URL: at(base, "/movie/2", nil)}, &body{})
		if se := sourceError(t, err, works.ErrUnavailable); !se.Until.Equal(until) || s.count() != 1 {
			t.Fatalf("a call inside the limit = %v after %d requests, want limited without sending", err, s.count())
		}
	})
}

func TestLineRetriesTimeoutsAnd5xxThreeTimesWithJitter(t *testing.T) {
	r := seeded()
	first, second := backoff.Delay(time.Second, 8*time.Second, 0, r), backoff.Delay(time.Second, 8*time.Second, 1, r)
	if first == time.Second || second == 2*time.Second {
		t.Fatalf("the seeded draws %v, %v carry no jitter", first, second)
	}
	t.Run("5xx", func(t *testing.T) { synctest.Test(t, func(t *testing.T) { retries5xx(t, first, second) }) })
	t.Run("timeout", func(t *testing.T) { synctest.Test(t, func(t *testing.T) { retriesTimeouts(t, first, second) }) })
}

func retries5xx(t *testing.T, first, second time.Duration) {
	lines, base, s := serve(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	ask, outcomes := metadata.Tallied(metadata.Ask{})
	err := lines.Line(tmdb).Get(t.Context(), ask, metadata.Request{URL: at(base, "/movie/1", nil)}, &body{})
	if se := sourceError(t, err, works.ErrUnavailable); se.Words != "responded with 503 Service Unavailable" {
		t.Fatalf("words = %q", se.Words)
	}
	if s.count() != 3 || s.at[1].Sub(s.at[0]) != first || s.at[2].Sub(s.at[1]) != second {
		t.Fatalf("sent at %v, want three tries %v then %v apart", s.at, first, second)
	}
	if o := outcomes(); len(o) != 1 || o[0].Kind != metadata.Unreachable {
		t.Fatalf("outcomes = %+v, want one Unreachable", o)
	}
}

func retriesTimeouts(t *testing.T, first, second time.Duration) {
	lines, base, s := serve(t, func(n int, w http.ResponseWriter, r *http.Request) {
		if n < 3 {
			<-r.Context().Done()
			return
		}
		ok(w, n)
	})
	var b body
	if err := lines.Line(tmdb).Get(t.Context(), metadata.Ask{}, metadata.Request{URL: at(base, "/movie/1", nil)}, &b); err != nil || b.N != 3 {
		t.Fatalf("Get = %+v, %v; want the third try's answer", b, err)
	}
	if s.at[1].Sub(s.at[0]) != timings.Try+first || s.at[2].Sub(s.at[1]) != timings.Try+second {
		t.Fatalf("sent at %v, want each retry a timeout plus %v then %v later", s.at, first, second)
	}
}

func TestLineClassifiesOKNotFoundRefusedLimitedUnreachable(t *testing.T) {
	omdb := metadata.Provider{
		ID: "omdb", Label: "OMDb", Rate: 2, Burst: 2,
		Classify: func(status int, _ http.Header, b []byte, now time.Time) (metadata.Outcome, bool) {
			if status == http.StatusOK && strings.Contains(string(b), "Request limit reached!") {
				return metadata.Outcome{Kind: metadata.Capped, Until: now.Add(time.Hour), Words: "is out of its daily allowance"}, true
			}
			return metadata.Outcome{}, false
		},
	}
	for _, c := range []classification{
		{"ok", tmdb, http.StatusOK, `{}`, true, metadata.OK, nil},
		{"lookup 404", tmdb, http.StatusNotFound, `{}`, true, metadata.NotFound, works.ErrNotFound},
		{"search 404", tmdb, http.StatusNotFound, `{}`, false, metadata.Unreachable, works.ErrUnavailable},
		{"401", tmdb, http.StatusUnauthorized, `{"status_code":7}`, true, metadata.Refused, works.ErrRefused},
		{"403", tmdb, http.StatusForbidden, `{}`, true, metadata.Refused, works.ErrRefused},
		{"429", tmdb, http.StatusTooManyRequests, `{}`, true, metadata.Limited, works.ErrUnavailable},
		{"400", tmdb, http.StatusBadRequest, `{}`, true, metadata.Unreachable, works.ErrUnavailable},
		{"classified cap", omdb, http.StatusOK, `{"Response":"False","Error":"Request limit reached!"}`, true, metadata.Capped, works.ErrUnavailable},
		{"unclassified ok", omdb, http.StatusOK, `{}`, true, metadata.OK, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { classifies(t, c) })
		})
	}
	t.Run("unreachable", func(t *testing.T) { synctest.Test(t, classifiesUnreachable) })
}

type classification struct {
	name     string
	provider metadata.Provider
	status   int
	body     string
	lookup   bool
	kind     metadata.OutcomeKind
	err      error
}

func classifies(t *testing.T, c classification) {
	lines, base, _ := serve(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(c.status)
		_, _ = w.Write([]byte(c.body))
	})
	ask, outcomes := metadata.Tallied(metadata.Ask{})
	err := lines.Line(c.provider).Get(t.Context(), ask, metadata.Request{URL: at(base, "/x", nil), Lookup: c.lookup}, &struct{}{})
	if c.err == nil && err != nil {
		t.Fatalf("Get = %v, want nil", err)
	}
	if c.err != nil {
		_ = sourceError(t, err, c.err)
	}
	if o := outcomes(); len(o) != 1 || o[0].Kind != c.kind || o[0].Provider != c.provider.ID || !o[0].At.Equal(time.Now()) {
		t.Fatalf("outcomes = %+v, want one %d from %s", o, c.kind, c.provider.ID)
	}
}

func classifiesUnreachable(t *testing.T) {
	lines := metadata.NewLines(&http.Client{Transport: &http.Transport{
		DialContext: func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("connection refused") },
	}}, "SlipStream/test", timings, seeded())
	ask, outcomes := metadata.Tallied(metadata.NewAsk(metadata.AskOptions{Keys: map[metadata.ProviderID]string{"tmdb": "secret-key"}}))
	err := lines.Line(tmdb).Get(t.Context(), ask, metadata.Request{URL: &url.URL{Scheme: "http", Host: "tmdb.test", Path: "/x"}}, &struct{}{})
	se := sourceError(t, err, works.ErrUnavailable)
	if strings.Contains(err.Error(), "secret-key") || !strings.HasPrefix(se.Words, "didn't respond") {
		t.Fatalf("err = %q, want unreachable words without the key", err)
	}
	if o := outcomes(); len(o) != 1 || o[0].Kind != metadata.Unreachable {
		t.Fatalf("outcomes = %+v, want one Unreachable", o)
	}
}

func TestTheBucketHoldsEachProviderToItsRate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lines, base, s := serve(t, func(n int, w http.ResponseWriter, _ *http.Request) { ok(w, n) })
		slow := metadata.Provider{ID: "slow", Label: "Slow", Rate: 2, Burst: 1}
		fast := metadata.Provider{ID: "fast", Label: "Fast", Rate: 20, Burst: 1}
		start := time.Now()
		var wg sync.WaitGroup
		for _, p := range []metadata.Provider{slow, fast} {
			line := lines.Line(p)
			wg.Go(func() {
				for range 5 {
					if err := line.Get(t.Context(), metadata.Ask{}, metadata.Request{URL: at(base, "/"+string(p.ID), nil)}, &body{}); err != nil {
						t.Error(err)
					}
				}
			})
		}
		wg.Wait()
		last := map[string]time.Duration{}
		for i, r := range s.reqs {
			last[r.URL.Path] = s.at[i].Sub(start)
		}
		near := func(got, want time.Duration) bool { return (got - want).Abs() < time.Millisecond }
		if !near(last["/slow"], 2*time.Second) || !near(last["/fast"], 200*time.Millisecond) {
			t.Fatalf("five calls each ended at %v, want slow at 2s and fast at 200ms", last)
		}
	})
}

func TestACappedProviderIsNotCalledUntilItsTime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		midnight := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
		omdb := metadata.Provider{
			ID: "omdb", Label: "OMDb", Rate: 2, Burst: 2,
			Classify: func(_ int, _ http.Header, b []byte, _ time.Time) (metadata.Outcome, bool) {
				if strings.Contains(string(b), "Request limit reached!") {
					return metadata.Outcome{Kind: metadata.Capped, Until: midnight, Words: "is out of its daily allowance"}, true
				}
				return metadata.Outcome{}, false
			},
		}
		lines, base, s := serve(t, func(n int, w http.ResponseWriter, _ *http.Request) {
			if n == 1 {
				_, _ = w.Write([]byte(`{"Response":"False","Error":"Request limit reached!"}`))
				return
			}
			ok(w, n)
		})
		line := lines.Line(omdb)
		get := func() ([]metadata.Outcome, error) {
			ask, outcomes := metadata.Tallied(metadata.Ask{})
			err := line.Get(t.Context(), ask, metadata.Request{URL: at(base, "/", url.Values{"i": {"tt1"}})}, &body{})
			return outcomes(), err
		}
		if o, err := get(); len(o) != 1 || o[0].Kind != metadata.Capped || sourceError(t, err, works.ErrUnavailable).Until != midnight {
			t.Fatalf("first call = %+v, %v; want Capped until %v", o, err, midnight)
		}
		time.Sleep(time.Hour)
		if o, err := get(); s.count() != 1 || len(o) != 1 || o[0].Kind != metadata.Capped || sourceError(t, err, works.ErrUnavailable).Words != "is out of its daily allowance" {
			t.Fatalf("a call before midnight = %+v, %v after %d requests; want Capped without sending", o, err, s.count())
		}
		time.Sleep(time.Until(midnight))
		if o, err := get(); err != nil || s.count() != 2 || o[0].Kind != metadata.OK {
			t.Fatalf("a call at midnight = %+v, %v after %d requests; want it sent and OK", o, err, s.count())
		}
	})
}

func TestCacheKeysNeverHoldTheAPIKey(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lines, base, s := serve(t, func(n int, w http.ResponseWriter, _ *http.Request) { ok(w, n) })
		line := lines.Line(tmdb)
		keyed := func(key string) metadata.Ask {
			return metadata.Cached(metadata.NewAsk(metadata.AskOptions{Keys: map[metadata.ProviderID]string{"tmdb": key}}), time.Hour, 1)
		}
		req := metadata.Request{URL: at(base, "/movie/1", url.Values{"language": {"en-US"}}), Cache: true}
		var b body
		if err := line.Get(t.Context(), keyed("first-key"), req, &b); err != nil || s.reqs[0].URL.Query().Get("api_key") != "first-key" {
			t.Fatalf("Get = %v, sent %v; want the key on the request", err, s.reqs[0].URL)
		}
		if err := line.Get(t.Context(), keyed("second-key"), req, &b); err != nil || b.N != 1 || s.count() != 1 {
			t.Fatalf("Get with another key = %+v, %v after %d requests; want the cached answer", b, err, s.count())
		}
	})
}

func TestCacheDropsOnAnotherGeneration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lines, base, _ := serve(t, func(n int, w http.ResponseWriter, _ *http.Request) { ok(w, n) })
		line := lines.Line(tmdb)
		req := metadata.Request{URL: at(base, "/movie/1", nil), Cache: true}
		read := func(ask metadata.Ask) int {
			var b body
			if err := line.Get(t.Context(), ask, req, &b); err != nil {
				t.Fatal(err)
			}
			return b.N
		}
		fresh := metadata.NewAsk(metadata.AskOptions{Fresh: true})
		for i, step := range []struct {
			ask  metadata.Ask
			want int
		}{
			{metadata.Cached(metadata.Ask{}, time.Hour, 1), 1},
			{metadata.Cached(metadata.Ask{}, time.Hour, 1), 1},
			{metadata.Cached(metadata.Ask{}, time.Hour, 2), 2},
			{metadata.Cached(fresh, time.Hour, 2), 3},
			{metadata.Cached(metadata.Ask{}, time.Hour, 2), 3},
			{metadata.Cached(metadata.Ask{}, 0, 2), 4},
		} {
			if got := read(step.ask); got != step.want {
				t.Fatalf("step %d read answer %d, want %d", i, got, step.want)
			}
		}
	})
}

func TestTheCacheDropsTheLeastRecentlyUsed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lines, base, s := serve(t, func(n int, w http.ResponseWriter, _ *http.Request) { ok(w, n) })
		line := lines.Line(tmdb)
		ask := metadata.Cached(metadata.Ask{}, time.Hour, 1)
		read := func(i int) {
			if err := line.Get(t.Context(), ask, metadata.Request{URL: at(base, fmt.Sprintf("/movie/%d", i), nil), Cache: true}, &body{}); err != nil {
				t.Fatal(err)
			}
		}
		for i := range timings.CacheEntries {
			read(i)
		}
		read(0)
		read(timings.CacheEntries)
		read(0)
		if s.count() != timings.CacheEntries+1 {
			t.Fatalf("%d requests, want the recently read entry kept", s.count())
		}
		read(1)
		if s.count() != timings.CacheEntries+2 {
			t.Fatalf("%d requests, want the least recently read entry dropped", s.count())
		}
	})
}

func TestLineRefusesAnAnswerOver16MiB(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lines, base, _ := serve(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(make([]byte, metadata.MaxBody+1))
		})
		_, err := lines.Line(metadata.Provider{ID: "cdn", Label: "Images", Rate: 20, Burst: 20}).Bytes(t.Context(), metadata.Ask{}, at(base, "/a.png", nil))
		if se := sourceError(t, err, works.ErrUnavailable); se.Words != "sent a response over 16 MiB" {
			t.Fatalf("words = %q", se.Words)
		}
	})
}

func TestASecondLineForAProviderPanics(t *testing.T) {
	lines := metadata.NewLines(&http.Client{}, "SlipStream/test", timings, seeded())
	lines.Line(tmdb)
	defer func() {
		if recover() == nil {
			t.Fatal("a second Line for tmdb did not panic")
		}
	}()
	lines.Line(tmdb)
}

// TestACappedKeyHoldsBackNoOtherKey spends one key's allowance: calls with
// it wait for the reset, and a call with another key goes out.
func TestACappedKeyHoldsBackNoOtherKey(t *testing.T) {
	midnight := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
	omdb := metadata.Provider{
		ID: "omdb", Label: "OMDb", Rate: 2, Burst: 2,
		Key: func(q url.Values, key string) { q.Set("apikey", key) },
		Classify: func(_ int, _ http.Header, b []byte, _ time.Time) (metadata.Outcome, bool) {
			if strings.Contains(string(b), "Request limit reached!") {
				return metadata.Outcome{Kind: metadata.Capped, Until: midnight, Words: "is out of its daily allowance"}, true
			}
			return metadata.Outcome{}, false
		},
	}
	lines, base, s := serve(t, func(n int, w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("apikey") == "spent" {
			_, _ = w.Write([]byte(`{"Response":"False","Error":"Request limit reached!"}`))
			return
		}
		ok(w, n)
	})
	line := lines.Line(omdb)
	get := func(key string) error {
		ask := metadata.NewAsk(metadata.AskOptions{Keys: map[metadata.ProviderID]string{"omdb": key}})
		return line.Get(t.Context(), ask, metadata.Request{URL: at(base, "/", url.Values{"i": {"tt1"}})}, &body{})
	}
	for range 2 {
		if err := get("spent"); !errors.Is(err, works.ErrUnavailable) {
			t.Fatalf("a call with the spent key = %v, want ErrUnavailable", err)
		}
	}
	if s.count() != 1 {
		t.Fatalf("%d calls went out with the spent key, want 1", s.count())
	}
	if err := get("fresh"); err != nil || s.count() != 2 {
		t.Fatalf("a call with another key = %v after %d requests, want it sent and answered", err, s.count())
	}
}

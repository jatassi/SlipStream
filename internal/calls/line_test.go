package calls_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
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

// busy reads a body such as SubDL's 429 {"error":"service_busy"}, which
// asks for 5 tries 5 s apart, and SubSource's {"retryAfter":4}.
func busy(a calls.Answer, now time.Time) (calls.Verdict, bool) {
	var b struct {
		Error      string `json:"error"`
		RetryAfter int    `json:"retryAfter"`
	}
	if a.Status != http.StatusTooManyRequests || json.Unmarshal(a.Body, &b) != nil {
		return calls.Verdict{}, false
	}
	if b.Error == "service_busy" {
		return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Unreachable, Words: "is busy"}, Retry: &calls.Schedule{Every: 5 * time.Second, Tries: 5}}, true
	}
	return calls.Verdict{Outcome: calls.Outcome{Kind: calls.Limited, Until: now.Add(time.Duration(b.RetryAfter) * time.Second), Words: "is rate limiting SlipStream"}}, true
}

type scheduled struct {
	name   string
	body   string
	busyN  int
	sent   int
	apart  time.Duration
	failed bool
}

func TestARetryScheduleIsTakenFromA429Body(t *testing.T) {
	for _, c := range []scheduled{
		{"retry after in the body", `{"retryAfter":4}`, 1, 2, 4 * time.Second, false},
		{"busy, then answered on the fifth try", `{"error":"service_busy"}`, 4, 5, 5 * time.Second, false},
		{"busy past the schedule", `{"error":"service_busy"}`, 10, 5, 5 * time.Second, true},
	} {
		t.Run(c.name, func(t *testing.T) { synctest.Test(t, func(t *testing.T) { followsSchedule(t, c) }) })
	}
}

func followsSchedule(t *testing.T, c scheduled) {
	subs := calls.Provider{ID: "subs", Label: "Subs", Rate: 20, Burst: 20, Classify: busy}
	s := serve(t, func(n int, w http.ResponseWriter, _ *http.Request) {
		if n <= c.busyN {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(c.body))
		}
	})
	_, err := s.lines(s.client).Line(subs).Do(t.Context(), calls.Credentials{}, s.get("/search"))
	if c.failed != (err != nil) {
		t.Fatalf("Do = %v, want failed %v", err, c.failed)
	}
	if s.count() != c.sent {
		t.Fatalf("sent %d requests, want %d", s.count(), c.sent)
	}
	for i := 1; i < len(s.at); i++ {
		if d := s.at[i].Sub(s.at[i-1]); d != c.apart {
			t.Fatalf("sent at %v, want each try %v apart", s.at, c.apart)
		}
	}
}

func TestAnOKAnswerComesBackWithItsHeaders(t *testing.T) {
	subs := calls.Provider{
		ID: "subs", Label: "Subs", Rate: 20, Burst: 20,
		Classify: func(a calls.Answer, _ time.Time) (calls.Verdict, bool) {
			left, err := strconv.Atoi(a.Header.Get("X-RateLimit-Remaining"))
			if a.Status != http.StatusOK || err != nil {
				return calls.Verdict{}, false
			}
			return calls.Verdict{Outcome: calls.Outcome{Kind: calls.OK, Allowance: &calls.Allowance{Downloads: &calls.Window{Remaining: left}}}}, true
		},
	}
	synctest.Test(t, func(t *testing.T) {
		s := serve(t, func(_ int, w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/zip")
			w.Header().Set("X-RateLimit-Remaining", "41")
			_, _ = w.Write([]byte("PK"))
		})
		tally := &calls.Tally{}
		a, err := s.lines(s.client).Line(subs).Do(t.Context(), calls.Credentials{Tally: tally}, s.get("/download"))
		if err != nil || a.Status != http.StatusOK || a.Header.Get("Content-Type") != "application/zip" || string(a.Body) != "PK" {
			t.Fatalf("Do = %+v, %v; want the zip with its headers", a, err)
		}
		if o := tally.Outcomes(); len(o) != 1 || o[0].Kind != calls.OK || o[0].Allowance == nil || o[0].Allowance.Downloads.Remaining != 41 {
			t.Fatalf("outcomes = %+v, want one OK reporting 41 downloads left", o)
		}
	})
}

func TestARequestToAHostOffTheListIsNeverSent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := serve(t, func(_ int, w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
		subs := calls.Provider{ID: "subs", Label: "Subs", Rate: 20, Burst: 20, Hosts: []string{s.base.Host}}
		line := s.lines(s.client).Line(subs)
		if _, err := line.Do(t.Context(), keys, s.get("/search")); err != nil {
			t.Fatalf("a request to a listed host = %v, want it sent", err)
		}
		start := time.Now()
		_, err := line.Do(t.Context(), keys, calls.Request{URL: &url.URL{Scheme: "http", Host: "elsewhere.test", Path: "/file.zip"}})
		if o := failed(t, err); o.Kind != calls.Unreachable || err.Error() != "Subs has no host elsewhere.test" {
			t.Fatalf("err = %q, want Subs to have no host elsewhere.test", err)
		}
		if s.count() != 1 || time.Since(start) != 0 {
			t.Fatalf("%d requests after %v, want the second never sent nor tried again", s.count(), time.Since(start))
		}
	})
}

func TestARedirectIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := serve(t, func(_ int, w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://10.0.0.1:80/elsewhere", http.StatusFound)
		})
		subs := calls.Provider{ID: "subs", Label: "Subs", Rate: 20, Burst: 20, Hosts: []string{s.base.Host}}
		_, err := s.lines(calls.NoRedirects(s.client)).Line(subs).Do(t.Context(), keys, s.get("/download"))
		if o := failed(t, err); o.Kind != calls.Unreachable || !strings.Contains(o.Words, "302") {
			t.Fatalf("err = %q, want the 302 itself", err)
		}
		if s.count() != 1 || s.reqs[0].URL.Path != "/download" {
			t.Fatalf("sent %d requests, want only /download", s.count())
		}
		if s.client.CheckRedirect != nil {
			t.Fatal("NoRedirects changed the client it was given")
		}
	})
}

func TestAProvidersOwnAttemptsAndWaitReplaceTheTimings(t *testing.T) {
	t.Run("attempts", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := serve(t, func(_ int, w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) })
			once := calls.Provider{ID: "once", Label: "Once", Rate: 20, Burst: 20, Attempts: 1}
			_, _ = s.lines(s.client).Line(once).Do(t.Context(), calls.Credentials{}, s.get("/x"))
			if s.count() != 1 {
				t.Fatalf("sent %d requests, want 1", s.count())
			}
		})
	})
	t.Run("wait", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := serve(t, func(n int, w http.ResponseWriter, _ *http.Request) {
				if n == 1 {
					w.Header().Set("Retry-After", "30")
					w.WriteHeader(http.StatusTooManyRequests)
				}
			})
			patient := calls.Provider{ID: "patient", Label: "Patient", Rate: 20, Burst: 20, Wait: time.Minute}
			if _, err := s.lines(s.client).Line(patient).Do(t.Context(), calls.Credentials{}, s.get("/x")); err != nil {
				t.Fatal(err)
			}
			if s.count() != 2 || s.at[1].Sub(s.at[0]) != 30*time.Second {
				t.Fatalf("sent at %v, want a second try 30s later", s.at)
			}
		})
	})
}

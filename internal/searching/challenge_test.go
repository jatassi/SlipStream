package searching_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/searching"
)

const challengePage = "<!DOCTYPE html><html><head><title>Just a moment...</title></head><body>Checking your browser.</body></html>"

// guarded is a Torznab feed behind a browser challenge that lets through
// only a request whose cf_clearance names its own User-Agent.
func guarded(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("cf_clearance"); err != nil || c.Value != "for "+r.UserAgent() {
			w.Header().Set("Server", "cloudflare")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(challengePage))
			return
		}
		_, _ = w.Write([]byte("<rss><channel></channel></rss>"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// solver is a Challenge solver speaking FlareSolverr's API that clears
// any page for its own User-Agent, or answers with failure when set.
type solveCommand struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url"`
	MaxTimeout int64  `json:"maxTimeout"`
}

type solver struct {
	srv       *httptest.Server
	userAgent string
	mu        sync.Mutex
	asked     []solveCommand
	failure   string
}

func newSolver(t *testing.T, userAgent string) *solver {
	t.Helper()
	s := &solver{userAgent: userAgent}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"status":"ok"}`)) })
	mux.HandleFunc("POST /v1", func(w http.ResponseWriter, r *http.Request) {
		var cmd solveCommand
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			t.Errorf("the solver's request: %v", err)
		}
		s.mu.Lock()
		s.asked = append(s.asked, cmd)
		failure := s.failure
		s.mu.Unlock()
		if failure != "" {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": failure})
			return
		}
		host := ""
		if u, err := url.Parse(cmd.URL); err == nil {
			host = u.Hostname()
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "message": "Challenge solved!", "solution": map[string]any{
			"url": cmd.URL, "status": 200, "userAgent": s.userAgent,
			"cookies": []map[string]any{{"name": "cf_clearance", "value": "for " + s.userAgent, "domain": host, "path": "/", "expires": -1}},
		}})
	})
	s.srv = httptest.NewServer(mux)
	t.Cleanup(s.srv.Close)
	return s
}

func (s *solver) times() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.asked)
}

func (s *solver) at(timeout time.Duration) *searching.Solver {
	return &searching.Solver{Address: s.srv.URL, Timeout: timeout}
}

// search runs a search of indexer at address on n with the Node's solver
// and the session the Controller holds for that Node.
func search(t *testing.T, n searching.Remote, indexer, address string, v *searching.Solver, session *searching.Session) searching.Answer {
	t.Helper()
	poll, err := n.Run(t.Context(), searching.Job{
		ID: t.Name() + "/" + indexer + "/" + time.Now().String(), Indexer: indexer, Kind: searching.Torznab, Address: address, APIKey: "k",
		Solver: v, Session: session, Call: searching.SearchCall{Query: searching.Query{Mode: "search", Text: "orchard"}},
	})
	if err != nil || poll.Answer == nil {
		t.Fatalf("Run = %+v, %v; want an answer within the hold", poll, err)
	}
	return *poll.Answer
}

func TestAChallengeIsAChallengeServicesPageWithItsMarker(t *testing.T) {
	for _, tc := range []struct {
		name, server string
		status       int
		body, want   string
	}{
		{"Cloudflare's page", "cloudflare", http.StatusForbidden, challengePage, "*searching.Stopped the site's browser challenge wasn't cleared"},
		{"cloudflare-nginx at 503", "cloudflare-nginx", http.StatusServiceUnavailable, "<title>Attention Required! | Cloudflare</title>", "*searching.Stopped the site's browser challenge wasn't cleared"},
		{"a firewall rule", "cloudflare", http.StatusForbidden, "error code: 1020", "*searching.Stopped the site's browser challenge wasn't cleared"},
		{"DDoS-Guard", "ddos-guard", http.StatusForbidden, "<html><title>DDoS-Guard</title></html>", "*searching.Stopped the site's browser challenge wasn't cleared"},
		{"a page without a marker", "cloudflare", http.StatusForbidden, "<title>Forbidden</title>", "*searching.Failing the Indexer responded with status 403"},
		{"a marker from another server", "nginx", http.StatusForbidden, challengePage, "*searching.Failing the Indexer responded with status 403"},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Server", tc.server)
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		got := outcome(search(t, node(t, &bytes.Buffer{}), "i", srv.URL, nil, nil))
		srv.Close()
		if got != tc.want {
			t.Errorf("%s = %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestTheSolverClearsTheChallengeAndTheNodeKeepsItsClearance(t *testing.T) {
	site := guarded(t)
	seedbox := newSolver(t, "Solver/seedbox")
	n := node(t, &bytes.Buffer{})

	first := search(t, n, "i", site.URL, seedbox.at(45*time.Second), nil)
	if first.Outcome != nil {
		t.Fatalf("a challenged search with a solver = %s, want ok", outcome(first))
	}
	seedbox.mu.Lock()
	asked := seedbox.asked
	seedbox.mu.Unlock()
	if len(asked) != 1 || asked[0].Cmd != "request.get" || !strings.HasPrefix(asked[0].URL, site.URL+"/api?") || asked[0].MaxTimeout != 45000 {
		t.Fatalf("the solver was asked %v, want one request.get of the search's URL with a 45 s maxTimeout", asked)
	}
	s := first.Session
	if s == nil || s.UserAgent != "Solver/seedbox" || len(s.Cookies) != 1 || s.Cookies[0].Name != "cf_clearance" || s.Cookies[0].Value != "for Solver/seedbox" {
		t.Fatalf("the answer's session = %+v, want the clearance and the solver's User-Agent for the Controller to keep", s)
	}

	if a := search(t, n, "i", site.URL, seedbox.at(45*time.Second), s); a.Outcome != nil || seedbox.times() != 1 {
		t.Fatalf("a second search on the Node = %s after %d solves, want ok without the solver", outcome(a), seedbox.times())
	}
	restarted := node(t, &bytes.Buffer{})
	if a := search(t, restarted, "i", site.URL, nil, s); a.Outcome != nil {
		t.Fatalf("a search on the restarted Node with the session the Controller kept = %s, want ok", outcome(a))
	}
	den := node(t, &bytes.Buffer{})
	if got, want := outcome(search(t, den, "i", site.URL, nil, nil)), "*searching.Stopped the site's browser challenge wasn't cleared"; got != want {
		t.Fatalf("a search from another Node with no solver = %s, want %s", got, want)
	}
}

func TestEveryRequestToAClearedHostSendsTheSolversUserAgent(t *testing.T) {
	site := guarded(t)
	var agents []string
	var mu sync.Mutex
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		agents = append(agents, r.UserAgent())
		mu.Unlock()
		_, _ = w.Write([]byte("<rss><channel></channel></rss>"))
	}))
	t.Cleanup(other.Close)
	v := newSolver(t, "Solver/seedbox")
	n := node(t, &bytes.Buffer{})
	if a := search(t, n, "i", site.URL, v.at(time.Minute), nil); a.Outcome != nil {
		t.Fatalf("the first Indexer's search = %s, want ok", outcome(a))
	}
	if a := search(t, n, "j", site.URL, nil, nil); a.Outcome != nil || v.times() != 1 {
		t.Fatalf("another Indexer's search of the cleared host = %s after %d solves, want ok with the Node's clearance", outcome(a), v.times())
	}
	search(t, n, "j", other.URL, nil, nil)
	if len(agents) != 1 || agents[0] == "Solver/seedbox" {
		t.Fatalf("another host got User-Agent %q, want Go's own, since only the cleared host gets the solver's", agents)
	}
}

func TestASolverThatTimesOutFailsAndOneThatMeetsACaptchaStops(t *testing.T) {
	site := guarded(t)
	for _, tc := range []struct{ failure, want string }{
		{"Error: Error solving the challenge. Timeout after 60.0 seconds.", "*searching.Failing the Challenge solver timed out: Error: Error solving the challenge. Timeout after 60.0 seconds."},
		{"Error: Captcha detected but no automatic solver is configured.", "*searching.Stopped the site's browser challenge wasn't cleared"},
	} {
		v := newSolver(t, "Solver/seedbox")
		v.failure = tc.failure
		a := search(t, node(t, &bytes.Buffer{}), "i", site.URL, v.at(time.Minute), nil)
		if got := outcome(a); got != tc.want {
			t.Errorf("a solver answering %q = %s, want %s", tc.failure, got, tc.want)
		}
		if stopped, ok := errors.AsType[*searching.Stopped](a.Outcome); ok && !stopped.SolverTried {
			t.Errorf("a solver answering %q stopped the Indexer without saying the solver tried", tc.failure)
		}
	}

	slow := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	t.Cleanup(slow.Close)
	got := outcome(search(t, node(t, &bytes.Buffer{}), "i", site.URL, &searching.Solver{Address: slow.URL, Timeout: 10 * time.Millisecond}, nil))
	if want := "*searching.Failing reach the Challenge solver: context deadline exceeded"; got != want {
		t.Errorf("a solver past its timeout = %s, want %s", got, want)
	}
	got = outcome(search(t, node(t, &bytes.Buffer{}), "i", site.URL, &searching.Solver{Address: "http://127.0.0.1:1", Timeout: time.Second}, nil))
	if !strings.HasPrefix(got, "*searching.Failing reach the Challenge solver: dial tcp") {
		t.Errorf("an unreachable solver = %s, want Failing", got)
	}

	wrong := newSolver(t, "Solver/other")
	stubborn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Server", "cloudflare")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(challengePage))
	}))
	t.Cleanup(stubborn.Close)
	a := search(t, node(t, &bytes.Buffer{}), "i", stubborn.URL, wrong.at(time.Minute), nil)
	if stopped, ok := errors.AsType[*searching.Stopped](a.Outcome); !ok || stopped.Cause != searching.ChallengeUnsolved || !stopped.SolverTried || wrong.times() != 1 {
		t.Fatalf("a page still challenged after the solver = %s after %d solves, want one solve and a stop the solver tried", outcome(a), wrong.times())
	}
}

func TestReachSolverAnswersTheHealthStatusOrWhyTheNodeCantReachIt(t *testing.T) {
	n := node(t, &bytes.Buffer{})
	v := newSolver(t, "Solver/seedbox")
	if status, err := n.ReachSolver(t.Context(), searching.Solver{Address: v.srv.URL, Timeout: time.Minute}); err != nil || status != http.StatusOK {
		t.Fatalf("ReachSolver of a running solver = %d, %v; want 200", status, err)
	}
	if status, err := n.ReachSolver(t.Context(), searching.Solver{Address: v.srv.URL + "/elsewhere", Timeout: time.Minute}); err != nil || status != http.StatusNotFound {
		t.Fatalf("ReachSolver of a path that serves no solver = %d, %v; want 404", status, err)
	}
	_, err := n.ReachSolver(t.Context(), searching.Solver{Address: "http://127.0.0.1:1", Timeout: time.Minute})
	if u, ok := errors.AsType[*searching.SolverUnreachable](err); !ok || !strings.Contains(u.Cause, "connection refused") {
		t.Fatalf("ReachSolver of an address nothing listens on = %v, want SolverUnreachable with the cause", err)
	}
}

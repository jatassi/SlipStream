// Package challengesolver is fake-challenge-solver: a Challenge solver
// that speaks FlareSolverr's API, one front per Node. Written from the
// API's shape: GET /health, and POST /v1 with the request.get command,
// whose solution carries cookies and a User-Agent. It opens no page. It
// answers fixture-site's clearance for the Node that asked, under a
// User-Agent naming that Node, so a clearance holds only for the Node
// that earned it.
package challengesolver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/internal/node"
)

var (
	// Timeout has every request.get answer that the page didn't clear in
	// time, from now on: {"on": true}, or {} to clear pages again.
	Timeout = fake.CueSpec{Name: "challenge-solver-timeout"}
	// Captcha has every request.get answer that the page showed a
	// captcha, from now on: {"on": true}, or {} to clear pages again.
	Captcha = fake.CueSpec{Name: "challenge-solver-captcha"}
)

// Solved records each page the solver cleared: its host.
var Solved = fake.CaptureSpec{Name: "solves", Kind: "challenge-solver.solve"}

// Fake is the Challenge solver an Admin sets on each Searching Node, at
// that Node's front. Nothing points a Node at it but that setting.
// Unreachable plays as a refuse fault on the Node's front.
var Fake = fake.Spec{
	Name: "challenge-solver",
	New:  newServer,
	// Endpoints sets nothing: an Admin types each Node's solver address.
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("a Challenge solver is a service the Admin runs, and SlipStream uses only FlareSolverr's fixed request.get"),
	Cues:      []fake.CueSpec{Timeout, Captcha},
	Captures:  []fake.CaptureSpec{Solved},
	Entered:   solvers,
}

// UserAgent is the User-Agent the solver clears pages with for the Node
// origin.
func UserAgent(origin string) string { return "Mozilla/5.0 (challenge-solver; " + origin + ")" }

// Play turns a cue's play on, or off with the zero value.
type Play struct {
	On bool `json:"on,omitempty"`
}

// Solve is a Solved record's data.
type Solve struct {
	Host string `json:"host"`
}

type server struct {
	scenario scenario.Scenario
	solved   *fake.Capture
	mu       sync.Mutex
	timeout  bool
	captcha  bool
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	solved, err := env.Capture(Solved.Name)
	if err != nil {
		return nil, err
	}
	srv := &server{scenario: s, solved: solved}
	if err := fake.OnCue(env, Timeout.Name, srv.play(&srv.timeout)); err != nil {
		return nil, err
	}
	if err := fake.OnCue(env, Captcha.Name, srv.play(&srv.captcha)); err != nil {
		return nil, err
	}
	return fake.HTTP(env, fake.Routes{
		"health": {Pattern: "GET /health", Handler: health},
		"v1":     {Pattern: "POST /v1", Handler: srv.command},
	}), nil
}

func (s *server) play(flag *bool) func(context.Context, Play) (any, error) {
	return func(_ context.Context, p Play) (any, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		*flag = p.On
		return p, nil
	}
}

func health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "msg": "The fake Challenge solver is ready."})
}

type command struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url"`
	MaxTimeout int64  `json:"maxTimeout"`
}

type cookie struct {
	Name    string  `json:"name"`
	Value   string  `json:"value"`
	Domain  string  `json:"domain"`
	Path    string  `json:"path"`
	Expires float64 `json:"expires"`
	Secure  bool    `json:"secure"`
}

type solution struct {
	URL       string   `json:"url"`
	Status    int      `json:"status"`
	Cookies   []cookie `json:"cookies"`
	UserAgent string   `json:"userAgent"`
	Response  string   `json:"response"`
}

func (s *server) command(w http.ResponseWriter, r *http.Request) {
	var c command
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		refuse(w, "The request body isn't a JSON command.")
		return
	}
	u, err := url.Parse(c.URL)
	switch {
	case c.Cmd != "request.get":
		refuse(w, fmt.Sprintf("This solver runs only request.get, not %q.", c.Cmd))
		return
	case err != nil || u.Host == "":
		refuse(w, "The url isn't an absolute URL.")
		return
	}
	s.mu.Lock()
	timeout, captcha := s.timeout, s.captcha
	s.mu.Unlock()
	switch {
	case timeout:
		refuse(w, fmt.Sprintf("Error solving the challenge. Timeout after %.1f seconds.", float64(c.MaxTimeout)/1000))
		return
	case captcha:
		refuse(w, "Captcha detected but no automatic solver is configured.")
		return
	}
	origin := fake.OriginOf(r.Context())
	agent := UserAgent(origin)
	s.solved.Record(r.Context(), Solve{Host: u.Hostname()})
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok", "message": "Challenge solved.",
		"solution": solution{
			URL: c.URL, Status: http.StatusOK, UserAgent: agent, Response: "<html><body>Cleared.</body></html>",
			Cookies: []cookie{{
				Name: site.ClearanceCookie, Value: site.Clearance(s.scenario, origin, agent), Domain: "." + u.Hostname(), Path: "/",
				Expires: -1, Secure: u.Scheme == "https",
			}},
		},
	})
}

// refuse answers an error, as the API does for a page it couldn't clear
// and a command it doesn't take.
func refuse(w http.ResponseWriter, message string) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "message": message})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

// solvers are the Challenge solvers an Admin set, each on the Node that
// reaches it.
func solvers(get func(path string) ([]byte, error)) ([]fake.Entered, error) {
	body, err := get("/api/v1/nodes")
	if err != nil {
		return nil, err
	}
	var list struct {
		Nodes []struct {
			ID    string     `json:"id"`
			Name  string     `json:"name"`
			Roles []roleView `json:"roles"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("listNodes: %w", err)
	}
	var out []fake.Entered
	for _, n := range list.Nodes {
		if !slices.ContainsFunc(n.Roles, func(r roleView) bool { return r.Role == "searching" }) {
			continue
		}
		body, err := get("/api/v1/nodes/" + n.ID + "/challenge-solver")
		if err != nil {
			return nil, err
		}
		var s struct {
			Address *string `json:"address"`
		}
		if err := json.Unmarshal(body, &s); err != nil {
			return nil, fmt.Errorf("getChallengeSolver of %s: %w", n.Name, err)
		}
		if s.Address != nil {
			out = append(out, fake.Entered{What: "Challenge solver of " + n.Name, URL: *s.Address, NodeID: n.ID})
		}
	}
	return out, nil
}

type roleView struct {
	Role string `json:"role"`
}

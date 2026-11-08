package searching

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
)

// challengeServers are the Server headers of the services that put a
// browser challenge in front of a site.
var challengeServers = []string{"cloudflare", "cloudflare-nginx", "ddos-guard"}

// challengeMarkers are what a challenge page holds, as Prowlarr reads it.
var challengeMarkers = []string{"<title>Just a moment...</title>", "<title>Access denied</title>", "<title>Attention Required! | Cloudflare</title>"}

// challenged reports whether res is a browser challenge: a 403 or 503 from
// a challenge service whose page holds one of its markers.
func challenged(res *http.Response, body []byte) bool {
	if !slices.Contains(challengeServers, strings.ToLower(res.Header.Get("Server"))) {
		return false
	}
	if res.StatusCode != http.StatusForbidden && res.StatusCode != http.StatusServiceUnavailable {
		return false
	}
	page := string(body)
	return slices.ContainsFunc(challengeMarkers, func(m string) bool { return strings.Contains(page, m) }) ||
		strings.TrimSpace(page) == "error code: 1020" || strings.Contains(strings.ToLower(page), "<title>ddos-guard</title>")
}

// errChallenged is a response that was a browser challenge.
var errChallenged = errors.New("the site responded with a browser challenge")

// SolverUnreachable is a Challenge solver the Node couldn't reach.
type SolverUnreachable struct{ Cause string }

func (e *SolverUnreachable) Error() string { return "reach the Challenge solver: " + e.Cause }

// clearance is what a Challenge solver cleared a host with: the cookies
// to send it and the User-Agent they were earned with.
type clearance struct {
	cookies   []Cookie
	userAgent string
	at        time.Time
}

// clearances are this Node's clearance of each host it cleared, so every
// later request to that host sends the same cookies and User-Agent,
// whichever Indexer sends it.
type clearances struct {
	mu     sync.Mutex
	byHost map[string]clearance
}

func (cs *clearances) of(host string) (clearance, bool) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	c, ok := cs.byHost[strings.ToLower(host)]
	return c, ok
}

func (cs *clearances) put(host string, c clearance) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.byHost[strings.ToLower(host)] = c
}

// clear has the Node's Challenge solver clear the challenge at u, one
// solve at a time for the Indexer, and keeps the clearance for u's host on
// this Node and in the Indexer's session there. A solve that finished
// while this one waited its turn is used as it is.
func (s *Site) clear(ctx context.Context, u *url.URL) error {
	if s.solver == nil {
		return &Stopped{Cause: ChallengeUnsolved}
	}
	before, _ := s.svc.cleared.of(u.Host)
	select {
	case s.slot.solving <- struct{}{}:
	case <-ctx.Done():
		return fmt.Errorf("wait for another challenge solve: %w", ctx.Err())
	}
	defer func() { <-s.slot.solving }()
	if now, ok := s.svc.cleared.of(u.Host); ok && now.at.After(before.at) {
		return nil
	}
	c, err := solve(ctx, s.svc.client, *s.solver, u, s.svc.cfg.Timings.Request)
	if err != nil {
		s.svc.cfg.Log.Info("A Challenge solver couldn't clear an Indexer's challenge", "indexer", s.Indexer, "host", u.Host, "err", err.Error())
		return err
	}
	s.svc.cleared.put(u.Host, c)
	s.slot.cleared(c)
	return nil
}

// solveRequest is FlareSolverr's request.get command: the solver opens url
// in its browser, waiting up to maxTimeout milliseconds for the page to
// clear.
type solveRequest struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url"`
	MaxTimeout int64  `json:"maxTimeout"`
}

// solveResponse is a solver's answer to a command: status "ok" with the
// solution, or "error" with why in message.
type solveResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		Cookies []struct {
			Name    string  `json:"name"`
			Value   string  `json:"value"`
			Domain  string  `json:"domain"`
			Path    string  `json:"path"`
			Expires float64 `json:"expires"`
			Secure  bool    `json:"secure"`
		} `json:"cookies"`
		UserAgent string `json:"userAgent"`
	} `json:"solution"`
}

// solve sends u to solver with request.get, waiting the solver's timeout
// and then grace for its response. A solver that can't be reached, times
// out or responds with something other than a solution is *Failing. One
// that reports it couldn't clear the page, such as at a captcha, stops the
// Indexer.
func solve(ctx context.Context, client *http.Client, solver Solver, u *url.URL, grace time.Duration) (clearance, error) {
	ctx, cancel := context.WithTimeout(ctx, solver.Timeout+grace)
	defer cancel()
	body, err := json.Marshal(solveRequest{Cmd: "request.get", URL: u.String(), MaxTimeout: solver.Timeout.Milliseconds()})
	if err != nil {
		return clearance{}, &Failing{Err: fmt.Errorf("build the Challenge solver's request: %w", err)}
	}
	at, err := url.JoinPath(solver.Address, "v1")
	if err != nil {
		return clearance{}, &Failing{Err: fmt.Errorf("the Challenge solver's address: %w", err)}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, at, bytes.NewReader(body))
	if err != nil {
		return clearance{}, &Failing{Err: fmt.Errorf("build the Challenge solver's request: %w", WithoutURL(err))}
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return clearance{}, &Failing{Err: fmt.Errorf("reach the Challenge solver: %w", WithoutURL(err))}
	}
	defer func() { _ = res.Body.Close() }()
	var out solveResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, readLimit)).Decode(&out); err != nil {
		return clearance{}, &Failing{Err: fmt.Errorf("the Challenge solver responded with status %d and no answer SlipStream reads", res.StatusCode)}
	}
	switch {
	case out.Status == "ok":
		return clearanceOf(out, u.Hostname()), nil
	case strings.Contains(strings.ToLower(out.Message), "timeout"):
		return clearance{}, &Failing{Err: fmt.Errorf("the Challenge solver timed out: %s", out.Message)}
	}
	return clearance{}, &Stopped{Cause: ChallengeUnsolved, SolverTried: true}
}

// clearanceOf is a solution's cookies and User-Agent. A cookie without a
// domain is host's, and one whose domain starts with a dot goes to the
// hosts under it too.
func clearanceOf(out solveResponse, host string) clearance {
	c := clearance{userAgent: out.Solution.UserAgent, at: time.Now()}
	for _, k := range out.Solution.Cookies {
		cookie := Cookie{Name: k.Name, Value: k.Value, Domain: strings.ToLower(strings.TrimPrefix(k.Domain, ".")), Path: k.Path, Secure: k.Secure}
		cookie.HostOnly = !strings.HasPrefix(k.Domain, ".")
		if cookie.Domain == "" {
			cookie.Domain, cookie.HostOnly = strings.ToLower(host), true
		}
		if cookie.Path == "" {
			cookie.Path = "/"
		}
		if k.Expires > 0 {
			cookie.Expires = time.UnixMilli(int64(k.Expires * 1000))
		}
		c.cookies = append(c.cookies, cookie)
	}
	return c
}

// reachSolver sends GET {address}/health from this Node, within timeout.
func reachSolver(ctx context.Context, client *http.Client, address string, timeout time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	at, err := url.JoinPath(address, "health")
	if err != nil {
		return 0, fmt.Errorf("the address: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, at, http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("the address: %w", WithoutURL(err))
	}
	res, err := client.Do(req)
	if err != nil {
		return 0, &SolverUnreachable{Cause: WithoutURL(err).Error()}
	}
	_ = res.Body.Close()
	return res.StatusCode, nil
}

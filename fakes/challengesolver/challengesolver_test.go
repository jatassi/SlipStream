package challengesolver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/challengesolver"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
)

type answer struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	Solution struct {
		Cookies []struct {
			Name   string `json:"name"`
			Value  string `json:"value"`
			Domain string `json:"domain"`
		} `json:"cookies"`
		UserAgent string `json:"userAgent"`
	} `json:"solution"`
}

func post(t *testing.T, f faketest.Served, cmd string) (int, answer) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"cmd": cmd, "url": "https://cinder.fixture-site.fake.localhost/torrents.php?q=orchard", "maxTimeout": 60000})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.Address.URL("/v1"), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	var a answer
	if err := json.NewDecoder(res.Body).Decode(&a); err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, a
}

func TestTheSolverClearsAPageForTheNodeThatAsked(t *testing.T) {
	s := scenario.Scenario{Seed: "solver"}
	f := faketest.Loopback(t, challengesolver.Fake, s, faketest.As("seedbox"))
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.Address.URL("/health"), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := f.Client.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("GET /health = %v, %v; want 200", res, err)
	}
	_ = res.Body.Close()

	status, a := post(t, f, "request.get")
	agent := challengesolver.UserAgent("seedbox")
	if status != http.StatusOK || a.Status != "ok" || a.Solution.UserAgent != agent || len(a.Solution.Cookies) != 1 ||
		a.Solution.Cookies[0].Name != site.ClearanceCookie || a.Solution.Cookies[0].Value != site.Clearance(s, "seedbox", agent) ||
		a.Solution.Cookies[0].Domain != ".cinder.fixture-site.fake.localhost" {
		t.Fatalf("request.get = %d %+v, want Cinder's clearance for seedbox under seedbox's User-Agent", status, a)
	}
	if recs := f.Records(t); len(recs) != 1 || recs[0].Kind != challengesolver.Solved.Kind || recs[0].Node != "seedbox" ||
		string(recs[0].Data) != `{"host":"cinder.fixture-site.fake.localhost"}` {
		t.Fatalf("the solves capture = %+v, want Cinder's host solved from seedbox", recs)
	}
	if status, a := post(t, f, "sessions.create"); status != http.StatusInternalServerError || a.Status != "error" {
		t.Fatalf("sessions.create = %d %+v, want an error, since only request.get runs", status, a)
	}
}

func TestTheCuesPlayATimeoutAndACaptcha(t *testing.T) {
	f := faketest.Loopback(t, challengesolver.Fake, scenario.Scenario{Seed: "solver"}, faketest.As("seedbox"))
	for _, tc := range []struct{ cue, want string }{
		{challengesolver.Timeout.Name, "Error solving the challenge. Timeout after 60.0 seconds."},
		{challengesolver.Captcha.Name, "Captcha detected but no automatic solver is configured."},
	} {
		f.Cue(t, tc.cue, challengesolver.Play{On: true})
		if status, a := post(t, f, "request.get"); status != http.StatusInternalServerError || a.Status != "error" || a.Message != tc.want {
			t.Errorf("request.get after %s = %d %+v, want the error %q", tc.cue, status, a, tc.want)
		}
		f.Cue(t, tc.cue, challengesolver.Play{})
	}
	if status, a := post(t, f, "request.get"); status != http.StatusOK || a.Status != "ok" {
		t.Fatalf("request.get after the cues cleared = %d %+v, want a solution", status, a)
	}
}

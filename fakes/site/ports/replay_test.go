package ports

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site/ports/internal/testport"
	"github.com/jatassi/slipstream/internal/ports/port"
)

// serve is fixture-site with routes already taken, and testport's
// fixture replayed wherever no route serves it.
func serve(t *testing.T, taken fake.Routes, load Loader) (site faketest.Served, served Served) {
	t.Helper()
	spec := fake.Spec{
		Name: "fixture-site", TLS: fake.FrontTLS, Cues: []fake.CueSpec{PortCase},
		New: func(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
			routes := fake.Routes{}
			for k, v := range taken {
				routes[k] = v
			}
			var err error
			served, err = Serve(env, testport.Registry(), load, routes)
			return fake.HTTP(env, routes), err
		},
	}
	return faketest.InMemory(t, spec, scenario.Scenario{}), served
}

func get(t *testing.T, s faketest.Served, path string, header ...string) (status int, body string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, Address(testport.ID)+strings.TrimPrefix(path, "/"), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	res, err := s.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(raw)
}

func TestWithNoCaseCuedAReplayAnswersFromWhicheverCaseMatches(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, _ := serve(t, nil, Embedded)
		if code, body := get(t, s, "/api/search?q=Juniper+Falls"); code != http.StatusOK || body != "{\"results\":[]}\n" {
			t.Errorf("the empty case's search = %d %q", code, body)
		}
		if code, _ := get(t, s, "/download/1"); code != http.StatusOK {
			t.Errorf("the grab case's download = %d", code)
		}
		if code, body := get(t, s, "/nowhere"); code != Stray || !strings.Contains(body, "GET testport.fixture-site.fake.localhost/nowhere") {
			t.Errorf("a request no exchange matches = %d %q", code, body)
		}
	})
}

func TestTheCaseCueAnswersOneCaseInOrderAndStartsItOver(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s, _ := serve(t, nil, Embedded)
		s.Cue(t, PortCase.Name, CaseArgs{Port: string(testport.ID), Case: "limit"})
		for range 2 {
			if code, _ := get(t, s, "/api/search?q="); code != http.StatusTooManyRequests {
				t.Errorf("the limit case's search = %d, want 429 each time", code)
			}
		}
		s.Cue(t, PortCase.Name, CaseArgs{Port: string(testport.ID), Case: "loggedout"})
		if code, body := get(t, s, "/api/search?q=", "Cookie", "sid={{sid}}"); code != Stray || !strings.Contains(body, `header Cookie is "sid={{sid}}", not "sid={{expiredSid}}"`) {
			t.Errorf("a request out of the case's order = %d %q", code, body)
		}
	})
}

func TestAFamilyFakeOnAPortsHostServesItInPlaceOfItsFixture(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		family := fake.Routes{"testport.search": {Pattern: "GET " + Host(testport.ID) + "/api/search", Handler: func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "family")
		}}}
		s, served := serve(t, family, Embedded)
		if _, ok := served[testport.ID]; ok {
			t.Error("Serve replays testport beside its family fake")
		}
		if code, body := get(t, s, "/api/search?q="); code != http.StatusOK || body != "family" {
			t.Errorf("the search = %d %q, want the family fake's", code, body)
		}
	})
}

func TestServeRefusesAPortWithNoFixture(t *testing.T) {
	missing := func(id port.ID) (Fixture, error) { return Load(FS, id+"-gone") }
	_, err := Serve(fake.Env{}, testport.Registry(), missing, fake.Routes{})
	if err == nil || !strings.Contains(err.Error(), "port testport-gone has no fixture") {
		t.Errorf("Serve = %v", err)
	}
}

func TestUseRefusesACaseTheFixtureLacks(t *testing.T) {
	f, err := Embedded(testport.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewReplay(f).Use("nonesuch"); err == nil || err.Error() != `the fixture of testport has no case "nonesuch"` {
		t.Errorf("Use = %v", err)
	}
}

func TestParseRefusesACaseWithTwoCalls(t *testing.T) {
	_, err := Parse("lantern", []byte("source: synthetic\ncases:\n  - name: search\n    call: {recent: {}, fetch: /x}\n    exchanges: [{request: {path: /}, response: {}}]\n"))
	if err == nil || !strings.Contains(err.Error(), "this one sets 2") {
		t.Errorf("Parse = %v", err)
	}
}

func TestAnExchangeMatchesOnlyAQueryWithoutTheParametersItRulesOut(t *testing.T) {
	f := Fixture{Port: testport.ID}
	m := Match{Path: "/api/search", Query: map[string]string{"q": "orchard"}, NoQuery: []string{"freeleech"}}
	for target, want := range map[string]string{
		"/api/search?q=orchard":             "",
		"/api/search?q=orchard&freeleech=1": "its query has freeleech",
		"/api/search?q=orchard&freeleech=":  "its query has freeleech",
	} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, Address(testport.ID)+strings.TrimPrefix(target, "/"), http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		if got := NewReplay(f).matches(m, req, nil); got != want {
			t.Errorf("%s matches with %q, want %q", target, got, want)
		}
	}
}

func TestParseTakesACaseThatSendsNothingOnlyWhenItSaysSo(t *testing.T) {
	const head = "source: synthetic\ncases:\n  - name: tv\n    call: {search: {mode: tv, text: Juniper Falls}}\n"
	f, err := Parse("lantern", []byte(head+"    exchanges: []\n"))
	if err != nil || len(f.Cases) != 1 || len(f.Cases[0].Exchanges) != 0 {
		t.Errorf("Parse of exchanges: [] = %+v, %v; want the case with no exchange", f.Cases, err)
	}
	if _, err := Parse("lantern", []byte(head)); err == nil || !strings.Contains(err.Error(), "exchanges: []") {
		t.Errorf("Parse of a case with no exchanges list = %v, want it refused", err)
	}
}

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/site/ports"
	"github.com/jatassi/slipstream/fakes/site/ports/internal/testport"
)

const (
	realCookie = "8f14e45fceea167a5a36dedd4bea2543"
	realHash   = "ab5c9a64e0e0a3b7c8c0e7d3f1d2c4b5a6978899"
)

// realSite is a site testport's protocol reaches, with a real account's
// values in what it sends. extra goes into each listing.
func realSite(extra string) http.RoundTripper {
	mux := http.NewServeMux()
	mux.HandleFunc("GET realsite.example/api/search", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("sid"); err != nil || c.Value != realCookie {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"name":"Inception.2010.1080p.BluRay.x264-GRP","size":1,"infohash":"` + realHash +
			`","download":"https://realsite.example/download/9","uploader":"jdoe"` + extra + `}]}`))
	})
	mux.HandleFunc("POST realsite.example/login", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("username") != "jdoe" || r.FormValue("password") != "hunter2hunter2" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: realCookie, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	})
	return roundTrip(func(req *http.Request) (*http.Response, error) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		res := w.Result()
		res.Request = req
		return res, nil
	})
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func liveConfigFor(t *testing.T, dir, extra string) liveConfig {
	t.Helper()
	c, err := parseLive([]string{
		"-port", "testport", "-case", "search", "-settings", settingsFile(t), "-address", "https://realsite.example/",
		"-dir", dir, "-search", "mode=search&q=Inception",
	}, testport.Registry())
	if err != nil {
		t.Fatal(err)
	}
	c.native, c.transport = testport.Driver{}, realSite(extra)
	return c
}

func settingsFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.yaml")
	if err := os.WriteFile(path, []byte("username: jdoe\npassword: hunter2hunter2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestALiveCaptureWritesTheExchangesAsAScrubbedCase(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		if err := live(t.Context(), liveConfigFor(t, dir, "")); err != nil {
			t.Fatal(err)
		}
		f := recorded(t, dir)
		c, _ := f.Case("search")
		if len(c.Exchanges) != 3 {
			t.Fatalf("the case has %d exchanges, want the search, the sign-in and the search again", len(c.Exchanges))
		}
		signIn, again := c.Exchanges[1], c.Exchanges[2]
		if signIn.Request.Form["password"] != "{{password}}" || signIn.Response.Headers["Set-Cookie"] != "sid={{sid}}; Path=/; HttpOnly; Secure; SameSite=Lax" {
			t.Errorf("the sign-in is %+v", signIn)
		}
		if again.Request.Headers["Cookie"] != "sid={{sid}}" || again.Request.Query["q"] != "Cedar Hollow" || again.Request.Host != "" {
			t.Errorf("the search again is %+v", again.Request)
		}
		body := string(f.Bodies[again.Response.Body])
		for _, want := range []string{"Cedar.Hollow.2010.1080p", ports.Mint(0), `"uploader":"fixture-user"`, "https://testport.fixture-site.fake.localhost/download/9"} {
			if !strings.Contains(body, want) {
				t.Errorf("the body %s holds no %s", body, want)
			}
		}
	})
}

// recorded is the fixture a capture wrote in dir, which must pass Scrub
// and hold its settings as placeholders.
func recorded(t *testing.T, dir string) ports.Fixture {
	t.Helper()
	f, err := ports.Load(os.DirFS(dir), testport.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found := ports.Scrub(f); len(found) > 0 {
		t.Errorf("Scrub = %q", found)
	}
	if f.Source != ports.Recorded || f.Settings["username"] != ports.FixtureUser || f.Settings["password"] != "{{password}}" {
		t.Errorf("the fixture is %s with settings %v", f.Source, f.Settings)
	}
	return f
}

func TestALiveCaptureWritesNothingTheScrubCheckRefuses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		err := live(t.Context(), liveConfigFor(t, dir, `,"authkey":"9b74c9897bac770ffc029102a200c5de"`))
		if err == nil || !strings.Contains(err.Error(), `"9b74c9897bac770ffc029102a200c5de" is shaped like a key`) {
			t.Errorf("live = %v", err)
		}
		if entries, _ := os.ReadDir(dir); len(entries) > 0 {
			t.Errorf("live wrote %v", entries)
		}
	})
}

func TestAnImportRewritesAProwlarrRecordingIntoABody(t *testing.T) {
	dir := t.TempDir()
	recording := filepath.Join(t.TempDir(), "imdbsearch.json")
	raw := "\xef\xbb\xbf" + `{"Movies":[{"Title":"The Night Of","Cover":"https:\/\/testport.fixture-site.fake.localhost\/c.jpg",` +
		`"Torrents":[{"ReleaseName":"The.Night.Of.2016.720p.BluRay-DON","InfoHash":"` + realHash + `"}]}],"AuthKey":"9b74c9897bac770ffc029102a200c5de"}`
	if err := os.WriteFile(recording, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"-port", "testport", "-recording", recording, "-body", "search.json", "-dir", dir}
	refused, err := parseImport(args, testport.Registry())
	if err != nil {
		t.Fatal(err)
	}
	if err := importRecording(refused); err == nil || !strings.Contains(err.Error(), "shaped like a key") {
		t.Errorf("an import with the key left in = %v", err)
	}
	c, err := parseImport(append(args, "-replace", "9b74c9897bac770ffc029102a200c5de={{authKey}}"), testport.Registry())
	if err != nil {
		t.Fatal(err)
	}
	if err := importRecording(c); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "testport", "bodies", "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"Movies":[{"Title":"Cedar Hollow","Cover":"https:\/\/testport.fixture-site.fake.localhost\/c.jpg",` +
		`"Torrents":[{"ReleaseName":"Cedar.Hollow.2016.720p.BluRay-DON","InfoHash":"` + ports.Mint(0) + `"}]}],"AuthKey":"{{authKey}}"}`
	if string(body) != want {
		t.Errorf("the body is\n%s\nwant\n%s", body, want)
	}
}

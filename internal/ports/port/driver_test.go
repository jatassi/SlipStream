package port

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/prowlarr/datetimeutil"
	"github.com/jatassi/slipstream/internal/searching"
)

// world is the sites a test Node reaches, each host's handler serving in
// memory, so a job runs inside a synctest bubble on its fake clock.
type world struct {
	mu    sync.Mutex
	hosts map[string]http.HandlerFunc
	hits  []hit
}

// hit is one request a site got.
type hit struct {
	at     time.Time
	method string
	host   string
	path   string
	query  url.Values
	form   url.Values
	header http.Header
}

func (h hit) String() string { return h.method + " " + h.host + h.path }

func (w *world) RoundTrip(r *http.Request) (*http.Response, error) {
	h, ok := w.hosts[r.URL.Host]
	if !ok {
		return nil, fmt.Errorf("no site at %s", r.URL.Host)
	}
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}
	form, _ := url.ParseQuery(string(body))
	w.mu.Lock()
	w.hits = append(w.hits, hit{at: time.Now(), method: r.Method, host: r.URL.Host, path: r.URL.Path, query: r.URL.Query(), form: form, header: r.Header.Clone()})
	w.mu.Unlock()
	r.Body = io.NopCloser(strings.NewReader(string(body)))
	rec := httptest.NewRecorder()
	h(rec, r)
	res := rec.Result()
	res.Request = r
	return res, nil
}

func (w *world) requests() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]string, len(w.hits))
	for i, h := range w.hits {
		out[i] = h.String()
	}
	return out
}

func (w *world) to(host string) []hit {
	w.mu.Lock()
	defer w.mu.Unlock()
	var out []hit
	for _, h := range w.hits {
		if h.host == host {
			out = append(out, h)
		}
	}
	return out
}

// node is a Node's Searching service over w, running ports, at the Node's
// own 2 s floor.
func node(w *world, ports ...Port) searching.Remote {
	timings := searching.DefaultTimings()
	timings.Hold = time.Hour
	return searching.RemoteOf(searching.New(searching.Config{
		HTTP: &http.Client{Transport: w}, Timings: timings, Log: slog.New(slog.DiscardHandler),
		Native: Driver(NewRegistry(ports, []Retired{{ID: "brook", Name: "Brook", Reason: "the site closed"}}, nil)),
	}))
}

var jobIDs atomic.Int32

// run runs one job of port p for the Indexer at lantern.test.
func run(t *testing.T, n searching.Remote, j searching.Job) searching.Answer {
	t.Helper()
	j.ID = fmt.Sprintf("%s/%d", t.Name(), jobIDs.Add(1))
	j.Indexer, j.Kind = "lantern", searching.Native
	if j.Address == "" {
		j.Address = "https://lantern.test/"
	}
	if j.Revision == 0 {
		j.Revision = 1
	}
	poll, err := n.Run(t.Context(), j)
	if err != nil || poll.Answer == nil {
		t.Fatalf("Run = %+v, %v; want an answer", poll, err)
	}
	return *poll.Answer
}

func result(a searching.Answer) string {
	if a.Outcome == nil {
		return "ok"
	}
	return fmt.Sprintf("%T %v", a.Outcome, a.Outcome)
}

func titles(rs []searching.Release) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Title)
	}
	return out
}

// listing is a test site's JSON page.
type listing struct {
	Error     string `json:"error,omitempty"`
	SignedOut bool   `json:"signedOut,omitempty"`
	Results   []item `json:"results,omitempty"`
}

type item struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Hash  string   `json:"hash,omitempty"`
	Down  float64  `json:"down"`
	Up    float64  `json:"up"`
	Flags []string `json:"flags,omitempty"`
}

func respond(w http.ResponseWriter, l listing) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(l); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func setUID(w http.ResponseWriter, uid string) {
	http.SetCookie(w, &http.Cookie{Name: "uid", Value: uid, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

func signedOut(r *IndexerResponse) bool { return strings.Contains(r.Content, `"signedOut"`) }

// parseListing is the test ports' parser.
func parseListing[S any](ix *Indexer[S], r *IndexerResponse) ([]TorrentInfo, error) {
	var l listing
	if err := json.Unmarshal(r.Body, &l); err != nil {
		return nil, IndexerError("the site's response isn't JSON")
	}
	switch l.Error {
	case "":
	case "auth":
		return nil, IndexerAuthError("Your key is wrong")
	case "limit":
		return nil, RequestLimitReached("Slow down")
	default:
		return nil, IndexerError("%s", l.Error)
	}
	var out []TorrentInfo
	for _, it := range l.Results {
		info := TorrentInfo{
			GUID: "lantern-" + it.ID, Title: it.Title, Size: 1 << 30, DownloadURL: ix.URL("dl/" + it.ID).String(), InfoHash: it.Hash,
			Seeders: 5, Peers: 7, Categories: ix.Categories.FromTracker("1"), DownloadVolumeFactor: it.Down, UploadVolumeFactor: it.Up,
		}
		for _, f := range it.Flags {
			info.IndexerFlags = append(info.IndexerFlags, SiteFlag(f))
		}
		out = append(out, info)
	}
	return out, nil
}

// browse is the test ports' generator: one GET of browse.php.
func browse[S any](ix *Indexer[S], q SearchCriteria) (Chain, error) {
	r := Get(ix.URL("browse.php"))
	if q.SearchTerm != "" {
		r.AddQueryParam("q", q.SearchTerm)
	}
	if q.ImdbID != "" {
		r.AddQueryParam("imdb", q.FullImdbID())
	}
	if q.TmdbID != 0 {
		r.AddQueryParam("tmdb", strconv.Itoa(q.TmdbID))
	}
	return One(r)
}

func lantern(id ID) Entry {
	return Entry{
		ID: id, Name: "Lantern", Description: "Lantern is a private tracker for tests.", Privacy: Private,
		IndexerURLs: []string{"https://lantern.test/"},
		Capabilities: Capabilities{
			MovieSearchParams: []Param{Q, ImdbID},
			Categories:        Categories(Map(1, newznab.Movies, "Movies")),
		},
	}
}

type (
	apiKey   struct{ APIKeyField }
	account  struct{ UserPass }
	noFields struct{}
)

// keyPort sends its API key as an ApiKey header.
func keyPort(e Entry) Port {
	return Define(e, Impl[apiKey]{
		Login: NoLogin{}, Movie: browse[apiKey], Basic: browse[apiKey], Parse: parseListing[apiKey],
		ModifyRequest: func(ix *Indexer[apiKey], r *Request) { r.SetHeader("ApiKey", ix.Settings.APIKey) },
	})
}

// formPort signs in at login.php, and the uid cookie it earns is the
// session.
func formPort(e Entry) Port {
	return Define(e, Impl[account]{
		Login: FormLogin[account]{DoLogin: func(in *SignIn[account]) error {
			r, err := in.Do(Post(in.BaseURL.JoinPath("login.php")).
				AddFormParameter("username", in.Settings.Username).
				AddFormParameter("password", string(in.Settings.Password)))
			if err != nil {
				return err
			}
			if r.Cookies()["uid"] == "" {
				return IndexerAuthError(strings.TrimSpace(r.Content))
			}
			return nil
		}, CheckIfLoginNeeded: signedOut},
		Movie: browse[account], Basic: browse[account], Parse: parseListing[account],
	})
}

// lanternSite serves browse.php to the uid cookie in, a sign-in at
// login.php to sam and pw that sets uid to next, and a .torrent at dl/.
func lanternSite(in, next string, results ...item) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login.php":
			if r.FormValue("username") != "sam" || r.FormValue("password") != "pw" {
				_, _ = w.Write([]byte("Wrong password"))
				return
			}
			in = next
			setUID(w, next)
			_, _ = w.Write([]byte("Welcome"))
		case "/browse.php":
			if c, err := r.Cookie("uid"); err != nil || c.Value != in {
				respond(w, listing{SignedOut: true})
				return
			}
			respond(w, listing{Results: results})
		default:
			http.NotFound(w, r)
		}
	}
}

var samAndPW = map[string]string{"username": "sam", "password": "pw"}

func heldUID(uid string) *searching.Session {
	return &searching.Session{
		Cookies: []searching.Cookie{{Name: "uid", Value: uid, Domain: "lantern.test", Path: "/", HostOnly: true}},
		Earned:  time.Unix(1, 0),
	}
}

func movieSearch(text string, ids map[string]string) searching.SearchCall {
	return searching.SearchCall{Query: searching.Query{Mode: "movie", Text: text, IDs: ids}}
}

func TestALoggedOutPageSignsInOnceAndRetriesOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": lanternSite("fresh", "fresh", item{ID: "1", Title: "Orchard.2026.1080p", Down: 1, Up: 1})}}
		a := run(t, node(w, formPort(lantern("lantern"))), searching.Job{Port: "lantern", Settings: samAndPW, Session: heldUID("stale"), Call: movieSearch("orchard", nil)})
		if result(a) != "ok" || !slices.Equal(titles(a.Releases), []string{"Orchard.2026.1080p"}) {
			t.Fatalf("a search on a stale session = %v, %s; want the Release after a sign-in", titles(a.Releases), result(a))
		}
		want := []string{"GET lantern.test/browse.php", "POST lantern.test/login.php", "GET lantern.test/browse.php"}
		if got := w.requests(); !slices.Equal(got, want) {
			t.Errorf("requests = %v, want %v", got, want)
		}
		if a.Session == nil || a.Session.Cookies[0].Value != "fresh" {
			t.Errorf("the changed session = %+v, want the uid the sign-in earned", a.Session)
		}

		w = &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/login.php" {
				setUID(w, "fresh")
				return
			}
			respond(w, listing{SignedOut: true})
		}}}
		a = run(t, node(w, formPort(lantern("lantern"))), searching.Job{Port: "lantern", Settings: samAndPW, Session: heldUID("stale"), Call: movieSearch("orchard", nil)})
		if got, want := result(a), "*searching.Failing the site signed SlipStream out again right after it signed in"; got != want {
			t.Errorf("a site that signs out again = %s, want %s", got, want)
		}
		if got := w.requests(); !slices.Equal(got, want) {
			t.Errorf("requests = %v, want exactly one sign-in and one retry: %v", got, want)
		}
	})
}

func TestAFormPortSignsInFirstAndStopsOnARejectedPassword(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": lanternSite("", "u1", item{ID: "1", Title: "Orchard.2026.1080p", Down: 1, Up: 1})}}
		n := node(w, formPort(lantern("lantern")))
		a := run(t, n, searching.Job{Port: "lantern", Settings: samAndPW, Call: movieSearch("orchard", nil)})
		if want := []string{"POST lantern.test/login.php", "GET lantern.test/browse.php"}; result(a) != "ok" || !slices.Equal(w.requests(), want) {
			t.Errorf("a search with no session = %s, requests %v; want ok after %v", result(a), w.requests(), want)
		}
		a = run(t, n, searching.Job{Port: "lantern", Revision: 2, Settings: map[string]string{"username": "sam", "password": "nope"}, Call: searching.TestCall{}})
		if got, want := result(a), "*searching.Stopped the site rejected the credentials: Wrong password"; got != want {
			t.Errorf("a Test with a wrong password = %s, want %s", got, want)
		}
	})
}

func TestSpacingKeepsThePortsFloorOnTheFakeClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := lantern("lantern")
		e.RateLimit, e.PageSize = 6100*time.Millisecond, 1
		pages := 0
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/login.php":
				setUID(w, "u1")
			default:
				pages++
				if pages <= 2 {
					respond(w, listing{Results: []item{{ID: strconv.Itoa(pages), Title: fmt.Sprintf("Orchard.Part.%d", pages), Down: 1, Up: 1}}})
					return
				}
				respond(w, listing{})
			}
		}}}
		p := Define(e, Impl[account]{
			Login: FormLogin[account]{DoLogin: func(in *SignIn[account]) error {
				_, err := in.Do(Post(in.BaseURL.JoinPath("login.php")).AddFormParameter("username", in.Settings.Username))
				return err
			}, CheckIfLoginNeeded: signedOut},
			Movie: func(ix *Indexer[account], _ SearchCriteria) (Chain, error) {
				var c Chain
				c.Add(Get(ix.URL("browse.php?page=1")), Get(ix.URL("browse.php?page=2")), Get(ix.URL("browse.php?page=3")))
				return c, nil
			},
			Basic: browse[account], Parse: parseListing[account],
		})
		a := run(t, node(w, p), searching.Job{Port: "lantern", Settings: samAndPW, Call: movieSearch("orchard", nil)})
		if result(a) != "ok" || len(a.Releases) != 2 {
			t.Fatalf("a search over full pages = %v, %s; want both pages' Releases", titles(a.Releases), result(a))
		}
		hits := w.to("lantern.test")
		if len(hits) != 4 {
			t.Fatalf("requests = %v, want the sign-in and three pages", w.requests())
		}
		for i := 1; i < len(hits); i++ {
			if gap := hits[i].at.Sub(hits[i-1].at); gap < e.RateLimit {
				t.Errorf("request %d came %s after the one before, want at least %s", i, gap, e.RateLimit)
			}
		}
	})
}

func TestARedirectedSignInWaitsItsTurnAndKeepsTheCookiesOnTheWay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := lantern("lantern")
		e.RateLimit = 5 * time.Second
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/login.php":
				setUID(w, "u1")
				http.Redirect(w, r, "/index.php", http.StatusFound)
			case "/index.php":
				_, _ = w.Write([]byte("Welcome"))
			case "/browse.php":
				if _, err := r.Cookie("uid"); err != nil {
					respond(w, listing{SignedOut: true})
					return
				}
				respond(w, listing{Results: []item{{ID: "1", Title: "Orchard.2026.1080p", Down: 1, Up: 1}}})
			}
		}}}
		a := run(t, node(w, formPort(e)), searching.Job{Port: "lantern", Settings: samAndPW, Call: movieSearch("orchard", nil)})
		if result(a) != "ok" || len(a.Releases) != 1 {
			t.Fatalf("a search behind a sign-in that redirects home = %v, %s; want its Release", titles(a.Releases), result(a))
		}
		hits := w.to("lantern.test")
		if got, want := w.requests(), []string{"POST lantern.test/login.php", "GET lantern.test/index.php", "GET lantern.test/browse.php"}; !slices.Equal(got, want) {
			t.Fatalf("requests = %v, want %v", got, want)
		}
		for i := 1; i < len(hits); i++ {
			if gap := hits[i].at.Sub(hits[i-1].at); gap < e.RateLimit {
				t.Errorf("%s came %s after the request before, want at least %s", hits[i], gap, e.RateLimit)
			}
		}
	})
}

func TestCredentialHeadersReachOnlyTheIndexersHosts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &world{hosts: map[string]http.HandlerFunc{
			"lantern.test": func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/dl/1":
					http.Redirect(w, r, "https://cdn.test/files/1.torrent", http.StatusFound)
				case "/dl/2":
					http.Redirect(w, r, "/dl/1", http.StatusFound)
				}
			},
			"cdn.test": func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("d4:infoe")) },
		}}
		n := node(w, keyPort(lantern("lantern")))
		for _, link := range []string{"https://lantern.test/dl/2", "https://cdn.test/files/1.torrent"} {
			a := run(t, n, searching.Job{Port: "lantern", Settings: map[string]string{"apiKey": "k3y"}, Call: searching.FetchCall{Link: searching.LinkOf(link)}})
			if got, ok := a.Got.(searching.Torrent); !ok || string(got.Raw) != "d4:infoe" {
				t.Errorf("a fetch of %s = %+v, %s; want the .torrent from the other host", link, a.Got, result(a))
			}
		}
		for _, h := range w.to("lantern.test") {
			if h.header.Get("ApiKey") != "k3y" {
				t.Errorf("%s went without the key", h)
			}
		}
		if cdn := w.to("cdn.test"); len(cdn) != 2 || cdn[0].header.Get("ApiKey") != "" || cdn[1].header.Get("ApiKey") != "" {
			t.Errorf("the other host got %d requests, the first with ApiKey %q; want two with no key", len(cdn), cdn[0].header.Get("ApiKey"))
		}
	})
}

func TestARedirectOffTheIndexersHostsIsNotFollowed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &world{hosts: map[string]http.HandlerFunc{
			"lantern.test": func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "https://evil.test/collect", http.StatusTemporaryRedirect)
			},
			"evil.test": func(http.ResponseWriter, *http.Request) {},
		}}
		n := node(w, formPort(lantern("lantern")), keyPort(lantern("ferry")))
		a := run(t, n, searching.Job{Port: "lantern", Settings: samAndPW, Call: movieSearch("orchard", nil)})
		if got, want := result(a), "*searching.Stopped the site rejected the credentials"; !strings.HasPrefix(got, want) {
			t.Errorf("a sign-in the site redirects away with 307 = %s, want %s", got, want)
		}
		a = run(t, n, searching.Job{Port: "ferry", Settings: map[string]string{"apiKey": "k3y"}, Call: movieSearch("orchard", nil)})
		if got, want := result(a), "*searching.Failing the site redirected the search off the Indexer's hosts"; got != want {
			t.Errorf("a search the site redirects away = %s, want %s", got, want)
		}
		if got := w.to("evil.test"); len(got) != 0 {
			t.Errorf("the other host got %v, want nothing", got)
		}
	})
}

// codePort signs in with a code when the site's form asks for one: the
// site asks for one when its account has two-factor sign-in.
func codePort() Port {
	return Define(lantern("lantern"), Impl[account]{
		Login: CodeLogin[account]{DoLogin: func(in *SignIn[account]) error {
			page, err := in.Do(Get(in.BaseURL.JoinPath("login.php")))
			if err != nil {
				return err
			}
			form := Post(in.BaseURL.JoinPath("login.php")).AddFormParameter("username", in.Settings.Username).AddFormParameter("password", string(in.Settings.Password))
			if strings.Contains(page.Content, "code") {
				code, ok := in.TwoFactorCode()
				if !ok {
					return ErrTwoFactorCodeNeeded
				}
				form.AddFormParameter("code", code)
			}
			r, err := in.Do(form)
			if err != nil {
				return err
			}
			if r.Cookies()["uid"] == "" {
				return IndexerAuthError(r.Content)
			}
			return nil
		}, CheckIfLoginNeeded: signedOut},
		Movie: browse[account], Basic: browse[account], Parse: parseListing[account],
	})
}

// codeSite asks for code when it isn't "", and signs in sam with it.
func codeSite(code string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/login.php" && r.Method == http.MethodGet && code != "":
			_, _ = w.Write([]byte("Enter your code"))
		case r.URL.Path == "/login.php" && r.Method == http.MethodPost:
			if r.FormValue("username") != "sam" || r.FormValue("code") != code {
				_, _ = w.Write([]byte("Wrong code"))
				return
			}
			setUID(w, "u1")
		case r.URL.Path == "/browse.php":
			if _, err := r.Cookie("uid"); err != nil {
				respond(w, listing{SignedOut: true})
				return
			}
			respond(w, listing{Results: []item{{ID: "1", Title: "Orchard.2026.1080p", Down: 1, Up: 1}}})
		}
	}
}

func TestATwoFactorSignInTakesTheKeysCodeOrTheTypedOne(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(time.Until(time.Unix(1111111109, 0)))
		const rfcKey = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": codeSite("081804")}}
		settings := map[string]string{"username": "sam", "password": "pw", "authenticatorKey": strings.ToLower(rfcKey)}
		if a := run(t, node(w, codePort()), searching.Job{Port: "lantern", Settings: settings, Call: searching.TestCall{}}); result(a) != "ok" || a.Report.Items != 1 {
			t.Errorf("a Test with the RFC 6238 key at 1111111109 = %s, %+v; want ok with the code 081804", result(a), a.Report)
		}

		w = &world{hosts: map[string]http.HandlerFunc{"lantern.test": codeSite("424242")}}
		n := node(w, codePort())
		a := run(t, n, searching.Job{Port: "lantern", Settings: samAndPW, Call: searching.TestCall{Code: "424242"}})
		if result(a) != "ok" || a.Session == nil {
			t.Errorf("a Test with a typed code = %s, session %+v; want ok and the session it earned", result(a), a.Session)
		}
		a = run(t, n, searching.Job{Port: "lantern", Revision: 2, Settings: samAndPW, Call: searching.TestCall{Code: "000000"}})
		if got, want := result(a), "*searching.Stopped the site rejected the credentials: Wrong code"; got != want {
			t.Errorf("a Test with a wrong code = %s, want %s", got, want)
		}
	})
}

func TestASignInStopsForACodeOnlyWhenTheSiteAsksForOne(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": codeSite("424242")}}
		a := run(t, node(w, codePort()), searching.Job{Port: "lantern", Settings: samAndPW, Call: movieSearch("orchard", nil)})
		if got, want := result(a), "*searching.Stopped the site needs a two-factor code"; got != want {
			t.Errorf("a sign-in the site asks a code of = %s, want %s", got, want)
		}
		if got := w.requests(); !slices.Equal(got, []string{"GET lantern.test/login.php"}) {
			t.Errorf("requests = %v, want only the sign-in page", got)
		}

		w = &world{hosts: map[string]http.HandlerFunc{"lantern.test": codeSite("")}}
		a = run(t, node(w, codePort()), searching.Job{Port: "lantern", Settings: samAndPW, Call: movieSearch("orchard", nil)})
		if result(a) != "ok" || len(a.Releases) != 1 {
			t.Errorf("a sign-in for an account without two-factor sign-in = %v, %s; want ok", titles(a.Releases), result(a))
		}
	})
}

func TestTheCodeOfAnRFC6238Key(t *testing.T) {
	const key = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	for at, want := range map[int64]string{59: "287082", 1111111109: "081804", 1111111111: "050471", 1234567890: "005924", 2000000000: "279037", 20000000000: "353130"} {
		if got, err := totp(key, time.Unix(at, 0)); got != want || err != nil {
			t.Errorf("the code at %d = %s, %v; want %s", at, got, err, want)
		}
	}
	if got, _ := totp("gezd gnbv gy3t qojq gezd gnbv gy3t qojq", time.Unix(59, 0)); got != "287082" {
		t.Errorf("a key in lower case with spaces gives %s, want 287082", got)
	}
	if _, err := totp("not base32!", time.Unix(59, 0)); err == nil {
		t.Error("a key that isn't base32 gave a code")
	}
}

// cookiePort runs on a pasted uid cookie.
func cookiePort(id ID, rotates bool) Port {
	return Define(lantern(id), Impl[noFields]{
		Login: CookieLogin{Rotates: rotates, CheckIfLoginNeeded: signedOut},
		Movie: browse[noFields], Basic: browse[noFields], Parse: parseListing[noFields],
	})
}

func pasted(uid string) *searching.Session {
	s := heldUID(uid)
	s.Pasted, s.UserAgent = true, "Firefox/140"
	return s
}

func TestAPastedCookieRotatesAndExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rotating := func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie("uid")
			if err != nil || c.Value == "gone" {
				respond(w, listing{SignedOut: true})
				return
			}
			setUID(w, c.Value+"+")
			respond(w, listing{Results: []item{{ID: "1", Title: "Orchard.2026.1080p", Down: 1, Up: 1}}})
		}
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": rotating}}
		a := run(t, node(w, cookiePort("lantern", true)), searching.Job{Port: "lantern", Session: pasted("a"), Call: movieSearch("orchard", nil)})
		if s := a.Session; result(a) != "ok" || s == nil || s.Cookies[0].Value != "a+" || !s.Pasted || s.UserAgent != "Firefox/140" {
			t.Errorf("a search on a rotating cookie = %s, session %+v; want the new cookie as the changed pasted session", result(a), s)
		}
		if ua := w.to("lantern.test")[0].header.Get("User-Agent"); ua != "Firefox/140" {
			t.Errorf("the request went with User-Agent %q, want the pasted one", ua)
		}

		a = run(t, node(w, cookiePort("lantern", false)), searching.Job{Port: "lantern", Session: pasted("a"), Call: movieSearch("orchard", nil)})
		if result(a) != "ok" || a.Session != nil {
			t.Errorf("a search on a cookie that doesn't rotate = %s, session %+v; want ok and no changed session", result(a), a.Session)
		}

		w = &world{hosts: map[string]http.HandlerFunc{"lantern.test": rotating}}
		a = run(t, node(w, cookiePort("lantern", true)), searching.Job{Port: "lantern", Session: pasted("gone"), Call: movieSearch("orchard", nil)})
		if got, want := result(a), "*searching.Stopped the site's cookie expired"; got != want || len(w.requests()) != 1 {
			t.Errorf("a stale pasted cookie = %s after %v, want %s after one request", got, w.requests(), want)
		}
	})
}

// tokenPort signs in to a bearer token at auth.
func tokenPort() Port {
	return Define(lantern("lantern"), Impl[account]{
		Login: TokenLogin[account]{DoLogin: func(in *SignIn[account]) (string, error) {
			r, err := in.Do(Post(in.BaseURL.JoinPath("auth")).SetJSON(map[string]string{"username": in.Settings.Username, "password": string(in.Settings.Password)}))
			if err != nil {
				return "", err
			}
			var res struct{ Token string }
			if err := json.Unmarshal(r.Body, &res); err != nil || res.Token == "" {
				return "", IndexerAuthError("")
			}
			return res.Token, nil
		}},
		Movie: browse[account], Basic: browse[account], Parse: parseListing[account],
	})
}

func TestARefusedTokenSignsInAgainInsideAFetch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/auth":
				var body struct{ Username, Password string }
				_ = json.NewDecoder(r.Body).Decode(&body)
				_, _ = fmt.Fprintf(w, `{"token":"new-for-%s"}`, body.Username)
			case r.Header.Get("Authorization") != "Bearer new-for-sam":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				_, _ = w.Write([]byte("d4:infoe"))
			}
		}}}
		held := &searching.Session{Token: "old", Earned: time.Unix(1, 0)}
		a := run(t, node(w, tokenPort()), searching.Job{Port: "lantern", Settings: samAndPW, Session: held, Call: searching.FetchCall{Link: searching.LinkOf("https://lantern.test/dl/1")}})
		if got, ok := a.Got.(searching.Torrent); !ok || string(got.Raw) != "d4:infoe" {
			t.Fatalf("a fetch with a refused token = %+v, %s; want the .torrent after a sign-in", a.Got, result(a))
		}
		var auths []string
		for _, h := range w.hits {
			auths = append(auths, h.path+" "+h.header.Get("Authorization"))
		}
		if want := []string{"/dl/1 Bearer old", "/auth ", "/dl/1 Bearer new-for-sam"}; !slices.Equal(auths, want) {
			t.Errorf("requests = %q, want %q", auths, want)
		}
		if a.Session == nil || a.Session.Token != "new-for-sam" {
			t.Errorf("the changed session = %+v, want the new token", a.Session)
		}
	})
}

func TestAFetchedTorrentOrMagnetNeverReadsAsLoggedOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := Define(lantern("lantern"), Impl[account]{
			Login: FormLogin[account]{
				DoLogin:            func(*SignIn[account]) error { return IndexerAuthError("signed in") },
				CheckIfLoginNeeded: func(r *IndexerResponse) bool { return !strings.Contains(r.Content, "logout.php") },
			},
			Movie: browse[account], Basic: browse[account], Parse: parseListing[account],
		})
		magnetLink := "magnet:?xt=urn:btih:" + strings.Repeat("ab", 20)
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/dl/2" {
				http.Redirect(w, r, magnetLink, http.StatusFound)
				return
			}
			_, _ = w.Write([]byte("d4:infod4:name1:aee"))
		}}}
		n := node(w, p)
		a := run(t, n, searching.Job{Port: "lantern", Settings: samAndPW, Session: heldUID("u1"), Call: searching.FetchCall{Link: searching.LinkOf("https://lantern.test/dl/1")}})
		if got, ok := a.Got.(searching.Torrent); !ok || string(got.Raw) != "d4:infod4:name1:aee" {
			t.Errorf("a fetch of a .torrent with no logout.php in it = %+v, %s; want the .torrent", a.Got, result(a))
		}
		a = run(t, n, searching.Job{Port: "lantern", Settings: samAndPW, Session: heldUID("u1"), Call: searching.FetchCall{Link: searching.LinkOf("https://lantern.test/dl/2")}})
		if got, ok := a.Got.(searching.Magnet); !ok || got.Link.Reveal() != magnetLink {
			t.Errorf("a fetch redirected to a magnet link = %+v, %s; want the magnet link", a.Got, result(a))
		}
		if want := []string{"GET lantern.test/dl/1", "GET lantern.test/dl/2"}; !slices.Equal(w.requests(), want) {
			t.Errorf("requests = %v, want %v and no sign-in", w.requests(), want)
		}
	})
}

func TestDeriveRunsOncePerRevisionOnANode(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		passkeys := 0
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/passkey":
				passkeys++
				_, _ = fmt.Fprintf(w, "pk%d", passkeys)
			default:
				respond(w, listing{Results: []item{{ID: r.URL.Query().Get("passkey"), Title: "Orchard.2026.1080p", Down: 1, Up: 1}}})
			}
		}}}
		p := Define(lantern("lantern"), Impl[apiKey]{
			Login: NoLogin{}, Basic: browse[apiKey], Parse: parseListing[apiKey],
			Movie: func(ix *Indexer[apiKey], _ SearchCriteria) (Chain, error) {
				return One(Get(ix.URL("browse.php")).AddQueryParam("passkey", ix.Derived["passkey"]))
			},
			Derive: func(ix *Indexer[apiKey]) (map[string]string, error) {
				r, err := ix.Fetch(Get(ix.URL("api/passkey")).AddQueryParam("key", ix.Settings.APIKey))
				if err != nil {
					return nil, err
				}
				return map[string]string{"passkey": r.Content}, nil
			},
		})
		n := node(w, p)
		var links []string
		for _, revision := range []int64{1, 1, 2} {
			a := run(t, n, searching.Job{Port: "lantern", Revision: revision, Settings: map[string]string{"apiKey": "k"}, Call: movieSearch("orchard", nil)})
			if result(a) != "ok" || len(a.Releases) != 1 {
				t.Fatalf("a search at revision %d = %s", revision, result(a))
			}
			links = append(links, a.Releases[0].Link.Reveal())
		}
		if want := []string{"https://lantern.test/dl/pk1", "https://lantern.test/dl/pk1", "https://lantern.test/dl/pk2"}; !slices.Equal(links, want) || passkeys != 2 {
			t.Errorf("links = %v after %d passkey reads, want %v after 2", links, passkeys, want)
		}
	})
}

func TestAnIDTheModeDoesntDeclareFallsBackToTextOrNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, _ *http.Request) {
			respond(w, listing{Results: []item{{ID: "1", Title: "Orchard.2026.1080p", Down: 1, Up: 1}}})
		}}}
		n := node(w, keyPort(lantern("lantern")))
		key := map[string]string{"apiKey": "k"}
		a := run(t, n, searching.Job{Port: "lantern", Settings: key, Call: movieSearch("", map[string]string{"tmdbid": "603"})})
		if result(a) != "ok" || len(a.Releases) != 0 || len(w.hits) != 0 {
			t.Errorf("a search by an undeclared id alone = %v, %s after %v; want an empty ok with no request", titles(a.Releases), result(a), w.requests())
		}
		a = run(t, n, searching.Job{Port: "lantern", Settings: key, Call: movieSearch("orchard", map[string]string{"tmdbid": "603"})})
		a2 := run(t, n, searching.Job{Port: "lantern", Settings: key, Call: movieSearch("", map[string]string{"imdbid": "133093", "tmdbid": "603"})})
		a3 := run(t, n, searching.Job{Port: "lantern", Settings: key, Call: searching.SearchCall{Query: searching.Query{Mode: "tvsearch", Text: "orchard"}}})
		if result(a) != "ok" || result(a2) != "ok" || result(a3) != "ok" || len(a3.Releases) != 0 {
			t.Fatalf("searches = %s, %s, %s; want ok, and nothing for a mode the port lacks", result(a), result(a2), result(a3))
		}
		if got := w.hits; len(got) != 2 || got[0].query.Encode() != "q=orchard" || got[1].query.Encode() != "imdb=tt0133093" {
			t.Errorf("requests = %v, want a text search and an IMDb search with neither TMDB id", got)
		}
	})
}

func TestEachPortErrorLandsInItsFamilysOutcome(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		answer := "auth"
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/login.php" {
				setUID(w, "u1")
				return
			}
			if answer == "limit" {
				w.Header().Set("Retry-After", "90")
			}
			respond(w, listing{Error: answer})
		}}}
		n := node(w, keyPort(lantern("lantern")), formPort(lantern("ferry")), cookiePort("marsh", false))
		for _, c := range []struct {
			port, answer, want string
		}{
			{"lantern", "auth", "*searching.Stopped the Indexer refused the account: Your key is wrong"},
			{"ferry", "auth", "*searching.Stopped the site rejected the credentials: Your key is wrong"},
			{"marsh", "auth", "*searching.Stopped the site's cookie expired"},
			{"lantern", "limit", "*searching.Limited the Indexer limited requests for 1m30s"},
			{"lantern", "Down for maintenance", "*searching.Failing Down for maintenance"},
			{"brook", "", "*searching.Stopped SlipStream no longer supports the site: the site closed"},
		} {
			answer = c.answer
			a := run(t, n, searching.Job{Port: c.port, Settings: map[string]string{"apiKey": "k", "username": "sam", "password": "pw"}, Call: movieSearch("orchard", nil)})
			if got := result(a); got != c.want {
				t.Errorf("%s answering %q = %s, want %s", c.port, c.answer, got, c.want)
			}
		}
	})
}

func TestReleasesCarryTheSitesFlagsAndTheFactorsFlags(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const hash = "0123456789abcdef0123456789abcdef01234567"
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, _ *http.Request) {
			respond(w, listing{Results: []item{
				{ID: "1", Title: "Orchard.Free", Down: 0, Up: 2, Flags: []string{"internal"}, Hash: hash},
				{ID: "2", Title: "Orchard.Quarter", Down: 0.25, Up: 1},
				{ID: "3", Title: "Orchard.Half", Down: 0.5, Up: 1},
				{ID: "4", Title: "Orchard.ThreeQuarters", Down: 0.75, Up: 1},
				{ID: "5", Title: "Orchard.Full", Down: 1, Up: 1, Flags: []string{"scene"}},
				{ID: "5", Title: "Orchard.Full.Again", Down: 1, Up: 1},
				{ID: "6", Title: " ", Down: 1, Up: 1},
			}})
		}}}
		a := run(t, node(w, keyPort(lantern("lantern"))), searching.Job{Port: "lantern", Settings: map[string]string{"apiKey": "k"}, Call: movieSearch("orchard", nil)})
		var got []string
		for _, r := range a.Releases {
			got = append(got, fmt.Sprint(r.Title, r.Flags))
		}
		want := []string{
			"Orchard.Free[internal freeleech doubleupload]", "Orchard.Quarter[freeleech75]", "Orchard.Half[halfleech]",
			"Orchard.ThreeQuarters[freeleech25]", "Orchard.Full[scene]",
		}
		if !slices.Equal(got, want) {
			t.Errorf("Releases = %v, want %v", got, want)
		}
		r := a.Releases[0]
		if r.InfoHash == nil || r.InfoHash.String() != hash || r.Leechers != 2 || r.Link.Reveal() != "https://lantern.test/dl/1" || !slices.Equal(r.Categories, []int{2000, 100001}) {
			t.Errorf("the first Release = %+v", r)
		}
	})
}

func TestAPortReadsDatesOnTheJobsClockInUTC(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		started := time.Now()
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, _ *http.Request) {
			respond(w, listing{Results: []item{{ID: "1", Title: "2 hours ago", Down: 1, Up: 1}, {ID: "2", Title: "2026-11-06 10:00", Down: 1, Up: 1}}})
		}}}
		p := Define(lantern("lantern"), Impl[apiKey]{
			Login: NoLogin{}, Movie: browse[apiKey], Basic: browse[apiKey],
			Parse: func(ix *Indexer[apiKey], r *IndexerResponse) ([]TorrentInfo, error) {
				out, err := parseListing(ix, r)
				for i := range out {
					if out[i].PublishDate, err = datetimeutil.FromUnknown(out[i].Title, ix.Clock); err != nil {
						return nil, err
					}
				}
				return out, err
			},
		})
		a := run(t, node(w, p), searching.Job{Port: "lantern", Settings: map[string]string{"apiKey": "k"}, Call: movieSearch("orchard", nil)})
		want := []time.Time{started.Add(-2 * time.Hour), time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)}
		if len(a.Releases) != 2 || !a.Releases[0].Published.Equal(want[0]) || !a.Releases[1].Published.Equal(want[1]) {
			t.Fatalf("Releases = %+v, want them published at %v", a.Releases, want)
		}
	})
}

func TestAPortsEncodingReadsAndWritesItsText(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e := lantern("lantern")
		e.Encoding = "windows-1251"
		var posted string
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/login.php":
				body, _ := io.ReadAll(r.Body)
				posted = string(body)
				setUID(w, "u1")
			default:
				_, _ = w.Write([]byte("{\"results\":[{\"id\":\"1\",\"title\":\"\xc8\xe2\xe0\xed\",\"down\":1,\"up\":1}]}"))
			}
		}}}
		p := Define(e, Impl[account]{
			Login: FormLogin[account]{DoLogin: func(in *SignIn[account]) error {
				_, err := in.Do(Post(in.BaseURL.JoinPath("login.php")).AddFormParameter("username", in.Settings.Username))
				return err
			}, CheckIfLoginNeeded: signedOut},
			Movie: browse[account], Basic: browse[account],
			Parse: func(_ *Indexer[account], r *IndexerResponse) ([]TorrentInfo, error) {
				var l listing
				if err := json.Unmarshal([]byte(r.Content), &l); err != nil {
					return nil, IndexerError("the site's response isn't JSON")
				}
				return []TorrentInfo{{Title: l.Results[0].Title, GUID: "1", DownloadVolumeFactor: 1, UploadVolumeFactor: 1}}, nil
			},
		})
		a := run(t, node(w, p), searching.Job{Port: "lantern", Settings: map[string]string{"username": "Иван", "password": "pw"}, Call: movieSearch("orchard", nil)})
		if result(a) != "ok" || !slices.Equal(titles(a.Releases), []string{"Иван"}) {
			t.Errorf("a windows-1251 page = %v, %s; want the title read as Иван", titles(a.Releases), result(a))
		}
		if posted != "username=%C8%E2%E0%ED" {
			t.Errorf("the sign-in posted %q, want Иван in windows-1251", posted)
		}
	})
}

func TestATestSignsInAndReportsTheStaticCapabilities(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": lanternSite("u1", "u2")}}
		a := run(t, node(w, formPort(lantern("lantern"))), searching.Job{Port: "lantern", Settings: samAndPW, Session: heldUID("u1"), Call: searching.TestCall{}})
		if result(a) != "ok" || a.Report == nil || a.Report.Items != 0 {
			t.Fatalf("a Test of a quiet site = %s, %+v; want ok with zero items", result(a), a.Report)
		}
		if want := []string{"POST lantern.test/login.php", "GET lantern.test/browse.php"}; !slices.Equal(w.requests(), want) {
			t.Errorf("requests = %v, want a sign-in although the Node held a session, then Recent: %v", w.requests(), want)
		}
		if got := fmt.Sprint(a.Report.Capabilities); got != "{map[movie:[q imdbid] search:[q]] [2000 100001]}" {
			t.Errorf("capabilities = %s", got)
		}
	})
}

func TestAPortThatPanicsFailsItsJobAndTheNodeRunsOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := &world{hosts: map[string]http.HandlerFunc{"lantern.test": func(w http.ResponseWriter, _ *http.Request) {
			respond(w, listing{Results: []item{{ID: "1", Title: "Orchard.2026.1080p", Down: 1, Up: 1}}})
		}}}
		broken := Define(lantern("ferry"), Impl[apiKey]{
			Login: NoLogin{}, Movie: browse[apiKey], Basic: browse[apiKey],
			Parse: func(*Indexer[apiKey], *IndexerResponse) ([]TorrentInfo, error) {
				panic("selector td:nth-of-type( doesn't compile")
			},
		})
		n := node(w, broken, keyPort(lantern("lantern")))
		key := map[string]string{"apiKey": "k"}
		a := run(t, n, searching.Job{Port: "ferry", Settings: key, Call: movieSearch("orchard", nil)})
		if got, want := result(a), "*searching.Failing the port's code failed: selector td:nth-of-type( doesn't compile"; got != want {
			t.Errorf("a port that panics = %s, want %s", got, want)
		}
		if a := run(t, n, searching.Job{Port: "lantern", Settings: key, Call: movieSearch("orchard", nil)}); result(a) != "ok" || len(a.Releases) != 1 {
			t.Errorf("another port's search after the panic = %v, %s; want ok", titles(a.Releases), result(a))
		}
	})
}

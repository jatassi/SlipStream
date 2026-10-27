package site

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// The labels of the sites that each run a Cardigann definition of the
// same id.
const (
	// Larkspur signs in with a form and an image captcha, lists HTML rows
	// and serves a .torrent only to a session that thanked for it first.
	Larkspur = "larkspur"
	// Quarry checks an API key with a get sign-in and lists JSON rows.
	Quarry = "quarry"
	// Cinder takes a pasted session cookie behind a browser challenge and
	// downloads by info hash.
	Cinder = "cinder"
	// Marsh signs in with a post in windows-1251 and lists XML rows.
	Marsh = "marsh"
	// Ferry signs in with its passkey in one URL, lists magnet links and
	// picks its .torrent with download selectors.
	Ferry = "ferry"
)

func catalogSite(label string) bool { return slices.Contains(scenario.FixtureSites["catalog"], label) }

var (
	// Expire signs every session of a catalog site out, and makes a pasted
	// session cookie stop working: {"site": "<label>"}.
	Expire = fake.CueSpec{Name: "fixture-site-expire"}
	// Maintenance puts a catalog site in maintenance, where every page
	// responds 503, or takes it out: {"site": "<label>", "on": true}.
	Maintenance = fake.CueSpec{Name: "fixture-site-maintenance"}
)

var (
	// CaptchaShown records each captcha image a site served: its site and
	// the text it shows.
	CaptchaShown = fake.CaptureSpec{Name: "captchas", Kind: "fixture-site.captcha"}
	// SignedIn records each sign-in a site accepted: its site and the
	// sign-in's method.
	SignedIn = fake.CaptureSpec{Name: "signins", Kind: "fixture-site.signin"}
)

// SiteArgs name a catalog site for a cue.
type SiteArgs struct {
	Site string `json:"site"`
	On   bool   `json:"on,omitempty"`
}

// Captcha is a CaptchaShown record's data.
type Captcha struct {
	Site string `json:"site"`
	Text string `json:"text"`
}

// SignIn is a SignedIn record's data.
type SignIn struct {
	Site   string `json:"site"`
	Method string `json:"method"`
}

// derive is a value only a world with seed derives from parts.
func derive(seed string, parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(append([]string{seed, "fixture-site"}, parts...), "\x00")))
	return hex.EncodeToString(sum[:])
}

// Username is the account name on every catalog site.
const Username = "slipstream"

// Password is the account's password on the catalog site label in a
// world with Scenario s.
func Password(s scenario.Scenario, label string) string {
	return derive(s.Seed, "password", label)[:16]
}

// Key is the account's API key on Quarry, or its passkey on Ferry, in a
// world with Scenario s.
func Key(s scenario.Scenario, label string) string { return derive(s.Seed, "key", label)[:32] }

// SessionCookie is the account's session cookie on Cinder in a world with
// Scenario s, as an Admin pastes it: name=value.
func SessionCookie(s scenario.Scenario) string {
	return sessionCookie(Cinder) + "=" + derive(s.Seed, "cookie", Cinder)[:24]
}

// ClearanceCookie is the name of the cookie that clears Cinder's browser
// challenge.
const ClearanceCookie = "cf_clearance"

// Clearance clears Cinder's browser challenge for the requests of origin,
// a Node's name, that send userAgent, in a world with Scenario s.
func Clearance(s scenario.Scenario, origin, userAgent string) string {
	return derive(s.Seed, "clearance", origin, userAgent)[:40]
}

// BrowserUserAgent is the User-Agent of the browser an Admin pastes
// Cinder's cookie from.
const BrowserUserAgent = "Mozilla/5.0 (fixture)"

// Credentials are what an Admin enters to add an Indexer for the catalog
// site label that searches from Node origin, in a world with Scenario s:
// its settings, and the browser cookie it pastes with BrowserUserAgent, ""
// for none.
func Credentials(s scenario.Scenario, label, origin string) (map[string]string, string) {
	switch label {
	case Quarry:
		return map[string]string{"apikey": Key(s, Quarry)}, ""
	case Ferry:
		return map[string]string{"passkey": Key(s, Ferry)}, ""
	case Cinder:
		return map[string]string{}, SessionCookie(s) + "; " + ClearanceCookie + "=" + Clearance(s, origin, BrowserUserAgent)
	}
	return map[string]string{"username": Username, "password": Password(s, label)}, ""
}

// session is one sign-in to a catalog site, valid only from the Node that
// signed in.
type session struct {
	site, origin string
}

func sessionCookie(label string) string { return label + "_session" }

// mint is a new token, unique in this run.
func (s *site) mint(parts ...string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.minted++
	return derive(s.scenario.Seed, append(parts, strconv.Itoa(s.minted))...)[:32]
}

// signIn starts a session of label for the request's origin, sets its
// cookie on w and records the sign-in.
func (s *site) signIn(w http.ResponseWriter, r *http.Request, label, method string) {
	token := s.mint("session", label)
	s.mu.Lock()
	s.sessions[token] = session{site: label, origin: fake.OriginOf(r.Context())}
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: sessionCookie(label), Value: token, Path: "/", HttpOnly: true})
	if s.signins != nil {
		s.signins.Record(r.Context(), SignIn{Site: label, Method: method})
	}
}

// signedIn is the token of r's session on label, false when r carries none
// the site accepts from its origin.
func (s *site) signedIn(r *http.Request, label string) (string, bool) {
	c, err := r.Cookie(sessionCookie(label))
	if err != nil {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	got, ok := s.sessions[c.Value]
	return c.Value, ok && got == session{site: label, origin: fake.OriginOf(r.Context())}
}

func (s *site) expire(_ context.Context, a SiteArgs) (any, error) {
	if !catalogSite(a.Site) {
		return nil, fmt.Errorf("fixture-site has no catalog site %q", a.Site)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for token, got := range s.sessions {
		if got.site == a.Site {
			delete(s.sessions, token)
		}
	}
	s.expired[a.Site] = true
	return struct{}{}, nil
}

func (s *site) maintain(_ context.Context, a SiteArgs) (any, error) {
	if !catalogSite(a.Site) {
		return nil, fmt.Errorf("fixture-site has no catalog site %q", a.Site)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.down[a.Site] = a.On
	return struct{}{}, nil
}

// serving is h while label isn't in maintenance.
func (s *site) serving(label string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		down := s.down[label]
		s.mu.Unlock()
		if down {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = fmt.Fprintf(w, "<html><body><h1>%s is down for maintenance.</h1></body></html>\n", label)
			return
		}
		h(w, r)
	}
}

// found are the releases the feeds list whose IMDb id is query, or whose
// name holds every word of it.
func (s *site) found(query string) []Release {
	var out []Release
	for _, rel := range s.releases {
		if !s.shown(rel) {
			continue
		}
		if strings.HasPrefix(query, "tt") && rel.IMDbID == query || matches(rel.Name, query) {
			out = append(out, rel)
		}
	}
	return out
}

// byHash is the release whose info hash is hash.
func (s *site) byHash(hash string) (Release, bool) {
	i := slices.IndexFunc(s.releases, func(rel Release) bool { return rel.InfoHash == hash })
	if i < 0 {
		return Release{}, false
	}
	return s.releases[i], true
}

// page writes an HTML page.
func page(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!DOCTYPE html><html><body>" + body + "</body></html>\n"))
}

// member writes an HTML page a signed-in account sees, its header
// linking the sign-out.
func member(w http.ResponseWriter, body string) {
	page(w, `<a class="logout" href="logout.php">Sign out</a>`+body)
}

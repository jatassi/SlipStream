package ports

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// FixtureUser is the one username a fixture holds, and FixtureEmail the
// one email.
const (
	FixtureUser  = "fixture-user"
	FixtureEmail = "fixture-user@fixture-site.fake.localhost"
)

var (
	placeholder = regexp.MustCompile(`\{\{[A-Za-z][A-Za-z0-9]*\}\}`)
	// secretRun is a run of letters and digits long enough to be a key,
	// a token or a cookie, when it holds both.
	secretRun = regexp.MustCompile(`[A-Za-z0-9]{16,}`)
	hexHash   = regexp.MustCompile(`(?i)\b[0-9a-f]{40}\b`)
	// releaseName is a dotted name with a year, a season or a
	// resolution in it, as a release's title is.
	releaseName = regexp.MustCompile(`(?i)\b[a-z0-9'&!+-]+(?:\.[a-z0-9'&!+-]+)*\.(?:(?:19|20)\d\d|s\d{1,3}(?:e\d{1,4})*|\d{3,4}[pi])(?:\.[a-z0-9'&!+-]+)*\b`)
	// userKey is a JSON key whose value names a person on the site.
	userKey = regexp.MustCompile(`(?i)"(user(?:_?name)?|uploader|owner|author|login)"\s*:\s*"([^"]*)"`)
)

// credentialNames are the setting, query, form, JSON and session names
// whose values must be placeholders, lowercased.
var credentialNames = []string{
	"apikey", "api_key", "passkey", "torrent_pass", "authkey", "rsskey", "apiuser", "password", "pass", "pid", "pin",
	"cookie", "authenticatorkey", "token", "secret", "key", "uid",
}

// userNames are the names whose values must be FixtureUser or
// FixtureEmail, lowercased.
var userNames = []string{"username", "user", "login", "uname", "email"}

// secretHeaders are the headers whose values must be placeholders.
var secretHeaders = []string{"Authorization", "Cookie", "Set-Cookie", "X-Api-Key", "Api-Key", "Apikey", "Apiuser"}

// Scrub is the scrub check. It reports every value in f shaped like a
// passkey, API key, token or cookie that isn't a placeholder, a username
// that isn't FixtureUser, a title outside the shared release catalog's
// fictional titles and an info hash Mint didn't make. The record tool
// writes nothing Scrub reports.
func Scrub(f Fixture) []string {
	s := scrubber{}
	for k, v := range sorted(f.Settings) {
		s.named("settings", k, v)
	}
	for _, sess := range append([]*Session{f.Session}, sessions(f.Cases)...) {
		if sess != nil {
			s.cookies("session cookie", sess.Cookie)
			s.secret("session token", sess.Token)
			s.text("session", sess.UserAgent)
		}
	}
	for _, c := range f.Cases {
		at := "case " + c.Name
		if q := c.Call.Search; q != nil {
			s.title(at+" search", q.Text)
			for k, v := range sorted(q.IDs) {
				s.text(at+" search "+k, v)
			}
		}
		s.text(at+" fetch", c.Call.Fetch)
		for i, x := range c.Exchanges {
			ex := fmt.Sprintf("%s exchange %d", at, i+1)
			s.text(ex+" path", x.Request.Path)
			s.headers(ex+" request", x.Request.Headers)
			s.headers(ex+" response", x.Response.Headers)
			for k, v := range sorted(x.Request.Query) {
				s.named(ex+" query", k, v)
			}
			for k, v := range sorted(x.Request.Form) {
				s.named(ex+" form", k, v)
			}
			s.jsonFields(ex+" json", x.Request.JSON)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(f.Bodies)) {
		s.body("bodies/"+name, string(f.Bodies[name]))
	}
	for _, name := range slices.Sorted(maps.Keys(f.Wants)) {
		s.want("want/"+name+".json", f.Wants[name])
	}
	return s.found
}

func sessions(cs []Case) []*Session {
	var out []*Session
	for _, c := range cs {
		out = append(out, c.Session)
	}
	return out
}

type scrubber struct{ found []string }

func (s *scrubber) report(at, format string, args ...any) {
	s.found = append(s.found, at+": "+fmt.Sprintf(format, args...))
}

// named checks a value by its name, then as text. A value with a space
// is no username, such as a submit button's "Log in!" named login.
func (s *scrubber) named(at, name, v string) {
	lower := strings.ToLower(name)
	switch {
	case slices.Contains(userNames, lower) && !strings.ContainsFunc(v, unicode.IsSpace):
		if v != FixtureUser && v != FixtureEmail {
			s.report(at, "%s is %q, which isn't %s or %s", name, v, FixtureUser, FixtureEmail)
		}
	case slices.Contains(credentialNames, lower):
		s.secret(at+" "+name, v)
	default:
		s.text(at+" "+name, v)
	}
}

// secret checks a credential: it must be placeholders and the words
// around them.
func (s *scrubber) secret(at, v string) {
	if v == "" {
		return
	}
	if !placeholder.MatchString(v) {
		s.report(at, "%q is a credential. Put a placeholder such as {{passkey}} in its place.", v)
		return
	}
	s.text(at, v)
}

// cookies checks name=value pairs joined by ";": each value must be a
// placeholder, and attributes such as Path pass.
func (s *scrubber) cookies(at, v string) {
	for pair := range strings.SplitSeq(v, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if !ok || slices.Contains(cookieAttributes, strings.ToLower(name)) {
			continue
		}
		s.secret(at+" "+name, value)
	}
}

var cookieAttributes = []string{"path", "domain", "expires", "max-age", "samesite"}

func (s *scrubber) headers(at string, h map[string]string) {
	for k, v := range sorted(h) {
		switch canonical := http.CanonicalHeaderKey(k); {
		case canonical == "Cookie" || canonical == "Set-Cookie":
			for line := range strings.SplitSeq(v, "\n") {
				first, _, _ := strings.Cut(line, ";")
				s.cookies(at+" header "+k, first)
			}
		case slices.Contains(secretHeaders, canonical):
			s.secret(at+" header "+k, v)
		default:
			s.text(at+" header "+k, v)
		}
	}
}

func (s *scrubber) jsonFields(at string, m map[string]any) {
	for _, k := range slices.Sorted(maps.Keys(m)) {
		switch v := m[k].(type) {
		case map[string]any:
			s.jsonFields(at+"."+k, v)
		case string:
			s.named(at, k, v)
		default:
			s.text(at+"."+k, fmt.Sprint(v))
		}
	}
}

// text checks free text for a key-shaped run, an unminted info hash and
// a release name outside the catalog.
func (s *scrubber) text(at, v string) {
	bare := placeholder.ReplaceAllString(v, "")
	for _, h := range hexHash.FindAllString(bare, -1) {
		if !Minted(h) {
			s.report(at, "info hash %s isn't one the fixture catalog minted. Rewrite it with the record tool.", h)
		}
	}
	for _, run := range secretRun.FindAllString(hexHash.ReplaceAllString(bare, ""), -1) {
		if mixed(run) {
			s.report(at, "%q is shaped like a key, a token or a cookie. Put a placeholder such as {{passkey}} in its place.", run)
		}
	}
	for _, name := range releaseName.FindAllString(bare, -1) {
		s.title(at, name)
	}
}

func mixed(run string) bool {
	var letters, digits int
	for _, r := range run {
		if r >= '0' && r <= '9' {
			digits++
		} else {
			letters++
		}
	}
	return letters >= 2 && digits >= 2
}

func (s *scrubber) title(at, title string) {
	if title == "" {
		return
	}
	ok, err := Fictional(title)
	switch {
	case err != nil:
		s.report(at, "%v", err)
	case !ok:
		s.report(at, "title %q isn't one of the release catalog's fictional titles. Rewrite it with the record tool.", title)
	}
}

func (s *scrubber) body(at, body string) {
	for _, m := range userKey.FindAllStringSubmatch(body, -1) {
		if m[2] != FixtureUser && m[2] != FixtureEmail {
			s.report(at, "%s is %q, which isn't %s or %s", m[1], m[2], FixtureUser, FixtureEmail)
		}
	}
	s.text(at, body)
}

// want checks a golden: its text, and that every Release title is
// fictional.
func (s *scrubber) want(at string, raw []byte) {
	s.text(at, string(raw))
	var w struct {
		Releases []struct {
			Title string `json:"title"`
		} `json:"releases"`
	}
	if err := json.Unmarshal(raw, &w); err != nil {
		s.report(at, "the golden isn't JSON: %v", err)
		return
	}
	for _, r := range w.Releases {
		s.title(at, r.Title)
	}
}

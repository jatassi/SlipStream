package searching

import (
	"cmp"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"
)

// jar is the cookie jar of one request and the redirects it follows: an
// http.CookieJar over a session's cookies, each sent only to its own host.
// It records each Set-Cookie it took, for a Cardigann exchange that reads
// them.
type jar struct {
	mu      sync.Mutex
	cookies []Cookie
	set     []string
	changed bool
}

func newJar(cookies []Cookie) *jar { return &jar{cookies: slices.Clone(cookies)} }

func (j *jar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := time.Now()
	var out []*http.Cookie
	for _, c := range j.cookies {
		if c.matches(u, now) {
			out = append(out, &http.Cookie{Name: c.Name, Value: c.Value}) //nolint:gosec // a request's cookie carries only its name and value
		}
	}
	return out
}

// SetCookies keeps each cookie a response from u set, dropping one for a
// domain u isn't under and one that has expired.
func (j *jar) SetCookies(u *url.URL, cs []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, c := range cs {
		j.set = append(j.set, c.String())
	}
	j.changed = j.changed || len(cs) > 0
	j.put(u, cs)
}

func (j *jar) put(u *url.URL, cs []*http.Cookie) {
	now := time.Now()
	host := strings.ToLower(u.Hostname())
	for _, c := range cs {
		k := Cookie{Name: c.Name, Value: c.Value, Domain: host, Path: cmp.Or(c.Path, "/"), Secure: c.Secure, HostOnly: true}
		if d := strings.ToLower(strings.TrimPrefix(c.Domain, ".")); d != "" {
			if host != d && !strings.HasSuffix(host, "."+d) {
				continue
			}
			k.Domain, k.HostOnly = d, false
		}
		switch {
		case c.MaxAge > 0:
			k.Expires = now.Add(time.Duration(c.MaxAge) * time.Second)
		case !c.Expires.IsZero():
			k.Expires = c.Expires
		}
		j.cookies = slices.DeleteFunc(j.cookies, func(o Cookie) bool { return o.Name == k.Name && o.Domain == k.Domain && o.Path == k.Path })
		if c.MaxAge >= 0 && (k.Expires.IsZero() || k.Expires.After(now)) {
			j.cookies = append(j.cookies, k)
		}
	}
}

// add puts pairs from a Cookie header in the jar as host-only cookies of
// host.
func (j *jar) add(header, host string) error {
	if header == "" {
		return nil
	}
	cs, err := http.ParseCookie(header)
	if err != nil {
		return fmt.Errorf("read the sign-in's cookies: %w", err)
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	j.put(&url.URL{Scheme: "https", Host: host}, cs)
	return nil
}

// held is what the jar holds now, and whether a response changed it.
func (j *jar) held() ([]Cookie, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return slices.Clone(j.cookies), j.changed
}

// taken is every Set-Cookie the jar took, as header values.
func (j *jar) taken() []string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return slices.Clone(j.set)
}

// matches reports whether c goes with a request to u at now: u's host is
// its domain, or under it unless it is host-only, and it hasn't expired.
func (c Cookie) matches(u *url.URL, now time.Time) bool {
	host, domain := strings.ToLower(u.Hostname()), strings.ToLower(strings.TrimPrefix(c.Domain, "."))
	switch {
	case !c.Expires.IsZero() && !c.Expires.After(now):
		return false
	case c.Secure && u.Scheme != "https":
		return false
	case host == domain:
		return true
	}
	return !c.HostOnly && strings.HasSuffix(host, "."+domain)
}

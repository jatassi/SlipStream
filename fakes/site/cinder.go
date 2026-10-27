package site

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
)

func (s *site) cinder() fake.Routes {
	return fake.Routes{
		"index":   {Pattern: "GET /index.php", Handler: s.cleared(s.cinderIndex)},
		"search":  {Pattern: "GET /torrents.php", Handler: s.cleared(s.cinderSearch)},
		"details": {Pattern: "GET /details.php", Handler: s.cleared(s.cinderDetails)},
	}
}

// cleared is h behind Cinder's browser challenge: a request without the
// clearance its origin earned with its User-Agent gets the challenge page.
func (s *site) cleared(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(ClearanceCookie)
		if err == nil && c.Value == Clearance(s.scenario, fake.OriginOf(r.Context()), r.UserAgent()) {
			h(w, r)
			return
		}
		w.Header().Set("Server", "cloudflare")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<!DOCTYPE html><html><head><title>Just a moment...</title></head><body>Checking your browser.</body></html>\n"))
	}
}

// cinderMember reports whether r carries the account's pasted session
// cookie, and it still works.
func (s *site) cinderMember(r *http.Request) bool {
	name, value, _ := strings.Cut(SessionCookie(s.scenario), "=")
	c, err := r.Cookie(name)
	s.mu.Lock()
	defer s.mu.Unlock()
	return err == nil && c.Value == value && !s.expired[Cinder]
}

func (s *site) cinderIndex(w http.ResponseWriter, r *http.Request) {
	if !s.cinderMember(r) {
		page(w, "<p>Sign in with your browser.</p>")
		return
	}
	member(w, "<p>Welcome back.</p>")
}

func (s *site) cinderSearch(w http.ResponseWriter, r *http.Request) {
	if !s.cinderMember(r) {
		http.Redirect(w, r, "/index.php", http.StatusFound)
		return
	}
	var rows strings.Builder
	for _, rel := range s.found(r.URL.Query().Get("q")) {
		fmt.Fprintf(&rows, `<div class="torrent"><span class="cat">1</span><a class="name" href="details.php?id=%[1]s">%[2]s</a>`+
			`<span class="hash">%[1]s</span><span class="size">%[3]d B</span><span class="seeders">%[4]d</span><span class="leechers">%[5]d</span></div>`,
			rel.InfoHash, html.EscapeString(rel.Name), rel.Size, rel.Seeders, rel.Leechers)
	}
	member(w, rows.String())
}

func (s *site) cinderDetails(w http.ResponseWriter, r *http.Request) {
	if !s.cinderMember(r) {
		http.Redirect(w, r, "/index.php", http.StatusFound)
		return
	}
	rel, ok := s.byHash(r.URL.Query().Get("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	member(w, fmt.Sprintf(`<h1>%s</h1><span class="hash">%s</span>`, html.EscapeString(rel.Name), rel.InfoHash))
}

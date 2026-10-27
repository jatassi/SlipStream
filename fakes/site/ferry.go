package site

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
)

func (s *site) ferry() fake.Routes {
	return fake.Routes{
		"signin":   {Pattern: "GET /login/{passkey}", Handler: s.ferrySignIn},
		"search":   {Pattern: "GET /browse", Handler: s.ferryBrowse},
		"details":  {Pattern: "GET /torrent/{hash}", Handler: s.ferryDetails},
		"mirror":   {Pattern: "GET /mirror/{file}", Handler: s.ferryMirror},
		"download": {Pattern: "GET /dl/{file}", Handler: s.ferryDownload},
	}
}

func (s *site) ferrySignIn(w http.ResponseWriter, r *http.Request) {
	if r.PathValue("passkey") != Key(s.scenario, Ferry) {
		page(w, "<p>That link doesn't sign anyone in.</p>")
		return
	}
	s.signIn(w, r, Ferry, "oneurl")
	http.Redirect(w, r, "/browse", http.StatusFound)
}

// ferryBrowse lists each release with its magnet link, its size written
// with thousands separators, its peers as seeders/leechers and its age.
func (s *site) ferryBrowse(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.signedIn(r, Ferry); !ok {
		page(w, `<a href="/login">Sign in with your passkey link.</a>`)
		return
	}
	var b strings.Builder
	b.WriteString("<ul>")
	for _, rel := range s.found(r.URL.Query().Get("q")) {
		magnet, err := s.magnet(rel, fake.OriginOf(r.Context()))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(&b, `<li class="release"><span class="name"> %[1]s [ferry]</span><a class="page" href="/torrent/%[2]s">Page</a>`+
			`<a class="magnet" href="%[3]s">Magnet</a><span class="size">%[4]s</span><span class="peers">%[5]d/%[6]d</span><span class="age">2 hours ago</span></li>`,
			html.EscapeString(rel.Name), rel.InfoHash, html.EscapeString(magnet), thousands(rel.Size), rel.Seeders, rel.Leechers)
	}
	b.WriteString("</ul>")
	member(w, b.String())
}

// thousands writes n with a comma between each group of three digits.
func thousands(n int64) string {
	d := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, c := range d {
		if i > 0 && (len(d)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}

// ferryDetails links a release's .torrent twice: first at a mirror that
// serves a page in its place, then at the site itself.
func (s *site) ferryDetails(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.signedIn(r, Ferry); !ok {
		http.Redirect(w, r, "/browse", http.StatusFound)
		return
	}
	hash := r.PathValue("hash")
	if _, ok := s.byHash(hash); !ok {
		http.NotFound(w, r)
		return
	}
	member(w, fmt.Sprintf(`<a class="mirror" href="/mirror/%[1]s.torrent">Mirror</a><a class="torrent" href="/dl/%[1]s.torrent">Download</a>`, hash))
}

func (s *site) ferryMirror(w http.ResponseWriter, _ *http.Request) {
	page(w, "<p>The mirror is offline.</p>")
}

func (s *site) ferryDownload(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.signedIn(r, Ferry); !ok {
		http.Redirect(w, r, "/browse", http.StatusFound)
		return
	}
	hash, _ := strings.CutSuffix(r.PathValue("file"), ".torrent")
	rel, ok := s.byHash(hash)
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.torrent(w, r, rel)
}

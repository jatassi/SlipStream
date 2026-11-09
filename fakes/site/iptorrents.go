package site

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// IPTorrentsCookie is the account's session cookie on IPTorrents in a
// world with Scenario s, as an Admin pastes it from a browser that sends
// BrowserUserAgent.
func IPTorrentsCookie(s scenario.Scenario) string {
	return "uid=" + derive(s.Seed, "uid", IPTorrents)[:6] + "; pass=" + derive(s.Seed, "cookie", IPTorrents)[:32]
}

func (s *site) ipTorrents() fake.Routes {
	return fake.Routes{
		"search":   {Pattern: "GET /t", Handler: s.iptSearch},
		"download": {Pattern: "GET /download.php/{id}/{file}", Handler: s.iptDownload},
	}
}

// iptMember reports whether r carries the pasted cookie, from the browser
// it came from, and it still works.
func (s *site) iptMember(r *http.Request) bool {
	for pair := range strings.SplitSeq(IPTorrentsCookie(s.scenario), "; ") {
		name, value, _ := strings.Cut(pair, "=")
		if c, err := r.Cookie(name); err != nil || c.Value != value {
			return false
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return r.UserAgent() == BrowserUserAgent && !s.expired[IPTorrents]
}

// iptPage writes an IPTorrents page. A member's links the sign-out,
// lout.php, which the port reads as signed in.
func iptPage(w http.ResponseWriter, member bool, body string) {
	nav := `<form action="/take_login.php" method="post"><input name="username"><input type="password" name="password"></form>`
	if member {
		nav = `<div id="nav"><a href="/u/` + Username + `">` + Username + `</a> <a href="/lout.php">Log out</a></div>`
	}
	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, "<!DOCTYPE html><html><body>%s%s</body></html>\n", nav, body)
}

// iptSearch lists the releases every query of q finds, the queries
// joined by commas, each written +(<terms>) and a season's terms ending
// in *.
func (s *site) iptSearch(w http.ResponseWriter, r *http.Request) {
	if !s.iptMember(r) {
		iptPage(w, false, "")
		return
	}
	var queries []string
	for q := range strings.SplitSeq(r.URL.Query().Get("q"), ",") {
		queries = append(queries, strings.Trim(q, "+ ()*"))
	}
	var rows strings.Builder
	rows.WriteString(`<table id="torrents"><thead><tr><th>Type</th><th>Sort by name</th><th>Bookmark</th><th>Download</th><th>Sort by files</th>` +
		`<th>Sort by size</th><th>Sort by comments</th><th>Sort by snatches</th><th>Sort by seeders</th><th>Sort by leechers</th></tr></thead><tbody>`)
	for _, rel := range s.found(r, queries...) {
		free := ""
		if rel.downloadFactor() == 0 {
			free = `<span class="free">FreeLeech</span>`
		}
		fmt.Fprintf(&rows, `<tr><td><a href="?72"><img src="/pic/cat.png" alt="cat"></a></td>`+
			`<td><a class="hv" href="/t/%[1]s">%[2]s</a>%[3]s<div class="sub">WEB-DL | 1 day ago by %[4]s</div></td>`+
			`<td><a href="/bookmark.php?id=%[1]s">B</a></td><td><a href="/download.php/%[1]s/%[2]s.torrent">D</a></td>`+
			`<td>1 Go to files</td><td>%[5]d B</td><td>0</td><td>%[6]d</td><td>%[7]d</td><td>%[8]d</td></tr>`,
			rel.InfoHash, html.EscapeString(rel.Name), free, Username, rel.Size, rel.Grabs, rel.Seeders, rel.Leechers)
	}
	rows.WriteString("</tbody></table>")
	iptPage(w, true, rows.String())
}

func (s *site) iptDownload(w http.ResponseWriter, r *http.Request) {
	if !s.iptMember(r) {
		iptPage(w, false, "")
		return
	}
	rel, ok := s.byHash(r, r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.torrent(w, r, rel)
}

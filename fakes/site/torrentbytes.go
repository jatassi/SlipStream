package site

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
)

// TorrentBytesRefused is what TorrentBytes' sign-in shows for a wrong
// username or password.
const TorrentBytesRefused = "Username or password incorrect"

func (s *site) torrentBytes() fake.Routes {
	return fake.Routes{
		"home":     {Pattern: "GET /", Handler: s.tbHome},
		"login":    {Pattern: "GET /login.php", Handler: s.tbLogin},
		"signin":   {Pattern: "POST /takelogin.php", Handler: s.tbSignIn},
		"browse":   {Pattern: "GET /browse.php", Handler: s.tbBrowse},
		"download": {Pattern: "GET /download.php", Handler: s.tbDownload},
	}
}

const tbLoginForm = `<form method="post" action="takelogin.php"><input type="text" name="username"><input type="password" name="password">` +
	`<input type="hidden" name="returnto" value="/"><input type="submit" name="login" value="Log in!"></form>`

// tbPage writes a TorrentBytes page. A signed-in account's links its
// profile, my.php, which the port reads as signed in.
func tbPage(w http.ResponseWriter, signedIn bool, body string) {
	nav := ""
	if signedIn {
		nav = `<div id="nav"><a href="browse.php">Browse</a> | <a href="my.php">Profile</a> | <a href="logout.php">Logout</a></div>`
	}
	w.Header().Set("Content-Type", "text/html; charset=iso-8859-1")
	_, _ = fmt.Fprintf(w, "<!DOCTYPE html><html><body><table class=\"mainouter\"><tbody><tr><td class=\"outer\">%s%s</td></tr></tbody></table></body></html>\n", nav, body)
}

func (s *site) tbHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if _, ok := s.signedIn(r, TorrentBytes); ok {
		tbPage(w, true, "<h2>Welcome back</h2>")
		return
	}
	tbPage(w, false, tbLoginForm)
}

func (s *site) tbLogin(w http.ResponseWriter, _ *http.Request) { tbPage(w, false, tbLoginForm) }

func (s *site) tbSignIn(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("username") != Username || r.FormValue("password") != Password(s.scenario, TorrentBytes) {
		tbPage(w, false, `<h2>Login failed!</h2><table><tbody><tr><td class="embedded">`+TorrentBytesRefused+`</td></tr></tbody></table>`+tbLoginForm)
		return
	}
	s.signIn(w, r, TorrentBytes, "form")
	http.Redirect(w, r, "/", http.StatusFound)
}

// tbBrowse lists the releases a title search finds, or a description
// search by IMDb id when sc is 2, with its header row of td.colhead.
func (s *site) tbBrowse(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.signedIn(r, TorrentBytes); !ok {
		http.Redirect(w, r, "/login.php?returnto=%2Fbrowse.php", http.StatusFound)
		return
	}
	var rows strings.Builder
	rows.WriteString(`<table><tbody><tr><td class="colhead">Type</td><td class="colhead">Name</td><td class="colhead">Files</td><td class="colhead">Comm.</td>` +
		`<td class="colhead">Added</td><td class="colhead">TTL</td><td class="colhead">Size</td><td class="colhead">Snatched</td>` +
		`<td class="colhead">Seeders</td><td class="colhead">Leechers</td><td class="colhead">Upped by</td></tr>`)
	for _, rel := range s.found(r, r.URL.Query().Get("search")) {
		free := ""
		if rel.downloadFactor() == 0 {
			free = ` <font color="green"><b>[FREE LEECH]</b></font>`
		}
		name := html.EscapeString(rel.Name)
		fmt.Fprintf(&rows, `<tr><td><a href="browse.php?cat=5"><img src="pic/cat5.gif" alt=""></a></td>`+
			`<td><a href="download.php?id=%[1]s&amp;SSL=1&amp;name=%[2]s.torrent"><img src="pic/download.gif" alt="Download"></a> `+
			`<a href="details.php?id=%[1]s&amp;hit=1" title="%[2]s"><b>%[2]s</b></a>%[3]s</td>`+
			`<td><a href="details.php?id=%[1]s&amp;filelist=1">1</a></td><td>0</td><td><nobr>2026-06-01<br>12:00:00</nobr></td><td>27<br>days</td>`+
			`<td>%[4]d<br>B</td><td>%[5]d<br>times</td><td><b>%[6]d</b></td><td><b>%[7]d</b></td><td><i>(unknown)</i></td></tr>`,
			rel.InfoHash, name, free, rel.Size, rel.Grabs, rel.Seeders, rel.Leechers)
	}
	rows.WriteString("</tbody></table>")
	tbPage(w, true, rows.String())
}

func (s *site) tbDownload(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.signedIn(r, TorrentBytes); !ok {
		http.Redirect(w, r, "/login.php", http.StatusFound)
		return
	}
	rel, ok := s.byHash(r, r.URL.Query().Get("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.torrent(w, r, rel)
}

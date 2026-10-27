package site

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
)

// larkspurPre is the cookie the sign-in page sets to tie its captcha to
// the sign-in that answers it.
const larkspurPre = "larkspur_pre"

// The words Larkspur's sign-in shows for a refused answer.
const (
	WrongCaptcha  = "The code doesn't match the picture."
	WrongPassword = "Wrong username or password."
)

func (s *site) larkspur() fake.Routes {
	return fake.Routes{
		"login":    {Pattern: "GET /login.php", Handler: s.larkspurLanding},
		"captcha":  {Pattern: "GET /captcha.php", Handler: s.larkspurCaptcha},
		"signin":   {Pattern: "POST /takelogin.php", Handler: s.larkspurSignIn},
		"search":   {Pattern: "GET /browse.php", Handler: s.larkspurBrowse},
		"thanks":   {Pattern: "POST /thanks.php", Handler: s.larkspurThanks},
		"download": {Pattern: "GET /download.php", Handler: s.larkspurDownload},
	}
}

func (s *site) larkspurLanding(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: larkspurPre, Value: s.mint("pre", Larkspur), Path: "/", HttpOnly: true}) //nolint:gosec // G124: a fixture site serves plain HTTP too, where a Secure cookie never returns
	page(w, `<form id="login" action="takelogin.php" method="post">
<input type="hidden" name="returnto" value="/browse.php">
<input name="username"><input type="password" name="password">
<img id="captcha" src="captcha.php"><input name="captcha">
</form>`)
}

// larkspurCode is the text the captcha of the sign-in pre shows.
func (s *site) larkspurCode(pre string) string { return derive(s.scenario.Seed, "captcha", pre)[:5] }

func (s *site) larkspurCaptcha(w http.ResponseWriter, r *http.Request) {
	pre, err := r.Cookie(larkspurPre)
	if err != nil {
		http.Error(w, "Open the sign-in page first.", http.StatusBadRequest)
		return
	}
	code := s.larkspurCode(pre.Value)
	img, err := captchaImage(code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if s.captchas != nil {
		s.captchas.Record(r.Context(), Captcha{Site: Larkspur, Text: code})
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(img)
}

func (s *site) larkspurSignIn(w http.ResponseWriter, r *http.Request) {
	pre, err := r.Cookie(larkspurPre)
	switch {
	case err != nil || r.FormValue("captcha") != s.larkspurCode(pre.Value):
		page(w, `<div class="error">`+WrongCaptcha+`</div>`)
	case r.FormValue("username") != Username || r.FormValue("password") != Password(s.scenario, Larkspur):
		page(w, `<div class="error">`+WrongPassword+`</div>`)
	default:
		s.signIn(w, r, Larkspur, "form")
		http.Redirect(w, r, "/browse.php", http.StatusFound)
	}
}

func (s *site) larkspurBrowse(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.signedIn(r, Larkspur); !ok {
		http.Redirect(w, r, "/login.php", http.StatusFound)
		return
	}
	var rows strings.Builder
	rows.WriteString(`<table id="torrents"><tr><th>Category</th><th>Name</th></tr>`)
	for _, rel := range s.found(r.URL.Query().Get("search")) {
		fmt.Fprintf(&rows, `<tr class="torrent"><td><a href="browse.php?cat=2">Films HD</a></td><td><a class="title" href="details.php?id=%[1]s">%[2]s</a></td>`+
			`<td><a class="download" href="download.php?id=%[1]s">Download</a></td><td class="size">%[3]d</td><td class="seeders">%[4]d</td>`+
			`<td class="leechers">%[5]d</td><td class="added">2026-06-01 12:00</td></tr>`, rel.InfoHash, html.EscapeString(rel.Name), rel.Size, rel.Seeders, rel.Leechers)
	}
	rows.WriteString(`<tr class="torrent"><td>A row the site garbled</td></tr></table>`)
	member(w, rows.String())
}

func (s *site) larkspurThanks(w http.ResponseWriter, r *http.Request) {
	token, ok := s.signedIn(r, Larkspur)
	if !ok {
		http.Redirect(w, r, "/login.php", http.StatusFound)
		return
	}
	s.mu.Lock()
	s.thanked[token+"/"+r.FormValue("id")] = true
	s.mu.Unlock()
	member(w, "<p>Thanks sent.</p>")
}

// larkspurDownload serves a .torrent to a session that thanked for it.
func (s *site) larkspurDownload(w http.ResponseWriter, r *http.Request) {
	token, ok := s.signedIn(r, Larkspur)
	if !ok {
		http.Redirect(w, r, "/login.php", http.StatusFound)
		return
	}
	hash := r.URL.Query().Get("id")
	rel, found := s.byHash(hash)
	s.mu.Lock()
	thanked := s.thanked[token+"/"+hash]
	s.mu.Unlock()
	switch {
	case !found:
		http.NotFound(w, r)
	case !thanked:
		http.Error(w, "Say thanks before you download.", http.StatusBadRequest)
	default:
		s.torrent(w, r, rel)
	}
}

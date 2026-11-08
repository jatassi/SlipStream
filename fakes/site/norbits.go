package site

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // G505: RFC 6238 codes are HMAC-SHA-1, as every authenticator app makes them
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// The words NorBits' sign-in shows when it refuses one. The port reads a
// refusal that names 2FA, after a sign-in with no code, as one that asks
// for a code.
const (
	NorBitsRefused   = "Feil brukernavn eller passord."
	NorBitsNeedsCode = "Kontoen har 2FA. Skriv inn koden fra autentiseringsappen."
	NorBitsWrongCode = "Feil 2FA-kode."
)

// AuthenticatorKey is the base32 key of the account's two-factor sign-in
// on NorBits in a world with Scenario s, as an authenticator app takes it.
func AuthenticatorKey(s scenario.Scenario) string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(nbSecret(s))
}

func nbSecret(s scenario.Scenario) []byte {
	secret, _ := hex.DecodeString(derive(s.Seed, "authenticator", NorBits)[:40])
	return secret
}

// TwoFactorCode is the code an authenticator app shows at t for NorBits'
// account in a world with Scenario s: RFC 6238, six digits per 30 s step.
func TwoFactorCode(s scenario.Scenario, t time.Time) string { return totp(nbSecret(s), t.Unix()/30) }

func totp(secret []byte, step int64) string {
	mac := hmac.New(sha1.New, secret)
	_ = binary.Write(mac, binary.BigEndian, step)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	return fmt.Sprintf("%06d", (binary.BigEndian.Uint32(sum[offset:offset+4])&0x7fffffff)%1_000_000)
}

// nbCodeFits reports whether code is the account's for the step of now or
// the step either side, as a site allows for a clock a little off.
func (s *site) nbCodeFits(code string) bool {
	step := time.Now().Unix() / 30
	for _, at := range []int64{step - 1, step, step + 1} {
		if code == totp(nbSecret(s.scenario), at) {
			return true
		}
	}
	return false
}

func (s *site) norBits() fake.Routes {
	return fake.Routes{
		"home":     {Pattern: "GET /", Handler: s.nbHome},
		"login":    {Pattern: "GET /login.php", Handler: s.nbLogin},
		"signin":   {Pattern: "POST /takelogin.php", Handler: s.nbSignIn},
		"browse":   {Pattern: "GET /browse.php", Handler: s.nbBrowse},
		"download": {Pattern: "GET /download.php", Handler: s.nbDownload},
	}
}

// nbPage writes a NorBits page. A member's links the sign-out,
// logout.php, which the port reads as signed in.
func nbPage(w http.ResponseWriter, member bool, body string) {
	nav := `<p><a href="/login.php">Logg inn</a></p>`
	if member {
		nav = `<div id="menu"><a href="/browse.php">Torrenter</a> <a href="/logout.php">Logg ut</a></div>`
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, "<!DOCTYPE html><html lang=\"nb\"><body>%s%s</body></html>\n", nav, body)
}

func (s *site) nbMember(r *http.Request) bool {
	_, ok := s.signedIn(r, NorBits)
	return ok
}

func (s *site) nbHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if s.nbMember(r) {
		nbPage(w, true, "<p>Velkommen tilbake, "+Username+".</p>")
		return
	}
	nbPage(w, false, "<p>Du må logge inn for å se denne siden.</p>")
}

func (s *site) nbLogin(w http.ResponseWriter, _ *http.Request) {
	nbPage(w, false, `<form method="post" action="takelogin.php"><input name="username"><input type="password" name="password"><input name="code"></form>`)
}

// nbSignIn signs in an account whose two-factor sign-in is on, so it
// takes the code of its authenticator key with the password. A sign-in
// sets uid, which the port reads as signed in.
func (s *site) nbSignIn(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	switch {
	case r.FormValue("username") != Username || r.FormValue("password") != Password(s.scenario, NorBits):
		nbPage(w, false, "<h2>Innlogging feilet</h2><p>"+NorBitsRefused+"</p>")
	case code == "":
		nbPage(w, false, "<h2>Innlogging feilet</h2><p>"+NorBitsNeedsCode+"</p>")
	case !s.nbCodeFits(code):
		nbPage(w, false, "<h2>Innlogging feilet</h2><p>"+NorBitsWrongCode+"</p>")
	default:
		http.SetCookie(w, &http.Cookie{Name: "uid", Value: derive(s.scenario.Seed, "uid", NorBits)[:6], Path: "/"}) //nolint:gosec // G124: a fixture site serves plain HTTP too, where a Secure cookie never returns
		s.signIn(w, r, NorBits, "code")
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// nbBrowse lists the releases an IMDb id or the search words find, under
// a header row the port skips.
func (s *site) nbBrowse(w http.ResponseWriter, r *http.Request) {
	if !s.nbMember(r) {
		nbPage(w, false, "<p>Du må logge inn for å se denne siden.</p>")
		return
	}
	q := r.URL.Query()
	var rows strings.Builder
	rows.WriteString(`<table id="torrentTable"><tbody><tr><td>Type</td><td>Navn</td><td>Filer</td><td>Kom.</td><td>Lagt til</td><td>TTL</td>` +
		`<td>Størrelse</td><td>Lastet ned</td><td>Seeders</td><td>Leechers</td><td>Opplaster</td></tr>`)
	for _, rel := range s.found(r, q.Get("imdbsearch"), q.Get("search")) {
		extra := ""
		if rel.Freeleech {
			extra = ` <img src="/pic/free.png" title="100% freeleech">`
		}
		if rel.IMDbID != "" {
			extra += ` <a href="https://www.imdb.com/title/` + rel.IMDbID + `/"><img src="/pic/imdb.png" alt="IMDb"></a>`
		}
		name := html.EscapeString(rel.Name)
		fmt.Fprintf(&rows, `<tr><td><a href="browse.php?incldead=1&amp;main_cat[]=1"><img src="/pic/cat_1.png" alt="kategori"></a></td>`+
			`<td><a href="details.php?id=%[1]s&amp;hit=1" title="%[2]s"><b>%[2]s</b></a>%[3]s <a href="download.php?id=%[1]s&amp;passkey=%[4]s"><img src="/pic/download.png" alt="Last ned"></a></td>`+
			`<td><a href="filelist.php?id=%[1]s">1</a></td><td>0</td><td>2026-06-01<br>12:00:00</td><td>1 uke</td>`+
			`<td>%[5]d<br>B</td><td>%[6]d<br>ganger</td><td>%[7]d</td><td>%[8]d</td><td><i>%[9]s</i></td></tr>`,
			rel.InfoHash, name, extra, Key(s.scenario, NorBits), rel.Size, rel.Grabs, rel.Seeders, rel.Leechers, Username)
	}
	rows.WriteString("</tbody></table>")
	nbPage(w, true, rows.String())
}

func (s *site) nbDownload(w http.ResponseWriter, r *http.Request) {
	if !s.nbMember(r) {
		nbPage(w, false, "<p>Du må logge inn for å se denne siden.</p>")
		return
	}
	rel, ok := s.byHash(r, r.URL.Query().Get("id"))
	if !ok || r.URL.Query().Get("passkey") != Key(s.scenario, NorBits) {
		http.NotFound(w, r)
		return
	}
	s.torrent(w, r, rel)
}

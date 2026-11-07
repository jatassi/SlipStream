package site

import (
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"

	"github.com/jatassi/slipstream/fakes/fake"
)

// MarshTooShort is the error Marsh's search lists for a query shorter
// than three letters.
const MarshTooShort = "Searches need at least 3 letters."

// marshFilms is the category Marsh lists every release under, which only
// a reader that decodes windows-1251 maps.
const marshFilms = "Фильмы"

func (s *site) marsh() fake.Routes {
	return fake.Routes{
		"index":    {Pattern: "GET /index.php", Handler: s.marshIndex},
		"signin":   {Pattern: "POST /takelogin.php", Handler: s.marshSignIn},
		"search":   {Pattern: "GET /search.php", Handler: s.marshSearch},
		"download": {Pattern: "GET /download.php", Handler: s.marshDownload},
	}
}

// windows1251 writes body in windows-1251.
func windows1251(w http.ResponseWriter, contentType, body string) {
	b, err := charmap.Windows1251.NewEncoder().String(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentType+"; charset=windows-1251")
	_, _ = w.Write([]byte(b))
}

func (s *site) marshIndex(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.signedIn(r, Marsh); !ok {
		windows1251(w, "text/html", `<html><body><form action="takelogin.php" method="post"><input name="username"><input type="password" name="password"></form></body></html>`)
		return
	}
	windows1251(w, "text/html", `<html><body><a class="logout" href="logout.php">Sign out</a></body></html>`)
}

func (s *site) marshSignIn(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("username") != Username || r.FormValue("password") != Password(s.scenario, Marsh) {
		windows1251(w, "text/html", `<html><body><div class="error">`+WrongPassword+`</div></body></html>`)
		return
	}
	s.signIn(w, r, Marsh, "post")
	http.Redirect(w, r, "/index.php", http.StatusFound)
}

// marshSearch lists rows in XML, each split over two elements, under the
// day they were posted.
func (s *site) marshSearch(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.signedIn(r, Marsh); !ok {
		http.Redirect(w, r, "/index.php", http.StatusFound)
		return
	}
	q, err := charmap.Windows1251.NewDecoder().String(r.URL.Query().Get("q"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="windows-1251"?>` + "\n<torrents>\n")
	if n := utf8.RuneCountInString(strings.TrimSpace(q)); n > 0 && n < 3 {
		b.WriteString("<error>" + MarshTooShort + "</error>\n")
	} else {
		b.WriteString("<day><date>01.06.2026</date></day>\n")
		for _, rel := range s.found(q) {
			fmt.Fprintf(&b, "<row><title>%s</title><cat>%s</cat><link>download.php?id=%s</link></row>\n", esc(rel.Name), marshFilms, rel.InfoHash)
			fmt.Fprintf(&b, "<row><size>%d</size><seeders>%d</seeders><leechers>%d</leechers></row>\n", rel.Size, rel.Seeders, rel.Leechers)
		}
	}
	b.WriteString("</torrents>\n")
	windows1251(w, "application/xml", b.String())
}

func (s *site) marshDownload(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.signedIn(r, Marsh); !ok {
		http.Redirect(w, r, "/index.php", http.StatusFound)
		return
	}
	rel, ok := s.byHash(r.URL.Query().Get("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.torrent(w, r, rel)
}

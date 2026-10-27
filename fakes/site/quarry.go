package site

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
)

func (s *site) quarry() fake.Routes {
	return fake.Routes{
		"auth": {Pattern: "GET /api/auth", Handler: s.quarryAuth},
		"me": {Pattern: "GET /api/me", Handler: func(w http.ResponseWriter, _ *http.Request) {
			quarryJSON(w, http.StatusOK, map[string]string{"user": Username})
		}},
		"search":   {Pattern: "GET /api/torrents", Handler: s.quarrySearch},
		"download": {Pattern: "GET /api/download", Handler: s.quarryDownload},
	}
}

func quarryJSON(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

// quarryKeyed reports whether r carries Quarry's API key, refusing it with
// 401 when it doesn't.
func (s *site) quarryKeyed(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Query().Get("apikey") != Key(s.scenario, Quarry) {
		quarryJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid API key"})
		return false
	}
	return true
}

func (s *site) quarryAuth(w http.ResponseWriter, r *http.Request) {
	if s.quarryKeyed(w, r) {
		quarryJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

type quarryRow struct {
	Category  string `json:"category"`
	Name      string `json:"name"`
	Link      string `json:"link"`
	Size      int64  `json:"size"`
	Seeders   int64  `json:"seeders"`
	Leechers  int64  `json:"leechers"`
	Added     string `json:"added"`
	Freeleech bool   `json:"freeleech"`
}

func (s *site) quarrySearch(w http.ResponseWriter, r *http.Request) {
	if !s.quarryKeyed(w, r) {
		return
	}
	q := r.URL.Query()
	query := q.Get("q")
	if imdb := q.Get("imdb"); imdb != "" {
		query = "tt" + imdb
	}
	rows := []quarryRow{}
	for _, rel := range s.found(query) {
		link := originOf(r) + "/api/download?" + url.Values{"id": {rel.InfoHash}, "apikey": {Key(s.scenario, Quarry)}}.Encode()
		rows = append(rows, quarryRow{
			Category: "movies", Name: rel.Name, Link: link, Size: rel.Size, Seeders: rel.Seeders, Leechers: rel.Leechers,
			Added: "2026-06-01T12:00:00Z", Freeleech: rel.Freeleech,
		})
	}
	quarryJSON(w, http.StatusOK, map[string]any{"data": rows})
}

func (s *site) quarryDownload(w http.ResponseWriter, r *http.Request) {
	if !s.quarryKeyed(w, r) {
		return
	}
	rel, ok := s.byHash(strings.ToLower(r.URL.Query().Get("id")))
	if !ok {
		http.NotFound(w, r)
		return
	}
	s.torrent(w, r, rel)
}

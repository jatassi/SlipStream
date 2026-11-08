package site

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// AvistaZRefused is the message AvistaZ's sign-in responds with for a
// wrong username, password or PID.
const AvistaZRefused = "Invalid credentials"

// PID is the account's PID on AvistaZ in a world with Scenario s.
func PID(s scenario.Scenario) string { return derive(s.Seed, "pid", AvistaZ)[:32] }

func (s *site) avistaZ() fake.Routes {
	return fake.Routes{
		"auth":     {Pattern: "POST /api/v1/jackett/auth", Handler: s.azAuth},
		"search":   {Pattern: "GET /api/v1/jackett/torrents", Handler: s.azSearch},
		"download": {Pattern: "GET /rss/download/{passkey}/{file}", Handler: s.azDownload},
	}
}

// azAuth answers a sign-in with a bearer token, which is a session of
// the Node that signed in.
func (s *site) azAuth(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("username") != Username || r.FormValue("password") != Password(s.scenario, AvistaZ) || r.FormValue("pid") != PID(s.scenario) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": AvistaZRefused})
		return
	}
	token := s.mint("token", AvistaZ)
	s.mu.Lock()
	s.sessions[token] = session{site: AvistaZ, origin: fake.OriginOf(r.Context())}
	s.mu.Unlock()
	if s.signIns != nil {
		s.signIns.Record(r.Context(), SignIn{Site: AvistaZ, Method: "token"})
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "expires_in": 86400})
}

type azRow struct {
	URL              string            `json:"url"`
	Download         string            `json:"download"`
	Type             string            `json:"type"`
	InfoHash         string            `json:"info_hash"`
	FileName         string            `json:"file_name"`
	FileSize         int64             `json:"file_size"`
	FileCount        int               `json:"file_count"`
	Seed             int64             `json:"seed"`
	Leech            int64             `json:"leech"`
	Completed        int64             `json:"completed"`
	DownloadMultiply float64           `json:"download_multiply"`
	UploadMultiply   float64           `json:"upload_multiply"`
	VideoQuality     string            `json:"video_quality"`
	CreatedAtIso     string            `json:"created_at_iso"`
	MovieTV          map[string]string `json:"movie_tv"`
}

// azSearch lists the releases an IMDb id, a TMDB id or the search words
// find. It responds 404 when it finds none, as AvistaZ does.
func (s *site) azSearch(w http.ResponseWriter, r *http.Request) {
	token, _ := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	s.mu.Lock()
	got, ok := s.sessions[token]
	s.mu.Unlock()
	if !ok || got != (session{site: AvistaZ, origin: fake.OriginOf(r.Context())}) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthenticated."})
		return
	}
	q := r.URL.Query()
	var rows []azRow
	for _, rel := range s.found(q.Get("imdb"), q.Get("search")) {
		if tmdb := q.Get("tmdb"); tmdb != "" && tmdb != strconv.FormatInt(rel.TMDBID, 10) {
			continue
		}
		download := 1.0
		if rel.Freeleech {
			download = 0
		}
		rows = append(rows, azRow{
			URL:      originOf(r) + "/torrent/" + rel.InfoHash,
			Download: originOf(r) + "/rss/download/" + Key(s.scenario, AvistaZ) + "/" + rel.InfoHash + ".torrent",
			Type:     "Movie", InfoHash: rel.InfoHash, FileName: rel.Name, FileSize: rel.Size, FileCount: 1,
			Seed: rel.Seeders, Leech: rel.Leechers, Completed: rel.Grabs, DownloadMultiply: download, UploadMultiply: 1,
			VideoQuality: videoQuality(rel.Name), CreatedAtIso: "2026-06-01T12:00:00+00:00",
			MovieTV: map[string]string{"imdb": rel.IMDbID, "tmdb": strconv.FormatInt(rel.TMDBID, 10)},
		})
	}
	if rows == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"message": "No torrents found."})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"current_page": 1, "data": rows})
}

// videoQuality is the resolution a release's name gives, "SD" for none.
func videoQuality(name string) string {
	for _, q := range []string{"2160p", "1080p", "720p"} {
		if strings.Contains(name, q) {
			return q
		}
	}
	return "SD"
}

// azDownload serves a .torrent to a link that carries the account's
// passkey, as an RSS link does.
func (s *site) azDownload(w http.ResponseWriter, r *http.Request) {
	hash, isTorrent := strings.CutSuffix(r.PathValue("file"), ".torrent")
	rel, found := s.byHash(hash)
	if r.PathValue("passkey") != Key(s.scenario, AvistaZ) || !isTorrent || !found {
		http.NotFound(w, r)
		return
	}
	s.torrent(w, r, rel)
}

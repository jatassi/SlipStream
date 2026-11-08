package site

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
)

func (s *site) passThePopcorn() fake.Routes {
	return fake.Routes{"torrents": {Pattern: "GET /torrents.php", Handler: s.ptpTorrents}}
}

type ptpMovie struct {
	GroupID    string       `json:"GroupId"`
	CategoryID string       `json:"CategoryId"`
	Title      string       `json:"Title"`
	ImdbID     string       `json:"ImdbId"`
	Torrents   []ptpTorrent `json:"Torrents"`
}

type ptpTorrent struct {
	ID            int    `json:"Id"`
	Size          string `json:"Size"`
	UploadTime    string `json:"UploadTime"`
	Snatched      string `json:"Snatched"`
	Seeders       string `json:"Seeders"`
	Leechers      string `json:"Leechers"`
	ReleaseName   string `json:"ReleaseName"`
	Checked       bool   `json:"Checked"`
	FreeleechType string `json:"FreeleechType,omitempty"`
}

// ptpTorrents is PassThePopcorn's one page: an advanced search or a
// download, each only with the account's API user and key as headers. A
// torrent's id is its release's place in the catalog, from 1.
func (s *site) ptpTorrents(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("ApiUser") != Username || r.Header.Get("ApiKey") != Key(s.scenario, PassThePopcorn) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"Message": "The API user or key is wrong."})
		return
	}
	q := r.URL.Query()
	switch q.Get("action") {
	case "advanced":
		s.ptpSearch(w, r, q.Get("searchstr"))
	case "download":
		id, err := strconv.Atoi(q.Get("id"))
		if err != nil || id < 1 || id > len(s.releases) {
			http.NotFound(w, r)
			return
		}
		s.torrent(w, r, s.releases[id-1])
	default:
		http.NotFound(w, r)
	}
}

func (s *site) ptpSearch(w http.ResponseWriter, r *http.Request, searchstr string) {
	movies := []ptpMovie{}
	for _, rel := range s.found(r, searchstr) {
		id := s.place(rel)
		t := ptpTorrent{
			ID: id, Size: strconv.FormatInt(rel.Size, 10), UploadTime: "2026-06-01 12:00:00", Snatched: strconv.FormatInt(rel.Grabs, 10),
			Seeders: strconv.FormatInt(rel.Seeders, 10), Leechers: strconv.FormatInt(rel.Leechers, 10), ReleaseName: rel.Name, Checked: true,
		}
		if rel.downloadFactor() == 0 {
			t.FreeleechType = "Freeleech"
		}
		movies = append(movies, ptpMovie{
			GroupID: strconv.Itoa(id), CategoryID: "1", Title: rel.Name, ImdbID: strings.TrimPrefix(rel.IMDbID, "tt"), Torrents: []ptpTorrent{t},
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"TotalResults": strconv.Itoa(len(movies)), "Movies": movies, "Page": "1"})
}

// place is rel's place in the catalog, from 1.
func (s *site) place(rel Release) int {
	for i, r := range s.releases {
		if r.InfoHash == rel.InfoHash {
			return i + 1
		}
	}
	return 0
}

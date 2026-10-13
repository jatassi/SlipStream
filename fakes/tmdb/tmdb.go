// Package tmdb is fake-tmdb: TMDB v3's movie search and detail, from the
// Scenario's tmdb catalog. It takes any API key or none.
package tmdb

import (
	"cmp"
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var Fake = fake.Spec{
	Name: "tmdb",
	TLS:  true,
	New:  newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.TMDB.API = endpoint.MustParseURL(at.URL(root))
	},
}

const root = "/3"

type server struct {
	movies []scenario.TMDBMovie
}

func newServer(s scenario.Scenario, _ fake.Env) (fake.Server, error) {
	srv := &server{}
	if s.TMDB != nil {
		srv.movies = slices.SortedFunc(slices.Values(s.TMDB.Movies), func(a, b scenario.TMDBMovie) int { return cmp.Compare(a.ID, b.ID) })
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+root+"/search/movie", srv.search)
	mux.HandleFunc("GET "+root+"/movie/{id}", srv.movie)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { notFound(w) })
	return fake.HTTP(mux), nil
}

type resultJSON struct {
	Adult         bool   `json:"adult"`
	ID            int64  `json:"id"`
	OriginalTitle string `json:"original_title"`
	ReleaseDate   string `json:"release_date"`
	Title         string `json:"title"`
}

type pageJSON struct {
	Page         int          `json:"page"`
	Results      []resultJSON `json:"results"`
	TotalPages   int          `json:"total_pages"`
	TotalResults int          `json:"total_results"`
}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query, year := strings.ToLower(q.Get("query")), q.Get("year")
	page := pageJSON{Page: 1, Results: []resultJSON{}}
	for _, m := range s.movies {
		if query == "" || !strings.Contains(strings.ToLower(m.Title), query) || (year != "" && !strings.HasPrefix(m.Released, year+"-")) {
			continue
		}
		page.Results = append(page.Results, resultJSON{ID: m.ID, OriginalTitle: m.Title, ReleaseDate: m.Released, Title: m.Title})
	}
	page.TotalResults = len(page.Results)
	if page.TotalResults > 0 {
		page.TotalPages = 1
	}
	reply(w, http.StatusOK, page)
}

type movieJSON struct {
	Adult         bool              `json:"adult"`
	ID            int64             `json:"id"`
	IMDbID        string            `json:"imdb_id"`
	OriginalTitle string            `json:"original_title"`
	ReleaseDate   string            `json:"release_date"`
	Runtime       int64             `json:"runtime"`
	Status        string            `json:"status"`
	Title         string            `json:"title"`
	ExternalIDs   *externalIDsJSON  `json:"external_ids,omitempty"`
	ReleaseDates  *releaseDatesJSON `json:"release_dates,omitempty"`
}

type externalIDsJSON struct {
	IMDbID string `json:"imdb_id"`
}

type releaseDatesJSON struct {
	Results []countryJSON `json:"results"`
}

type countryJSON struct {
	Country      string        `json:"iso_3166_1"`
	ReleaseDates []releaseJSON `json:"release_dates"`
}

type releaseJSON struct {
	Certification string `json:"certification"`
	Language      string `json:"iso_639_1"`
	Note          string `json:"note"`
	ReleaseDate   string `json:"release_date"`
	Type          int    `json:"type"`
}

const theatrical = 3

func (s *server) movie(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	i := slices.IndexFunc(s.movies, func(m scenario.TMDBMovie) bool { return m.ID == id })
	if err != nil || i < 0 {
		notFound(w)
		return
	}
	m := s.movies[i]
	out := movieJSON{
		ID: m.ID, IMDbID: m.IMDbID, OriginalTitle: m.Title, ReleaseDate: m.Released,
		Runtime: int64(m.Runtime.Minutes()), Status: "Released", Title: m.Title,
	}
	for _, part := range strings.Split(r.URL.Query().Get("append_to_response"), ",") {
		switch part {
		case "external_ids":
			out.ExternalIDs = &externalIDsJSON{IMDbID: m.IMDbID}
		case "release_dates":
			out.ReleaseDates = &releaseDatesJSON{Results: []countryJSON{{Country: "US", ReleaseDates: []releaseJSON{
				{ReleaseDate: m.Released + "T00:00:00.000Z", Type: theatrical},
			}}}}
		}
	}
	reply(w, http.StatusOK, out)
}

func notFound(w http.ResponseWriter) {
	reply(w, http.StatusNotFound, struct {
		Success       bool   `json:"success"`
		StatusCode    int    `json:"status_code"`
		StatusMessage string `json:"status_message"`
	}{false, 34, "The resource you requested could not be found."})
}

func reply(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

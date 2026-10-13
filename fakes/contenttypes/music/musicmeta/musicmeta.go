package musicmeta

import (
	"cmp"
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

var Fake = fake.Spec{
	Name:        "music-meta",
	TLS:         true,
	ContentType: "music",
	New:         newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.ContentTypes.Set("music", "metadata", at.URL(root))
	},
}

const root = "/ws/2"

type server struct {
	artists  map[string]scenario.MusicArtist
	artistOf map[string]string

	mu      sync.Mutex
	lookups map[string]int
	faults  map[string]*fault
}

type fault struct {
	answer string
	left   int
}

func newServer(s scenario.Scenario, _ fake.Env) (fake.Server, error) {
	srv := &server{artists: map[string]scenario.MusicArtist{}, artistOf: map[string]string{}, lookups: map[string]int{}, faults: map[string]*fault{}}
	if s.Music != nil {
		for _, a := range s.Music.Artists {
			srv.artists[a.ID] = a
			for _, rev := range a.Revisions {
				for _, al := range rev.Albums {
					srv.artistOf[al.ID] = a.ID
				}
			}
		}
		for _, f := range s.Music.Faults {
			srv.faults[f.Artist] = &fault{answer: f.Answer, left: f.Times}
		}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+root+"/artist", jsonOnly(srv.search))
	mux.HandleFunc("GET "+root+"/artist/{mbid}", jsonOnly(srv.artist))
	mux.HandleFunc("GET "+root+"/release-group", jsonOnly(srv.releaseGroups))
	mux.HandleFunc("GET "+root+"/release", jsonOnly(srv.releases))
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { notFound(w) })
	return fake.HTTP(mux), nil
}

type artistJSON struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SortName       string `json:"sort-name"`
	Country        string `json:"country,omitempty"`
	Disambiguation string `json:"disambiguation"`
	Score          int    `json:"score,omitempty"`
}

type releaseGroupJSON struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	PrimaryType      string   `json:"primary-type"`
	SecondaryTypes   []string `json:"secondary-types"`
	FirstReleaseDate string   `json:"first-release-date"`
}

type releaseJSON struct {
	ID             string       `json:"id"`
	Title          string       `json:"title"`
	Status         string       `json:"status"`
	Date           string       `json:"date"`
	Disambiguation string       `json:"disambiguation"`
	Media          []mediumJSON `json:"media"`
}

type mediumJSON struct {
	Position    int         `json:"position"`
	TrackCount  int         `json:"track-count"`
	TrackOffset int         `json:"track-offset"`
	Tracks      []trackJSON `json:"tracks"`
}

type trackJSON struct {
	Position  int           `json:"position"`
	Number    string        `json:"number"`
	Title     string        `json:"title"`
	Length    *int64        `json:"length"`
	Recording recordingJSON `json:"recording"`
}

type recordingJSON struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Length *int64 `json:"length"`
}

func (s *server) revision(a scenario.MusicArtist) scenario.MusicRevision {
	s.mu.Lock()
	defer s.mu.Unlock()
	return a.Revisions[min(max(s.lookups[a.ID]-1, 0), len(a.Revisions)-1)]
}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("query"))
	var found []artistJSON
	for _, a := range s.artists {
		rev := s.revision(a)
		name, sortName := strings.ToLower(rev.Name), strings.ToLower(rev.SortName)
		var score int
		switch {
		case name == q:
			score = 100
		case strings.HasPrefix(name, q) || strings.HasPrefix(sortName, q):
			score = 90
		case strings.Contains(name, q) || strings.Contains(sortName, q):
			score = 80
		default:
			continue
		}
		found = append(found, artistJSON{ID: a.ID, Name: rev.Name, SortName: rev.SortName, Country: rev.Country, Disambiguation: rev.Disambiguation, Score: score})
	}
	slices.SortFunc(found, func(a, b artistJSON) int {
		return cmp.Or(cmp.Compare(b.Score, a.Score), cmp.Compare(a.Name, b.Name), cmp.Compare(a.ID, b.ID))
	})
	items, offset, ok := page(w, r, found)
	if !ok {
		return
	}
	reply(w, struct {
		Count   int          `json:"count"`
		Offset  int          `json:"offset"`
		Artists []artistJSON `json:"artists"`
	}{len(found), offset, items})
}

func (s *server) artist(w http.ResponseWriter, r *http.Request) {
	a, ok := s.artists[r.PathValue("mbid")]
	if !ok {
		notFound(w)
		return
	}
	s.mu.Lock()
	f := s.faults[a.ID]
	faulted := f != nil && f.left > 0
	if faulted {
		f.left--
	} else {
		s.lookups[a.ID]++
	}
	s.mu.Unlock()
	if faulted {
		answers[f.answer](w, r)
		return
	}
	rev := s.revision(a)
	reply(w, artistJSON{ID: a.ID, Name: rev.Name, SortName: rev.SortName, Country: rev.Country, Disambiguation: rev.Disambiguation})
}

var answers = map[string]http.HandlerFunc{
	scenario.AnswerNotFound: func(w http.ResponseWriter, _ *http.Request) { notFound(w) },
	scenario.AnswerUnavailable: func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "1")
		errorReply(w, http.StatusServiceUnavailable)
	},
	scenario.AnswerSlow: func(_ http.ResponseWriter, r *http.Request) { <-r.Context().Done() },
}

func (s *server) releaseGroups(w http.ResponseWriter, r *http.Request) {
	a, ok := s.artists[r.URL.Query().Get("artist")]
	if !ok {
		notFound(w)
		return
	}
	types := strings.Split(strings.ToLower(r.URL.Query().Get("type")), "|")
	var groups []releaseGroupJSON
	for _, al := range s.revision(a).Albums {
		if slices.Contains(types, strings.ToLower(al.Kind)) {
			groups = append(groups, releaseGroupJSON{ID: al.ID, Title: al.Title, PrimaryType: al.Kind, SecondaryTypes: []string{}, FirstReleaseDate: al.FirstRelease})
		}
	}
	items, offset, ok := page(w, r, groups)
	if !ok {
		return
	}
	reply(w, struct {
		Count         int                `json:"release-group-count"`
		Offset        int                `json:"release-group-offset"`
		ReleaseGroups []releaseGroupJSON `json:"release-groups"`
	}{len(groups), offset, items})
}

func (s *server) releases(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("release-group")
	a, ok := s.artists[s.artistOf[id]]
	if !ok {
		notFound(w)
		return
	}
	rev := s.revision(a)
	i := slices.IndexFunc(rev.Albums, func(al scenario.MusicAlbum) bool { return al.ID == id })
	if i < 0 {
		notFound(w)
		return
	}
	all := []releaseJSON{release(rev.Albums[i])}
	items, offset, ok := page(w, r, all)
	if !ok {
		return
	}
	reply(w, struct {
		Count    int           `json:"release-count"`
		Offset   int           `json:"release-offset"`
		Releases []releaseJSON `json:"releases"`
	}{len(all), offset, items})
}

func release(al scenario.MusicAlbum) releaseJSON {
	out := releaseJSON{ID: al.Release, Title: al.Title, Status: "Official", Date: al.FirstRelease, Media: []mediumJSON{}}
	discs := map[int]*mediumJSON{}
	for _, t := range al.Tracks {
		m := discs[t.Disc]
		if m == nil {
			m = &mediumJSON{Position: t.Disc}
			discs[t.Disc] = m
		}
		var length *int64
		if t.Length > 0 {
			ms := t.Length.Milliseconds()
			length = &ms
		}
		m.TrackCount++
		m.Tracks = append(m.Tracks, trackJSON{
			Position: m.TrackCount, Number: strconv.Itoa(m.TrackCount), Title: t.Title, Length: length,
			Recording: recordingJSON{ID: t.Recording, Title: t.Title, Length: length},
		})
	}
	for _, d := range slices.Sorted(maps.Keys(discs)) {
		out.Media = append(out.Media, *discs[d])
	}
	return out
}

func page[T any](w http.ResponseWriter, r *http.Request, all []T) (items []T, offset int, ok bool) {
	limit := 25
	for name, v := range map[string]*int{"limit": &limit, "offset": &offset} {
		raw := r.URL.Query().Get(name)
		if raw == "" {
			continue
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			errorReply(w, http.StatusBadRequest)
			return nil, 0, false
		}
		*v = n
	}
	limit = min(max(limit, 1), 100)
	start := min(offset, len(all))
	end := min(start+limit, len(all))
	return append([]T{}, all[start:end]...), offset, true
}

func jsonOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fmt") != "json" {
			errorReply(w, http.StatusBadRequest)
			return
		}
		h(w, r)
	}
}

func notFound(w http.ResponseWriter) { errorReply(w, http.StatusNotFound) }

func errorReply(w http.ResponseWriter, status int) {
	write(w, status, struct {
		Error string `json:"error"`
	}{http.StatusText(status)})
}

func reply(w http.ResponseWriter, v any) { write(w, http.StatusOK, v) }

func write(w http.ResponseWriter, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

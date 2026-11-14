package plex

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

const (
	notFoundBody     = "<html><head><title>Not Found</title></head><body><h1>404 Not Found</h1></body></html>"
	unauthorizedBody = "<html><head><title>Unauthorized</title></head><body><h1>401 Unauthorized</h1></body></html>"
)

var collectionTag = regexp.MustCompile(`^collection\[\d+\]\.tag\.tag$`)

func (s *server) routes(refreshes *fake.Capture) fake.Routes {
	return fake.Routes{
		"identity":      {Pattern: "GET /identity", Handler: s.identity},
		"root":          {Pattern: "GET /{$}", Handler: s.authed(s.root)},
		"sections":      {Pattern: "GET /library/sections", Handler: s.authed(s.sections)},
		"refresh":       {Pattern: "/library/sections/{section}/refresh", Handler: s.authed(s.refresh(refreshes))},
		"section-all":   {Pattern: "GET /library/sections/{section}/all", Handler: s.authed(s.sectionAll)},
		"metadata":      {Pattern: "GET /library/metadata/{key}", Handler: s.authed(s.metadata)},
		"edit":          {Pattern: "PUT /library/metadata/{key}", Handler: s.authed(s.edit)},
		"item-refresh":  {Pattern: "PUT /library/metadata/{key}/refresh", Handler: s.authed(s.itemRefresh(refreshes))},
		"scrobble":      {Pattern: "GET /:/scrobble", Handler: s.authed(s.scrobble)},
		"sessions":      {Pattern: "GET /status/sessions", Handler: s.authed(s.sessions)},
		"transcodes":    {Pattern: "GET /transcode/sessions", Handler: s.authed(s.transcodes)},
		"notifications": {Pattern: "GET /:/websockets/notifications", Handler: s.authed(s.notifications)},
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}

func writeHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

type container struct {
	MediaContainer any
}

// authed refuses a request without a server token, from the
// X-Plex-Token header or query parameter. A world with no Plex answers
// 404 everywhere.
func (s *server) authed(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.srv == nil {
			writeHTML(w, http.StatusNotFound, notFoundBody)
			return
		}
		token := r.Header.Get("X-Plex-Token")
		if token == "" {
			token = r.URL.Query().Get("X-Plex-Token")
		}
		if token == "" || (!slices.Contains(s.srv.Tokens, token) && !s.account.Accepts(s.srv.Identifier, token)) {
			writeHTML(w, http.StatusUnauthorized, unauthorizedBody)
			return
		}
		h(w, r)
	}
}

func (s *server) identity(w http.ResponseWriter, _ *http.Request) {
	if s.srv == nil {
		writeHTML(w, http.StatusNotFound, notFoundBody)
		return
	}
	writeJSON(w, container{map[string]any{
		"size": 0, "claimed": true, "machineIdentifier": s.srv.Identifier, "version": s.srv.Version,
	}})
}

func (s *server) root(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, container{map[string]any{
		"size": 0, "friendlyName": s.srv.Name, "machineIdentifier": s.srv.Identifier, "version": s.srv.Version,
		"myPlexSubscription": s.srv.PlexPass, "myPlexUsername": s.srv.Owner, "claimed": true,
	}})
}

type location struct {
	ID   int    `json:"id"`
	Path string `json:"path"`
}

type directory struct {
	Key        string     `json:"key"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	Agent      string     `json:"agent"`
	Scanner    string     `json:"scanner"`
	UUID       string     `json:"uuid"`
	Refreshing bool       `json:"refreshing"`
	ScannedAt  int64      `json:"scannedAt"`
	Location   []location `json:"Location"`
}

func (s *server) sections(w http.ResponseWriter, _ *http.Request) {
	var dirs []directory
	id := 0
	for _, l := range s.core.Libraries() {
		d := directory{
			Key: l.Key, Type: l.Type, Title: l.Name, UUID: "fake-plex-section-" + l.Key,
			Agent: "tv.plex.agents." + l.Type, Scanner: "Plex " + l.Type + " scanner",
			Refreshing: l.Scan.Running, Location: []location{},
		}
		if !l.Scan.Ended.IsZero() {
			d.ScannedAt = l.Scan.Ended.Unix()
		}
		for _, f := range l.Folders {
			id++
			d.Location = append(d.Location, location{ID: id, Path: f.String()})
		}
		dirs = append(dirs, d)
	}
	writeJSON(w, container{map[string]any{"size": len(dirs), "Directory": dirs}})
}

// refresh is a partial refresh with path=, else a section refresh. Plex
// documents POST and clients send GET. Both answer 200 with no body.
func (s *server) refresh(refreshes *fake.Capture) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		section := r.PathValue("section")
		raw := r.URL.Query().Get("path")
		var err error
		if raw == "" {
			refreshes.Record(r.Context(), Refresh{Section: section})
			_, err = s.core.RefreshLibrary(section)
		} else {
			folder, perr := link.ParsePath(raw)
			if perr != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			refreshes.Record(r.Context(), Refresh{Section: section, Path: folder.String()})
			err = s.core.RefreshFolder(section, folder)
		}
		if errors.Is(err, mediaserver.ErrNoLibrary) {
			writeHTML(w, http.StatusNotFound, notFoundBody)
		}
	}
}

func (s *server) itemRefresh(refreshes *fake.Capture) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.PathValue("key")
		refreshes.Record(r.Context(), Refresh{Item: key})
		if !s.core.RefreshItem(key) {
			writeHTML(w, http.StatusNotFound, notFoundBody)
		}
	}
}

func (s *server) sectionAll(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	start, _ := strconv.Atoi(first(q.Get("X-Plex-Container-Start"), r.Header.Get("X-Plex-Container-Start")))
	size, err := strconv.Atoi(first(q.Get("X-Plex-Container-Size"), r.Header.Get("X-Plex-Container-Size")))
	if err != nil {
		size = 1 << 30
	}
	section := r.PathValue("section")
	if !slices.ContainsFunc(s.core.Libraries(), func(l mediaserver.LibraryState) bool { return l.Key == section }) {
		writeHTML(w, http.StatusNotFound, notFoundBody)
		return
	}
	all, _ := s.core.Items(section, 0, 1<<30)
	typ := q.Get("type")
	var picked []mediaserver.Item
	for _, it := range all {
		if typ == "" || typ == strconv.Itoa(plexType(it)) {
			picked = append(picked, it)
		}
	}
	start = min(start, len(picked))
	end := min(start+size, len(picked))
	page := make([]metadata, 0, end-start)
	for _, it := range picked[start:end] {
		page = append(page, s.render(it))
	}
	writeJSON(w, container{map[string]any{"size": len(page), "totalSize": len(picked), "offset": start, "Metadata": page}})
}

func first(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (s *server) metadata(w http.ResponseWriter, r *http.Request) {
	it, ok := s.core.Item(r.PathValue("key"))
	if !ok {
		writeHTML(w, http.StatusNotFound, notFoundBody)
		return
	}
	writeJSON(w, container{map[string]any{"size": 1, "Metadata": []metadata{s.render(it)}}})
}

// edit takes collection[N].tag.tag, the one item edit the suite seeds
// with.
func (s *server) edit(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if _, ok := s.core.Item(key); !ok {
		writeHTML(w, http.StatusNotFound, notFoundBody)
		return
	}
	for k, vs := range r.URL.Query() {
		if collectionTag.MatchString(k) {
			for _, v := range vs {
				s.core.AddToCollection(v, key)
			}
		}
	}
}

func (s *server) scrobble(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if _, ok := s.core.Item(key); !ok {
		writeHTML(w, http.StatusNotFound, notFoundBody)
		return
	}
	s.core.Played(s.srv.Owner, key, time.Now())
}

type tag struct {
	Tag string `json:"tag"`
}

type stream struct {
	ID         int    `json:"id"`
	StreamType int    `json:"streamType"`
	Codec      string `json:"codec"`
	Key        string `json:"key"`
}

type part struct {
	ID       int      `json:"id"`
	Key      string   `json:"key"`
	File     string   `json:"file"`
	Size     int64    `json:"size"`
	Decision string   `json:"decision,omitempty"`
	Stream   []stream `json:"Stream,omitempty"`
}

type media struct {
	ID   int    `json:"id"`
	Part []part `json:"Part"`
}

type metadata struct {
	RatingKey            string         `json:"ratingKey"`
	Key                  string         `json:"key"`
	GUID                 string         `json:"guid"`
	Type                 string         `json:"type"`
	Title                string         `json:"title"`
	Year                 int            `json:"year,omitempty"`
	LibrarySectionID     int            `json:"librarySectionID"`
	GrandparentRatingKey string         `json:"grandparentRatingKey,omitempty"`
	GrandparentTitle     string         `json:"grandparentTitle,omitempty"`
	ParentIndex          int            `json:"parentIndex,omitempty"`
	Index                int            `json:"index,omitempty"`
	EditionTitle         string         `json:"editionTitle,omitempty"`
	ViewCount            int            `json:"viewCount,omitempty"`
	ViewOffset           int64          `json:"viewOffset,omitempty"`
	LastViewedAt         int64          `json:"lastViewedAt,omitempty"`
	UserRating           float64        `json:"userRating,omitempty"`
	AddedAt              int64          `json:"addedAt"`
	Collection           []tag          `json:"Collection,omitempty"`
	Media                []media        `json:"Media"`
	SessionKey           string         `json:"sessionKey,omitempty"`
	User                 map[string]any `json:"User,omitempty"`
	Player               map[string]any `json:"Player,omitempty"`
	Session              map[string]any `json:"Session,omitempty"`
	TranscodeSession     *transcode     `json:"TranscodeSession,omitempty"`
}

func plexType(it mediaserver.Item) int {
	if it.Show != "" {
		return 4
	}
	return 1
}

func (s *server) render(it mediaserver.Item) metadata {
	section, _ := strconv.Atoi(it.Library)
	m := metadata{
		RatingKey: it.Key, Key: "/library/metadata/" + it.Key, GUID: "plex://" + it.GUID, Type: "movie",
		Title: it.Title, Year: it.Year, LibrarySectionID: section, EditionTitle: it.Edition, AddedAt: it.Added.Unix(),
		Media: []media{},
	}
	if it.Show != "" {
		m.Type, m.GrandparentRatingKey, m.GrandparentTitle, m.ParentIndex, m.Index = "episode", it.ShowKey, it.Show, it.Season, it.Episode
	}
	d := s.core.UserData(s.srv.Owner, it.Key)
	m.ViewCount, m.ViewOffset, m.UserRating = d.PlayCount, d.Position.Milliseconds(), d.Rating
	if !d.LastPlayed.IsZero() {
		m.LastViewedAt = d.LastPlayed.Unix()
	}
	for _, c := range s.core.Collections(it.Key) {
		m.Collection = append(m.Collection, tag{Tag: c})
	}
	for _, p := range it.Parts {
		if p.Missing {
			continue
		}
		pt := part{ID: p.ID, Key: "/library/parts/" + strconv.Itoa(p.ID) + "/file", File: p.File.String(), Size: p.Size}
		for i := range p.Sidecars {
			id := p.ID*100 + i + 1
			sc := p.Sidecars[i].String()
			pt.Stream = append(pt.Stream, stream{ID: id, StreamType: 3, Codec: sc[strings.LastIndexByte(sc, '.')+1:], Key: "/library/streams/" + strconv.Itoa(id)})
		}
		m.Media = append(m.Media, media{ID: p.ID, Part: []part{pt}})
	}
	return m
}

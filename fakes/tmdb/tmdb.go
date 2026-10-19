// Package tmdb is fake-tmdb: TMDB v3's movie search, whole movie records,
// changes list and configuration, from the Scenario's tmdb catalog. It
// takes any API key or none until a cue revokes it.
package tmdb

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
)

var (
	// RevokeKey refuses {"key": "…"} from now on with TMDB's 401 body, or,
	// with no key named, every key the fake has seen, none included.
	RevokeKey = fake.CueSpec{Name: "tmdb-revoke-key"}
	// Change makes the record of {"id": n} differ on its next fetch and
	// lists the id as changed now.
	Change = fake.CueSpec{Name: "tmdb-change"}
	// Delete has TMDB drop {"id": n}: it answers 404 with status_code 34
	// and is listed as changed now.
	Delete = fake.CueSpec{Name: "tmdb-delete"}
)

var Fake = fake.Spec{
	Name: "tmdb",
	TLS:  fake.FrontTLS,
	New:  newServer,
	Endpoints: func(e *node.EndpointsBlock, at fake.Address) {
		e.TMDB.API = endpoint.MustParseURL(at.URL(root))
	},
	Online: fake.OnlineSuite("tmdb"),
	Cues:   []fake.CueSpec{RevokeKey, Change, Delete},
}

const (
	root      = "/3"
	imageRoot = "/t/p"
	// pageSize and changesPageSize are TMDB's page lengths.
	pageSize        = 20
	changesPageSize = 100
)

type server struct {
	movies []scenario.TMDBMovie

	mu       sync.Mutex
	seen     map[string]bool
	revoked  map[string]bool
	revision map[int64]int
	deleted  map[int64]bool
	changes  []change
}

type change struct {
	id int64
	at time.Time
}

func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	srv := &server{seen: map[string]bool{}, revoked: map[string]bool{}, revision: map[int64]int{}, deleted: map[int64]bool{}}
	if s.TMDB != nil {
		srv.movies = slices.SortedFunc(slices.Values(s.TMDB.Movies), func(a, b scenario.TMDBMovie) int { return cmp.Compare(a.ID, b.ID) })
	}
	for _, m := range srv.movies {
		srv.deleted[m.ID] = m.Deleted
	}
	if err := fake.OnCue(env, RevokeKey.Name, srv.revoke); err != nil {
		return nil, err
	}
	if err := fake.OnCue(env, Change.Name, func(_ context.Context, a Movie) (any, error) { return srv.change(a.ID, false) }); err != nil {
		return nil, err
	}
	if err := fake.OnCue(env, Delete.Name, func(_ context.Context, a Movie) (any, error) { return srv.change(a.ID, true) }); err != nil {
		return nil, err
	}
	return fake.HTTP(env, fake.Routes{
		"search":        {Pattern: "GET " + root + "/search/movie", Handler: srv.keyed(srv.search)},
		"movie":         {Pattern: "GET " + root + "/movie/{id}", Handler: srv.keyed(srv.movie)},
		"changes":       {Pattern: "GET " + root + "/movie/changes", Handler: srv.keyed(srv.changed)},
		"configuration": {Pattern: "GET " + root + "/configuration", Handler: srv.keyed(srv.configuration)},
		"not-found":     {Pattern: "/", Handler: func(w http.ResponseWriter, _ *http.Request) { notFound(w) }},
	}), nil
}

// Key names the key a RevokeKey cue refuses; nil is every key seen.
type Key struct {
	Key *string `json:"key"`
}

// Movie names the movie a Change or Delete cue plays on.
type Movie struct {
	ID int64 `json:"id"`
}

type revoked struct {
	Revoked int `json:"revoked"`
}

func (s *server) revoke(_ context.Context, a Key) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := slices.Collect(maps.Keys(s.seen))
	if a.Key != nil {
		keys = []string{*a.Key}
	}
	for _, k := range keys {
		s.revoked[k] = true
	}
	return revoked{len(keys)}, nil
}

type changed struct {
	Revision int  `json:"revision"`
	Deleted  bool `json:"deleted"`
}

func (s *server) change(id int64, del bool) (any, error) {
	if !slices.ContainsFunc(s.movies, func(m scenario.TMDBMovie) bool { return m.ID == id }) {
		return nil, fmt.Errorf("tmdb lists no movie %d", id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if del {
		s.deleted[id] = true
	} else {
		s.revision[id]++
	}
	s.changes = append(s.changes, change{id: id, at: time.Now()})
	return changed{Revision: s.revision[id], Deleted: s.deleted[id]}, nil
}

// keyed refuses a revoked key, as TMDB does a bad one, before h runs.
func (s *server) keyed(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("api_key")
		s.mu.Lock()
		s.seen[key] = true
		refused := s.revoked[key]
		s.mu.Unlock()
		if refused {
			reply(w, http.StatusUnauthorized, struct {
				StatusCode    int    `json:"status_code"`
				StatusMessage string `json:"status_message"`
				Success       bool   `json:"success"`
			}{7, "Invalid API key: You must be granted a valid key.", false})
			return
		}
		h(w, r)
	}
}

// live is the movie TMDB holds as id now, with its revision.
func (s *server) live(id int64) (scenario.TMDBMovie, int, bool) {
	i := slices.IndexFunc(s.movies, func(m scenario.TMDBMovie) bool { return m.ID == id })
	s.mu.Lock()
	defer s.mu.Unlock()
	if i < 0 || s.deleted[id] {
		return scenario.TMDBMovie{}, 0, false
	}
	return s.movies[i], s.revision[id], true
}

type resultJSON struct {
	Adult            bool    `json:"adult"`
	BackdropPath     *string `json:"backdrop_path"`
	ID               int64   `json:"id"`
	OriginalLanguage string  `json:"original_language"`
	OriginalTitle    string  `json:"original_title"`
	Overview         string  `json:"overview"`
	PosterPath       *string `json:"poster_path"`
	ReleaseDate      string  `json:"release_date"`
	Title            string  `json:"title"`
}

type pageJSON struct {
	Page         int          `json:"page"`
	Results      []resultJSON `json:"results"`
	TotalPages   int          `json:"total_pages"`
	TotalResults int          `json:"total_results"`
}

func (s *server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	query, year, lang := strings.ToLower(q.Get("query")), q.Get("primary_release_year"), q.Get("language")
	var found []resultJSON
	for _, m := range s.movies {
		m, rev, ok := s.live(m.ID)
		if !ok || query == "" || !matches(m, query) || (year != "" && !strings.HasPrefix(primary(m), year+"-")) {
			continue
		}
		t := text(m, rev, lang)
		found = append(found, resultJSON{
			ID: m.ID, OriginalLanguage: originalLanguage(m), OriginalTitle: originalTitle(m), Overview: t.Overview,
			PosterPath: pathOf(m, "poster", lang), BackdropPath: pathOf(m, "backdrop", ""), ReleaseDate: primary(m), Title: t.Title,
		})
	}
	page := pageJSON{Page: pageParam(q.Get("page")), Results: []resultJSON{}, TotalResults: len(found)}
	page.TotalPages = (len(found) + pageSize - 1) / pageSize
	if from := (page.Page - 1) * pageSize; from < len(found) {
		page.Results = found[from:min(from+pageSize, len(found))]
	}
	reply(w, http.StatusOK, page)
}

func matches(m scenario.TMDBMovie, query string) bool {
	titles := []string{m.Title, m.OriginalTitle}
	for _, t := range m.Translations {
		titles = append(titles, t.Title)
	}
	return slices.ContainsFunc(titles, func(t string) bool { return t != "" && strings.Contains(strings.ToLower(t), query) })
}

func pageParam(v string) int {
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return n
	}
	return 1
}

type movieJSON struct {
	Adult               bool                   `json:"adult"`
	BackdropPath        *string                `json:"backdrop_path"`
	BelongsToCollection *collectionJSON        `json:"belongs_to_collection"`
	Genres              []genreJSON            `json:"genres"`
	ID                  int64                  `json:"id"`
	IMDbID              string                 `json:"imdb_id"`
	OriginalLanguage    string                 `json:"original_language"`
	OriginalTitle       string                 `json:"original_title"`
	Overview            string                 `json:"overview"`
	PosterPath          *string                `json:"poster_path"`
	ProductionCompanies []companyJSON          `json:"production_companies"`
	ReleaseDate         string                 `json:"release_date"`
	Runtime             int64                  `json:"runtime"`
	Status              string                 `json:"status"`
	Tagline             string                 `json:"tagline"`
	Title               string                 `json:"title"`
	VoteAverage         float64                `json:"vote_average"`
	VoteCount           int64                  `json:"vote_count"`
	ExternalIDs         *externalIDsJSON       `json:"external_ids,omitempty"`
	ReleaseDates        *releaseDatesJSON      `json:"release_dates,omitempty"`
	AlternativeTitles   *alternativeTitlesJSON `json:"alternative_titles,omitempty"`
	Credits             *creditsJSON           `json:"credits,omitempty"`
	Videos              *videosJSON            `json:"videos,omitempty"`
	Images              *imagesJSON            `json:"images,omitempty"`
	Translations        *translationsJSON      `json:"translations,omitempty"`
}

type collectionJSON struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	PosterPath   *string `json:"poster_path"`
	BackdropPath *string `json:"backdrop_path"`
}

type genreJSON struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type companyJSON struct {
	ID            int64   `json:"id"`
	LogoPath      *string `json:"logo_path"`
	Name          string  `json:"name"`
	OriginCountry string  `json:"origin_country"`
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
	Certification string   `json:"certification"`
	Descriptors   []string `json:"descriptors"`
	Language      string   `json:"iso_639_1"`
	Note          string   `json:"note"`
	ReleaseDate   string   `json:"release_date"`
	Type          int      `json:"type"`
}

type alternativeTitlesJSON struct {
	Titles []titleJSON `json:"titles"`
}

type titleJSON struct {
	Country string `json:"iso_3166_1"`
	Title   string `json:"title"`
	Type    string `json:"type"`
}

type creditsJSON struct {
	Cast []castJSON `json:"cast"`
	Crew []crewJSON `json:"crew"`
}

type castJSON struct {
	Adult       bool    `json:"adult"`
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	ProfilePath *string `json:"profile_path"`
	Character   string  `json:"character"`
	Order       int     `json:"order"`
}

type crewJSON struct {
	Adult       bool    `json:"adult"`
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	ProfilePath *string `json:"profile_path"`
	Department  string  `json:"department"`
	Job         string  `json:"job"`
}

type videosJSON struct {
	Results []videoJSON `json:"results"`
}

type videoJSON struct {
	Language    string `json:"iso_639_1"`
	Country     string `json:"iso_3166_1"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	Site        string `json:"site"`
	Size        int    `json:"size"`
	Type        string `json:"type"`
	Official    bool   `json:"official"`
	PublishedAt string `json:"published_at"`
	ID          string `json:"id"`
}

type imagesJSON struct {
	Backdrops []imageJSON `json:"backdrops"`
	Logos     []imageJSON `json:"logos"`
	Posters   []imageJSON `json:"posters"`
}

type imageJSON struct {
	AspectRatio float64 `json:"aspect_ratio"`
	Height      int     `json:"height"`
	Country     *string `json:"iso_3166_1"`
	Language    *string `json:"iso_639_1"`
	FilePath    string  `json:"file_path"`
	VoteAverage float64 `json:"vote_average"`
	VoteCount   int64   `json:"vote_count"`
	Width       int     `json:"width"`
}

type translationsJSON struct {
	Translations []translationJSON `json:"translations"`
}

type translationJSON struct {
	Country  string              `json:"iso_3166_1"`
	Language string              `json:"iso_639_1"`
	Data     translationDataJSON `json:"data"`
}

type translationDataJSON struct {
	Title    string `json:"title"`
	Overview string `json:"overview"`
	Tagline  string `json:"tagline"`
}

func (s *server) movie(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	m, rev, ok := s.live(id)
	if err != nil || !ok {
		notFound(w)
		return
	}
	q := r.URL.Query()
	lang := q.Get("language")
	t := text(m, rev, lang)
	out := movieJSON{
		ID: m.ID, IMDbID: m.IMDbID, OriginalLanguage: originalLanguage(m), OriginalTitle: originalTitle(m),
		Overview: t.Overview, Tagline: t.Tagline, Title: t.Title, ReleaseDate: primary(m),
		Runtime: int64(m.Runtime.Minutes()), Status: cmp.Or(m.Status, "Released"),
		VoteAverage: m.VoteAverage, VoteCount: m.VoteCount + int64(rev),
		PosterPath: pathOf(m, "poster", lang), BackdropPath: pathOf(m, "backdrop", ""),
		Genres: []genreJSON{}, ProductionCompanies: []companyJSON{},
	}
	for _, g := range m.Genres {
		out.Genres = append(out.Genres, genreJSON{ID: int64(hash("genre", g) % 100000), Name: g})
	}
	if m.Studio != nil {
		out.ProductionCompanies = append(out.ProductionCompanies, companyJSON{ID: m.Studio.ID, Name: m.Studio.Name, LogoPath: file("company", strconv.FormatInt(m.Studio.ID, 10), ".png"), OriginCountry: "US"})
	}
	if m.Collection != nil {
		out.BelongsToCollection = &collectionJSON{ID: m.Collection.ID, Name: m.Collection.Name}
	}
	imageLanguages := languagesParam(q.Get("include_image_language"), lang, true)
	videoLanguages := languagesParam(q.Get("include_video_language"), lang, false)
	for part := range strings.SplitSeq(q.Get("append_to_response"), ",") {
		switch part {
		case "external_ids":
			out.ExternalIDs = &externalIDsJSON{IMDbID: m.IMDbID}
		case "release_dates":
			out.ReleaseDates = releaseDates(m)
		case "alternative_titles":
			out.AlternativeTitles = &alternativeTitlesJSON{Titles: []titleJSON{}}
			for _, at := range m.Titles {
				out.AlternativeTitles.Titles = append(out.AlternativeTitles.Titles, titleJSON{Country: at.Country, Title: at.Title, Type: at.Kind})
			}
		case "credits":
			out.Credits = credits(m)
		case "videos":
			out.Videos = videos(m, videoLanguages)
		case "images":
			out.Images = imageSet(m, imageLanguages)
		case "translations":
			out.Translations = translations(m, rev)
		}
	}
	reply(w, http.StatusOK, out)
}

// text is the title, overview and tagline in lang: a translation's where
// the movie has one, else the English title and no overview or tagline.
func text(m scenario.TMDBMovie, rev int, lang string) translationDataJSON {
	english := translationDataJSON{Title: m.Title, Overview: overview(m.Overview, rev), Tagline: m.Tagline}
	if lang == "" || strings.HasPrefix(lang, "en") {
		return english
	}
	iso, _, _ := strings.Cut(lang, "-")
	i := slices.IndexFunc(m.Translations, func(t scenario.TMDBTranslation) bool { return t.Language == lang })
	if i < 0 {
		i = slices.IndexFunc(m.Translations, func(t scenario.TMDBTranslation) bool { return strings.HasPrefix(t.Language, iso+"-") })
	}
	if i < 0 {
		return translationDataJSON{Title: m.Title}
	}
	t := m.Translations[i]
	return translationDataJSON{Title: cmp.Or(t.Title, m.Title), Overview: overview(t.Overview, rev), Tagline: t.Tagline}
}

// overview is the Scenario's overview, marked by each Change cue.
func overview(s string, rev int) string {
	for n := range rev {
		s = strings.TrimSpace(fmt.Sprintf("%s Revision %d.", s, n+1))
	}
	return s
}

func originalTitle(m scenario.TMDBMovie) string { return cmp.Or(m.OriginalTitle, m.Title) }

func originalLanguage(m scenario.TMDBMovie) string { return cmp.Or(m.OriginalLanguage, "en") }

// dates is every release the movie lists, released as one US digital
// date among them.
func dates(m scenario.TMDBMovie) []scenario.TMDBDate {
	out := slices.Clone(m.Dates)
	if m.Released != "" {
		out = append(out, scenario.TMDBDate{Country: "US", Kind: "digital", Date: m.Released})
	}
	return out
}

func kind(d scenario.TMDBDate) int { return slices.Index(scenario.TMDBDateKinds, d.Kind) + 1 }

// primary is TMDB's primary release date: the earliest premiere or cinema
// date, else the earliest of any kind, else none.
func primary(m scenario.TMDBMovie) string {
	var cinema, all []string
	for _, d := range dates(m) {
		all = append(all, d.Date)
		if kind(d) <= 3 {
			cinema = append(cinema, d.Date)
		}
	}
	switch {
	case len(cinema) > 0:
		return slices.Min(cinema)
	case len(all) > 0:
		return slices.Min(all)
	}
	return ""
}

func releaseDates(m scenario.TMDBMovie) *releaseDatesJSON {
	out := &releaseDatesJSON{Results: []countryJSON{}}
	ds := dates(m)
	slices.SortStableFunc(ds, func(a, b scenario.TMDBDate) int {
		return cmp.Or(cmp.Compare(a.Country, b.Country), cmp.Compare(a.Date, b.Date), cmp.Compare(kind(a), kind(b)))
	})
	for _, d := range ds {
		if n := len(out.Results); n == 0 || out.Results[n-1].Country != d.Country {
			out.Results = append(out.Results, countryJSON{Country: d.Country})
		}
		c := &out.Results[len(out.Results)-1]
		c.ReleaseDates = append(c.ReleaseDates, releaseJSON{
			Certification: d.Certification, Descriptors: []string{}, Note: d.Note, ReleaseDate: d.Date + "T00:00:00.000Z", Type: kind(d),
		})
	}
	return out
}

func credits(m scenario.TMDBMovie) *creditsJSON {
	out := &creditsJSON{Cast: []castJSON{}, Crew: []crewJSON{}}
	for i, c := range m.Credits.Cast {
		out.Cast = append(out.Cast, castJSON{ID: c.ID, Name: c.Name, Character: c.Character, Order: i, ProfilePath: file("person", strconv.FormatInt(c.ID, 10), ".jpg")})
	}
	for _, c := range m.Credits.Crew {
		out.Crew = append(out.Crew, crewJSON{ID: c.ID, Name: c.Name, Department: c.Department, Job: c.Job, ProfilePath: file("person", strconv.FormatInt(c.ID, 10), ".jpg")})
	}
	return out
}

func videos(m scenario.TMDBMovie, langs []string) *videosJSON {
	out := &videosJSON{Results: []videoJSON{}}
	if m.Trailer != "" && slices.Contains(langs, "en") {
		out.Results = append(out.Results, videoJSON{
			Language: "en", Country: "US", Name: "Official Trailer", Key: m.Trailer, Site: "YouTube", Size: 1080,
			Type: "Trailer", Official: true, PublishedAt: primary(m) + "T16:00:00.000Z", ID: fmt.Sprintf("%024x", hash("video", m.Trailer)),
		})
	}
	return out
}

// images is the movie's images as the Scenario lists them, or a poster
// and title treatment in English and a textless backdrop.
func images(m scenario.TMDBMovie) []scenario.TMDBImage {
	if len(m.Images) > 0 {
		return m.Images
	}
	return []scenario.TMDBImage{{Shape: "poster", Language: "en", Votes: 5.5}, {Shape: "backdrop", Votes: 5.5}, {Shape: "logo", Language: "en", Votes: 5.5}}
}

var dimensions = map[string][2]int{"poster": {2000, 3000}, "backdrop": {3840, 2160}, "logo": {1000, 400}}

func imageSet(m scenario.TMDBMovie, langs []string) *imagesJSON {
	out := &imagesJSON{Backdrops: []imageJSON{}, Logos: []imageJSON{}, Posters: []imageJSON{}}
	lists := map[string]*[]imageJSON{"poster": &out.Posters, "backdrop": &out.Backdrops, "logo": &out.Logos}
	for i, im := range images(m) {
		if !slices.Contains(langs, cmp.Or(im.Language, "null")) {
			continue
		}
		d := dimensions[im.Shape]
		img := imageJSON{FilePath: *imagePath(m, i), VoteAverage: im.Votes, VoteCount: 1, Width: d[0], Height: d[1], AspectRatio: float64(d[0]) / float64(d[1])}
		if im.Language != "" {
			img.Language = &im.Language
		}
		*lists[im.Shape] = append(*lists[im.Shape], img)
	}
	return out
}

// pathOf is the file path of the movie's best image of shape in lang, or
// with no text when lang is "".
func pathOf(m scenario.TMDBMovie, shape, lang string) *string {
	iso, _, _ := strings.Cut(lang, "-")
	ims := images(m)
	for _, want := range []string{iso, "en", ""} {
		if i := slices.IndexFunc(ims, func(im scenario.TMDBImage) bool { return im.Shape == shape && im.Language == want }); i >= 0 {
			return imagePath(m, i)
		}
	}
	if i := slices.IndexFunc(ims, func(im scenario.TMDBImage) bool { return im.Shape == shape }); i >= 0 {
		return imagePath(m, i)
	}
	return nil
}

func imagePath(m scenario.TMDBMovie, i int) *string {
	ext := ".jpg"
	if images(m)[i].Shape == "logo" {
		ext = ".png"
	}
	return file("image", fmt.Sprintf("%d/%d", m.ID, i), ext)
}

// translations lists English first. TMDB keeps the title of the original
// language on the record, so that translation's title is empty.
func translations(m scenario.TMDBMovie, rev int) *translationsJSON {
	english := text(m, rev, "en-US")
	if originalLanguage(m) == "en" {
		english.Title = ""
	}
	out := &translationsJSON{Translations: []translationJSON{{Country: "US", Language: "en", Data: english}}}
	for _, t := range m.Translations {
		iso, country, _ := strings.Cut(t.Language, "-")
		out.Translations = append(out.Translations, translationJSON{Country: country, Language: iso, Data: translationDataJSON{Title: t.Title, Overview: overview(t.Overview, rev), Tagline: t.Tagline}})
	}
	return out
}

// languagesParam is an include_*_language list, or by default lang's
// two-letter code, with "null" for images with no text.
func languagesParam(v, lang string, null bool) []string {
	if v != "" {
		return strings.Split(v, ",")
	}
	iso, _, _ := strings.Cut(cmp.Or(lang, "en"), "-")
	if null {
		return []string{iso, "null"}
	}
	return []string{iso}
}

type changeJSON struct {
	ID       int64 `json:"id"`
	Adult    bool  `json:"adult"`
	Softcore bool  `json:"softcore"`
}

type changesJSON struct {
	Results      []changeJSON `json:"results"`
	Page         int          `json:"page"`
	TotalPages   int          `json:"total_pages"`
	TotalResults int          `json:"total_results"`
}

// changed lists each movie changed from start_date to end_date, both
// whole UTC days, once.
func (s *server) changed(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	now := time.Now().UTC()
	start, end := cmp.Or(q.Get("start_date"), now.Add(-24*time.Hour).Format(time.DateOnly)), cmp.Or(q.Get("end_date"), now.Format(time.DateOnly))
	s.mu.Lock()
	var ids []int64
	for _, c := range s.changes {
		day := c.at.UTC().Format(time.DateOnly)
		if day >= start && day <= end && !slices.Contains(ids, c.id) {
			ids = append(ids, c.id)
		}
	}
	s.mu.Unlock()
	out := changesJSON{Results: []changeJSON{}, Page: pageParam(q.Get("page")), TotalResults: len(ids)}
	out.TotalPages = (len(ids) + changesPageSize - 1) / changesPageSize
	if from := (out.Page - 1) * changesPageSize; from < len(ids) {
		for _, id := range ids[from:min(from+changesPageSize, len(ids))] {
			out.Results = append(out.Results, changeJSON{ID: id})
		}
	}
	reply(w, http.StatusOK, out)
}

type configurationJSON struct {
	ChangeKeys []string      `json:"change_keys"`
	Images     imageHostJSON `json:"images"`
}

type imageHostJSON struct {
	BaseURL       string   `json:"base_url"`
	SecureBaseURL string   `json:"secure_base_url"`
	BackdropSizes []string `json:"backdrop_sizes"`
	LogoSizes     []string `json:"logo_sizes"`
	PosterSizes   []string `json:"poster_sizes"`
	ProfileSizes  []string `json:"profile_sizes"`
	StillSizes    []string `json:"still_sizes"`
}

func (s *server) configuration(w http.ResponseWriter, r *http.Request) {
	base := "http://" + r.Host + imageRoot + "/"
	reply(w, http.StatusOK, configurationJSON{
		ChangeKeys: []string{"alternative_titles", "cast", "crew", "images", "overview", "release_dates", "runtime", "tagline", "title", "translations", "videos"},
		Images: imageHostJSON{
			BaseURL: base, SecureBaseURL: "https://" + r.Host + imageRoot + "/",
			BackdropSizes: []string{"w300", "w780", "w1280", "original"},
			LogoSizes:     []string{"w45", "w92", "w154", "w185", "w300", "w500", "original"},
			PosterSizes:   []string{"w92", "w154", "w185", "w342", "w500", "w780", "original"},
			ProfileSizes:  []string{"w45", "w185", "h632", "original"},
			StillSizes:    []string{"w92", "w185", "w300", "original"},
		},
	})
}

// file is a made-up TMDB file path, fixed for kind and key.
func file(kind, key, ext string) *string {
	p := "/" + strconv.FormatUint(hash(kind, key), 36) + ext
	return &p
}

func hash(parts ...string) uint64 {
	h := fnv.New64a()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
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

package movie

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/works"
)

// record is one movie as its sources have it: Fetched.Detail on the wire
// and the movie's rows at rest, so Store compares the two directly. Its
// lists are in the order the rows read back.
type record struct {
	TMDBID           db.TMDBID      `json:"tmdbId"`
	IMDbID           *db.IMDbID     `json:"imdbId,omitempty"`
	Title            string         `json:"title"`
	TitleEnglish     string         `json:"titleEnglish"`
	OriginalTitle    string         `json:"originalTitle"`
	OriginalLanguage string         `json:"originalLanguage"`
	Year             *int64         `json:"year,omitempty"`
	PrimaryRelease   *string        `json:"primaryRelease,omitempty"`
	Overview         string         `json:"overview,omitempty"`
	Tagline          string         `json:"tagline,omitempty"`
	RuntimeMinutes   *int64         `json:"runtimeMinutes,omitempty"`
	Status           status         `json:"status"`
	Certification    *certification `json:"certification,omitempty"`
	Studio           *studio        `json:"studio,omitempty"`
	Collection       *collection    `json:"collection,omitempty"`
	Trailer          *trailer       `json:"trailer,omitempty"`
	// Awards is nil when OMDb wasn't read, so Store keeps the stored line,
	// and empty when OMDb has none.
	Awards   *string       `json:"awards,omitempty"`
	Genres   []string      `json:"genres,omitempty"`
	Titles   []altTitle    `json:"titles,omitempty"`
	Releases []releaseDate `json:"releases,omitempty"`
	Credits  []credit      `json:"credits,omitempty"`
}

// status is TMDB's status of a film, as stored.
type status string

// statusWords are the statuses TMDB names, in its words.
var statusWords = map[status]string{
	"rumored": "Rumored", "planned": "Planned", "in-production": "In Production",
	"post-production": "Post Production", "released": "Released", "canceled": "Canceled",
}

// certification is the rating a film carries in Country's system.
type certification struct {
	Code    string `json:"code"`
	Country string `json:"country"`
}

type studio struct {
	ID   db.CompanyID `json:"id"`
	Name string       `json:"name"`
}

type collection struct {
	ID   db.CollectionID `json:"id"`
	Name string          `json:"name"`
}

// trailer names a video on Site, such as YouTube, by its Key there.
type trailer struct {
	Site string `json:"site"`
	Key  string `json:"key"`
}

type altTitle struct {
	Country string `json:"country"`
	Title   string `json:"title"`
	Kind    string `json:"kind,omitempty"`
}

type releaseDate struct {
	Country       string           `json:"country"`
	Kind          tmdb.ReleaseType `json:"kind"`
	Date          string           `json:"date"`
	Certification string           `json:"certification,omitempty"`
	Note          string           `json:"note,omitempty"`
}

type department string

const (
	directing department = "directing"
	writing   department = "writing"
	acting    department = "acting"
)

var departmentWords = map[department]string{directing: "Directing", writing: "Writing", acting: "Acting"}

// credit is one person's part: a job, or a character for the cast.
type credit struct {
	PersonID   db.PersonID `json:"personId"`
	Name       string      `json:"name"`
	Department department  `json:"department"`
	Role       string      `json:"role,omitempty"`
	Photo      *string     `json:"photo,omitempty"`
}

// topCast is how many of the cast a movie keeps, by billing.
const topCast = 20

// recordOf reads TMDB's movie m as ask asked for it: texts in its
// language, the certification of its region. Awards stay nil.
func recordOf(m tmdb.Movie, ask metadata.Ask) (record, error) {
	st, ok := parseStatus(m.Status)
	if !ok {
		return record{}, fmt.Errorf("TMDB movie %d has status %q: %w", m.ID, m.Status, works.ErrInvalid)
	}
	if m.OriginalLanguage == "" {
		return record{}, fmt.Errorf("TMDB movie %d names no original language: %w", m.ID, works.ErrInvalid)
	}
	title := cmp.Or(m.Title, m.OriginalTitle)
	r := record{
		TMDBID:           db.TMDBID(m.ID),
		Title:            title,
		TitleEnglish:     cmp.Or(englishTitle(m), m.OriginalTitle, title),
		OriginalTitle:    cmp.Or(m.OriginalTitle, title),
		OriginalLanguage: m.OriginalLanguage,
		Overview:         m.Overview,
		Tagline:          m.Tagline,
		Status:           st,
		Trailer:          trailerOf(m.Videos.Results, ask.Language),
		Releases:         releasesOf(m.ReleaseDates),
		Credits:          creditsOf(m.Credits),
	}
	if id := cmp.Or(m.ExternalIDs.IMDbID, m.IMDbID); id != "" {
		imdb := db.IMDbID(id)
		r.IMDbID = &imdb
	}
	if r.Year = year(m.ReleaseDate); r.Year != nil {
		r.PrimaryRelease = &m.ReleaseDate
	}
	if m.Runtime > 0 {
		r.RuntimeMinutes = &m.Runtime
	}
	r.Certification = certificationIn(r.Releases, ask.Region)
	if c, ok := studioOf(m); ok {
		r.Studio = &studio{ID: db.CompanyID(c.ID), Name: c.Name}
	}
	if c := m.BelongsToCollection; c != nil && c.ID >= 1 && c.Name != "" {
		r.Collection = &collection{ID: db.CollectionID(c.ID), Name: c.Name}
	}
	for _, g := range m.Genres {
		if g.Name != "" {
			r.Genres = append(r.Genres, g.Name)
		}
	}
	for _, t := range m.AlternativeTitles.Titles {
		if len(t.Country) == 2 && t.Title != "" {
			r.Titles = append(r.Titles, altTitle{Country: t.Country, Title: t.Title, Kind: t.Type})
		}
	}
	slices.SortFunc(r.Titles, func(a, b altTitle) int {
		return cmp.Or(strings.Compare(a.Country, b.Country), strings.Compare(a.Title, b.Title), strings.Compare(a.Kind, b.Kind))
	})
	return r, nil
}

func parseStatus(words string) (status, bool) {
	for s, w := range statusWords {
		if w == words {
			return s, true
		}
	}
	return "", false
}

func dayOf(date string) (time.Time, bool) {
	t, err := time.Parse(time.DateOnly, date)
	return t, err == nil
}

// englishTitle is the title of TMDB's US English translation, else of any
// English one. TMDB leaves it empty when English is the original
// language, whose title is the original title.
func englishTitle(m tmdb.Movie) string {
	var english string
	for _, t := range m.Translations.Translations {
		switch {
		case t.Language != "en" || t.Data.Title == "":
		case t.Country == "US":
			return t.Data.Title
		case english == "":
			english = t.Data.Title
		}
	}
	return english
}

// releasesOf is every dated release TMDB lists of a kind it documents, in
// the order the rows read back.
func releasesOf(rd tmdb.ReleaseDates) []releaseDate {
	var out []releaseDate
	for _, c := range rd.Results {
		for _, r := range c.Releases {
			date, _, _ := strings.Cut(r.Date, "T")
			if _, ok := dayOf(date); !ok || len(c.Country) != 2 || r.Type < tmdb.Premiere || r.Type > tmdb.TV {
				continue
			}
			out = append(out, releaseDate{Country: c.Country, Kind: r.Type, Date: date, Certification: r.Certification, Note: r.Note})
		}
	}
	slices.SortFunc(out, compareReleases)
	return out
}

func compareReleases(a, b releaseDate) int {
	return cmp.Or(strings.Compare(a.Country, b.Country), cmp.Compare(a.Kind, b.Kind), strings.Compare(a.Date, b.Date),
		strings.Compare(a.Certification, b.Certification), strings.Compare(a.Note, b.Note))
}

// certificationIn is the certification of region's earliest release that
// carries one, nil when none does.
func certificationIn(rs []releaseDate, region string) *certification {
	var first *releaseDate
	for i, r := range rs {
		if r.Country == region && r.Certification != "" && (first == nil || r.Date < first.Date) {
			first = &rs[i]
		}
	}
	if first == nil {
		return nil
	}
	return &certification{Code: first.Certification, Country: region}
}

// creditsOf keeps the directors, then the writers, then the top-billed
// cast.
func creditsOf(c tmdb.Credits) []credit {
	var out []credit
	add := func(id int64, name string, d department, role, photo string) {
		if id < 1 || name == "" {
			return
		}
		cr := credit{PersonID: db.PersonID(id), Name: name, Department: d, Role: role}
		if photo != "" {
			cr.Photo = &photo
		}
		out = append(out, cr)
	}
	for _, p := range c.Crew {
		if p.Job == "Director" {
			add(p.ID, p.Name, directing, p.Job, p.ProfilePath)
		}
	}
	for _, p := range c.Crew {
		if p.Department == "Writing" {
			add(p.ID, p.Name, writing, p.Job, p.ProfilePath)
		}
	}
	cast := slices.SortedStableFunc(slices.Values(c.Cast), func(a, b tmdb.Cast) int { return cmp.Compare(a.Order, b.Order) })
	for _, p := range cast[:min(len(cast), topCast)] {
		add(p.ID, p.Name, acting, p.Character, p.ProfilePath)
	}
	return out
}

// trailerOf is the official trailer first, in language, then English, then
// any, the earliest published first; nil when TMDB links none.
func trailerOf(vs []tmdb.Video, language string) *trailer {
	rank := languageRank(language)
	unofficial := func(v tmdb.Video) int {
		if v.Official {
			return 0
		}
		return 1
	}
	var first *tmdb.Video
	for i, v := range vs {
		if v.Type != "Trailer" || v.Site == "" || v.Key == "" {
			continue
		}
		if first == nil || cmp.Or(cmp.Compare(unofficial(v), unofficial(*first)), cmp.Compare(rank(v.Language), rank(first.Language)),
			strings.Compare(v.PublishedAt, first.PublishedAt)) < 0 {
			first = &vs[i]
		}
	}
	if first == nil {
		return nil
	}
	return &trailer{Site: first.Site, Key: first.Key}
}

// languageRank orders TMDB's two-letter languages: the metadata
// language's, then English, then any other or none.
func languageRank(language string) func(string) int {
	iso, _, _ := strings.Cut(language, "-")
	order := []string{iso, "en"}
	return func(l string) int {
		if i := slices.Index(order, l); i >= 0 && l != "" {
			return i
		}
		return len(order)
	}
}

// The artwork kinds a movie declares, each a TMDB image. The title
// treatment is the film's title in its own lettering, which TMDB calls a
// logo.
var (
	posterKind         = artwork.Kind{Key: "poster", Label: "Poster", Shape: artwork.ShapePoster, Prefetch: artwork.SizeMedium}
	backdropKind       = artwork.Kind{Key: "backdrop", Label: "Backdrop", Shape: artwork.ShapeBackdrop, Prefetch: artwork.SizeLarge}
	titleTreatmentKind = artwork.Kind{Key: "title_treatment", Label: "Title treatment", Shape: artwork.ShapeLogo}
	studioLogoKind     = artwork.Kind{Key: "studio_logo", Label: "Studio logo", Shape: artwork.ShapeLogo}
)

// artworkOf picks one image of each kind TMDB has. A poster and a title
// treatment prefer language, then English, then any; a backdrop prefers no
// text. Each prefers the highest voted next.
func artworkOf(m tmdb.Movie, language string) []works.Art {
	rank := languageRank(language)
	textless := func(l string) int {
		if l == "" {
			return 0
		}
		return 1 + rank(l)
	}
	company, _ := studioOf(m)
	var out []works.Art
	for _, p := range []struct {
		kind artwork.Kind
		path string
	}{
		{posterKind, cmp.Or(best(m.Images.Posters, rank), m.PosterPath)},
		{backdropKind, cmp.Or(best(m.Images.Backdrops, textless), m.BackdropPath)},
		{titleTreatmentKind, best(m.Images.Logos, rank)},
		{studioLogoKind, company.LogoPath},
	} {
		if p.path != "" {
			out = append(out, works.Art{Kind: p.kind.Key, Ref: imageRef(p.kind.Shape, p.path)})
		}
	}
	return out
}

// studioOf is the first production company TMDB names.
func studioOf(m tmdb.Movie) (tmdb.Company, bool) {
	i := slices.IndexFunc(m.ProductionCompanies, func(c tmdb.Company) bool { return c.ID >= 1 && c.Name != "" })
	if i < 0 {
		return tmdb.Company{}, false
	}
	return m.ProductionCompanies[i], true
}

func best(ims []tmdb.Image, rank func(string) int) string {
	if len(ims) == 0 {
		return ""
	}
	return slices.MinFunc(ims, func(a, b tmdb.Image) int {
		return cmp.Or(cmp.Compare(rank(a.Language), rank(b.Language)), cmp.Compare(b.VoteAverage, a.VoteAverage), cmp.Compare(b.VoteCount, a.VoteCount))
	}).FilePath
}

func imageRef(shape artwork.Shape, path string) artwork.Ref {
	return artwork.Ref{Source: tmdb.ImagesID, Shape: shape, Path: path}
}

// tmdbRating is TMDB's own rating, none while nobody has voted.
func tmdbRating(m tmdb.Movie) []works.Rating {
	if m.VoteCount < 1 {
		return nil
	}
	votes := m.VoteCount
	return []works.Rating{{Source: "tmdb", Value: m.VoteAverage, Votes: &votes}}
}

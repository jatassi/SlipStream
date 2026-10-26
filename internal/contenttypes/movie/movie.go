// Package movie is the movie Content Type: a movie is the Work, with one
// Item, read from TMDB.
package movie

import (
	"cmp"
	"context"
	"embed"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

var Def = contenttype.Define("movie", schemaFiles, DefaultEndpoints, contenttype.Halves[Endpoints]{
	Provider: func(d contenttype.ProviderDeps, _ Endpoints) metadata.Source {
		return &Source{tmdb: d.TMDB, omdb: d.OMDb}
	},
	Library: func(d contenttype.LibraryDeps, e Endpoints) contenttype.Library {
		return &Movie{region: d.Region, web: e}
	},
})

var Description = works.Description{
	Name:   "Movies",
	Icon:   works.IconFilm,
	Source: "TMDB",
	Work:   works.Noun{One: "Movie", Many: "Movies"},
	Item:   works.Noun{One: "Movie", Many: "Movies"},
	Items:  works.ItemsOne,
	Fields: []works.FieldSpec{
		{Key: "english_title", Label: "English title", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "original_title", Label: "Original title", Kind: works.KindText, In: in(works.PlaceAbout)},
		{Key: "original_language", Label: "Original language", Kind: works.KindText, In: in(works.PlaceAbout)},
		{Key: "year", Label: "Year", Kind: works.KindNumber, In: in(works.PlaceFacts), List: works.ListColumn | works.ListSort | works.ListFilter},
		{Key: "release_date", Label: "Release date", Kind: works.KindDate, In: in(works.PlaceAbout)},
		{Key: "runtime", Label: "Runtime", Kind: works.KindDuration, In: in(works.PlaceFacts), List: works.ListColumn | works.ListSort},
		{Key: "status", Label: "Status", Kind: works.KindText, In: in(works.PlaceDetails)},
		{
			Key: "certification", Label: "Certification", Kind: works.KindText, In: in(works.PlaceFacts, works.PlaceAbout),
			With: "certification_country", List: works.ListColumn | works.ListSort | works.ListFilter,
		},
		{Key: "certification_country", Label: "Certification country", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "genres", Label: "Genres", Kind: works.KindTexts, In: in(works.PlaceFacts, works.PlaceAbout), List: works.ListFilter},
		{Key: "tagline", Label: "Tagline", Kind: works.KindText, In: in(works.PlaceTagline)},
		{Key: "overview", Label: "Overview", Kind: works.KindText, In: in(works.PlaceOverview)},
		{Key: "studio", Label: "Studio", Kind: works.KindText, In: in(works.PlaceAbout), Art: studioLogoKind.Key},
		{Key: "collection", Label: "Collection", Kind: works.KindText, In: in(works.PlaceAbout)},
		{Key: "credits", Label: "Cast and crew", Kind: works.KindPeople, In: in(works.PlaceCredits)},
		{Key: "alternative_titles", Label: "Alternative titles", Kind: works.KindTexts, In: in(works.PlaceDetails)},
		{Key: "imdb_id", Label: "IMDb id", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "trailer_site", Label: "Trailer site", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "trailer_key", Label: "Trailer key", Kind: works.KindText, In: in(works.PlaceDetails)},
		{Key: "awards", Label: "Awards", Kind: works.KindText, In: in(works.PlaceAbout)},
	},
	Artwork: []artwork.Kind{posterKind, backdropKind, titleTreatmentKind, studioLogoKind},
	Art:     works.ArtRoles{Poster: posterKind.Key, Backdrop: backdropKind.Key, Title: titleTreatmentKind.Key},
	Ratings: []works.RatingSource{"imdb", "tmdb", "rottentomatoes", "metacritic"},
}

func in(places ...works.Place) []works.Place { return places }

// textFields are rec's text fields that hold a value.
func textFields(rec record) []works.FieldValue {
	values := map[string]string{
		"english_title": rec.TitleEnglish, "original_title": rec.OriginalTitle, "original_language": rec.OriginalLanguage,
		"status": statusWords[rec.Status], "overview": rec.Overview, "tagline": rec.Tagline,
	}
	if rec.IMDbID != nil {
		values["imdb_id"] = string(*rec.IMDbID)
	}
	if rec.Awards != nil {
		values["awards"] = *rec.Awards
	}
	if c := rec.Certification; c != nil {
		values["certification"], values["certification_country"] = c.Code, c.Country
	}
	if rec.Studio != nil {
		values["studio"] = rec.Studio.Name
	}
	if rec.Collection != nil {
		values["collection"] = rec.Collection.Name
	}
	if t := rec.Trailer; t != nil {
		values["trailer_site"], values["trailer_key"] = t.Site, t.Key
	}
	var out []works.FieldValue
	for _, f := range Description.Fields {
		if v := values[f.Key]; f.Kind == works.KindText && v != "" {
			out = append(out, works.TextField(f.Key, v))
		}
	}
	return out
}

// itemKey is the key of a movie's one Item, added with the Work.
const itemKey works.Key = "movie"

// Movie is the library half.
type Movie struct {
	region func(context.Context, storage.Reader[file]) (string, error)
	web    Endpoints
}

func (*Movie) Describe() works.Description { return Description }

// Fields are the movie's record, with the alternative titles of the
// region.
func (m *Movie) Fields(ctx context.Context, r storage.Reader[file], w works.WorkID) (works.FieldSet, error) {
	rec, err := load(ctx, db.New(r), w)
	if err != nil {
		return works.FieldSet{}, err
	}
	region, err := m.region(ctx, r)
	if err != nil {
		return works.FieldSet{}, err
	}
	fields := textFields(rec)
	if rec.Year != nil {
		fields = append(fields, works.NumberField("year", float64(*rec.Year)))
	}
	if rec.PrimaryRelease != nil {
		day, err := works.ParseDate(*rec.PrimaryRelease)
		if err != nil {
			return works.FieldSet{}, fmt.Errorf("movie %d: %w", rec.TMDBID, err)
		}
		fields = append(fields, works.DateField("release_date", day))
	}
	if rec.RuntimeMinutes != nil {
		fields = append(fields, works.DurationField("runtime", time.Duration(*rec.RuntimeMinutes)*time.Minute))
	}
	if len(rec.Genres) > 0 {
		fields = append(fields, works.TextsField("genres", rec.Genres))
	}
	var titles []string
	for _, t := range rec.Titles {
		if t.Country == region {
			titles = append(titles, t.Title)
		}
	}
	if len(titles) > 0 {
		fields = append(fields, works.TextsField("alternative_titles", titles))
	}
	if len(rec.Credits) > 0 {
		people := make([]works.Person, len(rec.Credits))
		for i, c := range rec.Credits {
			people[i] = works.Person{Name: c.Name, Role: c.Role, Department: departmentWords[c.Department]}
			if c.Photo != nil {
				people[i].Photo = imageRef(artwork.ShapePhoto, *c.Photo)
			}
		}
		fields = append(fields, works.PeopleField("credits", people))
	}
	return works.FieldSet{Work: fields}, nil
}

func (*Movie) NamingTokens(ctx context.Context, r storage.Reader[file], item works.ItemID) (map[string]string, error) {
	work, _, err := works.ItemOf(ctx, r, item)
	if err != nil {
		return nil, err
	}
	mv, err := db.New(r).GetMovie(ctx, work.ID)
	if err != nil {
		return nil, err
	}
	tokens := map[string]string{"Movie Title": mv.TitleEnglish, "Year": ""}
	if mv.Year != nil {
		tokens["Year"] = strconv.FormatInt(*mv.Year, 10)
	}
	return tokens, nil
}

func (*Movie) DefaultTemplates() naming.Templates {
	return naming.Templates{Folder: "{Movie Title} ({Year})", File: "{Movie Title} ({Year}) - {Edition Tag} {Target}"}
}

// SearchTerms searches by the movie's TMDB and IMDb ids, else by its
// English title and year.
func (*Movie) SearchTerms(ctx context.Context, r storage.Reader[file], item works.ItemID) (indexers.Terms, error) {
	work, _, err := works.ItemOf(ctx, r, item)
	if err != nil {
		return indexers.Terms{}, err
	}
	mv, err := db.New(r).GetMovie(ctx, work.ID)
	if err != nil {
		return indexers.Terms{}, err
	}
	t := indexers.Terms{Mode: "movie", IDs: map[string]string{"tmdbid": strconv.FormatInt(int64(mv.TmdbID), 10)}, Query: mv.TitleEnglish}
	if mv.ImdbID != nil {
		t.IDs["imdbid"] = string(*mv.ImdbID)
	}
	if mv.Year != nil {
		t.Query += " " + strconv.FormatInt(*mv.Year, 10)
	}
	return t, nil
}

// Timings answers each movie's runtime, and its release date as the day
// it counts as out under the movie settings in the region.
func (m *Movie) Timings(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.ItemID]targets.Timing, error) {
	q := db.New(r)
	movies, err := works.Read(ws, func() ([]db.ListEveryTimingRow, error) { return q.ListEveryTiming(ctx) }, func(ids []works.WorkID) ([]db.ListEveryTimingRow, error) {
		rows, err := q.ListTimingsOf(ctx, ids)
		return as(rows, func(r db.ListTimingsOfRow) db.ListEveryTimingRow { return db.ListEveryTimingRow(r) }), err
	})
	if err != nil {
		return nil, err
	}
	releases, err := works.Read(ws, func() ([]db.ListEveryReleaseDateRow, error) { return q.ListEveryReleaseDate(ctx) }, func(ids []works.WorkID) ([]db.ListEveryReleaseDateRow, error) {
		rows, err := q.ListReleaseDatesOf(ctx, ids)
		return as(rows, func(r db.ListReleaseDatesOfRow) db.ListEveryReleaseDateRow { return db.ListEveryReleaseDateRow(r) }), err
	})
	if err != nil {
		return nil, err
	}
	region, err := m.region(ctx, r)
	if err != nil {
		return nil, err
	}
	rule, err := ruleIn(ctx, q, region)
	if err != nil {
		return nil, err
	}
	items, err := works.ItemsOf(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	of := map[works.WorkID][]releaseDate{}
	for _, rel := range releases {
		of[rel.WorkID] = append(of[rel.WorkID], releaseDate{Country: rel.Country, Kind: tmdb.ReleaseType(rel.Kind), Date: rel.Date})
	}
	for _, rs := range of {
		slices.SortFunc(rs, func(a, b releaseDate) int {
			return cmp.Or(cmp.Compare(a.Country, b.Country), cmp.Compare(a.Kind, b.Kind), cmp.Compare(a.Date, b.Date))
		})
	}
	out := map[works.ItemID]targets.Timing{}
	for _, mv := range movies {
		var t targets.Timing
		if b, ok := rule.available(of[mv.WorkID], mv.PrimaryRelease); ok {
			day, words, err := b.day()
			if err != nil {
				return nil, fmt.Errorf("movie %d: %w", mv.TmdbID, err)
			}
			t.Released, t.Basis = release.Known(day), words
		}
		if mv.RuntimeMinutes != nil {
			t.Runtime = release.Known(time.Duration(*mv.RuntimeMinutes) * time.Minute)
		}
		for _, it := range items[mv.WorkID] {
			out[it.ID] = t
		}
	}
	return out, nil
}

// Cuts answers none: every cut a movie names is built in.
func (*Movie) Cuts() []release.Cut { return nil }

// Growing answers none: a movie never gains Items.
func (*Movie) Growing(context.Context, storage.Reader[file], []works.WorkID) (map[works.WorkID]bool, error) {
	return nil, nil
}

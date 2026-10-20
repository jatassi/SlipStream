// Package movie is the movie Content Type: a movie is the Work, with one
// Item, read from TMDB.
package movie

import (
	"cmp"
	"context"
	"embed"
	"fmt"
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

var Def = contenttype.Define("movie", schemaFiles, nil, contenttype.Halves[struct{}]{
	Provider: func(d contenttype.ProviderDeps, _ struct{}) metadata.Source {
		return &Source{tmdb: d.TMDB, omdb: d.OMDb}
	},
	Library: func(d contenttype.LibraryDeps) contenttype.Library { return &Movie{region: d.Region} },
})

var Description = works.Description{
	Name: "Movies",
	Icon: works.IconFilm,
	Work: works.Noun{One: "Movie", Many: "Movies"},
	Item: works.Noun{One: "Movie", Many: "Movies"},
	Fields: []works.FieldSpec{
		{Key: "english_title", Label: "English title", Tier: works.TierWork, Kind: works.KindText},
		{Key: "original_title", Label: "Original title", Tier: works.TierWork, Kind: works.KindText},
		{Key: "original_language", Label: "Original language", Tier: works.TierWork, Kind: works.KindText},
		{Key: "year", Label: "Year", Tier: works.TierWork, Kind: works.KindNumber},
		{Key: "release_date", Label: "Release date", Tier: works.TierWork, Kind: works.KindDate},
		{Key: "runtime", Label: "Runtime", Tier: works.TierWork, Kind: works.KindDuration},
		{Key: "status", Label: "Status", Tier: works.TierWork, Kind: works.KindText},
		{Key: "certification", Label: "Certification", Tier: works.TierWork, Kind: works.KindText},
		{Key: "certification_country", Label: "Certification country", Tier: works.TierWork, Kind: works.KindText},
		{Key: "genres", Label: "Genres", Tier: works.TierWork, Kind: works.KindTexts},
		{Key: "tagline", Label: "Tagline", Tier: works.TierWork, Kind: works.KindText},
		{Key: "overview", Label: "Overview", Tier: works.TierWork, Kind: works.KindText},
		{Key: "studio", Label: "Studio", Tier: works.TierWork, Kind: works.KindText},
		{Key: "collection", Label: "Collection", Tier: works.TierWork, Kind: works.KindText},
		{Key: "credits", Label: "Cast and crew", Tier: works.TierWork, Kind: works.KindPeople},
		{Key: "alternative_titles", Label: "Alternative titles", Tier: works.TierWork, Kind: works.KindTexts},
		{Key: "imdb_id", Label: "IMDb id", Tier: works.TierWork, Kind: works.KindText},
		{Key: "trailer_site", Label: "Trailer site", Tier: works.TierWork, Kind: works.KindText},
		{Key: "trailer_key", Label: "Trailer key", Tier: works.TierWork, Kind: works.KindText},
		{Key: "awards", Label: "Awards", Tier: works.TierWork, Kind: works.KindText},
	},
	Artwork: []artwork.Kind{posterKind, backdropKind, titleTreatmentKind, studioLogoKind},
}

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

// Timings answers each movie's runtime, and its release date as its
// earliest digital or physical release in the region, else in any country.
func (m *Movie) Timings(ctx context.Context, r storage.Reader[file], ws []works.WorkID) (map[works.ItemID]targets.Timing, error) {
	q := db.New(r)
	movies, err := q.ListMovies(ctx, ws)
	if err != nil {
		return nil, err
	}
	releases, err := q.ListReleases(ctx, ws)
	if err != nil {
		return nil, err
	}
	region, err := m.region(ctx, r)
	if err != nil {
		return nil, err
	}
	items, err := works.ItemsOf(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	of := map[works.WorkID][]releaseDate{}
	for _, rel := range releases {
		of[rel.WorkID] = append(of[rel.WorkID], releaseOf(rel))
	}
	out := map[works.ItemID]targets.Timing{}
	for _, mv := range movies {
		var t targets.Timing
		if date := atHome(of[mv.WorkID], region); date != "" {
			day, err := works.ParseDate(date)
			if err != nil {
				return nil, fmt.Errorf("movie %d: %w", mv.TmdbID, err)
			}
			t.Released = release.Known(day)
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

// atHome is the earliest digital or physical release date in region, else
// in any country, "" when there is none.
func atHome(rs []releaseDate, region string) string {
	var inRegion, anywhere string
	for _, r := range rs {
		if r.Kind != tmdb.Digital && r.Kind != tmdb.Physical {
			continue
		}
		if anywhere == "" || r.Date < anywhere {
			anywhere = r.Date
		}
		if r.Country == region && (inRegion == "" || r.Date < inRegion) {
			inRegion = r.Date
		}
	}
	return cmp.Or(inRegion, anywhere)
}

// Cuts answers none: every cut a movie names is built in.
func (*Movie) Cuts() []release.Cut { return nil }

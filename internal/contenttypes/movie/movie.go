// Package movie is the movie Content Type: a movie is the Work, with one
// Item, read from TMDB.
package movie

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

var Def = contenttype.Define("movie", schemaFiles, nil, New)

var Description = works.Description{
	Name: "Movies",
	Icon: works.IconFilm,
	Work: works.Noun{One: "Movie", Many: "Movies"},
	Item: works.Noun{One: "Movie", Many: "Movies"},
	Fields: []works.FieldSpec{
		{Key: "year", Label: "Year", Tier: works.TierWork, Kind: works.KindNumber},
		{Key: "released", Label: "Released", Tier: works.TierWork, Kind: works.KindDate},
		{Key: "runtime", Label: "Runtime", Tier: works.TierWork, Kind: works.KindDuration},
	},
}

// itemKey is the key of a movie's one Item, added with the Work.
const itemKey works.Key = "movie"

type Movie struct {
	source *tmdb.Client
}

func New(d contenttype.Deps, _ struct{}) contenttype.Type {
	return &Movie{source: d.TMDB}
}

func (*Movie) Describe() works.Description { return Description }

// Search answers TMDB's first page, whose 20 results fit works' 25.
func (m *Movie) Search(ctx context.Context, query string) ([]works.Candidate, error) {
	found, err := m.source.SearchMovies(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make([]works.Candidate, len(found))
	for i, r := range found {
		out[i] = works.Candidate{Key: works.Key(strconv.FormatInt(r.ID, 10)), Label: r.Title}
		if y := year(r.ReleaseDate); y != nil {
			out[i].Fields = []works.FieldValue{works.NumberField("year", float64(*y))}
		}
	}
	return out, nil
}

type detail struct {
	TMDBID         db.TMDBID  `json:"tmdbId"`
	IMDbID         *db.IMDbID `json:"imdbId"`
	Title          string     `json:"title"`
	Year           *int64     `json:"year"`
	Released       *string    `json:"released"`
	RuntimeMinutes *int64     `json:"runtimeMinutes"`
}

// Fetch reads the movie whose TMDB id is key.
func (m *Movie) Fetch(ctx context.Context, key works.Key) (works.Fetched, error) {
	tm, err := m.source.Movie(ctx, string(key))
	if err != nil {
		return works.Fetched{}, err
	}
	d := detail{TMDBID: db.TMDBID(tm.ID), Title: tm.Title, Year: year(tm.ReleaseDate)}
	if tm.ExternalIDs.IMDbID != "" {
		id := db.IMDbID(tm.ExternalIDs.IMDbID)
		d.IMDbID = &id
	}
	if released := tm.Released(); released != "" {
		d.Released = &released
	}
	if tm.Runtime > 0 {
		d.RuntimeMinutes = &tm.Runtime
	}
	f := works.Fetched{Label: tm.Title, Children: []works.Entry{{Key: itemKey, Label: tm.Title}}}
	if f.Detail, err = json.Marshal(d); err != nil {
		return works.Fetched{}, fmt.Errorf("encode movie %s: %w", key, err)
	}
	return f, nil
}

func year(date string) *int64 {
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return nil
	}
	y := int64(t.Year())
	return &y
}

func (*Movie) Store(ctx context.Context, tx *storage.Tx[file], at works.Placed, raw []byte) error {
	var d detail
	if err := json.Unmarshal(raw, &d); err != nil {
		return fmt.Errorf("decode the detail of work %s: %w", at.Work, err)
	}
	return db.New(tx).UpsertMovie(ctx, db.UpsertMovieParams{
		WorkID: at.Work, TmdbID: d.TMDBID, ImdbID: d.IMDbID, Title: d.Title, Year: d.Year, Released: d.Released, RuntimeMinutes: d.RuntimeMinutes,
	})
}

func (*Movie) Fields(ctx context.Context, r storage.Reader[file], w works.WorkID) (works.FieldSet, error) {
	mv, err := db.New(r).GetMovie(ctx, w)
	if err != nil {
		return works.FieldSet{}, err
	}
	var fields []works.FieldValue
	if mv.Year != nil {
		fields = append(fields, works.NumberField("year", float64(*mv.Year)))
	}
	if mv.Released != nil {
		released, err := works.ParseDate(*mv.Released)
		if err != nil {
			return works.FieldSet{}, fmt.Errorf("movie %d: %w", mv.TmdbID, err)
		}
		fields = append(fields, works.DateField("released", released))
	}
	if mv.RuntimeMinutes != nil {
		fields = append(fields, works.DurationField("runtime", time.Duration(*mv.RuntimeMinutes)*time.Minute))
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
	tokens := map[string]string{"Movie Title": mv.Title, "Year": ""}
	if mv.Year != nil {
		tokens["Year"] = strconv.FormatInt(*mv.Year, 10)
	}
	return tokens, nil
}

func (*Movie) DefaultTemplates() naming.Templates {
	return naming.Templates{Folder: "{Movie Title} ({Year})", File: "{Movie Title} ({Year}) - {Edition Tag} {Target}"}
}

// SearchTerms searches by the movie's TMDB and IMDb ids, else by its title
// and year.
func (*Movie) SearchTerms(ctx context.Context, r storage.Reader[file], item works.ItemID) (indexers.Terms, error) {
	work, _, err := works.ItemOf(ctx, r, item)
	if err != nil {
		return indexers.Terms{}, err
	}
	mv, err := db.New(r).GetMovie(ctx, work.ID)
	if err != nil {
		return indexers.Terms{}, err
	}
	t := indexers.Terms{Mode: "movie", IDs: map[string]string{"tmdbid": strconv.FormatInt(int64(mv.TmdbID), 10)}, Query: mv.Title}
	if mv.ImdbID != nil {
		t.IDs["imdbid"] = string(*mv.ImdbID)
	}
	if mv.Year != nil {
		t.Query += " " + strconv.FormatInt(*mv.Year, 10)
	}
	return t, nil
}

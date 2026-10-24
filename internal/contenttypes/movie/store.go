package movie

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/jatassi/slipstream/internal/contenttypes/movie/internal/db"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Store writes the movie detail holds only when it differs from the stored
// one, replacing every row of the movie's: TMDB's record is whole. A
// detail OMDb wasn't read for keeps the stored awards line.
func (*Movie) Store(ctx context.Context, tx *storage.Tx[file], at works.Placed, detail []byte) (bool, error) {
	var r record
	if err := json.Unmarshal(detail, &r); err != nil {
		return false, fmt.Errorf("decode the detail of work %s: %w", at.Work, err)
	}
	q := db.New(tx)
	stored, err := load(ctx, q, at.Work)
	found := !errors.Is(err, storage.ErrNoRows)
	if err != nil && found {
		return false, err
	}
	if r.Awards == nil {
		r.Awards = stored.Awards
	}
	if r.Awards != nil && *r.Awards == "" {
		r.Awards = nil
	}
	if found && reflect.DeepEqual(r, stored) {
		return false, nil
	}
	return true, replace(ctx, q, at.Work, r)
}

// load reads Work w's movie as record, storage.ErrNoRows when it has none.
func load(ctx context.Context, q *db.Queries, w works.WorkID) (record, error) {
	mv, err := q.GetMovie(ctx, w)
	if err != nil {
		return record{}, err
	}
	r := record{
		TMDBID: mv.TmdbID, IMDbID: mv.ImdbID, Title: mv.Title, TitleEnglish: mv.TitleEnglish, OriginalTitle: mv.OriginalTitle,
		OriginalLanguage: mv.OriginalLanguage, Year: mv.Year, PrimaryRelease: mv.PrimaryRelease, Overview: mv.Overview,
		Tagline: mv.Tagline, RuntimeMinutes: mv.RuntimeMinutes, Status: status(mv.Status), Awards: mv.Awards,
	}
	if mv.Certification != nil {
		r.Certification = &certification{Code: *mv.Certification, Country: *mv.CertificationCountry}
	}
	if mv.StudioID != nil {
		r.Studio = &studio{ID: *mv.StudioID, Name: *mv.StudioName}
	}
	if mv.CollectionID != nil {
		r.Collection = &collection{ID: *mv.CollectionID, Name: *mv.CollectionName}
	}
	if mv.TrailerSite != nil {
		r.Trailer = &trailer{Site: *mv.TrailerSite, Key: *mv.TrailerKey}
	}
	if r.Genres, err = q.ListGenres(ctx, w); err != nil {
		return record{}, err
	}
	titles, err := q.ListTitles(ctx, w)
	if err != nil {
		return record{}, err
	}
	for _, t := range titles {
		r.Titles = append(r.Titles, altTitle{Country: t.Country, Title: t.Title, Kind: t.Kind})
	}
	releases, err := q.ListReleases(ctx, []works.WorkID{w})
	if err != nil {
		return record{}, err
	}
	for _, rel := range releases {
		r.Releases = append(r.Releases, releaseOf(rel))
	}
	credits, err := q.ListCredits(ctx, w)
	if err != nil {
		return record{}, err
	}
	for _, c := range credits {
		r.Credits = append(r.Credits, credit{PersonID: c.PersonID, Name: c.Name, Department: department(c.Department), Role: c.Role, Photo: c.PhotoPath})
	}
	return r, nil
}

func releaseOf(r db.MovieRelease) releaseDate {
	return releaseDate{Country: r.Country, Kind: tmdb.ReleaseType(r.Kind), Date: r.Date, Certification: r.Certification, Note: r.Note}
}

func replace(ctx context.Context, q *db.Queries, w works.WorkID, r record) error {
	mv := db.UpsertMovieParams{
		WorkID: w, TmdbID: r.TMDBID, ImdbID: r.IMDbID, Title: r.Title, TitleEnglish: r.TitleEnglish, OriginalTitle: r.OriginalTitle,
		OriginalLanguage: r.OriginalLanguage, Year: r.Year, PrimaryRelease: r.PrimaryRelease, Overview: r.Overview,
		Tagline: r.Tagline, RuntimeMinutes: r.RuntimeMinutes, Status: string(r.Status), Awards: r.Awards,
	}
	if c := r.Certification; c != nil {
		mv.Certification, mv.CertificationCountry = &c.Code, &c.Country
	}
	if s := r.Studio; s != nil {
		mv.StudioID, mv.StudioName = &s.ID, &s.Name
	}
	if c := r.Collection; c != nil {
		mv.CollectionID, mv.CollectionName = &c.ID, &c.Name
	}
	if t := r.Trailer; t != nil {
		mv.TrailerSite, mv.TrailerKey = &t.Site, &t.Key
	}
	if err := q.UpsertMovie(ctx, mv); err != nil {
		return err
	}
	for _, del := range []func(context.Context, works.WorkID) error{q.DeleteGenres, q.DeleteTitles, q.DeleteReleases, q.DeleteCredits} {
		if err := del(ctx, w); err != nil {
			return err
		}
	}
	for i, g := range r.Genres {
		if err := q.InsertGenre(ctx, db.InsertGenreParams{WorkID: w, Position: int64(i + 1), Name: g}); err != nil {
			return err
		}
	}
	for _, t := range r.Titles {
		if err := q.InsertTitle(ctx, db.InsertTitleParams{WorkID: w, Country: t.Country, Title: t.Title, Kind: t.Kind}); err != nil {
			return err
		}
	}
	for _, rel := range r.Releases {
		err := q.InsertRelease(ctx, db.InsertReleaseParams{WorkID: w, Country: rel.Country, Kind: int64(rel.Kind), Date: rel.Date, Certification: rel.Certification, Note: rel.Note})
		if err != nil {
			return err
		}
	}
	for i, c := range r.Credits {
		err := q.InsertCredit(ctx, db.InsertCreditParams{
			WorkID: w, Position: int64(i + 1), PersonID: c.PersonID, Name: c.Name, Department: string(c.Department), Role: c.Role, PhotoPath: c.Photo,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

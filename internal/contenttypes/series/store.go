package series

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Store writes only the rows detail changes: the series row, and each
// season and episode row at's levels and Items hold, by their ids. A
// withdrawn season's or episode's row stays as last stored, and a 10,000-
// episode series whose title changed writes one row. A detail OMDb wasn't
// read for keeps the stored awards line.
func (*Series) Store(ctx context.Context, tx *storage.Tx[file], at works.Placed, detail []byte) (bool, error) {
	r, err := decode(detail)
	if err != nil {
		return false, fmt.Errorf("work %s: %w", at.Work, err)
	}
	q := db.New(tx)
	next, err := showRow(at.Work, r)
	if err != nil {
		return false, err
	}
	stored, err := q.GetSeries(ctx, at.Work)
	found := !errors.Is(err, storage.ErrNoRows)
	if err != nil && found {
		return false, err
	}
	if r.Show.Awards == nil {
		next.Awards = stored.Awards
	}
	changed := !found || !reflect.DeepEqual(db.UpsertSeriesParams(stored), next)
	if changed {
		if err := q.UpsertSeries(ctx, next); err != nil {
			return false, err
		}
	}
	seasons, err := seasonRows(at, r)
	if err != nil {
		return false, err
	}
	storedSeasons, err := q.ListSeasons(ctx, at.Work)
	if err != nil {
		return false, err
	}
	wrote, err := writeChanged(ctx, seasons, byID(storedSeasons, func(s db.SeriesSeason) (works.LevelID, db.UpsertSeasonParams) {
		return s.LevelID, db.UpsertSeasonParams(s)
	}), func(s db.UpsertSeasonParams) works.LevelID { return s.LevelID }, q.UpsertSeason)
	if err != nil {
		return false, err
	}
	changed = changed || wrote
	episodes, err := episodeRows(at, r)
	if err != nil {
		return false, err
	}
	storedEpisodes, err := q.ListEpisodes(ctx, at.Work)
	if err != nil {
		return false, err
	}
	wrote, err = writeChanged(ctx, episodes, byID(storedEpisodes, func(e db.SeriesEpisode) (works.ItemID, db.UpsertEpisodeParams) {
		return e.ItemID, db.UpsertEpisodeParams(e)
	}), func(e db.UpsertEpisodeParams) works.ItemID { return e.ItemID }, q.UpsertEpisode)
	return changed || wrote, err
}

func byID[ID comparable, Stored, Row any](rows []Stored, of func(Stored) (ID, Row)) map[ID]Row {
	out := make(map[ID]Row, len(rows))
	for _, s := range rows {
		id, row := of(s)
		out[id] = row
	}
	return out
}

// writeChanged writes each of next that stored lacks or holds otherwise.
func writeChanged[ID comparable, Row any](ctx context.Context, next []Row, stored map[ID]Row, id func(Row) ID, write func(context.Context, Row) error) (bool, error) {
	changed := false
	for _, row := range next {
		if was, ok := stored[id(row)]; ok && reflect.DeepEqual(was, row) {
			continue
		}
		if err := write(ctx, row); err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
}

func showRow(w works.WorkID, r record) (db.UpsertSeriesParams, error) {
	s := r.Show
	row := db.UpsertSeriesParams{
		WorkID: w, TvdbID: s.TVDBID, TmdbID: s.TMDBID, ImdbID: text(s.IMDbID), EnglishTitle: s.EnglishTitle,
		OriginalTitle: text(s.OriginalTitle), OriginalLanguage: text(s.OriginalLanguage), Country: text(s.Country),
		Network: text(s.Network), Status: string(s.Status), FirstAired: text(s.FirstAired), LastAired: text(s.LastAired),
		NextAired: text(s.NextAired), AverageRuntimeSeconds: seconds(s.AverageRuntime), Overview: text(s.Overview),
		EpisodeOrder: string(r.Order), Awards: s.Awards,
	}
	if row.Awards != nil && *row.Awards == "" {
		row.Awards = nil
	}
	if c := s.Certification; c != nil {
		row.Certification, row.CertificationCountry = &c.Code, &c.Country
	}
	if t := s.Trailer; t != nil {
		row.TrailerSite, row.TrailerKey = &t.Site, &t.Key
	}
	var errs [5]error
	row.Creators, errs[0] = blob(s.Creators)
	row.Credits, errs[1] = blob(s.Cast)
	row.Genres, errs[2] = blob(s.Genres)
	row.Aliases, errs[3] = blob(s.Aliases)
	row.Orders, errs[4] = blob(r.Orders)
	if err := errors.Join(errs[:]...); err != nil {
		return db.UpsertSeriesParams{}, fmt.Errorf("encode series %d: %w", s.TVDBID, err)
	}
	return row, nil
}

func seasonRows(at works.Placed, r record) ([]db.UpsertSeasonParams, error) {
	out := make([]db.UpsertSeasonParams, len(r.Seasons))
	for i, s := range r.Seasons {
		id, ok := at.Levels[keyOf(int64(s.ID))]
		if !ok {
			return nil, fmt.Errorf("series %d: season %d is in the record but not in the Work", r.Show.TVDBID, s.ID)
		}
		out[i] = db.UpsertSeasonParams{
			LevelID: id, WorkID: at.Work, TvdbSeasonID: s.ID, Number: s.Number, EpisodeOrder: string(r.Order),
			Name: text(s.Name), Overview: text(s.Overview),
		}
	}
	return out, nil
}

func episodeRows(at works.Placed, r record) ([]db.UpsertEpisodeParams, error) {
	out := make([]db.UpsertEpisodeParams, len(r.Episodes))
	for i, e := range r.Episodes {
		id, ok := at.Items[keyOf(int64(e.ID))]
		if !ok {
			return nil, fmt.Errorf("series %d: episode %d is in the record but not in the Work", r.Show.TVDBID, e.ID)
		}
		out[i] = db.UpsertEpisodeParams{
			ItemID: id, WorkID: at.Work, TvdbEpisodeID: e.ID, SeasonNumber: e.Season, EpisodeNumber: e.Number,
			AbsoluteNumber: e.Absolute, AiredSeason: e.AiredSeason, AiredNumber: e.AiredNumber, AirDate: text(e.AirDate),
			RuntimeSeconds: seconds(e.Runtime), Overview: text(e.Overview), ImdbID: text(e.IMDbID),
		}
	}
	return out, nil
}

func blob[T any](vs []T) ([]byte, error) {
	if len(vs) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(vs)
	if err != nil {
		return nil, fmt.Errorf("encode a list: %w", err)
	}
	return b, nil
}

func keyOf(id int64) works.Key { return works.Key(strconv.FormatInt(id, 10)) }

func text(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func seconds(d time.Duration) *int64 {
	if d <= 0 {
		return nil
	}
	s := int64(d / time.Second)
	return &s
}

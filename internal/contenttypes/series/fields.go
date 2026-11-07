package series

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

var statusWords = map[tvdb.Status]string{tvdb.Continuing: "Continuing", tvdb.Ended: "Ended", tvdb.Upcoming: "Upcoming"}

// Fields are the series' row, each season's and each episode's, with the
// counts of its live seasons and episodes outside the specials.
func (*Series) Fields(ctx context.Context, r storage.Reader[file], w works.WorkID) (works.FieldSet, error) {
	q := db.New(r)
	row, err := q.GetSeries(ctx, w)
	if err != nil {
		return works.FieldSet{}, err
	}
	items, err := works.ItemsOf(ctx, r, works.Only([]works.WorkID{w}))
	if err != nil {
		return works.FieldSet{}, err
	}
	fields, err := listedFields(db.ListEveryListedRow{
		WorkID: w, Network: row.Network, Certification: row.Certification, Status: row.Status, FirstAired: row.FirstAired, Genres: row.Genres,
	}, countsOf(items[w]))
	if err != nil {
		return works.FieldSet{}, err
	}
	more, err := detailFields(row)
	if err != nil {
		return works.FieldSet{}, err
	}
	set := works.FieldSet{Work: append(fields, more...), Levels: map[works.LevelID][]works.FieldValue{}, Items: map[works.ItemID][]works.FieldValue{}}
	seasons, err := q.ListSeasons(ctx, w)
	if err != nil {
		return works.FieldSet{}, err
	}
	for _, s := range seasons {
		set.Levels[s.LevelID] = append([]works.FieldValue{works.NumberField("season", float64(s.Number))}, texts(kv{"overview", s.Overview})...)
	}
	episodes, err := q.ListEpisodes(ctx, w)
	if err != nil {
		return works.FieldSet{}, err
	}
	for _, e := range episodes {
		if set.Items[e.ItemID], err = episodeFields(tvdb.Order(row.EpisodeOrder), e); err != nil {
			return works.FieldSet{}, fmt.Errorf("series %d: %w", row.TvdbID, err)
		}
	}
	return set, nil
}

// ListFields reads the listed fields of every one of ws in two queries,
// and their Items in one more for the season and episode counts.
func (*Series) ListFields(ctx context.Context, r storage.Reader[file], ws works.Among[works.WorkID]) (map[works.WorkID][]works.FieldValue, error) {
	q := db.New(r)
	rows, err := works.Read(ws, func() ([]db.ListEveryListedRow, error) { return q.ListEveryListed(ctx) }, func(ids []works.WorkID) ([]db.ListEveryListedRow, error) {
		rows, err := q.ListListedOf(ctx, ids)
		return as(rows, func(r db.ListListedOfRow) db.ListEveryListedRow { return db.ListEveryListedRow(r) }), err
	})
	if err != nil {
		return nil, err
	}
	items, err := works.ItemsOf(ctx, r, ws)
	if err != nil {
		return nil, err
	}
	out := make(map[works.WorkID][]works.FieldValue, len(rows))
	for _, row := range rows {
		if out[row.WorkID], err = listedFields(row, countsOf(items[row.WorkID])); err != nil {
			return nil, err
		}
	}
	return out, nil
}

type counts struct{ seasons, episodes int }

func countsOf(items []works.Item) counts {
	seasons := map[works.LevelID]bool{}
	var c counts
	for _, it := range items {
		if it.Live() && !it.Aside {
			seasons[it.LevelID] = true
			c.episodes++
		}
	}
	c.seasons = len(seasons)
	return c
}

func listedFields(row db.ListEveryListedRow, c counts) ([]works.FieldValue, error) {
	fields := texts(kv{"network", row.Network}, kv{"certification", row.Certification})
	fields = append(fields,
		works.TextField("status", statusWords[tvdb.Status(row.Status)]),
		works.NumberField("seasons", float64(c.seasons)),
		works.NumberField("episodes", float64(c.episodes)),
	)
	if row.FirstAired != nil {
		first, err := works.ParseDate(*row.FirstAired)
		if err != nil {
			return nil, fmt.Errorf("work %s: %w", row.WorkID, err)
		}
		fields = append(fields, works.NumberField("year", float64(first.Year)))
	}
	var genres []string
	if err := unblob(row.Genres, &genres); err != nil {
		return nil, fmt.Errorf("work %s: %w", row.WorkID, err)
	}
	if len(genres) > 0 {
		fields = append(fields, works.TextsField("genres", genres))
	}
	return fields, nil
}

// detailFields are the Work-tier fields the Library doesn't list.
func detailFields(row db.SeriesSeries) ([]works.FieldValue, error) {
	var tmdbID *string
	if row.TmdbID != nil {
		tmdbID = text(strconv.FormatInt(int64(*row.TmdbID), 10))
	}
	fields := texts(
		kv{"overview", row.Overview}, kv{"original_title", row.OriginalTitle}, kv{"original_language", row.OriginalLanguage},
		kv{"country", row.Country}, kv{"awards", row.Awards}, kv{"certification_country", row.CertificationCountry},
		kv{"english_title", &row.EnglishTitle}, kv{"episode_order", text(orderWords[tvdb.Order(row.EpisodeOrder)])},
		kv{"tvdb_id", text(strconv.FormatInt(int64(row.TvdbID), 10))}, kv{"tmdb_id", tmdbID}, kv{"imdb_id", row.ImdbID},
		kv{"trailer_site", row.TrailerSite}, kv{"trailer_key", row.TrailerKey},
	)
	aired := map[string]works.Date{}
	for _, d := range []kv{{"first_aired", row.FirstAired}, {"last_aired", row.LastAired}, {"next_aired", row.NextAired}} {
		if d.v == nil {
			continue
		}
		day, err := works.ParseDate(*d.v)
		if err != nil {
			return nil, fmt.Errorf("series %d: %w", row.TvdbID, err)
		}
		aired[d.key] = day
		fields = append(fields, works.DateField(d.key, day))
	}
	if first, ok := aired["first_aired"]; ok {
		fields = append(fields, works.TextField("years", yearsOf(tvdb.Status(row.Status), first, aired["last_aired"])))
	}
	for _, credits := range []struct {
		key  string
		blob []byte
	}{{"creators", row.Creators}, {"credits", row.Credits}} {
		var people []person
		if err := unblob(credits.blob, &people); err != nil {
			return nil, fmt.Errorf("series %d: %w", row.TvdbID, err)
		}
		if len(people) > 0 {
			fields = append(fields, works.PeopleField(credits.key, peopleOf(people, credits.key == "credits")))
		}
	}
	return fields, nil
}

// yearsOf is the span a series has aired: "2022 to present" while it
// runs, "2019 to 2022" once it has ended, and one year when that is all.
func yearsOf(st tvdb.Status, first, last works.Date) string {
	from := strconv.Itoa(first.Year)
	switch {
	case st == tvdb.Continuing:
		return from + " to present"
	case st == tvdb.Ended && last.Year > first.Year:
		return from + " to " + strconv.Itoa(last.Year)
	}
	return from
}

func peopleOf(ps []person, cast bool) []works.Person {
	out := make([]works.Person, len(ps))
	for i, p := range ps {
		out[i] = works.Person{Name: p.Name, Role: p.Role}
		if cast {
			out[i].Department = "Acting"
		}
		if p.Photo != "" {
			out[i].Photo = artwork.Ref{Source: tmdb.ImagesID, Shape: artwork.ShapePhoto, Path: p.Photo}
		}
	}
	return out
}

func episodeFields(order tvdb.Order, e db.SeriesEpisode) ([]works.FieldValue, error) {
	fields := append(texts(kv{"overview", e.Overview}, kv{"imdb_id", e.ImdbID}),
		works.TextField("code", code(order, e.SeasonNumber, e.EpisodeNumber)),
		works.NumberField("season", float64(e.SeasonNumber)),
		works.NumberField("episode", float64(e.EpisodeNumber)),
	)
	if e.AbsoluteNumber != nil {
		fields = append(fields, works.NumberField("absolute", float64(*e.AbsoluteNumber)))
	}
	if e.RuntimeSeconds != nil {
		fields = append(fields, works.DurationField("runtime", time.Duration(*e.RuntimeSeconds)*time.Second))
	}
	if e.AirDate != nil {
		day, err := works.ParseDate(*e.AirDate)
		if err != nil {
			return nil, fmt.Errorf("episode %d: %w", e.TvdbEpisodeID, err)
		}
		fields = append(fields, works.DateField("air_date", day))
	}
	return fields, nil
}

type kv struct {
	key string
	v   *string
}

func texts(pairs ...kv) []works.FieldValue {
	var out []works.FieldValue
	for _, p := range pairs {
		if p.v != nil && *p.v != "" {
			out = append(out, works.TextField(p.key, *p.v))
		}
	}
	return out
}

func unblob(b []byte, v any) error {
	if b == nil {
		return nil
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("decode a stored list: %w", err)
	}
	return nil
}

func as[F, T any](rows []F, to func(F) T) []T {
	out := make([]T, len(rows))
	for i, r := range rows {
		out[i] = to(r)
	}
	return out
}

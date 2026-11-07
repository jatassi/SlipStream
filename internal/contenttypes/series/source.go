package series

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/contenttypes/series/internal/db"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/providers/omdb"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/providers/tvdb"
	"github.com/jatassi/slipstream/internal/works"
)

// Source is the provider half: TVDB for the tree, TMDB for what TVDB
// lacks, and OMDb for ratings.
type Source struct {
	tvdb *tvdb.Client
	tmdb *tmdb.Client
	omdb *omdb.Client
}

// Sources are TMDB's image host and TVDB's artwork host.
func (s *Source) Sources() []artwork.Source {
	return []artwork.Source{s.tmdb.Images(), s.tvdb.Artwork()}
}

// Search asks TVDB. While TVDB can't be reached or refuses the key, it
// asks TMDB and reads each result's TVDB id from its external ids, one
// call a result. A result TVDB doesn't list is keyed "tmdb-<id>" and
// Unlisted. When TMDB fails too, Search answers TVDB's error.
func (s *Source) Search(ctx context.Context, ask metadata.Ask, query string) ([]works.Candidate, error) {
	found, err := s.tvdb.Search(ctx, ask, query)
	if err == nil {
		out := make([]works.Candidate, len(found))
		for i, f := range found {
			out[i] = candidateOf(f)
		}
		return out, nil
	}
	if !errors.Is(err, works.ErrUnavailable) && !errors.Is(err, works.ErrRefused) {
		return nil, err
	}
	results, terr := s.tmdb.SearchSeries(ctx, ask, query)
	if terr != nil {
		return nil, err
	}
	out := make([]works.Candidate, len(results))
	for i, r := range results {
		ids, terr := s.tmdb.SeriesExternalIDs(ctx, ask, r.ID)
		if terr != nil {
			return nil, err
		}
		out[i] = tmdbCandidateOf(r, ids)
	}
	return out, nil
}

func candidateOf(f tvdb.Found) works.Candidate {
	c := works.Candidate{Key: keyOf(int64(f.ID)), Label: f.Name, Overview: firstLine(f.Overview)}
	if f.Original != f.Name {
		c.Original = f.Original
	}
	if f.Poster != "" {
		c.Poster = tvdbRef(artwork.ShapePoster, f.Poster)
	}
	if f.Year > 0 {
		c.Fields = append(c.Fields, works.NumberField("year", float64(f.Year)))
	}
	if f.Network != "" {
		c.Fields = append(c.Fields, works.TextField("network", f.Network))
	}
	return c
}

func tmdbCandidateOf(r tmdb.SeriesFound, ids tmdb.SeriesIDs) works.Candidate {
	c := works.Candidate{Key: keyOf(ids.TVDB), Label: r.Name, Overview: firstLine(r.Overview)}
	if ids.TVDB < 1 {
		c.Key, c.Unlisted = works.Key(tmdbPrefix+strconv.FormatInt(r.ID, 10)), true
	}
	if r.OriginalName != r.Name {
		c.Original = r.OriginalName
	}
	if r.PosterPath != "" {
		c.Poster = tmdbRef(artwork.ShapePoster, r.PosterPath)
	}
	if first, err := works.ParseDate(r.FirstAired); err == nil {
		c.Fields = append(c.Fields, works.NumberField("year", float64(first.Year)))
	}
	return c
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

// Fetch reads the series TVDB numbers key, in the order options name, by
// that id alone: its record and every page of its episodes, its Aired
// order's episodes too in another order, and TMDB's record of it. A
// failed TVDB or TMDB call fails the whole Fetch. OMDb's ratings are read
// when ask asks for them. An OMDb that doesn't respond leaves them as
// stored.
func (s *Source) Fetch(ctx context.Context, ask metadata.Ask, key works.Key, options []byte) (works.Fetched, error) {
	id, err := seriesIDOf(key)
	if err != nil {
		return works.Fetched{}, err
	}
	order, err := orderIn(options)
	if err != nil {
		return works.Fetched{}, err
	}
	var (
		read                           sync.WaitGroup
		show                           tvdb.Series
		tm                             *tmdb.Series
		episodes, aired                []tvdb.Episode
		showErr, episodesErr, airedErr error
	)
	read.Go(func() {
		if show, showErr = s.tvdb.Series(ctx, ask, id); showErr == nil {
			tm, showErr = s.tmdbOf(ctx, ask, show)
		}
	})
	read.Go(func() { episodes, episodesErr = s.tvdb.Episodes(ctx, ask, id, order) })
	if order != tvdb.OrderAired {
		read.Go(func() { aired, airedErr = s.tvdb.Episodes(ctx, ask, id, tvdb.OrderAired) })
	}
	read.Wait()
	if showErr != nil {
		return works.Fetched{}, showErr
	}
	if !slices.Contains(show.Orders, order) {
		return works.Fetched{}, &works.SourceError{Source: "TVDB", Err: works.ErrOrderUnavailable}
	}
	if err := errors.Join(episodesErr, airedErr); err != nil {
		return works.Fetched{}, err
	}
	if order == tvdb.OrderAired {
		aired = episodes
	}
	t, err := treeOf(show, order, episodes, aired)
	if err != nil {
		return works.Fetched{}, err
	}
	t.rec.Show = showOf(show, tm, ask.Region)
	f := works.Fetched{
		Label:     show.Name,
		SortLabel: sortLabelOf(show.Name, ask.Language),
		Children:  t.children,
		Artwork:   append(workArt(show, tm), t.art...),
	}
	if tm != nil && tm.VoteCount > 0 {
		votes := tm.VoteCount
		f.Ratings = []works.Rating{{Source: "tmdb", Value: tm.VoteAverage, Votes: &votes}}
	}
	if imdb := t.rec.Show.IMDbID; ask.Ratings != metadata.RatingsNone && imdb != "" {
		title, err := s.omdb.Title(ctx, ask, imdb)
		switch {
		case err == nil:
			f.Ratings, f.RatingsRead, t.rec.Show.Awards = append(f.Ratings, title.Ratings...), true, &title.Awards
			if ask.Ratings == metadata.RatingsAll {
				f.Ratings = append(f.Ratings, s.episodeRatings(ctx, ask, imdb, t.rec.Episodes)...)
			}
		case errors.Is(err, works.ErrNotFound):
			f.RatingsRead = true
		}
	}
	if f.Detail, err = encode(t.rec); err != nil {
		return works.Fetched{}, err
	}
	return f, nil
}

func seriesIDOf(key works.Key) (tvdb.SeriesID, error) {
	if strings.HasPrefix(string(key), tmdbPrefix) {
		return 0, &works.SourceError{Source: "TVDB", Err: works.ErrNotOnSource}
	}
	n, err := strconv.ParseInt(string(key), 10, 64)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("series %q: %w", key, works.ErrNotFound)
	}
	return tvdb.SeriesID(n), nil
}

func orderIn(raw []byte) (tvdb.Order, error) {
	if raw == nil {
		return orderChoice.Default, nil
	}
	var o options
	if err := json.Unmarshal(raw, &o); err != nil {
		return "", fmt.Errorf("decode the series options %s: %w", raw, err)
	}
	return o.Order, nil
}

// tmdbOf is TMDB's record of show, by the TMDB id TVDB names, else the one
// TMDB finds for its TVDB id. It is nil when TMDB lists none.
func (s *Source) tmdbOf(ctx context.Context, ask metadata.Ask, show tvdb.Series) (*tmdb.Series, error) {
	id := show.TMDB
	if id == 0 {
		found, ok, err := s.tmdb.FindTVDB(ctx, ask, int64(show.ID))
		if err != nil || !ok {
			return nil, err
		}
		id = found
	}
	tm, err := s.tmdb.Series(ctx, ask, id)
	if errors.Is(err, works.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tm, nil
}

type tree struct {
	rec      record
	children []works.Entry
	art      []works.Art
}

// treeOf numbers episodes in order, each season of the order a level in
// number order with the specials last and aside. Each episode keeps its
// Aired numbers from aired, whatever the order.
func treeOf(show tvdb.Series, order tvdb.Order, episodes, aired []tvdb.Episode) (tree, error) {
	t := tree{rec: record{Order: order, Orders: show.Orders}}
	seasons := slices.DeleteFunc(slices.Clone(show.Seasons), func(s tvdb.Season) bool { return s.Order != order })
	slices.SortFunc(seasons, func(a, b tvdb.Season) int {
		return cmp.Or(cmp.Compare(specialsLast(a), specialsLast(b)), cmp.Compare(a.Number, b.Number))
	})
	at := map[int]int{}
	for i, s := range seasons {
		key := keyOf(int64(s.ID))
		t.rec.Seasons = append(t.rec.Seasons, season{ID: db.SeasonID(s.ID), Number: int64(s.Number), Name: s.Name})
		label := "Season " + strconv.Itoa(s.Number)
		if s.Number == 0 {
			label = Description.Aside.Many
		}
		t.children = append(t.children, works.Entry{Key: key, Label: label, Aside: s.Number == 0})
		if s.Poster != "" {
			t.art = append(t.art, works.Art{Level: key, Kind: seasonPosterKind.Key, Ref: tvdbRef(seasonPosterKind.Shape, s.Poster)})
		}
		at[s.Number] = i
	}
	airedOf := make(map[tvdb.EpisodeID]tvdb.Episode, len(aired))
	for _, e := range aired {
		airedOf[e.ID] = e
	}
	for _, e := range episodes {
		i, ok := at[e.Season]
		if !ok {
			return tree{}, &works.SourceError{Source: "TVDB", Words: fmt.Sprintf("listed episode %d in season %d, which series %d doesn't have", e.ID, e.Season, show.ID), Err: works.ErrInvalid}
		}
		key := keyOf(int64(e.ID))
		label := e.Name
		if label == "" {
			label = Description.Item.One + " " + strconv.Itoa(e.Number)
		}
		t.children[i].Children = append(t.children[i].Children, works.Entry{Key: key, Label: label})
		ep := episode{ID: db.EpisodeID(e.ID), Season: int64(e.Season), Number: int64(e.Number), AirDate: e.Aired, Runtime: e.Runtime, Overview: e.Overview}
		if e.Absolute > 0 {
			ep.Absolute = ptrTo(int64(e.Absolute))
		}
		if a, ok := airedOf[e.ID]; ok {
			ep.AiredSeason, ep.AiredNumber = ptrTo(int64(a.Season)), ptrTo(int64(a.Number))
		}
		t.rec.Episodes = append(t.rec.Episodes, ep)
		if e.Still != "" {
			t.art = append(t.art, works.Art{Item: key, Kind: stillKind.Key, Ref: tvdbRef(stillKind.Shape, e.Still)})
		}
	}
	return t, nil
}

func specialsLast(s tvdb.Season) int {
	if s.Number == 0 {
		return 1
	}
	return 0
}

func showOf(s tvdb.Series, tm *tmdb.Series, region string) show {
	out := show{
		TVDBID: db.TVDBID(s.ID), IMDbID: s.IMDb, EnglishTitle: s.English, OriginalLanguage: s.OriginalLanguage, Country: s.Country,
		Network: s.Network, Status: s.Status, FirstAired: s.FirstAired, LastAired: s.LastAired, NextAired: s.NextAired,
		AverageRuntime: s.AverageRuntime, Genres: s.Genres, Aliases: s.Aliases, Overview: s.Overview,
	}
	if s.Original != s.Name {
		out.OriginalTitle = s.Original
	}
	if tm == nil {
		return out
	}
	out.TMDBID = ptrTo(db.TMDBID(tm.ID))
	out.IMDbID = cmp.Or(out.IMDbID, tm.IMDb)
	out.Creators, out.Cast = peopleIn(tm.Creators), peopleIn(tm.Cast)
	if code := tm.Certifications[region]; code != "" {
		out.Certification = &certification{Code: code, Country: region}
	}
	if v := tm.Trailer; v != nil {
		out.Trailer = &trailer{Site: v.Site, Key: v.Key}
	}
	return out
}

func peopleIn(ps []tmdb.Person) []person {
	out := make([]person, len(ps))
	for i, p := range ps {
		out[i] = person{Name: p.Name, Role: p.Role, Photo: p.Photo}
	}
	return out
}

// workArt is the series' own images: TMDB's in the Ask's language where
// it has them, else TVDB's poster.
func workArt(s tvdb.Series, tm *tmdb.Series) []works.Art {
	var out []works.Art
	if tm != nil {
		for _, a := range []struct {
			kind artwork.Kind
			path string
		}{{posterKind, tm.Poster}, {backdropKind, tm.Backdrop}, {titleTreatmentKind, tm.TitleTreatment}, {networkLogoKind, tm.NetworkLogo}} {
			if a.path != "" {
				out = append(out, works.Art{Kind: a.kind.Key, Ref: tmdbRef(a.kind.Shape, a.path)})
			}
		}
	}
	if s.Poster != "" && !slices.ContainsFunc(out, func(a works.Art) bool { return a.Kind == posterKind.Key }) {
		out = append(out, works.Art{Kind: posterKind.Key, Ref: tvdbRef(posterKind.Shape, s.Poster)})
	}
	return out
}

// seasonsBudget is how long one Fetch spends on OMDb's season ratings.
// OMDb takes a call every half second, so a long series' older seasons
// keep their stored ratings until a refresh reaches them.
const seasonsBudget = 15 * time.Second

// episodeRatings are OMDb's IMDb ratings of episodes, read a main season
// at a time in Aired numbering, the latest season first, until
// seasonsBudget is spent or OMDb stops responding. OMDb numbers an
// episode by its Aired season and number, which map it to its TVDB key.
func (s *Source) episodeRatings(ctx context.Context, ask metadata.Ask, imdb string, episodes []episode) []works.Rating {
	type place struct{ season, number int64 }
	keys := map[place]works.Key{}
	var seasons []int64
	for _, e := range episodes {
		if e.AiredSeason == nil || *e.AiredSeason == 0 {
			continue
		}
		keys[place{*e.AiredSeason, *e.AiredNumber}] = keyOf(int64(e.ID))
		if !slices.Contains(seasons, *e.AiredSeason) {
			seasons = append(seasons, *e.AiredSeason)
		}
	}
	slices.SortFunc(seasons, func(a, b int64) int { return cmp.Compare(b, a) })
	var out []works.Rating
	start := time.Now()
	for _, n := range seasons {
		if time.Since(start) >= seasonsBudget {
			break
		}
		rated, err := s.omdb.Season(ctx, ask, imdb, int(n))
		if errors.Is(err, works.ErrNotFound) {
			continue
		}
		if err != nil {
			break
		}
		for _, r := range rated {
			if key, ok := keys[place{n, int64(r.Episode)}]; ok {
				r.Rating.Item = key
				out = append(out, r.Rating)
			}
		}
	}
	return out
}

func tvdbRef(shape artwork.Shape, path string) artwork.Ref {
	return artwork.Ref{Source: tvdb.ArtworkID, Shape: shape, Path: path}
}

func tmdbRef(shape artwork.Shape, path string) artwork.Ref {
	return artwork.Ref{Source: tmdb.ImagesID, Shape: shape, Path: path}
}

// sortLabelOf moves an English title's leading article to its end, so
// "The Lantern Keepers" sorts as "Lantern Keepers, The". It is empty when
// the title sorts as it reads.
func sortLabelOf(title, language string) string {
	if !strings.HasPrefix(language, "en") {
		return ""
	}
	for _, article := range []string{"The", "An", "A"} {
		if rest, ok := strings.CutPrefix(title, article+" "); ok && rest != "" {
			return rest + ", " + article
		}
	}
	return ""
}

func ptrTo[T any](v T) *T { return &v }

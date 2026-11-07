package main

import (
	"cmp"
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	tmdbapi "github.com/jatassi/slipstream/internal/providers/tmdb"
)

// Each movie keeps its dates and alternative titles in countries, its
// translations in languages, its first maxTitles alternative titles and
// its first maxCast cast. A series keeps its content ratings in countries
// and its first maxCast cast.
var (
	countries = []string{"US", "GB", "FR", "DE"}
	languages = []string{"fr-FR", "de-DE", "es-ES"}
)

const (
	maxTitles = 5
	maxCast   = 10
)

// sizes is the TMDB size each image is downloaded at: big enough for the
// sizes the app asks for, small enough to check in.
var sizes = map[string]string{"poster": "w500", "backdrop": "w1280", "logo": "w500", "profile": "w185"}

// movie is TMDB's movie id and OMDb's ratings for it, with its images
// saved in the store.
func (c config) movie(ctx context.Context, id int64) (scenario.TMDBMovie, scenario.OMDbTitle, error) {
	var m tmdbapi.Movie
	q := url.Values{
		"api_key":                {c.tmdbKey},
		"language":               {"en-US"},
		"append_to_response":     {"external_ids,release_dates,alternative_titles,credits,videos,images,translations"},
		"include_image_language": {"en,null"},
	}
	if err := c.getJSON(ctx, c.tmdbAPI+"/movie/"+strconv.FormatInt(id, 10), q, "", &m); err != nil {
		return scenario.TMDBMovie{}, scenario.OMDbTitle{}, err
	}
	rec, err := movieRecord(m)
	if err != nil {
		return scenario.TMDBMovie{}, scenario.OMDbTitle{}, err
	}
	if err := c.download(ctx, c.tmdbFiles(rec.Images, rec.Credits)); err != nil {
		return scenario.TMDBMovie{}, scenario.OMDbTitle{}, err
	}
	title, _, err := c.omdb(ctx, rec.IMDbID)
	return rec, title, err
}

// movieRecord is the Scenario's record of TMDB's movie m.
func movieRecord(m tmdbapi.Movie) (scenario.TMDBMovie, error) {
	out := scenario.TMDBMovie{
		ID: m.ID, IMDbID: cmp.Or(m.ExternalIDs.IMDbID, m.IMDbID), Title: m.Title,
		OriginalTitle: m.OriginalTitle, OriginalLanguage: m.OriginalLanguage, Overview: m.Overview, Tagline: m.Tagline,
		Status: m.Status, Runtime: time.Duration(m.Runtime) * time.Minute, VoteAverage: m.VoteAverage, VoteCount: m.VoteCount,
		Trailer: trailer(m.Videos.Results), Dates: dates(m.ReleaseDates), Titles: alternatives(m.AlternativeTitles.Titles),
		Translations: translations(m.Translations.Translations), Credits: credits(m.Credits), Images: pickImages(m.Images),
	}
	if out.IMDbID == "" {
		return scenario.TMDBMovie{}, fmt.Errorf("TMDB movie %d has no IMDb id", m.ID)
	}
	for _, g := range m.Genres {
		out.Genres = append(out.Genres, g.Name)
	}
	if c := m.BelongsToCollection; c != nil {
		out.Collection = &scenario.TMDBNamed{ID: c.ID, Name: c.Name}
	}
	if len(m.ProductionCompanies) > 0 {
		out.Studio = &scenario.TMDBNamed{ID: m.ProductionCompanies[0].ID, Name: m.ProductionCompanies[0].Name}
	}
	return out, nil
}

// pickImages is the best-voted English poster, textless backdrop and
// English logo.
func pickImages(ims tmdbapi.Images) []scenario.TMDBImage {
	var out []scenario.TMDBImage
	for _, pick := range []struct {
		shape, language string
		from            []tmdbapi.Image
	}{{"poster", "en", ims.Posters}, {"backdrop", "", ims.Backdrops}, {"logo", "en", ims.Logos}} {
		if im, ok := best(pick.from, pick.language); ok {
			out = append(out, scenario.TMDBImage{
				Shape: pick.shape, Language: pick.language, Votes: im.VoteAverage, Path: im.FilePath, Width: im.Width, Height: im.Height,
			})
		}
	}
	return out
}

// best is the best-voted image in language, "" for no text.
func best(ims []tmdbapi.Image, language string) (tmdbapi.Image, bool) {
	ims = slices.DeleteFunc(slices.Clone(ims), func(im tmdbapi.Image) bool {
		return im.Language != language || !scenario.ArtworkFile.MatchString(im.FilePath)
	})
	if len(ims) == 0 {
		return tmdbapi.Image{}, false
	}
	return slices.MinFunc(ims, func(a, b tmdbapi.Image) int {
		return cmp.Or(cmp.Compare(b.VoteAverage, a.VoteAverage), cmp.Compare(b.VoteCount, a.VoteCount), cmp.Compare(a.FilePath, b.FilePath))
	}), true
}

// trailer is the key of the first official YouTube trailer.
func trailer(videos []tmdbapi.Video) string {
	i := slices.IndexFunc(videos, func(v tmdbapi.Video) bool { return v.Site == "YouTube" && v.Type == "Trailer" && v.Official })
	if i < 0 {
		return ""
	}
	return videos[i].Key
}

func dates(r tmdbapi.ReleaseDates) []scenario.TMDBDate {
	var out []scenario.TMDBDate
	for _, country := range countries {
		i := slices.IndexFunc(r.Results, func(c tmdbapi.CountryReleases) bool { return c.Country == country })
		if i < 0 {
			continue
		}
		for _, rel := range r.Results[i].Releases {
			if rel.Type < tmdbapi.Premiere || rel.Type > tmdbapi.TV || len(rel.Date) < len(time.DateOnly) {
				continue
			}
			out = append(out, scenario.TMDBDate{
				Country: country, Kind: scenario.TMDBDateKinds[rel.Type-1], Date: rel.Date[:len(time.DateOnly)],
				Certification: rel.Certification, Note: rel.Note,
			})
		}
	}
	return out
}

func alternatives(titles []tmdbapi.AlternativeTitle) []scenario.TMDBTitle {
	var out []scenario.TMDBTitle
	for _, t := range titles {
		if len(out) < maxTitles && slices.Contains(countries, t.Country) && t.Title != "" {
			out = append(out, scenario.TMDBTitle{Country: t.Country, Title: t.Title, Kind: t.Type})
		}
	}
	return out
}

func translations(ts []tmdbapi.Translation) []scenario.TMDBTranslation {
	var out []scenario.TMDBTranslation
	for _, language := range languages {
		i := slices.IndexFunc(ts, func(t tmdbapi.Translation) bool { return t.Language+"-"+t.Country == language })
		if i < 0 || ts[i].Data == (tmdbapi.TranslationData{}) {
			continue
		}
		d := ts[i].Data
		out = append(out, scenario.TMDBTranslation{Language: language, Title: d.Title, Overview: d.Overview, Tagline: d.Tagline})
	}
	return out
}

// credits is the top-billed cast, the directors and the writers.
func credits(c tmdbapi.Credits) scenario.TMDBCredits {
	out := scenario.TMDBCredits{Cast: cast(c.Cast)}
	for _, p := range c.Crew {
		kept := slices.ContainsFunc(out.Crew, func(o scenario.TMDBCrew) bool { return o.ID == p.ID && o.Job == p.Job })
		if !kept && (p.Job == "Director" || p.Department == "Writing") {
			out.Crew = append(out.Crew, scenario.TMDBCrew{ID: p.ID, Name: p.Name, Department: p.Department, Job: p.Job, Profile: stored(p.ProfilePath)})
		}
	}
	return out
}

func cast(c []tmdbapi.Cast) []scenario.TMDBCast {
	var out []scenario.TMDBCast
	billed := slices.SortedStableFunc(slices.Values(c), func(a, b tmdbapi.Cast) int { return cmp.Compare(a.Order, b.Order) })
	for _, p := range billed[:min(len(billed), maxCast)] {
		out = append(out, scenario.TMDBCast{ID: p.ID, Name: p.Name, Character: p.Character, Profile: stored(p.ProfilePath)})
	}
	return out
}

// stored is TMDB file path p when the store can hold it, else empty.
func stored(p string) string {
	if scenario.ArtworkFile.MatchString(p) {
		return p
	}
	return ""
}

// tmdbFile is the TMDB file at p, downloaded at shape's size.
func (c config) tmdbFile(p, shape string) file {
	size := sizes[shape]
	if strings.HasSuffix(p, ".svg") {
		size = "original"
	}
	return file{path: p, from: c.tmdbImages + "/" + size + p}
}

// tmdbFiles are the images and photos a TMDB record names.
func (c config) tmdbFiles(ims []scenario.TMDBImage, cr scenario.TMDBCredits) []file {
	var out []file
	for _, im := range ims {
		out = append(out, c.tmdbFile(im.Path, im.Shape))
	}
	for _, p := range cr.Cast {
		out = append(out, c.tmdbFile(p.Profile, "profile"))
	}
	for _, p := range cr.Crew {
		out = append(out, c.tmdbFile(p.Profile, "profile"))
	}
	return out
}

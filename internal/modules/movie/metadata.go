package movie

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/module"
)

var ErrNoMetadataProvider = errors.New("no metadata provider configured")

type metadataProvider struct {
	metadataSvc *metadata.Service
	movieSvc    *movies.Service
	logger      zerolog.Logger
}

func newMetadataProvider(metadataSvc *metadata.Service, movieSvc *movies.Service, logger *zerolog.Logger) *metadataProvider {
	return &metadataProvider{
		metadataSvc: metadataSvc,
		movieSvc:    movieSvc,
		logger:      logger.With().Str("component", "movie-metadata").Logger(),
	}
}

func (p *metadataProvider) Search(ctx context.Context, query string, opts module.SearchOptions) ([]module.SearchResult, error) {
	results, err := p.metadataSvc.SearchMovies(ctx, query, opts.Year)
	if err != nil {
		return nil, err
	}

	out := make([]module.SearchResult, len(results))
	for i := range results {
		r := &results[i]
		out[i] = module.SearchResult{
			ExternalID:  strconv.Itoa(r.ID),
			Title:       r.Title,
			Year:        r.Year,
			Overview:    r.Overview,
			PosterURL:   r.PosterURL,
			BackdropURL: r.BackdropURL,
			ExternalIDs: map[string]string{
				"tmdb": strconv.Itoa(r.ID),
				"imdb": r.ImdbID,
			},
			Extra: &r,
		}
	}
	return out, nil
}

func (p *metadataProvider) GetByID(ctx context.Context, externalID string) (*module.MediaMetadata, error) {
	tmdbID, err := strconv.Atoi(externalID)
	if err != nil {
		return nil, fmt.Errorf("invalid TMDB ID: %w", err)
	}

	result, err := p.metadataSvc.GetMovie(ctx, tmdbID)
	if err != nil {
		return nil, err
	}

	return &module.MediaMetadata{
		ExternalID:  externalID,
		Title:       result.Title,
		Year:        result.Year,
		Overview:    result.Overview,
		PosterURL:   result.PosterURL,
		BackdropURL: result.BackdropURL,
		ExternalIDs: map[string]string{
			"tmdb": strconv.Itoa(result.ID),
			"imdb": result.ImdbID,
		},
		Extra: result,
	}, nil
}

func (p *metadataProvider) GetExtendedInfo(ctx context.Context, externalID string) (*module.ExtendedMetadata, error) {
	tmdbID, err := strconv.Atoi(externalID)
	if err != nil {
		return nil, fmt.Errorf("invalid TMDB ID: %w", err)
	}

	result, err := p.metadataSvc.GetExtendedMovie(ctx, tmdbID)
	if err != nil {
		return nil, err
	}

	return &module.ExtendedMetadata{
		Credits:       result.Credits,
		Ratings:       result.Ratings,
		ContentRating: result.ContentRating,
		TrailerURL:    result.TrailerURL,
		Extra:         result,
	}, nil
}

func (p *metadataProvider) RefreshMetadata(ctx context.Context, entityID int64) (*module.RefreshResult, error) {
	movie, err := p.movieSvc.Get(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	if !p.metadataSvc.HasMovieProvider() {
		return nil, ErrNoMetadataProvider
	}

	results, err := p.metadataSvc.SearchMovies(ctx, movie.Title, movie.Year)
	if err != nil {
		return nil, fmt.Errorf("metadata search failed: %w", err)
	}

	if len(results) == 0 {
		p.logger.Warn().Str("title", movie.Title).Int("year", movie.Year).Msg("No metadata results found")
		return &module.RefreshResult{EntityID: entityID}, nil
	}

	bestMatch := p.selectBestMatch(movie, results)
	p.enrichDetails(ctx, bestMatch)

	fieldsChanged := p.computeFieldsChanged(movie, bestMatch)
	if err := p.updateMovieFromMetadata(ctx, movie.ID, bestMatch); err != nil {
		return nil, err
	}

	return &module.RefreshResult{
		EntityID:      entityID,
		Updated:       len(fieldsChanged) > 0,
		FieldsChanged: fieldsChanged,
		ArtworkURLs: module.ArtworkURLs{
			PosterURL:     bestMatch.PosterURL,
			BackdropURL:   bestMatch.BackdropURL,
			LogoURL:       bestMatch.LogoURL,
			StudioLogoURL: bestMatch.StudioLogoURL,
		},
		Metadata: bestMatch,
	}, nil
}

func (p *metadataProvider) selectBestMatch(movie *movies.Movie, results []metadata.MovieResult) *metadata.MovieResult {
	movieTitleLower := strings.ToLower(movie.Title)

	for i := range results {
		if results[i].Year == movie.Year && strings.EqualFold(results[i].Title, movieTitleLower) {
			return &results[i]
		}
	}

	for i := range results {
		if results[i].Year == movie.Year && strings.HasPrefix(strings.ToLower(results[i].Title), movieTitleLower) {
			return &results[i]
		}
	}

	for i := range results {
		if results[i].Year == movie.Year {
			return &results[i]
		}
	}

	return &results[0]
}

func (p *metadataProvider) enrichDetails(ctx context.Context, match *metadata.MovieResult) {
	if match.ID > 0 {
		if details, err := p.metadataSvc.GetMovie(ctx, match.ID); err == nil {
			*match = *details
		}
	}

	if match.ID > 0 {
		if logoURL, err := p.metadataSvc.GetMovieLogoURL(ctx, match.ID); err == nil && logoURL != "" {
			match.LogoURL = logoURL
		}
	}
}

func (p *metadataProvider) computeFieldsChanged(movie *movies.Movie, match *metadata.MovieResult) []string {
	var changed []string
	if movie.Title != match.Title {
		changed = append(changed, "title")
	}
	if movie.Year != match.Year {
		changed = append(changed, "year")
	}
	if movie.TmdbID != match.ID {
		changed = append(changed, "tmdbId")
	}
	if movie.ImdbID != match.ImdbID {
		changed = append(changed, "imdbId")
	}
	if movie.Overview != match.Overview {
		changed = append(changed, "overview")
	}
	if movie.Runtime != match.Runtime {
		changed = append(changed, "runtime")
	}
	if movie.Studio != match.Studio {
		changed = append(changed, "studio")
	}
	return changed
}

func (p *metadataProvider) updateMovieFromMetadata(ctx context.Context, movieID int64, match *metadata.MovieResult) error {
	title := match.Title
	year := match.Year
	tmdbID := match.ID
	imdbID := match.ImdbID
	overview := match.Overview
	runtime := match.Runtime
	studio := match.Studio

	var releaseDate, physicalReleaseDate, theatricalReleaseDate, contentRating string
	if tmdbID > 0 {
		digital, physical, theatrical, err := p.metadataSvc.GetMovieReleaseDates(ctx, tmdbID)
		if err != nil {
			p.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to fetch release dates")
		} else {
			releaseDate = digital
			physicalReleaseDate = physical
			theatricalReleaseDate = theatrical
		}

		if cr, err := p.metadataSvc.GetMovieContentRating(ctx, tmdbID); err == nil && cr != "" {
			contentRating = cr
		}
	}

	_, err := p.movieSvc.Update(ctx, movieID, &movies.UpdateMovieInput{
		Title:                 &title,
		Year:                  &year,
		TmdbID:                &tmdbID,
		ImdbID:                &imdbID,
		Overview:              &overview,
		Runtime:               &runtime,
		Studio:                &studio,
		ReleaseDate:           &releaseDate,
		PhysicalReleaseDate:   &physicalReleaseDate,
		TheatricalReleaseDate: &theatricalReleaseDate,
		ContentRating:         &contentRating,
	})
	return err
}

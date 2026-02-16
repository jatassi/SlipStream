package librarymanager

import (
	"context"
	"fmt"
	"strings"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/progress"
)

// matchOrCreateMovie finds an existing movie or creates a new one from parsed media.
// Returns the movie, whether it was created, the metadata used (if any), and any error.
func (s *Service) matchOrCreateMovie(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsed *scanner.ParsedMedia,
	qualityProfileID int64,
) (*movies.Movie, bool, *metadata.MovieResult, error) {
	if !s.metadata.HasMovieProvider() {
		movie, created, err := s.createMovieFromParsed(ctx, folder, parsed, qualityProfileID, nil)
		return movie, created, nil, err
	}

	results, err := s.metadata.SearchMovies(ctx, parsed.Title, parsed.Year)
	if err != nil {
		s.logger.Warn().Err(err).Str("title", parsed.Title).Int("year", parsed.Year).Msg("Metadata search failed, creating movie without metadata")
		movie, created, err := s.createMovieFromParsed(ctx, folder, parsed, qualityProfileID, nil)
		return movie, created, nil, err
	}

	bestMatch := s.findBestMovieMatch(ctx, results, parsed)
	movie, created, err := s.createMovieFromParsed(ctx, folder, parsed, qualityProfileID, bestMatch)
	return movie, created, bestMatch, err
}

func (s *Service) findBestMovieMatch(
	ctx context.Context,
	results []metadata.MovieResult,
	parsed *scanner.ParsedMedia,
) *metadata.MovieResult {
	if len(results) == 0 {
		return nil
	}

	parsedTitleNorm := normalizeTitle(parsed.Title)

	if match := s.findExactTitleYearMatch(results, parsed.Year, parsedTitleNorm); match != nil {
		return s.checkExistingMovie(ctx, match)
	}

	if match := s.findPrefixMatch(results, parsed.Year, parsedTitleNorm); match != nil {
		return s.checkExistingMovie(ctx, match)
	}

	if match := s.findSimilarTitleMatch(results, parsed.Year, parsedTitleNorm); match != nil {
		return s.checkExistingMovie(ctx, match)
	}

	if match := s.findYearMatch(results, parsed.Year); match != nil {
		return s.checkExistingMovie(ctx, match)
	}

	return s.checkExistingMovie(ctx, &results[0])
}

func (s *Service) findExactTitleYearMatch(
	results []metadata.MovieResult,
	year int,
	parsedTitleNorm string,
) *metadata.MovieResult {
	for i := range results {
		if results[i].Year != year {
			continue
		}
		if normalizeTitle(results[i].Title) != parsedTitleNorm {
			continue
		}
		s.logger.Debug().Str("title", results[i].Title).Int("year", results[i].Year).Msg("Found exact normalized title and year match")
		return &results[i]
	}
	return nil
}

func (s *Service) findPrefixMatch(
	results []metadata.MovieResult,
	year int,
	parsedTitleNorm string,
) *metadata.MovieResult {
	for i := range results {
		if results[i].Year != year {
			continue
		}
		resultTitleNorm := normalizeTitle(results[i].Title)
		if !strings.HasPrefix(resultTitleNorm, parsedTitleNorm) {
			continue
		}
		if len(resultTitleNorm) > len(parsedTitleNorm)+5 {
			continue
		}
		s.logger.Debug().Str("title", results[i].Title).Int("year", results[i].Year).Msg("Found title prefix and year match")
		return &results[i]
	}
	return nil
}

func (s *Service) findSimilarTitleMatch(
	results []metadata.MovieResult,
	year int,
	parsedTitleNorm string,
) *metadata.MovieResult {
	for i := range results {
		if results[i].Year != year {
			continue
		}
		resultTitleNorm := normalizeTitle(results[i].Title)
		if !strings.Contains(resultTitleNorm, parsedTitleNorm) && !strings.Contains(parsedTitleNorm, resultTitleNorm) {
			continue
		}
		if len(resultTitleNorm) > len(parsedTitleNorm)+10 || len(parsedTitleNorm) > len(resultTitleNorm)+10 {
			continue
		}
		return &results[i]
	}
	return nil
}

func (s *Service) findYearMatch(results []metadata.MovieResult, year int) *metadata.MovieResult {
	for i := range results {
		if results[i].Year == year {
			return &results[i]
		}
	}
	return nil
}

func (s *Service) checkExistingMovie(ctx context.Context, match *metadata.MovieResult) *metadata.MovieResult {
	if match == nil {
		return nil
	}
	existing, err := s.movies.GetByTmdbID(ctx, match.ID)
	if err == nil && existing != nil {
		return nil
	}
	return match
}

// matchOrCreateSeries finds an existing series or creates a new one from parsed media.
// Returns the series, whether it was created, the metadata used (if any), and any error.
func (s *Service) matchOrCreateSeries(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	parsed *scanner.ParsedMedia,
	qualityProfileID int64,
) (*tv.Series, bool, *metadata.SeriesResult, error) {
	if !s.metadata.HasSeriesProvider() {
		series, created, err := s.createSeriesFromParsed(ctx, folder, parsed, qualityProfileID, nil)
		return series, created, nil, err
	}

	results, err := s.metadata.SearchSeries(ctx, parsed.Title)
	if err != nil {
		s.logger.Warn().Err(err).Str("title", parsed.Title).Msg("Metadata search failed, creating series without metadata")
		series, created, err := s.createSeriesFromParsed(ctx, folder, parsed, qualityProfileID, nil)
		return series, created, nil, err
	}

	bestMatch := s.findBestSeriesMatch(ctx, results)
	series, created, err := s.createSeriesFromParsed(ctx, folder, parsed, qualityProfileID, bestMatch)
	return series, created, bestMatch, err
}

func (s *Service) findBestSeriesMatch(ctx context.Context, results []metadata.SeriesResult) *metadata.SeriesResult {
	if len(results) == 0 {
		return nil
	}

	bestMatch := &results[0]

	if existingSeries := s.checkExistingSeries(ctx, bestMatch.TvdbID); existingSeries != nil {
		return nil
	}

	if bestMatch.Status == "" {
		bestMatch = s.enrichSeriesStatus(ctx, bestMatch)
	}

	return bestMatch
}

func (s *Service) checkExistingSeries(ctx context.Context, tvdbID int) *tv.Series {
	if tvdbID == 0 {
		return nil
	}
	existing, err := s.tv.GetSeriesByTvdbID(ctx, tvdbID)
	if err == nil && existing != nil {
		return existing
	}
	return nil
}

func (s *Service) enrichSeriesStatus(ctx context.Context, bestMatch *metadata.SeriesResult) *metadata.SeriesResult {
	if bestMatch.TmdbID > 0 {
		if fullDetails, err := s.metadata.GetSeriesByTMDB(ctx, bestMatch.TmdbID); err == nil {
			return fullDetails
		}
	} else if bestMatch.TvdbID > 0 {
		if fullDetails, err := s.metadata.GetSeriesByTVDB(ctx, bestMatch.TvdbID); err == nil {
			return fullDetails
		}
	}
	return bestMatch
}

// matchUnmatchedMovies finds movies without metadata and attempts to match them.
func (s *Service) matchUnmatchedMovies(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	result *ScanResult,
	activity *progress.ActivityBuilder,
	pending *pendingArtwork,
) {
	if !s.metadata.HasMovieProvider() {
		return
	}

	unmatched, err := s.movies.ListUnmatchedByRootFolder(ctx, folder.ID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("rootFolderId", folder.ID).Msg("Failed to list unmatched movies")
		return
	}

	if len(unmatched) == 0 {
		return
	}

	s.logger.Info().Int("count", len(unmatched)).Msg("Attempting to match unmatched movies")

	if activity != nil {
		activity.Update("Matching unmatched movies...", -1)
		activity.SetMetadata("unmatchedMovies", len(unmatched))
	}

	for i, movie := range unmatched {
		s.updateMatchingProgress(activity, i, len(unmatched), movie.Title)
		s.matchSingleUnmatchedMovie(ctx, movie, result, pending)
	}
}

func (s *Service) updateMatchingProgress(activity *progress.ActivityBuilder, index, total int, title string) {
	if activity == nil {
		return
	}
	pct := (index + 1) * 100 / total
	activity.Update(fmt.Sprintf("Matching: %s", title), pct)
}

func (s *Service) matchSingleUnmatchedMovie(ctx context.Context, movie *movies.Movie, result *ScanResult, pending *pendingArtwork) {
	results, err := s.metadata.SearchMovies(ctx, movie.Title, movie.Year)
	if err != nil {
		s.logger.Warn().Err(err).Str("title", movie.Title).Int("year", movie.Year).Msg("Metadata search failed for unmatched movie")
		return
	}

	if len(results) == 0 {
		return
	}

	bestMatch := s.selectBestUnmatchedMovieResult(results, movie.Title, movie.Year)
	updateInput := s.buildMovieUpdateInput(ctx, bestMatch)

	if _, err := s.movies.Update(ctx, movie.ID, &updateInput); err != nil {
		s.logger.Warn().Err(err).Int64("movieId", movie.ID).Msg("Failed to update movie with metadata")
		return
	}

	result.MetadataMatched++

	if bestMatch.PosterURL != "" || bestMatch.BackdropURL != "" {
		pending.movieMeta = append(pending.movieMeta, bestMatch)
	}

	s.logger.Info().
		Int64("movieId", movie.ID).
		Str("title", bestMatch.Title).
		Int("tmdbId", bestMatch.ID).
		Msg("Matched unmatched movie")
}

func (s *Service) selectBestUnmatchedMovieResult(results []metadata.MovieResult, title string, year int) *metadata.MovieResult {
	movieTitleLower := strings.ToLower(title)

	for i := range results {
		if results[i].Year == year && strings.EqualFold(results[i].Title, movieTitleLower) {
			return &results[i]
		}
	}

	for i := range results {
		if results[i].Year == year && strings.HasPrefix(strings.ToLower(results[i].Title), movieTitleLower) {
			return &results[i]
		}
	}

	for i := range results {
		if results[i].Year == year {
			return &results[i]
		}
	}

	return &results[0]
}

func (s *Service) buildMovieUpdateInput(ctx context.Context, bestMatch *metadata.MovieResult) movies.UpdateMovieInput {
	title := bestMatch.Title
	year := bestMatch.Year
	tmdbID := bestMatch.ID
	imdbID := bestMatch.ImdbID
	overview := bestMatch.Overview
	runtime := bestMatch.Runtime

	updateInput := movies.UpdateMovieInput{
		Title:    &title,
		Year:     &year,
		TmdbID:   &tmdbID,
		ImdbID:   &imdbID,
		Overview: &overview,
		Runtime:  &runtime,
	}

	if tmdbID > 0 {
		digital, physical, theatrical, err := s.metadata.GetMovieReleaseDates(ctx, tmdbID)
		if err != nil {
			s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to fetch release dates for unmatched movie")
		} else {
			updateInput.ReleaseDate = &digital
			updateInput.PhysicalReleaseDate = &physical
			updateInput.TheatricalReleaseDate = &theatrical
		}
	}

	return updateInput
}

// matchUnmatchedSeries finds series without metadata and attempts to match them.
func (s *Service) matchUnmatchedSeries(
	ctx context.Context,
	folder *rootfolder.RootFolder,
	result *ScanResult,
	activity *progress.ActivityBuilder,
	pending *pendingArtwork,
) {
	if !s.metadata.HasSeriesProvider() {
		return
	}

	unmatched, err := s.tv.ListUnmatchedByRootFolder(ctx, folder.ID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("rootFolderId", folder.ID).Msg("Failed to list unmatched series")
		return
	}

	if len(unmatched) == 0 {
		return
	}

	s.logger.Info().Int("count", len(unmatched)).Msg("Attempting to match unmatched series")

	if activity != nil {
		activity.Update("Matching unmatched series...", -1)
		activity.SetMetadata("unmatchedSeries", len(unmatched))
	}

	for i, series := range unmatched {
		s.updateMatchingProgress(activity, i, len(unmatched), series.Title)
		s.matchSingleUnmatchedSeries(ctx, series, result, pending)
	}
}

func (s *Service) matchSingleUnmatchedSeries(ctx context.Context, series *tv.Series, result *ScanResult, pending *pendingArtwork) {
	results, err := s.metadata.SearchSeries(ctx, series.Title)
	if err != nil {
		s.logger.Warn().Err(err).Str("title", series.Title).Msg("Metadata search failed for unmatched series")
		return
	}

	if len(results) == 0 {
		return
	}

	bestMatch := &results[0]

	title := bestMatch.Title
	year := bestMatch.Year
	tvdbID := bestMatch.TvdbID
	tmdbID := bestMatch.TmdbID
	imdbID := bestMatch.ImdbID
	overview := bestMatch.Overview
	runtime := bestMatch.Runtime

	_, err = s.tv.UpdateSeries(ctx, series.ID, &tv.UpdateSeriesInput{
		Title:    &title,
		Year:     &year,
		TvdbID:   &tvdbID,
		TmdbID:   &tmdbID,
		ImdbID:   &imdbID,
		Overview: &overview,
		Runtime:  &runtime,
	})
	if err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", series.ID).Msg("Failed to update series with metadata")
		return
	}

	result.MetadataMatched++

	if bestMatch.PosterURL != "" || bestMatch.BackdropURL != "" {
		pending.seriesMeta = append(pending.seriesMeta, bestMatch)
	}

	s.logger.Info().
		Int64("seriesId", series.ID).
		Str("title", bestMatch.Title).
		Int("tvdbId", bestMatch.TvdbID).
		Msg("Matched unmatched series")
}

package librarymanager

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/progress"
)

func (s *Service) downloadPendingArtwork(
	ctx context.Context,
	pending *pendingArtwork,
	result *ScanResult,
	activity *progress.ActivityBuilder,
) {
	totalItems := len(pending.movieMeta) + len(pending.seriesMeta)
	if totalItems == 0 {
		return
	}

	s.logger.Info().
		Int("movies", len(pending.movieMeta)).
		Int("series", len(pending.seriesMeta)).
		Msg("Downloading artwork for newly added items")

	if activity != nil {
		activity.Update("Downloading artwork...", -1)
		activity.SetMetadata("artworkTotal", totalItems)
	}

	downloaded := 0

	// Download movie artwork
	for i, movie := range pending.movieMeta {
		if activity != nil {
			pct := (i + 1) * 100 / totalItems
			activity.Update(fmt.Sprintf("Downloading artwork: %s", movie.Title), pct)
		}

		if err := s.artwork.DownloadMovieArtwork(ctx, movie); err != nil {
			s.logger.Warn().Err(err).
				Int("tmdbId", movie.ID).
				Str("title", movie.Title).
				Msg("Failed to download movie artwork")
		} else {
			downloaded++
		}
	}

	// Download series artwork
	for i, series := range pending.seriesMeta {
		if activity != nil {
			pct := (len(pending.movieMeta) + i + 1) * 100 / totalItems
			activity.Update(fmt.Sprintf("Downloading artwork: %s", series.Title), pct)
		}

		if err := s.artwork.DownloadSeriesArtwork(ctx, series); err != nil {
			s.logger.Warn().Err(err).
				Int("tvdbId", series.TvdbID).
				Str("title", series.Title).
				Msg("Failed to download series artwork")
		} else {
			downloaded++
		}
	}

	result.ArtworksFetched = downloaded

	if activity != nil {
		activity.SetMetadata("artworkDownloaded", downloaded)
	}

	s.logger.Info().
		Int("downloaded", downloaded).
		Int("total", totalItems).
		Msg("Artwork download complete")
}

// RefreshMovieMetadata fetches metadata for a single movie and downloads artwork.
func (s *Service) RefreshMovieMetadata(ctx context.Context, movieID int64) (*movies.Movie, error) {
	s.logger.Debug().Int64("movieId", movieID).Msg("[REFRESH] Starting movie metadata refresh")

	movie, err := s.movies.Get(ctx, movieID)
	if err != nil {
		s.logger.Error().Err(err).Int64("movieId", movieID).Msg("[REFRESH] Failed to get movie from database")
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	s.logger.Debug().
		Int64("movieId", movieID).
		Str("title", movie.Title).
		Int("year", movie.Year).
		Int("currentTmdbId", movie.TmdbID).
		Msg("[REFRESH] Retrieved movie from database")

	if !s.metadata.HasMovieProvider() {
		s.logger.Warn().Msg("[REFRESH] No metadata provider configured")
		return nil, ErrNoMetadataProvider
	}

	results, err := s.searchMovieMetadata(ctx, movie)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		s.logger.Warn().Str("title", movie.Title).Int("year", movie.Year).Msg("[REFRESH] No metadata results found")
		return movie, nil
	}

	s.logSearchResults(results)
	bestMatch := s.selectBestMovieMatch(movie, results)
	s.enrichMovieDetails(ctx, bestMatch)

	updatedMovie, err := s.updateMovieWithMetadata(ctx, movie.ID, bestMatch)
	if err != nil {
		return nil, err
	}

	s.downloadMovieArtworkAsync(bestMatch)
	s.scanMovieFolder(ctx, updatedMovie)

	s.logger.Info().
		Int64("movieId", movie.ID).
		Str("title", bestMatch.Title).
		Int("tmdbId", bestMatch.ID).
		Msg("[REFRESH] Movie metadata refresh completed")

	return updatedMovie, nil
}

func (s *Service) searchMovieMetadata(ctx context.Context, movie *movies.Movie) ([]metadata.MovieResult, error) {
	s.logger.Debug().Str("title", movie.Title).Int("year", movie.Year).Msg("[REFRESH] Searching for metadata")

	results, err := s.metadata.SearchMovies(ctx, movie.Title, movie.Year)
	if err != nil {
		s.logger.Error().Err(err).Str("title", movie.Title).Int("year", movie.Year).Msg("[REFRESH] Metadata search failed")
		return nil, fmt.Errorf("metadata search failed: %w", err)
	}

	s.logger.Debug().Int("resultCount", len(results)).Msg("[REFRESH] Metadata search completed")
	return results, nil
}

func (s *Service) logSearchResults(results []metadata.MovieResult) {
	for i := range results {
		r := &results[i]
		s.logger.Debug().
			Int("index", i).
			Int("tmdbId", r.ID).
			Str("title", r.Title).
			Int("year", r.Year).
			Str("imdbId", r.ImdbID).
			Str("posterUrl", r.PosterURL).
			Msg("[REFRESH] Search result")
	}
}

func (s *Service) selectBestMovieMatch(movie *movies.Movie, results []metadata.MovieResult) *metadata.MovieResult {
	movieTitleLower := strings.ToLower(movie.Title)

	for i := range results {
		if results[i].Year == movie.Year && strings.EqualFold(results[i].Title, movieTitleLower) {
			s.logger.Debug().Int("index", i).Msg("[REFRESH] Found exact title and year match")
			return &results[i]
		}
	}

	for i := range results {
		if results[i].Year == movie.Year && strings.HasPrefix(strings.ToLower(results[i].Title), movieTitleLower) {
			s.logger.Debug().Int("index", i).Msg("[REFRESH] Found title prefix and year match")
			return &results[i]
		}
	}

	for i := range results {
		if results[i].Year == movie.Year {
			s.logger.Debug().Int("index", i).Msg("[REFRESH] Found year match")
			return &results[i]
		}
	}

	s.logger.Debug().Msg("[REFRESH] No year match, using first result")
	return &results[0]
}

func (s *Service) enrichMovieDetails(ctx context.Context, bestMatch *metadata.MovieResult) {
	if bestMatch.ID > 0 {
		if details, err := s.metadata.GetMovie(ctx, bestMatch.ID); err == nil {
			*bestMatch = *details
		} else {
			s.logger.Warn().Err(err).Int("tmdbId", bestMatch.ID).Msg("[REFRESH] Failed to fetch full movie details")
		}
	}

	if bestMatch.ID > 0 {
		if logoURL, err := s.metadata.GetMovieLogoURL(ctx, bestMatch.ID); err == nil && logoURL != "" {
			bestMatch.LogoURL = logoURL
		}
	}
}

func (s *Service) updateMovieWithMetadata(ctx context.Context, movieID int64, bestMatch *metadata.MovieResult) (*movies.Movie, error) {
	title := bestMatch.Title
	year := bestMatch.Year
	tmdbID := bestMatch.ID
	imdbID := bestMatch.ImdbID
	overview := bestMatch.Overview
	runtime := bestMatch.Runtime
	studio := bestMatch.Studio

	var releaseDate, physicalReleaseDate, theatricalReleaseDate, contentRating string
	if tmdbID > 0 {
		digital, physical, theatrical, err := s.metadata.GetMovieReleaseDates(ctx, tmdbID)
		if err != nil {
			s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("[REFRESH] Failed to fetch release dates")
		} else {
			releaseDate = digital
			physicalReleaseDate = physical
			theatricalReleaseDate = theatrical
		}

		if cr, err := s.metadata.GetMovieContentRating(ctx, tmdbID); err == nil && cr != "" {
			contentRating = cr
		}
	}

	updateInput := movies.UpdateMovieInput{
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
	}

	updatedMovie, err := s.movies.Update(ctx, movieID, &updateInput)
	if err != nil {
		s.logger.Error().Err(err).Int64("movieId", movieID).Msg("[REFRESH] Failed to update movie in database")
		return nil, fmt.Errorf("failed to update movie: %w", err)
	}

	return updatedMovie, nil
}

func (s *Service) downloadMovieArtworkAsync(bestMatch *metadata.MovieResult) {
	if s.artwork == nil || (bestMatch.PosterURL == "" && bestMatch.BackdropURL == "" && bestMatch.LogoURL == "" && bestMatch.StudioLogoURL == "") {
		return
	}

	go func() {
		if err := s.artwork.DownloadMovieArtwork(context.Background(), bestMatch); err != nil {
			s.logger.Warn().Err(err).Int("tmdbId", bestMatch.ID).Msg("[REFRESH] Failed to download movie artwork")
		} else {
			s.logger.Info().Int("tmdbId", bestMatch.ID).Msg("[REFRESH] Artwork download completed")
		}
	}()
}

// RefreshSeriesMetadata fetches metadata for a single series and downloads artwork.
func (s *Service) RefreshSeriesMetadata(ctx context.Context, seriesID int64) (*tv.Series, error) {
	series, err := s.tv.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	if !s.metadata.HasSeriesProvider() {
		return nil, ErrNoMetadataProvider
	}

	results, err := s.metadata.SearchSeries(ctx, series.Title)
	if err != nil {
		return nil, fmt.Errorf("metadata search failed: %w", err)
	}

	if len(results) == 0 {
		return series, nil
	}

	bestMatch := s.enrichSeriesDetails(ctx, &results[0])

	if err := s.updateSeriesWithMetadata(ctx, series.ID, bestMatch); err != nil {
		return nil, err
	}

	s.updateSeriesSeasons(ctx, seriesID, bestMatch.TmdbID, bestMatch.TvdbID)
	s.downloadSeriesArtworkFromResult(bestMatch)

	s.logger.Info().
		Int64("seriesId", series.ID).
		Str("title", bestMatch.Title).
		Int("tvdbId", bestMatch.TvdbID).
		Msg("Refreshed series metadata")

	refreshedSeries, err := s.tv.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}

	s.scanSeriesFolder(ctx, refreshedSeries)
	return refreshedSeries, nil
}

func (s *Service) enrichSeriesDetails(ctx context.Context, bestMatch *metadata.SeriesResult) *metadata.SeriesResult {
	if bestMatch.TmdbID > 0 {
		if detail, err := s.metadata.GetSeriesByTMDB(ctx, bestMatch.TmdbID); err == nil {
			bestMatch = detail
		}
	}

	if bestMatch.TmdbID > 0 {
		if logoURL, err := s.metadata.GetSeriesLogoURL(ctx, bestMatch.TmdbID); err == nil && logoURL != "" {
			bestMatch.LogoURL = logoURL
		}
	}

	return bestMatch
}

func (s *Service) updateSeriesWithMetadata(ctx context.Context, seriesID int64, bestMatch *metadata.SeriesResult) error {
	title := bestMatch.Title
	year := bestMatch.Year
	tvdbID := bestMatch.TvdbID
	tmdbID := bestMatch.TmdbID
	imdbID := bestMatch.ImdbID
	overview := bestMatch.Overview
	runtime := bestMatch.Runtime
	status := bestMatch.Status
	network := bestMatch.Network
	networkLogoURL := bestMatch.NetworkLogoURL

	_, err := s.tv.UpdateSeries(ctx, seriesID, &tv.UpdateSeriesInput{
		Title:            &title,
		Year:             &year,
		TvdbID:           &tvdbID,
		TmdbID:           &tmdbID,
		ImdbID:           &imdbID,
		Overview:         &overview,
		Runtime:          &runtime,
		ProductionStatus: &status,
		Network:          &network,
		NetworkLogoURL:   &networkLogoURL,
	})
	return err
}

func (s *Service) updateSeriesSeasons(ctx context.Context, seriesID int64, tmdbID, tvdbID int) {
	if tmdbID == 0 && tvdbID == 0 {
		return
	}

	seasonResults, err := s.metadata.GetSeriesSeasons(ctx, tmdbID, tvdbID)
	if err != nil {
		s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Int("tvdbId", tvdbID).Msg("Failed to fetch season metadata")
		return
	}

	seasonMeta := s.convertToSeasonMetadata(seasonResults)

	if err := s.tv.UpdateSeasonsFromMetadata(ctx, seriesID, seasonMeta); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", seriesID).Msg("Failed to update seasons from metadata")
		return
	}

	totalEpisodes := 0
	for _, sm := range seasonMeta {
		totalEpisodes += len(sm.Episodes)
	}
	s.logger.Info().
		Int64("seriesId", seriesID).
		Int("seasons", len(seasonMeta)).
		Int("episodes", totalEpisodes).
		Msg("Updated seasons and episodes from metadata")
}

func (s *Service) convertToSeasonMetadata(seasonResults []metadata.SeasonResult) []tv.SeasonMetadata {
	seasonMeta := make([]tv.SeasonMetadata, len(seasonResults))
	for i, sr := range seasonResults {
		episodes := make([]tv.EpisodeMetadata, len(sr.Episodes))
		for j, ep := range sr.Episodes {
			episodes[j] = tv.EpisodeMetadata{
				EpisodeNumber: ep.EpisodeNumber,
				SeasonNumber:  ep.SeasonNumber,
				Title:         ep.Title,
				Overview:      ep.Overview,
				AirDate:       ep.AirDate,
				Runtime:       ep.Runtime,
			}
		}
		seasonMeta[i] = tv.SeasonMetadata{
			SeasonNumber: sr.SeasonNumber,
			Name:         sr.Name,
			Overview:     sr.Overview,
			PosterURL:    sr.PosterURL,
			AirDate:      sr.AirDate,
			Episodes:     episodes,
		}
	}
	return seasonMeta
}

func (s *Service) downloadSeriesArtworkFromResult(bestMatch *metadata.SeriesResult) {
	if s.artwork == nil || (bestMatch.PosterURL == "" && bestMatch.BackdropURL == "" && bestMatch.LogoURL == "") {
		return
	}

	go func() {
		if err := s.artwork.DownloadSeriesArtwork(context.Background(), bestMatch); err != nil {
			s.logger.Warn().Err(err).Int("tvdbId", bestMatch.TvdbID).Msg("Failed to download series artwork")
		}
	}()
}

// RefreshMonitoredSeriesMetadata refreshes metadata for all monitored series.
// This is called before auto-search to ensure we have the latest episode lists.
func (s *Service) RefreshMonitoredSeriesMetadata(ctx context.Context) (int, error) {
	// Get all monitored series
	monitored := true
	seriesList, err := s.tv.ListSeries(ctx, tv.ListSeriesOptions{Monitored: &monitored})
	if err != nil {
		return 0, fmt.Errorf("failed to list monitored series: %w", err)
	}

	if len(seriesList) == 0 {
		return 0, nil
	}

	s.logger.Info().Int("count", len(seriesList)).Msg("Refreshing metadata for monitored series")

	refreshed := 0
	for _, series := range seriesList {
		select {
		case <-ctx.Done():
			return refreshed, ctx.Err()
		default:
		}

		_, err := s.RefreshSeriesMetadata(ctx, series.ID)
		if err != nil {
			if errors.Is(err, ErrNoMetadataProvider) {
				s.logger.Warn().Msg("No metadata provider configured, stopping series refresh")
				return refreshed, nil
			}
			s.logger.Debug().Err(err).Int64("seriesId", series.ID).Str("title", series.Title).Msg("Failed to refresh series metadata")
			continue
		}
		refreshed++
	}

	return refreshed, nil
}

// RefreshAllMovies scans all movie root folders and refreshes metadata for all movies.
func (s *Service) RefreshAllMovies(ctx context.Context) error {
	activityID := fmt.Sprintf("refresh-movies-%d", time.Now().UnixNano())
	var activity *progress.ActivityBuilder
	if s.progress != nil {
		activity = s.progress.NewActivityBuilder(activityID, progress.ActivityTypeScan, "Refreshing all movies")
	}

	if err := s.scanAllMovieFolders(ctx, activity); err != nil {
		return err
	}

	allMovies, err := s.movies.List(ctx, movies.ListMoviesOptions{})
	if err != nil {
		s.failActivity(activity, err.Error())
		return fmt.Errorf("failed to list movies: %w", err)
	}

	refreshed := s.refreshAllMovieMetadata(ctx, allMovies, activity)

	if activity != nil {
		activity.Complete(fmt.Sprintf("Refreshed %d of %d movies", refreshed, len(allMovies)))
	}
	s.logger.Info().Int("refreshed", refreshed).Int("total", len(allMovies)).Msg("Completed refresh all movies")
	return nil
}

func (s *Service) scanAllMovieFolders(ctx context.Context, activity *progress.ActivityBuilder) error {
	if activity != nil {
		activity.Update("Scanning movie folders...", -1)
	}

	movieFolders, err := s.rootfolders.ListByType(ctx, "movie")
	if err != nil {
		s.failActivity(activity, err.Error())
		return fmt.Errorf("failed to list movie root folders: %w", err)
	}

	for _, folder := range movieFolders {
		if ctx.Err() != nil {
			s.failActivity(activity, "cancelled")
			return ctx.Err()
		}
		if _, err := s.ScanRootFolder(ctx, folder.ID); err != nil {
			s.logger.Warn().Err(err).Int64("rootFolderId", folder.ID).Msg("Failed to scan movie root folder during refresh all")
		}
	}
	return nil
}

func (s *Service) refreshAllMovieMetadata(ctx context.Context, allMovies []*movies.Movie, activity *progress.ActivityBuilder) int {
	if activity != nil {
		activity.Update("Refreshing movie metadata...", -1)
	}

	total := len(allMovies)
	refreshed := 0

	for i, movie := range allMovies {
		if ctx.Err() != nil {
			s.failActivity(activity, "cancelled")
			return refreshed
		}

		if activity != nil {
			pct := (i + 1) * 100 / total
			activity.Update(fmt.Sprintf("Refreshing: %s", movie.Title), pct)
		}

		if _, err := s.RefreshMovieMetadata(ctx, movie.ID); err != nil {
			if errors.Is(err, ErrNoMetadataProvider) {
				break
			}
			s.logger.Debug().Err(err).Int64("movieId", movie.ID).Str("title", movie.Title).Msg("Failed to refresh movie metadata")
			continue
		}
		refreshed++
	}

	return refreshed
}

// RefreshAllSeries scans all TV root folders and refreshes metadata for all series.
func (s *Service) RefreshAllSeries(ctx context.Context) error {
	activityID := fmt.Sprintf("refresh-series-%d", time.Now().UnixNano())
	var activity *progress.ActivityBuilder
	if s.progress != nil {
		activity = s.progress.NewActivityBuilder(activityID, progress.ActivityTypeScan, "Refreshing all series")
	}

	if err := s.scanAllTVFolders(ctx, activity); err != nil {
		return err
	}

	allSeries, err := s.tv.ListSeries(ctx, tv.ListSeriesOptions{})
	if err != nil {
		s.failActivity(activity, err.Error())
		return fmt.Errorf("failed to list series: %w", err)
	}

	refreshed := s.refreshAllSeriesMetadata(ctx, allSeries, activity)

	if activity != nil {
		activity.Complete(fmt.Sprintf("Refreshed %d of %d series", refreshed, len(allSeries)))
	}
	s.logger.Info().Int("refreshed", refreshed).Int("total", len(allSeries)).Msg("Completed refresh all series")
	return nil
}

func (s *Service) scanAllTVFolders(ctx context.Context, activity *progress.ActivityBuilder) error {
	if activity != nil {
		activity.Update("Scanning TV folders...", -1)
	}

	tvFolders, err := s.rootfolders.ListByType(ctx, "tv")
	if err != nil {
		s.failActivity(activity, err.Error())
		return fmt.Errorf("failed to list TV root folders: %w", err)
	}

	for _, folder := range tvFolders {
		if ctx.Err() != nil {
			s.failActivity(activity, "cancelled")
			return ctx.Err()
		}
		if _, err := s.ScanRootFolder(ctx, folder.ID); err != nil {
			s.logger.Warn().Err(err).Int64("rootFolderId", folder.ID).Msg("Failed to scan TV root folder during refresh all")
		}
	}
	return nil
}

func (s *Service) refreshAllSeriesMetadata(ctx context.Context, allSeries []*tv.Series, activity *progress.ActivityBuilder) int {
	if activity != nil {
		activity.Update("Refreshing series metadata...", -1)
	}

	total := len(allSeries)
	refreshed := 0

	for i, series := range allSeries {
		if ctx.Err() != nil {
			s.failActivity(activity, "cancelled")
			return refreshed
		}

		if activity != nil {
			pct := (i + 1) * 100 / total
			activity.Update(fmt.Sprintf("Refreshing: %s", series.Title), pct)
		}

		if _, err := s.RefreshSeriesMetadata(ctx, series.ID); err != nil {
			if errors.Is(err, ErrNoMetadataProvider) {
				break
			}
			s.logger.Debug().Err(err).Int64("seriesId", series.ID).Str("title", series.Title).Msg("Failed to refresh series metadata")
			continue
		}
		refreshed++
	}

	return refreshed
}

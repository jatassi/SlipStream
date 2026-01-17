package metadata

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/metadata/tmdb"
	"github.com/slipstream/slipstream/internal/metadata/tvdb"
)

var (
	ErrNoProvidersConfigured = errors.New("no metadata providers configured")
	ErrNotFound              = errors.New("metadata not found")
)

// HealthService is the interface for central health tracking.
type HealthService interface {
	RegisterItemStr(category, id, name string)
	UnregisterItemStr(category, id string)
	SetErrorStr(category, id, message string)
	SetWarningStr(category, id, message string)
	ClearStatusStr(category, id string)
}

// Service orchestrates metadata lookups across multiple providers.
type Service struct {
	tmdb          TMDBClient
	tvdb          TVDBClient
	cache         *Cache
	logger        zerolog.Logger
	healthService HealthService
}

// NewService creates a new metadata service with real API clients.
func NewService(cfg config.MetadataConfig, logger zerolog.Logger) *Service {
	return &Service{
		tmdb:   tmdb.NewClient(cfg.TMDB, logger),
		tvdb:   tvdb.NewClient(cfg.TVDB, logger),
		cache:  NewCache(DefaultCacheConfig()),
		logger: logger.With().Str("component", "metadata").Logger(),
	}
}

// NewServiceWithClients creates a new metadata service with custom clients (for testing/mocking).
func NewServiceWithClients(tmdbClient TMDBClient, tvdbClient TVDBClient, logger zerolog.Logger) *Service {
	return &Service{
		tmdb:   tmdbClient,
		tvdb:   tvdbClient,
		cache:  NewCache(DefaultCacheConfig()),
		logger: logger.With().Str("component", "metadata").Logger(),
	}
}

// SetClients replaces the TMDB and TVDB clients (for dev mode switching).
func (s *Service) SetClients(tmdbClient TMDBClient, tvdbClient TVDBClient) {
	s.tmdb = tmdbClient
	s.tvdb = tvdbClient
	s.cache.Clear()
}

// SetHealthService sets the central health service for registration tracking.
func (s *Service) SetHealthService(hs HealthService) {
	s.healthService = hs
}

// RegisterMetadataProviders registers configured metadata providers with the health service.
func (s *Service) RegisterMetadataProviders() {
	if s.healthService == nil {
		return
	}

	// Register TMDB if configured
	if s.tmdb.IsConfigured() {
		s.healthService.RegisterItemStr("metadata", "tmdb", "TMDB")
		s.logger.Debug().Msg("Registered TMDB with health service")
	}

	// Register TVDB if configured
	if s.tvdb.IsConfigured() {
		s.healthService.RegisterItemStr("metadata", "tvdb", "TVDB")
		s.logger.Debug().Msg("Registered TVDB with health service")
	}
}

// tmdbMovieToResult converts a TMDB movie result to metadata.MovieResult.
func tmdbMovieToResult(m tmdb.NormalizedMovieResult) MovieResult {
	return MovieResult{
		ID:          m.ID,
		Title:       m.Title,
		Year:        m.Year,
		Overview:    m.Overview,
		PosterURL:   m.PosterURL,
		BackdropURL: m.BackdropURL,
		ImdbID:      m.ImdbID,
		Genres:      m.Genres,
		Runtime:     m.Runtime,
	}
}

// tmdbSeriesToResult converts a TMDB series result to metadata.SeriesResult.
func tmdbSeriesToResult(s tmdb.NormalizedSeriesResult) SeriesResult {
	return SeriesResult{
		ID:          s.ID,
		Title:       s.Title,
		Year:        s.Year,
		Overview:    s.Overview,
		PosterURL:   s.PosterURL,
		BackdropURL: s.BackdropURL,
		ImdbID:      s.ImdbID,
		TvdbID:      s.TvdbID,
		TmdbID:      s.TmdbID,
		Genres:      s.Genres,
		Status:      s.Status,
		Runtime:     s.Runtime,
	}
}

// tvdbSeriesToResult converts a TVDB series result to metadata.SeriesResult.
func tvdbSeriesToResult(s tvdb.NormalizedSeriesResult) SeriesResult {
	return SeriesResult{
		ID:          s.ID,
		Title:       s.Title,
		Year:        s.Year,
		Overview:    s.Overview,
		PosterURL:   s.PosterURL,
		BackdropURL: s.BackdropURL,
		ImdbID:      s.ImdbID,
		TvdbID:      s.TvdbID,
		TmdbID:      s.TmdbID,
		Genres:      s.Genres,
		Status:      s.Status,
		Runtime:     s.Runtime,
	}
}

// HasMovieProvider returns true if a movie metadata provider is configured.
func (s *Service) HasMovieProvider() bool {
	return s.tmdb.IsConfigured()
}

// HasSeriesProvider returns true if a series metadata provider is configured.
func (s *Service) HasSeriesProvider() bool {
	return s.tmdb.IsConfigured() || s.tvdb.IsConfigured()
}

// SearchMovies searches for movies using available providers.
// If year is > 0, it will be used to filter results.
func (s *Service) SearchMovies(ctx context.Context, query string, year int) ([]MovieResult, error) {
	if !s.HasMovieProvider() {
		return nil, ErrNoProvidersConfigured
	}

	// Check cache
	cacheKey := fmt.Sprintf("movie:search:%s:%d", query, year)
	if results, ok := s.cache.GetMovieResults(cacheKey); ok {
		s.logger.Debug().Str("query", query).Int("year", year).Msg("Movie search cache hit")
		return results, nil
	}

	// Search TMDB (primary provider for movies)
	tmdbResults, err := s.tmdb.SearchMovies(ctx, query, year)
	if err != nil {
		s.logger.Error().Err(err).Str("query", query).Int("year", year).Msg("TMDB movie search failed")
		return nil, fmt.Errorf("movie search failed: %w", err)
	}

	// Convert to metadata.MovieResult
	results := make([]MovieResult, len(tmdbResults))
	for i, r := range tmdbResults {
		results[i] = tmdbMovieToResult(r)
	}

	// Cache results
	s.cache.Set(cacheKey, results)

	s.logger.Info().
		Str("query", query).
		Int("year", year).
		Int("results", len(results)).
		Msg("Movie search completed")

	return results, nil
}

// GetMovie gets detailed movie info by TMDB ID.
func (s *Service) GetMovie(ctx context.Context, tmdbID int) (*MovieResult, error) {
	if !s.HasMovieProvider() {
		return nil, ErrNoProvidersConfigured
	}

	// Check cache
	cacheKey := fmt.Sprintf("movie:%d", tmdbID)
	if result, ok := s.cache.GetMovieResult(cacheKey); ok {
		s.logger.Debug().Int("tmdbId", tmdbID).Msg("Movie cache hit")
		return result, nil
	}

	// Get from TMDB
	tmdbResult, err := s.tmdb.GetMovie(ctx, tmdbID)
	if err != nil {
		s.logger.Error().Err(err).Int("tmdbId", tmdbID).Msg("TMDB get movie failed")
		return nil, fmt.Errorf("get movie failed: %w", err)
	}

	// Convert to metadata.MovieResult
	result := tmdbMovieToResult(*tmdbResult)

	// Cache result
	s.cache.Set(cacheKey, &result)

	s.logger.Info().
		Int("tmdbId", tmdbID).
		Str("title", result.Title).
		Msg("Got movie details")

	return &result, nil
}

// SearchSeries searches for TV series using available providers.
func (s *Service) SearchSeries(ctx context.Context, query string) ([]SeriesResult, error) {
	if !s.HasSeriesProvider() {
		return nil, ErrNoProvidersConfigured
	}

	// Check cache
	cacheKey := fmt.Sprintf("series:search:%s", query)
	if results, ok := s.cache.GetSeriesResults(cacheKey); ok {
		s.logger.Debug().Str("query", query).Msg("Series search cache hit")
		return results, nil
	}

	// Try TMDB first (more comprehensive data)
	var results []SeriesResult
	var err error

	if s.tmdb.IsConfigured() {
		tmdbResults, tmdbErr := s.tmdb.SearchSeries(ctx, query)
		if tmdbErr != nil {
			s.logger.Warn().Err(tmdbErr).Str("query", query).Msg("TMDB series search failed, trying TVDB")
			err = tmdbErr
		} else {
			results = make([]SeriesResult, len(tmdbResults))
			for i, r := range tmdbResults {
				results[i] = tmdbSeriesToResult(r)
			}
		}
	}

	// Fall back to TVDB if TMDB failed or not configured
	if len(results) == 0 && s.tvdb.IsConfigured() {
		tvdbResults, tvdbErr := s.tvdb.SearchSeries(ctx, query)
		if tvdbErr != nil {
			s.logger.Error().Err(tvdbErr).Str("query", query).Msg("TVDB series search failed")
			return nil, fmt.Errorf("series search failed: %w", tvdbErr)
		}
		results = make([]SeriesResult, len(tvdbResults))
		for i, r := range tvdbResults {
			results[i] = tvdbSeriesToResult(r)
		}
		err = nil
	}

	// If we still have no results but had an error, return it
	if len(results) == 0 && err != nil {
		return nil, fmt.Errorf("series search failed: %w", err)
	}

	// Cache results
	if len(results) > 0 {
		s.cache.Set(cacheKey, results)
	}

	s.logger.Info().
		Str("query", query).
		Int("results", len(results)).
		Msg("Series search completed")

	return results, nil
}

// GetSeriesByTMDB gets detailed series info by TMDB ID.
func (s *Service) GetSeriesByTMDB(ctx context.Context, tmdbID int) (*SeriesResult, error) {
	if !s.tmdb.IsConfigured() {
		return nil, ErrNoProvidersConfigured
	}

	// Check cache
	cacheKey := fmt.Sprintf("series:tmdb:%d", tmdbID)
	if result, ok := s.cache.GetSeriesResult(cacheKey); ok {
		s.logger.Debug().Int("tmdbId", tmdbID).Msg("Series cache hit (TMDB)")
		return result, nil
	}

	// Get from TMDB
	tmdbResult, err := s.tmdb.GetSeries(ctx, tmdbID)
	if err != nil {
		s.logger.Error().Err(err).Int("tmdbId", tmdbID).Msg("TMDB get series failed")
		return nil, fmt.Errorf("get series failed: %w", err)
	}

	// Convert to metadata.SeriesResult
	result := tmdbSeriesToResult(*tmdbResult)

	// Ensure TMDB ID is set
	result.TmdbID = tmdbID

	// Cache result
	s.cache.Set(cacheKey, &result)

	s.logger.Info().
		Int("tmdbId", tmdbID).
		Str("title", result.Title).
		Msg("Got series details from TMDB")

	return &result, nil
}

// GetSeriesByTVDB gets detailed series info by TVDB ID.
func (s *Service) GetSeriesByTVDB(ctx context.Context, tvdbID int) (*SeriesResult, error) {
	if !s.tvdb.IsConfigured() {
		return nil, ErrNoProvidersConfigured
	}

	// Check cache
	cacheKey := fmt.Sprintf("series:tvdb:%d", tvdbID)
	if result, ok := s.cache.GetSeriesResult(cacheKey); ok {
		s.logger.Debug().Int("tvdbId", tvdbID).Msg("Series cache hit (TVDB)")
		return result, nil
	}

	// Get from TVDB
	tvdbResult, err := s.tvdb.GetSeries(ctx, tvdbID)
	if err != nil {
		s.logger.Error().Err(err).Int("tvdbId", tvdbID).Msg("TVDB get series failed")
		return nil, fmt.Errorf("get series failed: %w", err)
	}

	// Convert to metadata.SeriesResult
	result := tvdbSeriesToResult(*tvdbResult)

	// Cache result
	s.cache.Set(cacheKey, &result)

	s.logger.Info().
		Int("tvdbId", tvdbID).
		Str("title", result.Title).
		Msg("Got series details from TVDB")

	return &result, nil
}

// GetSeries gets series details, trying TVDB first (primary for TV), then TMDB.
func (s *Service) GetSeries(ctx context.Context, tvdbID, tmdbID int) (*SeriesResult, error) {
	// Try TVDB first if we have an ID
	if tvdbID > 0 && s.tvdb.IsConfigured() {
		result, err := s.GetSeriesByTVDB(ctx, tvdbID)
		if err == nil {
			return result, nil
		}
		s.logger.Warn().Err(err).Int("tvdbId", tvdbID).Msg("TVDB lookup failed, trying TMDB")
	}

	// Try TMDB as fallback
	if tmdbID > 0 && s.tmdb.IsConfigured() {
		return s.GetSeriesByTMDB(ctx, tmdbID)
	}

	return nil, ErrNotFound
}

// ClearCache clears the metadata cache.
func (s *Service) ClearCache() {
	s.cache.Clear()
	s.logger.Info().Msg("Metadata cache cleared")
}

// GetMovieReleaseDates fetches release dates for a movie by TMDB ID.
// Returns digital (streaming/VOD) and physical (Bluray) release dates.
func (s *Service) GetMovieReleaseDates(ctx context.Context, tmdbID int) (digital, physical string, err error) {
	if !s.tmdb.IsConfigured() {
		return "", "", ErrNoProvidersConfigured
	}

	return s.tmdb.GetMovieReleaseDates(ctx, tmdbID)
}

// GetTMDBImageURL returns a full TMDB image URL for a given path.
func (s *Service) GetTMDBImageURL(path, size string) string {
	return s.tmdb.GetImageURL(path, size)
}

// IsTMDBConfigured returns true if TMDB is configured.
func (s *Service) IsTMDBConfigured() bool {
	return s.tmdb.IsConfigured()
}

// IsTVDBConfigured returns true if TVDB is configured.
func (s *Service) IsTVDBConfigured() bool {
	return s.tvdb.IsConfigured()
}

// TestTMDB tests connectivity to the TMDB API.
func (s *Service) TestTMDB(ctx context.Context) error {
	return s.tmdb.Test(ctx)
}

// TestTVDB tests connectivity to the TVDB API.
func (s *Service) TestTVDB(ctx context.Context) error {
	return s.tvdb.Test(ctx)
}

// tmdbSeasonToResult converts a TMDB season result to metadata.SeasonResult.
func tmdbSeasonToResult(s tmdb.NormalizedSeasonResult) SeasonResult {
	episodes := make([]EpisodeResult, len(s.Episodes))
	for i, ep := range s.Episodes {
		episodes[i] = EpisodeResult{
			EpisodeNumber: ep.EpisodeNumber,
			SeasonNumber:  ep.SeasonNumber,
			Title:         ep.Title,
			Overview:      ep.Overview,
			AirDate:       ep.AirDate,
			Runtime:       ep.Runtime,
		}
	}
	return SeasonResult{
		SeasonNumber: s.SeasonNumber,
		Name:         s.Name,
		Overview:     s.Overview,
		PosterURL:    s.PosterURL,
		AirDate:      s.AirDate,
		Episodes:     episodes,
	}
}

// tvdbSeasonToResult converts a TVDB season result to metadata.SeasonResult.
func tvdbSeasonToResult(s tvdb.NormalizedSeasonResult) SeasonResult {
	episodes := make([]EpisodeResult, len(s.Episodes))
	for i, ep := range s.Episodes {
		episodes[i] = EpisodeResult{
			EpisodeNumber: ep.EpisodeNumber,
			SeasonNumber:  ep.SeasonNumber,
			Title:         ep.Title,
			Overview:      ep.Overview,
			AirDate:       ep.AirDate,
			Runtime:       ep.Runtime,
		}
	}
	return SeasonResult{
		SeasonNumber: s.SeasonNumber,
		Name:         s.Name,
		Overview:     s.Overview,
		PosterURL:    s.PosterURL,
		AirDate:      s.AirDate,
		Episodes:     episodes,
	}
}

// GetSeriesSeasons gets all seasons and episodes for a series.
// Uses TMDB as primary, TVDB as fallback.
func (s *Service) GetSeriesSeasons(ctx context.Context, tmdbID, tvdbID int) ([]SeasonResult, error) {
	if !s.HasSeriesProvider() {
		return nil, ErrNoProvidersConfigured
	}

	// Check cache
	cacheKey := fmt.Sprintf("series:seasons:tmdb:%d:tvdb:%d", tmdbID, tvdbID)
	if results, ok := s.cache.GetSeasonResults(cacheKey); ok {
		s.logger.Debug().Int("tmdbId", tmdbID).Int("tvdbId", tvdbID).Msg("Series seasons cache hit")
		return results, nil
	}

	var results []SeasonResult
	var err error

	// Try TMDB first (more comprehensive data)
	if tmdbID > 0 && s.tmdb.IsConfigured() {
		tmdbResults, tmdbErr := s.tmdb.GetAllSeasons(ctx, tmdbID)
		if tmdbErr != nil {
			s.logger.Warn().Err(tmdbErr).Int("tmdbId", tmdbID).Msg("TMDB get seasons failed, trying TVDB")
			err = tmdbErr
		} else {
			results = make([]SeasonResult, len(tmdbResults))
			for i, r := range tmdbResults {
				results[i] = tmdbSeasonToResult(r)
			}
		}
	}

	// Fall back to TVDB if TMDB failed or not configured
	if len(results) == 0 && tvdbID > 0 && s.tvdb.IsConfigured() {
		tvdbResults, tvdbErr := s.tvdb.GetSeriesEpisodes(ctx, tvdbID)
		if tvdbErr != nil {
			s.logger.Error().Err(tvdbErr).Int("tvdbId", tvdbID).Msg("TVDB get episodes failed")
			if err != nil {
				return nil, fmt.Errorf("get series seasons failed: %w", err)
			}
			return nil, fmt.Errorf("get series seasons failed: %w", tvdbErr)
		}
		results = make([]SeasonResult, len(tvdbResults))
		for i, r := range tvdbResults {
			results[i] = tvdbSeasonToResult(r)
		}
		err = nil
	}

	// If we still have no results but had an error, return it
	if len(results) == 0 && err != nil {
		return nil, fmt.Errorf("get series seasons failed: %w", err)
	}

	// Cache results
	if len(results) > 0 {
		s.cache.Set(cacheKey, results)
	}

	totalEpisodes := 0
	for _, season := range results {
		totalEpisodes += len(season.Episodes)
	}

	s.logger.Info().
		Int("tmdbId", tmdbID).
		Int("tvdbId", tvdbID).
		Int("seasons", len(results)).
		Int("episodes", totalEpisodes).
		Msg("Got series seasons")

	return results, nil
}

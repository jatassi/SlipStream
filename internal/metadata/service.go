package metadata

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/metadata/omdb"
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
	tmdb             TMDBClient
	tvdb             TVDBClient
	omdb             OMDBClient
	cache            *Cache
	logger           zerolog.Logger
	healthService    HealthService
	networkLogoStore NetworkLogoStore
}

// NewService creates a new metadata service with real API clients.
func NewService(cfg *config.MetadataConfig, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "metadata").Logger()
	return &Service{
		tmdb:   tmdb.NewClient(cfg.TMDB, logger),
		tvdb:   tvdb.NewClient(cfg.TVDB, logger),
		omdb:   omdb.NewClient(cfg.OMDB, logger),
		cache:  NewCache(DefaultCacheConfig()),
		logger: subLogger,
	}
}

// NewServiceWithClients creates a new metadata service with custom clients (for testing/mocking).
func NewServiceWithClients(tmdbClient TMDBClient, tvdbClient TVDBClient, omdbClient OMDBClient, logger *zerolog.Logger) *Service {
	return &Service{
		tmdb:   tmdbClient,
		tvdb:   tvdbClient,
		omdb:   omdbClient,
		cache:  NewCache(DefaultCacheConfig()),
		logger: logger.With().Str("component", "metadata").Logger(),
	}
}

// SetClients replaces the TMDB, TVDB, and OMDb clients (for dev mode switching).
func (s *Service) SetClients(tmdbClient TMDBClient, tvdbClient TVDBClient, omdbClient OMDBClient) {
	s.tmdb = tmdbClient
	s.tvdb = tvdbClient
	s.omdb = omdbClient
	s.cache.Clear()
}

// SetOMDBClient sets the OMDb client.
func (s *Service) SetOMDBClient(client OMDBClient) {
	s.omdb = client
}

// SetHealthService sets the central health service for registration tracking.
func (s *Service) SetHealthService(hs HealthService) {
	s.healthService = hs
}

// SetNetworkLogoStore sets the store for caching network logo URLs.
func (s *Service) SetNetworkLogoStore(store NetworkLogoStore) {
	s.networkLogoStore = store
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

	// Register OMDb if configured
	if s.omdb != nil && s.omdb.IsConfigured() {
		s.healthService.RegisterItemStr("metadata", "omdb", "OMDb")
		s.logger.Debug().Msg("Registered OMDb with health service")
	}
}

// tmdbMovieToResult converts a TMDB movie result to metadata.MovieResult.
func tmdbMovieToResult(m *tmdb.NormalizedMovieResult) MovieResult {
	return MovieResult{
		ID:            m.ID,
		Title:         m.Title,
		Year:          m.Year,
		Overview:      m.Overview,
		PosterURL:     m.PosterURL,
		BackdropURL:   m.BackdropURL,
		ImdbID:        m.ImdbID,
		Genres:        m.Genres,
		Runtime:       m.Runtime,
		Studio:        m.Studio,
		StudioLogoURL: m.StudioLogoURL,
	}
}

// tmdbSeriesToResult converts a TMDB series result to metadata.SeriesResult.
func tmdbSeriesToResult(s *tmdb.NormalizedSeriesResult) SeriesResult {
	return SeriesResult{
		ID:             s.ID,
		Title:          s.Title,
		Year:           s.Year,
		Overview:       s.Overview,
		PosterURL:      s.PosterURL,
		BackdropURL:    s.BackdropURL,
		ImdbID:         s.ImdbID,
		TvdbID:         s.TvdbID,
		TmdbID:         s.TmdbID,
		Genres:         s.Genres,
		Status:         s.Status,
		Runtime:        s.Runtime,
		Network:        s.Network,
		NetworkLogoURL: s.NetworkLogoURL,
	}
}

// tvdbSeriesToResult converts a TVDB series result to metadata.SeriesResult.
func tvdbSeriesToResult(s *tvdb.NormalizedSeriesResult) SeriesResult {
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
		Network:     s.Network,
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
	for i := range tmdbResults {
		results[i] = tmdbMovieToResult(&tmdbResults[i])
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
	result := tmdbMovieToResult(tmdbResult)

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

	cacheKey := fmt.Sprintf("series:search:%s", query)
	if results, ok := s.cache.GetSeriesResults(cacheKey); ok {
		s.logger.Debug().Str("query", query).Msg("Series search cache hit")
		return results, nil
	}

	results, err := s.searchSeriesFromProviders(ctx, query)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		s.cache.Set(cacheKey, results)
	}

	s.logger.Info().Str("query", query).Int("results", len(results)).Msg("Series search completed")
	return results, nil
}

func (s *Service) searchSeriesFromProviders(ctx context.Context, query string) ([]SeriesResult, error) {
	results, err := s.tryTVDBSearch(ctx, query)
	if len(results) > 0 {
		return results, nil
	}

	tmdbResults, tmdbErr := s.tryTMDBSeriesSearch(ctx, query)
	if tmdbErr != nil {
		if err != nil {
			return nil, fmt.Errorf("series search failed: %w", err)
		}
		return nil, fmt.Errorf("series search failed: %w", tmdbErr)
	}
	return tmdbResults, nil
}

func (s *Service) tryTVDBSearch(ctx context.Context, query string) ([]SeriesResult, error) {
	if !s.tvdb.IsConfigured() {
		return nil, nil
	}

	tvdbResults, err := s.tvdb.SearchSeries(ctx, query)
	if err != nil {
		s.logger.Warn().Err(err).Str("query", query).Msg("TVDB series search failed, trying TMDB")
		return nil, err
	}

	results := make([]SeriesResult, len(tvdbResults))
	for i := range tvdbResults {
		results[i] = tvdbSeriesToResult(&tvdbResults[i])
	}
	s.enrichWithNetworkLogos(ctx, results)
	return results, nil
}

func (s *Service) tryTMDBSeriesSearch(ctx context.Context, query string) ([]SeriesResult, error) {
	if !s.tmdb.IsConfigured() {
		return nil, ErrNoProvidersConfigured
	}

	tmdbResults, err := s.tmdb.SearchSeries(ctx, query)
	if err != nil {
		s.logger.Error().Err(err).Str("query", query).Msg("TMDB series search failed")
		return nil, fmt.Errorf("series search failed: %w", err)
	}

	results := make([]SeriesResult, len(tmdbResults))
	for i := range tmdbResults {
		results[i] = tmdbSeriesToResult(&tmdbResults[i])
	}
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
	result := tmdbSeriesToResult(tmdbResult)

	// Ensure TMDB ID is set
	result.TmdbID = tmdbID

	// Cache network logo for future TVDB search enrichment
	if s.networkLogoStore != nil && result.Network != "" && result.NetworkLogoURL != "" {
		if err := s.networkLogoStore.UpsertNetworkLogo(ctx, result.Network, result.NetworkLogoURL); err != nil {
			s.logger.Warn().Err(err).Str("network", result.Network).Msg("Failed to cache network logo")
		}
	}

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
	result := tvdbSeriesToResult(tvdbResult)

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
// Returns digital (streaming/VOD), physical (Bluray), and theatrical release dates.
func (s *Service) GetMovieReleaseDates(ctx context.Context, tmdbID int) (digital, physical, theatrical string, err error) {
	if !s.tmdb.IsConfigured() {
		return "", "", "", ErrNoProvidersConfigured
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

// IsOMDBConfigured returns true if OMDb is configured.
func (s *Service) IsOMDBConfigured() bool {
	return s.omdb != nil && s.omdb.IsConfigured()
}

// TestOMDB tests connectivity to the OMDb API.
func (s *Service) TestOMDB(ctx context.Context) error {
	if s.omdb == nil {
		return ErrNoProvidersConfigured
	}
	return s.omdb.Test(ctx)
}

// GetOMDBRatings gets ratings from OMDb by IMDb ID.
func (s *Service) GetOMDBRatings(ctx context.Context, imdbID string) (*omdb.NormalizedRatings, error) {
	if s.omdb == nil || !s.omdb.IsConfigured() {
		return nil, ErrNoProvidersConfigured
	}
	return s.omdb.GetByIMDbID(ctx, imdbID)
}

// GetMovieLogoURL fetches the title treatment logo URL for a movie from TMDB.
func (s *Service) GetMovieLogoURL(ctx context.Context, tmdbID int) (string, error) {
	if !s.tmdb.IsConfigured() {
		return "", ErrNoProvidersConfigured
	}
	return s.tmdb.GetMovieLogoURL(ctx, tmdbID)
}

// GetMovieContentRating fetches the content rating (e.g. PG-13, R) for a movie from TMDB.
func (s *Service) GetMovieContentRating(ctx context.Context, tmdbID int) (string, error) {
	if !s.tmdb.IsConfigured() {
		return "", ErrNoProvidersConfigured
	}
	return s.tmdb.GetMovieContentRating(ctx, tmdbID)
}

// GetSeriesLogoURL fetches the title treatment logo URL for a series from TMDB.
func (s *Service) GetSeriesLogoURL(ctx context.Context, tmdbID int) (string, error) {
	if !s.tmdb.IsConfigured() {
		return "", ErrNoProvidersConfigured
	}
	return s.tmdb.GetSeriesLogoURL(ctx, tmdbID)
}

// enrichWithNetworkLogos populates NetworkLogoURL from the cache for results that have a network name but no logo.
func (s *Service) enrichWithNetworkLogos(ctx context.Context, results []SeriesResult) {
	if s.networkLogoStore == nil || !s.hasResultsNeedingLogos(results) {
		return
	}

	logos, err := s.networkLogoStore.GetAllNetworkLogos(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to get cached network logos")
		return
	}

	for i := range results {
		if results[i].Network != "" && results[i].NetworkLogoURL == "" {
			if logoURL, ok := logos[results[i].Network]; ok {
				results[i].NetworkLogoURL = logoURL
			}
		}
	}
}

func (s *Service) hasResultsNeedingLogos(results []SeriesResult) bool {
	for i := range results {
		if results[i].Network != "" && results[i].NetworkLogoURL == "" {
			return true
		}
	}
	return false
}

// tmdbSeasonToResult converts a TMDB season result to metadata.SeasonResult.
func tmdbSeasonToResult(s *tmdb.NormalizedSeasonResult) SeasonResult {
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
func tvdbSeasonToResult(s *tvdb.NormalizedSeasonResult) SeasonResult {
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

	cacheKey := fmt.Sprintf("series:seasons:tmdb:%d:tvdb:%d", tmdbID, tvdbID)
	if results, ok := s.cache.GetSeasonResults(cacheKey); ok {
		s.logger.Debug().Int("tmdbId", tmdbID).Int("tvdbId", tvdbID).Msg("Series seasons cache hit")
		return results, nil
	}

	results, err := s.fetchSeasonsFromProviders(ctx, tmdbID, tvdbID)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		s.cache.Set(cacheKey, results)
	}

	s.logSeasonsFetched(tmdbID, tvdbID, results)
	return results, nil
}

func (s *Service) fetchSeasonsFromProviders(ctx context.Context, tmdbID, tvdbID int) ([]SeasonResult, error) {
	results, err := s.tryTVDBSeasons(ctx, tvdbID)
	if len(results) > 0 {
		return results, nil
	}

	tmdbResults, tmdbErr := s.tryTMDBSeasons(ctx, tmdbID)
	if tmdbErr != nil {
		if err != nil {
			return nil, fmt.Errorf("get series seasons failed: %w", err)
		}
		return nil, fmt.Errorf("get series seasons failed: %w", tmdbErr)
	}
	return tmdbResults, nil
}

func (s *Service) tryTVDBSeasons(ctx context.Context, tvdbID int) ([]SeasonResult, error) {
	if tvdbID <= 0 || !s.tvdb.IsConfigured() {
		return nil, nil
	}

	tvdbResults, err := s.tvdb.GetSeriesEpisodes(ctx, tvdbID)
	if err != nil {
		s.logger.Warn().Err(err).Int("tvdbId", tvdbID).Msg("TVDB get episodes failed, trying TMDB")
		return nil, err
	}

	results := make([]SeasonResult, len(tvdbResults))
	for i := range tvdbResults {
		results[i] = tvdbSeasonToResult(&tvdbResults[i])
	}
	return results, nil
}

func (s *Service) tryTMDBSeasons(ctx context.Context, tmdbID int) ([]SeasonResult, error) {
	if tmdbID <= 0 || !s.tmdb.IsConfigured() {
		return nil, ErrNoProvidersConfigured
	}

	tmdbResults, err := s.tmdb.GetAllSeasons(ctx, tmdbID)
	if err != nil {
		s.logger.Error().Err(err).Int("tmdbId", tmdbID).Msg("TMDB get seasons failed")
		return nil, fmt.Errorf("get series seasons failed: %w", err)
	}

	results := make([]SeasonResult, len(tmdbResults))
	for i := range tmdbResults {
		results[i] = tmdbSeasonToResult(&tmdbResults[i])
	}
	return results, nil
}

func (s *Service) logSeasonsFetched(tmdbID, tvdbID int, results []SeasonResult) {
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
}

// GetExtendedMovie gets extended movie metadata including credits, ratings, and content rating.
func (s *Service) GetExtendedMovie(ctx context.Context, tmdbID int) (*ExtendedMovieResult, error) {
	if !s.HasMovieProvider() {
		return nil, ErrNoProvidersConfigured
	}

	movie, err := s.GetMovie(ctx, tmdbID)
	if err != nil {
		return nil, err
	}

	result := &ExtendedMovieResult{MovieResult: *movie}
	s.enrichMovieMetadata(ctx, tmdbID, movie, result)

	s.logger.Info().Int("tmdbId", tmdbID).Str("title", movie.Title).Msg("Got extended movie metadata")
	return result, nil
}

func (s *Service) enrichMovieMetadata(ctx context.Context, tmdbID int, movie *MovieResult, result *ExtendedMovieResult) {
	if credits, err := s.tmdb.GetMovieCredits(ctx, tmdbID); err == nil {
		result.Credits = tmdbCreditsToCredits(credits)
	} else {
		s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to get movie credits")
	}

	if contentRating, err := s.tmdb.GetMovieContentRating(ctx, tmdbID); err == nil {
		result.ContentRating = contentRating
	} else {
		s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to get movie content rating")
	}

	if studio, err := s.tmdb.GetMovieStudio(ctx, tmdbID); err == nil {
		result.Studio = studio
	} else {
		s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to get movie studio")
	}

	if trailerURL, err := s.tmdb.GetMovieTrailerURL(ctx, tmdbID); err == nil {
		result.TrailerURL = trailerURL
	} else {
		s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to get movie trailer URL")
	}

	if movie.ImdbID != "" && s.omdb != nil && s.omdb.IsConfigured() {
		if omdbRatings, err := s.omdb.GetByIMDbID(ctx, movie.ImdbID); err == nil {
			result.Ratings = omdbRatingsToExternalRatings(omdbRatings)
		} else {
			s.logger.Warn().Err(err).Str("imdbId", movie.ImdbID).Msg("Failed to get OMDb ratings")
		}
	}
}

// GetExtendedSeries gets extended series metadata including credits, ratings, seasons, and content rating.
func (s *Service) GetExtendedSeries(ctx context.Context, tmdbID int) (*ExtendedSeriesResult, error) {
	if !s.HasSeriesProvider() {
		return nil, ErrNoProvidersConfigured
	}

	series, err := s.GetSeriesByTMDB(ctx, tmdbID)
	if err != nil {
		return nil, err
	}

	result := &ExtendedSeriesResult{SeriesResult: *series}
	s.enrichSeriesMetadata(ctx, tmdbID, series, result)

	s.logger.Info().Int("tmdbId", tmdbID).Str("title", series.Title).Msg("Got extended series metadata")
	return result, nil
}

func (s *Service) enrichSeriesMetadata(ctx context.Context, tmdbID int, series *SeriesResult, result *ExtendedSeriesResult) {
	if credits, err := s.tmdb.GetSeriesCredits(ctx, tmdbID); err == nil {
		result.Credits = tmdbCreditsToCredits(credits)
	} else {
		s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to get series credits")
	}

	if contentRating, err := s.tmdb.GetSeriesContentRating(ctx, tmdbID); err == nil {
		result.ContentRating = contentRating
	} else {
		s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to get series content rating")
	}

	if seasons, err := s.GetSeriesSeasons(ctx, tmdbID, series.TvdbID); err == nil {
		result.Seasons = seasons
	} else {
		s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to get series seasons")
	}

	if trailerURL, err := s.tmdb.GetSeriesTrailerURL(ctx, tmdbID); err == nil {
		result.TrailerURL = trailerURL
	} else {
		s.logger.Warn().Err(err).Int("tmdbId", tmdbID).Msg("Failed to get series trailer URL")
	}

	s.enrichSeriesOMDbData(ctx, series, result)
}

func (s *Service) enrichSeriesOMDbData(ctx context.Context, series *SeriesResult, result *ExtendedSeriesResult) {
	if series.ImdbID == "" || s.omdb == nil || !s.omdb.IsConfigured() {
		return
	}

	omdbRatings, err := s.omdb.GetByIMDbID(ctx, series.ImdbID)
	if err != nil {
		s.logger.Warn().Err(err).Str("imdbId", series.ImdbID).Msg("Failed to get OMDb ratings")
		return
	}

	result.Ratings = omdbRatingsToExternalRatings(omdbRatings)
	s.enrichEpisodeRatings(ctx, series.ImdbID, result)
}

func (s *Service) enrichEpisodeRatings(ctx context.Context, imdbID string, result *ExtendedSeriesResult) {
	for i, season := range result.Seasons {
		epRatings, err := s.omdb.GetSeasonEpisodes(ctx, imdbID, season.SeasonNumber)
		if err != nil {
			s.logger.Debug().Err(err).Int("season", season.SeasonNumber).Msg("Failed to get episode ratings")
			continue
		}
		for j := range result.Seasons[i].Episodes {
			if rating, ok := epRatings[result.Seasons[i].Episodes[j].EpisodeNumber]; ok {
				result.Seasons[i].Episodes[j].ImdbRating = rating
			}
		}
	}
}

// tmdbCreditsToCredits converts TMDB credits to metadata.Credits.
func tmdbCreditsToCredits(c *tmdb.NormalizedCredits) *Credits {
	if c == nil {
		return nil
	}

	result := &Credits{
		Cast: make([]Person, len(c.Cast)),
	}

	for _, p := range c.Directors {
		result.Directors = append(result.Directors, Person{
			ID:       p.ID,
			Name:     p.Name,
			Role:     p.Role,
			PhotoURL: p.PhotoURL,
		})
	}

	for _, p := range c.Writers {
		result.Writers = append(result.Writers, Person{
			ID:       p.ID,
			Name:     p.Name,
			Role:     p.Role,
			PhotoURL: p.PhotoURL,
		})
	}

	for _, p := range c.Creators {
		result.Creators = append(result.Creators, Person{
			ID:       p.ID,
			Name:     p.Name,
			Role:     p.Role,
			PhotoURL: p.PhotoURL,
		})
	}

	for i, p := range c.Cast {
		result.Cast[i] = Person{
			ID:       p.ID,
			Name:     p.Name,
			Role:     p.Role,
			PhotoURL: p.PhotoURL,
		}
	}

	return result
}

// omdbRatingsToExternalRatings converts OMDb ratings to metadata.ExternalRatings.
func omdbRatingsToExternalRatings(r *omdb.NormalizedRatings) *ExternalRatings {
	if r == nil {
		return nil
	}

	return &ExternalRatings{
		ImdbRating:     r.ImdbRating,
		ImdbVotes:      r.ImdbVotes,
		RottenTomatoes: r.RottenTomatoes,
		RottenAudience: r.RottenAudience,
		Metacritic:     r.Metacritic,
		Awards:         r.Awards,
	}
}

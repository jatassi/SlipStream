package prowlarr

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/indexer/types"
)

// Error variables are defined in errors.go

const (
	searchCacheTTL           = 5 * time.Minute
	capabilitiesRefreshTTL   = 15 * time.Minute
	indexerRefreshTTL        = 15 * time.Minute
)

// Service manages Prowlarr integration.
type Service struct {
	db          *sql.DB
	queries     *sqlc.Queries
	logger      zerolog.Logger
	rateLimiter *RateLimiter

	mu            sync.RWMutex
	client        *Client
	config        *Config
	capabilities  *Capabilities
	indexers      []Indexer
	lastCapFetch  time.Time
	lastIdxFetch  time.Time

	searchCache   map[string]*searchCacheEntry
	searchCacheMu sync.RWMutex
}

type searchCacheEntry struct {
	results   []types.TorrentInfo
	fetchedAt time.Time
}

// NewService creates a new Prowlarr service.
func NewService(db *sql.DB, logger zerolog.Logger) *Service {
	svcLogger := logger.With().Str("component", "prowlarr-service").Logger()

	return &Service{
		db:          db,
		queries:     sqlc.New(db),
		logger:      svcLogger,
		rateLimiter: NewRateLimiter(DefaultRateLimiterConfig(svcLogger)),
		searchCache: make(map[string]*searchCacheEntry),
	}
}

// SetDB updates the database connection used by this service.
func (s *Service) SetDB(db *sql.DB) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db = db
	s.queries = sqlc.New(db)
	s.client = nil // Reset client when DB changes
	s.config = nil
}

// GetConfig returns the current Prowlarr configuration.
func (s *Service) GetConfig(ctx context.Context) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config != nil {
		return s.config, nil
	}

	return s.loadConfigLocked(ctx)
}

// loadConfigLocked loads config from database (must hold mu lock).
func (s *Service) loadConfigLocked(ctx context.Context) (*Config, error) {
	row, err := s.queries.GetProwlarrConfig(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotConfigured
		}
		return nil, fmt.Errorf("failed to load prowlarr config: %w", err)
	}

	config := s.rowToConfig(row)
	s.config = config
	return config, nil
}

// UpdateConfig updates the Prowlarr configuration.
func (s *Service) UpdateConfig(ctx context.Context, input ConfigInput) (*Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	movieCats, _ := json.Marshal(input.MovieCategories)
	tvCats, _ := json.Marshal(input.TVCategories)

	row, err := s.queries.UpdateProwlarrConfig(ctx, sqlc.UpdateProwlarrConfigParams{
		Enabled:         boolToInt64(input.Enabled),
		Url:             input.URL,
		ApiKey:          input.APIKey,
		MovieCategories: string(movieCats),
		TvCategories:    string(tvCats),
		Timeout:         int64(input.Timeout),
		SkipSslVerify:   boolToInt64(input.SkipSSLVerify),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update prowlarr config: %w", err)
	}

	config := s.rowToConfig(row)
	s.config = config
	s.client = nil // Reset client to pick up new config
	s.capabilities = nil
	s.indexers = nil

	s.logger.Info().
		Str("url", config.URL).
		Bool("enabled", config.Enabled).
		Msg("prowlarr config updated")

	return config, nil
}

// ConfigInput represents input for updating Prowlarr configuration.
type ConfigInput struct {
	Enabled         bool   `json:"enabled"`
	URL             string `json:"url"`
	APIKey          string `json:"apiKey"`
	MovieCategories []int  `json:"movieCategories"`
	TVCategories    []int  `json:"tvCategories"`
	Timeout         int    `json:"timeout"`
	SkipSSLVerify   bool   `json:"skipSslVerify"`
}

// IsEnabled returns whether Prowlarr mode is currently enabled.
func (s *Service) IsEnabled(ctx context.Context) (bool, error) {
	config, err := s.GetConfig(ctx)
	if err != nil {
		if errors.Is(err, ErrNotConfigured) {
			return false, nil
		}
		return false, err
	}
	return config.Enabled, nil
}

// SetEnabled enables or disables Prowlarr mode.
func (s *Service) SetEnabled(ctx context.Context, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.queries.SetProwlarrEnabled(ctx, boolToInt64(enabled)); err != nil {
		return fmt.Errorf("failed to set prowlarr enabled: %w", err)
	}

	if s.config != nil {
		s.config.Enabled = enabled
	}

	s.logger.Info().
		Bool("enabled", enabled).
		Msg("prowlarr mode changed")

	return nil
}

// TestConnection tests the Prowlarr connection.
func (s *Service) TestConnection(ctx context.Context, url, apiKey string, timeout int, skipSSL bool) error {
	client, err := NewClient(ClientConfig{
		URL:           url,
		APIKey:        apiKey,
		Timeout:       timeout,
		SkipSSLVerify: skipSSL,
		Logger:        s.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if err := client.TestConnection(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	return nil
}

// GetClient returns or creates the Prowlarr HTTP client.
func (s *Service) GetClient(ctx context.Context) (*Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		return s.client, nil
	}

	config, err := s.loadConfigLocked(ctx)
	if err != nil {
		return nil, err
	}

	if !config.Enabled {
		return nil, ErrNotConfigured
	}

	client, err := NewClient(ClientConfig{
		URL:           config.URL,
		APIKey:        config.APIKey,
		Timeout:       config.Timeout,
		SkipSSLVerify: config.SkipSSLVerify,
		Logger:        s.logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create prowlarr client: %w", err)
	}

	s.client = client
	return client, nil
}

// GetCapabilities returns the cached capabilities or fetches fresh ones.
func (s *Service) GetCapabilities(ctx context.Context) (*Capabilities, error) {
	s.mu.RLock()
	if s.capabilities != nil && time.Since(s.lastCapFetch) < capabilitiesRefreshTTL {
		caps := s.capabilities
		s.mu.RUnlock()
		return caps, nil
	}
	s.mu.RUnlock()

	return s.RefreshCapabilities(ctx)
}

// RefreshCapabilities fetches fresh capabilities from Prowlarr.
func (s *Service) RefreshCapabilities(ctx context.Context) (*Capabilities, error) {
	client, err := s.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	s.rateLimiter.Wait()

	caps, err := client.GetCapabilities(ctx)
	if err != nil {
		s.rateLimiter.RecordError()
		return nil, fmt.Errorf("failed to fetch capabilities: %w", err)
	}

	s.rateLimiter.RecordSuccess()

	s.mu.Lock()
	s.capabilities = caps
	s.lastCapFetch = time.Now()
	s.mu.Unlock()

	// Cache to database
	capsJSON, _ := json.Marshal(caps)
	_ = s.queries.UpdateProwlarrConfigCapabilities(ctx, sql.NullString{
		String: string(capsJSON),
		Valid:  true,
	})

	s.logger.Debug().Msg("capabilities refreshed")

	return caps, nil
}

// GetIndexers returns the cached indexer list or fetches fresh ones.
func (s *Service) GetIndexers(ctx context.Context) ([]Indexer, error) {
	s.mu.RLock()
	if s.indexers != nil && time.Since(s.lastIdxFetch) < indexerRefreshTTL {
		idxs := s.indexers
		s.mu.RUnlock()
		return idxs, nil
	}
	s.mu.RUnlock()

	return s.RefreshIndexers(ctx)
}

// RefreshIndexers fetches fresh indexer list from Prowlarr.
func (s *Service) RefreshIndexers(ctx context.Context) ([]Indexer, error) {
	client, err := s.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	s.rateLimiter.Wait()

	indexers, err := client.GetIndexers(ctx)
	if err != nil {
		s.rateLimiter.RecordError()
		return nil, fmt.Errorf("failed to fetch indexers: %w", err)
	}

	s.rateLimiter.RecordSuccess()

	s.mu.Lock()
	s.indexers = indexers
	s.lastIdxFetch = time.Now()
	s.mu.Unlock()

	s.logger.Debug().
		Int("count", len(indexers)).
		Msg("indexers refreshed")

	return indexers, nil
}

// GetConnectionStatus returns the current Prowlarr connection status.
func (s *Service) GetConnectionStatus(ctx context.Context) (*ConnectionStatus, error) {
	client, err := s.GetClient(ctx)
	if err != nil {
		now := time.Now()
		return &ConnectionStatus{
			Connected:   false,
			LastChecked: &now,
			Error:       err.Error(),
		}, nil
	}

	return client.GetSystemStatus(ctx)
}

// Search executes a search through Prowlarr.
func (s *Service) Search(ctx context.Context, criteria types.SearchCriteria) ([]types.TorrentInfo, error) {
	config, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	if !config.Enabled {
		return nil, ErrNotConfigured
	}

	// Check cache
	cacheKey := s.searchCacheKey(criteria)
	if cached := s.getFromSearchCache(cacheKey); cached != nil {
		s.logger.Debug().
			Str("cacheKey", cacheKey).
			Int("results", len(cached)).
			Msg("returning cached search results")
		return cached, nil
	}

	client, err := s.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	// Load indexers and settings for filtering and priority
	indexers, _ := s.GetIndexers(ctx)
	allSettings, _ := s.GetAllIndexerSettings(ctx)

	// Build indexer name -> (ID, settings) map
	indexerNameToID := make(map[string]int)
	for _, idx := range indexers {
		indexerNameToID[idx.Name] = idx.ID
	}

	settingsMap := make(map[int64]*IndexerSettings)
	for i := range allSettings {
		settingsMap[allSettings[i].ProwlarrIndexerID] = &allSettings[i]
	}

	// Build search request with per-indexer categories if available
	req := s.buildSearchRequestWithSettings(criteria, config, settingsMap, indexerNameToID)

	s.rateLimiter.Wait()

	feed, err := client.Search(ctx, req)
	if err != nil {
		s.rateLimiter.RecordError()
		return nil, fmt.Errorf("%w: %v", ErrSearchFailed, err)
	}

	s.rateLimiter.RecordSuccess()

	// Determine content type filter based on search type
	var contentTypeFilter ContentType
	switch criteria.Type {
	case "movie":
		contentTypeFilter = ContentTypeMovies
	case "tvsearch":
		contentTypeFilter = ContentTypeSeries
	default:
		contentTypeFilter = ContentTypeBoth
	}

	// Convert to TorrentInfo with priority and filtering
	results := make([]types.TorrentInfo, 0, len(feed.Channel.Items))
	indexerSuccesses := make(map[int64]bool)

	for _, item := range feed.Channel.Items {
		indexerName := item.GetAttribute("indexer")
		if indexerName == "" {
			indexerName = "Prowlarr"
		}

		// Get indexer ID and settings
		indexerID := indexerNameToID[indexerName]
		settings := settingsMap[int64(indexerID)]

		// Filter by content type if settings exist
		if settings != nil && contentTypeFilter != ContentTypeBoth {
			if settings.ContentType != ContentTypeBoth && settings.ContentType != contentTypeFilter {
				continue
			}
		}

		info := item.ToTorrentInfo(indexerName)
		info.IndexerID = int64(indexerID)

		// Set priority from settings (default 25 if no settings)
		if settings != nil {
			info.IndexerPriority = settings.Priority
		} else {
			info.IndexerPriority = 25
		}

		results = append(results, info)

		// Track success for indexers with settings
		if indexerID > 0 {
			indexerSuccesses[int64(indexerID)] = true
		}
	}

	// Record successes for indexers that returned results
	for indexerID := range indexerSuccesses {
		s.RecordIndexerSuccess(ctx, indexerID)
	}

	// Sort by priority (lower priority number = preferred)
	s.sortResultsByPriority(results)

	// Cache results
	s.setSearchCache(cacheKey, results)

	s.logger.Debug().
		Str("type", criteria.Type).
		Str("query", criteria.Query).
		Int("results", len(results)).
		Msg("search completed")

	return results, nil
}

// sortResultsByPriority sorts results so lower priority numbers come first.
func (s *Service) sortResultsByPriority(results []types.TorrentInfo) {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].IndexerPriority < results[i].IndexerPriority {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// buildSearchRequestWithSettings constructs a SearchRequest using per-indexer settings.
func (s *Service) buildSearchRequestWithSettings(criteria types.SearchCriteria, config *Config, settingsMap map[int64]*IndexerSettings, indexerNameToID map[string]int) SearchRequest {
	req := SearchRequest{
		Query:   criteria.Query,
		Type:    criteria.Type,
		ImdbID:  criteria.ImdbID,
		TmdbID:  criteria.TmdbID,
		TvdbID:  criteria.TvdbID,
		Season:  criteria.Season,
		Episode: criteria.Episode,
		Limit:   criteria.Limit,
		Offset:  criteria.Offset,
	}

	// Use criteria categories if provided, otherwise build from settings + config defaults
	if len(criteria.Categories) > 0 {
		req.Categories = criteria.Categories
	} else {
		// Collect categories from all indexers with settings, plus global defaults
		categorySet := make(map[int]bool)

		// Add global default categories
		var defaultCats []int
		switch criteria.Type {
		case "movie":
			defaultCats = config.MovieCategories
		case "tvsearch":
			defaultCats = config.TVCategories
		}
		for _, cat := range defaultCats {
			categorySet[cat] = true
		}

		// Add per-indexer categories (they might be more specific)
		for _, settings := range settingsMap {
			var indexerCats []int
			switch criteria.Type {
			case "movie":
				indexerCats = settings.MovieCategories
			case "tvsearch":
				indexerCats = settings.TVCategories
			}
			for _, cat := range indexerCats {
				categorySet[cat] = true
			}
		}

		req.Categories = make([]int, 0, len(categorySet))
		for cat := range categorySet {
			req.Categories = append(req.Categories, cat)
		}
	}

	return req
}

// buildSearchRequest constructs a SearchRequest from SearchCriteria.
func (s *Service) buildSearchRequest(criteria types.SearchCriteria, config *Config) SearchRequest {
	req := SearchRequest{
		Query:   criteria.Query,
		Type:    criteria.Type,
		ImdbID:  criteria.ImdbID,
		TmdbID:  criteria.TmdbID,
		TvdbID:  criteria.TvdbID,
		Season:  criteria.Season,
		Episode: criteria.Episode,
		Limit:   criteria.Limit,
		Offset:  criteria.Offset,
	}

	// Use criteria categories if provided, otherwise use config defaults
	if len(criteria.Categories) > 0 {
		req.Categories = criteria.Categories
	} else {
		switch criteria.Type {
		case "movie":
			req.Categories = config.MovieCategories
		case "tvsearch":
			req.Categories = config.TVCategories
		}
	}

	return req
}

// Download retrieves a torrent/NZB file through Prowlarr.
func (s *Service) Download(ctx context.Context, downloadURL string) ([]byte, error) {
	client, err := s.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	s.rateLimiter.Wait()

	data, err := client.Download(ctx, downloadURL)
	if err != nil {
		s.rateLimiter.RecordError()
		return nil, fmt.Errorf("download failed: %w", err)
	}

	s.rateLimiter.RecordSuccess()

	return data, nil
}

// ClearSearchCache clears the search result cache.
func (s *Service) ClearSearchCache() {
	s.searchCacheMu.Lock()
	defer s.searchCacheMu.Unlock()
	s.searchCache = make(map[string]*searchCacheEntry)
	s.logger.Debug().Msg("search cache cleared")
}

// searchCacheKey generates a cache key for search criteria.
func (s *Service) searchCacheKey(criteria types.SearchCriteria) string {
	data, _ := json.Marshal(criteria)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8])
}

// getFromSearchCache retrieves cached search results.
func (s *Service) getFromSearchCache(key string) []types.TorrentInfo {
	s.searchCacheMu.RLock()
	defer s.searchCacheMu.RUnlock()

	entry, ok := s.searchCache[key]
	if !ok {
		return nil
	}

	if time.Since(entry.fetchedAt) > searchCacheTTL {
		return nil
	}

	return entry.results
}

// setSearchCache stores search results in cache.
func (s *Service) setSearchCache(key string, results []types.TorrentInfo) {
	s.searchCacheMu.Lock()
	defer s.searchCacheMu.Unlock()

	s.searchCache[key] = &searchCacheEntry{
		results:   results,
		fetchedAt: time.Now(),
	}
}

// rowToConfig converts a database row to Config.
func (s *Service) rowToConfig(row *sqlc.ProwlarrConfig) *Config {
	config := &Config{
		ID:            row.ID,
		Enabled:       row.Enabled != 0,
		URL:           row.Url,
		APIKey:        row.ApiKey,
		Timeout:       int(row.Timeout),
		SkipSSLVerify: row.SkipSslVerify != 0,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}

	// Parse movie categories
	if row.MovieCategories != "" {
		_ = json.Unmarshal([]byte(row.MovieCategories), &config.MovieCategories)
	}
	if len(config.MovieCategories) == 0 {
		config.MovieCategories = DefaultMovieCategories()
	}

	// Parse TV categories
	if row.TvCategories != "" {
		_ = json.Unmarshal([]byte(row.TvCategories), &config.TVCategories)
	}
	if len(config.TVCategories) == 0 {
		config.TVCategories = DefaultTVCategories()
	}

	// Parse capabilities
	if row.Capabilities.Valid && row.Capabilities.String != "" {
		var caps Capabilities
		if err := json.Unmarshal([]byte(row.Capabilities.String), &caps); err == nil {
			config.Capabilities = &caps
		}
	}

	if row.CapabilitiesUpdatedAt.Valid {
		config.CapabilitiesUpdatedAt = &row.CapabilitiesUpdatedAt.Time
	}

	return config
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// GetIndexerSettings returns settings for a specific Prowlarr indexer.
func (s *Service) GetIndexerSettings(ctx context.Context, indexerID int64) (*IndexerSettings, error) {
	row, err := s.queries.GetProwlarrIndexerSettings(ctx, indexerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get indexer settings: %w", err)
	}
	return s.rowToIndexerSettings(row), nil
}

// GetAllIndexerSettings returns all stored indexer settings.
func (s *Service) GetAllIndexerSettings(ctx context.Context) ([]IndexerSettings, error) {
	rows, err := s.queries.GetAllProwlarrIndexerSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all indexer settings: %w", err)
	}

	settings := make([]IndexerSettings, 0, len(rows))
	for _, row := range rows {
		settings = append(settings, *s.rowToIndexerSettings(row))
	}
	return settings, nil
}

// UpdateIndexerSettings creates or updates settings for a Prowlarr indexer.
func (s *Service) UpdateIndexerSettings(ctx context.Context, indexerID int64, input IndexerSettingsInput) (*IndexerSettings, error) {
	if input.Priority < 1 {
		input.Priority = 1
	} else if input.Priority > 50 {
		input.Priority = 50
	}

	if input.ContentType == "" {
		input.ContentType = ContentTypeBoth
	}

	var movieCats, tvCats sql.NullString
	if len(input.MovieCategories) > 0 {
		data, _ := json.Marshal(input.MovieCategories)
		movieCats = sql.NullString{String: string(data), Valid: true}
	}
	if len(input.TVCategories) > 0 {
		data, _ := json.Marshal(input.TVCategories)
		tvCats = sql.NullString{String: string(data), Valid: true}
	}

	row, err := s.queries.UpsertProwlarrIndexerSettings(ctx, sqlc.UpsertProwlarrIndexerSettingsParams{
		ProwlarrIndexerID: indexerID,
		Priority:          int64(input.Priority),
		ContentType:       string(input.ContentType),
		MovieCategories:   movieCats,
		TvCategories:      tvCats,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update indexer settings: %w", err)
	}

	s.logger.Info().
		Int64("indexerId", indexerID).
		Int("priority", input.Priority).
		Str("contentType", string(input.ContentType)).
		Msg("indexer settings updated")

	return s.rowToIndexerSettings(row), nil
}

// DeleteIndexerSettings removes settings for a Prowlarr indexer.
func (s *Service) DeleteIndexerSettings(ctx context.Context, indexerID int64) error {
	if err := s.queries.DeleteProwlarrIndexerSettings(ctx, indexerID); err != nil {
		return fmt.Errorf("failed to delete indexer settings: %w", err)
	}
	return nil
}

// GetIndexersWithSettings returns all Prowlarr indexers with their SlipStream settings.
func (s *Service) GetIndexersWithSettings(ctx context.Context) ([]IndexerWithSettings, error) {
	indexers, err := s.GetIndexers(ctx)
	if err != nil {
		return nil, err
	}

	allSettings, err := s.GetAllIndexerSettings(ctx)
	if err != nil {
		return nil, err
	}

	settingsMap := make(map[int64]*IndexerSettings)
	for i := range allSettings {
		settingsMap[allSettings[i].ProwlarrIndexerID] = &allSettings[i]
	}

	result := make([]IndexerWithSettings, 0, len(indexers))
	for _, idx := range indexers {
		iws := IndexerWithSettings{Indexer: idx}
		if settings, ok := settingsMap[int64(idx.ID)]; ok {
			iws.Settings = settings
		}
		result = append(result, iws)
	}

	return result, nil
}

// RecordIndexerSuccess records a successful search for an indexer.
func (s *Service) RecordIndexerSuccess(ctx context.Context, indexerID int64) {
	if err := s.queries.RecordProwlarrIndexerSuccess(ctx, indexerID); err != nil {
		s.logger.Debug().
			Err(err).
			Int64("indexerId", indexerID).
			Msg("failed to record indexer success (settings may not exist)")
	}
}

// RecordIndexerFailure records a failed search for an indexer.
func (s *Service) RecordIndexerFailure(ctx context.Context, indexerID int64, reason string) {
	if err := s.queries.RecordProwlarrIndexerFailure(ctx, sqlc.RecordProwlarrIndexerFailureParams{
		ProwlarrIndexerID: indexerID,
		LastFailureReason: sql.NullString{String: reason, Valid: reason != ""},
	}); err != nil {
		s.logger.Debug().
			Err(err).
			Int64("indexerId", indexerID).
			Msg("failed to record indexer failure (settings may not exist)")
	}
}

// ResetIndexerStats resets the success/failure counts for an indexer.
func (s *Service) ResetIndexerStats(ctx context.Context, indexerID int64) error {
	if err := s.queries.ResetProwlarrIndexerStats(ctx, indexerID); err != nil {
		return fmt.Errorf("failed to reset indexer stats: %w", err)
	}
	return nil
}

// rowToIndexerSettings converts a database row to IndexerSettings.
func (s *Service) rowToIndexerSettings(row *sqlc.ProwlarrIndexerSetting) *IndexerSettings {
	settings := &IndexerSettings{
		ProwlarrIndexerID: row.ProwlarrIndexerID,
		Priority:          int(row.Priority),
		ContentType:       ContentType(row.ContentType),
		SuccessCount:      row.SuccessCount,
		FailureCount:      row.FailureCount,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}

	if row.MovieCategories.Valid && row.MovieCategories.String != "" {
		_ = json.Unmarshal([]byte(row.MovieCategories.String), &settings.MovieCategories)
	}
	if row.TvCategories.Valid && row.TvCategories.String != "" {
		_ = json.Unmarshal([]byte(row.TvCategories.String), &settings.TVCategories)
	}
	if row.LastFailureAt.Valid {
		settings.LastFailureAt = &row.LastFailureAt.Time
	}
	if row.LastFailureReason.Valid {
		settings.LastFailureReason = row.LastFailureReason.String
	}

	return settings
}

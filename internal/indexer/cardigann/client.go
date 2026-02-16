package cardigann

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/indexer/types"
)

// CookieStore provides persistent cookie storage for indexer sessions.
type CookieStore interface {
	// GetCookies retrieves cached cookies for an indexer. Returns nil if none exist or expired.
	GetCookies(ctx context.Context, indexerID int64) (cookies string, err error)
	// SaveCookies stores cookies for an indexer with the given expiration.
	SaveCookies(ctx context.Context, indexerID int64, cookies string, expiresAt time.Time) error
	// ClearCookies removes cached cookies for an indexer.
	ClearCookies(ctx context.Context, indexerID int64) error
}

// Client implements the indexer.Indexer interface using a Cardigann definition.
type Client struct {
	def           *Definition
	indexerDef    *types.IndexerDefinition
	settings      map[string]string
	loginHandler  *LoginHandler
	searchEngine  *SearchEngine
	httpClient    *http.Client
	logger        *zerolog.Logger
	cookieStore   CookieStore
	authenticated bool
	lastLogin     time.Time
}

// ClientConfig contains configuration options for creating a new Client.
type ClientConfig struct {
	Definition  *Definition
	IndexerDef  *types.IndexerDefinition
	Settings    map[string]string
	Logger      *zerolog.Logger
	CookieStore CookieStore // Optional: provides persistent cookie storage
}

// NewClient creates a new Cardigann client.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Definition == nil {
		return nil, fmt.Errorf("definition is required")
	}

	baseURL := cfg.Definition.GetBaseURL()
	if baseURL == "" {
		return nil, fmt.Errorf("definition has no base URL")
	}

	logger := cfg.Logger.With().
		Str("component", "cardigann").
		Str("indexer", cfg.Definition.ID).
		Logger()

	// Create login handler
	loginHandler, err := NewLoginHandler(baseURL, &logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create login handler: %w", err)
	}

	// Create search engine using the authenticated HTTP client
	searchEngine := NewSearchEngine(cfg.Definition, loginHandler.GetHTTPClient(), &logger)

	return &Client{
		def:          cfg.Definition,
		indexerDef:   cfg.IndexerDef,
		settings:     cfg.Settings,
		loginHandler: loginHandler,
		searchEngine: searchEngine,
		httpClient:   loginHandler.GetHTTPClient(),
		logger:       &logger,
		cookieStore:  cfg.CookieStore,
	}, nil
}

// Name returns the configured name of the indexer.
func (c *Client) Name() string {
	if c.indexerDef != nil && c.indexerDef.Name != "" {
		return c.indexerDef.Name
	}
	return c.def.Name
}

// Definition returns the indexer definition.
func (c *Client) Definition() *types.IndexerDefinition {
	if c.indexerDef != nil {
		return c.indexerDef
	}

	// Build a definition from the Cardigann definition
	return &types.IndexerDefinition{
		Name:           c.def.Name,
		Protocol:       types.Protocol(c.def.GetProtocol()),
		Privacy:        types.Privacy(c.def.GetPrivacy()),
		SupportsSearch: c.def.SupportsSearch("search"),
		SupportsMovies: c.def.SupportsSearch("movie-search"),
		SupportsTV:     c.def.SupportsSearch("tv-search"),
		SupportsRSS:    true, // Most indexers support RSS-style browsing
	}
}

// Test verifies that the indexer is configured correctly and accessible.
func (c *Client) Test(ctx context.Context) error {
	// Authenticate if required
	if err := c.ensureAuthenticated(ctx); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Run the test check
	if err := c.loginHandler.Test(ctx, c.def.Login); err != nil {
		return fmt.Errorf("test failed: %w", err)
	}

	c.logger.Info().Msg("Indexer test passed")
	return nil
}

// Search executes a search query and returns results.
func (c *Client) Search(ctx context.Context, criteria *types.SearchCriteria) ([]types.ReleaseInfo, error) {
	// Ensure we're authenticated
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Convert criteria to search query
	query := c.buildSearchQuery(criteria)

	// Execute search
	results, err := c.searchEngine.Search(ctx, &query, c.settings)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results to ReleaseInfo
	releases := make([]types.ReleaseInfo, 0, len(results))
	for i := range results {
		release := c.convertToReleaseInfo(&results[i])
		releases = append(releases, release)
	}

	c.logger.Debug().
		Int("results", len(releases)).
		Str("query", criteria.Query).
		Msg("Search completed")

	return releases, nil
}

// SearchTorrents executes a search and returns torrent-specific results.
func (c *Client) SearchTorrents(ctx context.Context, criteria *types.SearchCriteria) ([]types.TorrentInfo, error) {
	// Ensure we're authenticated
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Convert criteria to search query
	query := c.buildSearchQuery(criteria)

	c.logger.Info().
		Str("originalQuery", criteria.Query).
		Str("enhancedQuery", query.Query).
		Str("type", criteria.Type).
		Int("season", criteria.Season).
		Int("episode", criteria.Episode).
		Int("year", criteria.Year).
		Msg("Executing search with enhanced query")

	// Execute search
	results, err := c.searchEngine.Search(ctx, &query, c.settings)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results to TorrentInfo
	torrents := make([]types.TorrentInfo, 0, len(results))
	for i := range results {
		torrent := c.convertToTorrentInfo(&results[i])
		torrents = append(torrents, torrent)
	}

	c.logger.Info().
		Int("rawResults", len(torrents)).
		Str("enhancedQuery", query.Query).
		Msg("Indexer returned raw results")

	return torrents, nil
}

// Download retrieves the torrent/nzb file from the given URL.
func (c *Client) Download(ctx context.Context, url string) ([]byte, error) {
	// Ensure we're authenticated
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Handle download block if defined
	if c.def.Download != nil && c.def.Download.Before != nil {
		// Execute pre-download request
		if err := c.executePreDownload(ctx); err != nil {
			c.logger.Warn().Err(err).Msg("Pre-download request failed")
		}
	}

	// Create download request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read download: %w", err)
	}

	c.logger.Debug().
		Int("size", len(data)).
		Str("url", url).
		Msg("Download completed")

	return data, nil
}

// Capabilities returns the indexer's capabilities.
func (c *Client) Capabilities() *types.Capabilities {
	caps := &types.Capabilities{
		SupportsSearch: c.def.SupportsSearch("search"),
		SupportsTV:     c.def.SupportsSearch("tv-search"),
		SupportsMovies: c.def.SupportsSearch("movie-search"),
		SupportsRSS:    true,
	}

	// Extract supported parameters for each search mode
	if params, ok := c.def.Caps.Modes["search"]; ok {
		caps.SearchParams = params
	}
	if params, ok := c.def.Caps.Modes["tv-search"]; ok {
		caps.TvSearchParams = params
	}
	if params, ok := c.def.Caps.Modes["movie-search"]; ok {
		caps.MovieSearchParams = params
	}

	// Convert category mappings
	caps.Categories = make([]types.CategoryMapping, 0, len(c.def.Caps.CategoryMappings))
	for _, cat := range c.def.Caps.CategoryMappings {
		caps.Categories = append(caps.Categories, types.CategoryMapping{
			ID:          parseIntSafe(cat.ID),
			Name:        cat.Cat,
			Description: cat.Desc,
		})
	}

	return caps
}

// SupportsSearch returns true if the indexer supports general search.
func (c *Client) SupportsSearch() bool {
	return c.def.SupportsSearch("search")
}

// SupportsRSS returns true if the indexer supports RSS-style browsing.
func (c *Client) SupportsRSS() bool {
	// Most indexers support browsing without a query
	return true
}

// GetDefinition returns the underlying Cardigann definition.
func (c *Client) GetDefinition() *Definition {
	return c.def
}

// GetSettings returns the current settings.
func (c *Client) GetSettings() map[string]string {
	return c.settings
}

// SetSettings updates the client settings.
func (c *Client) SetSettings(settings map[string]string) {
	c.settings = settings
	c.authenticated = false // Force re-authentication
}

// ensureAuthenticated performs authentication if needed.
func (c *Client) ensureAuthenticated(ctx context.Context) error {
	// No authentication required
	if !c.def.HasLogin() {
		return nil
	}

	// Already authenticated recently (within 30 minutes)
	if c.authenticated && time.Since(c.lastLogin) < 30*time.Minute {
		return nil
	}

	// Try to use cached cookies first
	if c.tryUseCachedCookies(ctx) {
		return nil
	}

	// Perform fresh authentication
	// Pass search headers as fallback for login methods that need them (e.g., "get" method with API key auth)
	var searchHeaders map[string]StringOrArray
	if c.def.Search.Headers != nil {
		searchHeaders = c.def.Search.Headers
	}
	if err := c.loginHandler.Authenticate(ctx, c.def.Login, c.settings, searchHeaders); err != nil {
		return err
	}

	c.authenticated = true
	c.lastLogin = time.Now()

	// Save cookies for future use
	c.saveCookiesToStore(ctx)

	c.logger.Debug().Msg("Authentication successful")

	return nil
}

// tryUseCachedCookies attempts to restore and validate cached session cookies.
// Returns true if cached cookies are valid and can be used.
func (c *Client) tryUseCachedCookies(ctx context.Context) bool {
	if c.cookieStore == nil || c.indexerDef == nil {
		return false
	}

	// Try to load cached cookies
	cookies, err := c.cookieStore.GetCookies(ctx, c.indexerDef.ID)
	if err != nil {
		c.logger.Debug().Err(err).Msg("Failed to load cached cookies")
		return false
	}

	if cookies == "" {
		return false
	}

	// Import the cached cookies into the login handler
	c.loginHandler.ImportCookies(cookies)

	// Test if cookies are still valid
	if err := c.loginHandler.Test(ctx, c.def.Login); err != nil {
		c.logger.Debug().Err(err).Msg("Cached cookies are invalid, will re-authenticate")
		// Clear invalid cookies
		_ = c.cookieStore.ClearCookies(ctx, c.indexerDef.ID)
		return false
	}

	c.authenticated = true
	c.lastLogin = time.Now()

	c.logger.Info().Msg("Using cached session cookies")

	return true
}

// saveCookiesToStore persists current session cookies.
func (c *Client) saveCookiesToStore(ctx context.Context) {
	if c.cookieStore == nil || c.indexerDef == nil {
		return
	}

	cookies := c.loginHandler.ExportCookies()
	if cookies == "" {
		return
	}

	// Set expiration to 30 days from now (similar to Prowlarr)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	if err := c.cookieStore.SaveCookies(ctx, c.indexerDef.ID, cookies, expiresAt); err != nil {
		c.logger.Warn().Err(err).Msg("Failed to save session cookies")
	} else {
		c.logger.Debug().Msg("Saved session cookies for future use")
	}
}

// buildSearchQuery converts SearchCriteria to a cardigann SearchQuery.
func (c *Client) buildSearchQuery(criteria *types.SearchCriteria) SearchQuery {
	query := SearchQuery{
		Query:   buildEnhancedQueryKeywords(criteria),
		Type:    criteria.Type,
		Year:    criteria.Year,
		Season:  criteria.Season,
		Episode: criteria.Episode,
		IMDBID:  criteria.ImdbID,
		TMDBID:  criteria.TmdbID,
		TVDBID:  criteria.TvdbID,
		Limit:   criteria.Limit,
		Offset:  criteria.Offset,
	}

	// Convert category IDs to strings for matching
	if len(criteria.Categories) > 0 {
		query.Categories = make([]string, len(criteria.Categories))
		for i, cat := range criteria.Categories {
			query.Categories[i] = c.mapCategoryToString(cat)
		}
	}

	return query
}

// buildEnhancedQueryKeywords enhances the search query with season/episode for TV
// or year for movies to improve indexer-side filtering.
func buildEnhancedQueryKeywords(criteria *types.SearchCriteria) string {
	if criteria.Query == "" {
		return ""
	}

	switch criteria.Type {
	case "tvsearch":
		if criteria.Season > 0 {
			if criteria.Episode > 0 {
				return fmt.Sprintf("%s S%02dE%02d", criteria.Query, criteria.Season, criteria.Episode)
			}
			return fmt.Sprintf("%s S%02d", criteria.Query, criteria.Season)
		}
	case "movie":
		if criteria.Year > 0 {
			return fmt.Sprintf("%s %d", criteria.Query, criteria.Year)
		}
	}

	return criteria.Query
}

// mapCategoryToString maps a numeric category ID to the indexer's category string.
func (c *Client) mapCategoryToString(catID int) string {
	// Look up in category mappings
	for _, mapping := range c.def.Caps.CategoryMappings {
		if id := parseIntSafe(mapping.ID); id == catID {
			return mapping.Cat
		}
	}
	// Fallback to string representation
	return strconv.Itoa(catID)
}

// convertToReleaseInfo converts a cardigann SearchResult to types.ReleaseInfo.
func (c *Client) convertToReleaseInfo(result *SearchResult) types.ReleaseInfo {
	release := types.ReleaseInfo{
		GUID:        result.GUID,
		Title:       result.Title,
		DownloadURL: result.DownloadURL,
		InfoURL:     result.InfoURL,
		Size:        result.Size,
		PublishDate: result.PublishDate,
		Protocol:    types.Protocol(c.def.GetProtocol()),
	}

	// Set indexer info
	if c.indexerDef != nil {
		release.IndexerID = c.indexerDef.ID
		release.IndexerName = c.indexerDef.Name
	} else {
		release.IndexerName = c.def.Name
	}

	// Parse category
	if result.CategoryID > 0 {
		release.Categories = []int{result.CategoryID}
	}

	// External IDs
	if result.IMDBID != "" {
		release.ImdbID = parseIMDBID(result.IMDBID)
	}
	release.TmdbID = result.TMDBID
	release.TvdbID = result.TVDBID

	return release
}

// convertToTorrentInfo converts a cardigann SearchResult to types.TorrentInfo.
func (c *Client) convertToTorrentInfo(result *SearchResult) types.TorrentInfo {
	torrent := types.TorrentInfo{
		ReleaseInfo:          c.convertToReleaseInfo(result),
		Seeders:              result.Seeders,
		Leechers:             result.Leechers,
		InfoHash:             result.InfoHash,
		MagnetURL:            result.MagnetURL,
		MinimumRatio:         result.MinimumRatio,
		MinimumSeedTime:      result.MinimumSeedTime,
		DownloadVolumeFactor: result.DownloadVolumeFactor,
		UploadVolumeFactor:   result.UploadVolumeFactor,
	}

	return torrent
}

// executePreDownload executes a pre-download request if configured.
func (c *Client) executePreDownload(ctx context.Context) error {
	before := c.def.Download.Before
	if before == nil {
		return nil
	}

	// Build the pre-download URL
	preURL := c.def.GetBaseURL() + before.Path

	// Create template context
	tmplCtx := NewTemplateContext()
	tmplCtx.Config = c.settings
	// TODO: Add download URL info to context

	engine := NewTemplateEngine()

	// Evaluate path template
	evaluatedPath, err := engine.Evaluate(before.Path, tmplCtx)
	if err == nil {
		preURL = c.def.GetBaseURL() + evaluatedPath
	}

	// Create request
	method := before.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), preURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create pre-download request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	// Add custom headers
	for key, val := range before.Headers {
		evaluated, _ := engine.Evaluate(string(val), tmplCtx)
		req.Header.Set(key, evaluated)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("pre-download request failed: %w", err)
	}
	defer resp.Body.Close()

	// Drain body
	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}

// Helper functions

func parseIntSafe(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func parseIMDBID(s string) int {
	// Remove "tt" prefix if present
	s = strings.TrimPrefix(s, "tt")
	n, _ := strconv.Atoi(s)
	return n
}

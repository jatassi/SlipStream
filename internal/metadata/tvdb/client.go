package tvdb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
)

var (
	ErrAPIKeyMissing  = errors.New("TVDB API key is not configured")
	ErrSeriesNotFound = errors.New("series not found")
	ErrAPIError       = errors.New("TVDB API error")
	ErrAuthFailed     = errors.New("TVDB authentication failed")
	ErrRateLimited    = errors.New("TVDB API rate limited")
)

// Client is a TVDB API client.
type Client struct {
	httpClient *http.Client
	config     config.TVDBConfig
	logger     zerolog.Logger

	// Token management
	mu          sync.RWMutex
	token       string
	tokenExpiry time.Time
}

// NewClient creates a new TVDB client.
func NewClient(cfg config.TVDBConfig, logger zerolog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		config: cfg,
		logger: logger.With().Str("component", "tvdb").Logger(),
	}
}

// Name returns the provider name.
func (c *Client) Name() string {
	return "tvdb"
}

// IsConfigured returns true if the API key is set.
func (c *Client) IsConfigured() bool {
	return c.config.APIKey != ""
}

// authenticate gets or refreshes the authentication token.
func (c *Client) authenticate(ctx context.Context) error {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		c.mu.RUnlock()
		return nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.token != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	loginURL := fmt.Sprintf("%s/login", c.config.BaseURL)
	loginReq := LoginRequest{APIKey: c.config.APIKey}

	body, err := json.Marshal(loginReq)
	if err != nil {
		return fmt.Errorf("failed to marshal login request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Error().Int("status", resp.StatusCode).Msg("TVDB authentication failed")
		return ErrAuthFailed
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to decode login response: %w", err)
	}

	c.token = loginResp.Data.Token
	// TVDB tokens expire after 30 days, but we refresh after 24 hours to be safe
	c.tokenExpiry = time.Now().Add(24 * time.Hour)

	c.logger.Debug().Msg("TVDB authentication successful")
	return nil
}

// SearchMovies searches for movies (TVDB is TV-focused, returns empty).
func (c *Client) SearchMovies(ctx context.Context, query string) ([]NormalizedMovieResult, error) {
	// TVDB is primarily for TV shows, not movies
	return []NormalizedMovieResult{}, nil
}

// GetMovie gets movie details (TVDB is TV-focused, returns not found).
func (c *Client) GetMovie(ctx context.Context, id int) (*NormalizedMovieResult, error) {
	// TVDB is primarily for TV shows, not movies
	return nil, errors.New("TVDB does not support movies")
}

// SearchSeries searches for TV series by query.
func (c *Client) SearchSeries(ctx context.Context, query string) ([]NormalizedSeriesResult, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	if err := c.authenticate(ctx); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/search", c.config.BaseURL)
	params := url.Values{}
	params.Set("query", query)
	params.Set("type", "series")

	var response SearchResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		return nil, err
	}

	results := make([]NormalizedSeriesResult, 0, len(response.Data))
	for _, item := range response.Data {
		if item.Type == "series" {
			results = append(results, c.searchResultToSeries(item))
		}
	}

	c.logger.Debug().
		Str("query", query).
		Int("results", len(results)).
		Msg("TV search completed")

	return results, nil
}

// GetSeries gets detailed TV series info by TVDB ID.
func (c *Client) GetSeries(ctx context.Context, id int) (*NormalizedSeriesResult, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	if err := c.authenticate(ctx); err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/series/%d/extended", c.config.BaseURL, id)

	var response SeriesResponse
	if err := c.doRequest(ctx, endpoint, nil, &response); err != nil {
		return nil, err
	}

	result := c.seriesDetailToResult(response.Data)

	c.logger.Debug().
		Int("id", id).
		Str("title", result.Title).
		Msg("Got series details")

	return &result, nil
}

// doRequest performs an HTTP GET request with authentication.
func (c *Client) doRequest(ctx context.Context, endpoint string, params url.Values, result interface{}) error {
	reqURL := endpoint
	if len(params) > 0 {
		reqURL = fmt.Sprintf("%s?%s", endpoint, params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error().Err(err).Str("url", endpoint).Msg("HTTP request failed")
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusNotFound:
			return ErrSeriesNotFound
		case http.StatusUnauthorized:
			// Token might be expired, clear it
			c.mu.Lock()
			c.token = ""
			c.mu.Unlock()
			return fmt.Errorf("%w: unauthorized", ErrAPIError)
		case http.StatusTooManyRequests:
			return ErrRateLimited
		default:
			return fmt.Errorf("%w: status %d", ErrAPIError, resp.StatusCode)
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// searchResultToSeries converts a TVDB search result to a NormalizedSeriesResult.
func (c *Client) searchResultToSeries(item SearchResult) NormalizedSeriesResult {
	year := 0
	if item.Year != "" {
		year, _ = strconv.Atoi(item.Year)
	}

	// Parse TVDB ID from the string
	tvdbID := 0
	if item.TvdbID != "" {
		tvdbID, _ = strconv.Atoi(item.TvdbID)
	}

	// Get overview, preferring English
	overview := item.Overview
	if overview == "" && item.Overviews != nil {
		if eng, ok := item.Overviews["eng"]; ok {
			overview = eng
		}
	}

	// Extract IMDB ID from remote IDs
	imdbID := ""
	for _, rid := range item.RemoteIDs {
		if rid.SourceName == "IMDB" {
			imdbID = rid.ID
			break
		}
	}

	// Map status
	status := "continuing"
	switch item.Status {
	case "Ended":
		status = "ended"
	case "Upcoming":
		status = "upcoming"
	}

	return NormalizedSeriesResult{
		ID:        tvdbID,
		TvdbID:    tvdbID,
		Title:     item.Name,
		Year:      year,
		Overview:  overview,
		PosterURL: item.ImageURL,
		ImdbID:    imdbID,
		Status:    status,
	}
}

// seriesDetailToResult converts a TVDB series detail to a NormalizedSeriesResult.
func (c *Client) seriesDetailToResult(detail SeriesDetail) NormalizedSeriesResult {
	year := 0
	if detail.Year != "" {
		year, _ = strconv.Atoi(detail.Year)
	}

	genres := make([]string, len(detail.Genres))
	for i, g := range detail.Genres {
		genres[i] = g.Name
	}

	// Map status
	status := "continuing"
	switch detail.Status.Name {
	case "Ended":
		status = "ended"
	case "Upcoming":
		status = "upcoming"
	}

	// Extract IMDB and TMDB IDs from remote IDs
	imdbID := ""
	tmdbID := 0
	for _, rid := range detail.RemoteIDs {
		switch rid.SourceName {
		case "IMDB":
			imdbID = rid.ID
		case "TheMovieDB.com":
			tmdbID, _ = strconv.Atoi(rid.ID)
		}
	}

	// Find poster artwork
	posterURL := detail.Image
	backdropURL := ""
	for _, art := range detail.Artworks {
		// Type 1 = poster, Type 3 = background
		if art.Type == 1 && posterURL == "" {
			posterURL = art.Image
		}
		if art.Type == 3 && backdropURL == "" {
			backdropURL = art.Image
		}
	}

	return NormalizedSeriesResult{
		ID:          detail.ID,
		TvdbID:      detail.ID,
		TmdbID:      tmdbID,
		Title:       detail.Name,
		Year:        year,
		Overview:    detail.Overview,
		PosterURL:   posterURL,
		BackdropURL: backdropURL,
		ImdbID:      imdbID,
		Genres:      genres,
		Status:      status,
		Runtime:     detail.AverageRuntime,
	}
}

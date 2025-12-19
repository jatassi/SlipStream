package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
)

var (
	ErrAPIKeyMissing  = errors.New("TMDB API key is not configured")
	ErrMovieNotFound  = errors.New("movie not found")
	ErrSeriesNotFound = errors.New("series not found")
	ErrAPIError       = errors.New("TMDB API error")
	ErrRateLimited    = errors.New("TMDB API rate limited")
)

// Client is a TMDB API client.
type Client struct {
	httpClient *http.Client
	config     config.TMDBConfig
	logger     zerolog.Logger
}

// NewClient creates a new TMDB client.
func NewClient(cfg config.TMDBConfig, logger zerolog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		config: cfg,
		logger: logger.With().Str("component", "tmdb").Logger(),
	}
}

// Name returns the provider name.
func (c *Client) Name() string {
	return "tmdb"
}

// IsConfigured returns true if the API key is set.
func (c *Client) IsConfigured() bool {
	return c.config.APIKey != ""
}

// SearchMovies searches for movies by query.
func (c *Client) SearchMovies(ctx context.Context, query string) ([]NormalizedMovieResult, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/search/movie", c.config.BaseURL)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)
	params.Set("query", query)
	params.Set("include_adult", "false")

	var response SearchMoviesResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		return nil, err
	}

	results := make([]NormalizedMovieResult, len(response.Results))
	for i, movie := range response.Results {
		results[i] = c.toMovieResult(movie)
	}

	c.logger.Debug().
		Str("query", query).
		Int("results", len(results)).
		Msg("Movie search completed")

	return results, nil
}

// GetMovie gets detailed movie info by TMDB ID.
func (c *Client) GetMovie(ctx context.Context, id int) (*NormalizedMovieResult, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/movie/%d", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)
	params.Set("append_to_response", "external_ids")

	var details MovieDetails
	if err := c.doRequest(ctx, endpoint, params, &details); err != nil {
		return nil, err
	}

	result := c.movieDetailsToResult(details)

	c.logger.Debug().
		Int("id", id).
		Str("title", result.Title).
		Msg("Got movie details")

	return &result, nil
}

// SearchSeries searches for TV series by query.
func (c *Client) SearchSeries(ctx context.Context, query string) ([]NormalizedSeriesResult, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/search/tv", c.config.BaseURL)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)
	params.Set("query", query)

	var response SearchTVResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		return nil, err
	}

	results := make([]NormalizedSeriesResult, len(response.Results))
	for i, tv := range response.Results {
		results[i] = c.toSeriesResult(tv)
	}

	c.logger.Debug().
		Str("query", query).
		Int("results", len(results)).
		Msg("TV search completed")

	return results, nil
}

// GetSeries gets detailed TV series info by TMDB ID.
func (c *Client) GetSeries(ctx context.Context, id int) (*NormalizedSeriesResult, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/tv/%d", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)
	params.Set("append_to_response", "external_ids")

	var details TVDetails
	if err := c.doRequest(ctx, endpoint, params, &details); err != nil {
		return nil, err
	}

	result := c.tvDetailsToResult(details)

	c.logger.Debug().
		Int("id", id).
		Str("title", result.Title).
		Msg("Got TV series details")

	return &result, nil
}

// GetImageURL returns a full image URL for a given path and size.
// Size options: "w92", "w154", "w185", "w342", "w500", "w780", "original"
func (c *Client) GetImageURL(path string, size string) string {
	if path == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s%s", c.config.ImageBaseURL, size, path)
}

// doRequest performs an HTTP GET request and decodes the JSON response.
func (c *Client) doRequest(ctx context.Context, endpoint string, params url.Values, result interface{}) error {
	reqURL := endpoint
	if len(params) > 0 {
		reqURL = fmt.Sprintf("%s?%s", endpoint, params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error().Err(err).Str("url", endpoint).Msg("HTTP request failed")
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			c.logger.Error().
				Int("status", resp.StatusCode).
				Str("message", errResp.StatusMessage).
				Msg("TMDB API error")
		}

		switch resp.StatusCode {
		case http.StatusNotFound:
			return ErrMovieNotFound
		case http.StatusUnauthorized:
			return fmt.Errorf("%w: invalid API key", ErrAPIError)
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

// toMovieResult converts a TMDB movie search result to a NormalizedMovieResult.
func (c *Client) toMovieResult(movie MovieResult) NormalizedMovieResult {
	year := 0
	if len(movie.ReleaseDate) >= 4 {
		year, _ = strconv.Atoi(movie.ReleaseDate[:4])
	}

	result := NormalizedMovieResult{
		ID:       movie.ID,
		Title:    movie.Title,
		Year:     year,
		Overview: movie.Overview,
	}

	if movie.PosterPath != nil {
		result.PosterURL = c.GetImageURL(*movie.PosterPath, "w500")
	}
	if movie.BackdropPath != nil {
		result.BackdropURL = c.GetImageURL(*movie.BackdropPath, "w780")
	}

	return result
}

// movieDetailsToResult converts TMDB movie details to a NormalizedMovieResult.
func (c *Client) movieDetailsToResult(details MovieDetails) NormalizedMovieResult {
	year := 0
	if len(details.ReleaseDate) >= 4 {
		year, _ = strconv.Atoi(details.ReleaseDate[:4])
	}

	genres := make([]string, len(details.Genres))
	for i, g := range details.Genres {
		genres[i] = g.Name
	}

	result := NormalizedMovieResult{
		ID:       details.ID,
		Title:    details.Title,
		Year:     year,
		Overview: details.Overview,
		Runtime:  details.Runtime,
		ImdbID:   details.ImdbID,
		Genres:   genres,
	}

	if details.PosterPath != nil {
		result.PosterURL = c.GetImageURL(*details.PosterPath, "w500")
	}
	if details.BackdropPath != nil {
		result.BackdropURL = c.GetImageURL(*details.BackdropPath, "w780")
	}

	return result
}

// toSeriesResult converts a TMDB TV search result to a NormalizedSeriesResult.
func (c *Client) toSeriesResult(tv TVResult) NormalizedSeriesResult {
	year := 0
	if len(tv.FirstAirDate) >= 4 {
		year, _ = strconv.Atoi(tv.FirstAirDate[:4])
	}

	result := NormalizedSeriesResult{
		ID:       tv.ID,
		Title:    tv.Name,
		Year:     year,
		Overview: tv.Overview,
	}

	if tv.PosterPath != nil {
		result.PosterURL = c.GetImageURL(*tv.PosterPath, "w500")
	}
	if tv.BackdropPath != nil {
		result.BackdropURL = c.GetImageURL(*tv.BackdropPath, "w780")
	}

	return result
}

// tvDetailsToResult converts TMDB TV details to a NormalizedSeriesResult.
func (c *Client) tvDetailsToResult(details TVDetails) NormalizedSeriesResult {
	year := 0
	if len(details.FirstAirDate) >= 4 {
		year, _ = strconv.Atoi(details.FirstAirDate[:4])
	}

	genres := make([]string, len(details.Genres))
	for i, g := range details.Genres {
		genres[i] = g.Name
	}

	// Map TMDB status to our status format
	status := "continuing"
	switch details.Status {
	case "Ended", "Canceled":
		status = "ended"
	case "Returning Series", "In Production":
		status = "continuing"
	case "Planned":
		status = "upcoming"
	}

	result := NormalizedSeriesResult{
		ID:       details.ID,
		TmdbID:   details.ID,
		Title:    details.Name,
		Year:     year,
		Overview: details.Overview,
		Status:   status,
		Genres:   genres,
	}

	if details.PosterPath != nil {
		result.PosterURL = c.GetImageURL(*details.PosterPath, "w500")
	}
	if details.BackdropPath != nil {
		result.BackdropURL = c.GetImageURL(*details.BackdropPath, "w780")
	}

	// Get external IDs if available
	if details.ExternalIDs != nil {
		result.ImdbID = details.ExternalIDs.ImdbID
		result.TvdbID = details.ExternalIDs.TvdbID
	}

	// Get runtime from episode run time (use first if available)
	if len(details.EpisodeRunTime) > 0 {
		result.Runtime = details.EpisodeRunTime[0]
	}

	return result
}

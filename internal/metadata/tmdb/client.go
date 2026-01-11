package tmdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sort"
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

// Test verifies connectivity to the TMDB API by making a configuration request.
func (c *Client) Test(ctx context.Context) error {
	if !c.IsConfigured() {
		return ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/configuration", c.config.BaseURL)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var result struct {
		Images struct {
			BaseURL string `json:"base_url"`
		} `json:"images"`
	}

	return c.doRequest(ctx, endpoint, params, &result)
}

// SearchMovies searches for movies by query with optional year filter.
func (c *Client) SearchMovies(ctx context.Context, query string, year int) ([]NormalizedMovieResult, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/search/movie", c.config.BaseURL)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)
	params.Set("query", query)
	params.Set("include_adult", "false")
	if year > 0 {
		params.Set("year", fmt.Sprintf("%d", year))
	}

	var response SearchMoviesResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		return nil, err
	}

	// Apply relevance scoring if not disabled
	if len(response.Results) > 0 {
		if !c.config.DisableSearchOrdering {
			currentYear := time.Now().Year()

			// Find max vote count for normalization
			maxVoteCount := 0
			for _, movie := range response.Results {
				if movie.VoteCount > maxVoteCount {
					maxVoteCount = movie.VoteCount
				}
			}

			// Calculate scores for all movies
			scoreableMovies := make([]scoreableMovie, len(response.Results))
			for i, movie := range response.Results {
				scoreableMovies[i] = scoreableMovie{
					MovieResult: movie,
					Score:       c.calculateMovieScore(movie, maxVoteCount, currentYear),
				}
			}

			// Sort by score (highest first)
			sort.Slice(scoreableMovies, func(i, j int) bool {
				return scoreableMovies[i].Score > scoreableMovies[j].Score
			})

			// Convert to normalized results with scoring
			results := make([]NormalizedMovieResult, len(scoreableMovies))
			for i, sm := range scoreableMovies {
				results[i] = c.toMovieResult(sm.MovieResult)
			}

			c.logger.Debug().
				Str("query", query).
				Int("year", year).
				Int("results", len(results)).
				Msg("Movie search completed with scoring")

			return results, nil
		} else {
			// Scoring disabled, return results in original order
			c.logger.Debug().
				Str("query", query).
				Int("year", year).
				Bool("scoring_disabled", true).
				Msg("Movie search completed without scoring")

			results := make([]NormalizedMovieResult, len(response.Results))
			for i, movie := range response.Results {
				results[i] = c.toMovieResult(movie)
			}
			return results, nil
		}
	}

	results := make([]NormalizedMovieResult, 0)
	c.logger.Debug().
		Str("query", query).
		Int("year", year).
		Int("results", len(results)).
		Msg("Movie search completed with no results")

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

	// Apply relevance scoring if not disabled
	if len(response.Results) > 0 {
		if !c.config.DisableSearchOrdering {
			currentYear := time.Now().Year()

			// Find max vote count for normalization
			maxVoteCount := 0
			for _, series := range response.Results {
				if series.VoteCount > maxVoteCount {
					maxVoteCount = series.VoteCount
				}
			}

			// Calculate scores for all series
			scoreableSeriesList := make([]scoreableSeries, len(response.Results))
			for i, series := range response.Results {
				scoreableSeriesList[i] = scoreableSeries{
					TVResult: series,
					Score:    c.calculateSeriesScore(series, maxVoteCount, currentYear),
				}
			}

			// Sort by score (highest first)
			sort.Slice(scoreableSeriesList, func(i, j int) bool {
				return scoreableSeriesList[i].Score > scoreableSeriesList[j].Score
			})

			// Convert to normalized results with scoring
			results := make([]NormalizedSeriesResult, len(scoreableSeriesList))
			for i, ss := range scoreableSeriesList {
				results[i] = c.toSeriesResult(ss.TVResult)
			}

			c.logger.Debug().
				Str("query", query).
				Int("results", len(results)).
				Msg("TV search completed with scoring")

			return results, nil
		} else {
			// Scoring disabled, return results in original order
			c.logger.Debug().
				Str("query", query).
				Bool("scoring_disabled", true).
				Msg("TV search completed without scoring")

			results := make([]NormalizedSeriesResult, len(response.Results))
			for i, series := range response.Results {
				results[i] = c.toSeriesResult(series)
			}
			return results, nil
		}
	}

	results := make([]NormalizedSeriesResult, 0)
	c.logger.Debug().
		Str("query", query).
		Int("results", len(results)).
		Msg("TV search completed with no results")

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

// GetMovieReleaseDates fetches release dates for a movie by TMDB ID.
// Returns digital (streaming/VOD) and physical (Bluray) release dates.
// US release dates are preferred, with fallback to other regions.
func (c *Client) GetMovieReleaseDates(ctx context.Context, id int) (digital, physical string, err error) {
	if !c.IsConfigured() {
		return "", "", ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/movie/%d/release_dates", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var response ReleaseDatesResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		return "", "", err
	}

	// Find US region first, fallback to first available
	var regionData *ReleaseDatesByRegion
	for i := range response.Results {
		if response.Results[i].ISO3166_1 == "US" {
			regionData = &response.Results[i]
			break
		}
	}
	if regionData == nil && len(response.Results) > 0 {
		regionData = &response.Results[0]
	}

	if regionData == nil {
		return "", "", nil
	}

	// Extract digital and physical release dates
	for _, rd := range regionData.ReleaseDates {
		dateStr := ""
		if len(rd.ReleaseDate) >= 10 {
			dateStr = rd.ReleaseDate[:10] // YYYY-MM-DD
		}
		if dateStr == "" {
			continue
		}

		switch rd.Type {
		case ReleaseDateTypeDigital:
			if digital == "" {
				digital = dateStr
			}
		case ReleaseDateTypePhysical:
			if physical == "" {
				physical = dateStr
			}
		}
	}

	c.logger.Debug().
		Int("id", id).
		Str("digital", digital).
		Str("physical", physical).
		Msg("Got movie release dates")

	return digital, physical, nil
}

// GetSeasonDetails gets detailed info for a specific season including all episodes.
func (c *Client) GetSeasonDetails(ctx context.Context, seriesID, seasonNumber int) (*NormalizedSeasonResult, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/tv/%d/season/%d", c.config.BaseURL, seriesID, seasonNumber)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var details SeasonDetails
	if err := c.doRequest(ctx, endpoint, params, &details); err != nil {
		return nil, err
	}

	result := c.seasonDetailsToResult(details)

	c.logger.Debug().
		Int("seriesID", seriesID).
		Int("seasonNumber", seasonNumber).
		Int("episodes", len(result.Episodes)).
		Msg("Got season details")

	return &result, nil
}

// GetAllSeasons gets all seasons with episodes for a series.
func (c *Client) GetAllSeasons(ctx context.Context, seriesID int) ([]NormalizedSeasonResult, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	// First get series details to know which seasons exist
	endpoint := fmt.Sprintf("%s/tv/%d", c.config.BaseURL, seriesID)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var details TVDetails
	if err := c.doRequest(ctx, endpoint, params, &details); err != nil {
		return nil, err
	}

	results := make([]NormalizedSeasonResult, 0, len(details.Seasons))

	// Fetch each season's details including episodes
	for _, season := range details.Seasons {
		seasonResult, err := c.GetSeasonDetails(ctx, seriesID, season.SeasonNumber)
		if err != nil {
			c.logger.Warn().
				Err(err).
				Int("seriesID", seriesID).
				Int("seasonNumber", season.SeasonNumber).
				Msg("Failed to get season details, skipping")
			continue
		}
		results = append(results, *seasonResult)
	}

	c.logger.Debug().
		Int("seriesID", seriesID).
		Int("seasons", len(results)).
		Msg("Got all seasons")

	return results, nil
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
		ID:          details.ID,
		Title:       details.Title,
		Year:        year,
		Overview:    details.Overview,
		Runtime:     details.Runtime,
		ImdbID:      details.ImdbID,
		Genres:      genres,
		ReleaseDate: details.ReleaseDate, // Basic release date from details
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
		TmdbID:   tv.ID, // Set TmdbID same as ID for TMDB search results
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

// seasonDetailsToResult converts TMDB season details to a NormalizedSeasonResult.
func (c *Client) seasonDetailsToResult(details SeasonDetails) NormalizedSeasonResult {
	episodes := make([]NormalizedEpisodeResult, len(details.Episodes))
	for i, ep := range details.Episodes {
		episodes[i] = NormalizedEpisodeResult{
			EpisodeNumber: ep.EpisodeNumber,
			SeasonNumber:  ep.SeasonNumber,
			Title:         ep.Name,
			Overview:      ep.Overview,
			AirDate:       ep.AirDate,
			Runtime:       ep.Runtime,
		}
	}

	result := NormalizedSeasonResult{
		SeasonNumber: details.SeasonNumber,
		Name:         details.Name,
		Overview:     details.Overview,
		AirDate:      details.AirDate,
		Episodes:     episodes,
	}

	if details.PosterPath != nil {
		result.PosterURL = c.GetImageURL(*details.PosterPath, "w500")
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

	// Get primary network (first in list)
	network := ""
	if len(details.Networks) > 0 {
		network = details.Networks[0].Name
	}

	result := NormalizedSeriesResult{
		ID:       details.ID,
		TmdbID:   details.ID,
		Title:    details.Name,
		Year:     year,
		Overview: details.Overview,
		Status:   status,
		Genres:   genres,
		Network:  network,
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

// scoreableMovie represents a movie with scoring data
type scoreableMovie struct {
	MovieResult MovieResult
	Score       float64
}

// scoreableSeries represents a series with scoring data
type scoreableSeries struct {
	TVResult TVResult
	Score    float64
}

// calculateMovieScore calculates the relevance score for a movie result
func (c *Client) calculateMovieScore(movie MovieResult, maxVoteCount int, currentYear int) float64 {
	// Normalize vote count to 0-1 scale
	voteCountNormalized := 0.0
	if maxVoteCount > 0 {
		voteCountNormalized = float64(movie.VoteCount) / float64(maxVoteCount)
	}

	// Calculate recency factor (0-1 scale, newer is better)
	recencyFactor := 0.0
	year := 0
	if len(movie.ReleaseDate) >= 4 {
		year, _ = strconv.Atoi(movie.ReleaseDate[:4])
	}
	if year > 0 {
		yearsDiff := currentYear - year
		// Max 10 years for recency scoring, exponential decay
		recencyFactor = math.Exp(-float64(yearsDiff) * 0.2)
	}

	// Calculate completeness factor
	completenessFactor := 0.0
	if movie.PosterPath != nil {
		completenessFactor += 0.05
	}
	if movie.Overview != "" {
		completenessFactor += 0.03
	}
	if len(movie.GenreIDs) > 0 {
		completenessFactor += 0.02
	}

	// Final score calculation
	score := (movie.Popularity * 0.3) +
		(movie.VoteAverage * voteCountNormalized * 0.4) +
		(recencyFactor * 0.2) +
		(completenessFactor * 0.1)

	return score
}

// calculateSeriesScore calculates the relevance score for a series result
func (c *Client) calculateSeriesScore(series TVResult, maxVoteCount int, currentYear int) float64 {
	// Normalize vote count to 0-1 scale
	voteCountNormalized := 0.0
	if maxVoteCount > 0 {
		voteCountNormalized = float64(series.VoteCount) / float64(maxVoteCount)
	}

	// Calculate recency factor (0-1 scale, newer is better)
	recencyFactor := 0.0
	year := 0
	if len(series.FirstAirDate) >= 4 {
		year, _ = strconv.Atoi(series.FirstAirDate[:4])
	}
	if year > 0 {
		yearsDiff := currentYear - year
		// Max 10 years for recency scoring, exponential decay
		recencyFactor = math.Exp(-float64(yearsDiff) * 0.2)
	}

	// Calculate completeness factor
	completenessFactor := 0.0
	if series.PosterPath != nil {
		completenessFactor += 0.05
	}
	if series.Overview != "" {
		completenessFactor += 0.03
	}
	if len(series.GenreIDs) > 0 {
		completenessFactor += 0.02
	}

	// Final score calculation
	score := (series.Popularity * 0.3) +
		(series.VoteAverage * voteCountNormalized * 0.4) +
		(recencyFactor * 0.2) +
		(completenessFactor * 0.1)

	return score
}

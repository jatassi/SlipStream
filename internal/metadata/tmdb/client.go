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
	logger     *zerolog.Logger
}

// NewClient creates a new TMDB client.
func NewClient(cfg config.TMDBConfig, logger *zerolog.Logger) *Client {
	subLogger := logger.With().Str("component", "tmdb").Logger()
	return &Client{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		config: cfg,
		logger: &subLogger,
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

	if len(response.Results) == 0 {
		c.logger.Debug().Str("query", query).Int("year", year).Msg("Movie search completed with no results")
		return []NormalizedMovieResult{}, nil
	}

	if c.config.DisableSearchOrdering {
		return c.convertMovieResultsUnscored(response.Results, query, year), nil
	}

	return c.convertMovieResultsScored(response.Results, query, year), nil
}

func (c *Client) convertMovieResultsUnscored(movies []MovieResult, query string, year int) []NormalizedMovieResult {
	results := make([]NormalizedMovieResult, len(movies))
	for i := range movies {
		results[i] = c.toMovieResult(&movies[i])
	}
	c.logger.Debug().Str("query", query).Int("year", year).Bool("scoring_disabled", true).Msg("Movie search completed without scoring")
	return results
}

func (c *Client) convertMovieResultsScored(movies []MovieResult, query string, year int) []NormalizedMovieResult {
	currentYear := time.Now().Year()
	maxVoteCount := c.findMaxVoteCount(movies)

	scoreableMovies := make([]scoreableMovie, len(movies))
	for i := range movies {
		scoreableMovies[i] = scoreableMovie{
			MovieResult: movies[i],
			Score:       c.calculateMovieScore(&movies[i], maxVoteCount, currentYear),
		}
	}

	sort.Slice(scoreableMovies, func(i, j int) bool {
		return scoreableMovies[i].Score > scoreableMovies[j].Score
	})

	results := make([]NormalizedMovieResult, len(scoreableMovies))
	for i := range scoreableMovies {
		results[i] = c.toMovieResult(&scoreableMovies[i].MovieResult)
	}

	c.logger.Debug().Str("query", query).Int("year", year).Int("results", len(results)).Msg("Movie search completed with scoring")
	return results
}

func (c *Client) findMaxVoteCount(movies []MovieResult) int {
	maxVoteCount := 0
	for i := range movies {
		if movies[i].VoteCount > maxVoteCount {
			maxVoteCount = movies[i].VoteCount
		}
	}
	return maxVoteCount
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

	result := c.movieDetailsToResult(&details)

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

	if len(response.Results) == 0 {
		c.logger.Debug().Str("query", query).Msg("TV search completed with no results")
		return []NormalizedSeriesResult{}, nil
	}

	if c.config.DisableSearchOrdering {
		return c.convertSeriesResultsUnscored(response.Results, query), nil
	}

	return c.convertSeriesResultsScored(response.Results, query), nil
}

func (c *Client) convertSeriesResultsUnscored(series []TVResult, query string) []NormalizedSeriesResult {
	results := make([]NormalizedSeriesResult, len(series))
	for i := range series {
		results[i] = c.toSeriesResult(&series[i])
	}
	c.logger.Debug().Str("query", query).Bool("scoring_disabled", true).Msg("TV search completed without scoring")
	return results
}

func (c *Client) convertSeriesResultsScored(series []TVResult, query string) []NormalizedSeriesResult {
	currentYear := time.Now().Year()
	maxVoteCount := c.findMaxSeriesVoteCount(series)

	scoreableSeriesList := make([]scoreableSeries, len(series))
	for i := range series {
		scoreableSeriesList[i] = scoreableSeries{
			TVResult: series[i],
			Score:    c.calculateSeriesScore(&series[i], maxVoteCount, currentYear),
		}
	}

	sort.Slice(scoreableSeriesList, func(i, j int) bool {
		return scoreableSeriesList[i].Score > scoreableSeriesList[j].Score
	})

	results := make([]NormalizedSeriesResult, len(scoreableSeriesList))
	for i := range scoreableSeriesList {
		results[i] = c.toSeriesResult(&scoreableSeriesList[i].TVResult)
	}

	c.logger.Debug().Str("query", query).Int("results", len(results)).Msg("TV search completed with scoring")
	return results
}

func (c *Client) findMaxSeriesVoteCount(series []TVResult) int {
	maxVoteCount := 0
	for i := range series {
		if series[i].VoteCount > maxVoteCount {
			maxVoteCount = series[i].VoteCount
		}
	}
	return maxVoteCount
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
		// Convert generic "movie not found" to "series not found" for TV endpoints
		if errors.Is(err, ErrMovieNotFound) {
			return nil, ErrSeriesNotFound
		}
		return nil, err
	}

	result := c.tvDetailsToResult(&details)

	c.logger.Debug().
		Int("id", id).
		Str("title", result.Title).
		Msg("Got TV series details")

	return &result, nil
}

// GetMovieReleaseDates fetches release dates for a movie by TMDB ID.
// Returns digital (streaming/VOD), physical (Bluray), and theatrical release dates.
// US release dates are preferred, with fallback to other regions.
func (c *Client) GetMovieReleaseDates(ctx context.Context, id int) (digital, physical, theatrical string, err error) {
	if !c.IsConfigured() {
		return "", "", "", ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/movie/%d/release_dates", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var response ReleaseDatesResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		return "", "", "", err
	}

	regionData := c.findPreferredRegion(response.Results)
	if regionData == nil {
		return "", "", "", nil
	}

	digital, physical, theatrical = c.extractReleaseDates(regionData.ReleaseDates)

	c.logger.Debug().
		Int("id", id).
		Str("digital", digital).
		Str("physical", physical).
		Str("theatrical", theatrical).
		Msg("Got movie release dates")

	return digital, physical, theatrical, nil
}

func (c *Client) findPreferredRegion(results []ReleaseDatesByRegion) *ReleaseDatesByRegion {
	for i := range results {
		if results[i].Iso31661 == "US" {
			return &results[i]
		}
	}
	if len(results) > 0 {
		return &results[0]
	}
	return nil
}

func (c *Client) extractReleaseDates(releaseDates []ReleaseDate) (digital, physical, theatrical string) {
	firsts := c.collectFirstReleaseDates(releaseDates)
	theatrical = c.selectTheatricalDate(
		firsts[ReleaseDateTypeTheatrical],
		firsts[ReleaseDateTypeTheatricalLimited],
		firsts[ReleaseDateTypePremiere],
	)
	return firsts[ReleaseDateTypeDigital], firsts[ReleaseDateTypePhysical], theatrical
}

func (c *Client) collectFirstReleaseDates(releaseDates []ReleaseDate) map[int]string {
	firsts := make(map[int]string)
	for _, rd := range releaseDates {
		dateStr := c.extractDateString(rd.ReleaseDate)
		if dateStr == "" {
			continue
		}
		if _, exists := firsts[rd.Type]; !exists {
			firsts[rd.Type] = dateStr
		}
	}
	return firsts
}

func (c *Client) extractDateString(releaseDate string) string {
	if len(releaseDate) >= 10 {
		return releaseDate[:10]
	}
	return ""
}

func (c *Client) selectTheatricalDate(theatricalDate, theatricalLimited, premiere string) string {
	if theatricalDate != "" {
		return theatricalDate
	}
	if theatricalLimited != "" {
		return theatricalLimited
	}
	return premiere
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
		if errors.Is(err, ErrMovieNotFound) {
			return nil, ErrSeriesNotFound
		}
		return nil, err
	}

	result := c.seasonDetailsToResult(&details)

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
		if errors.Is(err, ErrMovieNotFound) {
			return nil, ErrSeriesNotFound
		}
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
func (c *Client) GetImageURL(path, size string) string {
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
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
func (c *Client) toMovieResult(movie *MovieResult) NormalizedMovieResult {
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
func (c *Client) movieDetailsToResult(details *MovieDetails) NormalizedMovieResult {
	year := 0
	if len(details.ReleaseDate) >= 4 {
		year, _ = strconv.Atoi(details.ReleaseDate[:4])
	}

	genres := make([]string, len(details.Genres))
	for i, g := range details.Genres {
		genres[i] = g.Name
	}

	studio := ""
	studioLogoURL := ""
	if len(details.ProductionCompanies) > 0 {
		studio = details.ProductionCompanies[0].Name
		if details.ProductionCompanies[0].LogoPath != nil {
			studioLogoURL = c.GetImageURL(*details.ProductionCompanies[0].LogoPath, "w500")
		}
	}

	result := NormalizedMovieResult{
		ID:            details.ID,
		Title:         details.Title,
		Year:          year,
		Overview:      details.Overview,
		Runtime:       details.Runtime,
		ImdbID:        details.ImdbID,
		Genres:        genres,
		ReleaseDate:   details.ReleaseDate,
		Studio:        studio,
		StudioLogoURL: studioLogoURL,
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
func (c *Client) toSeriesResult(tv *TVResult) NormalizedSeriesResult {
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
func (c *Client) seasonDetailsToResult(details *SeasonDetails) NormalizedSeasonResult {
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
func (c *Client) tvDetailsToResult(details *TVDetails) NormalizedSeriesResult {
	year := 0
	if len(details.FirstAirDate) >= 4 {
		year, _ = strconv.Atoi(details.FirstAirDate[:4])
	}

	genres := make([]string, len(details.Genres))
	for i, g := range details.Genres {
		genres[i] = g.Name
	}

	network, networkLogoURL := c.extractNetworkInfo(details.Networks)

	result := NormalizedSeriesResult{
		ID:             details.ID,
		TmdbID:         details.ID,
		Title:          details.Name,
		Year:           year,
		Overview:       details.Overview,
		Status:         c.mapTMDBStatus(details.Status),
		Genres:         genres,
		Network:        network,
		NetworkLogoURL: networkLogoURL,
	}

	if details.PosterPath != nil {
		result.PosterURL = c.GetImageURL(*details.PosterPath, "w500")
	}
	if details.BackdropPath != nil {
		result.BackdropURL = c.GetImageURL(*details.BackdropPath, "w780")
	}

	if details.ExternalIDs != nil {
		result.ImdbID = details.ExternalIDs.ImdbID
		result.TvdbID = details.ExternalIDs.TvdbID
	}

	if len(details.EpisodeRunTime) > 0 {
		result.Runtime = details.EpisodeRunTime[0]
	}

	return result
}

func (c *Client) mapTMDBStatus(status string) string {
	switch status {
	case "Ended", "Canceled":
		return "ended"
	case "Returning Series", "In Production":
		return "continuing"
	case "Planned":
		return "upcoming"
	default:
		return "continuing"
	}
}

func (c *Client) extractNetworkInfo(networks []Network) (network, networkLogoURL string) {
	if len(networks) == 0 {
		return "", ""
	}
	network = networks[0].Name
	if networks[0].LogoPath != "" {
		networkLogoURL = c.GetImageURL(networks[0].LogoPath, "w500")
	}
	return network, networkLogoURL
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
func (c *Client) calculateMovieScore(movie *MovieResult, maxVoteCount, currentYear int) float64 {
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
func (c *Client) calculateSeriesScore(series *TVResult, maxVoteCount, currentYear int) float64 {
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

// GetMovieCredits fetches credits for a movie by TMDB ID.
func (c *Client) GetMovieCredits(ctx context.Context, id int) (*NormalizedCredits, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/movie/%d/credits", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var response CreditsResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		return nil, err
	}

	return c.normalizeCredits(response, nil), nil
}

// GetSeriesCredits fetches credits for a TV series by TMDB ID.
func (c *Client) GetSeriesCredits(ctx context.Context, id int) (*NormalizedCredits, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	// First get series details to get creators
	endpoint := fmt.Sprintf("%s/tv/%d", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var details TVDetails
	if err := c.doRequest(ctx, endpoint, params, &details); err != nil {
		if errors.Is(err, ErrMovieNotFound) {
			return nil, ErrSeriesNotFound
		}
		return nil, err
	}

	// Then get credits
	creditsEndpoint := fmt.Sprintf("%s/tv/%d/credits", c.config.BaseURL, id)
	creditsParams := url.Values{}
	creditsParams.Set("api_key", c.config.APIKey)

	var creditsResponse CreditsResponse
	if err := c.doRequest(ctx, creditsEndpoint, creditsParams, &creditsResponse); err != nil {
		if errors.Is(err, ErrMovieNotFound) {
			return nil, ErrSeriesNotFound
		}
		return nil, err
	}

	return c.normalizeCredits(creditsResponse, details.CreatedBy), nil
}

// GetMovieContentRating fetches the US content rating for a movie.
func (c *Client) GetMovieContentRating(ctx context.Context, id int) (string, error) {
	if !c.IsConfigured() {
		return "", ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/movie/%d/release_dates", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var response ReleaseDatesResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		return "", err
	}

	if rating := c.findUSContentRating(response.Results); rating != "" {
		return rating, nil
	}

	return c.findAnyContentRating(response.Results), nil
}

func (c *Client) findUSContentRating(results []ReleaseDatesByRegion) string {
	for _, region := range results {
		if region.Iso31661 != "US" {
			continue
		}
		for _, rd := range region.ReleaseDates {
			if rd.Certification != "" {
				return rd.Certification
			}
		}
	}
	return ""
}

func (c *Client) findAnyContentRating(results []ReleaseDatesByRegion) string {
	for _, region := range results {
		for _, rd := range region.ReleaseDates {
			if rd.Certification != "" {
				return rd.Certification
			}
		}
	}
	return ""
}

// GetSeriesContentRating fetches the US content rating for a TV series.
func (c *Client) GetSeriesContentRating(ctx context.Context, id int) (string, error) {
	if !c.IsConfigured() {
		return "", ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/tv/%d/content_ratings", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var response ContentRatingsResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		if errors.Is(err, ErrMovieNotFound) {
			return "", ErrSeriesNotFound
		}
		return "", err
	}

	// Look for US rating first
	for _, rating := range response.Results {
		if rating.Iso31661 == "US" && rating.Rating != "" {
			return rating.Rating, nil
		}
	}

	// Fallback to first available rating
	for _, rating := range response.Results {
		if rating.Rating != "" {
			return rating.Rating, nil
		}
	}

	return "", nil
}

// normalizeCredits converts TMDB credits response to normalized format.
func (c *Client) normalizeCredits(credits CreditsResponse, creators []TVCreator) *NormalizedCredits {
	result := &NormalizedCredits{
		Cast: make([]NormalizedPerson, 0),
	}

	// Process directors and writers from crew
	for _, crew := range credits.Crew {
		person := NormalizedPerson{
			ID:   crew.ID,
			Name: crew.Name,
		}
		if crew.ProfilePath != nil {
			person.PhotoURL = c.GetImageURL(*crew.ProfilePath, "w185")
		}

		switch crew.Job {
		case "Director":
			result.Directors = append(result.Directors, person)
		case "Writer", "Screenplay", "Story":
			person.Role = crew.Job
			result.Writers = append(result.Writers, person)
		}
	}

	// Process creators for TV series
	for _, creator := range creators {
		person := NormalizedPerson{
			ID:   creator.ID,
			Name: creator.Name,
		}
		if creator.ProfilePath != nil {
			person.PhotoURL = c.GetImageURL(*creator.ProfilePath, "w185")
		}
		result.Creators = append(result.Creators, person)
	}

	// Process cast (limit to top 20)
	limit := 20
	if len(credits.Cast) < limit {
		limit = len(credits.Cast)
	}
	for i := 0; i < limit; i++ {
		cast := credits.Cast[i]
		person := NormalizedPerson{
			ID:   cast.ID,
			Name: cast.Name,
			Role: cast.Character,
		}
		if cast.ProfilePath != nil {
			person.PhotoURL = c.GetImageURL(*cast.ProfilePath, "w185")
		}
		result.Cast = append(result.Cast, person)
	}

	return result
}

// GetMovieLogoURL fetches the best English title treatment logo for a movie.
func (c *Client) GetMovieLogoURL(ctx context.Context, id int) (string, error) {
	if !c.IsConfigured() {
		return "", ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/movie/%d/images", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var response ImagesResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		return "", err
	}

	return c.pickBestLogo(response.Logos), nil
}

// GetSeriesLogoURL fetches the best English title treatment logo for a TV series.
func (c *Client) GetSeriesLogoURL(ctx context.Context, id int) (string, error) {
	if !c.IsConfigured() {
		return "", ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/tv/%d/images", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var response ImagesResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		if errors.Is(err, ErrMovieNotFound) {
			return "", ErrSeriesNotFound
		}
		return "", err
	}

	return c.pickBestLogo(response.Logos), nil
}

// pickBestLogo selects the best logo from a list, preferring English, then highest voted.
func (c *Client) pickBestLogo(logos []ImageResult) string {
	if len(logos) == 0 {
		return ""
	}

	// Filter for English logos first
	var english []ImageResult
	for _, logo := range logos {
		if logo.Iso6391 == "en" {
			english = append(english, logo)
		}
	}

	// Pick from English if available, otherwise all logos
	candidates := english
	if len(candidates) == 0 {
		candidates = logos
	}

	// Pick the highest voted
	best := candidates[0]
	for _, logo := range candidates[1:] {
		if logo.VoteAverage > best.VoteAverage {
			best = logo
		}
	}

	return c.GetImageURL(best.FilePath, "w500")
}

// GetMovieTrailerURL fetches the YouTube trailer URL for a movie.
func (c *Client) GetMovieTrailerURL(ctx context.Context, id int) (string, error) {
	if !c.IsConfigured() {
		return "", ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/movie/%d/videos", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var response VideosResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		return "", err
	}

	return c.pickTrailerURL(response.Results), nil
}

// GetSeriesTrailerURL fetches the YouTube trailer URL for a TV series.
func (c *Client) GetSeriesTrailerURL(ctx context.Context, id int) (string, error) {
	if !c.IsConfigured() {
		return "", ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/tv/%d/videos", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var response VideosResponse
	if err := c.doRequest(ctx, endpoint, params, &response); err != nil {
		if errors.Is(err, ErrMovieNotFound) {
			return "", ErrSeriesNotFound
		}
		return "", err
	}

	return c.pickTrailerURL(response.Results), nil
}

// pickTrailerURL selects the best YouTube trailer URL from a list of videos.
// Prefers official trailers, falls back to any trailer.
func (c *Client) pickTrailerURL(videos []Video) string {
	var fallback string
	for _, v := range videos {
		if v.Site != "YouTube" || v.Type != "Trailer" {
			continue
		}
		if v.Official {
			return fmt.Sprintf("https://www.youtube.com/watch?v=%s", v.Key)
		}
		if fallback == "" {
			fallback = fmt.Sprintf("https://www.youtube.com/watch?v=%s", v.Key)
		}
	}
	return fallback
}

// GetMovieStudio returns the primary production company for a movie.
func (c *Client) GetMovieStudio(ctx context.Context, id int) (string, error) {
	if !c.IsConfigured() {
		return "", ErrAPIKeyMissing
	}

	endpoint := fmt.Sprintf("%s/movie/%d", c.config.BaseURL, id)
	params := url.Values{}
	params.Set("api_key", c.config.APIKey)

	var details MovieDetails
	if err := c.doRequest(ctx, endpoint, params, &details); err != nil {
		return "", err
	}

	if len(details.ProductionCompanies) > 0 {
		return details.ProductionCompanies[0].Name, nil
	}

	return "", nil
}

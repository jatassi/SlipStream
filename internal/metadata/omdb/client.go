package omdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
)

var (
	ErrAPIKeyMissing = errors.New("OMDb API key is not configured")
	ErrNotFound      = errors.New("not found on OMDb")
	ErrAPIError      = errors.New("OMDb API error")
)

const omdbNotAvailable = "N/A"

// Client is an OMDb API client.
type Client struct {
	httpClient *http.Client
	config     config.OMDBConfig
	logger     zerolog.Logger
}

// NewClient creates a new OMDb client.
func NewClient(cfg config.OMDBConfig, logger *zerolog.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		config: cfg,
		logger: logger.With().Str("component", "omdb").Logger(),
	}
}

// Name returns the provider name.
func (c *Client) Name() string {
	return "omdb"
}

// IsConfigured returns true if the API key is set.
func (c *Client) IsConfigured() bool {
	return c.config.APIKey != ""
}

// Test verifies connectivity to the OMDb API.
func (c *Client) Test(ctx context.Context) error {
	if !c.IsConfigured() {
		return ErrAPIKeyMissing
	}

	// Try to fetch data for a known movie
	_, err := c.GetByIMDbID(ctx, "tt0133093") // The Matrix
	return err
}

// GetByIMDbID fetches ratings and awards for a title by IMDb ID.
func (c *Client) GetByIMDbID(ctx context.Context, imdbID string) (*NormalizedRatings, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}

	if imdbID == "" {
		return nil, ErrNotFound
	}

	params := url.Values{}
	params.Set("apikey", c.config.APIKey)
	params.Set("i", imdbID)

	reqURL := fmt.Sprintf("%s?%s", c.config.BaseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error().Err(err).Str("imdbId", imdbID).Msg("HTTP request failed")
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrAPIError, resp.StatusCode)
	}

	var omdbResp Response
	if err := json.NewDecoder(resp.Body).Decode(&omdbResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if omdbResp.Response == "False" {
		if omdbResp.Error == "Movie not found!" || omdbResp.Error == "Incorrect IMDb ID." {
			return nil, ErrNotFound
		}
		c.logger.Warn().Str("error", omdbResp.Error).Str("imdbId", imdbID).Msg("OMDb API returned error")
		return nil, fmt.Errorf("%w: %s", ErrAPIError, omdbResp.Error)
	}

	return c.normalizeRatings(&omdbResp), nil
}

// GetSeasonEpisodes fetches episode ratings for a season by IMDb series ID.
// Returns a map of episode number to IMDB rating.
func (c *Client) GetSeasonEpisodes(ctx context.Context, imdbID string, season int) (map[int]float64, error) {
	if !c.IsConfigured() {
		return nil, ErrAPIKeyMissing
	}
	if imdbID == "" {
		return nil, ErrNotFound
	}

	seasonResp, err := c.fetchSeasonData(ctx, imdbID, season)
	if err != nil {
		return nil, err
	}

	ratings := c.parseEpisodeRatings(seasonResp.Episodes)

	c.logger.Debug().
		Str("imdbId", imdbID).
		Int("season", season).
		Int("episodes", len(ratings)).
		Msg("Fetched season episode ratings from OMDb")

	return ratings, nil
}

func (c *Client) fetchSeasonData(ctx context.Context, imdbID string, season int) (*SeasonEpisodesResponse, error) {
	params := url.Values{}
	params.Set("apikey", c.config.APIKey)
	params.Set("i", imdbID)
	params.Set("Season", strconv.Itoa(season))
	params.Set("type", "episode")

	reqURL := fmt.Sprintf("%s?%s", c.config.BaseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error().Err(err).Str("imdbId", imdbID).Int("season", season).Msg("HTTP request failed")
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrAPIError, resp.StatusCode)
	}

	var seasonResp SeasonEpisodesResponse
	if err := json.NewDecoder(resp.Body).Decode(&seasonResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if seasonResp.Response == "False" {
		return nil, fmt.Errorf("%w: %s", ErrAPIError, seasonResp.Error)
	}

	return &seasonResp, nil
}

func (c *Client) parseEpisodeRatings(episodes []EpisodeRating) map[int]float64 {
	ratings := make(map[int]float64, len(episodes))
	for _, ep := range episodes {
		epNum, err := strconv.Atoi(ep.Episode)
		if err != nil {
			continue
		}
		if ep.ImdbRating == "" || ep.ImdbRating == omdbNotAvailable {
			continue
		}
		if rating, err := strconv.ParseFloat(ep.ImdbRating, 64); err == nil {
			ratings[epNum] = rating
		}
	}
	return ratings
}

// normalizeRatings converts OMDb response to normalized format.
func (c *Client) normalizeRatings(resp *Response) *NormalizedRatings {
	result := &NormalizedRatings{
		Awards: resp.Awards,
	}

	c.parseImdbRating(resp.ImdbRating, result)
	c.parseImdbVotes(resp.ImdbVotes, result)
	c.parseMetacritic(resp.Metascore, result)
	c.parseRottenTomatoes(resp.Ratings, result)

	c.logger.Debug().
		Str("imdbId", resp.ImdbID).
		Float64("imdbRating", result.ImdbRating).
		Int("rottenTomatoes", result.RottenTomatoes).
		Msg("Normalized OMDb ratings")

	return result
}

func (c *Client) parseImdbRating(imdbRating string, result *NormalizedRatings) {
	if imdbRating == "" || imdbRating == omdbNotAvailable {
		return
	}
	if rating, err := strconv.ParseFloat(imdbRating, 64); err == nil {
		result.ImdbRating = rating
	}
}

func (c *Client) parseImdbVotes(imdbVotes string, result *NormalizedRatings) {
	if imdbVotes == "" || imdbVotes == omdbNotAvailable {
		return
	}
	votesStr := strings.ReplaceAll(imdbVotes, ",", "")
	if votes, err := strconv.Atoi(votesStr); err == nil {
		result.ImdbVotes = votes
	}
}

func (c *Client) parseMetacritic(metascore string, result *NormalizedRatings) {
	if metascore == "" || metascore == omdbNotAvailable {
		return
	}
	if score, err := strconv.Atoi(metascore); err == nil {
		result.Metacritic = score
	}
}

func (c *Client) parseRottenTomatoes(ratings []Rating, result *NormalizedRatings) {
	for _, rating := range ratings {
		if rating.Source != "Rotten Tomatoes" {
			continue
		}
		scoreStr := strings.TrimSuffix(rating.Value, "%")
		if score, err := strconv.Atoi(scoreStr); err == nil {
			result.RottenTomatoes = score
		}
		break
	}
}

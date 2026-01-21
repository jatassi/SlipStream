package prowlarr

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	defaultTimeout = 90 * time.Second
	apiKeyHeader   = "X-Api-Key"
)

// Client provides HTTP communication with a Prowlarr server.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     zerolog.Logger
}

// ClientConfig contains configuration for creating a new Prowlarr client.
type ClientConfig struct {
	URL           string
	APIKey        string
	Timeout       int
	SkipSSLVerify bool
	Logger        zerolog.Logger
}

// NewClient creates a new Prowlarr HTTP client.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("prowlarr URL is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("prowlarr API key is required")
	}

	baseURL := strings.TrimSuffix(cfg.URL, "/")

	timeout := defaultTimeout
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	transport := &http.Transport{}
	if cfg.SkipSSLVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	logger := cfg.Logger.With().
		Str("component", "prowlarr-client").
		Str("url", baseURL).
		Logger()

	return &Client{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		logger: logger,
	}, nil
}

// do executes an HTTP request with the API key header.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(apiKeyHeader, c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.logger.Debug().
		Str("method", method).
		Str("path", path).
		Msg("executing request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error().Err(err).
			Str("method", method).
			Str("path", path).
			Msg("request failed")
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// doJSON executes an HTTP request and decodes the JSON response.
func (c *Client) doJSON(ctx context.Context, method, path string, body io.Reader, result interface{}) error {
	resp, err := c.do(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(bodyBytes)).
			Msg("request returned error status")
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// doXML executes an HTTP request and decodes the XML response.
func (c *Client) doXML(ctx context.Context, method, path string, body io.Reader, result interface{}) error {
	resp, err := c.do(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.logger.Error().
			Int("status", resp.StatusCode).
			Str("body", string(bodyBytes)).
			Msg("request returned error status")
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := xml.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode XML response: %w", err)
		}
	}

	return nil
}

// doRaw executes an HTTP request and returns the raw response body.
func (c *Client) doRaw(ctx context.Context, method, path string, body io.Reader) ([]byte, error) {
	resp, err := c.do(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}

// TestConnection verifies connectivity to Prowlarr by fetching system status.
func (c *Client) TestConnection(ctx context.Context) error {
	var status struct {
		Version string `json:"version"`
	}

	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/system/status", nil, &status); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	c.logger.Info().
		Str("version", status.Version).
		Msg("connection test successful")

	return nil
}

// GetCapabilities fetches the Torznab capabilities from Prowlarr's aggregated endpoint.
func (c *Client) GetCapabilities(ctx context.Context) (*Capabilities, error) {
	var caps TorznabCaps

	if err := c.doXML(ctx, http.MethodGet, "/api?t=caps", nil, &caps); err != nil {
		return nil, fmt.Errorf("failed to fetch capabilities: %w", err)
	}

	return c.convertCapabilities(&caps), nil
}

// convertCapabilities transforms Torznab XML capabilities to our internal format.
func (c *Client) convertCapabilities(caps *TorznabCaps) *Capabilities {
	result := &Capabilities{
		Server: ServerInfo{
			Title:   caps.Server.Title,
			Version: caps.Server.Version,
		},
		Limits: LimitsInfo{
			Max:     caps.Limits.Max,
			Default: caps.Limits.Default,
		},
		Searching: SearchingInfo{
			Search:      c.convertSearchType(caps.Searching.Search),
			TVSearch:    c.convertSearchType(caps.Searching.TVSearch),
			MovieSearch: c.convertSearchType(caps.Searching.MovieSearch),
		},
		Categories: c.convertCategories(caps.Categories.Categories),
	}

	return result
}

func (c *Client) convertSearchType(st TorznabSearchType) SearchTypeInfo {
	return SearchTypeInfo{
		Available:       st.Available == "yes",
		SupportedParams: strings.Split(st.SupportedParams, ","),
	}
}

func (c *Client) convertCategories(cats []TorznabCategory) []Category {
	result := make([]Category, 0, len(cats))
	for _, cat := range cats {
		result = append(result, Category{
			ID:            cat.ID,
			Name:          cat.Name,
			Subcategories: c.convertCategories(cat.Subcategories),
		})
	}
	return result
}

// GetIndexers fetches the list of indexers configured in Prowlarr.
func (c *Client) GetIndexers(ctx context.Context) ([]Indexer, error) {
	var indexers []prowlarrIndexerResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/indexer", nil, &indexers); err != nil {
		return nil, fmt.Errorf("failed to fetch indexers: %w", err)
	}

	result := make([]Indexer, 0, len(indexers))
	for _, idx := range indexers {
		result = append(result, c.convertIndexer(idx))
	}

	c.logger.Debug().
		Int("count", len(result)).
		Msg("fetched indexers")

	return result, nil
}

// prowlarrIndexerResponse represents the Prowlarr API response for indexers.
type prowlarrIndexerResponse struct {
	ID           int                       `json:"id"`
	Name         string                    `json:"name"`
	Protocol     string                    `json:"protocol"`
	Privacy      string                    `json:"privacy,omitempty"`
	Priority     int                       `json:"priority"`
	Enable       bool                      `json:"enable"`
	Status       *prowlarrIndexerStatus    `json:"status,omitempty"`
	Capabilities *prowlarrIndexerCaps      `json:"capabilities,omitempty"`
	Fields       []prowlarrIndexerField    `json:"fields,omitempty"`
}

type prowlarrIndexerStatus struct {
	IsDisabled          bool   `json:"isDisabled"`
	MostRecentFailure   string `json:"mostRecentFailure,omitempty"`
	DisabledTill        string `json:"disabledTill,omitempty"`
}

type prowlarrIndexerCaps struct {
	SupportsRawSearch   bool              `json:"supportsRawSearch"`
	SearchParams        []string          `json:"searchParams,omitempty"`
	TvSearchParams      []string          `json:"tvSearchParams,omitempty"`
	MovieSearchParams   []string          `json:"movieSearchParams,omitempty"`
	Categories          []prowlarrCategory `json:"categories,omitempty"`
}

type prowlarrCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type prowlarrIndexerField struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

func (c *Client) convertIndexer(idx prowlarrIndexerResponse) Indexer {
	status := IndexerStatusHealthy
	if idx.Status != nil && idx.Status.IsDisabled {
		status = IndexerStatusDisabled
	}

	var caps IndexerCaps
	if idx.Capabilities != nil {
		caps.SupportsSearch = len(idx.Capabilities.SearchParams) > 0
		caps.SupportsTV = len(idx.Capabilities.TvSearchParams) > 0
		caps.SupportsMovies = len(idx.Capabilities.MovieSearchParams) > 0

		for _, cat := range idx.Capabilities.Categories {
			caps.Categories = append(caps.Categories, cat.ID)
		}
	}

	fields := make([]IndexerField, 0, len(idx.Fields))
	for _, f := range idx.Fields {
		fields = append(fields, IndexerField{
			Name:  f.Name,
			Value: f.Value,
		})
	}

	return Indexer{
		ID:           idx.ID,
		Name:         idx.Name,
		Protocol:     protocolFromString(idx.Protocol),
		Priority:     idx.Priority,
		Enable:       idx.Enable,
		Status:       status,
		Capabilities: caps,
		Fields:       fields,
	}
}

// Search executes a search query through Prowlarr's aggregated endpoint.
func (c *Client) Search(ctx context.Context, req SearchRequest) (*TorznabFeed, error) {
	params := url.Values{}
	params.Set("extended", "1") // Always request extended attributes

	switch req.Type {
	case "movie":
		params.Set("t", "movie")
		if req.ImdbID != "" {
			params.Set("imdbid", req.ImdbID)
		}
		if req.TmdbID > 0 {
			params.Set("tmdbid", strconv.Itoa(req.TmdbID))
		}
	case "tvsearch":
		params.Set("t", "tvsearch")
		if req.TvdbID > 0 {
			params.Set("tvdbid", strconv.Itoa(req.TvdbID))
		}
		if req.TmdbID > 0 {
			params.Set("tmdbid", strconv.Itoa(req.TmdbID))
		}
		if req.Season > 0 {
			params.Set("season", strconv.Itoa(req.Season))
		}
		if req.Episode > 0 {
			params.Set("ep", strconv.Itoa(req.Episode))
		}
	default:
		params.Set("t", "search")
	}

	if req.Query != "" {
		params.Set("q", req.Query)
	}

	if len(req.Categories) > 0 {
		catStrs := make([]string, len(req.Categories))
		for i, cat := range req.Categories {
			catStrs[i] = strconv.Itoa(cat)
		}
		params.Set("cat", strings.Join(catStrs, ","))
	}

	if req.Limit > 0 {
		params.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Offset > 0 {
		params.Set("offset", strconv.Itoa(req.Offset))
	}

	path := "/api?" + params.Encode()

	c.logger.Debug().
		Str("type", req.Type).
		Str("query", req.Query).
		Ints("categories", req.Categories).
		Msg("executing search")

	var feed TorznabFeed
	if err := c.doXML(ctx, http.MethodGet, path, nil, &feed); err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	c.logger.Debug().
		Int("results", len(feed.Channel.Items)).
		Msg("search completed")

	return &feed, nil
}

// Download retrieves the torrent/NZB file from Prowlarr.
func (c *Client) Download(ctx context.Context, downloadURL string) ([]byte, error) {
	// The download URL from Prowlarr results already includes the full path
	// but we need to ensure it goes through our client for auth

	parsedURL, err := url.Parse(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("invalid download URL: %w", err)
	}

	// Extract the path and query
	path := parsedURL.Path
	if parsedURL.RawQuery != "" {
		path += "?" + parsedURL.RawQuery
	}

	c.logger.Debug().
		Str("url", downloadURL).
		Msg("downloading release")

	data, err := c.doRaw(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	c.logger.Debug().
		Int("size", len(data)).
		Msg("download completed")

	return data, nil
}

// GetSystemStatus returns Prowlarr system information.
func (c *Client) GetSystemStatus(ctx context.Context) (*ConnectionStatus, error) {
	var status struct {
		Version string `json:"version"`
	}

	err := c.doJSON(ctx, http.MethodGet, "/api/v1/system/status", nil, &status)
	now := time.Now()

	result := &ConnectionStatus{
		Connected:   err == nil,
		LastChecked: &now,
		Version:     status.Version,
	}

	if err != nil {
		result.Error = err.Error()
	}

	return result, nil
}

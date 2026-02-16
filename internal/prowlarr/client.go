package prowlarr

import (
	"context"
	"crypto/tls"
	"encoding/json"
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
	//nolint:gosec // header name constant, not a credential
	apiKeyHeader = "X-Api-Key"
)

// Client provides HTTP communication with a Prowlarr server.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *zerolog.Logger
}

// ClientConfig contains configuration for creating a new Prowlarr client.
type ClientConfig struct {
	URL           string
	APIKey        string
	Timeout       int
	SkipSSLVerify bool
	Logger        *zerolog.Logger
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
		//nolint:gosec // admin-configured endpoint, TLS verification optional
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
		logger: &logger,
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
func (c *Client) doJSON(ctx context.Context, path string, result interface{}) error {
	resp, err := c.do(ctx, http.MethodGet, path, nil)
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

	if err := c.doJSON(ctx, "/api/v1/system/status", &status); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	c.logger.Info().
		Str("version", status.Version).
		Msg("connection test successful")

	return nil
}

// GetCapabilities returns Prowlarr capabilities based on available indexers.
// Since Prowlarr's REST API doesn't have an aggregated caps endpoint,
// we construct capabilities from the system status and indexer list.
func (c *Client) GetCapabilities(ctx context.Context) (*Capabilities, error) {
	// Get system status for version info
	var status struct {
		Version string `json:"version"`
	}
	if err := c.doJSON(ctx, "/api/v1/system/status", &status); err != nil {
		return nil, fmt.Errorf("failed to fetch capabilities: %w", err)
	}

	// Build capabilities from known Prowlarr features
	caps := &Capabilities{
		Server: ServerInfo{
			Title:   "Prowlarr",
			Version: status.Version,
		},
		Limits: LimitsInfo{
			Max:     100,
			Default: 100,
		},
		Searching: SearchingInfo{
			Search: SearchTypeInfo{
				Available:       true,
				SupportedParams: []string{"q"},
			},
			TVSearch: SearchTypeInfo{
				Available:       true,
				SupportedParams: []string{"q", "season", "ep", "tvdbid"},
			},
			MovieSearch: SearchTypeInfo{
				Available:       true,
				SupportedParams: []string{"q", "imdbid", "tmdbid"},
			},
		},
		Categories: []Category{
			{ID: 2000, Name: "Movies"},
			{ID: 5000, Name: "TV"},
		},
	}

	return caps, nil
}

// GetIndexers fetches the list of indexers configured in Prowlarr.
func (c *Client) GetIndexers(ctx context.Context) ([]Indexer, error) {
	var indexers []prowlarrIndexerResponse
	if err := c.doJSON(ctx, "/api/v1/indexer", &indexers); err != nil {
		return nil, fmt.Errorf("failed to fetch indexers: %w", err)
	}

	result := make([]Indexer, 0, len(indexers))
	for i := range indexers {
		result = append(result, c.convertIndexer(&indexers[i]))
	}

	c.logger.Debug().
		Int("count", len(result)).
		Msg("fetched indexers")

	return result, nil
}

// prowlarrIndexerResponse represents the Prowlarr API response for indexers.
type prowlarrIndexerResponse struct {
	ID           int                    `json:"id"`
	Name         string                 `json:"name"`
	Protocol     string                 `json:"protocol"`
	Privacy      string                 `json:"privacy,omitempty"`
	Priority     int                    `json:"priority"`
	Enable       bool                   `json:"enable"`
	Status       *prowlarrIndexerStatus `json:"status,omitempty"`
	Capabilities *prowlarrIndexerCaps   `json:"capabilities,omitempty"`
	Fields       []prowlarrIndexerField `json:"fields,omitempty"`
}

type prowlarrIndexerStatus struct {
	IsDisabled        bool   `json:"isDisabled"`
	MostRecentFailure string `json:"mostRecentFailure,omitempty"`
	DisabledTill      string `json:"disabledTill,omitempty"`
}

type prowlarrIndexerCaps struct {
	SupportsRawSearch bool               `json:"supportsRawSearch"`
	SearchParams      []string           `json:"searchParams,omitempty"`
	TvSearchParams    []string           `json:"tvSearchParams,omitempty"`
	MovieSearchParams []string           `json:"movieSearchParams,omitempty"`
	Categories        []prowlarrCategory `json:"categories,omitempty"`
}

type prowlarrCategory struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type prowlarrIndexerField struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

func (c *Client) convertIndexer(idx *prowlarrIndexerResponse) Indexer {
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
		fields = append(fields, IndexerField(f))
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

// ProwlarrSearchResult represents a single result from Prowlarr's REST API search.
type ProwlarrSearchResult struct {
	GUID        string  `json:"guid"`
	Age         int     `json:"age"`
	AgeHours    float64 `json:"ageHours"`
	AgeMinutes  float64 `json:"ageMinutes"`
	Size        int64   `json:"size"`
	Grabs       int     `json:"grabs"`
	IndexerID   int     `json:"indexerId"`
	Indexer     string  `json:"indexer"`
	Title       string  `json:"title"`
	SortTitle   string  `json:"sortTitle"`
	ImdbID      int     `json:"imdbId"`
	TmdbID      int     `json:"tmdbId"`
	TvdbID      int     `json:"tvdbId"`
	PublishDate string  `json:"publishDate"`
	DownloadURL string  `json:"downloadUrl"`
	InfoURL     string  `json:"infoUrl"`
	Seeders     int     `json:"seeders"`
	Leechers    int     `json:"leechers"`
	Protocol    string  `json:"protocol"`
	Categories  []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"categories"`
	DownloadVolumeFactor float64 `json:"downloadVolumeFactor"`
	UploadVolumeFactor   float64 `json:"uploadVolumeFactor"`
	InfoHash             string  `json:"infoHash"`
}

// Search executes a search query through Prowlarr's REST API.
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*TorznabFeed, error) {
	path := "/api/v1/search?" + buildSearchParams(req).Encode()

	c.logger.Info().
		Str("type", req.Type).
		Str("query", req.Query).
		Ints("categories", req.Categories).
		Str("path", path).
		Msg("executing Prowlarr search request")

	var results []ProwlarrSearchResult
	if err := c.doJSON(ctx, path, &results); err != nil {
		c.logger.Error().Err(err).Str("path", path).Msg("Prowlarr search request failed")
		return nil, fmt.Errorf("search failed: %w", err)
	}

	c.logger.Info().Int("results", len(results)).Msg("Prowlarr search completed")

	return resultsToTorznabFeed(results), nil
}

func buildSearchParams(req *SearchRequest) url.Values {
	params := url.Values{}
	if req.Query != "" {
		params.Set("query", req.Query)
	}
	switch req.Type {
	case "movie":
		params.Set("type", "movie")
	case "tvsearch":
		params.Set("type", "tv")
	default:
		params.Set("type", "search")
	}
	for _, cat := range req.Categories {
		params.Add("categories", strconv.Itoa(cat))
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Offset > 0 {
		params.Set("offset", strconv.Itoa(req.Offset))
	}
	return params
}

func resultsToTorznabFeed(results []ProwlarrSearchResult) *TorznabFeed {
	feed := &TorznabFeed{}
	feed.Channel.Items = make([]TorznabItem, 0, len(results))

	for i := range results {
		feed.Channel.Items = append(feed.Channel.Items, resultToTorznabItem(&results[i]))
	}

	return feed
}

func resultToTorznabItem(r *ProwlarrSearchResult) TorznabItem {
	item := TorznabItem{
		Title:       r.Title,
		GUID:        r.GUID,
		Link:        r.DownloadURL,
		Size:        r.Size,
		PubDate:     r.PublishDate,
		Description: r.Title,
	}

	item.Attributes = append(item.Attributes,
		TorznabAttribute{Name: "indexer", Value: r.Indexer},
		TorznabAttribute{Name: "seeders", Value: strconv.Itoa(r.Seeders)},
		TorznabAttribute{Name: "peers", Value: strconv.Itoa(r.Seeders + r.Leechers)},
		TorznabAttribute{Name: "grabs", Value: strconv.Itoa(r.Grabs)},
		TorznabAttribute{Name: "downloadvolumefactor", Value: strconv.FormatFloat(r.DownloadVolumeFactor, 'f', -1, 64)},
		TorznabAttribute{Name: "uploadvolumefactor", Value: strconv.FormatFloat(r.UploadVolumeFactor, 'f', -1, 64)},
	)

	if r.InfoURL != "" {
		item.Attributes = append(item.Attributes, TorznabAttribute{Name: "comments", Value: r.InfoURL})
	}
	if r.InfoHash != "" {
		item.Attributes = append(item.Attributes, TorznabAttribute{Name: "infohash", Value: r.InfoHash})
	}
	if r.ImdbID > 0 {
		item.Attributes = append(item.Attributes, TorznabAttribute{Name: "imdb", Value: fmt.Sprintf("tt%07d", r.ImdbID)})
	}

	return item
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

	err := c.doJSON(ctx, "/api/v1/system/status", &status)
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

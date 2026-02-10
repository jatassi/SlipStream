package genericrss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/slipstream/slipstream/internal/indexer/cardigann"
	"github.com/slipstream/slipstream/internal/indexer/types"
)

const DefinitionID = "generic-rss"

// Settings holds the per-indexer configuration for a generic RSS feed.
type Settings struct {
	URL         string `json:"url"`
	Cookie      string `json:"cookie,omitempty"`
	ContentType string `json:"contentType"` // "movies", "tv", "both"
}

// Client implements the Indexer and TorrentIndexer interfaces for a generic RSS feed.
type Client struct {
	def      *types.IndexerDefinition
	settings Settings
	client   *http.Client
}

// NewClient creates a new generic RSS indexer client.
func NewClient(def *types.IndexerDefinition, settingsJSON map[string]string) *Client {
	s := Settings{
		URL:         settingsJSON["url"],
		Cookie:      settingsJSON["cookie"],
		ContentType: settingsJSON["contentType"],
	}
	if s.ContentType == "" {
		s.ContentType = "both"
	}

	return &Client{
		def:      def,
		settings: s,
		client:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Name() string {
	if c.def != nil {
		return c.def.Name
	}
	return "Generic RSS"
}

func (c *Client) Definition() *types.IndexerDefinition {
	return c.def
}

func (c *Client) GetSettings() map[string]string {
	return map[string]string{
		"url":         c.settings.URL,
		"cookie":      c.settings.Cookie,
		"contentType": c.settings.ContentType,
	}
}

func (c *Client) Test(ctx context.Context) error {
	releases, err := c.fetchFeed(ctx)
	if err != nil {
		return fmt.Errorf("feed fetch failed: %w", err)
	}
	if len(releases) == 0 {
		return fmt.Errorf("feed returned no items")
	}
	return nil
}

func (c *Client) Search(ctx context.Context, criteria types.SearchCriteria) ([]types.ReleaseInfo, error) {
	torrents, err := c.SearchTorrents(ctx, criteria)
	if err != nil {
		return nil, err
	}
	results := make([]types.ReleaseInfo, len(torrents))
	for i, t := range torrents {
		results[i] = t.ReleaseInfo
	}
	return results, nil
}

func (c *Client) SearchTorrents(ctx context.Context, criteria types.SearchCriteria) ([]types.TorrentInfo, error) {
	return c.fetchFeed(ctx)
}

func (c *Client) Download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.settings.Cookie != "" {
		req.Header.Set("Cookie", c.settings.Cookie)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	const maxDownloadSize = 50 * 1024 * 1024 // 50 MB
	return io.ReadAll(io.LimitReader(resp.Body, maxDownloadSize))
}

func (c *Client) Capabilities() *types.Capabilities {
	return &types.Capabilities{
		SupportsMovies: c.settings.ContentType == "movies" || c.settings.ContentType == "both",
		SupportsTV:     c.settings.ContentType == "tv" || c.settings.ContentType == "both",
		SupportsSearch: false,
		SupportsRSS:    true,
	}
}

func (c *Client) SupportsSearch() bool { return false }
func (c *Client) SupportsRSS() bool    { return true }

// fetchFeed fetches and parses the RSS feed.
func (c *Client) fetchFeed(ctx context.Context) ([]types.TorrentInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.settings.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "SlipStream/1.0")
	if c.settings.Cookie != "" {
		req.Header.Set("Cookie", c.settings.Cookie)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	const maxResponseSize = 10 * 1024 * 1024 // 10 MB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	indexerID := int64(0)
	indexerName := c.Name()
	if c.def != nil {
		indexerID = c.def.ID
	}

	return ParseFeed(body, indexerID, indexerName)
}

// DefinitionSchema returns the settings schema for the generic RSS indexer.
func DefinitionSchema() []cardigann.Setting {
	return []cardigann.Setting{
		{
			Name:    "url",
			Type:    "text",
			Label:   "RSS Feed URL",
			Default: "",
		},
		{
			Name:    "cookie",
			Type:    "password",
			Label:   "Cookie (optional)",
			Default: "",
		},
		{
			Name:  "contentType",
			Type:  "select",
			Label: "Content Type",
			Options: map[string]string{
				"both":   "Movies & TV",
				"movies": "Movies Only",
				"tv":     "TV Only",
			},
			Default: "both",
		},
	}
}

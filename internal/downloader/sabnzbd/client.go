// Package sabnzbd implements a SABnzbd API client.
// This is a stub implementation - full functionality will be added later.
package sabnzbd

import (
	"context"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

// Config holds the configuration for a SABnzbd client.
type Config struct {
	Host     string
	Port     int
	APIKey   string
	UseSSL   bool
	Category string
}

// Client implements a SABnzbd API client that satisfies the types.UsenetClient interface.
// This is currently a stub implementation.
type Client struct {
	config Config
}

// Compile-time check that Client implements UsenetClient.
var _ types.UsenetClient = (*Client)(nil)

// New creates a new SABnzbd client.
func New(cfg *Config) *Client {
	return &Client{
		config: *cfg,
	}
}

// NewFromConfig creates a client from a ClientConfig.
func NewFromConfig(cfg *types.ClientConfig) *Client {
	return New(&Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		APIKey:   cfg.APIKey,
		UseSSL:   cfg.UseSSL,
		Category: cfg.Category,
	})
}

// Type returns the client type.
func (c *Client) Type() types.ClientType {
	return types.ClientTypeSABnzbd
}

// Protocol returns the protocol.
func (c *Client) Protocol() types.Protocol {
	return types.ProtocolUsenet
}

// Test verifies the client connection.
// STUB: Returns not implemented error.
func (c *Client) Test(ctx context.Context) error {
	return types.ErrNotImplemented
}

// Connect establishes a connection.
// STUB: Returns not implemented error.
func (c *Client) Connect(ctx context.Context) error {
	return types.ErrNotImplemented
}

// Add adds an NZB to the client.
// STUB: Returns not implemented error.
func (c *Client) Add(ctx context.Context, opts *types.AddOptions) (string, error) {
	return "", types.ErrNotImplemented
}

// List returns all downloads.
// STUB: Returns empty list.
func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	return []types.DownloadItem{}, types.ErrNotImplemented
}

// Get retrieves a specific download by ID.
// STUB: Returns not implemented error.
func (c *Client) Get(ctx context.Context, id string) (*types.DownloadItem, error) {
	return nil, types.ErrNotImplemented
}

// Remove removes a download.
// STUB: Returns not implemented error.
func (c *Client) Remove(ctx context.Context, id string, deleteFiles bool) error {
	return types.ErrNotImplemented
}

// Pause pauses a download.
// STUB: Returns not implemented error.
func (c *Client) Pause(ctx context.Context, id string) error {
	return types.ErrNotImplemented
}

// Resume resumes a download.
// STUB: Returns not implemented error.
func (c *Client) Resume(ctx context.Context, id string) error {
	return types.ErrNotImplemented
}

// GetDownloadDir returns the default download directory.
// STUB: Returns not implemented error.
func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	return "", types.ErrNotImplemented
}

// GetQueue returns the current download queue.
// STUB: Returns empty list.
func (c *Client) GetQueue(ctx context.Context) ([]types.UsenetQueueItem, error) {
	return []types.UsenetQueueItem{}, types.ErrNotImplemented
}

// GetHistory returns download history.
// STUB: Returns empty list.
func (c *Client) GetHistory(ctx context.Context) ([]types.UsenetHistoryItem, error) {
	return []types.UsenetHistoryItem{}, types.ErrNotImplemented
}

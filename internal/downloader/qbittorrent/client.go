// Package qbittorrent implements a qBittorrent Web API client.
// This is a stub implementation - full functionality will be added later.
package qbittorrent

import (
	"context"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

// Config holds the configuration for a qBittorrent client.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	UseSSL   bool
	Category string
}

// Client implements a qBittorrent Web API client that satisfies the types.TorrentClient interface.
// This is currently a stub implementation.
type Client struct {
	config Config
}

// Compile-time check that Client implements TorrentClient.
var _ types.TorrentClient = (*Client)(nil)

// New creates a new qBittorrent client.
func New(cfg Config) *Client {
	return &Client{
		config: cfg,
	}
}

// NewFromConfig creates a client from a ClientConfig.
func NewFromConfig(cfg types.ClientConfig) *Client {
	return New(Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Username: cfg.Username,
		Password: cfg.Password,
		UseSSL:   cfg.UseSSL,
		Category: cfg.Category,
	})
}

// Type returns the client type.
func (c *Client) Type() types.ClientType {
	return types.ClientTypeQBittorrent
}

// Protocol returns the protocol.
func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
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

// Add adds a torrent to the client.
// STUB: Returns not implemented error.
func (c *Client) Add(ctx context.Context, opts types.AddOptions) (string, error) {
	return "", types.ErrNotImplemented
}

// AddMagnet adds a torrent from a magnet URL.
// STUB: Returns not implemented error.
func (c *Client) AddMagnet(ctx context.Context, magnetURL string, opts types.AddOptions) (string, error) {
	return "", types.ErrNotImplemented
}

// List returns all torrents.
// STUB: Returns empty list.
func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	return []types.DownloadItem{}, types.ErrNotImplemented
}

// Get retrieves a specific torrent by ID.
// STUB: Returns not implemented error.
func (c *Client) Get(ctx context.Context, id string) (*types.DownloadItem, error) {
	return nil, types.ErrNotImplemented
}

// Remove removes a torrent.
// STUB: Returns not implemented error.
func (c *Client) Remove(ctx context.Context, id string, deleteFiles bool) error {
	return types.ErrNotImplemented
}

// Pause stops a torrent.
// STUB: Returns not implemented error.
func (c *Client) Pause(ctx context.Context, id string) error {
	return types.ErrNotImplemented
}

// Resume starts a torrent.
// STUB: Returns not implemented error.
func (c *Client) Resume(ctx context.Context, id string) error {
	return types.ErrNotImplemented
}

// GetDownloadDir returns the default download directory.
// STUB: Returns not implemented error.
func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	return "", types.ErrNotImplemented
}

// SetSeedLimits configures seed ratio/time limits for a torrent.
// STUB: Returns not implemented error.
func (c *Client) SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error {
	return types.ErrNotImplemented
}

// GetTorrentInfo returns torrent-specific information.
// STUB: Returns not implemented error.
func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	return nil, types.ErrNotImplemented
}

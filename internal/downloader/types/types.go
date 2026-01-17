// Package types defines shared types for download clients.
package types

import (
	"context"
	"errors"
	"time"
)

// Common errors for download clients.
var (
	ErrNotImplemented = errors.New("operation not implemented")
	ErrNotConnected   = errors.New("client not connected")
	ErrAuthFailed     = errors.New("authentication failed")
	ErrNotFound       = errors.New("download not found")
)

// Protocol represents the download protocol.
type Protocol string

const (
	ProtocolTorrent Protocol = "torrent"
	ProtocolUsenet  Protocol = "usenet"
)

// ClientType represents the type of download client.
type ClientType string

const (
	ClientTypeTransmission ClientType = "transmission"
	ClientTypeQBittorrent  ClientType = "qbittorrent"
	ClientTypeDeluge       ClientType = "deluge"
	ClientTypeRTorrent     ClientType = "rtorrent"
	ClientTypeSABnzbd      ClientType = "sabnzbd"
	ClientTypeNZBGet       ClientType = "nzbget"
	ClientTypeMock         ClientType = "mock" // Mock client for developer mode
)

// ProtocolForClient returns the protocol for a given client type.
func ProtocolForClient(clientType ClientType) Protocol {
	switch clientType {
	case ClientTypeTransmission, ClientTypeQBittorrent, ClientTypeDeluge, ClientTypeRTorrent, ClientTypeMock:
		return ProtocolTorrent
	case ClientTypeSABnzbd, ClientTypeNZBGet:
		return ProtocolUsenet
	default:
		return ""
	}
}

// ClientConfig holds common configuration for all download clients.
type ClientConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	UseSSL   bool
	APIKey   string // For clients that use API keys (SABnzbd)
	Category string // Default category/label for downloads
}

// Client defines the common interface for all download clients.
type Client interface {
	// Info returns information about the client.
	Type() ClientType
	Protocol() Protocol

	// Connection
	Test(ctx context.Context) error
	Connect(ctx context.Context) error

	// Download operations
	Add(ctx context.Context, opts AddOptions) (string, error)
	List(ctx context.Context) ([]DownloadItem, error)
	Get(ctx context.Context, id string) (*DownloadItem, error)
	Remove(ctx context.Context, id string, deleteFiles bool) error

	// Control operations
	Pause(ctx context.Context, id string) error
	Resume(ctx context.Context, id string) error

	// Settings
	GetDownloadDir(ctx context.Context) (string, error)
}

// TorrentClient extends Client with torrent-specific operations.
type TorrentClient interface {
	Client

	// Torrent-specific operations
	AddMagnet(ctx context.Context, magnetURL string, opts AddOptions) (string, error)
	SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error
	GetTorrentInfo(ctx context.Context, id string) (*TorrentInfo, error)
}

// UsenetClient extends Client with usenet-specific operations.
type UsenetClient interface {
	Client

	// Usenet-specific operations
	GetQueue(ctx context.Context) ([]UsenetQueueItem, error)
	GetHistory(ctx context.Context) ([]UsenetHistoryItem, error)
}

// AddOptions specifies options for adding a download.
type AddOptions struct {
	// URL or file path/content for the download
	URL         string // URL to torrent/nzb file or magnet link
	FileContent []byte // Raw torrent/nzb file content

	// Metadata
	Name string // Display name for the download (used by mock client)

	// Destination
	DownloadDir string // Override default download directory
	Category    string // Category/label for the download

	// Control
	Paused bool // Add in paused state

	// Torrent-specific options
	SeedRatioLimit float64       // Stop seeding after this ratio (0 = use default)
	SeedTimeLimit  time.Duration // Stop seeding after this time (0 = use default)
}

// DownloadItem represents a download in progress or completed.
type DownloadItem struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Status         Status    `json:"status"`
	Progress       float64   `json:"progress"` // 0-100
	Size           int64     `json:"size"`
	DownloadedSize int64     `json:"downloadedSize"`
	DownloadSpeed  int64     `json:"downloadSpeed"` // bytes/sec
	UploadSpeed    int64     `json:"uploadSpeed"`   // bytes/sec (torrents only)
	ETA            int64     `json:"eta"`           // seconds, -1 if unavailable
	DownloadDir    string    `json:"downloadDir"`
	AddedAt        time.Time `json:"addedAt,omitempty"`
	CompletedAt    time.Time `json:"completedAt,omitempty"`
	Error          string    `json:"error,omitempty"`
}

// Status represents the status of a download.
type Status string

const (
	StatusQueued      Status = "queued"
	StatusDownloading Status = "downloading"
	StatusPaused      Status = "paused"
	StatusCompleted   Status = "completed"
	StatusSeeding     Status = "seeding"
	StatusError       Status = "error"
	StatusUnknown     Status = "unknown"
)

// TorrentInfo contains torrent-specific information.
type TorrentInfo struct {
	DownloadItem

	// Torrent-specific fields
	InfoHash  string  `json:"infoHash"`
	Seeders   int     `json:"seeders"`
	Leechers  int     `json:"leechers"`
	Ratio     float64 `json:"ratio"`
	IsPrivate bool    `json:"isPrivate"`
}

// UsenetQueueItem represents a usenet download in the queue.
type UsenetQueueItem struct {
	DownloadItem

	// Usenet-specific fields
	NZBName  string `json:"nzbName"`
	Category string `json:"category"`
	Priority int    `json:"priority"`
}

// UsenetHistoryItem represents a completed usenet download.
type UsenetHistoryItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"` // completed, failed, etc.
	Size        int64     `json:"size"`
	Category    string    `json:"category"`
	CompletedAt time.Time `json:"completedAt"`
	DownloadDir string    `json:"downloadDir"`
	Error       string    `json:"error,omitempty"`
}

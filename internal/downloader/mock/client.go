// Package mock provides a mock download client for developer mode.
package mock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

const (
	// DownloadDuration is how long a mock download takes to complete (seconds)
	DownloadDuration = 300.0
	// QueueDelay is how long items stay queued before starting (seconds)
	QueueDelay = 2.0
	// MockDownloadDir is the simulated download directory
	MockDownloadDir = "/mock/downloads/SlipStream"
)

// mockDownload represents an in-progress mock download.
type mockDownload struct {
	ID          string
	Name        string
	Size        int64
	DownloadDir string
	AddedAt     time.Time
	PausedAt    time.Time // Zero if not paused
	PausedTime  float64   // Total seconds spent paused
	Status      types.Status
	Completed   bool
}

// Client implements a mock download client for developer mode testing.
// It simulates download progress over time without actually downloading anything.
type Client struct {
	mu        sync.RWMutex
	downloads map[string]*mockDownload
}

// Singleton instance - shared across all mock client instances
var (
	instance     *Client
	instanceOnce sync.Once
)

// Compile-time check that Client implements TorrentClient.
var _ types.TorrentClient = (*Client)(nil)

// GetInstance returns the singleton mock client instance.
func GetInstance() *Client {
	instanceOnce.Do(func() {
		instance = &Client{
			downloads: make(map[string]*mockDownload),
		}
	})
	return instance
}

// New creates a reference to the singleton mock client.
func New() *Client {
	return GetInstance()
}

// NewFromConfig creates a client from a ClientConfig (config is ignored for mock).
func NewFromConfig(_ *types.ClientConfig) *Client {
	return GetInstance()
}

// Type returns the client type.
func (c *Client) Type() types.ClientType {
	return types.ClientTypeMock
}

// Protocol returns the protocol.
func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

// Test verifies the client connection (always succeeds for mock).
func (c *Client) Test(_ context.Context) error {
	return nil
}

// Connect establishes a connection (always succeeds for mock).
func (c *Client) Connect(_ context.Context) error {
	return nil
}

// Add adds a mock download.
func (c *Client) Add(_ context.Context, opts *types.AddOptions) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := generateMockID()

	// Determine name: prefer explicit Name, fallback to URL
	name := opts.Name
	if name == "" && opts.URL != "" {
		name = opts.URL
	}
	if name == "" {
		name = "Mock Download"
	}

	// Determine download directory
	downloadDir := MockDownloadDir
	if opts.DownloadDir != "" {
		downloadDir = opts.DownloadDir
	}

	// Generate a realistic file size (5-50 GB)
	size := int64(5+randInt(45)) * 1024 * 1024 * 1024

	download := &mockDownload{
		ID:          id,
		Name:        name,
		Size:        size,
		DownloadDir: downloadDir,
		AddedAt:     time.Now(),
		Status:      types.StatusQueued,
	}

	if opts.Paused {
		download.Status = types.StatusPaused
		download.PausedAt = time.Now()
	}

	c.downloads[id] = download

	return id, nil
}

// AddMagnet adds a mock magnet download.
func (c *Client) AddMagnet(_ context.Context, magnetURL string, opts *types.AddOptions) (string, error) {
	opts.URL = magnetURL
	return c.Add(context.TODO(), opts)
}

// List returns all mock downloads with calculated progress.
func (c *Client) List(_ context.Context) ([]types.DownloadItem, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	items := make([]types.DownloadItem, 0, len(c.downloads))
	now := time.Now()

	for _, d := range c.downloads {
		item := c.calculateProgress(d, now)
		items = append(items, item)
	}

	return items, nil
}

// Get returns a specific mock download.
func (c *Client) Get(_ context.Context, id string) (*types.DownloadItem, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	d, ok := c.downloads[id]
	if !ok {
		return nil, types.ErrNotFound
	}

	item := c.calculateProgress(d, time.Now())
	return &item, nil
}

// Remove removes a mock download.
func (c *Client) Remove(_ context.Context, id string, _ bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.downloads, id)
	return nil
}

// Pause pauses a mock download.
func (c *Client) Pause(_ context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	d, ok := c.downloads[id]
	if !ok {
		return types.ErrNotFound
	}

	if d.Status != types.StatusPaused && !d.Completed {
		d.Status = types.StatusPaused
		d.PausedAt = time.Now()
	}

	return nil
}

// Resume resumes a paused mock download.
func (c *Client) Resume(_ context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	d, ok := c.downloads[id]
	if !ok {
		return types.ErrNotFound
	}

	if d.Status == types.StatusPaused {
		// Accumulate paused time
		d.PausedTime += time.Since(d.PausedAt).Seconds()
		d.PausedAt = time.Time{}
		d.Status = types.StatusDownloading
	}

	return nil
}

// FastForward instantly completes a mock download.
func (c *Client) FastForward(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	d, ok := c.downloads[id]
	if !ok {
		return types.ErrNotFound
	}

	if d.Completed {
		return nil
	}

	d.Completed = true
	d.Status = types.StatusSeeding
	d.PausedAt = time.Time{}
	return nil
}

// GetDownloadDir returns the mock download directory.
func (c *Client) GetDownloadDir(_ context.Context) (string, error) {
	return MockDownloadDir, nil
}

// SetSeedLimits sets seed limits (no-op for mock).
func (c *Client) SetSeedLimits(_ context.Context, _ string, _ float64, _ time.Duration) error {
	return nil
}

// GetTorrentInfo returns torrent-specific info.
func (c *Client) GetTorrentInfo(_ context.Context, id string) (*types.TorrentInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	d, ok := c.downloads[id]
	if !ok {
		return nil, types.ErrNotFound
	}

	item := c.calculateProgress(d, time.Now())

	return &types.TorrentInfo{
		DownloadItem: item,
		InfoHash:     generateMockInfoHash(),
		Seeders:      50 + randInt(100),
		Leechers:     5 + randInt(20),
		Ratio:        0.0,
		IsPrivate:    true,
	}, nil
}

// calculateProgress computes the current state of a mock download.
func (c *Client) calculateProgress(d *mockDownload, now time.Time) types.DownloadItem {
	// Calculate effective elapsed time (excluding paused time)
	elapsed := now.Sub(d.AddedAt).Seconds() - d.PausedTime

	// If currently paused, don't count time since pause started
	if d.Status == types.StatusPaused && !d.PausedAt.IsZero() {
		elapsed -= now.Sub(d.PausedAt).Seconds()
	}

	var progress float64
	var status types.Status
	var downloadSpeed int64
	var eta int64

	switch {
	case d.Status == types.StatusPaused:
		// Calculate what progress was when paused
		downloadTime := elapsed - QueueDelay
		if downloadTime < 0 {
			downloadTime = 0
		}
		progress = (downloadTime / DownloadDuration) * 100
		if progress > 100 {
			progress = 100
		}
		status = types.StatusPaused
		downloadSpeed = 0
		eta = -1

	case d.Completed || elapsed >= QueueDelay+DownloadDuration:
		// Download complete
		progress = 100
		status = types.StatusSeeding
		downloadSpeed = 0
		eta = 0
		d.Completed = true
		d.Status = types.StatusSeeding

	case elapsed < QueueDelay:
		// Still in queue
		progress = 0
		status = types.StatusQueued
		downloadSpeed = 0
		eta = int64(QueueDelay + DownloadDuration - elapsed)

	default:
		// Actively downloading
		downloadTime := elapsed - QueueDelay
		progress = (downloadTime / DownloadDuration) * 100
		status = types.StatusDownloading
		d.Status = types.StatusDownloading

		// Calculate speed and ETA
		downloadSpeed = int64(float64(d.Size) / DownloadDuration)
		remainingProgress := 100 - progress
		eta = int64((remainingProgress / 100) * DownloadDuration)
	}

	downloadedSize := int64(float64(d.Size) * progress / 100)

	return types.DownloadItem{
		ID:             d.ID,
		Name:           d.Name,
		Status:         status,
		Progress:       progress,
		Size:           d.Size,
		DownloadedSize: downloadedSize,
		DownloadSpeed:  downloadSpeed,
		UploadSpeed:    0,
		ETA:            eta,
		DownloadDir:    d.DownloadDir,
		AddedAt:        d.AddedAt,
	}
}

// Clear removes all mock downloads (useful when disabling dev mode).
func (c *Client) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.downloads = make(map[string]*mockDownload)
}

// DownloadCount returns the number of active mock downloads.
func (c *Client) DownloadCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.downloads)
}

// generateMockID generates a random mock download ID.
func generateMockID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return "mock-" + hex.EncodeToString(bytes)
}

// generateMockInfoHash generates a random mock info hash.
func generateMockInfoHash() string {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

// randInt returns a random int between 0 and maxVal-1.
func randInt(maxVal int) int {
	if maxVal <= 0 {
		return 0
	}
	bytes := make([]byte, 1)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return int(bytes[0]) % maxVal
}

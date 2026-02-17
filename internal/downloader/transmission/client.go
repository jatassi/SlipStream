// Package transmission implements a Transmission RPC client.
package transmission

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

const (
	sessionIDHeader = "X-Transmission-Session-Id"
)

// Config holds the configuration for a Transmission client.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	UseSSL   bool
}

// Client implements a Transmission RPC client that satisfies the types.TorrentClient interface.
type Client struct {
	config     Config
	sessionID  string
	httpClient *http.Client
}

// Compile-time check that Client implements TorrentClient.
var _ types.TorrentClient = (*Client)(nil)

// New creates a new Transmission client.
func New(cfg *Config) *Client {
	return &Client{
		config: *cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewFromConfig creates a client from a ClientConfig.
func NewFromConfig(cfg *types.ClientConfig) *Client {
	return New(&Config{
		Host:     cfg.Host,
		Port:     cfg.Port,
		Username: cfg.Username,
		Password: cfg.Password,
		UseSSL:   cfg.UseSSL,
	})
}

// Type returns the client type.
func (c *Client) Type() types.ClientType {
	return types.ClientTypeTransmission
}

// Protocol returns the protocol.
func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

// Test verifies the client connection.
func (c *Client) Test(ctx context.Context) error {
	_, err := c.call("session-get", nil)
	return err
}

// GetSessionInfo returns the session-get arguments from the Transmission RPC.
func (c *Client) GetSessionInfo() (map[string]interface{}, error) {
	resp, err := c.call("session-get", nil)
	if err != nil {
		return nil, err
	}
	return resp.Arguments, nil
}

// Connect establishes a connection (for Transmission, this just validates the connection).
func (c *Client) Connect(ctx context.Context) error {
	return c.Test(ctx)
}

// Add adds a torrent to the client.
func (c *Client) Add(ctx context.Context, opts *types.AddOptions) (string, error) {
	args := make(map[string]interface{})

	// Set source
	switch {
	case opts.URL != "":
		args["filename"] = opts.URL
	case len(opts.FileContent) > 0:
		args["metainfo"] = base64.StdEncoding.EncodeToString(opts.FileContent)
	default:
		return "", fmt.Errorf("either URL or FileContent must be provided")
	}

	// Set download directory
	if opts.DownloadDir != "" {
		args["download-dir"] = opts.DownloadDir
	}

	// Set paused state
	if opts.Paused {
		args["paused"] = true
	}

	resp, err := c.call("torrent-add", args)
	if err != nil {
		return "", err
	}

	return c.extractTorrentID(resp)
}

// AddMagnet adds a torrent from a magnet URL.
func (c *Client) AddMagnet(ctx context.Context, magnetURL string, opts *types.AddOptions) (string, error) {
	opts.URL = magnetURL
	return c.Add(ctx, opts)
}

// AddURL adds a torrent from a URL with a subdirectory.
// This is a convenience method that constructs the full download path.
func (c *Client) AddURL(url, subDir string) (string, error) {
	opts := &types.AddOptions{
		URL: url,
	}

	// If a subdirectory is specified, get the default download dir and append it
	if subDir != "" {
		defaultDir, err := c.GetDownloadDir(context.Background())
		if err == nil {
			// Normalize the base path to use forward slashes
			normalizedBase := strings.ReplaceAll(defaultDir, "\\", "/")
			fullPath := path.Join(normalizedBase, subDir)
			opts.DownloadDir = fullPath
		}
	}

	id, err := c.Add(context.Background(), opts)
	if err != nil {
		return "", err
	}

	// Start the torrent
	if err := c.Resume(context.Background(), id); err != nil { //nolint:revive,staticcheck // Non-fatal error, intentionally ignored
		// Non-fatal error when starting torrent, log but continue
	}

	return id, nil
}

// List returns all torrents.
func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	args := map[string]interface{}{
		"fields": []string{
			"id", "name", "status", "percentDone",
			"totalSize", "downloadDir", "hashString",
			"eta", "rateDownload", "rateUpload",
			"downloadedEver", "sizeWhenDone", "error", "errorString",
		},
	}

	resp, err := c.call("torrent-get", args)
	if err != nil {
		return nil, err
	}

	torrentsRaw, ok := resp.Arguments["torrents"].([]interface{})
	if !ok {
		return []types.DownloadItem{}, nil
	}

	items := make([]types.DownloadItem, 0, len(torrentsRaw))
	for _, t := range torrentsRaw {
		torrent, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		items = append(items, c.mapToDownloadItem(torrent))
	}

	return items, nil
}

// Get retrieves a specific torrent by ID.
func (c *Client) Get(ctx context.Context, id string) (*types.DownloadItem, error) {
	args := map[string]interface{}{
		"ids": []string{id},
		"fields": []string{
			"id", "name", "status", "percentDone",
			"totalSize", "downloadDir", "hashString",
			"eta", "rateDownload", "rateUpload",
			"downloadedEver", "sizeWhenDone", "error", "errorString",
		},
	}

	resp, err := c.call("torrent-get", args)
	if err != nil {
		return nil, err
	}

	torrentsRaw, ok := resp.Arguments["torrents"].([]interface{})
	if !ok || len(torrentsRaw) == 0 {
		return nil, types.ErrNotFound
	}

	torrent, ok := torrentsRaw[0].(map[string]interface{})
	if !ok {
		return nil, types.ErrNotFound
	}

	item := c.mapToDownloadItem(torrent)
	return &item, nil
}

// Remove removes a torrent.
func (c *Client) Remove(ctx context.Context, id string, deleteFiles bool) error {
	args := map[string]interface{}{
		"ids":               []string{id},
		"delete-local-data": deleteFiles,
	}

	_, err := c.call("torrent-remove", args)
	return err
}

// Pause stops a torrent.
func (c *Client) Pause(ctx context.Context, id string) error {
	args := map[string]interface{}{
		"ids": []string{id},
	}

	_, err := c.call("torrent-stop", args)
	return err
}

// Resume starts a torrent.
func (c *Client) Resume(ctx context.Context, id string) error {
	args := map[string]interface{}{
		"ids": []string{id},
	}

	_, err := c.call("torrent-start", args)
	return err
}

// GetDownloadDir returns the default download directory from Transmission.
func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	resp, err := c.call("session-get", nil)
	if err != nil {
		return "", err
	}

	if downloadDir, ok := resp.Arguments["download-dir"].(string); ok {
		return downloadDir, nil
	}

	return "", fmt.Errorf("download-dir not found in session response")
}

// SetSeedLimits configures seed ratio/time limits for a torrent.
func (c *Client) SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error {
	args := map[string]interface{}{
		"ids": []string{id},
	}

	if ratio > 0 {
		args["seedRatioLimit"] = ratio
		args["seedRatioMode"] = 1 // Use torrent-specific limit
	}

	if seedTime > 0 {
		args["seedIdleLimit"] = int(seedTime.Minutes())
		args["seedIdleMode"] = 1 // Use torrent-specific limit
	}

	_, err := c.call("torrent-set", args)
	return err
}

// GetTorrentInfo returns torrent-specific information.
func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	args := map[string]interface{}{
		"ids": []string{id},
		"fields": []string{
			"id", "name", "status", "percentDone",
			"totalSize", "downloadDir", "hashString",
			"eta", "rateDownload", "rateUpload",
			"downloadedEver", "uploadedEver", "sizeWhenDone",
			"error", "errorString", "uploadRatio",
			"trackerStats", "isPrivate",
		},
	}

	resp, err := c.call("torrent-get", args)
	if err != nil {
		return nil, err
	}

	torrentsRaw, ok := resp.Arguments["torrents"].([]interface{})
	if !ok || len(torrentsRaw) == 0 {
		return nil, types.ErrNotFound
	}

	torrent, ok := torrentsRaw[0].(map[string]interface{})
	if !ok {
		return nil, types.ErrNotFound
	}

	item := c.mapToDownloadItem(torrent)
	info := &types.TorrentInfo{
		DownloadItem: item,
		InfoHash:     getString(torrent, "hashString"),
		Ratio:        getFloat(torrent, "uploadRatio"),
		IsPrivate:    getBool(torrent, "isPrivate"),
	}

	// Extract seeders/leechers from tracker stats
	if trackerStats, ok := torrent["trackerStats"].([]interface{}); ok && len(trackerStats) > 0 {
		if stat, ok := trackerStats[0].(map[string]interface{}); ok {
			info.Seeders = getInt(stat, "seederCount")
			info.Leechers = getInt(stat, "leecherCount")
		}
	}

	return info, nil
}

// Legacy methods for backwards compatibility

// Start starts a torrent (legacy method).
func (c *Client) Start(id string) error {
	return c.Resume(context.Background(), id)
}

// Stop stops a torrent (legacy method).
func (c *Client) Stop(id string) error {
	return c.Pause(context.Background(), id)
}

// RemoveWithData removes a torrent and deletes its data (legacy method).
func (c *Client) RemoveWithData(id string) error {
	return c.Remove(context.Background(), id, true)
}

// ListLegacy returns all torrents using the legacy Torrent struct.
func (c *Client) ListLegacy() ([]Torrent, error) {
	items, err := c.List(context.Background())
	if err != nil {
		return nil, err
	}

	torrents := make([]Torrent, 0, len(items))
	for i := range items {
		item := &items[i]
		torrents = append(torrents, Torrent{
			ID:             item.ID,
			Name:           item.Name,
			Status:         string(item.Status),
			Progress:       item.Progress / 100, // Convert from 0-100 to 0-1
			Size:           item.Size,
			DownloadedSize: item.DownloadedSize,
			DownloadSpeed:  item.DownloadSpeed,
			ETA:            item.ETA,
			Path:           item.DownloadDir,
		})
	}

	return torrents, nil
}

// Torrent represents a torrent in Transmission (legacy struct).
type Torrent struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Progress       float64 `json:"progress"`
	Size           int64   `json:"size"`
	DownloadedSize int64   `json:"downloadedSize"`
	DownloadSpeed  int64   `json:"downloadSpeed"`
	ETA            int64   `json:"eta"`
	Path           string  `json:"path"`
}

// rpcRequest represents a Transmission RPC request.
type rpcRequest struct {
	Method    string                 `json:"method"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// rpcResponse represents a Transmission RPC response.
type rpcResponse struct {
	Result    string                 `json:"result"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

func (c *Client) call(method string, args map[string]interface{}) (*rpcResponse, error) {
	req, err := c.buildRPCRequest(method, args)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return c.handleSessionConflict(resp, method, args)
	}

	return c.parseRPCResponse(resp)
}

func (c *Client) buildRPCRequest(method string, args map[string]interface{}) (*http.Request, error) {
	scheme := "http"
	if c.config.UseSSL {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d/transmission/rpc", scheme, c.config.Host, c.config.Port)

	body, err := json.Marshal(rpcRequest{Method: method, Arguments: args})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.sessionID != "" {
		req.Header.Set(sessionIDHeader, c.sessionID)
	}
	if c.config.Username != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.config.Username + ":" + c.config.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	return req, nil
}

func (c *Client) handleSessionConflict(resp *http.Response, method string, args map[string]interface{}) (*rpcResponse, error) {
	c.sessionID = resp.Header.Get(sessionIDHeader)
	if c.sessionID == "" {
		return nil, fmt.Errorf("received 409 but no session ID in response")
	}
	return c.call(method, args)
}

func (c *Client) parseRPCResponse(resp *http.Response) (*rpcResponse, error) {
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, types.ErrAuthFailed
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if rpcResp.Result != "success" {
		return nil, fmt.Errorf("RPC error: %s", rpcResp.Result)
	}

	return &rpcResp, nil
}

// mapToDownloadItem converts a Transmission torrent response to a DownloadItem.
func (c *Client) mapToDownloadItem(torrent map[string]interface{}) types.DownloadItem {
	status := mapStatus(getInt(torrent, "status"))
	progress := getFloat(torrent, "percentDone") * 100 // Convert from 0-1 to 0-100

	item := types.DownloadItem{
		ID:             getString(torrent, "hashString"),
		Name:           getString(torrent, "name"),
		Status:         status,
		Progress:       progress,
		Size:           int64(getFloat(torrent, "sizeWhenDone")),
		DownloadedSize: int64(getFloat(torrent, "downloadedEver")),
		DownloadSpeed:  int64(getFloat(torrent, "rateDownload")),
		UploadSpeed:    int64(getFloat(torrent, "rateUpload")),
		ETA:            int64(getFloat(torrent, "eta")),
		DownloadDir:    getString(torrent, "downloadDir"),
	}

	// Check for errors
	if errNum := getInt(torrent, "error"); errNum > 0 {
		item.Error = getString(torrent, "errorString")
		item.Status = types.StatusWarning
	}

	return item
}

// extractTorrentID extracts the torrent ID from an add response.
func (c *Client) extractTorrentID(resp *rpcResponse) (string, error) {
	// Check for torrent-added
	if torrentAdded, ok := resp.Arguments["torrent-added"].(map[string]interface{}); ok {
		if hashString, ok := torrentAdded["hashString"].(string); ok {
			return hashString, nil
		}
		if id, ok := torrentAdded["id"].(float64); ok {
			return fmt.Sprintf("%d", int(id)), nil
		}
	}

	// Check for duplicate
	if torrentDupe, ok := resp.Arguments["torrent-duplicate"].(map[string]interface{}); ok {
		if hashString, ok := torrentDupe["hashString"].(string); ok {
			return hashString, nil
		}
		if id, ok := torrentDupe["id"].(float64); ok {
			return fmt.Sprintf("%d", int(id)), nil
		}
	}

	return "", fmt.Errorf("could not extract torrent ID from response")
}

// mapStatus maps Transmission status codes to our status strings.
func mapStatus(status int) types.Status {
	switch status {
	case 0: // Stopped
		return types.StatusPaused
	case 1: // Queued to verify
		return types.StatusQueued
	case 2: // Verifying
		return types.StatusDownloading
	case 3: // Queued to download
		return types.StatusQueued
	case 4: // Downloading
		return types.StatusDownloading
	case 5: // Queued to seed
		return types.StatusSeeding
	case 6: // Seeding
		return types.StatusSeeding
	default:
		return types.StatusUnknown
	}
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

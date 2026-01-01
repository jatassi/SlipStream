package transmission

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
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

// Torrent represents a torrent in Transmission.
type Torrent struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Progress       float64 `json:"progress"`
	Size           int64   `json:"size"`
	DownloadedSize int64   `json:"downloadedSize"`
	DownloadSpeed  int64   `json:"downloadSpeed"`
	ETA            int64   `json:"eta"` // seconds, -1 if unavailable
	Path           string  `json:"path"`
}

// Client implements a Transmission RPC client.
type Client struct {
	config    Config
	sessionID string
	client    *http.Client
}

// New creates a new Transmission client.
func New(cfg Config) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the client name.
func (c *Client) Name() string {
	return "Transmission"
}

// Test verifies the client connection.
func (c *Client) Test() error {
	_, err := c.call("session-get", nil)
	return err
}

// Add adds a torrent to the client.
func (c *Client) Add(torrent Torrent) error {
	args := map[string]interface{}{}

	// Check if the path is a URL or file path
	if isURL(torrent.Path) {
		args["filename"] = torrent.Path
	} else {
		// Read file and encode as base64
		args["metainfo"] = torrent.Path // Assume already base64 encoded
	}

	if torrent.Name != "" {
		args["download-dir"] = torrent.Name
	}

	_, err := c.call("torrent-add", args)
	return err
}

// GetDownloadDir returns the default download directory from Transmission.
func (c *Client) GetDownloadDir() (string, error) {
	resp, err := c.call("session-get", nil)
	if err != nil {
		return "", err
	}

	if downloadDir, ok := resp.Arguments["download-dir"].(string); ok {
		return downloadDir, nil
	}

	return "", fmt.Errorf("download-dir not found in session response")
}

// AddURL adds a torrent from a URL (magnet or .torrent file URL).
// If subDir is provided, the torrent will be downloaded to {default-download-dir}/{subDir}
func (c *Client) AddURL(url string, subDir string) (string, error) {
	args := map[string]interface{}{
		"filename": url,
	}

	// If a subdirectory is specified, get the default download dir and append it
	if subDir != "" {
		defaultDir, err := c.GetDownloadDir()
		if err != nil {
			log.Printf("[Transmission] Warning: could not get default download dir: %v", err)
		} else {
			// Normalize the base path to use forward slashes (works on both Windows and Linux)
			// Transmission accepts forward slashes on all platforms
			normalizedBase := strings.ReplaceAll(defaultDir, "\\", "/")
			fullPath := path.Join(normalizedBase, subDir)
			args["download-dir"] = fullPath
			log.Printf("[Transmission] Using download dir: %s", fullPath)
		}
	}

	log.Printf("[Transmission] AddURL called with url=%s", url)

	resp, err := c.call("torrent-add", args)
	if err != nil {
		log.Printf("[Transmission] AddURL error: %v", err)
		return "", err
	}

	// Log the full response for debugging
	respJSON, _ := json.Marshal(resp.Arguments)
	log.Printf("[Transmission] AddURL response: result=%s, arguments=%s", resp.Result, string(respJSON))

	// Parse the response to get the torrent ID
	if torrentAdded, ok := resp.Arguments["torrent-added"].(map[string]interface{}); ok {
		log.Printf("[Transmission] Found torrent-added: %+v", torrentAdded)
		if id, ok := torrentAdded["id"].(float64); ok {
			return fmt.Sprintf("%d", int(id)), nil
		}
		if hashString, ok := torrentAdded["hashString"].(string); ok {
			return hashString, nil
		}
	}

	// Check for duplicate
	if torrentDupe, ok := resp.Arguments["torrent-duplicate"].(map[string]interface{}); ok {
		log.Printf("[Transmission] Found torrent-duplicate: %+v", torrentDupe)
		if id, ok := torrentDupe["id"].(float64); ok {
			return fmt.Sprintf("%d", int(id)), nil
		}
		if hashString, ok := torrentDupe["hashString"].(string); ok {
			return hashString, nil
		}
	}

	log.Printf("[Transmission] AddURL: no torrent-added or torrent-duplicate found in response")
	return "", nil
}

// List returns all torrents.
func (c *Client) List() ([]Torrent, error) {
	args := map[string]interface{}{
		"fields": []string{
			"id", "name", "status", "percentDone",
			"totalSize", "downloadDir", "hashString",
			"eta", "rateDownload", "downloadedEver", "sizeWhenDone",
		},
	}

	resp, err := c.call("torrent-get", args)
	if err != nil {
		return nil, err
	}

	torrentsRaw, ok := resp.Arguments["torrents"].([]interface{})
	if !ok {
		return []Torrent{}, nil
	}

	torrents := make([]Torrent, 0, len(torrentsRaw))
	for _, t := range torrentsRaw {
		torrent, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		item := Torrent{
			ID:             fmt.Sprintf("%v", torrent["hashString"]),
			Name:           getString(torrent, "name"),
			Status:         mapStatus(getInt(torrent, "status")),
			Progress:       getFloat(torrent, "percentDone"),
			Size:           int64(getFloat(torrent, "sizeWhenDone")),
			DownloadedSize: int64(getFloat(torrent, "downloadedEver")),
			DownloadSpeed:  int64(getFloat(torrent, "rateDownload")),
			ETA:            int64(getFloat(torrent, "eta")),
			Path:           getString(torrent, "downloadDir"),
		}
		torrents = append(torrents, item)
	}

	return torrents, nil
}

// Remove removes a torrent.
func (c *Client) Remove(id string) error {
	args := map[string]interface{}{
		"ids":               []string{id},
		"delete-local-data": false,
	}

	_, err := c.call("torrent-remove", args)
	return err
}

// Start starts a torrent.
func (c *Client) Start(id string) error {
	args := map[string]interface{}{
		"ids": []string{id},
	}

	_, err := c.call("torrent-start", args)
	return err
}

// Stop stops a torrent.
func (c *Client) Stop(id string) error {
	args := map[string]interface{}{
		"ids": []string{id},
	}

	_, err := c.call("torrent-stop", args)
	return err
}

// RemoveWithData removes a torrent and deletes its data.
func (c *Client) RemoveWithData(id string) error {
	args := map[string]interface{}{
		"ids":               []string{id},
		"delete-local-data": true,
	}

	_, err := c.call("torrent-remove", args)
	return err
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
	// Build URL
	scheme := "http"
	if c.config.UseSSL {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d/transmission/rpc", scheme, c.config.Host, c.config.Port)

	// Build request body
	reqBody := rpcRequest{
		Method:    method,
		Arguments: args,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add session ID if we have one
	if c.sessionID != "" {
		req.Header.Set(sessionIDHeader, c.sessionID)
	}

	// Add authentication if configured
	if c.config.Username != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(c.config.Username + ":" + c.config.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Handle 409 Conflict - need to get session ID
	if resp.StatusCode == http.StatusConflict {
		c.sessionID = resp.Header.Get(sessionIDHeader)
		if c.sessionID == "" {
			return nil, fmt.Errorf("received 409 but no session ID in response")
		}
		// Retry the request with the new session ID
		return c.call(method, args)
	}

	// Handle unauthorized
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed: invalid username or password")
	}

	// Handle other errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
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

// mapStatus maps Transmission status codes to our status strings.
func mapStatus(status int) string {
	switch status {
	case 0: // Stopped
		return "paused"
	case 1: // Queued to verify
		return "downloading"
	case 2: // Verifying
		return "downloading"
	case 3: // Queued to download
		return "downloading"
	case 4: // Downloading
		return "downloading"
	case 5: // Queued to seed
		return "completed"
	case 6: // Seeding
		return "completed"
	default:
		return "unknown"
	}
}

func isURL(s string) bool {
	return len(s) > 7 && (s[:7] == "http://" || s[:8] == "https://" || s[:7] == "magnet:")
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

package flood

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

var _ types.TorrentClient = (*Client)(nil)

type Client struct {
	config     types.ClientConfig
	httpClient *http.Client
}

func NewFromConfig(cfg *types.ClientConfig) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		config: *cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
	}
}

func (c *Client) Type() types.ClientType {
	return types.ClientTypeFlood
}

func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

func (c *Client) Test(ctx context.Context) error {
	if err := c.authenticate(ctx); err != nil {
		return err
	}

	resp, err := c.doRequest(ctx, http.MethodGet, "/api/client/connection-test", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result struct {
		IsConnected bool `json:"isConnected"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode connection test response: %w", err)
	}

	if !result.IsConnected {
		return types.ErrNotConnected
	}

	return nil
}

func (c *Client) Connect(ctx context.Context) error {
	return c.authenticate(ctx)
}

func (c *Client) Add(ctx context.Context, opts *types.AddOptions) (string, error) {
	if opts.URL != "" {
		return c.AddMagnet(ctx, opts.URL, opts)
	}
	if len(opts.FileContent) > 0 {
		return c.addFile(ctx, opts)
	}
	return "", fmt.Errorf("either URL or FileContent must be provided")
}

func (c *Client) AddMagnet(ctx context.Context, magnetURL string, opts *types.AddOptions) (string, error) {
	body := map[string]any{
		"urls":  []string{magnetURL},
		"start": true,
	}
	if opts != nil {
		if opts.Paused {
			body["start"] = false
		}
		if opts.DownloadDir != "" {
			body["destination"] = opts.DownloadDir
		}
		if tags := c.buildTags(opts); len(tags) > 0 {
			body["tags"] = tags
		}
	}

	resp, err := c.request(ctx, http.MethodPost, "/api/torrents/add-urls", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	items, err := c.List(ctx)
	if err != nil {
		return "", err
	}
	if len(items) > 0 {
		return items[len(items)-1].ID, nil
	}
	return "", nil
}

func (c *Client) addFile(ctx context.Context, opts *types.AddOptions) (string, error) {
	b64Content := base64.StdEncoding.EncodeToString(opts.FileContent)

	body := map[string]any{
		"files": []string{b64Content},
		"start": true,
	}
	if opts.Paused {
		body["start"] = false
	}
	if opts.DownloadDir != "" {
		body["destination"] = opts.DownloadDir
	}
	if tags := c.buildTags(opts); len(tags) > 0 {
		body["tags"] = tags
	}

	resp, err := c.request(ctx, http.MethodPost, "/api/torrents/add-files", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	items, err := c.List(ctx)
	if err != nil {
		return "", err
	}
	if len(items) > 0 {
		return items[len(items)-1].ID, nil
	}
	return "", nil
}

func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	resp, err := c.request(ctx, http.MethodGet, "/api/torrents", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID       int                      `json:"id"`
		Torrents map[string]*floodTorrent `json:"torrents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return []types.DownloadItem{}, nil
	}

	items := make([]types.DownloadItem, 0, len(result.Torrents))
	for hash, torrent := range result.Torrents {
		items = append(items, mapToDownloadItem(hash, torrent))
	}

	return items, nil
}

func (c *Client) Get(ctx context.Context, id string) (*types.DownloadItem, error) {
	items, err := c.List(ctx)
	if err != nil {
		return nil, err
	}

	lowerID := strings.ToLower(id)
	for i := range items {
		if items[i].ID == lowerID {
			return &items[i], nil
		}
	}

	return nil, types.ErrNotFound
}

func (c *Client) Remove(ctx context.Context, id string, deleteFiles bool) error {
	body := map[string]any{
		"hashes":     []string{strings.ToUpper(id)},
		"deleteData": deleteFiles,
	}

	resp, err := c.request(ctx, http.MethodPost, "/api/torrents/delete", body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) Pause(ctx context.Context, id string) error {
	body := map[string]any{
		"hashes": []string{strings.ToUpper(id)},
	}

	resp, err := c.request(ctx, http.MethodPost, "/api/torrents/stop", body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) Resume(ctx context.Context, id string) error {
	body := map[string]any{
		"hashes": []string{strings.ToUpper(id)},
	}

	resp, err := c.request(ctx, http.MethodPost, "/api/torrents/start", body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	resp, err := c.request(ctx, http.MethodGet, "/api/client/settings", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		DirectoryDefault string `json:"directoryDefault"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode settings response: %w", err)
	}

	return result.DirectoryDefault, nil
}

func (c *Client) SetSeedLimits(_ context.Context, _ string, _ float64, _ time.Duration) error {
	return types.ErrNotImplemented
}

func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	resp, err := c.request(ctx, http.MethodGet, "/api/torrents", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID       int                      `json:"id"`
		Torrents map[string]*floodTorrent `json:"torrents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode torrents response: %w", err)
	}

	upperID := strings.ToUpper(id)
	torrent, ok := result.Torrents[upperID]
	if !ok {
		return nil, types.ErrNotFound
	}

	item := mapToDownloadItem(upperID, torrent)
	return &types.TorrentInfo{
		DownloadItem: item,
		InfoHash:     strings.ToLower(upperID),
		Ratio:        torrent.Ratio,
	}, nil
}

type floodTorrent struct {
	Hash            string   `json:"hash"`
	Name            string   `json:"name"`
	Status          []string `json:"status"`
	PercentComplete float64  `json:"percentComplete"`
	SizeBytes       int64    `json:"sizeBytes"`
	BytesDone       int64    `json:"bytesDone"`
	DownRate        int64    `json:"downRate"`
	UpRate          int64    `json:"upRate"`
	ETA             int64    `json:"eta"`
	Directory       string   `json:"directory"`
	Ratio           float64  `json:"ratio"`
	DateAdded       int64    `json:"dateAdded"`
	Message         string   `json:"message"`
	Tags            []string `json:"tags"`
}

func mapToDownloadItem(hash string, t *floodTorrent) types.DownloadItem {
	status := mapStatus(t.Status)

	item := types.DownloadItem{
		ID:             strings.ToLower(hash),
		Name:           t.Name,
		Status:         status,
		Progress:       t.PercentComplete,
		Size:           t.SizeBytes,
		DownloadedSize: t.BytesDone,
		DownloadSpeed:  t.DownRate,
		UploadSpeed:    t.UpRate,
		ETA:            t.ETA,
		DownloadDir:    t.Directory,
	}

	if t.DateAdded > 0 {
		item.AddedAt = time.Unix(t.DateAdded, 0)
	}

	if status == types.StatusWarning {
		item.Error = t.Message
	}

	return item
}

func mapStatus(statuses []string) types.Status {
	statusSet := make(map[string]bool, len(statuses))
	for _, s := range statuses {
		statusSet[s] = true
	}

	if statusSet["error"] {
		return types.StatusWarning
	}
	if statusSet["checking"] {
		return types.StatusQueued
	}
	if statusSet["downloading"] {
		return types.StatusDownloading
	}
	if statusSet["seeding"] {
		return types.StatusSeeding
	}
	if statusSet["complete"] {
		return types.StatusCompleted
	}
	if statusSet["stopped"] || statusSet["inactive"] {
		return types.StatusPaused
	}

	return types.StatusUnknown
}

func (c *Client) authenticate(ctx context.Context) error {
	c.httpClient.Jar, _ = cookiejar.New(nil)

	body := map[string]string{
		"username": c.config.Username,
		"password": c.config.Password,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/auth/authenticate", body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return types.ErrAuthFailed
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) request(ctx context.Context, method, path string, body any) (*http.Response, error) {
	resp, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := c.authenticate(ctx); err != nil {
			return nil, err
		}
		return c.doRequest(ctx, method, path, body)
	}

	return resp, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	scheme := "http"
	if c.config.UseSSL {
		scheme = "https"
	}

	urlBase := ""
	if c.config.URLBase != "" {
		urlBase = "/" + strings.Trim(c.config.URLBase, "/")
	}

	reqURL := fmt.Sprintf("%s://%s:%d%s%s", scheme, c.config.Host, c.config.Port, urlBase, path)

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

func (c *Client) buildTags(opts *types.AddOptions) []string {
	var tags []string
	category := opts.Category
	if category == "" {
		category = c.config.Category
	}
	if category != "" {
		tags = append(tags, category)
	}
	return tags
}

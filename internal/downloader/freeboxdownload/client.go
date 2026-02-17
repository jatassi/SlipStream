package freeboxdownload

import (
	"context"
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // SHA1 required by Freebox API
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

var _ types.TorrentClient = (*Client)(nil)

type Client struct {
	config       *types.ClientConfig
	baseURL      string
	httpClient   *http.Client
	sessionToken string
}

type responseEnvelope struct {
	Success   bool            `json:"success"`
	Result    json.RawMessage `json:"result,omitempty"`
	Msg       string          `json:"msg,omitempty"`
	ErrorCode string          `json:"error_code,omitempty"`
}

type loginResult struct {
	Challenge string `json:"challenge"`
}

type sessionResult struct {
	SessionToken string `json:"session_token"`
}

type downloadItem struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	RxBytes     int64  `json:"rx_bytes"`
	TxBytes     int64  `json:"tx_bytes"`
	RxRate      int64  `json:"rx_rate"`
	TxRate      int64  `json:"tx_rate"`
	RxPct       int    `json:"rx_pct"`
	Status      string `json:"status"`
	ETA         int    `json:"eta"`
	Error       string `json:"error"`
	DownloadDir string `json:"download_dir"`
	InfoHash    string `json:"info_hash"`
	CreatedTS   int64  `json:"created_ts"`
	StopRatio   int    `json:"stop_ratio"`
}

type downloadConfig struct {
	DownloadDir string `json:"download_dir"`
}

type addResult struct {
	ID int `json:"id"`
}

func NewFromConfig(cfg *types.ClientConfig) *Client {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s:%d/api/v1", scheme, cfg.Host, cfg.Port)

	return &Client{
		config:  cfg,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Type() types.ClientType {
	return types.ClientTypeFreeboxDownload
}

func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

func (c *Client) Test(ctx context.Context) error {
	return c.Connect(ctx)
}

func (c *Client) Connect(ctx context.Context) error {
	loginResp, err := c.doRequest(ctx, "GET", "/login", nil, nil, false)
	if err != nil {
		return err
	}

	var result loginResult
	if err := json.Unmarshal(loginResp, &result); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	h := hmac.New(sha1.New, []byte(c.config.APIKey))
	h.Write([]byte(result.Challenge))
	password := hex.EncodeToString(h.Sum(nil))

	sessionPayload := map[string]string{
		"app_id":   "slipstream",
		"password": password,
	}
	sessionResp, err := c.doRequest(ctx, "POST", "/login/session", sessionPayload, nil, false)
	if err != nil {
		return err
	}

	var sessionRes sessionResult
	if err := json.Unmarshal(sessionResp, &sessionRes); err != nil {
		return fmt.Errorf("failed to parse session response: %w", err)
	}

	c.sessionToken = sessionRes.SessionToken
	return nil
}

func (c *Client) Add(ctx context.Context, opts *types.AddOptions) (string, error) {
	if opts.FileContent != nil {
		return c.addFile(ctx, opts)
	}
	if opts.URL != "" {
		return c.addURL(ctx, opts)
	}
	return "", fmt.Errorf("either URL or FileContent must be provided")
}

func (c *Client) AddMagnet(ctx context.Context, magnetURL string, opts *types.AddOptions) (string, error) {
	if opts == nil {
		opts = &types.AddOptions{}
	}
	opts.URL = magnetURL
	return c.Add(ctx, opts)
}

func (c *Client) addURL(ctx context.Context, opts *types.AddOptions) (string, error) {
	form := url.Values{}
	form.Set("download_url", opts.URL)

	if opts.DownloadDir != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(opts.DownloadDir))
		form.Set("download_dir", encoded)
	}

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	result, err := c.doRequest(ctx, "POST", "/downloads/add", []byte(form.Encode()), headers, true)
	if err != nil {
		return "", err
	}

	var addRes addResult
	if err := json.Unmarshal(result, &addRes); err != nil {
		return "", fmt.Errorf("failed to parse add response: %w", err)
	}

	return strconv.Itoa(addRes.ID), nil
}

func (c *Client) addFile(ctx context.Context, opts *types.AddOptions) (string, error) {
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer writer.Close()

		part, err := writer.CreateFormFile("download_file", opts.Name)
		if err != nil {
			return
		}
		if _, err := part.Write(opts.FileContent); err != nil {
			return
		}

		if opts.DownloadDir != "" {
			encoded := base64.StdEncoding.EncodeToString([]byte(opts.DownloadDir))
			if err := writer.WriteField("download_dir", encoded); err != nil {
				return
			}
		}
	}()

	headers := map[string]string{
		"Content-Type": writer.FormDataContentType(),
	}

	result, err := c.doRequest(ctx, "POST", "/downloads/add", pr, headers, true)
	if err != nil {
		return "", err
	}

	var addRes addResult
	if err := json.Unmarshal(result, &addRes); err != nil {
		return "", fmt.Errorf("failed to parse add response: %w", err)
	}

	return strconv.Itoa(addRes.ID), nil
}

func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	result, err := c.doRequest(ctx, "GET", "/downloads/", nil, nil, true)
	if err != nil {
		return nil, err
	}

	var items []downloadItem
	if err := json.Unmarshal(result, &items); err != nil {
		return nil, fmt.Errorf("failed to parse list response: %w", err)
	}

	downloads := make([]types.DownloadItem, 0, len(items))
	for i := range items {
		downloads = append(downloads, c.convertDownloadItem(&items[i]))
	}

	return downloads, nil
}

func (c *Client) Get(ctx context.Context, id string) (*types.DownloadItem, error) {
	items, err := c.List(ctx)
	if err != nil {
		return nil, err
	}

	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}

	return nil, types.ErrNotFound
}

func (c *Client) Remove(ctx context.Context, id string, deleteFiles bool) error {
	endpoint := fmt.Sprintf("/downloads/%s", id)
	if deleteFiles {
		endpoint += "/erase"
	}

	_, err := c.doRequest(ctx, "DELETE", endpoint, nil, nil, true)
	return err
}

func (c *Client) Pause(ctx context.Context, id string) error {
	payload := map[string]string{
		"status": "stopped",
	}
	_, err := c.doRequest(ctx, "PUT", fmt.Sprintf("/downloads/%s", id), payload, nil, true)
	return err
}

func (c *Client) Resume(ctx context.Context, id string) error {
	payload := map[string]string{
		"status": "downloading",
	}
	_, err := c.doRequest(ctx, "PUT", fmt.Sprintf("/downloads/%s", id), payload, nil, true)
	return err
}

func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	result, err := c.doRequest(ctx, "GET", "/downloads/config/", nil, nil, true)
	if err != nil {
		return "", err
	}

	var config downloadConfig
	if err := json.Unmarshal(result, &config); err != nil {
		return "", fmt.Errorf("failed to parse config response: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(config.DownloadDir)
	if err != nil {
		return "", fmt.Errorf("failed to decode download dir: %w", err)
	}

	return string(decoded), nil
}

func (c *Client) SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error {
	payload := map[string]int{
		"stop_ratio": int(ratio * 100),
	}
	_, err := c.doRequest(ctx, "PUT", fmt.Sprintf("/downloads/%s", id), payload, nil, true)
	return err
}

func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	result, err := c.doRequest(ctx, "GET", "/downloads/", nil, nil, true)
	if err != nil {
		return nil, err
	}

	var items []downloadItem
	if err := json.Unmarshal(result, &items); err != nil {
		return nil, fmt.Errorf("failed to parse list response: %w", err)
	}

	var found *downloadItem
	for i := range items {
		if strconv.Itoa(items[i].ID) == id {
			found = &items[i]
			break
		}
	}

	if found == nil {
		return nil, types.ErrNotFound
	}

	ratio := 0.0
	if found.RxBytes > 0 {
		ratio = float64(found.TxBytes) / float64(found.RxBytes)
	}

	return &types.TorrentInfo{
		DownloadItem: c.convertDownloadItem(found),
		InfoHash:     found.InfoHash,
		Ratio:        ratio,
	}, nil
}

func (c *Client) convertDownloadItem(item *downloadItem) types.DownloadItem {
	status := c.mapStatus(item.Status, item.Error)
	progress := float64(item.RxPct) / 100.0

	eta := int64(item.ETA)
	if eta <= 0 {
		eta = -1
	}

	downloadDir := item.DownloadDir
	if decoded, err := base64.StdEncoding.DecodeString(item.DownloadDir); err == nil {
		downloadDir = string(decoded)
	}

	var addedAt, completedAt time.Time
	if item.CreatedTS > 0 {
		addedAt = time.Unix(item.CreatedTS, 0)
	}

	if status == types.StatusCompleted || status == types.StatusSeeding {
		completedAt = time.Now()
	}

	dl := types.DownloadItem{
		ID:             strconv.Itoa(item.ID),
		Name:           item.Name,
		Status:         status,
		Progress:       progress,
		Size:           item.Size,
		DownloadedSize: item.RxBytes,
		DownloadSpeed:  item.RxRate,
		UploadSpeed:    item.TxRate,
		ETA:            eta,
		DownloadDir:    downloadDir,
		AddedAt:        addedAt,
		CompletedAt:    completedAt,
	}

	if item.Error != "" && item.Error != "none" {
		dl.Error = item.Error
	}

	return dl
}

func (c *Client) mapStatus(status, errMsg string) types.Status {
	if errMsg != "" && errMsg != "none" {
		return types.StatusWarning
	}

	switch status {
	case "stopped", "stopping":
		return types.StatusPaused
	case "queued":
		return types.StatusQueued
	case "starting", "downloading", "retry", "checking":
		return types.StatusDownloading
	case "error":
		return types.StatusError
	case "done":
		return types.StatusCompleted
	case "seeding":
		return types.StatusSeeding
	default:
		return types.StatusDownloading
	}
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}, headers map[string]string, needsAuth bool) (json.RawMessage, error) {
	if needsAuth && c.sessionToken == "" {
		if err := c.Connect(ctx); err != nil {
			return nil, err
		}
	}

	result, err := c.doRequestInternal(ctx, method, endpoint, body, headers, needsAuth)
	if err != nil && needsAuth && (errors.Is(err, types.ErrAuthFailed) || strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403")) {
		c.sessionToken = ""
		if err := c.Connect(ctx); err != nil {
			return nil, err
		}
		return c.doRequestInternal(ctx, method, endpoint, body, headers, needsAuth)
	}

	return result, err
}

//nolint:gocognit,gocyclo // Complexity required for handling multiple body types
func (c *Client) doRequestInternal(ctx context.Context, method, endpoint string, body interface{}, headers map[string]string, needsAuth bool) (json.RawMessage, error) {
	var bodyReader io.Reader

	switch v := body.(type) {
	case []byte:
		bodyReader = strings.NewReader(string(v))
	case io.Reader:
		bodyReader = v
	case map[string]string, map[string]int:
		jsonBody, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = strings.NewReader(string(jsonBody))
		if headers == nil {
			headers = make(map[string]string)
		}
		if _, exists := headers["Content-Type"]; !exists {
			headers["Content-Type"] = "application/json"
		}
	case nil:
	default:
		return nil, fmt.Errorf("unsupported body type: %T", body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if needsAuth && c.sessionToken != "" {
		req.Header.Set("X-Fbx-App-Auth", c.sessionToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, types.ErrAuthFailed
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var envelope responseEnvelope
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse response envelope: %w", err)
	}

	if !envelope.Success {
		if envelope.ErrorCode != "" {
			return nil, fmt.Errorf("API error: %s - %s", envelope.ErrorCode, envelope.Msg)
		}
		return nil, fmt.Errorf("API error: %s", envelope.Msg)
	}

	return envelope.Result, nil
}

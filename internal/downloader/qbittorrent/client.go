package qbittorrent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

var _ types.TorrentClient = (*Client)(nil)

type Client struct {
	config     *types.ClientConfig
	httpClient *http.Client
	baseURL    string
	mu         sync.Mutex
	cookies    []*http.Cookie
}

type qbitTorrent struct {
	Hash        string  `json:"hash"`
	Name        string  `json:"name"`
	Size        int64   `json:"size"`
	Progress    float64 `json:"progress"`
	ETA         int64   `json:"eta"`
	State       string  `json:"state"`
	Category    string  `json:"category"`
	SavePath    string  `json:"save_path"`
	ContentPath string  `json:"content_path"`
	Ratio       float64 `json:"ratio"`
	DLSpeed     int64   `json:"dlspeed"`
	UPSpeed     int64   `json:"upspeed"`
	AmountLeft  int64   `json:"amount_left"`
	Completed   int64   `json:"completed"`
	TotalSize   int64   `json:"total_size"`
}

type qbitPreferences struct {
	SavePath string `json:"save_path"`
}

type qbitProperties struct {
	Hash        string  `json:"hash"`
	SavePath    string  `json:"save_path"`
	SeedingTime int     `json:"seeding_time"`
	ShareRatio  float64 `json:"share_ratio"`
}

func NewFromConfig(cfg *types.ClientConfig) *Client {
	jar, _ := cookiejar.New(nil)

	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}

	urlBase := cfg.URLBase
	if urlBase == "" {
		urlBase = "/"
	}
	if !strings.HasPrefix(urlBase, "/") {
		urlBase = "/" + urlBase
	}
	if !strings.HasSuffix(urlBase, "/") {
		urlBase += "/"
	}

	baseURL := fmt.Sprintf("%s://%s:%d%s", scheme, cfg.Host, cfg.Port, urlBase)

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
		baseURL: baseURL,
	}
}

func (c *Client) Type() types.ClientType {
	return types.ClientTypeQBittorrent
}

func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

func (c *Client) Test(ctx context.Context) error { //nolint:gocyclo // auth + version validation
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"api/v2/app/version", http.NoBody)
	if err != nil {
		return err
	}

	if err := c.authenticate(ctx); err != nil {
		return err
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return types.ErrAuthFailed
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read version response: %w", err)
	}

	version := strings.TrimSpace(string(body))
	if version == "" {
		return fmt.Errorf("empty version response from qBittorrent")
	}

	ok, err := types.CompareVersions(version, "4.1.0")
	if err != nil {
		return fmt.Errorf("failed to parse qBittorrent version %q: %w", version, err)
	}
	if !ok {
		return fmt.Errorf("qBittorrent version %s is below minimum required version 4.1.0 (Web API v2)", version)
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
		return c.addTorrentFile(ctx, opts)
	}

	return "", fmt.Errorf("no URL or file content provided")
}

func (c *Client) AddMagnet(ctx context.Context, magnetURL string, opts *types.AddOptions) (string, error) {
	if err := c.authenticate(ctx); err != nil {
		return "", err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("urls", magnetURL); err != nil {
		return "", err
	}

	if err := c.writeAddOptions(writer, opts); err != nil {
		return "", err
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	if err := c.submitAdd(ctx, body, writer.FormDataContentType()); err != nil {
		return "", err
	}

	hash := extractHashFromMagnet(magnetURL)

	if opts != nil && (opts.SeedRatioLimit > 0 || opts.SeedTimeLimit > 0) && hash != "" {
		_ = c.SetSeedLimits(ctx, hash, opts.SeedRatioLimit, opts.SeedTimeLimit)
	}

	return strings.ToLower(hash), nil
}

func (c *Client) addTorrentFile(ctx context.Context, opts *types.AddOptions) (string, error) {
	if err := c.authenticate(ctx); err != nil {
		return "", err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("torrents", "file.torrent")
	if err != nil {
		return "", err
	}

	if _, err := part.Write(opts.FileContent); err != nil {
		return "", err
	}

	if err := c.writeAddOptions(writer, opts); err != nil {
		return "", err
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	return "", c.submitAdd(ctx, body, writer.FormDataContentType())
}

func (c *Client) writeAddOptions(writer *multipart.Writer, opts *types.AddOptions) error {
	if opts == nil {
		return nil
	}

	category := opts.Category
	if category == "" {
		category = c.config.Category
	}
	if category != "" {
		if err := writer.WriteField("category", category); err != nil {
			return err
		}
	}

	if opts.DownloadDir != "" {
		if err := writer.WriteField("savepath", opts.DownloadDir); err != nil {
			return err
		}
	}

	if opts.Paused {
		if err := writer.WriteField("paused", "true"); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) submitAdd(ctx context.Context, body *bytes.Buffer, contentType string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"api/v2/torrents/add", body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", contentType)
	c.setAuthHeaders(req)

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add torrent: status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	if err := c.authenticate(ctx); err != nil {
		return nil, err
	}

	u := c.baseURL + "api/v2/torrents/info"
	if c.config.Category != "" {
		u += "?category=" + url.QueryEscape(c.config.Category)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, err
	}

	c.setAuthHeaders(req)

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list torrents: status %d", resp.StatusCode)
	}

	var torrents []qbitTorrent
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, err
	}

	items := make([]types.DownloadItem, 0, len(torrents))
	for i := range torrents {
		items = append(items, c.mapTorrentToItem(&torrents[i]))
	}

	return items, nil
}

func (c *Client) Get(ctx context.Context, id string) (*types.DownloadItem, error) {
	items, err := c.List(ctx)
	if err != nil {
		return nil, err
	}

	id = strings.ToLower(id)
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}

	return nil, types.ErrNotFound
}

func (c *Client) Remove(ctx context.Context, id string, deleteFiles bool) error {
	if err := c.authenticate(ctx); err != nil {
		return err
	}

	id = strings.ToLower(id)

	data := url.Values{}
	data.Set("hashes", id)
	data.Set("deleteFiles", strconv.FormatBool(deleteFiles))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"api/v2/torrents/delete", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.setAuthHeaders(req)

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to remove torrent: status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Pause(ctx context.Context, id string) error {
	if err := c.authenticate(ctx); err != nil {
		return err
	}

	id = strings.ToLower(id)

	data := url.Values{}
	data.Set("hashes", id)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"api/v2/torrents/pause", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.setAuthHeaders(req)

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to pause torrent: status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Resume(ctx context.Context, id string) error {
	if err := c.authenticate(ctx); err != nil {
		return err
	}

	id = strings.ToLower(id)

	data := url.Values{}
	data.Set("hashes", id)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"api/v2/torrents/resume", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.setAuthHeaders(req)

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to resume torrent: status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	if err := c.authenticate(ctx); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"api/v2/app/preferences", http.NoBody)
	if err != nil {
		return "", err
	}

	c.setAuthHeaders(req)

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get preferences: status %d", resp.StatusCode)
	}

	var prefs qbitPreferences
	if err := json.NewDecoder(resp.Body).Decode(&prefs); err != nil {
		return "", err
	}

	return prefs.SavePath, nil
}

func (c *Client) SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error {
	if err := c.authenticate(ctx); err != nil {
		return err
	}

	id = strings.ToLower(id)

	data := url.Values{}
	data.Set("hashes", id)

	ratioLimit := -2.0
	if ratio > 0 {
		ratioLimit = ratio
	}
	data.Set("ratioLimit", fmt.Sprintf("%.2f", ratioLimit))

	seedingTimeLimit := -2
	if seedTime > 0 {
		seedingTimeLimit = int(seedTime.Minutes())
	}
	data.Set("seedingTimeLimit", strconv.Itoa(seedingTimeLimit))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"api/v2/torrents/setShareLimits", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.setAuthHeaders(req)

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to set seed limits: status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	item, err := c.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := c.authenticate(ctx); err != nil {
		return nil, err
	}

	id = strings.ToLower(id)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"api/v2/torrents/properties?hash="+url.QueryEscape(id), http.NoBody)
	if err != nil {
		return nil, err
	}

	c.setAuthHeaders(req)

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get torrent properties: status %d", resp.StatusCode)
	}

	var props qbitProperties
	if err := json.NewDecoder(resp.Body).Decode(&props); err != nil {
		return nil, err
	}

	return &types.TorrentInfo{
		DownloadItem: *item,
		InfoHash:     strings.ToLower(id),
		Ratio:        props.ShareRatio,
	}, nil
}

func (c *Client) authenticate(ctx context.Context) error {
	if c.config.APIKey != "" || (c.config.Username == "" && c.config.Password == "") {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cookies) > 0 {
		return nil
	}

	return c.doLogin(ctx)
}

func (c *Client) doLogin(ctx context.Context) error {
	data := url.Values{}
	data.Set("username", c.config.Username)
	data.Set("password", c.config.Password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"api/v2/auth/login", strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if strings.TrimSpace(string(body)) == "Fails." {
		return types.ErrAuthFailed
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return types.ErrAuthFailed
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed: status %d", resp.StatusCode)
	}

	parsedURL, _ := url.Parse(c.baseURL)
	c.cookies = c.httpClient.Jar.Cookies(parsedURL)

	return nil
}

func (c *Client) setAuthHeaders(req *http.Request) {
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
}

func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusForbidden && c.config.APIKey == "" && (c.config.Username != "" || c.config.Password != "") {
		resp.Body.Close()

		c.mu.Lock()
		c.cookies = nil
		parsedURL, _ := url.Parse(c.baseURL)
		c.httpClient.Jar.SetCookies(parsedURL, []*http.Cookie{})
		c.mu.Unlock()

		if err := c.authenticate(ctx); err != nil {
			return nil, err
		}

		clonedReq := req.Clone(ctx)
		c.setAuthHeaders(clonedReq)

		resp, err = c.httpClient.Do(clonedReq)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			return nil, types.ErrAuthFailed
		}
	}

	return resp, nil
}

func (c *Client) mapTorrentToItem(t *qbitTorrent) types.DownloadItem {
	downloadDir := t.SavePath
	if t.ContentPath != "" && t.ContentPath != t.SavePath {
		downloadDir = t.ContentPath
	}

	eta := t.ETA
	if eta == 8640000 || eta < 0 {
		eta = -1
	}

	return types.DownloadItem{
		ID:             strings.ToLower(t.Hash),
		Name:           t.Name,
		Status:         mapStatus(t.State),
		Progress:       t.Progress * 100,
		Size:           t.Size,
		DownloadedSize: t.Completed,
		DownloadSpeed:  t.DLSpeed,
		UploadSpeed:    t.UPSpeed,
		ETA:            eta,
		DownloadDir:    downloadDir,
	}
}

func mapStatus(state string) types.Status {
	switch state {
	case "error", "missingFiles":
		return types.StatusWarning
	case "pausedDL", "stoppedDL":
		return types.StatusPaused
	case "queuedDL", "checkingDL", "checkingUP", "checkingResumeData":
		return types.StatusQueued
	case "pausedUP", "stoppedUP", "uploading", "stalledUP", "queuedUP", "forcedUP":
		return types.StatusSeeding
	case "metaDL", "forcedMetaDL":
		return types.StatusQueued
	case "forcedDL", "moving", "downloading":
		return types.StatusDownloading
	case "stalledDL":
		return types.StatusWarning
	default:
		return types.StatusUnknown
	}
}

func extractHashFromMagnet(magnetURL string) string {
	if !strings.HasPrefix(magnetURL, "magnet:") {
		return ""
	}

	parts := strings.Split(magnetURL, "?")
	if len(parts) < 2 {
		return ""
	}

	params := strings.Split(parts[1], "&")
	for _, param := range params {
		if strings.HasPrefix(param, "xt=urn:btih:") {
			hash := strings.TrimPrefix(param, "xt=urn:btih:")
			return strings.ToLower(hash)
		}
	}

	return ""
}

package utorrent

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // SHA1 is required for BitTorrent info hash computation
	"encoding/hex"
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
	token      string
	tokenMu    sync.RWMutex
}

func NewFromConfig(cfg *types.ClientConfig) *Client {
	jar, _ := cookiejar.New(nil)

	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}

	urlBase := cfg.URLBase
	if urlBase == "" {
		urlBase = "/gui/"
	}
	urlBase = strings.TrimSuffix(urlBase, "/") + "/"

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
	return types.ClientTypeUTorrent
}

func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

func (c *Client) Test(ctx context.Context) error {
	if err := c.fetchToken(ctx); err != nil {
		return err
	}
	_, err := c.GetDownloadDir(ctx)
	if err != nil {
		return err
	}

	params := url.Values{}
	params.Set("action", "getversion")
	body, err := c.doRequest(ctx, params)
	if err != nil {
		return nil //nolint:nilerr // version check is best-effort; connectivity already verified
	}

	var versionResp struct {
		Version struct {
			Build int `json:"build"`
		} `json:"version"`
	}
	if json.Unmarshal(body, &versionResp) == nil && versionResp.Version.Build > 0 {
		if versionResp.Version.Build < 25406 {
			return fmt.Errorf("uTorrent build %d is below minimum required build 25406", versionResp.Version.Build)
		}
	}

	return nil
}

func (c *Client) Connect(ctx context.Context) error {
	return c.fetchToken(ctx)
}

func (c *Client) Add(ctx context.Context, opts *types.AddOptions) (string, error) {
	if opts.FileContent != nil {
		return c.addFile(ctx, opts)
	}
	if opts.URL != "" {
		return c.AddMagnet(ctx, opts.URL, opts)
	}
	return "", fmt.Errorf("either URL or FileContent must be provided")
}

func (c *Client) AddMagnet(ctx context.Context, magnetURL string, opts *types.AddOptions) (string, error) {
	hash := extractMagnetHash(magnetURL)
	if hash == "" {
		return "", fmt.Errorf("invalid magnet URL: no info hash found")
	}

	params := url.Values{}
	params.Set("action", "add-url")
	params.Set("s", magnetURL)

	if _, err := c.doRequest(ctx, params); err != nil {
		return "", err
	}

	if opts != nil && opts.Category != "" {
		labelParams := url.Values{}
		labelParams.Set("action", "setprops")
		labelParams.Set("hash", hash)
		labelParams.Set("s", "label")
		labelParams.Set("v", opts.Category)
		if _, err := c.doRequest(ctx, labelParams); err != nil {
			return "", err
		}
	}

	if opts != nil && (opts.SeedRatioLimit > 0 || opts.SeedTimeLimit > 0) {
		if err := c.SetSeedLimits(ctx, hash, opts.SeedRatioLimit, opts.SeedTimeLimit); err != nil {
			return "", err
		}
	}

	return hash, nil
}

func (c *Client) addFile(ctx context.Context, opts *types.AddOptions) (string, error) { //nolint:gocyclo // multipart upload with retry
	token, err := c.getToken(ctx)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("torrent_file", "file.torrent")
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(opts.FileContent); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	reqURL := c.baseURL + "?token=" + token + "&action=add-file"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, &buf)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.config.Username, c.config.Password)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
		if err := c.fetchToken(ctx); err != nil {
			return "", err
		}
		return c.addFile(ctx, opts)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("add file failed: %s", string(body))
	}

	hash := extractInfoHash(opts.FileContent)
	return hash, nil
}

// extractInfoHash computes the SHA1 info hash from raw .torrent file bytes
// by finding the bencoded "info" dictionary and hashing it.
func extractInfoHash(torrentData []byte) string {
	infoKey := []byte("4:info")
	idx := bytes.Index(torrentData, infoKey)
	if idx < 0 {
		return ""
	}
	infoStart := idx + len(infoKey)
	if infoStart >= len(torrentData) {
		return ""
	}
	infoBytes := torrentData[infoStart:]
	// Find the matching end of the bencoded dict by counting depth
	end := findBencodeEnd(infoBytes)
	if end <= 0 {
		return ""
	}
	h := sha1.Sum(infoBytes[:end]) //nolint:gosec // SHA1 is required for BitTorrent info hash
	return strings.ToUpper(hex.EncodeToString(h[:]))
}

// findBencodeEnd finds the end position of a bencoded value starting at position 0.
func findBencodeEnd(data []byte) int { //nolint:gocognit,gocyclo // recursive bencode parser requires branching
	if len(data) == 0 {
		return -1
	}
	switch data[0] {
	case 'd', 'l': // dict or list
		pos := 1
		for pos < len(data) && data[pos] != 'e' {
			if data[0] == 'd' {
				// skip key (always a string)
				n := findBencodeEnd(data[pos:])
				if n <= 0 {
					return -1
				}
				pos += n
			}
			// skip value
			n := findBencodeEnd(data[pos:])
			if n <= 0 {
				return -1
			}
			pos += n
		}
		if pos >= len(data) {
			return -1
		}
		return pos + 1 // include 'e'
	case 'i': // integer
		end := bytes.IndexByte(data[1:], 'e')
		if end < 0 {
			return -1
		}
		return end + 2
	default: // string: "len:..."
		colon := bytes.IndexByte(data, ':')
		if colon < 0 {
			return -1
		}
		length, err := strconv.Atoi(string(data[:colon]))
		if err != nil {
			return -1
		}
		return colon + 1 + length
	}
}

func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	params := url.Values{}
	params.Set("list", "1")

	body, err := c.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Torrents [][]any `json:"torrents"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	items := make([]types.DownloadItem, 0, len(resp.Torrents))
	for _, t := range resp.Torrents {
		if len(t) < 12 {
			continue
		}
		item := parseTorrentArray(t)
		items = append(items, item)
	}

	return items, nil
}

func (c *Client) Get(ctx context.Context, id string) (*types.DownloadItem, error) {
	items, err := c.List(ctx)
	if err != nil {
		return nil, err
	}

	for i := range items {
		if strings.EqualFold(items[i].ID, id) {
			return &items[i], nil
		}
	}

	return nil, types.ErrNotFound
}

func (c *Client) Remove(ctx context.Context, id string, deleteFiles bool) error {
	action := "remove"
	if deleteFiles {
		action = "removedata"
	}

	params := url.Values{}
	params.Set("action", action)
	params.Set("hash", strings.ToUpper(id))

	_, err := c.doRequest(ctx, params)
	return err
}

func (c *Client) Pause(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("action", "pause")
	params.Set("hash", strings.ToUpper(id))

	_, err := c.doRequest(ctx, params)
	return err
}

func (c *Client) Resume(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("action", "start")
	params.Set("hash", strings.ToUpper(id))

	_, err := c.doRequest(ctx, params)
	return err
}

func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	params := url.Values{}
	params.Set("action", "getsettings")

	body, err := c.doRequest(ctx, params)
	if err != nil {
		return "", err
	}

	var resp struct {
		Settings [][]any `json:"settings"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}

	for _, setting := range resp.Settings {
		if len(setting) < 3 {
			continue
		}
		name, ok := setting[0].(string)
		if !ok || name != "dir_active_download" {
			continue
		}
		dir, ok := setting[2].(string)
		if ok {
			return dir, nil
		}
	}

	return "", fmt.Errorf("download directory not found in settings")
}

func (c *Client) SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error {
	params := url.Values{}
	params.Set("action", "setprops")
	params.Set("hash", strings.ToUpper(id))
	params.Add("s", "seed_override")
	params.Add("v", "1")
	if ratio > 0 {
		params.Add("s", "seed_ratio")
		params.Add("v", strconv.Itoa(int(ratio*1000)))
	}
	if seedTime > 0 {
		params.Add("s", "seed_time")
		params.Add("v", strconv.Itoa(int(seedTime.Seconds())))
	}

	_, err := c.doRequest(ctx, params)
	return err
}

func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	item, err := c.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return &types.TorrentInfo{
		DownloadItem: *item,
		InfoHash:     strings.ToUpper(id),
	}, nil
}

func (c *Client) fetchToken(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"token.html", http.NoBody)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.config.Username, c.config.Password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return types.ErrAuthFailed
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token fetch failed: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	token := parseToken(string(body))
	if token == "" {
		return fmt.Errorf("token not found in response")
	}

	c.tokenMu.Lock()
	c.token = token
	c.tokenMu.Unlock()

	return nil
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	c.tokenMu.RLock()
	token := c.token
	c.tokenMu.RUnlock()

	if token == "" {
		if err := c.fetchToken(ctx); err != nil {
			return "", err
		}
		c.tokenMu.RLock()
		token = c.token
		c.tokenMu.RUnlock()
	}

	return token, nil
}

func (c *Client) doRequest(ctx context.Context, params url.Values) ([]byte, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	params.Set("token", token)
	reqURL := c.baseURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.config.Username, c.config.Password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest {
		if err := c.fetchToken(ctx); err != nil {
			return nil, err
		}
		return c.doRequest(ctx, params)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed: %s", string(body))
	}

	return io.ReadAll(resp.Body)
}

func parseToken(html string) string {
	start := strings.Index(html, ">")
	if start == -1 {
		return ""
	}
	end := strings.Index(html[start+1:], "</")
	if end == -1 {
		return ""
	}
	return html[start+1 : start+1+end]
}

func extractMagnetHash(magnetURL string) string {
	u, err := url.Parse(magnetURL)
	if err != nil {
		return ""
	}

	xt := u.Query().Get("xt")
	if !strings.HasPrefix(xt, "urn:btih:") {
		return ""
	}

	hash := strings.TrimPrefix(xt, "urn:btih:")
	return strings.ToUpper(hash)
}

func parseTorrentArray(t []any) types.DownloadItem {
	getString := func(idx int) string {
		if idx >= len(t) {
			return ""
		}
		s, _ := t[idx].(string)
		return s
	}

	getInt64 := func(idx int) int64 {
		if idx >= len(t) {
			return 0
		}
		switch v := t[idx].(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case int:
			return int64(v)
		}
		return 0
	}

	hash := strings.ToUpper(getString(0))
	statusFlags := int(getInt64(1))
	name := getString(2)
	size := getInt64(3)
	progress := float64(getInt64(4)) / 10.0
	downloaded := getInt64(5)
	uploadSpeed := getInt64(8)
	downloadSpeed := getInt64(9)
	eta := getInt64(10)
	downloadDir := getString(26)

	status := mapStatus(statusFlags, int(progress))

	return types.DownloadItem{
		ID:             hash,
		Name:           name,
		Status:         status,
		Progress:       progress,
		Size:           size,
		DownloadedSize: downloaded,
		DownloadSpeed:  downloadSpeed,
		UploadSpeed:    uploadSpeed,
		ETA:            eta,
		DownloadDir:    downloadDir,
	}
}

func mapStatus(flags, progress int) types.Status {
	const (
		flagStarted = 1
		flagChecked = 8
		flagError   = 16
		flagPaused  = 32
		flagLoaded  = 128
	)

	if flags&flagError != 0 {
		return types.StatusWarning
	}

	if flags&flagLoaded != 0 && flags&flagChecked != 0 && progress == 100 {
		return types.StatusSeeding
	}

	if flags&flagPaused != 0 {
		return types.StatusPaused
	}

	if flags&flagStarted != 0 {
		return types.StatusDownloading
	}

	return types.StatusQueued
}

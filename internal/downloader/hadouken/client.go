package hadouken

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

var _ types.TorrentClient = (*Client)(nil)

type Client struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

func NewFromConfig(cfg *types.ClientConfig) *Client {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s:%d%s/api", scheme, cfg.Host, cfg.Port, cfg.URLBase)
	return &Client{
		baseURL:  baseURL,
		username: cfg.Username,
		password: cfg.Password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Type() types.ClientType {
	return types.ClientTypeHadouken
}

func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

func (c *Client) Connect(ctx context.Context) error {
	return c.Test(ctx)
}

func (c *Client) Test(ctx context.Context) error {
	var result struct {
		Versions struct {
			Hadouken string `json:"hadouken"`
		} `json:"versions"`
	}
	if err := c.call(ctx, "core.getSystemInfo", []any{}, &result); err != nil {
		return err
	}
	if result.Versions.Hadouken == "" {
		return errors.New("invalid response from hadouken")
	}

	meetsMin, err := types.CompareVersions(result.Versions.Hadouken, "5.1.0")
	if err != nil {
		return fmt.Errorf("failed to parse Hadouken version %q: %w", result.Versions.Hadouken, err)
	}
	if !meetsMin {
		return fmt.Errorf("hadouken version %s is below minimum required version 5.1", result.Versions.Hadouken)
	}

	return nil
}

func (c *Client) Add(ctx context.Context, opts *types.AddOptions) (string, error) {
	if opts.FileContent != nil {
		encoded := base64.StdEncoding.EncodeToString(opts.FileContent)
		params := []any{"file", encoded}
		if opts.Category != "" {
			params = append(params, map[string]string{"label": opts.Category})
		} else {
			params = append(params, map[string]string{})
		}
		var infohash string
		if err := c.call(ctx, "webui.addTorrent", params, &infohash); err != nil {
			return "", err
		}
		return infohash, nil
	}

	if opts.URL != "" {
		params := []any{"url", opts.URL}
		if opts.Category != "" {
			params = append(params, map[string]string{"label": opts.Category})
		} else {
			params = append(params, map[string]string{})
		}
		var infohash string
		if err := c.call(ctx, "webui.addTorrent", params, &infohash); err != nil {
			return "", err
		}
		return infohash, nil
	}

	return "", errors.New("either URL or FileContent must be provided")
}

func (c *Client) AddMagnet(ctx context.Context, magnetURL string, opts *types.AddOptions) (string, error) {
	if opts == nil {
		opts = &types.AddOptions{}
	}
	opts.URL = magnetURL
	return c.Add(ctx, opts)
}

func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	var result struct {
		Torrents [][]any `json:"torrents"`
	}
	if err := c.call(ctx, "webui.list", []any{}, &result); err != nil {
		return nil, err
	}

	items := make([]types.DownloadItem, 0, len(result.Torrents))
	for _, t := range result.Torrents {
		item, err := c.parseTorrentArray(t)
		if err != nil {
			continue
		}
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
		if items[i].ID == id {
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
	var result any
	return c.call(ctx, "webui.perform", []any{action, []string{id}}, &result)
}

func (c *Client) Pause(ctx context.Context, id string) error {
	var result any
	return c.call(ctx, "webui.perform", []any{"pause", []string{id}}, &result)
}

func (c *Client) Resume(ctx context.Context, id string) error {
	var result any
	return c.call(ctx, "webui.perform", []any{"start", []string{id}}, &result)
}

func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	var result [][]any
	if err := c.call(ctx, "webui.getSettings", []any{}, &result); err != nil {
		return "", err
	}

	for _, setting := range result {
		if len(setting) >= 3 {
			if key, ok := setting[0].(string); ok && key == "bittorrent.defaultSavePath" {
				if path, ok := setting[2].(string); ok {
					return path, nil
				}
			}
		}
	}
	return "", errors.New("default save path not found in settings")
}

func (c *Client) SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error {
	return nil
}

func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	var result struct {
		Torrents [][]any `json:"torrents"`
	}
	if err := c.call(ctx, "webui.list", []any{}, &result); err != nil {
		return nil, err
	}

	for _, t := range result.Torrents {
		if len(t) < 27 {
			continue
		}
		infohash, _ := t[0].(string)
		if infohash != id {
			continue
		}

		item, err := c.parseTorrentArray(t)
		if err != nil {
			return nil, err
		}

		uploadedFloat, _ := t[6].(float64)
		uploadedBytes := int64(uploadedFloat)

		ratio := 0.0
		if item.Size > 0 {
			ratio = float64(uploadedBytes) / float64(item.Size)
		}
		return &types.TorrentInfo{
			DownloadItem: item,
			InfoHash:     id,
			Seeders:      0,
			Leechers:     0,
			Ratio:        ratio,
			IsPrivate:    false,
		}, nil
	}

	return nil, types.ErrNotFound
}

func (c *Client) parseTorrentArray(t []any) (types.DownloadItem, error) {
	if len(t) < 27 {
		return types.DownloadItem{}, errors.New("invalid torrent array length")
	}

	infohash, _ := t[0].(string)
	stateFloat, _ := t[1].(float64)
	state := int(stateFloat)
	name, _ := t[2].(string)
	totalSizeFloat, _ := t[3].(float64)
	totalSize := int64(totalSizeFloat)
	progressFloat, _ := t[4].(float64)
	downloadedFloat, _ := t[5].(float64)
	downloaded := int64(downloadedFloat)
	downloadRateFloat, _ := t[9].(float64)
	downloadRate := int64(downloadRateFloat)
	errorMsg, _ := t[21].(string)
	savePath, _ := t[26].(string)

	progress := progressFloat / 10.0
	if progress > 100 {
		progress = 100
	}

	status := c.parseStatus(state, int(progressFloat), errorMsg)

	return types.DownloadItem{
		ID:             infohash,
		Name:           name,
		Status:         status,
		Progress:       progress,
		Size:           totalSize,
		DownloadedSize: downloaded,
		DownloadSpeed:  downloadRate,
		UploadSpeed:    0,
		ETA:            -1,
		DownloadDir:    savePath,
		Error:          errorMsg,
	}, nil
}

func (c *Client) parseStatus(state, progress int, errorMsg string) types.Status {
	if errorMsg != "" {
		return types.StatusWarning
	}
	if progress >= 1000 && (state&2) == 0 {
		return types.StatusSeeding
	}
	if (state & 64) != 0 {
		return types.StatusQueued
	}
	if (state & 32) != 0 {
		return types.StatusPaused
	}
	if (state & 1) != 0 {
		return types.StatusDownloading
	}
	return types.StatusQueued
}

func (c *Client) call(ctx context.Context, method string, params []any, result any) error {
	reqBody := map[string]any{
		"method": method,
		"params": params,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return types.ErrAuthFailed
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  any             `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return err
	}

	if rpcResp.Error != nil {
		return fmt.Errorf("rpc error: %v", rpcResp.Error)
	}

	if result != nil {
		return json.Unmarshal(rpcResp.Result, result)
	}

	return nil
}

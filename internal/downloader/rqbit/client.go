package rqbit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

var _ types.TorrentClient = (*Client)(nil)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewFromConfig(cfg *types.ClientConfig) *Client {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s:%d", scheme, cfg.Host, cfg.Port)
	if cfg.URLBase != "" {
		baseURL += cfg.URLBase
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Type() types.ClientType {
	return types.ClientTypeRQBit
}

func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

func (c *Client) Test(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/", http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var serverInfo struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&serverInfo); err == nil && serverInfo.Version != "" {
		meetsMin, verErr := types.CompareVersions(serverInfo.Version, "8.0.0")
		if verErr != nil {
			return fmt.Errorf("failed to parse rqbit version %q: %w", serverInfo.Version, verErr)
		}
		if !meetsMin {
			return fmt.Errorf("rqbit version %s is below minimum required version 8.0.0", serverInfo.Version)
		}
	}

	return nil
}

func (c *Client) Connect(ctx context.Context) error {
	return c.Test(ctx)
}

func (c *Client) Add(ctx context.Context, opts *types.AddOptions) (string, error) {
	if opts == nil {
		return "", errors.New("options required")
	}

	if opts.URL != "" {
		return c.AddMagnet(ctx, opts.URL, opts)
	}

	if len(opts.FileContent) > 0 {
		return c.addTorrentFile(ctx, opts.FileContent)
	}

	return "", errors.New("either URL or FileContent required")
}

func (c *Client) AddMagnet(ctx context.Context, magnetURL string, opts *types.AddOptions) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/torrents?overwrite=true", bytes.NewBufferString(magnetURL))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result addResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Details.InfoHash, nil
}

func (c *Client) addTorrentFile(ctx context.Context, fileContent []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/torrents?overwrite=true", bytes.NewBuffer(fileContent))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-bittorrent")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result addResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Details.InfoHash, nil
}

func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/torrents?with_stats=true", http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result listResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	items := make([]types.DownloadItem, 0, len(result.Torrents))
	for i := range result.Torrents {
		items = append(items, c.torrentToDownloadItem(&result.Torrents[i]))
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
	endpoint := "/forget"
	if deleteFiles {
		endpoint = "/delete"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/torrents/"+id+endpoint, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Pause(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/torrents/"+id+"/pause", http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Resume(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/torrents/"+id+"/start", http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	items, err := c.List(ctx)
	if err != nil {
		return "", err
	}

	if len(items) > 0 {
		return items[0].DownloadDir, nil
	}

	return "", nil
}

func (c *Client) SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error {
	return nil
}

func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/torrents?with_stats=true", http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result listResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	for i := range result.Torrents {
		t := &result.Torrents[i]
		if t.InfoHash != id {
			continue
		}
		item := c.torrentToDownloadItem(t)
		ratio := 0.0
		if t.Stats.TotalBytes > 0 {
			ratio = float64(t.Stats.UploadedBytes) / float64(t.Stats.TotalBytes)
		}

		seeders := 0
		leechers := 0
		if t.Stats.Live != nil && t.Stats.Live.Snapshot != nil {
			seeders = t.Stats.Live.Snapshot.PeerStats.Live
			leechers = t.Stats.Live.Snapshot.PeerStats.Seen - seeders
		}

		return &types.TorrentInfo{
			DownloadItem: item,
			InfoHash:     t.InfoHash,
			Seeders:      seeders,
			Leechers:     leechers,
			Ratio:        ratio,
			IsPrivate:    false,
		}, nil
	}

	return nil, types.ErrNotFound
}

func (c *Client) torrentToDownloadItem(t *torrent) types.DownloadItem {
	status := c.mapStatus(t.Stats.State, t.Stats.Finished)

	progress := 0.0
	if t.Stats.TotalBytes > 0 {
		progress = float64(t.Stats.ProgressBytes) / float64(t.Stats.TotalBytes) * 100
	}

	downloadSpeed := int64(0)
	uploadSpeed := int64(0)
	eta := int64(-1)

	if t.Stats.Live != nil {
		downloadSpeed = int64(t.Stats.Live.DownloadSpeed.Mbps * 1048576)
		uploadSpeed = int64(t.Stats.Live.UploadSpeed.Mbps * 1048576)

		if t.Stats.Live.TimeRemaining != nil && t.Stats.Live.TimeRemaining.Duration != nil {
			eta = t.Stats.Live.TimeRemaining.Duration.Secs
		}
	}

	item := types.DownloadItem{
		ID:             t.InfoHash,
		Name:           t.Name,
		Status:         status,
		Progress:       progress,
		Size:           t.Stats.TotalBytes,
		DownloadedSize: t.Stats.ProgressBytes,
		DownloadSpeed:  downloadSpeed,
		UploadSpeed:    uploadSpeed,
		ETA:            eta,
		DownloadDir:    t.OutputFolder,
	}

	if t.Stats.Error != nil {
		item.Error = *t.Stats.Error
	}

	return item
}

func (c *Client) mapStatus(state int, finished bool) types.Status {
	if finished {
		if state == 1 {
			return types.StatusCompleted
		}
		return types.StatusSeeding
	}

	switch state {
	case 0:
		return types.StatusDownloading
	case 1:
		return types.StatusPaused
	case 2:
		return types.StatusDownloading
	case 3:
		return types.StatusWarning
	case 4:
		return types.StatusWarning
	default:
		return types.StatusUnknown
	}
}

type addResponse struct {
	ID           int            `json:"id"`
	Details      torrentDetails `json:"details"`
	OutputFolder string         `json:"output_folder"`
}

type torrentDetails struct {
	InfoHash string `json:"info_hash"`
	Name     string `json:"name"`
}

type listResponse struct {
	Torrents []torrent `json:"torrents"`
}

type torrent struct {
	ID           int    `json:"id"`
	InfoHash     string `json:"info_hash"`
	Name         string `json:"name"`
	OutputFolder string `json:"output_folder"`
	Stats        stats  `json:"stats"`
}

type stats struct {
	State         int     `json:"state"`
	Error         *string `json:"error"`
	ProgressBytes int64   `json:"progress_bytes"`
	UploadedBytes int64   `json:"uploaded_bytes"`
	TotalBytes    int64   `json:"total_bytes"`
	Finished      bool    `json:"finished"`
	Live          *live   `json:"live"`
}

type live struct {
	DownloadSpeed speed          `json:"download_speed"`
	UploadSpeed   speed          `json:"upload_speed"`
	TimeRemaining *timeRemaining `json:"time_remaining"`
	Snapshot      *snapshot      `json:"snapshot"`
}

type speed struct {
	Mbps float64 `json:"mbps"`
}

type timeRemaining struct {
	Duration *duration `json:"duration"`
}

type duration struct {
	Secs  int64 `json:"secs"`
	Nanos int64 `json:"nanos"`
}

type snapshot struct {
	DownloadedAndCheckedBytes int64     `json:"downloaded_and_checked_bytes"`
	UploadedBytes             int64     `json:"uploaded_bytes"`
	PeerStats                 peerStats `json:"peer_stats"`
}

type peerStats struct {
	Live int `json:"live"`
	Seen int `json:"seen"`
}

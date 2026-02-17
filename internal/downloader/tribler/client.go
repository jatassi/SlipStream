package tribler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

var _ types.TorrentClient = (*Client)(nil)

type Client struct {
	config     *types.ClientConfig
	httpClient *http.Client
	baseURL    string
}

func NewFromConfig(cfg *types.ClientConfig) *Client {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: fmt.Sprintf("%s://%s:%d/", scheme, cfg.Host, cfg.Port),
	}
}

func (c *Client) Type() types.ClientType {
	return types.ClientTypeTribler
}

func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

func (c *Client) Test(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"api/settings", http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.httpClient.Do(req)
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

	return nil
}

func (c *Client) Connect(ctx context.Context) error {
	return c.Test(ctx)
}

type settingsResponse struct {
	Settings struct {
		Libtorrent struct {
			DownloadDefaults struct {
				Saveas string `json:"saveas"`
			} `json:"download_defaults"`
		} `json:"libtorrent"`
	} `json:"settings"`
}

func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"api/settings", http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", types.ErrAuthFailed
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var settings settingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return "", err
	}

	return settings.Settings.Libtorrent.DownloadDefaults.Saveas, nil
}

type downloadsResponse struct {
	Downloads []downloadEntry `json:"downloads"`
}

type downloadEntry struct {
	Name         string  `json:"name"`
	Progress     float64 `json:"progress"`
	Infohash     string  `json:"infohash"`
	ETA          float64 `json:"eta"`
	NumSeeds     int     `json:"num_seeds"`
	NumPeers     int     `json:"num_peers"`
	AllTimeRatio float64 `json:"all_time_ratio"`
	TimeAdded    int64   `json:"time_added"`
	Status       string  `json:"status"`
	Error        string  `json:"error"`
	Size         int64   `json:"size"`
	Destination  string  `json:"destination"`
	SpeedDown    float64 `json:"speed_down"`
	SpeedUp      float64 `json:"speed_up"`
}

func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"api/downloads", http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, types.ErrAuthFailed
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var downloads downloadsResponse
	if err := json.NewDecoder(resp.Body).Decode(&downloads); err != nil {
		return nil, err
	}

	items := make([]types.DownloadItem, 0, len(downloads.Downloads))
	for i := range downloads.Downloads {
		d := &downloads.Downloads[i]
		if d.Size == 0 {
			continue
		}

		item := types.DownloadItem{
			ID:             d.Infohash,
			Name:           d.Name,
			Status:         c.mapStatus(d.Status, d.Progress, d.Error),
			Progress:       d.Progress * 100,
			Size:           d.Size,
			DownloadedSize: int64(float64(d.Size) * d.Progress),
			DownloadSpeed:  int64(d.SpeedDown),
			UploadSpeed:    int64(d.SpeedUp),
			ETA:            c.mapETA(d.ETA),
			DownloadDir:    d.Destination,
			AddedAt:        time.Unix(d.TimeAdded, 0),
			Error:          d.Error,
		}

		if d.Progress >= 1.0 {
			item.CompletedAt = time.Now()
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

type addRequest struct {
	URI         string `json:"uri"`
	AnonHops    int    `json:"anon_hops"`
	SafeSeeding bool   `json:"safe_seeding"`
	Destination string `json:"destination,omitempty"`
}

type addResponse struct {
	Infohash string `json:"infohash"`
	Started  bool   `json:"started"`
}

func (c *Client) Add(ctx context.Context, opts *types.AddOptions) (string, error) {
	if len(opts.FileContent) > 0 {
		return "", errors.New("torrent file upload is not supported by Tribler")
	}

	if opts.URL == "" {
		return "", errors.New("URL is required")
	}

	return c.AddMagnet(ctx, opts.URL, opts)
}

func (c *Client) AddMagnet(ctx context.Context, magnetURL string, opts *types.AddOptions) (string, error) {
	if opts == nil {
		opts = &types.AddOptions{}
	}

	reqBody := addRequest{
		URI:         magnetURL,
		AnonHops:    1,
		SafeSeeding: true,
	}

	if opts.DownloadDir != "" {
		reqBody.Destination = opts.DownloadDir
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+"api/downloads", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", types.ErrAuthFailed
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var addResp addResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		return "", err
	}

	return addResp.Infohash, nil
}

type removeRequest struct {
	RemoveData bool `json:"remove_data"`
}

func (c *Client) Remove(ctx context.Context, id string, deleteFiles bool) error {
	reqBody := removeRequest{
		RemoveData: deleteFiles,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	infohash := strings.ToLower(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+"api/downloads/"+url.PathEscape(infohash), strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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

	return nil
}

type stateRequest struct {
	State string `json:"state"`
}

func (c *Client) Pause(ctx context.Context, id string) error {
	return c.updateState(ctx, id, "stop")
}

func (c *Client) Resume(ctx context.Context, id string) error {
	return c.updateState(ctx, id, "resume")
}

func (c *Client) updateState(ctx context.Context, id, state string) error {
	reqBody := stateRequest{
		State: state,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	infohash := strings.ToLower(id)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.baseURL+"api/downloads/"+url.PathEscape(infohash), strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
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

	return nil
}

func (c *Client) SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error {
	return types.ErrNotImplemented
}

func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"api/downloads", http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, types.ErrAuthFailed
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var downloads downloadsResponse
	if err := json.NewDecoder(resp.Body).Decode(&downloads); err != nil {
		return nil, err
	}

	for i := range downloads.Downloads {
		d := &downloads.Downloads[i]
		if d.Infohash != id {
			continue
		}

		return &types.TorrentInfo{
			DownloadItem: types.DownloadItem{
				ID:             d.Infohash,
				Name:           d.Name,
				Status:         c.mapStatus(d.Status, d.Progress, d.Error),
				Progress:       d.Progress * 100,
				Size:           d.Size,
				DownloadedSize: int64(float64(d.Size) * d.Progress),
				DownloadSpeed:  int64(d.SpeedDown),
				UploadSpeed:    int64(d.SpeedUp),
				ETA:            c.mapETA(d.ETA),
				DownloadDir:    d.Destination,
				AddedAt:        time.Unix(d.TimeAdded, 0),
				Error:          d.Error,
			},
			InfoHash:  d.Infohash,
			Seeders:   d.NumSeeds,
			Leechers:  d.NumPeers,
			Ratio:     d.AllTimeRatio,
			IsPrivate: false,
		}, nil
	}

	return nil, types.ErrNotFound
}

func (c *Client) mapStatus(status string, progress float64, errMsg string) types.Status {
	if errMsg != "" {
		return types.StatusWarning
	}

	switch status {
	case "DOWNLOADING":
		return types.StatusDownloading
	case "SEEDING":
		return types.StatusSeeding
	case "STOPPED":
		if progress >= 1.0 {
			return types.StatusCompleted
		}
		return types.StatusPaused
	case "WAITING4HASHCHECK", "HASHCHECKING", "CIRCUITS", "EXIT_NODES":
		return types.StatusDownloading
	case "METADATA", "ALLOCATING_DISKSPACE":
		return types.StatusQueued
	case "STOPPED_ON_ERROR":
		return types.StatusError
	default:
		return types.StatusDownloading
	}
}

func (c *Client) mapETA(eta float64) int64 {
	if eta <= 0 {
		return -1
	}

	const maxETA = 31536000
	if eta > maxETA {
		return maxETA
	}

	return int64(eta)
}

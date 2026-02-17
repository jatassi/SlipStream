package downloadstation

import (
	"bytes"
	"context"
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
	config     *types.ClientConfig
	httpClient *http.Client
	baseURL    string
	sid        string
}

type apiResponse struct {
	Success bool             `json:"success"`
	Data    *json.RawMessage `json:"data,omitempty"`
	Error   *apiError        `json:"error,omitempty"`
}

type apiError struct {
	Code int `json:"code"`
}

type authData struct {
	SID string `json:"sid"`
}

type configData struct {
	DefaultDestination string `json:"default_destination"`
}

type listData struct {
	Tasks []taskData `json:"tasks"`
}

type taskData struct {
	ID         string              `json:"id"`
	Title      string              `json:"title"`
	Size       int64               `json:"size"`
	Status     string              `json:"status"`
	Additional *taskAdditionalData `json:"additional,omitempty"`
}

type taskAdditionalData struct {
	Detail   *taskDetailData   `json:"detail,omitempty"`
	Transfer *taskTransferData `json:"transfer,omitempty"`
}

type taskDetailData struct {
	Destination string `json:"destination"`
	URI         string `json:"uri"`
}

type taskTransferData struct {
	SizeDownloaded string `json:"size_downloaded"`
	SpeedDownload  string `json:"speed_download"`
	SizeUploaded   string `json:"size_uploaded"`
	SpeedUpload    string `json:"speed_upload"`
}

func NewFromConfig(cfg *types.ClientConfig) *Client {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s:%d", scheme, cfg.Host, cfg.Port)

	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (c *Client) Type() types.ClientType {
	return types.ClientTypeDownloadStation
}

func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

func (c *Client) Test(ctx context.Context) error {
	params := url.Values{}
	params.Set("api", "SYNO.DownloadStation.Info")
	params.Set("version", "1")
	params.Set("method", "getConfig")

	if err := c.doAPICall(ctx, "DownloadStation/info.cgi", params, nil); err != nil {
		return err
	}

	infoParams := url.Values{}
	infoParams.Set("api", "SYNO.API.Info")
	infoParams.Set("version", "1")
	infoParams.Set("method", "query")
	infoParams.Set("query", "SYNO.DownloadStation.Task")

	var apiInfo map[string]struct {
		MaxVersion int `json:"maxVersion"`
	}
	if err := c.doAPICall(ctx, "query.cgi", infoParams, &apiInfo); err == nil {
		if taskInfo, ok := apiInfo["SYNO.DownloadStation.Task"]; ok {
			if taskInfo.MaxVersion < 2 {
				return fmt.Errorf("download station task API version %d is below minimum required version 2", taskInfo.MaxVersion)
			}
		}
	}

	return nil
}

func (c *Client) Connect(ctx context.Context) error {
	return c.authenticate(ctx)
}

func (c *Client) Add(ctx context.Context, opts *types.AddOptions) (string, error) {
	if opts.FileContent != nil {
		return c.addFile(ctx, opts)
	}
	if opts.URL != "" {
		return c.addURL(ctx, opts)
	}
	return "", errors.New("either URL or FileContent must be provided")
}

func (c *Client) addURL(ctx context.Context, opts *types.AddOptions) (string, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("api", "SYNO.DownloadStation.Task")
	params.Set("version", "3")
	params.Set("method", "create")
	params.Set("uri", opts.URL)
	if opts.DownloadDir != "" {
		params.Set("destination", opts.DownloadDir)
	}
	params.Set("_sid", c.sid)

	u := fmt.Sprintf("%s/webapi/DownloadStation/task.cgi?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return "", err
	}

	if !apiResp.Success {
		if apiResp.Error != nil && c.isAuthError(apiResp.Error.Code) {
			if err := c.authenticate(ctx); err != nil {
				return "", err
			}
			return c.addURL(ctx, opts)
		}
		return "", fmt.Errorf("API error: code %d", apiResp.Error.Code)
	}

	return c.extractIDFromURI(opts.URL), nil
}

func (c *Client) addFile(ctx context.Context, opts *types.AddOptions) (string, error) { //nolint:gocyclo // multipart upload with auth retry and ID lookup
	if err := c.ensureAuth(ctx); err != nil {
		return "", err
	}

	body, contentType, err := c.createMultipartBody(opts)
	if err != nil {
		return "", err
	}

	u := fmt.Sprintf("%s/webapi/DownloadStation/task.cgi", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", contentType)

	var apiResp apiResponse
	if err := c.doRequest(req, &apiResp); err != nil {
		return "", err
	}

	if !apiResp.Success {
		if apiResp.Error != nil && c.isAuthError(apiResp.Error.Code) {
			if err := c.authenticate(ctx); err != nil {
				return "", err
			}
			return c.addFile(ctx, opts)
		}
		return "", fmt.Errorf("API error: code %d", apiResp.Error.Code)
	}

	// DS does not return a task ID from file uploads; query the task list
	// and return the most recently created task's ID as a best-effort fallback
	taskID, findErr := c.findLatestTaskID(ctx)
	if findErr != nil || taskID == "" {
		return "file-upload", nil //nolint:nilerr // best-effort ID lookup; upload itself succeeded
	}
	return taskID, nil
}

func (c *Client) findLatestTaskID(ctx context.Context) (string, error) {
	params := url.Values{}
	params.Set("api", "SYNO.DownloadStation.Task")
	params.Set("version", "1")
	params.Set("method", "list")

	var data listData
	if err := c.doAPICall(ctx, "DownloadStation/task.cgi", params, &data); err != nil {
		return "", err
	}

	if len(data.Tasks) == 0 {
		return "", nil
	}

	// Return the last task in the list (most recently added)
	return data.Tasks[len(data.Tasks)-1].ID, nil
}

func (c *Client) createMultipartBody(opts *types.AddOptions) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	fields := map[string]string{
		"api":     "SYNO.DownloadStation.Task",
		"version": "2",
		"method":  "create",
		"_sid":    c.sid,
	}
	if opts.DownloadDir != "" {
		fields["destination"] = opts.DownloadDir
	}

	for key, val := range fields {
		if err := writer.WriteField(key, val); err != nil {
			return nil, "", err
		}
	}

	part, err := writer.CreateFormFile("file", "file.torrent")
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(opts.FileContent); err != nil {
		return nil, "", err
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return &buf, writer.FormDataContentType(), nil
}

func (c *Client) doRequest(req *http.Request, result interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) doAPICall(ctx context.Context, endpoint string, params url.Values, result interface{}) error {
	if err := c.ensureAuth(ctx); err != nil {
		return err
	}

	params.Set("_sid", c.sid)
	u := fmt.Sprintf("%s/webapi/%s?%s", c.baseURL, endpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return err
	}

	var apiResp apiResponse
	if err := c.doRequest(req, &apiResp); err != nil {
		return err
	}

	if !apiResp.Success {
		if apiResp.Error != nil && c.isAuthError(apiResp.Error.Code) {
			if err := c.authenticate(ctx); err != nil {
				return err
			}
			return c.doAPICall(ctx, endpoint, params, result)
		}
		return fmt.Errorf("API error: code %d", apiResp.Error.Code)
	}

	if apiResp.Data != nil && result != nil {
		return json.Unmarshal(*apiResp.Data, result)
	}

	return nil
}

func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	params := url.Values{}
	params.Set("api", "SYNO.DownloadStation.Task")
	params.Set("version", "1")
	params.Set("method", "list")
	params.Set("additional", "detail,transfer")

	var data listData
	if err := c.doAPICall(ctx, "DownloadStation/task.cgi", params, &data); err != nil {
		return nil, err
	}

	items := make([]types.DownloadItem, 0, len(data.Tasks))
	for _, task := range data.Tasks {
		items = append(items, c.taskToDownloadItem(task))
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
	if err := c.ensureAuth(ctx); err != nil {
		return err
	}

	params := url.Values{}
	params.Set("api", "SYNO.DownloadStation.Task")
	params.Set("version", "1")
	params.Set("method", "delete")
	params.Set("id", id)
	params.Set("force_complete", "false")
	params.Set("_sid", c.sid)

	u := fmt.Sprintf("%s/webapi/DownloadStation/task.cgi?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return err
	}

	if !apiResp.Success {
		if apiResp.Error != nil && c.isAuthError(apiResp.Error.Code) {
			if err := c.authenticate(ctx); err != nil {
				return err
			}
			return c.Remove(ctx, id, deleteFiles)
		}
		return fmt.Errorf("API error: code %d", apiResp.Error.Code)
	}

	return nil
}

func (c *Client) Pause(ctx context.Context, id string) error {
	if err := c.ensureAuth(ctx); err != nil {
		return err
	}

	params := url.Values{}
	params.Set("api", "SYNO.DownloadStation.Task")
	params.Set("version", "1")
	params.Set("method", "pause")
	params.Set("id", id)
	params.Set("_sid", c.sid)

	u := fmt.Sprintf("%s/webapi/DownloadStation/task.cgi?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return err
	}

	if !apiResp.Success {
		if apiResp.Error != nil && c.isAuthError(apiResp.Error.Code) {
			if err := c.authenticate(ctx); err != nil {
				return err
			}
			return c.Pause(ctx, id)
		}
		return fmt.Errorf("API error: code %d", apiResp.Error.Code)
	}

	return nil
}

func (c *Client) Resume(ctx context.Context, id string) error {
	if err := c.ensureAuth(ctx); err != nil {
		return err
	}

	params := url.Values{}
	params.Set("api", "SYNO.DownloadStation.Task")
	params.Set("version", "1")
	params.Set("method", "resume")
	params.Set("id", id)
	params.Set("_sid", c.sid)

	u := fmt.Sprintf("%s/webapi/DownloadStation/task.cgi?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var apiResp apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return err
	}

	if !apiResp.Success {
		if apiResp.Error != nil && c.isAuthError(apiResp.Error.Code) {
			if err := c.authenticate(ctx); err != nil {
				return err
			}
			return c.Resume(ctx, id)
		}
		return fmt.Errorf("API error: code %d", apiResp.Error.Code)
	}

	return nil
}

func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	params := url.Values{}
	params.Set("api", "SYNO.DownloadStation.Info")
	params.Set("version", "1")
	params.Set("method", "getConfig")

	var data configData
	if err := c.doAPICall(ctx, "DownloadStation/info.cgi", params, &data); err != nil {
		return "", err
	}

	return data.DefaultDestination, nil
}

func (c *Client) AddMagnet(ctx context.Context, magnetURL string, opts *types.AddOptions) (string, error) {
	if opts == nil {
		opts = &types.AddOptions{}
	}
	opts.URL = magnetURL
	return c.Add(ctx, opts)
}

func (c *Client) SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error {
	return nil
}

func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	item, err := c.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	info := &types.TorrentInfo{
		DownloadItem: *item,
		InfoHash:     "",
		Seeders:      0,
		Leechers:     0,
		Ratio:        0,
		IsPrivate:    false,
	}

	return info, nil
}

func (c *Client) ensureAuth(ctx context.Context) error {
	if c.sid != "" {
		return nil
	}
	return c.authenticate(ctx)
}

func (c *Client) authenticate(ctx context.Context) error {
	params := url.Values{}
	params.Set("api", "SYNO.API.Auth")
	params.Set("version", "2")
	params.Set("method", "login")
	params.Set("account", c.config.Username)
	params.Set("passwd", c.config.Password)
	params.Set("format", "sid")
	params.Set("session", "DownloadStation")

	u := fmt.Sprintf("%s/webapi/auth.cgi?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return err
	}

	if !apiResp.Success {
		if apiResp.Error != nil && apiResp.Error.Code == 105 {
			return types.ErrAuthFailed
		}
		return fmt.Errorf("authentication failed: code %d", apiResp.Error.Code)
	}

	if apiResp.Data == nil {
		return errors.New("no data in auth response")
	}

	var auth authData
	if err := json.Unmarshal(*apiResp.Data, &auth); err != nil {
		return err
	}

	c.sid = auth.SID
	return nil
}

func (c *Client) isAuthError(code int) bool {
	return code == 105 || code == 106 || code == 107 || code == 119
}

func (c *Client) taskToDownloadItem(task taskData) types.DownloadItem {
	item := types.DownloadItem{
		ID:             task.ID,
		Name:           task.Title,
		Status:         c.mapStatus(task.Status),
		Progress:       0,
		Size:           task.Size,
		DownloadedSize: 0,
		DownloadSpeed:  0,
		UploadSpeed:    0,
		ETA:            -1,
		DownloadDir:    "",
	}

	if task.Additional == nil {
		return item
	}

	c.fillDownloadDetails(&item, task.Additional)
	c.calculateProgress(&item)

	return item
}

func (c *Client) fillDownloadDetails(item *types.DownloadItem, additional *taskAdditionalData) {
	if additional.Detail != nil {
		item.DownloadDir = additional.Detail.Destination
	}

	if additional.Transfer == nil {
		return
	}

	if downloaded, err := strconv.ParseInt(additional.Transfer.SizeDownloaded, 10, 64); err == nil {
		item.DownloadedSize = downloaded
	}
	if downloadSpeed, err := strconv.ParseInt(additional.Transfer.SpeedDownload, 10, 64); err == nil {
		item.DownloadSpeed = downloadSpeed
	}
	if uploadSpeed, err := strconv.ParseInt(additional.Transfer.SpeedUpload, 10, 64); err == nil {
		item.UploadSpeed = uploadSpeed
	}
}

func (c *Client) calculateProgress(item *types.DownloadItem) {
	if item.Size > 0 && item.DownloadedSize > 0 {
		item.Progress = float64(item.DownloadedSize) / float64(item.Size) * 100
	}
}

func (c *Client) mapStatus(status string) types.Status {
	switch status {
	case "downloading", "finishing", "hash_checking", "extracting", "captcha_needed":
		return types.StatusDownloading
	case "paused":
		return types.StatusPaused
	case "finished":
		return types.StatusCompleted
	case "seeding":
		return types.StatusSeeding
	case "error":
		return types.StatusError
	case "waiting":
		return types.StatusQueued
	default:
		return types.StatusQueued
	}
}

func (c *Client) extractIDFromURI(uri string) string {
	if strings.HasPrefix(uri, "magnet:") {
		if idx := strings.Index(uri, "xt=urn:btih:"); idx != -1 {
			hash := uri[idx+12:]
			if endIdx := strings.Index(hash, "&"); endIdx != -1 {
				hash = hash[:endIdx]
			}
			return strings.ToLower(hash)
		}
		return uri
	}
	return uri
}

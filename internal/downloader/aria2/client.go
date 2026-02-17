package aria2

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

type Client struct {
	config     types.ClientConfig
	httpClient *http.Client
	requestID  int
}

var _ types.TorrentClient = (*Client)(nil)

func NewFromConfig(cfg *types.ClientConfig) *Client {
	return &Client{
		config: *cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Type() types.ClientType {
	return types.ClientTypeAria2
}

func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

func (c *Client) Test(ctx context.Context) error {
	result, err := c.call(ctx, "aria2.getVersion", nil)
	if err != nil {
		return err
	}

	versionMap, ok := result.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid version response from aria2")
	}

	version, _ := versionMap["version"].(string)
	if version == "" {
		return fmt.Errorf("empty version response from aria2")
	}

	meetsMin, err := types.CompareVersions(version, "1.34.0")
	if err != nil {
		return fmt.Errorf("failed to parse aria2 version %q: %w", version, err)
	}
	if !meetsMin {
		return fmt.Errorf("aria2 version %s is below minimum required version 1.34.0", version)
	}

	return nil
}

func (c *Client) Connect(ctx context.Context) error {
	return c.Test(ctx)
}

func (c *Client) Add(ctx context.Context, opts *types.AddOptions) (string, error) {
	if opts.URL != "" {
		return c.addURI(ctx, opts.URL, opts)
	}
	if len(opts.FileContent) > 0 {
		return c.addTorrentFile(ctx, opts)
	}
	return "", fmt.Errorf("either URL or FileContent must be provided")
}

func (c *Client) AddMagnet(ctx context.Context, magnetURL string, opts *types.AddOptions) (string, error) {
	return c.addURI(ctx, magnetURL, opts)
}

func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	active, err := c.call(ctx, "aria2.tellActive", nil)
	if err != nil {
		return nil, fmt.Errorf("tellActive: %w", err)
	}

	waiting, err := c.call(ctx, "aria2.tellWaiting", []any{0, 1000})
	if err != nil {
		return nil, fmt.Errorf("tellWaiting: %w", err)
	}

	stopped, err := c.call(ctx, "aria2.tellStopped", []any{0, 1000})
	if err != nil {
		return nil, fmt.Errorf("tellStopped: %w", err)
	}

	var items []types.DownloadItem
	for _, list := range []any{active, waiting, stopped} {
		entries, ok := list.([]any)
		if !ok {
			continue
		}
		for _, entry := range entries {
			statusObj, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			items = append(items, c.mapToDownloadItem(statusObj))
		}
	}

	if items == nil {
		items = []types.DownloadItem{}
	}

	return items, nil
}

func (c *Client) Get(ctx context.Context, id string) (*types.DownloadItem, error) {
	resp, err := c.call(ctx, "aria2.tellStatus", []any{id})
	if err != nil {
		return nil, err
	}

	statusObj, ok := resp.(map[string]any)
	if !ok {
		return nil, types.ErrNotFound
	}

	item := c.mapToDownloadItem(statusObj)
	return &item, nil
}

func (c *Client) Remove(ctx context.Context, id string, _ bool) error {
	_, err := c.call(ctx, "aria2.forceRemove", []any{id})
	if err != nil {
		// forceRemove only works for active/waiting downloads;
		// for completed/error/removed items, use removeDownloadResult
		_, err = c.call(ctx, "aria2.removeDownloadResult", []any{id})
	}
	return err
}

func (c *Client) Pause(ctx context.Context, id string) error {
	_, err := c.call(ctx, "aria2.forcePause", []any{id})
	return err
}

func (c *Client) Resume(ctx context.Context, id string) error {
	_, err := c.call(ctx, "aria2.unpause", []any{id})
	return err
}

func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	resp, err := c.call(ctx, "aria2.getGlobalOption", nil)
	if err != nil {
		return "", err
	}

	opts, ok := resp.(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected response type for getGlobalOption")
	}

	dir, ok := opts["dir"].(string)
	if !ok {
		return "", fmt.Errorf("dir not found in global options")
	}

	return dir, nil
}

func (c *Client) SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error {
	options := make(map[string]any)

	if ratio > 0 {
		options["seed-ratio"] = strconv.FormatFloat(ratio, 'f', -1, 64)
	}

	if seedTime > 0 {
		minutes := int(seedTime.Minutes())
		options["seed-time"] = strconv.Itoa(minutes)
	}

	if len(options) == 0 {
		return nil
	}

	_, err := c.call(ctx, "aria2.changeOption", []any{id, options})
	return err
}

func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	resp, err := c.call(ctx, "aria2.tellStatus", []any{id})
	if err != nil {
		return nil, err
	}

	statusObj, ok := resp.(map[string]any)
	if !ok {
		return nil, types.ErrNotFound
	}

	item := c.mapToDownloadItem(statusObj)
	infoHash := getString(statusObj, "infoHash")

	uploaded := parseIntString(getString(statusObj, "uploadLength"))
	totalLen := parseIntString(getString(statusObj, "totalLength"))

	var ratio float64
	if totalLen > 0 {
		ratio = float64(uploaded) / float64(totalLen)
	}

	return &types.TorrentInfo{
		DownloadItem: item,
		InfoHash:     infoHash,
		Ratio:        ratio,
	}, nil
}

func (c *Client) addURI(ctx context.Context, uri string, opts *types.AddOptions) (string, error) {
	options := c.buildAddOptions(opts)
	resp, err := c.call(ctx, "aria2.addUri", []any{[]string{uri}, options})
	if err != nil {
		return "", err
	}

	gid, ok := resp.(string)
	if !ok {
		return "", fmt.Errorf("unexpected response type for addUri")
	}

	return gid, nil
}

func (c *Client) addTorrentFile(ctx context.Context, opts *types.AddOptions) (string, error) {
	b64Content := base64.StdEncoding.EncodeToString(opts.FileContent)
	options := c.buildAddOptions(opts)

	resp, err := c.call(ctx, "aria2.addTorrent", []any{b64Content, []string{}, options})
	if err != nil {
		return "", err
	}

	gid, ok := resp.(string)
	if !ok {
		return "", fmt.Errorf("unexpected response type for addTorrent")
	}

	return gid, nil
}

func (c *Client) buildAddOptions(opts *types.AddOptions) map[string]any {
	options := make(map[string]any)
	if opts == nil {
		return options
	}

	if opts.DownloadDir != "" {
		options["dir"] = opts.DownloadDir
	}

	if opts.Paused {
		options["pause"] = "true"
	}

	return options
}

func (c *Client) call(ctx context.Context, method string, extraParams []any) (any, error) {
	c.requestID++

	var params []any
	if c.config.APIKey != "" {
		params = append(params, "token:"+c.config.APIKey)
	}
	params = append(params, extraParams...)

	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      strconv.Itoa(c.requestID),
		"method":  method,
		"params":  params,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.buildURL(), bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp struct {
		Result any              `json:"result"`
		Error  *json.RawMessage `json:"error"`
	}

	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, c.parseRPCError(*rpcResp.Error)
	}

	return rpcResp.Result, nil
}

func (c *Client) parseRPCError(raw json.RawMessage) error {
	var errObj struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &errObj); err == nil {
		if errObj.Code == 1 && strings.Contains(strings.ToLower(errObj.Message), "unauthorized") {
			return types.ErrAuthFailed
		}
		return fmt.Errorf("RPC error: %s (code %d)", errObj.Message, errObj.Code)
	}
	return fmt.Errorf("RPC error: %s", string(raw))
}

func (c *Client) buildURL() string {
	scheme := "http"
	if c.config.UseSSL {
		scheme = "https"
	}

	urlPath := "/jsonrpc"
	if c.config.URLBase != "" {
		urlPath = "/" + strings.Trim(c.config.URLBase, "/") + "/jsonrpc"
	}

	return fmt.Sprintf("%s://%s:%d%s", scheme, c.config.Host, c.config.Port, urlPath)
}

func (c *Client) mapToDownloadItem(status map[string]any) types.DownloadItem {
	gid := getString(status, "gid")
	totalLength := parseIntString(getString(status, "totalLength"))
	completedLength := parseIntString(getString(status, "completedLength"))
	downloadSpeed := parseIntString(getString(status, "downloadSpeed"))
	uploadSpeed := parseIntString(getString(status, "uploadSpeed"))
	aria2Status := getString(status, "status")
	dir := getString(status, "dir")
	errorMessage := getString(status, "errorMessage")

	name := c.extractName(status)

	var progress float64
	if totalLength > 0 {
		progress = float64(completedLength) / float64(totalLength) * 100
	}

	var eta int64 = -1
	if downloadSpeed > 0 && totalLength > completedLength {
		remaining := totalLength - completedLength
		eta = remaining / downloadSpeed
	}

	mappedStatus := mapStatus(aria2Status, totalLength, completedLength)

	item := types.DownloadItem{
		ID:             gid,
		Name:           name,
		Status:         mappedStatus,
		Progress:       progress,
		Size:           totalLength,
		DownloadedSize: completedLength,
		DownloadSpeed:  downloadSpeed,
		UploadSpeed:    uploadSpeed,
		ETA:            eta,
		DownloadDir:    dir,
	}

	if mappedStatus == types.StatusError {
		item.Error = errorMessage
	}

	return item
}

func (c *Client) extractName(status map[string]any) string {
	if bt, ok := status["bittorrent"].(map[string]any); ok {
		if info, ok := bt["info"].(map[string]any); ok {
			if name, ok := info["name"].(string); ok && name != "" {
				return name
			}
		}
	}

	gid := getString(status, "gid")
	if gid != "" {
		return gid
	}

	return "unknown"
}

func mapStatus(aria2Status string, totalLength, completedLength int64) types.Status {
	switch aria2Status {
	case "active":
		if totalLength > 0 && completedLength >= totalLength {
			return types.StatusSeeding
		}
		return types.StatusDownloading
	case "waiting":
		return types.StatusQueued
	case "paused":
		return types.StatusPaused
	case "error":
		return types.StatusError
	case "complete":
		return types.StatusCompleted
	case "removed":
		return types.StatusUnknown
	default:
		return types.StatusUnknown
	}
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func parseIntString(s string) int64 {
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

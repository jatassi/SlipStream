package deluge

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
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
	jar, _ := cookiejar.New(nil)
	return &Client{
		config: *cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
		requestID: 0,
	}
}

func (c *Client) Type() types.ClientType {
	return types.ClientTypeDeluge
}

func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

func (c *Client) Test(ctx context.Context) error {
	if err := c.authenticate(ctx); err != nil {
		return err
	}
	_, err := c.call(ctx, "daemon.get_version", []any{})
	return err
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
	options := make(map[string]any)
	if opts != nil {
		if opts.Paused {
			options["add_paused"] = true
		}
		if opts.DownloadDir != "" {
			options["download_location"] = opts.DownloadDir
		}
	}

	resp, err := c.call(ctx, "core.add_torrent_magnet", []any{magnetURL, options})
	if err != nil {
		return "", err
	}

	hash, ok := resp.(string)
	if !ok {
		return "", fmt.Errorf("unexpected response type for add_torrent_magnet")
	}

	if opts != nil && opts.Category != "" {
		_, _ = c.call(ctx, "label.set_torrent", []any{hash, opts.Category})
	}

	return hash, nil
}

func (c *Client) addFile(ctx context.Context, opts *types.AddOptions) (string, error) {
	options := make(map[string]any)
	if opts.Paused {
		options["add_paused"] = true
	}
	if opts.DownloadDir != "" {
		options["download_location"] = opts.DownloadDir
	}

	filename := "torrent.torrent"
	if opts.Name != "" {
		filename = opts.Name
	}

	b64Content := base64.StdEncoding.EncodeToString(opts.FileContent)
	resp, err := c.call(ctx, "core.add_torrent_file", []any{filename, b64Content, options})
	if err != nil {
		return "", err
	}

	hash, ok := resp.(string)
	if !ok {
		return "", fmt.Errorf("unexpected response type for add_torrent_file")
	}

	if opts.Category != "" {
		_, _ = c.call(ctx, "label.set_torrent", []any{hash, opts.Category})
	}

	return hash, nil
}

func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	fields := []string{
		"hash", "name", "state", "progress", "eta", "message", "is_finished",
		"save_path", "total_size", "total_done", "time_added", "active_time",
		"ratio", "is_auto_managed", "stop_at_ratio", "remove_at_ratio", "stop_ratio",
	}

	resp, err := c.call(ctx, "web.update_ui", []any{fields, map[string]any{}})
	if err != nil {
		return nil, err
	}

	resultMap, ok := resp.(map[string]any)
	if !ok {
		return []types.DownloadItem{}, nil
	}

	torrentsMap, ok := resultMap["torrents"].(map[string]any)
	if !ok || torrentsMap == nil {
		return []types.DownloadItem{}, nil
	}

	items := make([]types.DownloadItem, 0, len(torrentsMap))
	for hash, torrentData := range torrentsMap {
		torrent, ok := torrentData.(map[string]any)
		if !ok {
			continue
		}
		items = append(items, c.mapToDownloadItem(hash, torrent))
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
		if strings.EqualFold(items[i].ID, lowerID) {
			return &items[i], nil
		}
	}

	return nil, types.ErrNotFound
}

func (c *Client) Remove(ctx context.Context, id string, deleteFiles bool) error {
	_, err := c.call(ctx, "core.remove_torrent", []any{strings.ToLower(id), deleteFiles})
	return err
}

func (c *Client) Pause(ctx context.Context, id string) error {
	_, err := c.call(ctx, "core.pause_torrent", []any{[]string{strings.ToLower(id)}})
	return err
}

func (c *Client) Resume(ctx context.Context, id string) error {
	_, err := c.call(ctx, "core.resume_torrent", []any{[]string{strings.ToLower(id)}})
	return err
}

func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	resp, err := c.call(ctx, "core.get_config", []any{})
	if err != nil {
		return "", err
	}

	configMap, ok := resp.(map[string]any)
	if !ok {
		return "", fmt.Errorf("unexpected response type for get_config")
	}

	downloadDir, ok := configMap["download_location"].(string)
	if !ok {
		return "", fmt.Errorf("download_location not found in config")
	}

	return downloadDir, nil
}

func (c *Client) SetSeedLimits(ctx context.Context, id string, ratio float64, seedTime time.Duration) error {
	options := make(map[string]any)

	if ratio > 0 {
		options["stop_at_ratio"] = true
		options["stop_ratio"] = ratio
	}

	if len(options) == 0 {
		return nil
	}

	_, err := c.call(ctx, "core.set_torrent_options", []any{[]string{strings.ToLower(id)}, options})
	return err
}

func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	item, err := c.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	fields := []string{"hash", "ratio"}
	resp, err := c.call(ctx, "web.update_ui", []any{fields, map[string]any{}})
	if err != nil {
		return nil, err
	}

	resultMap, ok := resp.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response type")
	}

	torrentsMap, ok := resultMap["torrents"].(map[string]any)
	if !ok {
		return nil, types.ErrNotFound
	}

	lowerID := strings.ToLower(id)
	torrentData, ok := torrentsMap[lowerID]
	if !ok {
		return nil, types.ErrNotFound
	}

	torrent, ok := torrentData.(map[string]any)
	if !ok {
		return nil, types.ErrNotFound
	}

	info := &types.TorrentInfo{
		DownloadItem: *item,
		InfoHash:     strings.ToLower(id),
		Ratio:        getFloat(torrent, "ratio"),
	}

	return info, nil
}

func (c *Client) authenticate(ctx context.Context) error {
	c.httpClient.Jar, _ = cookiejar.New(nil)

	resp, err := c.call(ctx, "auth.login", []any{c.config.Password})
	if err != nil {
		return err
	}

	success, ok := resp.(bool)
	if !ok || !success {
		return types.ErrAuthFailed
	}

	connected, err := c.call(ctx, "web.connected", []any{})
	if err != nil {
		return err
	}

	isConnected, ok := connected.(bool)
	if !ok {
		return fmt.Errorf("unexpected response from web.connected")
	}

	if isConnected {
		return nil
	}

	return c.connectToDaemon(ctx)
}

func (c *Client) connectToDaemon(ctx context.Context) error {
	hostsResp, err := c.call(ctx, "web.get_hosts", []any{})
	if err != nil {
		return err
	}

	hosts, ok := hostsResp.([]any)
	if !ok {
		return fmt.Errorf("unexpected response from web.get_hosts")
	}

	hostID := findLocalHostID(hosts)
	if hostID == "" {
		return fmt.Errorf("no local daemon found")
	}

	_, err = c.call(ctx, "web.connect", []any{hostID})
	return err
}

func findLocalHostID(hosts []any) string {
	for _, h := range hosts {
		host, ok := h.([]any)
		if !ok || len(host) < 2 {
			continue
		}
		id, _ := host[0].(string)
		ip, _ := host[1].(string)
		if id != "" && ip == "127.0.0.1" {
			return id
		}
	}
	return ""
}

func (c *Client) call(ctx context.Context, method string, params []any) (any, error) {
	result, err := c.doCall(ctx, method, params)
	if err != nil {
		if isAuthError(err) {
			if authErr := c.authenticate(ctx); authErr != nil {
				return nil, authErr
			}
			return c.doCall(ctx, method, params)
		}
		return nil, err
	}
	return result, nil
}

func (c *Client) doCall(ctx context.Context, method string, params []any) (any, error) {
	c.requestID++

	reqBody := map[string]any{
		"method": method,
		"params": params,
		"id":     c.requestID,
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
		ID     int              `json:"id"`
	}

	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, c.parseRPCError(*rpcResp.Error)
	}

	return rpcResp.Result, nil
}

func (c *Client) buildURL() string {
	scheme := "http"
	if c.config.UseSSL {
		scheme = "https"
	}

	urlPath := "/json"
	if c.config.URLBase != "" {
		urlPath = "/" + strings.Trim(c.config.URLBase, "/") + "/json"
	}

	return fmt.Sprintf("%s://%s:%d%s", scheme, c.config.Host, c.config.Port, urlPath)
}

func (c *Client) parseRPCError(raw json.RawMessage) error {
	var errObj struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}
	if err := json.Unmarshal(raw, &errObj); err == nil {
		if errObj.Code == 1 || errObj.Code == 2 {
			return &authError{msg: errObj.Message}
		}
		return fmt.Errorf("RPC error: %s (code %d)", errObj.Message, errObj.Code)
	}
	return fmt.Errorf("RPC error: %s", string(raw))
}

func (c *Client) mapToDownloadItem(hash string, torrent map[string]any) types.DownloadItem {
	state := getString(torrent, "state")
	isFinished := getBool(torrent, "is_finished")
	progress := getFloat(torrent, "progress")
	eta := getFloat(torrent, "eta")

	status := c.mapStatus(state, isFinished)

	etaSeconds := int64(-1)
	if eta > 0 {
		etaSeconds = int64(eta)
	}

	item := types.DownloadItem{
		ID:             strings.ToLower(hash),
		Name:           getString(torrent, "name"),
		Status:         status,
		Progress:       progress,
		Size:           int64(getFloat(torrent, "total_size")),
		DownloadedSize: int64(getFloat(torrent, "total_done")),
		ETA:            etaSeconds,
		DownloadDir:    getString(torrent, "save_path"),
	}

	if timeAdded := getFloat(torrent, "time_added"); timeAdded > 0 {
		item.AddedAt = time.Unix(int64(timeAdded), 0)
	}

	if status == types.StatusWarning {
		item.Error = getString(torrent, "message")
	}

	return item
}

func (c *Client) mapStatus(state string, isFinished bool) types.Status {
	if state == "Error" {
		return types.StatusWarning
	}

	if isFinished && state != "Checking" && state != "Error" {
		return types.StatusSeeding
	}

	switch state {
	case "Paused":
		return types.StatusPaused
	case "Queued":
		return types.StatusQueued
	case "Checking", "Moving", "Downloading":
		return types.StatusDownloading
	case "Seeding":
		return types.StatusSeeding
	default:
		return types.StatusUnknown
	}
}

type authError struct {
	msg string
}

func (e *authError) Error() string {
	return e.msg
}

func isAuthError(err error) bool {
	var authErr *authError
	return errors.As(err, &authErr)
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat(m map[string]any, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

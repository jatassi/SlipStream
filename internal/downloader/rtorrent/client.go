package rtorrent

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

var _ types.TorrentClient = (*Client)(nil)

const xmlValueTag = "value"

// fieldSelectors are the d.multicall2 fields used to list torrents.
var fieldSelectors = []string{
	"d.hash=",
	"d.name=",
	"d.base_path=",
	"d.custom1=",
	"d.size_bytes=",
	"d.left_bytes=",
	"d.down.rate=",
	"d.up.rate=",
	"d.ratio=",
	"d.is_open=",
	"d.is_active=",
	"d.complete=",
	"d.timestamp.finished=",
	"d.message=",
}

type Client struct {
	config     types.ClientConfig
	httpClient *http.Client
	baseURL    string
}

func NewFromConfig(cfg *types.ClientConfig) *Client {
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}

	urlBase := cfg.URLBase
	if urlBase == "" {
		urlBase = "RPC2"
	}
	urlBase = strings.TrimPrefix(urlBase, "/")
	urlBase = strings.TrimSuffix(urlBase, "/")

	baseURL := fmt.Sprintf("%s://%s:%d/%s", scheme, cfg.Host, cfg.Port, urlBase)

	return &Client{
		config: *cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (c *Client) Type() types.ClientType {
	return types.ClientTypeRTorrent
}

func (c *Client) Protocol() types.Protocol {
	return types.ProtocolTorrent
}

func (c *Client) Test(ctx context.Context) error {
	result, err := c.call(ctx, "system.client_version", nil)
	if err != nil {
		return err
	}

	version, ok := result.(string)
	if !ok || version == "" {
		return fmt.Errorf("invalid version response from rTorrent")
	}

	meetsMin, err := types.CompareVersions(version, "0.9.0")
	if err != nil {
		return fmt.Errorf("failed to parse rTorrent version %q: %w", version, err)
	}
	if !meetsMin {
		return fmt.Errorf("rTorrent version %s is below minimum required version 0.9.0", version)
	}

	return nil
}

func (c *Client) Connect(ctx context.Context) error {
	return c.Test(ctx)
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
	methodName := "load.start"
	if opts != nil && opts.Paused {
		methodName = "load.normal"
	}

	params := []xmlRPCValue{
		{Type: "string", Value: ""},
		{Type: "string", Value: magnetURL},
	}

	params = append(params, addCommandParams(opts, c.config.Category)...)

	_, err := c.call(ctx, methodName, params)
	if err != nil {
		return "", err
	}

	return extractHashFromMagnet(magnetURL), nil
}

func (c *Client) addFile(ctx context.Context, opts *types.AddOptions) (string, error) {
	methodName := "load.raw_start"
	if opts != nil && opts.Paused {
		methodName = "load.raw"
	}

	encoded := base64.StdEncoding.EncodeToString(opts.FileContent)

	params := []xmlRPCValue{
		{Type: "string", Value: ""},
		{Type: "base64", Value: encoded},
	}

	params = append(params, addCommandParams(opts, c.config.Category)...)

	_, err := c.call(ctx, methodName, params)
	if err != nil {
		return "", err
	}

	return "", nil
}

func (c *Client) List(ctx context.Context) ([]types.DownloadItem, error) {
	params := []xmlRPCValue{
		{Type: "string", Value: ""},
		{Type: "string", Value: ""},
	}
	for _, sel := range fieldSelectors {
		params = append(params, xmlRPCValue{Type: "string", Value: sel})
	}

	resp, err := c.call(ctx, "d.multicall2", params)
	if err != nil {
		return nil, err
	}

	outerArray, ok := resp.([]any)
	if !ok {
		return []types.DownloadItem{}, nil
	}

	items := make([]types.DownloadItem, 0, len(outerArray))
	for _, row := range outerArray {
		fields, ok := row.([]any)
		if !ok || len(fields) < len(fieldSelectors) {
			continue
		}
		items = append(items, mapTorrentFields(fields))
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

func (c *Client) Remove(_ context.Context, id string, _ bool) error {
	_, err := c.call(context.Background(), "d.erase", []xmlRPCValue{
		{Type: "string", Value: strings.ToUpper(id)},
	})
	return err
}

func (c *Client) Pause(_ context.Context, id string) error {
	_, err := c.call(context.Background(), "d.stop", []xmlRPCValue{
		{Type: "string", Value: strings.ToUpper(id)},
	})
	return err
}

func (c *Client) Resume(_ context.Context, id string) error {
	_, err := c.call(context.Background(), "d.start", []xmlRPCValue{
		{Type: "string", Value: strings.ToUpper(id)},
	})
	return err
}

func (c *Client) GetDownloadDir(ctx context.Context) (string, error) {
	items, err := c.List(ctx)
	if err != nil {
		return "", err
	}

	for i := range items {
		if items[i].DownloadDir != "" {
			return filepath.Dir(items[i].DownloadDir), nil
		}
	}

	return "", fmt.Errorf("no torrents available to determine download directory")
}

func (c *Client) SetSeedLimits(_ context.Context, _ string, _ float64, _ time.Duration) error {
	return types.ErrNotImplemented
}

func (c *Client) GetTorrentInfo(ctx context.Context, id string) (*types.TorrentInfo, error) {
	item, err := c.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return &types.TorrentInfo{
		DownloadItem: *item,
		InfoHash:     strings.ToLower(id),
	}, nil
}

// xmlRPCValue represents a typed XML-RPC parameter.
type xmlRPCValue struct {
	Type  string // "string", "int", "base64"
	Value string
}

func (c *Client) call(ctx context.Context, method string, params []xmlRPCValue) (any, error) {
	reqBody, err := buildXMLRPCRequest(method, params)
	if err != nil {
		return nil, fmt.Errorf("failed to build XML-RPC request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml")

	if c.config.Username != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, types.ErrAuthFailed
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return parseXMLRPCResponse(body)
}

func buildXMLRPCRequest(method string, params []xmlRPCValue) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0"?>`)
	buf.WriteString(`<methodCall>`)
	buf.WriteString(`<methodName>`)
	if err := xml.EscapeText(&buf, []byte(method)); err != nil {
		return nil, err
	}
	buf.WriteString(`</methodName>`)

	if len(params) > 0 {
		buf.WriteString(`<params>`)
		for _, p := range params {
			buf.WriteString(`<param><value>`)
			switch p.Type {
			case "base64":
				buf.WriteString(`<base64>`)
				buf.WriteString(p.Value)
				buf.WriteString(`</base64>`)
			case "int":
				buf.WriteString(`<i4>`)
				buf.WriteString(p.Value)
				buf.WriteString(`</i4>`)
			default:
				buf.WriteString(`<string>`)
				if err := xml.EscapeText(&buf, []byte(p.Value)); err != nil {
					return nil, err
				}
				buf.WriteString(`</string>`)
			}
			buf.WriteString(`</value></param>`)
		}
		buf.WriteString(`</params>`)
	}

	buf.WriteString(`</methodCall>`)
	return buf.Bytes(), nil
}

// XML-RPC response parsing types.

type methodResponse struct {
	Params *responseParams `xml:"params"`
	Fault  *responseFault  `xml:"fault"`
}

type responseParams struct {
	Param []responseParam `xml:"param"`
}

type responseParam struct {
	Value responseValue `xml:"value"`
}

type responseFault struct {
	Value responseValue `xml:"value"`
}

type responseValue struct {
	Inner []byte `xml:",innerxml"`
}

func parseXMLRPCResponse(data []byte) (any, error) {
	var resp methodResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse XML-RPC response: %w", err)
	}

	if resp.Fault != nil {
		return nil, parseFault(resp.Fault.Value.Inner)
	}

	if resp.Params == nil || len(resp.Params.Param) == 0 {
		return "", nil
	}

	return parseValue(resp.Params.Param[0].Value.Inner)
}

func parseFault(inner []byte) error {
	val, err := parseValue(inner)
	if err != nil {
		return fmt.Errorf("XML-RPC fault: %s", string(inner))
	}

	if m, ok := val.(map[string]any); ok {
		faultString, _ := m["faultString"].(string)
		return fmt.Errorf("XML-RPC fault: %s", faultString)
	}

	return fmt.Errorf("XML-RPC fault: %v", val)
}

func parseValue(raw []byte) (any, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return "", nil
	}

	decoder := xml.NewDecoder(bytes.NewReader(trimmed))
	return decodeValue(decoder)
}

func decodeValue(decoder *xml.Decoder) (any, error) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}

		switch t := token.(type) {
		case xml.StartElement:
			return decodeTypedValue(decoder, t.Name.Local)
		case xml.CharData:
			s := strings.TrimSpace(string(t))
			if s != "" {
				return s, nil
			}
		}
	}
}

//nolint:cyclop // XML-RPC type dispatch requires handling many type variants
func decodeTypedValue(decoder *xml.Decoder, typeName string) (any, error) {
	switch typeName {
	case "string":
		return decodeStringContent(decoder, "string")
	case "int", "i4", "i8":
		return decodeIntContent(decoder, typeName)
	case "base64":
		return decodeStringContent(decoder, "base64")
	case "array":
		return decodeArray(decoder)
	case "struct":
		return decodeStruct(decoder)
	case xmlValueTag:
		return decodeValue(decoder)
	case "boolean":
		content, _ := decodeStringContent(decoder, "boolean")
		s, _ := content.(string)
		return s == "1", nil
	default:
		return decodeStringContent(decoder, typeName)
	}
}

func decodeStringContent(decoder *xml.Decoder, endTag string) (any, error) {
	var content strings.Builder
	for {
		token, err := decoder.Token()
		if err != nil {
			return content.String(), err
		}
		switch t := token.(type) {
		case xml.CharData:
			content.Write(t)
		case xml.EndElement:
			if t.Name.Local == endTag {
				return content.String(), nil
			}
		}
	}
}

func decodeIntContent(decoder *xml.Decoder, endTag string) (any, error) {
	s, err := decodeStringContent(decoder, endTag)
	if err != nil {
		return int64(0), err
	}
	str, ok := s.(string)
	if !ok {
		return int64(0), nil
	}
	return parseIntString(strings.TrimSpace(str)), nil
}

func parseIntString(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

func decodeArray(decoder *xml.Decoder) ([]any, error) {
	items := []any{}

	for {
		token, err := decoder.Token()
		if err != nil {
			return items, err
		}

		if end, ok := token.(xml.EndElement); ok {
			if end.Name.Local == "array" || end.Name.Local == "data" {
				return items, nil
			}
			continue
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != xmlValueTag {
			continue
		}

		val, valErr := decodeValue(decoder)
		if valErr != nil {
			return items, valErr
		}
		items = append(items, val)
		consumeEndElement(decoder, xmlValueTag)
	}
}

func decodeStruct(decoder *xml.Decoder) (any, error) {
	result := make(map[string]any)

	for {
		token, err := decoder.Token()
		if err != nil {
			return result, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "member" {
				name, val := decodeMember(decoder)
				if name != "" {
					result[name] = val
				}
			}
		case xml.EndElement:
			if t.Name.Local == "struct" {
				return result, nil
			}
		}
	}
}

func decodeMember(decoder *xml.Decoder) (memberName string, memberVal any) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return memberName, memberVal
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "name":
				s, _ := decodeStringContent(decoder, "name")
				memberName, _ = s.(string)
			case xmlValueTag:
				memberVal, _ = decodeValue(decoder)
				consumeEndElement(decoder, xmlValueTag)
			}
		case xml.EndElement:
			if t.Name.Local == "member" {
				return memberName, memberVal
			}
		}
	}
}

func consumeEndElement(decoder *xml.Decoder, name string) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return
		}
		if end, ok := token.(xml.EndElement); ok && end.Name.Local == name {
			return
		}
	}
}

func mapTorrentFields(fields []any) types.DownloadItem {
	hash := asString(fields[0])
	name := asString(fields[1])
	basePath := asString(fields[2])
	sizeBytes := asInt64(fields[4])
	leftBytes := asInt64(fields[5])
	downRate := asInt64(fields[6])
	upRate := asInt64(fields[7])
	isActive := asInt64(fields[10])
	complete := asInt64(fields[11])
	timestampFinished := asInt64(fields[12])
	message := asString(fields[13])

	downloaded := sizeBytes - leftBytes
	var progress float64
	if sizeBytes > 0 {
		progress = float64(downloaded) / float64(sizeBytes) * 100
	}

	var eta int64 = -1
	if downRate > 0 && leftBytes > 0 {
		eta = leftBytes / downRate
	}

	status := mapStatus(complete == 1, isActive == 1, message)

	item := types.DownloadItem{
		ID:             strings.ToLower(hash),
		Name:           name,
		Status:         status,
		Progress:       progress,
		Size:           sizeBytes,
		DownloadedSize: downloaded,
		DownloadSpeed:  downRate,
		UploadSpeed:    upRate,
		ETA:            eta,
		DownloadDir:    basePath,
	}

	if timestampFinished > 0 {
		item.CompletedAt = time.Unix(timestampFinished, 0)
	}

	if status == types.StatusWarning {
		item.Error = message
	}

	return item
}

func mapStatus(isComplete, isActive bool, message string) types.Status {
	if message != "" {
		return types.StatusWarning
	}
	if isComplete {
		if isActive {
			return types.StatusSeeding
		}
		return types.StatusCompleted
	}
	if isActive {
		return types.StatusDownloading
	}
	return types.StatusPaused
}

func addCommandParams(opts *types.AddOptions, defaultCategory string) []xmlRPCValue {
	var params []xmlRPCValue

	category := ""
	if opts != nil {
		category = opts.Category
	}
	if category == "" {
		category = defaultCategory
	}
	if category != "" {
		params = append(params, xmlRPCValue{Type: "string", Value: "d.custom1.set=" + category})
	}

	if opts != nil && opts.DownloadDir != "" {
		params = append(params, xmlRPCValue{Type: "string", Value: "d.directory.set=" + opts.DownloadDir})
	}

	return params
}

func extractHashFromMagnet(magnetURL string) string {
	if !strings.HasPrefix(magnetURL, "magnet:") {
		return ""
	}

	parts := strings.SplitN(magnetURL, "?", 2)
	if len(parts) < 2 {
		return ""
	}

	for _, param := range strings.Split(parts[1], "&") {
		if strings.HasPrefix(param, "xt=urn:btih:") {
			return strings.ToLower(strings.TrimPrefix(param, "xt=urn:btih:"))
		}
	}

	return ""
}

func asString(v any) string {
	if val, ok := v.(string); ok {
		return val
	}
	return fmt.Sprintf("%v", v)
}

func asInt64(v any) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case int:
		return int64(val)
	case float64:
		return int64(val)
	case string:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}

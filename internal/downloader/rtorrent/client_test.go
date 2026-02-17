package rtorrent

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

func TestClient_Type(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{Host: "localhost", Port: 8080})
	if client.Type() != types.ClientTypeRTorrent {
		t.Errorf("expected %s, got %s", types.ClientTypeRTorrent, client.Type())
	}
}

func TestClient_Protocol(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{Host: "localhost", Port: 8080})
	if client.Protocol() != types.ProtocolTorrent {
		t.Errorf("expected %s, got %s", types.ProtocolTorrent, client.Protocol())
	}
}

func TestClient_Test_Success(t *testing.T) {
	server := httptest.NewServer(xmlRPCHandler(t, map[string]string{
		"system.client_version": xmlRPCStringResponse("0.9.8"),
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Test() failed: %v", err)
	}
}

func TestClient_Test_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{
		Username: "admin",
		Password: "wrong",
	})

	err := client.Test(context.Background())
	if !errors.Is(err, types.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

func TestClient_Test_BasicAuth(t *testing.T) {
	var receivedUser, receivedPass string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUser, receivedPass, _ = r.BasicAuth()
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(xmlRPCStringResponse("0.9.8")))
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{
		Username: "myuser",
		Password: "mypass",
	})

	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Test() failed: %v", err)
	}

	if receivedUser != "myuser" {
		t.Errorf("expected username 'myuser', got '%s'", receivedUser)
	}
	if receivedPass != "mypass" {
		t.Errorf("expected password 'mypass', got '%s'", receivedPass)
	}
}

func TestClient_List(t *testing.T) {
	respXML := `<?xml version="1.0"?>
<methodResponse>
<params><param><value><array><data>
<value><array><data>
  <value><string>AABB00112233445566778899AABB00112233CCDD</string></value>
  <value><string>Ubuntu 24.04</string></value>
  <value><string>/downloads/Ubuntu 24.04</string></value>
  <value><string>linux</string></value>
  <value><i8>4294967296</i8></value>
  <value><i8>1073741824</i8></value>
  <value><i8>1048576</i8></value>
  <value><i8>524288</i8></value>
  <value><i8>500</i8></value>
  <value><i8>1</i8></value>
  <value><i8>1</i8></value>
  <value><i8>0</i8></value>
  <value><i8>0</i8></value>
  <value><string></string></value>
</data></array></value>
<value><array><data>
  <value><string>DEADBEEF00112233445566778899AABBCCDDEEFF</string></value>
  <value><string>Debian 12</string></value>
  <value><string>/downloads/Debian 12</string></value>
  <value><string></string></value>
  <value><i8>2147483648</i8></value>
  <value><i8>0</i8></value>
  <value><i8>0</i8></value>
  <value><i8>2097152</i8></value>
  <value><i8>1500</i8></value>
  <value><i8>1</i8></value>
  <value><i8>1</i8></value>
  <value><i8>1</i8></value>
  <value><i8>1700000000</i8></value>
  <value><string></string></value>
</data></array></value>
<value><array><data>
  <value><string>1122334455667788990011223344556677889900</string></value>
  <value><string>Fedora 40</string></value>
  <value><string>/downloads/Fedora 40</string></value>
  <value><string>linux%20distros</string></value>
  <value><i8>3221225472</i8></value>
  <value><i8>3221225472</i8></value>
  <value><i8>0</i8></value>
  <value><i8>0</i8></value>
  <value><i8>0</i8></value>
  <value><i8>1</i8></value>
  <value><i8>0</i8></value>
  <value><i8>0</i8></value>
  <value><i8>0</i8></value>
  <value><string></string></value>
</data></array></value>
</data></array></value></param></params>
</methodResponse>`

	server := httptest.NewServer(xmlRPCHandler(t, map[string]string{
		"d.multicall2": respXML,
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	// First torrent: downloading (incomplete, active)
	item0 := items[0]
	if item0.ID != "aabb00112233445566778899aabb00112233ccdd" {
		t.Errorf("expected lowercase hash, got '%s'", item0.ID)
	}
	if item0.Name != "Ubuntu 24.04" {
		t.Errorf("expected name 'Ubuntu 24.04', got '%s'", item0.Name)
	}
	if item0.Status != types.StatusDownloading {
		t.Errorf("expected StatusDownloading, got %s", item0.Status)
	}
	if item0.Size != 4294967296 {
		t.Errorf("expected size 4294967296, got %d", item0.Size)
	}
	expectedDownloaded := int64(4294967296 - 1073741824)
	if item0.DownloadedSize != expectedDownloaded {
		t.Errorf("expected downloaded %d, got %d", expectedDownloaded, item0.DownloadedSize)
	}
	if item0.DownloadSpeed != 1048576 {
		t.Errorf("expected download speed 1048576, got %d", item0.DownloadSpeed)
	}
	if item0.UploadSpeed != 524288 {
		t.Errorf("expected upload speed 524288, got %d", item0.UploadSpeed)
	}

	// Second torrent: seeding (complete, active)
	item1 := items[1]
	if item1.Status != types.StatusSeeding {
		t.Errorf("expected StatusSeeding, got %s", item1.Status)
	}
	if item1.CompletedAt.Unix() != 1700000000 {
		t.Errorf("expected completedAt 1700000000, got %d", item1.CompletedAt.Unix())
	}

	// Third torrent: paused (incomplete, not active)
	item2 := items[2]
	if item2.Status != types.StatusPaused {
		t.Errorf("expected StatusPaused, got %s", item2.Status)
	}
	if item2.DownloadDir != "/downloads/Fedora 40" {
		t.Errorf("expected download dir '/downloads/Fedora 40', got '%s'", item2.DownloadDir)
	}
}

func TestClient_List_Empty(t *testing.T) {
	respXML := `<?xml version="1.0"?>
<methodResponse>
<params><param><value><array><data>
</data></array></value></param></params>
</methodResponse>`

	server := httptest.NewServer(xmlRPCHandler(t, map[string]string{
		"d.multicall2": respXML,
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestClient_Add_URL(t *testing.T) {
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedMethod = extractMethodName(body)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(xmlRPCSuccessResponse()))
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	magnetURL := "magnet:?xt=urn:btih:AABBCCDD1122334455&dn=test"
	hash, err := client.Add(context.Background(), &types.AddOptions{
		URL: magnetURL,
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if receivedMethod != "load.start" {
		t.Errorf("expected method 'load.start', got '%s'", receivedMethod)
	}

	if hash != "aabbccdd1122334455" {
		t.Errorf("expected hash 'aabbccdd1122334455', got '%s'", hash)
	}
}

func TestClient_Add_URL_Paused(t *testing.T) {
	var receivedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedMethod = extractMethodName(body)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(xmlRPCSuccessResponse()))
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	magnetURL := "magnet:?xt=urn:btih:AABBCCDD1122334455&dn=test"
	_, err := client.Add(context.Background(), &types.AddOptions{
		URL:    magnetURL,
		Paused: true,
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if receivedMethod != "load.normal" {
		t.Errorf("expected method 'load.normal', got '%s'", receivedMethod)
	}
}

func TestClient_Add_FileContent(t *testing.T) {
	var receivedMethod string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		receivedMethod = extractMethodName(receivedBody)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(xmlRPCSuccessResponse()))
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	_, err := client.Add(context.Background(), &types.AddOptions{
		FileContent: []byte("fake torrent content"),
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if receivedMethod != "load.raw_start" {
		t.Errorf("expected method 'load.raw_start', got '%s'", receivedMethod)
	}

	if !strings.Contains(string(receivedBody), "<base64>") {
		t.Error("expected base64 encoded content in request")
	}
}

func TestClient_Remove(t *testing.T) {
	var receivedMethod string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		receivedMethod = extractMethodName(receivedBody)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(xmlRPCSuccessResponse()))
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	err := client.Remove(context.Background(), "aabbccdd", false)
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	if receivedMethod != "d.erase" {
		t.Errorf("expected method 'd.erase', got '%s'", receivedMethod)
	}

	if !strings.Contains(string(receivedBody), "AABBCCDD") {
		t.Error("expected uppercase hash in d.erase request")
	}
}

func TestClient_Pause(t *testing.T) {
	var receivedMethod string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		receivedMethod = extractMethodName(receivedBody)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(xmlRPCSuccessResponse()))
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	err := client.Pause(context.Background(), "aabbccdd")
	if err != nil {
		t.Fatalf("Pause() failed: %v", err)
	}

	if receivedMethod != "d.stop" {
		t.Errorf("expected method 'd.stop', got '%s'", receivedMethod)
	}

	if !strings.Contains(string(receivedBody), "AABBCCDD") {
		t.Error("expected uppercase hash in d.stop request")
	}
}

func TestClient_Resume(t *testing.T) {
	var receivedMethod string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		receivedMethod = extractMethodName(receivedBody)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(xmlRPCSuccessResponse()))
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	err := client.Resume(context.Background(), "aabbccdd")
	if err != nil {
		t.Fatalf("Resume() failed: %v", err)
	}

	if receivedMethod != "d.start" {
		t.Errorf("expected method 'd.start', got '%s'", receivedMethod)
	}

	if !strings.Contains(string(receivedBody), "AABBCCDD") {
		t.Error("expected uppercase hash in d.start request")
	}
}

func TestClient_GetDownloadDir(t *testing.T) {
	respXML := `<?xml version="1.0"?>
<methodResponse>
<params><param><value><array><data>
<value><array><data>
  <value><string>AABB00112233445566778899AABB00112233CCDD</string></value>
  <value><string>Ubuntu 24.04</string></value>
  <value><string>/downloads/complete/Ubuntu 24.04</string></value>
  <value><string></string></value>
  <value><i8>4294967296</i8></value>
  <value><i8>0</i8></value>
  <value><i8>0</i8></value>
  <value><i8>0</i8></value>
  <value><i8>1000</i8></value>
  <value><i8>1</i8></value>
  <value><i8>1</i8></value>
  <value><i8>1</i8></value>
  <value><i8>1700000000</i8></value>
  <value><string></string></value>
</data></array></value>
</data></array></value></param></params>
</methodResponse>`

	server := httptest.NewServer(xmlRPCHandler(t, map[string]string{
		"d.multicall2": respXML,
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	dir, err := client.GetDownloadDir(context.Background())
	if err != nil {
		t.Fatalf("GetDownloadDir() failed: %v", err)
	}

	if dir != "/downloads/complete" {
		t.Errorf("expected '/downloads/complete', got '%s'", dir)
	}
}

func TestClient_Get_Found(t *testing.T) {
	respXML := `<?xml version="1.0"?>
<methodResponse>
<params><param><value><array><data>
<value><array><data>
  <value><string>AABB00112233445566778899AABB00112233CCDD</string></value>
  <value><string>Ubuntu 24.04</string></value>
  <value><string>/downloads/Ubuntu 24.04</string></value>
  <value><string></string></value>
  <value><i8>4294967296</i8></value>
  <value><i8>0</i8></value>
  <value><i8>0</i8></value>
  <value><i8>0</i8></value>
  <value><i8>1000</i8></value>
  <value><i8>1</i8></value>
  <value><i8>1</i8></value>
  <value><i8>1</i8></value>
  <value><i8>0</i8></value>
  <value><string></string></value>
</data></array></value>
</data></array></value></param></params>
</methodResponse>`

	server := httptest.NewServer(xmlRPCHandler(t, map[string]string{
		"d.multicall2": respXML,
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	item, err := client.Get(context.Background(), "AABB00112233445566778899AABB00112233CCDD")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if item.Name != "Ubuntu 24.04" {
		t.Errorf("expected name 'Ubuntu 24.04', got '%s'", item.Name)
	}
}

func TestClient_Get_NotFound(t *testing.T) {
	respXML := `<?xml version="1.0"?>
<methodResponse>
<params><param><value><array><data>
</data></array></value></param></params>
</methodResponse>`

	server := httptest.NewServer(xmlRPCHandler(t, map[string]string{
		"d.multicall2": respXML,
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	_, err := client.Get(context.Background(), "nonexistenthash")
	if !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestClient_XMLRPCFault(t *testing.T) {
	faultXML := `<?xml version="1.0"?>
<methodResponse>
<fault><value><struct>
<member><name>faultCode</name><value><int>-503</int></value></member>
<member><name>faultString</name><value><string>Method not found</string></value></member>
</struct></value></fault>
</methodResponse>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(faultXML))
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	err := client.Test(context.Background())
	if err == nil {
		t.Fatal("expected error from fault response, got nil")
	}

	if !strings.Contains(err.Error(), "Method not found") {
		t.Errorf("expected fault message in error, got: %s", err.Error())
	}
}

func TestMapStatus(t *testing.T) {
	tests := []struct {
		name       string
		isComplete bool
		isActive   bool
		message    string
		expected   types.Status
	}{
		{"downloading", false, true, "", types.StatusDownloading},
		{"seeding", true, true, "", types.StatusSeeding},
		{"completed_inactive", true, false, "", types.StatusCompleted},
		{"paused", false, false, "", types.StatusPaused},
		{"error_downloading", false, true, "Tracker error", types.StatusWarning},
		{"error_seeding", true, true, "I/O error", types.StatusWarning},
		{"error_paused", false, false, "Disk full", types.StatusWarning},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapStatus(tt.isComplete, tt.isActive, tt.message)
			if result != tt.expected {
				t.Errorf("mapStatus(complete=%v, active=%v, msg=%q) = %s, expected %s",
					tt.isComplete, tt.isActive, tt.message, result, tt.expected)
			}
		})
	}
}

func TestExtractHashFromMagnet(t *testing.T) {
	tests := []struct {
		name     string
		magnet   string
		expected string
	}{
		{"valid", "magnet:?xt=urn:btih:ABC123&dn=test", "abc123"},
		{"uppercase", "magnet:?xt=urn:btih:DEADBEEF&dn=test", "deadbeef"},
		{"not_magnet", "http://example.com/file.torrent", ""},
		{"no_hash", "magnet:?dn=test", ""},
		{"no_query", "magnet:", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHashFromMagnet(tt.magnet)
			if result != tt.expected {
				t.Errorf("extractHashFromMagnet(%s) = %s, expected %s", tt.magnet, result, tt.expected)
			}
		})
	}
}

func TestClient_DefaultURLBase(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{
		Host: "localhost",
		Port: 8080,
	})

	if !strings.HasSuffix(client.baseURL, "/RPC2") {
		t.Errorf("expected URL to end with '/RPC2', got '%s'", client.baseURL)
	}
}

func TestClient_CustomURLBase(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{
		Host:    "localhost",
		Port:    8080,
		URLBase: "custom/rpc",
	})

	if !strings.HasSuffix(client.baseURL, "/custom/rpc") {
		t.Errorf("expected URL to end with '/custom/rpc', got '%s'", client.baseURL)
	}
}

func TestClient_Add_WithCategory(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(xmlRPCSuccessResponse()))
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{
		Category: "slipstream",
	})

	magnetURL := "magnet:?xt=urn:btih:AABBCCDD&dn=test"
	_, err := client.Add(context.Background(), &types.AddOptions{
		URL: magnetURL,
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if !strings.Contains(string(receivedBody), "d.custom1.set=slipstream") {
		t.Error("expected d.custom1.set=slipstream in request body")
	}
}

func TestClient_Add_WithDownloadDir(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(xmlRPCSuccessResponse()))
	}))
	defer server.Close()

	client := setupTestClient(t, server, &types.ClientConfig{})

	magnetURL := "magnet:?xt=urn:btih:AABBCCDD&dn=test"
	_, err := client.Add(context.Background(), &types.AddOptions{
		URL:         magnetURL,
		DownloadDir: "/custom/downloads",
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if !strings.Contains(string(receivedBody), "d.directory.set=/custom/downloads") {
		t.Error("expected d.directory.set=/custom/downloads in request body")
	}
}

// xmlRPCHandler returns an http.Handler that routes XML-RPC calls to response strings.
func xmlRPCHandler(t *testing.T, responses map[string]string) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		method := extractMethodName(body)
		resp, ok := responses[method]
		if !ok {
			t.Errorf("unexpected XML-RPC method: %s", method)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/xml")
		w.Write([]byte(resp))
	})
}

func extractMethodName(body []byte) string {
	type methodCall struct {
		MethodName string `xml:"methodName"`
	}
	var mc methodCall
	if err := xml.Unmarshal(body, &mc); err != nil {
		return ""
	}
	return mc.MethodName
}

func xmlRPCStringResponse(value string) string {
	return fmt.Sprintf(`<?xml version="1.0"?>
<methodResponse>
<params><param><value><string>%s</string></value></param></params>
</methodResponse>`, value)
}

func xmlRPCSuccessResponse() string {
	return `<?xml version="1.0"?>
<methodResponse>
<params><param><value><int>0</int></value></param></params>
</methodResponse>`
}

func setupTestClient(t *testing.T, server *httptest.Server, baseCfg *types.ClientConfig) *Client {
	t.Helper()

	addr := server.Listener.Addr().(*net.TCPAddr)
	host := addr.IP.String()
	port := addr.Port

	cfg := &types.ClientConfig{
		Host:     host,
		Port:     port,
		UseSSL:   false,
		Username: baseCfg.Username,
		Password: baseCfg.Password,
		APIKey:   baseCfg.APIKey,
		Category: baseCfg.Category,
		URLBase:  "",
	}

	client := NewFromConfig(cfg)
	client.baseURL = fmt.Sprintf("http://%s:%d/RPC2", host, port)

	return client
}

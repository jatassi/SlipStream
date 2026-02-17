package flood

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

func TestClient_Type(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})
	if client.Type() != types.ClientTypeFlood {
		t.Errorf("expected type %s, got %s", types.ClientTypeFlood, client.Type())
	}
}

func TestClient_Protocol(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})
	if client.Protocol() != types.ProtocolTorrent {
		t.Errorf("expected protocol %s, got %s", types.ProtocolTorrent, client.Protocol())
	}
}

func TestClient_Test_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/authenticate":
			w.Header().Set("Set-Cookie", "jwt=test123; Path=/")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"username": "admin"})
		case "/api/client/connection-test":
			json.NewEncoder(w).Encode(map[string]any{"isConnected": true})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Test() failed: %v", err)
	}
}

func TestClient_Test_AuthFailure(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth/authenticate" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	err := client.Test(context.Background())
	if !errors.Is(err, types.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

func TestClient_List(t *testing.T) {
	handler := newFloodHandler(t, func(torrents map[string]any) {
		torrents["ABC123DEF456"] = map[string]any{
			"hash":            "ABC123DEF456",
			"name":            "Test Torrent 1",
			"status":          []string{"downloading"},
			"percentComplete": 45.5,
			"sizeBytes":       float64(1073741824),
			"bytesDone":       float64(488636416),
			"downRate":        float64(1048576),
			"upRate":          float64(524288),
			"eta":             float64(300),
			"directory":       "/downloads",
			"ratio":           0.5,
			"dateAdded":       float64(1609459200),
			"message":         "",
			"tags":            []string{"movies"},
		}
		torrents["789GHI000JKL"] = map[string]any{
			"hash":            "789GHI000JKL",
			"name":            "Test Torrent 2",
			"status":          []string{"seeding", "complete"},
			"percentComplete": float64(100),
			"sizeBytes":       float64(2147483648),
			"bytesDone":       float64(2147483648),
			"downRate":        float64(0),
			"upRate":          float64(262144),
			"eta":             float64(-1),
			"directory":       "/downloads",
			"ratio":           1.5,
			"dateAdded":       float64(1609459300),
			"message":         "",
			"tags":            []string{},
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	found := make(map[string]types.DownloadItem)
	for _, item := range items {
		found[item.ID] = item
	}

	item1, ok := found["abc123def456"]
	if !ok {
		t.Fatal("expected torrent abc123def456 not found")
	}
	if item1.Name != "Test Torrent 1" {
		t.Errorf("expected name 'Test Torrent 1', got %s", item1.Name)
	}
	if item1.Status != types.StatusDownloading {
		t.Errorf("expected status downloading, got %s", item1.Status)
	}
	if item1.Progress != 45.5 {
		t.Errorf("expected progress 45.5, got %f", item1.Progress)
	}
	if item1.Size != 1073741824 {
		t.Errorf("expected size 1073741824, got %d", item1.Size)
	}
	if item1.DownloadedSize != 488636416 {
		t.Errorf("expected downloaded 488636416, got %d", item1.DownloadedSize)
	}
	if item1.DownloadSpeed != 1048576 {
		t.Errorf("expected download speed 1048576, got %d", item1.DownloadSpeed)
	}
	if item1.UploadSpeed != 524288 {
		t.Errorf("expected upload speed 524288, got %d", item1.UploadSpeed)
	}
	if item1.ETA != 300 {
		t.Errorf("expected ETA 300, got %d", item1.ETA)
	}
	if item1.DownloadDir != "/downloads" {
		t.Errorf("expected download dir '/downloads', got %s", item1.DownloadDir)
	}
	if item1.AddedAt != time.Unix(1609459200, 0) {
		t.Errorf("expected addedAt 1609459200, got %v", item1.AddedAt)
	}

	item2, ok := found["789ghi000jkl"]
	if !ok {
		t.Fatal("expected torrent 789ghi000jkl not found")
	}
	if item2.Status != types.StatusSeeding {
		t.Errorf("expected status seeding, got %s", item2.Status)
	}
	if item2.Progress != 100 {
		t.Errorf("expected progress 100, got %f", item2.Progress)
	}
}

func TestClient_List_Empty(t *testing.T) {
	handler := newFloodHandler(t, nil)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestClient_Add_URL(t *testing.T) {
	var capturedBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/authenticate":
			w.Header().Set("Set-Cookie", "jwt=test123; Path=/")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"username": "admin"})
		case "/api/torrents/add-urls":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.WriteHeader(http.StatusOK)
		case "/api/torrents":
			json.NewEncoder(w).Encode(map[string]any{
				"id": 1,
				"torrents": map[string]any{
					"ABC123": map[string]any{
						"hash":            "ABC123",
						"name":            "Test",
						"status":          []string{"downloading"},
						"percentComplete": float64(0),
						"sizeBytes":       float64(100),
						"bytesDone":       float64(0),
						"downRate":        float64(0),
						"upRate":          float64(0),
						"eta":             float64(-1),
						"directory":       "/downloads",
						"ratio":           float64(0),
						"dateAdded":       float64(0),
						"message":         "",
						"tags":            []string{"movies"},
					},
				},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)
	client.config.Category = "movies"

	hash, err := client.Add(context.Background(), &types.AddOptions{
		URL:      "magnet:?xt=urn:btih:abc123",
		Category: "movies",
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if hash != "abc123" {
		t.Errorf("expected hash 'abc123', got %s", hash)
	}

	urls, ok := capturedBody["urls"].([]any)
	if !ok || len(urls) != 1 || urls[0] != "magnet:?xt=urn:btih:abc123" {
		t.Errorf("unexpected urls in request body: %v", capturedBody["urls"])
	}

	start, ok := capturedBody["start"].(bool)
	if !ok || !start {
		t.Errorf("expected start=true, got %v", capturedBody["start"])
	}

	tags, ok := capturedBody["tags"].([]any)
	if !ok || len(tags) != 1 || tags[0] != "movies" {
		t.Errorf("expected tags=[movies], got %v", capturedBody["tags"])
	}
}

func TestClient_Add_File(t *testing.T) {
	var capturedBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/authenticate":
			w.Header().Set("Set-Cookie", "jwt=test123; Path=/")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"username": "admin"})
		case "/api/torrents/add-files":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.WriteHeader(http.StatusOK)
		case "/api/torrents":
			json.NewEncoder(w).Encode(map[string]any{
				"id": 1,
				"torrents": map[string]any{
					"DEF456": map[string]any{
						"hash":            "DEF456",
						"name":            "File Torrent",
						"status":          []string{"downloading"},
						"percentComplete": float64(0),
						"sizeBytes":       float64(200),
						"bytesDone":       float64(0),
						"downRate":        float64(0),
						"upRate":          float64(0),
						"eta":             float64(-1),
						"directory":       "/downloads",
						"ratio":           float64(0),
						"dateAdded":       float64(0),
						"message":         "",
						"tags":            []string{},
					},
				},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	hash, err := client.Add(context.Background(), &types.AddOptions{
		FileContent: []byte("fake torrent data"),
		DownloadDir: "/custom/path",
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if hash != "def456" {
		t.Errorf("expected hash 'def456', got %s", hash)
	}

	files, ok := capturedBody["files"].([]any)
	if !ok || len(files) != 1 {
		t.Errorf("expected 1 file in request, got %v", capturedBody["files"])
	}

	dest, ok := capturedBody["destination"].(string)
	if !ok || dest != "/custom/path" {
		t.Errorf("expected destination '/custom/path', got %v", capturedBody["destination"])
	}
}

func TestClient_Remove(t *testing.T) {
	var capturedBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/authenticate":
			w.Header().Set("Set-Cookie", "jwt=test123; Path=/")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"username": "admin"})
		case "/api/torrents/delete":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	err := client.Remove(context.Background(), "abc123", true)
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	hashes, ok := capturedBody["hashes"].([]any)
	if !ok || len(hashes) != 1 || hashes[0] != "ABC123" {
		t.Errorf("expected hashes=[ABC123], got %v", capturedBody["hashes"])
	}

	deleteData, ok := capturedBody["deleteData"].(bool)
	if !ok || !deleteData {
		t.Errorf("expected deleteData=true, got %v", capturedBody["deleteData"])
	}
}

func TestClient_Pause(t *testing.T) {
	var capturedPath string
	var capturedBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/authenticate":
			w.Header().Set("Set-Cookie", "jwt=test123; Path=/")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"username": "admin"})
		case "/api/torrents/stop":
			capturedPath = r.URL.Path
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	err := client.Pause(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("Pause() failed: %v", err)
	}

	if capturedPath != "/api/torrents/stop" {
		t.Errorf("expected path /api/torrents/stop, got %s", capturedPath)
	}

	hashes, ok := capturedBody["hashes"].([]any)
	if !ok || len(hashes) != 1 || hashes[0] != "ABC123" {
		t.Errorf("expected hashes=[ABC123], got %v", capturedBody["hashes"])
	}
}

func TestClient_Resume(t *testing.T) {
	var capturedPath string
	var capturedBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/authenticate":
			w.Header().Set("Set-Cookie", "jwt=test123; Path=/")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"username": "admin"})
		case "/api/torrents/start":
			capturedPath = r.URL.Path
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &capturedBody)
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	err := client.Resume(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("Resume() failed: %v", err)
	}

	if capturedPath != "/api/torrents/start" {
		t.Errorf("expected path /api/torrents/start, got %s", capturedPath)
	}

	hashes, ok := capturedBody["hashes"].([]any)
	if !ok || len(hashes) != 1 || hashes[0] != "ABC123" {
		t.Errorf("expected hashes=[ABC123], got %v", capturedBody["hashes"])
	}
}

func TestClient_GetDownloadDir(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/authenticate":
			w.Header().Set("Set-Cookie", "jwt=test123; Path=/")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"username": "admin"})
		case "/api/client/settings":
			json.NewEncoder(w).Encode(map[string]any{"directoryDefault": "/data/downloads"})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	dir, err := client.GetDownloadDir(context.Background())
	if err != nil {
		t.Fatalf("GetDownloadDir() failed: %v", err)
	}

	if dir != "/data/downloads" {
		t.Errorf("expected '/data/downloads', got %s", dir)
	}
}

func TestClient_GetTorrentInfo(t *testing.T) {
	handler := newFloodHandler(t, func(torrents map[string]any) {
		torrents["ABC123DEF456"] = map[string]any{
			"hash":            "ABC123DEF456",
			"name":            "Info Torrent",
			"status":          []string{"seeding", "complete"},
			"percentComplete": float64(100),
			"sizeBytes":       float64(1073741824),
			"bytesDone":       float64(1073741824),
			"downRate":        float64(0),
			"upRate":          float64(524288),
			"eta":             float64(-1),
			"directory":       "/downloads",
			"ratio":           2.5,
			"dateAdded":       float64(1609459200),
			"message":         "",
			"tags":            []string{},
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	info, err := client.GetTorrentInfo(context.Background(), "abc123def456")
	if err != nil {
		t.Fatalf("GetTorrentInfo() failed: %v", err)
	}

	if info.InfoHash != "abc123def456" {
		t.Errorf("expected infoHash 'abc123def456', got %s", info.InfoHash)
	}
	if info.Ratio != 2.5 {
		t.Errorf("expected ratio 2.5, got %f", info.Ratio)
	}
	if info.Name != "Info Torrent" {
		t.Errorf("expected name 'Info Torrent', got %s", info.Name)
	}
	if info.Status != types.StatusSeeding {
		t.Errorf("expected status seeding, got %s", info.Status)
	}
}

func TestClient_GetTorrentInfo_NotFound(t *testing.T) {
	handler := newFloodHandler(t, nil)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	_, err := client.GetTorrentInfo(context.Background(), "nonexistent")
	if !errors.Is(err, types.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestClient_SetSeedLimits(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})

	err := client.SetSeedLimits(context.Background(), "abc123", 2.0, time.Hour)
	if !errors.Is(err, types.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestMapStatus(t *testing.T) {
	tests := []struct {
		name     string
		statuses []string
		expected types.Status
	}{
		{"error status", []string{"error", "downloading"}, types.StatusWarning},
		{"checking status", []string{"checking"}, types.StatusQueued},
		{"downloading status", []string{"downloading"}, types.StatusDownloading},
		{"seeding status", []string{"seeding", "complete"}, types.StatusSeeding},
		{"complete status", []string{"complete"}, types.StatusCompleted},
		{"stopped status", []string{"stopped"}, types.StatusPaused},
		{"inactive status", []string{"inactive"}, types.StatusPaused},
		{"unknown status", []string{"some_other_status"}, types.StatusUnknown},
		{"empty status", []string{}, types.StatusUnknown},
		{"error takes priority over downloading", []string{"downloading", "error"}, types.StatusWarning},
		{"checking takes priority over seeding", []string{"seeding", "checking"}, types.StatusQueued},
		{"downloading takes priority over complete", []string{"complete", "downloading"}, types.StatusDownloading},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapStatus(tt.statuses)
			if result != tt.expected {
				t.Errorf("mapStatus(%v) = %s, want %s", tt.statuses, result, tt.expected)
			}
		})
	}
}

func TestClient_SessionReauth(t *testing.T) {
	authCalls := 0
	listCallCount := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/authenticate":
			authCalls++
			w.Header().Set("Set-Cookie", "jwt=test123; Path=/")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"username": "admin"})
		case "/api/torrents":
			listCallCount++
			if listCallCount == 2 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"id":       1,
				"torrents": map[string]any{},
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(t, server)

	_, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("first List() failed: %v", err)
	}

	if authCalls != 0 {
		t.Errorf("expected 0 auth calls after first List, got %d", authCalls)
	}

	_, err = client.List(context.Background())
	if err != nil {
		t.Fatalf("second List() with re-auth failed: %v", err)
	}

	if authCalls != 1 {
		t.Errorf("expected 1 auth call after re-auth, got %d", authCalls)
	}
}

func newFloodHandler(t *testing.T, populateTorrents func(map[string]any)) http.Handler {
	t.Helper()

	torrents := make(map[string]any)
	if populateTorrents != nil {
		populateTorrents(torrents)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/auth/authenticate":
			w.Header().Set("Set-Cookie", "jwt=test123; Path=/")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"username": "admin"})
		case "/api/torrents":
			json.NewEncoder(w).Encode(map[string]any{
				"id":       1,
				"torrents": torrents,
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})
}

func setupTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()

	host, portStr, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to parse server address: %v", err)
	}

	port := server.Listener.Addr().(*net.TCPAddr).Port
	_ = portStr

	return NewFromConfig(&types.ClientConfig{
		Host:     host,
		Port:     port,
		Username: "admin",
		Password: "password",
	})
}

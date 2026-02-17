package qbittorrent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

func TestClient_Type(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{
		Host: "localhost",
		Port: 8080,
	})

	if client.Type() != types.ClientTypeQBittorrent {
		t.Errorf("expected ClientTypeQBittorrent, got %s", client.Type())
	}
}

func TestClient_Protocol(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{
		Host: "localhost",
		Port: 8080,
	})

	if client.Protocol() != types.ProtocolTorrent {
		t.Errorf("expected ProtocolTorrent, got %s", client.Protocol())
	}
}

func TestClient_Test_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/app/version" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("v4.6.2"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	if err := client.Test(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestClient_Test_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/app/version" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	err := client.Test(context.Background())
	if !errors.Is(err, types.ErrAuthFailed) {
		t.Errorf("expected ErrAuthFailed, got %v", err)
	}
}

func TestClient_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/torrents/info" {
			torrents := []qbitTorrent{
				{
					Hash:        "abc123",
					Name:        "Ubuntu 24.04",
					Size:        4294967296,
					Progress:    0.75,
					ETA:         3600,
					State:       "downloading",
					Category:    "linux",
					SavePath:    "/downloads/",
					ContentPath: "/downloads/Ubuntu 24.04/",
					Ratio:       0.5,
					DLSpeed:     1048576,
					UPSpeed:     524288,
					Completed:   3221225472,
				},
				{
					Hash:        "def456",
					Name:        "Debian 12",
					Size:        2147483648,
					Progress:    0.0,
					ETA:         -1,
					State:       "pausedDL",
					Category:    "linux",
					SavePath:    "/downloads/",
					ContentPath: "/downloads/Debian 12/",
					DLSpeed:     0,
					UPSpeed:     0,
					Completed:   0,
				},
				{
					Hash:        "ghi789",
					Name:        "Fedora 40",
					Size:        3221225472,
					Progress:    1.0,
					ETA:         8640000,
					State:       "uploading",
					Category:    "linux",
					SavePath:    "/downloads/",
					ContentPath: "/downloads/Fedora 40/",
					Ratio:       2.5,
					DLSpeed:     0,
					UPSpeed:     1048576,
					Completed:   3221225472,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(torrents)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	if items[0].ID != "abc123" {
		t.Errorf("expected ID 'abc123', got '%s'", items[0].ID)
	}
	if items[0].Status != types.StatusDownloading {
		t.Errorf("expected StatusDownloading, got %s", items[0].Status)
	}
	if items[0].Progress != 75.0 {
		t.Errorf("expected progress 75.0, got %f", items[0].Progress)
	}
	if items[0].Size != 4294967296 {
		t.Errorf("expected size 4294967296, got %d", items[0].Size)
	}
	if items[0].ETA != 3600 {
		t.Errorf("expected ETA 3600, got %d", items[0].ETA)
	}
	if items[0].DownloadSpeed != 1048576 {
		t.Errorf("expected download speed 1048576, got %d", items[0].DownloadSpeed)
	}

	if items[1].Status != types.StatusPaused {
		t.Errorf("expected StatusPaused, got %s", items[1].Status)
	}

	if items[2].Status != types.StatusSeeding {
		t.Errorf("expected StatusSeeding, got %s", items[2].Status)
	}
	if items[2].ETA != -1 {
		t.Errorf("expected ETA -1 (from 8640000), got %d", items[2].ETA)
	}
}

func TestClient_List_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/torrents/info" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if items == nil {
		t.Error("expected non-nil slice, got nil")
	}

	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestClient_Add_URL(t *testing.T) {
	var receivedURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/torrents/add" {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Errorf("failed to parse multipart form: %v", err)
			}
			receivedURL = r.FormValue("urls")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ok."))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	magnetURL := "magnet:?xt=urn:btih:ABC123&dn=test"
	hash, err := client.Add(context.Background(), &types.AddOptions{
		URL: magnetURL,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if hash != "abc123" {
		t.Errorf("expected hash 'abc123', got '%s'", hash)
	}

	if receivedURL != magnetURL {
		t.Errorf("expected URL '%s', got '%s'", magnetURL, receivedURL)
	}
}

func TestClient_Add_FileContent(t *testing.T) {
	var receivedFile bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/torrents/add" {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Errorf("failed to parse multipart form: %v", err)
			}
			file, _, err := r.FormFile("torrents")
			if err == nil {
				receivedFile = true
				file.Close()
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ok."))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	_, err := client.Add(context.Background(), &types.AddOptions{
		FileContent: []byte("fake torrent content"),
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !receivedFile {
		t.Error("expected file to be received")
	}
}

func TestClient_Remove(t *testing.T) {
	var receivedHash string
	var receivedDeleteFiles string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/torrents/delete" {
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			receivedHash = values.Get("hashes")
			receivedDeleteFiles = values.Get("deleteFiles")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	err := client.Remove(context.Background(), "ABC123", true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if receivedHash != "abc123" {
		t.Errorf("expected hash 'abc123', got '%s'", receivedHash)
	}

	if receivedDeleteFiles != "true" {
		t.Errorf("expected deleteFiles 'true', got '%s'", receivedDeleteFiles)
	}
}

func TestClient_Pause(t *testing.T) {
	var receivedHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/torrents/pause" {
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			receivedHash = values.Get("hashes")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	err := client.Pause(context.Background(), "ABC123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if receivedHash != "abc123" {
		t.Errorf("expected hash 'abc123', got '%s'", receivedHash)
	}
}

func TestClient_Resume(t *testing.T) {
	var receivedHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/torrents/resume" {
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			receivedHash = values.Get("hashes")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	err := client.Resume(context.Background(), "ABC123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if receivedHash != "abc123" {
		t.Errorf("expected hash 'abc123', got '%s'", receivedHash)
	}
}

func TestClient_GetDownloadDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/app/preferences" {
			prefs := qbitPreferences{
				SavePath: "/downloads/",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(prefs)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	dir, err := client.GetDownloadDir(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if dir != "/downloads/" {
		t.Errorf("expected '/downloads/', got '%s'", dir)
	}
}

func TestClient_SessionReuse(t *testing.T) {
	var loginCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			loginCount.Add(1)
			http.SetCookie(w, &http.Cookie{
				Name:  "SID",
				Value: "test-session",
				Path:  "/",
			})
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ok."))
			return
		}
		if r.URL.Path == "/api/v2/torrents/info" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{
		Username: "admin",
		Password: "password",
	})

	_, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("first List() failed: %v", err)
	}

	_, err = client.List(context.Background())
	if err != nil {
		t.Fatalf("second List() failed: %v", err)
	}

	if loginCount.Load() != 1 {
		t.Errorf("expected 1 login call, got %d", loginCount.Load())
	}
}

func TestClient_SessionReauth(t *testing.T) {
	var loginCount atomic.Int32
	var listCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/auth/login" {
			loginCount.Add(1)
			http.SetCookie(w, &http.Cookie{
				Name:  "SID",
				Value: "test-session",
				Path:  "/",
			})
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Ok."))
			return
		}
		if r.URL.Path == "/api/v2/torrents/info" {
			count := listCount.Add(1)
			if count == 2 {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("[]"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{
		Username: "admin",
		Password: "password",
	})

	_, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("first List() failed: %v", err)
	}

	_, err = client.List(context.Background())
	if err != nil {
		t.Fatalf("second List() (with reauth) failed: %v", err)
	}

	if loginCount.Load() != 2 {
		t.Errorf("expected 2 login calls, got %d", loginCount.Load())
	}

	if listCount.Load() != 3 {
		t.Errorf("expected 3 list calls (first success, second 403, third retry success), got %d", listCount.Load())
	}
}

func TestMapStatus(t *testing.T) {
	tests := []struct {
		state    string
		expected types.Status
	}{
		{"error", types.StatusWarning},
		{"missingFiles", types.StatusWarning},
		{"pausedDL", types.StatusPaused},
		{"stoppedDL", types.StatusPaused},
		{"queuedDL", types.StatusQueued},
		{"checkingDL", types.StatusQueued},
		{"checkingUP", types.StatusQueued},
		{"checkingResumeData", types.StatusQueued},
		{"pausedUP", types.StatusSeeding},
		{"stoppedUP", types.StatusSeeding},
		{"uploading", types.StatusSeeding},
		{"stalledUP", types.StatusSeeding},
		{"queuedUP", types.StatusSeeding},
		{"forcedUP", types.StatusSeeding},
		{"metaDL", types.StatusQueued},
		{"forcedMetaDL", types.StatusQueued},
		{"forcedDL", types.StatusDownloading},
		{"moving", types.StatusDownloading},
		{"downloading", types.StatusDownloading},
		{"stalledDL", types.StatusWarning},
		{"unknown_state", types.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := mapStatus(tt.state)
			if result != tt.expected {
				t.Errorf("mapStatus(%s) = %s, expected %s", tt.state, result, tt.expected)
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
		{
			name:     "valid magnet",
			magnet:   "magnet:?xt=urn:btih:ABC123&dn=test",
			expected: "abc123",
		},
		{
			name:     "valid magnet uppercase",
			magnet:   "magnet:?xt=urn:btih:DEADBEEF&dn=test",
			expected: "deadbeef",
		},
		{
			name:     "not a magnet",
			magnet:   "http://example.com/file.torrent",
			expected: "",
		},
		{
			name:     "magnet without hash",
			magnet:   "magnet:?dn=test",
			expected: "",
		},
		{
			name:     "magnet without query",
			magnet:   "magnet:",
			expected: "",
		},
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

func TestClient_SetSeedLimits(t *testing.T) {
	var receivedHash string
	var receivedRatioLimit string
	var receivedTimeLimit string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/torrents/setShareLimits" {
			body, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(body))
			receivedHash = values.Get("hashes")
			receivedRatioLimit = values.Get("ratioLimit")
			receivedTimeLimit = values.Get("seedingTimeLimit")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	err := client.SetSeedLimits(context.Background(), "ABC123", 2.0, 120*time.Minute)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if receivedHash != "abc123" {
		t.Errorf("expected hash 'abc123', got '%s'", receivedHash)
	}

	if receivedRatioLimit != "2.00" {
		t.Errorf("expected ratioLimit '2.00', got '%s'", receivedRatioLimit)
	}

	if receivedTimeLimit != "120" {
		t.Errorf("expected seedingTimeLimit '120', got '%s'", receivedTimeLimit)
	}
}

func TestClient_GetTorrentInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/torrents/info" {
			torrents := []qbitTorrent{
				{
					Hash:        "abc123",
					Name:        "Test Torrent",
					Size:        1073741824,
					Progress:    1.0,
					State:       "uploading",
					SavePath:    "/downloads/",
					ContentPath: "/downloads/Test Torrent/",
					Ratio:       1.5,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(torrents)
			return
		}
		if r.URL.Path == "/api/v2/torrents/properties" {
			props := qbitProperties{
				Hash:       "abc123",
				SavePath:   "/downloads/",
				ShareRatio: 1.5,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(props)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := createClientFromServer(t, server, &types.ClientConfig{})

	info, err := client.GetTorrentInfo(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if info.InfoHash != "abc123" {
		t.Errorf("expected InfoHash 'abc123', got '%s'", info.InfoHash)
	}

	if info.Ratio != 1.5 {
		t.Errorf("expected Ratio 1.5, got %f", info.Ratio)
	}

	if info.Name != "Test Torrent" {
		t.Errorf("expected Name 'Test Torrent', got '%s'", info.Name)
	}
}

func createClientFromServer(t *testing.T, server *httptest.Server, baseCfg *types.ClientConfig) *Client {
	t.Helper()

	parsedURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}

	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		if parsedURL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	portInt := 0
	if _, err := fmt.Sscanf(port, "%d", &portInt); err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	cfg := &types.ClientConfig{
		Host:     host,
		Port:     portInt,
		UseSSL:   parsedURL.Scheme == "https",
		Username: baseCfg.Username,
		Password: baseCfg.Password,
		APIKey:   baseCfg.APIKey,
		Category: baseCfg.Category,
		URLBase:  baseCfg.URLBase,
	}

	return NewFromConfig(cfg)
}

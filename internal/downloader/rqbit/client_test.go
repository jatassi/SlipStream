package rqbit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

func configFromTestServer(t *testing.T, serverURL string) *types.ClientConfig {
	t.Helper()
	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}
	port, _ := strconv.Atoi(u.Port())
	return &types.ClientConfig{
		Host: u.Hostname(),
		Port: port,
	}
}

func TestClient_Type(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{Host: "localhost", Port: 3030})
	if got := client.Type(); got != types.ClientTypeRQBit {
		t.Errorf("Type() = %v, want %v", got, types.ClientTypeRQBit)
	}
}

func TestClient_Protocol(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{Host: "localhost", Port: 3030})
	if got := client.Protocol(); got != types.ProtocolTorrent {
		t.Errorf("Protocol() = %v, want %v", got, types.ProtocolTorrent)
	}
}

func TestClient_Test(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version": "8.0.0"}`))
	}))
	defer server.Close()

	client := NewFromConfig(configFromTestServer(t, server.URL))
	if err := client.Test(context.Background()); err != nil {
		t.Errorf("Test() error = %v", err)
	}
}

func TestClient_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/torrents" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("with_stats") != "true" {
			t.Errorf("expected with_stats=true query param")
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"torrents": [
				{
					"id": 1,
					"info_hash": "abc123def456",
					"name": "Test.Torrent.1",
					"output_folder": "/downloads/",
					"stats": {
						"state": 2,
						"error": null,
						"progress_bytes": 500000000,
						"uploaded_bytes": 100000000,
						"total_bytes": 1000000000,
						"finished": false,
						"live": {
							"download_speed": {"mbps": 2.5},
							"upload_speed": {"mbps": 0.1},
							"time_remaining": {"duration": {"secs": 200, "nanos": 0}},
							"snapshot": {
								"downloaded_and_checked_bytes": 500000000,
								"uploaded_bytes": 100000000,
								"peer_stats": {"live": 5, "seen": 10}
							}
						}
					}
				},
				{
					"id": 2,
					"info_hash": "xyz789ghi012",
					"name": "Test.Torrent.2",
					"output_folder": "/downloads/",
					"stats": {
						"state": 2,
						"error": null,
						"progress_bytes": 1000000000,
						"uploaded_bytes": 200000000,
						"total_bytes": 1000000000,
						"finished": true,
						"live": {
							"download_speed": {"mbps": 0.0},
							"upload_speed": {"mbps": 0.5},
							"time_remaining": null,
							"snapshot": {
								"downloaded_and_checked_bytes": 1000000000,
								"uploaded_bytes": 200000000,
								"peer_stats": {"live": 3, "seen": 8}
							}
						}
					}
				},
				{
					"id": 3,
					"info_hash": "paused123",
					"name": "Paused.Torrent",
					"output_folder": "/downloads/",
					"stats": {
						"state": 1,
						"error": null,
						"progress_bytes": 250000000,
						"uploaded_bytes": 50000000,
						"total_bytes": 1000000000,
						"finished": false,
						"live": null
					}
				},
				{
					"id": 4,
					"info_hash": "error123",
					"name": "Error.Torrent",
					"output_folder": "/downloads/",
					"stats": {
						"state": 3,
						"error": "tracker error",
						"progress_bytes": 100000000,
						"uploaded_bytes": 0,
						"total_bytes": 1000000000,
						"finished": false,
						"live": null
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewFromConfig(configFromTestServer(t, server.URL))
	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != 4 {
		t.Fatalf("List() returned %d items, want 4", len(items))
	}

	tests := []struct {
		idx            int
		wantID         string
		wantName       string
		wantStatus     types.Status
		wantProgress   float64
		wantSize       int64
		wantDownloaded int64
		wantDownSpeed  int64
		wantUpSpeed    int64
		wantETA        int64
		wantError      string
	}{
		{
			idx:            0,
			wantID:         "abc123def456",
			wantName:       "Test.Torrent.1",
			wantStatus:     types.StatusDownloading,
			wantProgress:   50.0,
			wantSize:       1000000000,
			wantDownloaded: 500000000,
			wantDownSpeed:  2621440,
			wantUpSpeed:    104857,
			wantETA:        200,
		},
		{
			idx:            1,
			wantID:         "xyz789ghi012",
			wantName:       "Test.Torrent.2",
			wantStatus:     types.StatusSeeding,
			wantProgress:   100.0,
			wantSize:       1000000000,
			wantDownloaded: 1000000000,
			wantDownSpeed:  0,
			wantUpSpeed:    524288,
			wantETA:        -1,
		},
		{
			idx:            2,
			wantID:         "paused123",
			wantName:       "Paused.Torrent",
			wantStatus:     types.StatusPaused,
			wantProgress:   25.0,
			wantSize:       1000000000,
			wantDownloaded: 250000000,
			wantDownSpeed:  0,
			wantUpSpeed:    0,
			wantETA:        -1,
		},
		{
			idx:            3,
			wantID:         "error123",
			wantName:       "Error.Torrent",
			wantStatus:     types.StatusWarning,
			wantProgress:   10.0,
			wantSize:       1000000000,
			wantDownloaded: 100000000,
			wantDownSpeed:  0,
			wantUpSpeed:    0,
			wantETA:        -1,
			wantError:      "tracker error",
		},
	}

	for _, tt := range tests {
		item := items[tt.idx]
		if item.ID != tt.wantID {
			t.Errorf("items[%d].ID = %v, want %v", tt.idx, item.ID, tt.wantID)
		}
		if item.Name != tt.wantName {
			t.Errorf("items[%d].Name = %v, want %v", tt.idx, item.Name, tt.wantName)
		}
		if item.Status != tt.wantStatus {
			t.Errorf("items[%d].Status = %v, want %v", tt.idx, item.Status, tt.wantStatus)
		}
		if item.Progress != tt.wantProgress {
			t.Errorf("items[%d].Progress = %v, want %v", tt.idx, item.Progress, tt.wantProgress)
		}
		if item.Size != tt.wantSize {
			t.Errorf("items[%d].Size = %v, want %v", tt.idx, item.Size, tt.wantSize)
		}
		if item.DownloadedSize != tt.wantDownloaded {
			t.Errorf("items[%d].DownloadedSize = %v, want %v", tt.idx, item.DownloadedSize, tt.wantDownloaded)
		}
		if item.DownloadSpeed != tt.wantDownSpeed {
			t.Errorf("items[%d].DownloadSpeed = %v, want %v", tt.idx, item.DownloadSpeed, tt.wantDownSpeed)
		}
		if item.UploadSpeed != tt.wantUpSpeed {
			t.Errorf("items[%d].UploadSpeed = %v, want %v", tt.idx, item.UploadSpeed, tt.wantUpSpeed)
		}
		if item.ETA != tt.wantETA {
			t.Errorf("items[%d].ETA = %v, want %v", tt.idx, item.ETA, tt.wantETA)
		}
		if item.Error != tt.wantError {
			t.Errorf("items[%d].Error = %v, want %v", tt.idx, item.Error, tt.wantError)
		}
		if item.DownloadDir != "/downloads/" {
			t.Errorf("items[%d].DownloadDir = %v, want /downloads/", tt.idx, item.DownloadDir)
		}
	}
}

func TestClient_List_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"torrents": []}`))
	}))
	defer server.Close()

	client := NewFromConfig(configFromTestServer(t, server.URL))
	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if items == nil {
		t.Error("List() returned nil, want empty slice")
	}
	if len(items) != 0 {
		t.Errorf("List() returned %d items, want 0", len(items))
	}
}

func TestClient_Add_URL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/torrents" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("overwrite") != "true" {
			t.Errorf("expected overwrite=true query param")
		}
		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("expected Content-Type: text/plain, got %s", r.Header.Get("Content-Type"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 1,
			"details": {
				"info_hash": "abc123def456",
				"name": "Test.Torrent"
			},
			"output_folder": "/downloads/"
		}`))
	}))
	defer server.Close()

	client := NewFromConfig(configFromTestServer(t, server.URL))
	id, err := client.Add(context.Background(), &types.AddOptions{
		URL: "magnet:?xt=urn:btih:abc123def456",
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if id != "abc123def456" {
		t.Errorf("Add() returned id %v, want abc123def456", id)
	}
}

func TestClient_Add_FileContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/torrents" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("overwrite") != "true" {
			t.Errorf("expected overwrite=true query param")
		}
		if r.Header.Get("Content-Type") != "application/x-bittorrent" {
			t.Errorf("expected Content-Type: application/x-bittorrent, got %s", r.Header.Get("Content-Type"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": 2,
			"details": {
				"info_hash": "xyz789ghi012",
				"name": "File.Torrent"
			},
			"output_folder": "/downloads/"
		}`))
	}))
	defer server.Close()

	client := NewFromConfig(configFromTestServer(t, server.URL))
	id, err := client.Add(context.Background(), &types.AddOptions{
		FileContent: []byte("fake torrent file content"),
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if id != "xyz789ghi012" {
		t.Errorf("Add() returned id %v, want xyz789ghi012", id)
	}
}

func TestClient_Remove(t *testing.T) {
	tests := []struct {
		name        string
		deleteFiles bool
		wantPath    string
	}{
		{
			name:        "remove without delete",
			deleteFiles: false,
			wantPath:    "/torrents/abc123def456/forget",
		},
		{
			name:        "remove with delete",
			deleteFiles: true,
			wantPath:    "/torrents/abc123def456/delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("unexpected method: %s", r.Method)
				}
				if r.URL.Path != tt.wantPath {
					t.Errorf("unexpected path: %s, want %s", r.URL.Path, tt.wantPath)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := NewFromConfig(configFromTestServer(t, server.URL))
			if err := client.Remove(context.Background(), "abc123def456", tt.deleteFiles); err != nil {
				t.Errorf("Remove() error = %v", err)
			}
		})
	}
}

func TestClient_Pause(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/torrents/abc123def456/pause" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewFromConfig(configFromTestServer(t, server.URL))
	if err := client.Pause(context.Background(), "abc123def456"); err != nil {
		t.Errorf("Pause() error = %v", err)
	}
}

func TestClient_Resume(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/torrents/abc123def456/start" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewFromConfig(configFromTestServer(t, server.URL))
	if err := client.Resume(context.Background(), "abc123def456"); err != nil {
		t.Errorf("Resume() error = %v", err)
	}
}

func TestClient_GetDownloadDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"torrents": [
				{
					"id": 1,
					"info_hash": "abc123def456",
					"name": "Test.Torrent",
					"output_folder": "/custom/download/path",
					"stats": {
						"state": 2,
						"error": null,
						"progress_bytes": 500000000,
						"uploaded_bytes": 100000000,
						"total_bytes": 1000000000,
						"finished": false,
						"live": null
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewFromConfig(configFromTestServer(t, server.URL))
	dir, err := client.GetDownloadDir(context.Background())
	if err != nil {
		t.Fatalf("GetDownloadDir() error = %v", err)
	}

	if dir != "/custom/download/path" {
		t.Errorf("GetDownloadDir() = %v, want /custom/download/path", dir)
	}
}

func TestClient_GetDownloadDir_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"torrents": []}`))
	}))
	defer server.Close()

	client := NewFromConfig(configFromTestServer(t, server.URL))
	dir, err := client.GetDownloadDir(context.Background())
	if err != nil {
		t.Fatalf("GetDownloadDir() error = %v", err)
	}

	if dir != "" {
		t.Errorf("GetDownloadDir() = %v, want empty string", dir)
	}
}

func TestClient_GetTorrentInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"torrents": [
				{
					"id": 1,
					"info_hash": "abc123def456",
					"name": "Test.Torrent",
					"output_folder": "/downloads/",
					"stats": {
						"state": 2,
						"error": null,
						"progress_bytes": 500000000,
						"uploaded_bytes": 200000000,
						"total_bytes": 1000000000,
						"finished": false,
						"live": {
							"download_speed": {"mbps": 2.5},
							"upload_speed": {"mbps": 0.1},
							"time_remaining": {"duration": {"secs": 200, "nanos": 0}},
							"snapshot": {
								"downloaded_and_checked_bytes": 500000000,
								"uploaded_bytes": 200000000,
								"peer_stats": {"live": 5, "seen": 12}
							}
						}
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := NewFromConfig(configFromTestServer(t, server.URL))
	info, err := client.GetTorrentInfo(context.Background(), "abc123def456")
	if err != nil {
		t.Fatalf("GetTorrentInfo() error = %v", err)
	}

	if info.InfoHash != "abc123def456" {
		t.Errorf("InfoHash = %v, want abc123def456", info.InfoHash)
	}
	if info.Seeders != 5 {
		t.Errorf("Seeders = %v, want 5", info.Seeders)
	}
	if info.Leechers != 7 {
		t.Errorf("Leechers = %v, want 7", info.Leechers)
	}
	if info.Ratio != 0.2 {
		t.Errorf("Ratio = %v, want 0.2", info.Ratio)
	}
	if info.IsPrivate {
		t.Errorf("IsPrivate = %v, want false", info.IsPrivate)
	}
}

func TestClient_SetSeedLimits(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{Host: "localhost", Port: 3030})
	err := client.SetSeedLimits(context.Background(), "abc123", 2.0, 0)
	if err != nil {
		t.Errorf("SetSeedLimits() error = %v, want nil (no-op)", err)
	}
}

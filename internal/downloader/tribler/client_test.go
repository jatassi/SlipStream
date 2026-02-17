package tribler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

func TestClient_Type(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})
	if got := client.Type(); got != types.ClientTypeTribler {
		t.Errorf("Type() = %v, want %v", got, types.ClientTypeTribler)
	}
}

func TestClient_Protocol(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})
	if got := client.Protocol(); got != types.ProtocolTorrent {
		t.Errorf("Protocol() = %v, want %v", got, types.ProtocolTorrent)
	}
}

func TestClient_Test(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/settings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		apiKey := r.Header.Get("X-Api-Key")
		if apiKey != "test-api-key" {
			t.Errorf("expected API key header, got: %s", apiKey)
		}

		resp := map[string]interface{}{
			"settings": map[string]interface{}{
				"libtorrent": map[string]interface{}{
					"download_defaults": map[string]interface{}{
						"saveas": "/downloads/",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.URL[7:],
		Port:   80,
		UseSSL: false,
		APIKey: "test-api-key",
	})
	client.baseURL = server.URL + "/"

	if err := client.Test(context.Background()); err != nil {
		t.Errorf("Test() error = %v", err)
	}
}

func TestClient_Test_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.URL[7:],
		Port:   80,
		UseSSL: false,
		APIKey: "wrong-key",
	})
	client.baseURL = server.URL + "/"

	err := client.Test(context.Background())
	if !errors.Is(err, types.ErrAuthFailed) {
		t.Errorf("Test() error = %v, want %v", err, types.ErrAuthFailed)
	}
}

func TestClient_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"downloads": []map[string]interface{}{
				{
					"name":           "Test.Torrent",
					"progress":       0.75,
					"infohash":       "abc123",
					"eta":            120.0,
					"num_seeds":      5,
					"num_peers":      10,
					"all_time_ratio": 0.5,
					"time_added":     1700000000,
					"status":         "DOWNLOADING",
					"error":          "",
					"size":           1000000,
					"destination":    "/downloads/",
					"speed_down":     500000.0,
					"speed_up":       50000.0,
				},
				{
					"name":           "Seeding.Torrent",
					"progress":       1.0,
					"infohash":       "def456",
					"eta":            0.0,
					"num_seeds":      15,
					"num_peers":      2,
					"all_time_ratio": 2.5,
					"time_added":     1700000000,
					"status":         "SEEDING",
					"error":          "",
					"size":           2000000,
					"destination":    "/downloads/",
					"speed_down":     0.0,
					"speed_up":       100000.0,
				},
				{
					"name":           "Metadata.Torrent",
					"progress":       0.0,
					"infohash":       "ghi789",
					"eta":            -1.0,
					"num_seeds":      0,
					"num_peers":      0,
					"all_time_ratio": 0.0,
					"time_added":     1700000000,
					"status":         "METADATA",
					"error":          "",
					"size":           0,
					"destination":    "/downloads/",
					"speed_down":     0.0,
					"speed_up":       0.0,
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{})
	client.baseURL = server.URL + "/"

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != 2 {
		t.Errorf("List() returned %d items, want 2 (should skip size=0)", len(items))
	}

	if items[0].Status != types.StatusDownloading {
		t.Errorf("items[0].Status = %v, want %v", items[0].Status, types.StatusDownloading)
	}

	if items[0].Progress != 75.0 {
		t.Errorf("items[0].Progress = %v, want 75.0", items[0].Progress)
	}

	if items[1].Status != types.StatusSeeding {
		t.Errorf("items[1].Status = %v, want %v", items[1].Status, types.StatusSeeding)
	}
}

func TestClient_List_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"downloads": []map[string]interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{})
	client.baseURL = server.URL + "/"

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != 0 {
		t.Errorf("List() returned %d items, want 0", len(items))
	}
}

func TestClient_Add_URL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		var req addRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req.URI != "magnet:?xt=urn:btih:test" {
			t.Errorf("URI = %s, want magnet:?xt=urn:btih:test", req.URI)
		}

		resp := map[string]interface{}{
			"infohash": "testhash",
			"started":  true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{})
	client.baseURL = server.URL + "/"

	id, err := client.Add(context.Background(), &types.AddOptions{
		URL: "magnet:?xt=urn:btih:test",
	})
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if id != "testhash" {
		t.Errorf("Add() returned id = %s, want testhash", id)
	}
}

func TestClient_Add_FileContent(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})

	_, err := client.Add(context.Background(), &types.AddOptions{
		FileContent: []byte("fake torrent data"),
	})

	if err == nil {
		t.Error("Add() with FileContent should return error")
	}
}

func TestClient_Remove(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		if !strings.Contains(r.URL.Path, "abc123") {
			t.Errorf("path should contain infohash: %s", r.URL.Path)
		}

		var req removeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if !req.RemoveData {
			t.Error("RemoveData should be true")
		}

		resp := map[string]interface{}{
			"removed":  true,
			"infohash": "abc123",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{})
	client.baseURL = server.URL + "/"

	err := client.Remove(context.Background(), "ABC123", true)
	if err != nil {
		t.Errorf("Remove() error = %v", err)
	}
}

func TestClient_Pause(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		var req stateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req.State != "stop" {
			t.Errorf("State = %s, want stop", req.State)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{})
	client.baseURL = server.URL + "/"

	err := client.Pause(context.Background(), "abc123")
	if err != nil {
		t.Errorf("Pause() error = %v", err)
	}
}

func TestClient_Resume(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		var req stateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		if req.State != "resume" {
			t.Errorf("State = %s, want resume", req.State)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{})
	client.baseURL = server.URL + "/"

	err := client.Resume(context.Background(), "abc123")
	if err != nil {
		t.Errorf("Resume() error = %v", err)
	}
}

func TestClient_GetDownloadDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"settings": map[string]interface{}{
				"libtorrent": map[string]interface{}{
					"download_defaults": map[string]interface{}{
						"saveas": "/custom/downloads/",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{})
	client.baseURL = server.URL + "/"

	dir, err := client.GetDownloadDir(context.Background())
	if err != nil {
		t.Fatalf("GetDownloadDir() error = %v", err)
	}

	if dir != "/custom/downloads/" {
		t.Errorf("GetDownloadDir() = %s, want /custom/downloads/", dir)
	}
}

func TestClient_mapStatus(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})

	tests := []struct {
		status   string
		progress float64
		errMsg   string
		want     types.Status
	}{
		{"DOWNLOADING", 0.5, "", types.StatusDownloading},
		{"SEEDING", 1.0, "", types.StatusSeeding},
		{"STOPPED", 1.0, "", types.StatusCompleted},
		{"STOPPED", 0.5, "", types.StatusPaused},
		{"WAITING4HASHCHECK", 0.0, "", types.StatusDownloading},
		{"HASHCHECKING", 0.0, "", types.StatusDownloading},
		{"CIRCUITS", 0.0, "", types.StatusDownloading},
		{"EXIT_NODES", 0.0, "", types.StatusDownloading},
		{"METADATA", 0.0, "", types.StatusQueued},
		{"ALLOCATING_DISKSPACE", 0.0, "", types.StatusQueued},
		{"STOPPED_ON_ERROR", 0.5, "", types.StatusError},
		{"UNKNOWN_STATUS", 0.5, "", types.StatusDownloading},
		{"DOWNLOADING", 0.5, "some error", types.StatusWarning},
	}

	for _, tt := range tests {
		got := client.mapStatus(tt.status, tt.progress, tt.errMsg)
		if got != tt.want {
			t.Errorf("mapStatus(%q, %v, %q) = %v, want %v", tt.status, tt.progress, tt.errMsg, got, tt.want)
		}
	}
}

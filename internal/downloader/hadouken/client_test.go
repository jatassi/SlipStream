package hadouken

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

func TestClient_Type(t *testing.T) {
	c := &Client{}
	if got := c.Type(); got != types.ClientTypeHadouken {
		t.Errorf("Type() = %v, want %v", got, types.ClientTypeHadouken)
	}
}

func TestClient_Protocol(t *testing.T) {
	c := &Client{}
	if got := c.Protocol(); got != types.ProtocolTorrent {
		t.Errorf("Protocol() = %v, want %v", got, types.ProtocolTorrent)
	}
}

func TestClient_Test(t *testing.T) {
	var receivedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "core.getSystemInfo" {
			resp := map[string]any{
				"result": map[string]any{
					"versions": map[string]string{
						"hadouken": "5.1.0",
					},
				},
				"error": nil,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	cfg := &types.ClientConfig{
		Host: strings.TrimPrefix(srv.URL, "http://"),
	}

	parts := strings.Split(cfg.Host, ":")
	if len(parts) == 2 {
		cfg.Host = parts[0]
	}

	c := &Client{
		baseURL:  srv.URL + "/api",
		username: "testuser",
		password: "testpass",
		client:   &http.Client{},
	}

	err := c.Test(context.Background())
	if err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
	if receivedAuth != expectedAuth {
		t.Errorf("Auth header = %v, want %v", receivedAuth, expectedAuth)
	}
}

func TestClient_Test_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := &Client{
		baseURL:  srv.URL + "/api",
		username: "baduser",
		password: "badpass",
		client:   &http.Client{},
	}

	err := c.Test(context.Background())
	if !errors.Is(err, types.ErrAuthFailed) {
		t.Errorf("Test() error = %v, want %v", err, types.ErrAuthFailed)
	}
}

func TestClient_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "webui.list" {
			resp := map[string]any{
				"result": map[string]any{
					"torrents": [][]any{
						{"hash1", 1.0, "Downloading Torrent", 1000000.0, 500.0, 500000.0, 100000.0, nil, nil, 50000.0, nil, "movies", nil, nil, nil, nil, nil, nil, nil, nil, nil, "", nil, nil, nil, nil, nil, "/downloads/path1"},
						{"hash2", 32.0, "Paused Torrent", 2000000.0, 300.0, 600000.0, 0.0, nil, nil, 0.0, nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, "", nil, nil, nil, nil, nil, "/downloads/path2"},
						{"hash3", 0.0, "Seeding Torrent", 3000000.0, 1000.0, 3000000.0, 3500000.0, nil, nil, 0.0, nil, "tv", nil, nil, nil, nil, nil, nil, nil, nil, nil, "", nil, nil, nil, nil, nil, "/downloads/path3"},
						{"hash4", 64.0, "Queued Torrent", 4000000.0, 0.0, 0.0, 0.0, nil, nil, 0.0, nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, "", nil, nil, nil, nil, nil, "/downloads/path4"},
						{"hash5", 1.0, "Error Torrent", 5000000.0, 200.0, 1000000.0, 0.0, nil, nil, 0.0, nil, "", nil, nil, nil, nil, nil, nil, nil, nil, nil, "Disk full", nil, nil, nil, nil, nil, "/downloads/path5"},
					},
				},
				"error": nil,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	c := &Client{
		baseURL:  srv.URL + "/api",
		username: "test",
		password: "test",
		client:   &http.Client{},
	}

	items, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != 5 {
		t.Fatalf("List() returned %d items, want 5", len(items))
	}

	tests := []struct {
		idx          int
		wantID       string
		wantName     string
		wantStatus   types.Status
		wantProgress float64
	}{
		{0, "hash1", "Downloading Torrent", types.StatusDownloading, 50.0},
		{1, "hash2", "Paused Torrent", types.StatusPaused, 30.0},
		{2, "hash3", "Seeding Torrent", types.StatusSeeding, 100.0},
		{3, "hash4", "Queued Torrent", types.StatusQueued, 0.0},
		{4, "hash5", "Error Torrent", types.StatusWarning, 20.0},
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
		if item.ETA != -1 {
			t.Errorf("items[%d].ETA = %v, want -1", tt.idx, item.ETA)
		}
	}
}

func TestClient_List_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "webui.list" {
			resp := map[string]any{
				"result": map[string]any{
					"torrents": [][]any{},
				},
				"error": nil,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	c := &Client{
		baseURL:  srv.URL + "/api",
		username: "test",
		password: "test",
		client:   &http.Client{},
	}

	items, err := c.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != 0 {
		t.Errorf("List() returned %d items, want 0", len(items))
	}
}

func TestClient_Add_URL(t *testing.T) {
	var receivedParams []any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "webui.addTorrent" {
			receivedParams = req["params"].([]any)
			resp := map[string]any{
				"result": "newhash123",
				"error":  nil,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	c := &Client{
		baseURL:  srv.URL + "/api",
		username: "test",
		password: "test",
		client:   &http.Client{},
	}

	opts := &types.AddOptions{
		URL:      "http://example.com/torrent.torrent",
		Category: "movies",
	}

	id, err := c.Add(context.Background(), opts)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if id != "newhash123" {
		t.Errorf("Add() id = %v, want newhash123", id)
	}

	if len(receivedParams) != 3 {
		t.Fatalf("params length = %d, want 3", len(receivedParams))
	}

	if receivedParams[0] != "url" {
		t.Errorf("params[0] = %v, want url", receivedParams[0])
	}
	if receivedParams[1] != opts.URL {
		t.Errorf("params[1] = %v, want %v", receivedParams[1], opts.URL)
	}

	paramsMap, ok := receivedParams[2].(map[string]any)
	if !ok {
		t.Fatalf("params[2] type = %T, want map[string]any", receivedParams[2])
	}
	if paramsMap["label"] != "movies" {
		t.Errorf("params[2][label] = %v, want movies", paramsMap["label"])
	}
}

func TestClient_Add_FileContent(t *testing.T) {
	var receivedParams []any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "webui.addTorrent" {
			receivedParams = req["params"].([]any)
			resp := map[string]any{
				"result": "filehash456",
				"error":  nil,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	c := &Client{
		baseURL:  srv.URL + "/api",
		username: "test",
		password: "test",
		client:   &http.Client{},
	}

	fileContent := []byte("fake torrent data")
	opts := &types.AddOptions{
		FileContent: fileContent,
		Category:    "",
	}

	id, err := c.Add(context.Background(), opts)
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if id != "filehash456" {
		t.Errorf("Add() id = %v, want filehash456", id)
	}

	if len(receivedParams) != 3 {
		t.Fatalf("params length = %d, want 3", len(receivedParams))
	}

	if receivedParams[0] != "file" {
		t.Errorf("params[0] = %v, want file", receivedParams[0])
	}

	expectedEncoded := base64.StdEncoding.EncodeToString(fileContent)
	if receivedParams[1] != expectedEncoded {
		t.Errorf("params[1] not base64 encoded correctly")
	}

	paramsMap, ok := receivedParams[2].(map[string]any)
	if !ok {
		t.Fatalf("params[2] type = %T, want map[string]any", receivedParams[2])
	}
	if len(paramsMap) != 0 {
		t.Errorf("params[2] should be empty map when no category, got %v", paramsMap)
	}
}

func TestClient_Remove(t *testing.T) {
	tests := []struct {
		name        string
		deleteFiles bool
		wantAction  string
	}{
		{"without delete", false, "remove"},
		{"with delete", true, "removedata"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedParams []any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req map[string]any
				json.NewDecoder(r.Body).Decode(&req)

				if req["method"] == "webui.perform" {
					receivedParams = req["params"].([]any)
					resp := map[string]any{
						"result": nil,
						"error":  nil,
					}
					json.NewEncoder(w).Encode(resp)
				}
			}))
			defer srv.Close()

			c := &Client{
				baseURL:  srv.URL + "/api",
				username: "test",
				password: "test",
				client:   &http.Client{},
			}

			err := c.Remove(context.Background(), "hash123", tt.deleteFiles)
			if err != nil {
				t.Fatalf("Remove() error = %v", err)
			}

			if len(receivedParams) != 2 {
				t.Fatalf("params length = %d, want 2", len(receivedParams))
			}

			if receivedParams[0] != tt.wantAction {
				t.Errorf("params[0] = %v, want %v", receivedParams[0], tt.wantAction)
			}

			hashes, ok := receivedParams[1].([]any)
			if !ok {
				t.Fatalf("params[1] type = %T, want []any", receivedParams[1])
			}
			if len(hashes) != 1 || hashes[0] != "hash123" {
				t.Errorf("params[1] = %v, want [hash123]", hashes)
			}
		})
	}
}

func TestClient_Pause(t *testing.T) {
	var receivedParams []any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "webui.perform" {
			receivedParams = req["params"].([]any)
			resp := map[string]any{
				"result": nil,
				"error":  nil,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	c := &Client{
		baseURL:  srv.URL + "/api",
		username: "test",
		password: "test",
		client:   &http.Client{},
	}

	err := c.Pause(context.Background(), "hash456")
	if err != nil {
		t.Fatalf("Pause() error = %v", err)
	}

	if receivedParams[0] != "pause" {
		t.Errorf("params[0] = %v, want pause", receivedParams[0])
	}
}

func TestClient_Resume(t *testing.T) {
	var receivedParams []any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "webui.perform" {
			receivedParams = req["params"].([]any)
			resp := map[string]any{
				"result": nil,
				"error":  nil,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	c := &Client{
		baseURL:  srv.URL + "/api",
		username: "test",
		password: "test",
		client:   &http.Client{},
	}

	err := c.Resume(context.Background(), "hash789")
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}

	if receivedParams[0] != "start" {
		t.Errorf("params[0] = %v, want start", receivedParams[0])
	}
}

func TestClient_GetDownloadDir(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)

		if req["method"] == "webui.getSettings" {
			resp := map[string]any{
				"result": [][]any{
					{"some.other.setting", 2.0, "value1", map[string]string{"access": "Y"}},
					{"bittorrent.defaultSavePath", 2.0, "/downloads/complete", map[string]string{"access": "Y"}},
					{"another.setting", 1.0, "value2", map[string]string{"access": "N"}},
				},
				"error": nil,
			}
			json.NewEncoder(w).Encode(resp)
		}
	}))
	defer srv.Close()

	c := &Client{
		baseURL:  srv.URL + "/api",
		username: "test",
		password: "test",
		client:   &http.Client{},
	}

	dir, err := c.GetDownloadDir(context.Background())
	if err != nil {
		t.Fatalf("GetDownloadDir() error = %v", err)
	}

	if dir != "/downloads/complete" {
		t.Errorf("GetDownloadDir() = %v, want /downloads/complete", dir)
	}
}

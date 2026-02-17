package utorrent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

func makeTestConfig(serverURL string) *types.ClientConfig {
	u, _ := url.Parse(serverURL)
	host := u.Hostname()
	port := 80
	if u.Port() != "" {
		fmt.Sscanf(u.Port(), "%d", &port)
	}

	return &types.ClientConfig{
		Host:     host,
		Port:     port,
		Username: "admin",
		Password: "password",
		URLBase:  "/gui/",
	}
}

func TestClient_Type(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{Host: "localhost", Port: 8080})
	if got := client.Type(); got != types.ClientTypeUTorrent {
		t.Errorf("Type() = %v, want %v", got, types.ClientTypeUTorrent)
	}
}

func TestClient_Protocol(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{Host: "localhost", Port: 8080})
	if got := client.Protocol(); got != types.ProtocolTorrent {
		t.Errorf("Protocol() = %v, want %v", got, types.ProtocolTorrent)
	}
}

func TestClient_Test(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("action") == "getsettings" {
			resp := map[string]any{
				"settings": [][]any{
					{"dir_active_download", 2, "/downloads/", map[string]string{"access": "Y"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	if err := client.Test(context.Background()); err != nil {
		t.Errorf("Test() error = %v", err)
	}
}

func TestClient_Test_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}))
	defer srv.Close()

	cfg := makeTestConfig(srv.URL)
	cfg.Username = "wrong"
	cfg.Password = "creds"
	client := NewFromConfig(cfg)

	err := client.Test(context.Background())
	if !errors.Is(err, types.ErrAuthFailed) {
		t.Errorf("Test() error = %v, want %v", err, types.ErrAuthFailed)
	}
}

func TestClient_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("list") == "1" {
			resp := map[string]any{
				"torrents": [][]any{
					{"HASH1", 137, "Torrent1", 1000000, 1000, 1000000, 500000, 500, 100, 50000, 300, "label", 5, 20, 3, 15, 65536, 1, 0, "", nil, nil, nil, 1700000000, 0, nil, "/downloads/"},
					{"HASH2", 33, "Torrent2", 2000000, 500, 1000000, 200000, 200, 50, 25000, 600, "label2", 3, 10, 2, 8, 32768, 0, 0, "", nil, nil, nil, 1700000100, 0, nil, "/downloads/"},
					{"HASH3", 1, "Torrent3", 3000000, 300, 900000, 100000, 100, 75, 30000, 900, "label3", 7, 15, 4, 12, 16384, 1, 0, "", nil, nil, nil, 1700000200, 0, nil, "/downloads/"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != 3 {
		t.Errorf("List() returned %d items, want 3", len(items))
	}

	if items[0].ID != "HASH1" || items[0].Status != types.StatusSeeding {
		t.Errorf("Item 0: ID=%s Status=%s, want HASH1 seeding", items[0].ID, items[0].Status)
	}

	if items[1].ID != "HASH2" || items[1].Status != types.StatusPaused {
		t.Errorf("Item 1: ID=%s Status=%s, want HASH2 paused", items[1].ID, items[1].Status)
	}

	if items[2].ID != "HASH3" || items[2].Status != types.StatusDownloading {
		t.Errorf("Item 2: ID=%s Status=%s, want HASH3 downloading", items[2].ID, items[2].Status)
	}
}

func TestClient_List_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("list") == "1" {
			resp := map[string]any{
				"torrents": [][]any{},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(items) != 0 {
		t.Errorf("List() returned %d items, want 0", len(items))
	}
}

func TestClient_Add_URL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		token := r.URL.Query().Get("token")
		if token != "TEST_TOKEN" {
			http.Error(w, "invalid token", http.StatusBadRequest)
			return
		}

		if r.URL.Query().Get("action") == "add-url" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.URL.Query().Get("action") == "setprops" {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	magnetURL := "magnet:?xt=urn:btih:ABCD1234&dn=Test"
	hash, err := client.AddMagnet(context.Background(), magnetURL, &types.AddOptions{
		Category: "movies",
	})

	if err != nil {
		t.Fatalf("AddMagnet() error = %v", err)
	}

	if hash != "ABCD1234" {
		t.Errorf("AddMagnet() hash = %s, want ABCD1234", hash)
	}
}

func TestClient_Add_FileContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("action") == "add-file" {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			contentType := r.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "multipart/form-data") {
				http.Error(w, "wrong content type", http.StatusBadRequest)
				return
			}

			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	fileContent := []byte("fake torrent file content")
	_, err := client.Add(context.Background(), &types.AddOptions{
		FileContent: fileContent,
	})

	if err != nil {
		t.Fatalf("Add() with FileContent error = %v", err)
	}
}

func TestClient_Remove(t *testing.T) {
	tests := []struct {
		name        string
		deleteFiles bool
		wantAction  string
	}{
		{"keep files", false, "remove"},
		{"delete files", true, "removedata"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAction := ""
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "token.html") {
					w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
					return
				}

				gotAction = r.URL.Query().Get("action")
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			client := NewFromConfig(makeTestConfig(srv.URL))

			err := client.Remove(context.Background(), "HASH123", tt.deleteFiles)
			if err != nil {
				t.Fatalf("Remove() error = %v", err)
			}

			if gotAction != tt.wantAction {
				t.Errorf("Remove() action = %s, want %s", gotAction, tt.wantAction)
			}
		})
	}
}

func TestClient_Pause(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("action") == "pause" {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	err := client.Pause(context.Background(), "HASH123")
	if err != nil {
		t.Errorf("Pause() error = %v", err)
	}
}

func TestClient_Resume(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("action") == "start" {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	err := client.Resume(context.Background(), "HASH123")
	if err != nil {
		t.Errorf("Resume() error = %v", err)
	}
}

func TestClient_GetDownloadDir(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("action") == "getsettings" {
			resp := map[string]any{
				"settings": [][]any{
					{"some_setting", 1, "value1", nil},
					{"dir_active_download", 2, "/my/download/path", map[string]string{"access": "Y"}},
					{"another_setting", 0, "value2", nil},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	dir, err := client.GetDownloadDir(context.Background())
	if err != nil {
		t.Fatalf("GetDownloadDir() error = %v", err)
	}

	if dir != "/my/download/path" {
		t.Errorf("GetDownloadDir() = %s, want /my/download/path", dir)
	}
}

func TestClient_SessionReuse(t *testing.T) {
	tokenFetchCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			tokenFetchCount++
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("list") == "1" {
			resp := map[string]any{
				"torrents": [][]any{},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	for i := 0; i < 3; i++ {
		_, err := client.List(context.Background())
		if err != nil {
			t.Fatalf("List() call %d error = %v", i+1, err)
		}
	}

	if tokenFetchCount != 1 {
		t.Errorf("token fetched %d times, want 1", tokenFetchCount)
	}
}

func TestClient_SessionReauth(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>NEW_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("list") == "1" {
			requestCount++
			if requestCount == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			resp := map[string]any{
				"torrents": [][]any{},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	_, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v (should have retried after reauth)", err)
	}

	if requestCount != 2 {
		t.Errorf("List() called %d times, want 2 (initial fail + retry)", requestCount)
	}
}

func TestClient_SetSeedLimits(t *testing.T) {
	var gotParams url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("action") == "setprops" {
			gotParams = r.URL.Query()
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	err := client.SetSeedLimits(context.Background(), "HASH123", 1.5, 24*time.Hour)
	if err != nil {
		t.Fatalf("SetSeedLimits() error = %v", err)
	}

	if gotParams.Get("hash") != "HASH123" {
		t.Errorf("hash = %s, want HASH123", gotParams.Get("hash"))
	}

	sValues := gotParams["s"]
	vValues := gotParams["v"]

	expectedS := map[string]bool{"seed_override": false, "seed_ratio": false, "seed_time": false}
	expectedV := map[string]bool{"1": false, "1500": false, "86400": false}

	for _, s := range sValues {
		if _, exists := expectedS[s]; exists {
			expectedS[s] = true
		}
	}

	for _, v := range vValues {
		if _, exists := expectedV[v]; exists {
			expectedV[v] = true
		}
	}

	for k, found := range expectedS {
		if !found {
			t.Errorf("expected s parameter %s not found", k)
		}
	}

	for k, found := range expectedV {
		if !found {
			t.Errorf("expected v parameter %s not found", k)
		}
	}
}

func TestClient_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("list") == "1" {
			resp := map[string]any{
				"torrents": [][]any{
					{"HASH1", 137, "Torrent1", 1000000, 1000, 1000000, 500000, 500, 100, 50000, 300, "label", 5, 20, 3, 15, 65536, 1, 0, "", nil, nil, nil, 1700000000, 0, nil, "/downloads/"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	t.Run("found", func(t *testing.T) {
		item, err := client.Get(context.Background(), "hash1")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if item.ID != "HASH1" {
			t.Errorf("Get() ID = %s, want HASH1", item.ID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := client.Get(context.Background(), "NONEXISTENT")
		if !errors.Is(err, types.ErrNotFound) {
			t.Errorf("Get() error = %v, want %v", err, types.ErrNotFound)
		}
	})
}

func TestClient_GetTorrentInfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("list") == "1" {
			resp := map[string]any{
				"torrents": [][]any{
					{"abcd1234", 137, "Torrent1", 1000000, 1000, 1000000, 500000, 500, 100, 50000, 300, "label", 5, 20, 3, 15, 65536, 1, 0, "", nil, nil, nil, 1700000000, 0, nil, "/downloads/"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	info, err := client.GetTorrentInfo(context.Background(), "abcd1234")
	if err != nil {
		t.Fatalf("GetTorrentInfo() error = %v", err)
	}

	if info.InfoHash != "ABCD1234" {
		t.Errorf("GetTorrentInfo() InfoHash = %s, want ABCD1234", info.InfoHash)
	}

	if info.Name != "Torrent1" {
		t.Errorf("GetTorrentInfo() Name = %s, want Torrent1", info.Name)
	}
}

func TestParseToken(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "standard format",
			html: "<div id='token' style='display:none;'>ABC123</div>",
			want: "ABC123",
		},
		{
			name: "no closing tag",
			html: "<div id='token' style='display:none;'>ABC123",
			want: "",
		},
		{
			name: "no opening tag",
			html: "ABC123</div>",
			want: "",
		},
		{
			name: "empty",
			html: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseToken(tt.html)
			if got != tt.want {
				t.Errorf("parseToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractMagnetHash(t *testing.T) {
	tests := []struct {
		name      string
		magnetURL string
		want      string
	}{
		{
			name:      "standard magnet",
			magnetURL: "magnet:?xt=urn:btih:ABCD1234&dn=Test",
			want:      "ABCD1234",
		},
		{
			name:      "lowercase hash",
			magnetURL: "magnet:?xt=urn:btih:abcd1234&dn=Test",
			want:      "ABCD1234",
		},
		{
			name:      "no btih",
			magnetURL: "magnet:?xt=urn:sha1:ABCD1234",
			want:      "",
		},
		{
			name:      "invalid url",
			magnetURL: "not a url",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMagnetHash(tt.magnetURL)
			if got != tt.want {
				t.Errorf("extractMagnetHash() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMapStatus(t *testing.T) {
	tests := []struct {
		name     string
		flags    int
		progress int
		want     types.Status
	}{
		{"error", 16, 50, types.StatusWarning},
		{"seeding", 137, 100, types.StatusSeeding},
		{"paused", 32, 50, types.StatusPaused},
		{"downloading", 1, 50, types.StatusDownloading},
		{"queued", 64, 30, types.StatusQueued},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapStatus(tt.flags, tt.progress)
			if got != tt.want {
				t.Errorf("mapStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Add_InvalidOptions(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{
		Host:     "localhost",
		Port:     8080,
		Username: "admin",
		Password: "password",
	})

	_, err := client.Add(context.Background(), &types.AddOptions{})
	if err == nil {
		t.Error("Add() with empty options should return error")
	}
}

func TestClient_AddMagnet_InvalidURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	_, err := client.AddMagnet(context.Background(), "not-a-magnet-url", nil)
	if err == nil {
		t.Error("AddMagnet() with invalid URL should return error")
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	blockCh := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blockCh
		w.Write([]byte("response"))
	}))
	defer srv.Close()
	defer close(blockCh)

	client := NewFromConfig(makeTestConfig(srv.URL))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.Test(ctx)
	if err == nil {
		t.Error("Test() with cancelled context should return error")
	}
}

func TestClient_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	_, err := client.List(context.Background())
	if err == nil {
		t.Error("List() with server error should return error")
	}
}

func TestClient_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("list") == "1" {
			w.Write([]byte("not valid json"))
			return
		}
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	_, err := client.List(context.Background())
	if err == nil {
		t.Error("List() with malformed JSON should return error")
	}
}

func TestClient_GetDownloadDir_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			w.Write([]byte("<div id='token' style='display:none;'>TEST_TOKEN</div>"))
			return
		}

		if r.URL.Query().Get("action") == "getsettings" {
			resp := map[string]any{
				"settings": [][]any{
					{"some_other_setting", 1, "value", nil},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	_, err := client.GetDownloadDir(context.Background())
	if err == nil {
		t.Error("GetDownloadDir() should return error when dir_active_download not found")
	}
}

func TestClient_AddFile_ReauthOnFailure(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "token.html") {
			fmt.Fprintf(w, "<div id='token' style='display:none;'>TOKEN_%d</div>", requestCount)
			return
		}

		if r.URL.Query().Get("action") == "add-file" {
			requestCount++
			if requestCount == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusOK)
			return
		}
	}))
	defer srv.Close()

	client := NewFromConfig(makeTestConfig(srv.URL))

	fileContent := []byte("fake torrent file")
	_, err := client.Add(context.Background(), &types.AddOptions{
		FileContent: fileContent,
	})

	if err != nil {
		t.Fatalf("Add() should succeed after reauth, got error: %v", err)
	}

	if requestCount != 2 {
		t.Errorf("add-file called %d times, want 2 (initial fail + retry)", requestCount)
	}
}

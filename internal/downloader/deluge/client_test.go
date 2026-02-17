package deluge

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

func TestClient_Type(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})
	if client.Type() != types.ClientTypeDeluge {
		t.Errorf("expected type %s, got %s", types.ClientTypeDeluge, client.Type())
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
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "auth.login":
			w.Header().Set("Set-Cookie", "session=test123; Path=/")
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.connected":
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "daemon.get_version":
			json.NewEncoder(w).Encode(map[string]any{"result": "2.0.5", "error": nil, "id": req.ID})
		default:
			t.Errorf("unexpected method: %s", req.Method)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	err := client.Test(context.Background())
	if err != nil {
		t.Fatalf("Test() failed: %v", err)
	}
}

func TestClient_Test_AuthFailure(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method == "auth.login" {
			json.NewEncoder(w).Encode(map[string]any{"result": false, "error": nil, "id": req.ID})
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	err := client.Test(context.Background())
	if !errors.Is(err, types.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

func TestClient_List(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "auth.login":
			w.Header().Set("Set-Cookie", "session=test123; Path=/")
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.connected":
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.update_ui":
			result := map[string]any{
				"torrents": map[string]any{
					"abc123": map[string]any{
						"name":        "Test Torrent 1",
						"state":       "Downloading",
						"progress":    45.5,
						"eta":         300.0,
						"is_finished": false,
						"save_path":   "/downloads",
						"total_size":  1073741824.0,
						"total_done":  488636416.0,
						"time_added":  1609459200.0,
						"ratio":       0.5,
					},
					"def456": map[string]any{
						"name":        "Test Torrent 2",
						"state":       "Seeding",
						"progress":    100.0,
						"eta":         0.0,
						"is_finished": true,
						"save_path":   "/downloads",
						"total_size":  2147483648.0,
						"total_done":  2147483648.0,
						"time_added":  1609459300.0,
						"ratio":       1.5,
					},
					"ghi789": map[string]any{
						"name":        "Test Torrent 3",
						"state":       "Paused",
						"progress":    25.0,
						"eta":         0.0,
						"is_finished": false,
						"save_path":   "/downloads",
						"total_size":  524288000.0,
						"total_done":  131072000.0,
						"time_added":  1609459400.0,
						"ratio":       0.0,
					},
				},
				"connected": true,
			}
			json.NewEncoder(w).Encode(map[string]any{"result": result, "error": nil, "id": req.ID})
		default:
			t.Errorf("unexpected method: %s", req.Method)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	for _, item := range items {
		switch item.ID {
		case "abc123":
			if item.Name != "Test Torrent 1" {
				t.Errorf("expected name 'Test Torrent 1', got %s", item.Name)
			}
			if item.Status != types.StatusDownloading {
				t.Errorf("expected status downloading, got %s", item.Status)
			}
			if item.Progress != 45.5 {
				t.Errorf("expected progress 45.5, got %f", item.Progress)
			}
			if item.Size != 1073741824 {
				t.Errorf("expected size 1073741824, got %d", item.Size)
			}
			if item.DownloadedSize != 488636416 {
				t.Errorf("expected downloaded 488636416, got %d", item.DownloadedSize)
			}
			if item.ETA != 300 {
				t.Errorf("expected ETA 300, got %d", item.ETA)
			}
		case "def456":
			if item.Status != types.StatusSeeding {
				t.Errorf("expected status seeding, got %s", item.Status)
			}
			if item.Progress != 100.0 {
				t.Errorf("expected progress 100, got %f", item.Progress)
			}
		case "ghi789":
			if item.Status != types.StatusPaused {
				t.Errorf("expected status paused, got %s", item.Status)
			}
			if item.Progress != 25.0 {
				t.Errorf("expected progress 25, got %f", item.Progress)
			}
		}
	}
}

func TestClient_List_Empty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "auth.login":
			w.Header().Set("Set-Cookie", "session=test123; Path=/")
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.connected":
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.update_ui":
			result := map[string]any{
				"torrents":  map[string]any{},
				"connected": true,
			}
			json.NewEncoder(w).Encode(map[string]any{"result": result, "error": nil, "id": req.ID})
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestClient_Add_URL(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "auth.login":
			w.Header().Set("Set-Cookie", "session=test123; Path=/")
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.connected":
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "core.add_torrent_magnet":
			json.NewEncoder(w).Encode(map[string]any{"result": "abc123def456", "error": nil, "id": req.ID})
		case "label.set_torrent":
			json.NewEncoder(w).Encode(map[string]any{"result": nil, "error": nil, "id": req.ID})
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	hash, err := client.Add(context.Background(), &types.AddOptions{
		URL:      "magnet:?xt=urn:btih:abc123",
		Category: "movies",
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if hash != "abc123def456" {
		t.Errorf("expected hash 'abc123def456', got %s", hash)
	}
}

func TestClient_Add_FileContent(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "auth.login":
			w.Header().Set("Set-Cookie", "session=test123; Path=/")
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.connected":
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "core.add_torrent_file":
			json.NewEncoder(w).Encode(map[string]any{"result": "xyz789abc456", "error": nil, "id": req.ID})
		case "label.set_torrent":
			json.NewEncoder(w).Encode(map[string]any{"result": nil, "error": nil, "id": req.ID})
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	hash, err := client.Add(context.Background(), &types.AddOptions{
		FileContent: []byte("fake torrent data"),
		Category:    "tv",
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if hash != "xyz789abc456" {
		t.Errorf("expected hash 'xyz789abc456', got %s", hash)
	}
}

func TestClient_Remove(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "auth.login":
			w.Header().Set("Set-Cookie", "session=test123; Path=/")
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.connected":
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "core.remove_torrent":
			if len(req.Params) != 2 {
				t.Errorf("expected 2 params, got %d", len(req.Params))
			}
			if hash, ok := req.Params[0].(string); ok && hash != "abc123" {
				t.Errorf("expected hash 'abc123', got %s", hash)
			}
			if deleteFiles, ok := req.Params[1].(bool); ok && !deleteFiles {
				t.Errorf("expected deleteFiles true, got false")
			}
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	err := client.Remove(context.Background(), "abc123", true)
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}
}

func TestClient_Pause(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "auth.login":
			w.Header().Set("Set-Cookie", "session=test123; Path=/")
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.connected":
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "core.pause_torrent":
			if len(req.Params) != 1 {
				t.Errorf("expected 1 param, got %d", len(req.Params))
			}
			json.NewEncoder(w).Encode(map[string]any{"result": nil, "error": nil, "id": req.ID})
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	err := client.Pause(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("Pause() failed: %v", err)
	}
}

func TestClient_Resume(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "auth.login":
			w.Header().Set("Set-Cookie", "session=test123; Path=/")
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.connected":
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "core.resume_torrent":
			if len(req.Params) != 1 {
				t.Errorf("expected 1 param, got %d", len(req.Params))
			}
			json.NewEncoder(w).Encode(map[string]any{"result": nil, "error": nil, "id": req.ID})
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	err := client.Resume(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("Resume() failed: %v", err)
	}
}

func TestClient_GetDownloadDir(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "auth.login":
			w.Header().Set("Set-Cookie", "session=test123; Path=/")
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.connected":
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "core.get_config":
			config := map[string]any{
				"download_location": "/downloads/",
			}
			json.NewEncoder(w).Encode(map[string]any{"result": config, "error": nil, "id": req.ID})
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	dir, err := client.GetDownloadDir(context.Background())
	if err != nil {
		t.Fatalf("GetDownloadDir() failed: %v", err)
	}

	if dir != "/downloads/" {
		t.Errorf("expected '/downloads/', got %s", dir)
	}
}

func TestClient_SessionReuse(t *testing.T) {
	authCalls := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "auth.login":
			authCalls++
			w.Header().Set("Set-Cookie", "session=test123; Path=/")
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.connected":
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.update_ui":
			hasCookie := false
			for _, cookie := range r.Cookies() {
				if cookie.Name == "session" && cookie.Value == "test123" {
					hasCookie = true
					break
				}
			}

			if !hasCookie {
				errObj := map[string]any{
					"message": "Not authenticated",
					"code":    1,
				}
				errData, _ := json.Marshal(errObj)
				errRaw := json.RawMessage(errData)
				json.NewEncoder(w).Encode(map[string]any{"result": nil, "error": &errRaw, "id": req.ID})
			} else {
				result := map[string]any{
					"torrents":  map[string]any{},
					"connected": true,
				}
				json.NewEncoder(w).Encode(map[string]any{"result": result, "error": nil, "id": req.ID})
			}
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	_, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("first List() failed: %v", err)
	}

	_, err = client.List(context.Background())
	if err != nil {
		t.Fatalf("second List() failed: %v", err)
	}

	if authCalls != 1 {
		t.Errorf("expected 1 auth call, got %d", authCalls)
	}
}

func TestClient_SessionReauth(t *testing.T) {
	updateCallCount := 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params []any  `json:"params"`
			ID     int    `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "auth.login":
			w.Header().Set("Set-Cookie", "session=test123; Path=/")
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.connected":
			json.NewEncoder(w).Encode(map[string]any{"result": true, "error": nil, "id": req.ID})
		case "web.update_ui":
			updateCallCount++
			switch updateCallCount {
			case 1:
				result := map[string]any{
					"torrents":  map[string]any{},
					"connected": true,
				}
				json.NewEncoder(w).Encode(map[string]any{"result": result, "error": nil, "id": req.ID})
			case 2:
				errObj := map[string]any{
					"message": "Not authenticated",
					"code":    1,
				}
				errData, _ := json.Marshal(errObj)
				errRaw := json.RawMessage(errData)
				json.NewEncoder(w).Encode(map[string]any{"result": nil, "error": &errRaw, "id": req.ID})
			default:
				result := map[string]any{
					"torrents":  map[string]any{},
					"connected": true,
				}
				json.NewEncoder(w).Encode(map[string]any{"result": result, "error": nil, "id": req.ID})
			}
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server)

	_, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("first List() failed: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	_, err = client.List(context.Background())
	if err != nil {
		t.Fatalf("second List() with re-auth failed: %v", err)
	}
}

func setupTestClient(server *httptest.Server) *Client {
	addr := server.Listener.Addr().String()
	colonIdx := len(addr) - 1
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			colonIdx = i
			break
		}
	}

	host := addr[:colonIdx]
	port := server.Listener.Addr().(*net.TCPAddr).Port

	return NewFromConfig(&types.ClientConfig{
		Host:     host,
		Port:     port,
		Password: "deluge",
	})
}

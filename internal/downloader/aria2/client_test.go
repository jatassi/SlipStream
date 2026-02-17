package aria2

import (
	"context"
	"encoding/base64"
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
	if client.Type() != types.ClientTypeAria2 {
		t.Errorf("expected type %s, got %s", types.ClientTypeAria2, client.Type())
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
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.getVersion" {
			t.Errorf("expected method aria2.getVersion, got %s", req.Method)
		}

		if len(req.Params) != 1 || req.Params[0] != "token:mysecret" {
			t.Errorf("expected token param, got %v", req.Params)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"version":         "1.37.0",
				"enabledFeatures": []string{"BitTorrent"},
			},
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "mysecret")

	err := client.Test(context.Background())
	if err != nil {
		t.Fatalf("Test() failed: %v", err)
	}
}

func TestClient_Test_NoSecret(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.Params) != 0 {
			t.Errorf("expected no params when no secret, got %v", req.Params)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"version": "1.37.0",
			},
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "")

	err := client.Test(context.Background())
	if err != nil {
		t.Fatalf("Test() failed: %v", err)
	}
}

func TestClient_Test_AuthFailure(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		errObj := map[string]any{
			"code":    1,
			"message": "Unauthorized",
		}
		errData, _ := json.Marshal(errObj)
		errRaw := json.RawMessage(errData)

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"error":   &errRaw,
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "wrongsecret")

	err := client.Test(context.Background())
	if !errors.Is(err, types.ErrAuthFailed) {
		t.Fatalf("expected ErrAuthFailed, got %v", err)
	}
}

func TestClient_List(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		switch req.Method {
		case "aria2.tellActive":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": []any{
					map[string]any{
						"gid":             "a1b2c3d4e5f60001",
						"status":          "active",
						"totalLength":     "1073741824",
						"completedLength": "536870912",
						"downloadSpeed":   "1048576",
						"uploadSpeed":     "524288",
						"dir":             "/downloads",
						"bittorrent": map[string]any{
							"info": map[string]any{
								"name": "Test.Movie.2024.1080p",
							},
						},
						"infoHash": "abcdef1234567890abcd",
					},
					map[string]any{
						"gid":             "a1b2c3d4e5f60002",
						"status":          "active",
						"totalLength":     "2147483648",
						"completedLength": "2147483648",
						"downloadSpeed":   "0",
						"uploadSpeed":     "262144",
						"dir":             "/downloads",
						"bittorrent": map[string]any{
							"info": map[string]any{
								"name": "Test.Show.S01.1080p",
							},
						},
						"infoHash": "1234567890abcdef1234",
					},
				},
			})
		case "aria2.tellWaiting":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": []any{
					map[string]any{
						"gid":             "a1b2c3d4e5f60003",
						"status":          "waiting",
						"totalLength":     "524288000",
						"completedLength": "0",
						"downloadSpeed":   "0",
						"uploadSpeed":     "0",
						"dir":             "/downloads",
						"bittorrent": map[string]any{
							"info": map[string]any{
								"name": "Queued.Download",
							},
						},
					},
				},
			})
		case "aria2.tellStopped":
			json.NewEncoder(w).Encode(map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": []any{
					map[string]any{
						"gid":             "a1b2c3d4e5f60004",
						"status":          "complete",
						"totalLength":     "104857600",
						"completedLength": "104857600",
						"downloadSpeed":   "0",
						"uploadSpeed":     "0",
						"dir":             "/downloads",
						"bittorrent": map[string]any{
							"info": map[string]any{
								"name": "Completed.Download",
							},
						},
					},
				},
			})
		default:
			t.Errorf("unexpected method: %s", req.Method)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}

	for _, item := range items {
		switch item.ID {
		case "a1b2c3d4e5f60001":
			if item.Name != "Test.Movie.2024.1080p" {
				t.Errorf("expected name 'Test.Movie.2024.1080p', got %s", item.Name)
			}
			if item.Status != types.StatusDownloading {
				t.Errorf("expected status downloading, got %s", item.Status)
			}
			if item.Size != 1073741824 {
				t.Errorf("expected size 1073741824, got %d", item.Size)
			}
			if item.DownloadedSize != 536870912 {
				t.Errorf("expected downloaded 536870912, got %d", item.DownloadedSize)
			}
			if item.DownloadSpeed != 1048576 {
				t.Errorf("expected download speed 1048576, got %d", item.DownloadSpeed)
			}
			if item.UploadSpeed != 524288 {
				t.Errorf("expected upload speed 524288, got %d", item.UploadSpeed)
			}
			expectedProgress := 50.0
			if item.Progress != expectedProgress {
				t.Errorf("expected progress %f, got %f", expectedProgress, item.Progress)
			}
			if item.ETA <= 0 {
				t.Errorf("expected positive ETA, got %d", item.ETA)
			}
		case "a1b2c3d4e5f60002":
			if item.Status != types.StatusSeeding {
				t.Errorf("expected status seeding, got %s", item.Status)
			}
			if item.Progress != 100.0 {
				t.Errorf("expected progress 100, got %f", item.Progress)
			}
		case "a1b2c3d4e5f60003":
			if item.Status != types.StatusQueued {
				t.Errorf("expected status queued, got %s", item.Status)
			}
		case "a1b2c3d4e5f60004":
			if item.Status != types.StatusCompleted {
				t.Errorf("expected status completed, got %s", item.Status)
			}
		}
	}
}

func TestClient_List_Empty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  []any{},
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

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
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.addUri" {
			t.Errorf("expected method aria2.addUri, got %s", req.Method)
		}

		// params: [token, [uris], {options}]
		if len(req.Params) != 3 {
			t.Errorf("expected 3 params (token, uris, options), got %d", len(req.Params))
		}

		uris, ok := req.Params[1].([]any)
		if !ok || len(uris) != 1 {
			t.Errorf("expected 1 URI, got %v", req.Params[1])
		}

		opts, ok := req.Params[2].(map[string]any)
		if !ok {
			t.Errorf("expected options map, got %T", req.Params[2])
		}
		if opts["dir"] != "/custom/dir" {
			t.Errorf("expected dir '/custom/dir', got %v", opts["dir"])
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "a1b2c3d4e5f60001",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	gid, err := client.Add(context.Background(), &types.AddOptions{
		URL:         "magnet:?xt=urn:btih:abc123",
		DownloadDir: "/custom/dir",
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if gid != "a1b2c3d4e5f60001" {
		t.Errorf("expected GID 'a1b2c3d4e5f60001', got %s", gid)
	}
}

func TestClient_Add_FileContent(t *testing.T) {
	torrentData := []byte("fake torrent file data")
	expectedB64 := base64.StdEncoding.EncodeToString(torrentData)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.addTorrent" {
			t.Errorf("expected method aria2.addTorrent, got %s", req.Method)
		}

		// params: [token, base64, [], {options}]
		if len(req.Params) != 4 {
			t.Errorf("expected 4 params (token, b64, uris, options), got %d", len(req.Params))
		}

		b64, ok := req.Params[1].(string)
		if !ok || b64 != expectedB64 {
			t.Errorf("expected base64 %s, got %v", expectedB64, req.Params[1])
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "b2c3d4e5f6000002",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	gid, err := client.Add(context.Background(), &types.AddOptions{
		FileContent: torrentData,
	})
	if err != nil {
		t.Fatalf("Add() failed: %v", err)
	}

	if gid != "b2c3d4e5f6000002" {
		t.Errorf("expected GID 'b2c3d4e5f6000002', got %s", gid)
	}
}

func TestClient_Remove(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.forceRemove" {
			t.Errorf("expected method aria2.forceRemove, got %s", req.Method)
		}

		if len(req.Params) != 2 || req.Params[1] != "a1b2c3d4e5f60001" {
			t.Errorf("expected GID param, got %v", req.Params)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "a1b2c3d4e5f60001",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	err := client.Remove(context.Background(), "a1b2c3d4e5f60001", true)
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}
}

func TestClient_Pause(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.forcePause" {
			t.Errorf("expected method aria2.forcePause, got %s", req.Method)
		}

		if len(req.Params) != 2 || req.Params[1] != "a1b2c3d4e5f60001" {
			t.Errorf("expected GID param, got %v", req.Params)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "a1b2c3d4e5f60001",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	err := client.Pause(context.Background(), "a1b2c3d4e5f60001")
	if err != nil {
		t.Fatalf("Pause() failed: %v", err)
	}
}

func TestClient_Resume(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.unpause" {
			t.Errorf("expected method aria2.unpause, got %s", req.Method)
		}

		if len(req.Params) != 2 || req.Params[1] != "a1b2c3d4e5f60001" {
			t.Errorf("expected GID param, got %v", req.Params)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "a1b2c3d4e5f60001",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	err := client.Resume(context.Background(), "a1b2c3d4e5f60001")
	if err != nil {
		t.Fatalf("Resume() failed: %v", err)
	}
}

func TestClient_GetDownloadDir(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.getGlobalOption" {
			t.Errorf("expected method aria2.getGlobalOption, got %s", req.Method)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"dir":                      "/data/downloads",
				"max-concurrent-downloads": "5",
			},
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	dir, err := client.GetDownloadDir(context.Background())
	if err != nil {
		t.Fatalf("GetDownloadDir() failed: %v", err)
	}

	if dir != "/data/downloads" {
		t.Errorf("expected '/data/downloads', got %s", dir)
	}
}

func TestClient_MapStatus(t *testing.T) {
	tests := []struct {
		name            string
		aria2Status     string
		totalLength     int64
		completedLength int64
		expected        types.Status
	}{
		{"active downloading", "active", 1000, 500, types.StatusDownloading},
		{"active seeding", "active", 1000, 1000, types.StatusSeeding},
		{"active zero length", "active", 0, 0, types.StatusDownloading},
		{"waiting", "waiting", 1000, 0, types.StatusQueued},
		{"paused", "paused", 1000, 500, types.StatusPaused},
		{"error", "error", 1000, 500, types.StatusError},
		{"complete", "complete", 1000, 1000, types.StatusCompleted},
		{"removed", "removed", 1000, 500, types.StatusUnknown},
		{"unknown status", "bogus", 1000, 500, types.StatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapStatus(tt.aria2Status, tt.totalLength, tt.completedLength)
			if result != tt.expected {
				t.Errorf("mapStatus(%s, %d, %d) = %s, want %s",
					tt.aria2Status, tt.totalLength, tt.completedLength, result, tt.expected)
			}
		})
	}
}

func TestClient_GetTorrentInfo(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.tellStatus" {
			t.Errorf("expected method aria2.tellStatus, got %s", req.Method)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"gid":             "a1b2c3d4e5f60001",
				"status":          "active",
				"totalLength":     "1073741824",
				"completedLength": "536870912",
				"uploadLength":    "268435456",
				"downloadSpeed":   "1048576",
				"uploadSpeed":     "524288",
				"dir":             "/downloads",
				"infoHash":        "abcdef1234567890abcd",
				"bittorrent": map[string]any{
					"info": map[string]any{
						"name": "Test.Torrent",
					},
				},
			},
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	info, err := client.GetTorrentInfo(context.Background(), "a1b2c3d4e5f60001")
	if err != nil {
		t.Fatalf("GetTorrentInfo() failed: %v", err)
	}

	if info.InfoHash != "abcdef1234567890abcd" {
		t.Errorf("expected infoHash 'abcdef1234567890abcd', got %s", info.InfoHash)
	}

	if info.Name != "Test.Torrent" {
		t.Errorf("expected name 'Test.Torrent', got %s", info.Name)
	}

	if info.Status != types.StatusDownloading {
		t.Errorf("expected status downloading, got %s", info.Status)
	}

	expectedRatio := float64(268435456) / float64(1073741824)
	if info.Ratio != expectedRatio {
		t.Errorf("expected ratio %f, got %f", expectedRatio, info.Ratio)
	}
}

func TestClient_SetSeedLimits(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.changeOption" {
			t.Errorf("expected method aria2.changeOption, got %s", req.Method)
		}

		// params: [token, gid, {options}]
		if len(req.Params) != 3 {
			t.Errorf("expected 3 params, got %d", len(req.Params))
		}

		gid, ok := req.Params[1].(string)
		if !ok || gid != "a1b2c3d4e5f60001" {
			t.Errorf("expected GID 'a1b2c3d4e5f60001', got %v", req.Params[1])
		}

		opts, ok := req.Params[2].(map[string]any)
		if !ok {
			t.Fatalf("expected options map, got %T", req.Params[2])
		}

		if opts["seed-ratio"] != "1.5" {
			t.Errorf("expected seed-ratio '1.5', got %v", opts["seed-ratio"])
		}

		if opts["seed-time"] != "60" {
			t.Errorf("expected seed-time '60', got %v", opts["seed-time"])
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "OK",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	err := client.SetSeedLimits(context.Background(), "a1b2c3d4e5f60001", 1.5, 60*time.Minute)
	if err != nil {
		t.Fatalf("SetSeedLimits() failed: %v", err)
	}
}

func TestClient_SetSeedLimits_NoOp(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no RPC call should be made when both ratio and seedTime are zero")
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	err := client.SetSeedLimits(context.Background(), "a1b2c3d4e5f60001", 0, 0)
	if err != nil {
		t.Fatalf("SetSeedLimits() failed: %v", err)
	}
}

func TestClient_AddMagnet(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.addUri" {
			t.Errorf("expected method aria2.addUri, got %s", req.Method)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result":  "c3d4e5f600000003",
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	gid, err := client.AddMagnet(context.Background(), "magnet:?xt=urn:btih:abc123", &types.AddOptions{})
	if err != nil {
		t.Fatalf("AddMagnet() failed: %v", err)
	}

	if gid != "c3d4e5f600000003" {
		t.Errorf("expected GID 'c3d4e5f600000003', got %s", gid)
	}
}

func TestClient_Get(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method != "aria2.tellStatus" {
			t.Errorf("expected method aria2.tellStatus, got %s", req.Method)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"gid":             "a1b2c3d4e5f60001",
				"status":          "paused",
				"totalLength":     "500000",
				"completedLength": "250000",
				"downloadSpeed":   "0",
				"uploadSpeed":     "0",
				"dir":             "/downloads",
				"bittorrent": map[string]any{
					"info": map[string]any{
						"name": "Paused.Torrent",
					},
				},
			},
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	item, err := client.Get(context.Background(), "a1b2c3d4e5f60001")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if item.ID != "a1b2c3d4e5f60001" {
		t.Errorf("expected ID 'a1b2c3d4e5f60001', got %s", item.ID)
	}

	if item.Name != "Paused.Torrent" {
		t.Errorf("expected name 'Paused.Torrent', got %s", item.Name)
	}

	if item.Status != types.StatusPaused {
		t.Errorf("expected status paused, got %s", item.Status)
	}
}

func TestClient_ErrorItem(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		json.NewDecoder(r.Body).Decode(&req)

		json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      req.ID,
			"result": map[string]any{
				"gid":             "a1b2c3d4e5f60001",
				"status":          "error",
				"totalLength":     "1000",
				"completedLength": "500",
				"downloadSpeed":   "0",
				"uploadSpeed":     "0",
				"dir":             "/downloads",
				"errorCode":       "1",
				"errorMessage":    "network problem has occurred",
			},
		})
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client := setupTestClient(server, "secret")

	item, err := client.Get(context.Background(), "a1b2c3d4e5f60001")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if item.Status != types.StatusError {
		t.Errorf("expected status error, got %s", item.Status)
	}

	if item.Error != "network problem has occurred" {
		t.Errorf("expected error message, got %s", item.Error)
	}
}

func TestClient_ExtractName_Fallback(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})

	status := map[string]any{
		"gid":    "abc123",
		"status": "active",
	}
	name := client.extractName(status)
	if name != "abc123" {
		t.Errorf("expected GID fallback 'abc123', got %s", name)
	}

	status2 := map[string]any{
		"status": "active",
	}
	name2 := client.extractName(status2)
	if name2 != "unknown" {
		t.Errorf("expected 'unknown' fallback, got %s", name2)
	}
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

func setupTestClient(server *httptest.Server, apiKey string) *Client {
	host, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port := server.Listener.Addr().(*net.TCPAddr).Port
	_ = portStr

	return NewFromConfig(&types.ClientConfig{
		Host:   host,
		Port:   port,
		APIKey: apiKey,
	})
}

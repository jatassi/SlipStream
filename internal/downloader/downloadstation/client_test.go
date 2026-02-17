package downloadstation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/slipstream/slipstream/internal/downloader/types"
)

func TestType(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})
	if client.Type() != types.ClientTypeDownloadStation {
		t.Errorf("expected type %s, got %s", types.ClientTypeDownloadStation, client.Type())
	}
}

func TestProtocol(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})
	if client.Protocol() != types.ProtocolTorrent {
		t.Errorf("expected protocol %s, got %s", types.ProtocolTorrent, client.Protocol())
	}
}

func TestTest(t *testing.T) {
	authCalled := false
	testCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := r.URL.Query().Get("api")

		if api == "SYNO.API.Auth" {
			authCalled = true
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, authData{SID: "test-sid"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.DownloadStation.Info" {
			testCalled = true
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, configData{DefaultDestination: "/downloads"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.API.Info" {
			resp := apiResponse{
				Success: true,
				Data: mustMarshal(t, map[string]any{
					"SYNO.DownloadStation.Task": map[string]any{
						"maxVersion": 3,
					},
				}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("unexpected API call: %s", api)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "user",
		Password: "pass",
	})

	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if !authCalled {
		t.Error("auth was not called")
	}
	if !testCalled {
		t.Error("test endpoint was not called")
	}
}

func TestTest_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := apiResponse{
			Success: false,
			Error:   &apiError{Code: 105},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "baduser",
		Password: "badpass",
	})

	err := client.Test(context.Background())
	if !errors.Is(err, types.ErrAuthFailed) {
		t.Errorf("expected ErrAuthFailed, got %v", err)
	}
}

func TestList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := r.URL.Query().Get("api")

		if api == "SYNO.API.Auth" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, authData{SID: "test-sid"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.DownloadStation.Task" {
			tasks := []taskData{
				{
					ID:     "dbid_1",
					Title:  "Test.Torrent",
					Size:   1000,
					Status: "downloading",
					Additional: &taskAdditionalData{
						Detail: &taskDetailData{
							Destination: "/downloads/tv",
							URI:         "magnet:?xt=urn:btih:abc123",
						},
						Transfer: &taskTransferData{
							SizeDownloaded: "500",
							SpeedDownload:  "1000",
							SizeUploaded:   "100",
							SpeedUpload:    "50",
						},
					},
				},
				{
					ID:     "dbid_2",
					Title:  "Test2.Torrent",
					Size:   2000,
					Status: "paused",
					Additional: &taskAdditionalData{
						Detail: &taskDetailData{
							Destination: "/downloads/movies",
							URI:         "magnet:?xt=urn:btih:def456",
						},
						Transfer: &taskTransferData{
							SizeDownloaded: "1000",
							SpeedDownload:  "0",
							SizeUploaded:   "200",
							SpeedUpload:    "0",
						},
					},
				},
				{
					ID:     "dbid_3",
					Title:  "Test3.Torrent",
					Size:   3000,
					Status: "seeding",
					Additional: &taskAdditionalData{
						Detail: &taskDetailData{
							Destination: "/downloads/tv",
							URI:         "magnet:?xt=urn:btih:ghi789",
						},
						Transfer: &taskTransferData{
							SizeDownloaded: "3000",
							SpeedDownload:  "0",
							SizeUploaded:   "6000",
							SpeedUpload:    "500",
						},
					},
				},
				{
					ID:     "dbid_4",
					Title:  "Test4.Torrent",
					Size:   4000,
					Status: "finished",
					Additional: &taskAdditionalData{
						Detail: &taskDetailData{
							Destination: "/downloads/movies",
							URI:         "magnet:?xt=urn:btih:jkl012",
						},
						Transfer: &taskTransferData{
							SizeDownloaded: "4000",
							SpeedDownload:  "0",
							SizeUploaded:   "1000",
							SpeedUpload:    "0",
						},
					},
				},
				{
					ID:     "dbid_5",
					Title:  "Test5.Torrent",
					Size:   5000,
					Status: "error",
					Additional: &taskAdditionalData{
						Detail: &taskDetailData{
							Destination: "/downloads/tv",
							URI:         "magnet:?xt=urn:btih:mno345",
						},
						Transfer: &taskTransferData{
							SizeDownloaded: "0",
							SpeedDownload:  "0",
							SizeUploaded:   "0",
							SpeedUpload:    "0",
						},
					},
				},
				{
					ID:     "dbid_6",
					Title:  "Test6.Torrent",
					Size:   6000,
					Status: "waiting",
					Additional: &taskAdditionalData{
						Detail: &taskDetailData{
							Destination: "/downloads/movies",
							URI:         "magnet:?xt=urn:btih:pqr678",
						},
						Transfer: &taskTransferData{
							SizeDownloaded: "0",
							SpeedDownload:  "0",
							SizeUploaded:   "0",
							SpeedUpload:    "0",
						},
					},
				},
			}

			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, listData{Tasks: tasks}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("unexpected API call: %s", api)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "user",
		Password: "pass",
	})

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(items) != 6 {
		t.Fatalf("expected 6 items, got %d", len(items))
	}

	tests := []struct {
		idx            int
		expectedStatus types.Status
		expectedSpeed  int64
		expectedPct    float64
	}{
		{0, types.StatusDownloading, 1000, 50.0},
		{1, types.StatusPaused, 0, 50.0},
		{2, types.StatusSeeding, 0, 100.0},
		{3, types.StatusCompleted, 0, 100.0},
		{4, types.StatusError, 0, 0.0},
		{5, types.StatusQueued, 0, 0.0},
	}

	for _, tt := range tests {
		item := items[tt.idx]
		if item.Status != tt.expectedStatus {
			t.Errorf("item[%d]: expected status %s, got %s", tt.idx, tt.expectedStatus, item.Status)
		}
		if item.DownloadSpeed != tt.expectedSpeed {
			t.Errorf("item[%d]: expected speed %d, got %d", tt.idx, tt.expectedSpeed, item.DownloadSpeed)
		}
		if item.Progress != tt.expectedPct {
			t.Errorf("item[%d]: expected progress %.1f, got %.1f", tt.idx, tt.expectedPct, item.Progress)
		}
	}
}

func TestList_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := r.URL.Query().Get("api")

		if api == "SYNO.API.Auth" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, authData{SID: "test-sid"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.DownloadStation.Task" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, listData{Tasks: []taskData{}}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("unexpected API call: %s", api)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "user",
		Password: "pass",
	})

	items, err := client.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected empty list, got %d items", len(items))
	}
}

func TestAdd_URL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := r.URL.Query().Get("api")

		if api == "SYNO.API.Auth" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, authData{SID: "test-sid"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.DownloadStation.Task" {
			method := r.URL.Query().Get("method")
			if method == "create" {
				uri := r.URL.Query().Get("uri")
				if !strings.HasPrefix(uri, "magnet:") {
					t.Errorf("expected magnet URI, got %s", uri)
				}

				dest := r.URL.Query().Get("destination")
				if dest != "/downloads/tv" {
					t.Errorf("expected destination /downloads/tv, got %s", dest)
				}

				resp := apiResponse{Success: true}
				json.NewEncoder(w).Encode(resp)
				return
			}
		}

		t.Errorf("unexpected API call: %s", api)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "user",
		Password: "pass",
	})

	magnetURL := "magnet:?xt=urn:btih:abc123&dn=Test"
	id, err := client.Add(context.Background(), &types.AddOptions{
		URL:         magnetURL,
		DownloadDir: "/downloads/tv",
	})

	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if id != "abc123" {
		t.Errorf("expected ID abc123, got %s", id)
	}
}

func TestAdd_FileContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := r.URL.Query().Get("api")

		if api == "SYNO.API.Auth" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, authData{SID: "test-sid"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.DownloadStation.Task" && r.URL.Query().Get("method") == "list" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, listData{Tasks: []taskData{{ID: "dbid_99", Title: "uploaded.torrent", Size: 100, Status: "downloading"}}}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.Method == http.MethodPost {
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("ParseMultipartForm failed: %v", err)
			}

			if r.FormValue("api") != "SYNO.DownloadStation.Task" {
				t.Errorf("expected api=SYNO.DownloadStation.Task, got %s", r.FormValue("api"))
			}
			if r.FormValue("method") != "create" {
				t.Errorf("expected method=create, got %s", r.FormValue("method"))
			}
			if r.FormValue("destination") != "/downloads/movies" {
				t.Errorf("expected destination=/downloads/movies, got %s", r.FormValue("destination"))
			}

			file, _, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("FormFile failed: %v", err)
			}
			defer file.Close()

			resp := apiResponse{Success: true}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "user",
		Password: "pass",
	})

	id, err := client.Add(context.Background(), &types.AddOptions{
		FileContent: []byte("fake torrent file"),
		DownloadDir: "/downloads/movies",
	})

	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if id != "dbid_99" {
		t.Errorf("expected ID dbid_99, got %s", id)
	}
}

func TestRemove(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := r.URL.Query().Get("api")

		if api == "SYNO.API.Auth" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, authData{SID: "test-sid"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.DownloadStation.Task" {
			method := r.URL.Query().Get("method")
			if method == "delete" {
				id := r.URL.Query().Get("id")
				if id != "dbid_1" {
					t.Errorf("expected id=dbid_1, got %s", id)
				}

				resp := apiResponse{Success: true}
				json.NewEncoder(w).Encode(resp)
				return
			}
		}

		t.Errorf("unexpected API call: %s", api)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "user",
		Password: "pass",
	})

	if err := client.Remove(context.Background(), "dbid_1", false); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
}

func TestPause(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := r.URL.Query().Get("api")

		if api == "SYNO.API.Auth" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, authData{SID: "test-sid"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.DownloadStation.Task" {
			method := r.URL.Query().Get("method")
			if method == "pause" {
				id := r.URL.Query().Get("id")
				if id != "dbid_1" {
					t.Errorf("expected id=dbid_1, got %s", id)
				}

				resp := apiResponse{Success: true}
				json.NewEncoder(w).Encode(resp)
				return
			}
		}

		t.Errorf("unexpected API call: %s", api)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "user",
		Password: "pass",
	})

	if err := client.Pause(context.Background(), "dbid_1"); err != nil {
		t.Fatalf("Pause failed: %v", err)
	}
}

func TestResume(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := r.URL.Query().Get("api")

		if api == "SYNO.API.Auth" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, authData{SID: "test-sid"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.DownloadStation.Task" {
			method := r.URL.Query().Get("method")
			if method == "resume" {
				id := r.URL.Query().Get("id")
				if id != "dbid_1" {
					t.Errorf("expected id=dbid_1, got %s", id)
				}

				resp := apiResponse{Success: true}
				json.NewEncoder(w).Encode(resp)
				return
			}
		}

		t.Errorf("unexpected API call: %s", api)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "user",
		Password: "pass",
	})

	if err := client.Resume(context.Background(), "dbid_1"); err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
}

func TestGetDownloadDir(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := r.URL.Query().Get("api")

		if api == "SYNO.API.Auth" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, authData{SID: "test-sid"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.DownloadStation.Info" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, configData{DefaultDestination: "/volume1/downloads"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("unexpected API call: %s", api)
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "user",
		Password: "pass",
	})

	dir, err := client.GetDownloadDir(context.Background())
	if err != nil {
		t.Fatalf("GetDownloadDir failed: %v", err)
	}

	if dir != "/volume1/downloads" {
		t.Errorf("expected /volume1/downloads, got %s", dir)
	}
}

func TestSessionReuse(t *testing.T) {
	authCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := r.URL.Query().Get("api")

		if api == "SYNO.API.Auth" {
			authCount++
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, authData{SID: "test-sid"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.DownloadStation.Info" {
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, configData{DefaultDestination: "/downloads"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "user",
		Password: "pass",
	})

	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("First Test failed: %v", err)
	}

	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Second Test failed: %v", err)
	}

	if authCount != 1 {
		t.Errorf("expected 1 auth call, got %d", authCount)
	}
}

func TestSessionReauth(t *testing.T) {
	authCount := 0
	testCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api := r.URL.Query().Get("api")

		if api == "SYNO.API.Auth" {
			authCount++
			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, authData{SID: fmt.Sprintf("test-sid-%d", authCount)}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if api == "SYNO.DownloadStation.Info" {
			testCount++
			if testCount == 1 {
				resp := apiResponse{
					Success: false,
					Error:   &apiError{Code: 105},
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			resp := apiResponse{
				Success: true,
				Data:    mustMarshal(t, configData{DefaultDestination: "/downloads"}),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:     server.Listener.Addr().(*net.TCPAddr).IP.String(),
		Port:     server.Listener.Addr().(*net.TCPAddr).Port,
		Username: "user",
		Password: "pass",
	})

	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	if authCount != 2 {
		t.Errorf("expected 2 auth calls, got %d", authCount)
	}

	if testCount != 2 {
		t.Errorf("expected 2 test calls, got %d", testCount)
	}
}

func mustMarshal(t *testing.T, v interface{}) *json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	raw := json.RawMessage(data)
	return &raw
}

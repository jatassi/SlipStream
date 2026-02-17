package freeboxdownload

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/slipstream/slipstream/internal/downloader/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testChallenge = "test-challenge-string"
const testAPIKey = "test-api-key"

func expectedPassword(challenge, apiKey string) string {
	h := hmac.New(sha1.New, []byte(apiKey))
	h.Write([]byte(challenge))
	return hex.EncodeToString(h.Sum(nil))
}

func TestClient_Type(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})
	assert.Equal(t, types.ClientTypeFreeboxDownload, client.Type())
}

func TestClient_Protocol(t *testing.T) {
	client := NewFromConfig(&types.ClientConfig{})
	assert.Equal(t, types.ProtocolTorrent, client.Protocol())
}

func TestClient_Test(t *testing.T) {
	sessionToken := "test-session-token"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
			})
			return
		}
		if r.URL.Path == "/api/v1/login/session" {
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)

			expectedPwd := expectedPassword(testChallenge, testAPIKey)
			if body["app_id"] != "slipstream" || body["password"] != expectedPwd {
				http.Error(w, "invalid credentials", http.StatusUnauthorized)
				return
			}

			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"session_token":%q}`, sessionToken)),
			})
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.Listener.Addr().String(),
		Port:   80,
		APIKey: testAPIKey,
		UseSSL: false,
	})
	client.baseURL = server.URL + "/api/v1"

	err := client.Test(context.Background())
	require.NoError(t, err)
	assert.Equal(t, sessionToken, client.sessionToken)
}

func TestClient_Test_AuthFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
			})
			return
		}
		if r.URL.Path == "/api/v1/login/session" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.Listener.Addr().String(),
		Port:   80,
		APIKey: "wrong-key",
		UseSSL: false,
	})
	client.baseURL = server.URL + "/api/v1"

	err := client.Test(context.Background())
	assert.Error(t, err)
}

func TestClient_List(t *testing.T) {
	sessionToken := "test-session-token"
	downloadDir := "/downloads/"
	encodedDir := base64.StdEncoding.EncodeToString([]byte(downloadDir))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
			})
			return
		}
		if r.URL.Path == "/api/v1/login/session" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"session_token":%q}`, sessionToken)),
			})
			return
		}
		if r.URL.Path == "/api/v1/downloads/" {
			if r.Header.Get("X-Fbx-App-Auth") != sessionToken {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			items := []downloadItem{
				{
					ID:          42,
					Name:        "Test File",
					Size:        1000,
					RxBytes:     500,
					TxBytes:     100,
					RxRate:      1000,
					TxRate:      50,
					RxPct:       5000,
					Status:      "downloading",
					ETA:         300,
					Error:       "none",
					DownloadDir: encodedDir,
					InfoHash:    "abc123",
					CreatedTS:   1700000000,
					StopRatio:   150,
				},
			}
			result, _ := json.Marshal(items)
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(result),
			})
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.Listener.Addr().String(),
		Port:   80,
		APIKey: testAPIKey,
		UseSSL: false,
	})
	client.baseURL = server.URL + "/api/v1"

	items, err := client.List(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)

	item := items[0]
	assert.Equal(t, "42", item.ID)
	assert.Equal(t, "Test File", item.Name)
	assert.Equal(t, types.StatusDownloading, item.Status)
	assert.Equal(t, 50.0, item.Progress)
	assert.Equal(t, int64(1000), item.Size)
	assert.Equal(t, int64(500), item.DownloadedSize)
	assert.Equal(t, int64(1000), item.DownloadSpeed)
	assert.Equal(t, int64(50), item.UploadSpeed)
	assert.Equal(t, int64(300), item.ETA)
	assert.Equal(t, downloadDir, item.DownloadDir)
	assert.Empty(t, item.Error)
}

func TestClient_List_Empty(t *testing.T) {
	sessionToken := "test-session-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
			})
			return
		}
		if r.URL.Path == "/api/v1/login/session" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"session_token":%q}`, sessionToken)),
			})
			return
		}
		if r.URL.Path == "/api/v1/downloads/" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(`[]`),
			})
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.Listener.Addr().String(),
		Port:   80,
		APIKey: testAPIKey,
		UseSSL: false,
	})
	client.baseURL = server.URL + "/api/v1"

	items, err := client.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestClient_Add_URL(t *testing.T) {
	sessionToken := "test-session-token"
	testURL := "http://example.com/file.torrent"
	downloadDir := "/custom/"
	encodedDir := base64.StdEncoding.EncodeToString([]byte(downloadDir))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
			})
			return
		}
		if r.URL.Path == "/api/v1/login/session" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"session_token":%q}`, sessionToken)),
			})
			return
		}
		if r.URL.Path == "/api/v1/downloads/add" && r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			bodyStr := string(body)

			values, _ := url.ParseQuery(bodyStr)
			assert.Equal(t, testURL, values.Get("download_url"))
			assert.Equal(t, encodedDir, values.Get("download_dir"))

			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(`{"id":43}`),
			})
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.Listener.Addr().String(),
		Port:   80,
		APIKey: testAPIKey,
		UseSSL: false,
	})
	client.baseURL = server.URL + "/api/v1"

	id, err := client.Add(context.Background(), &types.AddOptions{
		URL:         testURL,
		DownloadDir: downloadDir,
	})
	require.NoError(t, err)
	assert.Equal(t, "43", id)
}

func TestClient_Add_FileContent(t *testing.T) {
	sessionToken := "test-session-token"
	fileContent := []byte("fake torrent content")
	fileName := "test.torrent"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
			})
			return
		}
		if r.URL.Path == "/api/v1/login/session" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"session_token":%q}`, sessionToken)),
			})
			return
		}
		if r.URL.Path == "/api/v1/downloads/add" && r.Method == http.MethodPost {
			assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(`{"id":44}`),
			})
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.Listener.Addr().String(),
		Port:   80,
		APIKey: testAPIKey,
		UseSSL: false,
	})
	client.baseURL = server.URL + "/api/v1"

	id, err := client.Add(context.Background(), &types.AddOptions{
		FileContent: fileContent,
		Name:        fileName,
	})
	require.NoError(t, err)
	assert.Equal(t, "44", id)
}

func TestClient_Remove(t *testing.T) {
	sessionToken := "test-session-token"

	tests := []struct {
		name        string
		deleteFiles bool
		expectPath  string
	}{
		{
			name:        "keep files",
			deleteFiles: false,
			expectPath:  "/api/v1/downloads/42",
		},
		{
			name:        "delete files",
			deleteFiles: true,
			expectPath:  "/api/v1/downloads/42/erase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/login" {
					json.NewEncoder(w).Encode(responseEnvelope{
						Success: true,
						Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
					})
					return
				}
				if r.URL.Path == "/api/v1/login/session" {
					json.NewEncoder(w).Encode(responseEnvelope{
						Success: true,
						Result:  json.RawMessage(fmt.Sprintf(`{"session_token":%q}`, sessionToken)),
					})
					return
				}
				if strings.HasPrefix(r.URL.Path, "/api/v1/downloads/") && r.Method == http.MethodDelete {
					assert.Equal(t, tt.expectPath, r.URL.Path)
					json.NewEncoder(w).Encode(responseEnvelope{
						Success: true,
						Result:  json.RawMessage(`{}`),
					})
					return
				}
			}))
			defer server.Close()

			client := NewFromConfig(&types.ClientConfig{
				Host:   server.Listener.Addr().String(),
				Port:   80,
				APIKey: testAPIKey,
				UseSSL: false,
			})
			client.baseURL = server.URL + "/api/v1"

			err := client.Remove(context.Background(), "42", tt.deleteFiles)
			require.NoError(t, err)
		})
	}
}

func TestClient_Pause(t *testing.T) {
	sessionToken := "test-session-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
			})
			return
		}
		if r.URL.Path == "/api/v1/login/session" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"session_token":%q}`, sessionToken)),
			})
			return
		}
		if r.URL.Path == "/api/v1/downloads/42" && r.Method == http.MethodPut {
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "stopped", body["status"])

			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(`{}`),
			})
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.Listener.Addr().String(),
		Port:   80,
		APIKey: testAPIKey,
		UseSSL: false,
	})
	client.baseURL = server.URL + "/api/v1"

	err := client.Pause(context.Background(), "42")
	require.NoError(t, err)
}

func TestClient_Resume(t *testing.T) {
	sessionToken := "test-session-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
			})
			return
		}
		if r.URL.Path == "/api/v1/login/session" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"session_token":%q}`, sessionToken)),
			})
			return
		}
		if r.URL.Path == "/api/v1/downloads/42" && r.Method == http.MethodPut {
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "downloading", body["status"])

			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(`{}`),
			})
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.Listener.Addr().String(),
		Port:   80,
		APIKey: testAPIKey,
		UseSSL: false,
	})
	client.baseURL = server.URL + "/api/v1"

	err := client.Resume(context.Background(), "42")
	require.NoError(t, err)
}

func TestClient_GetDownloadDir(t *testing.T) {
	sessionToken := "test-session-token"
	downloadDir := "/downloads/"
	encodedDir := base64.StdEncoding.EncodeToString([]byte(downloadDir))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
			})
			return
		}
		if r.URL.Path == "/api/v1/login/session" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"session_token":%q}`, sessionToken)),
			})
			return
		}
		if r.URL.Path == "/api/v1/downloads/config/" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"download_dir":%q}`, encodedDir)),
			})
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.Listener.Addr().String(),
		Port:   80,
		APIKey: testAPIKey,
		UseSSL: false,
	})
	client.baseURL = server.URL + "/api/v1"

	dir, err := client.GetDownloadDir(context.Background())
	require.NoError(t, err)
	assert.Equal(t, downloadDir, dir)
}

func TestClient_SessionReuse(t *testing.T) {
	sessionToken := "test-session-token"
	loginCalls := 0
	sessionCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			loginCalls++
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
			})
			return
		}
		if r.URL.Path == "/api/v1/login/session" {
			sessionCalls++
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"session_token":%q}`, sessionToken)),
			})
			return
		}
		if r.URL.Path == "/api/v1/downloads/" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(`[]`),
			})
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.Listener.Addr().String(),
		Port:   80,
		APIKey: testAPIKey,
		UseSSL: false,
	})
	client.baseURL = server.URL + "/api/v1"

	_, err := client.List(context.Background())
	require.NoError(t, err)

	_, err = client.List(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, loginCalls)
	assert.Equal(t, 1, sessionCalls)
}

func TestClient_SessionReauth(t *testing.T) {
	sessionToken1 := "token1"
	sessionToken2 := "token2"
	currentToken := sessionToken1
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/login" {
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"challenge":%q}`, testChallenge)),
			})
			return
		}
		if r.URL.Path == "/api/v1/login/session" {
			if currentToken == sessionToken1 {
				currentToken = sessionToken2
			}
			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"session_token":%q}`, currentToken)),
			})
			return
		}
		if r.URL.Path == "/api/v1/downloads/" {
			requestCount++
			authToken := r.Header.Get("X-Fbx-App-Auth")

			if requestCount == 1 || authToken != currentToken {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			json.NewEncoder(w).Encode(responseEnvelope{
				Success: true,
				Result:  json.RawMessage(`[]`),
			})
			return
		}
	}))
	defer server.Close()

	client := NewFromConfig(&types.ClientConfig{
		Host:   server.Listener.Addr().String(),
		Port:   80,
		APIKey: testAPIKey,
		UseSSL: false,
	})
	client.baseURL = server.URL + "/api/v1"

	client.sessionToken = sessionToken1

	_, err := client.List(context.Background())
	require.NoError(t, err)
	assert.Equal(t, sessionToken2, client.sessionToken)
}

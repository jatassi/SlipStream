package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/slipstream/slipstream/internal/testutil"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	tdb := testutil.NewTestDB(t)
	server := NewServer(tdb.Conn, nil, tdb.Logger)

	cleanup := func() {
		tdb.Close()
	}

	return server, cleanup
}

func TestHealthCheck(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HealthCheck status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("HealthCheck status = %q, want %q", response["status"], "ok")
	}
}

func TestGetStatus(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GetStatus status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, ok := response["version"]; !ok {
		t.Error("GetStatus missing version field")
	}
	if _, ok := response["movieCount"]; !ok {
		t.Error("GetStatus missing movieCount field")
	}
	if _, ok := response["seriesCount"]; !ok {
		t.Error("GetStatus missing seriesCount field")
	}
}

func TestAuthStatus(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("AuthStatus status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &response)

	if response["requiresAuth"] != false {
		t.Error("AuthStatus requiresAuth should be false by default")
	}
}

// Movies API Tests

func TestMoviesAPI_Create(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"title": "The Matrix", "year": 1999, "tmdbId": 603, "monitored": true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/movies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Create movie status = %d, want %d. Body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var movie map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &movie); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if movie["title"] != "The Matrix" {
		t.Errorf("Create movie title = %v, want %q", movie["title"], "The Matrix")
	}
	if movie["id"] == nil || movie["id"].(float64) == 0 {
		t.Error("Create movie should return an ID")
	}
}

func TestMoviesAPI_List(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create some movies first
	movies := []string{
		`{"title": "Movie 1", "year": 2020}`,
		`{"title": "Movie 2", "year": 2021}`,
	}

	for _, body := range movies {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/movies", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.echo.ServeHTTP(rec, req)
	}

	// List movies
	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies", nil)
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("List movies status = %d, want %d", rec.Code, http.StatusOK)
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(list) != 2 {
		t.Errorf("List movies returned %d movies, want 2", len(list))
	}
}

func TestMoviesAPI_Get(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a movie
	body := `{"title": "Test Movie", "year": 2020}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/movies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	var created map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &created)
	id := int(created["id"].(float64))

	// Get the movie
	req = httptest.NewRequest(http.MethodGet, "/api/v1/movies/"+strings.TrimSpace(rec.Body.String()[:10]), nil)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/movies/1", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Get movie status = %d, want %d. Body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var movie map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &movie)

	if int(movie["id"].(float64)) != id {
		t.Errorf("Get movie ID mismatch")
	}
}

func TestMoviesAPI_Get_NotFound(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/movies/99999", nil)
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Get non-existent movie status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMoviesAPI_Update(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a movie
	createBody := `{"title": "Original", "year": 2020}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/movies", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	// Update the movie
	updateBody := `{"title": "Updated"}`
	req = httptest.NewRequest(http.MethodPut, "/api/v1/movies/1", strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Update movie status = %d, want %d. Body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var movie map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &movie)

	if movie["title"] != "Updated" {
		t.Errorf("Update movie title = %v, want %q", movie["title"], "Updated")
	}
}

func TestMoviesAPI_Delete(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a movie
	createBody := `{"title": "To Delete"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/movies", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	// Delete the movie
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/movies/1", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Delete movie status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	// Verify it's gone
	req = httptest.NewRequest(http.MethodGet, "/api/v1/movies/1", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Get deleted movie status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// Series API Tests

func TestSeriesAPI_Create(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"title": "Breaking Bad", "year": 2008, "tvdbId": 81189, "monitored": true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/series", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Create series status = %d, want %d. Body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var series map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &series)

	if series["title"] != "Breaking Bad" {
		t.Errorf("Create series title = %v, want %q", series["title"], "Breaking Bad")
	}
}

func TestSeriesAPI_List(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create some series
	series := []string{
		`{"title": "Series 1"}`,
		`{"title": "Series 2"}`,
	}

	for _, body := range series {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/series", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		server.echo.ServeHTTP(rec, req)
	}

	// List series
	req := httptest.NewRequest(http.MethodGet, "/api/v1/series", nil)
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("List series status = %d, want %d", rec.Code, http.StatusOK)
	}

	var list []map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &list)

	if len(list) != 2 {
		t.Errorf("List series returned %d series, want 2", len(list))
	}
}

func TestSeriesAPI_Get(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a series
	body := `{"title": "Test Series"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/series", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	// Get the series
	req = httptest.NewRequest(http.MethodGet, "/api/v1/series/1", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Get series status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestSeriesAPI_Delete(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a series
	body := `{"title": "To Delete"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/series", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	// Delete the series
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/series/1", nil)
	rec = httptest.NewRecorder()
	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Delete series status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

// Quality Profiles API Tests

func TestQualityProfilesAPI_List(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/qualityprofiles", nil)
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("List quality profiles status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestQualityProfilesAPI_Create(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"name": "HD-1080p", "cutoff": 11, "items": []}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/qualityprofiles", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("Create quality profile status = %d, want %d. Body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

// Root Folders API Tests

func TestRootFoldersAPI_List(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/rootfolders", nil)
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("List root folders status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// Placeholder endpoints tests

func TestPlaceholderEndpoints(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/api/v1/settings", http.StatusOK},
		{http.MethodGet, "/api/v1/indexers", http.StatusOK},
		{http.MethodGet, "/api/v1/downloadclients", http.StatusOK},
		{http.MethodGet, "/api/v1/queue", http.StatusOK},
		{http.MethodGet, "/api/v1/history", http.StatusOK},
		{http.MethodGet, "/api/v1/search", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			server.echo.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, rec.Code, tt.status)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/movies", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	// Should have CORS headers
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("CORS: Missing Access-Control-Allow-Origin header")
	}
}

func TestInvalidJSON(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/movies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Invalid JSON status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMoviesAPI_CreateEmptyTitle(t *testing.T) {
	server, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"title": ""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/movies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	server.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Create movie with empty title status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

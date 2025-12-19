package metadata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/metadata/tmdb"
)

func setupTestHandlers(t *testing.T) (*httptest.Server, *Handlers) {
	// Create mock TMDB server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search/movie":
			json.NewEncoder(w).Encode(tmdb.SearchMoviesResponse{
				Results: []tmdb.MovieResult{
					{ID: 603, Title: "The Matrix", ReleaseDate: "1999-03-30"},
				},
			})
		case r.URL.Path == "/movie/603":
			poster := "/poster.jpg"
			json.NewEncoder(w).Encode(tmdb.MovieDetails{
				ID: 603, Title: "The Matrix", ReleaseDate: "1999-03-30",
				PosterPath: &poster,
			})
		case r.URL.Path == "/search/tv":
			json.NewEncoder(w).Encode(tmdb.SearchTVResponse{
				Results: []tmdb.TVResult{
					{ID: 1396, Name: "Breaking Bad", FirstAirDate: "2008-01-20"},
				},
			})
		case r.URL.Path == "/tv/1396":
			json.NewEncoder(w).Encode(tmdb.TVDetails{
				ID: 1396, Name: "Breaking Bad", FirstAirDate: "2008-01-20", Status: "Ended",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	cfg := config.MetadataConfig{
		TMDB: config.TMDBConfig{
			APIKey:       "test-key",
			BaseURL:      mockServer.URL,
			ImageBaseURL: mockServer.URL,
			Timeout:      5,
		},
	}

	service := NewService(cfg, zerolog.Nop())
	artwork := NewArtworkDownloader(ArtworkConfig{
		BaseDir: t.TempDir(),
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	handlers := NewHandlers(service, artwork)

	return mockServer, handlers
}

func TestHandlers_SearchMovies(t *testing.T) {
	mockServer, handlers := setupTestHandlers(t)
	defer mockServer.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metadata/movie/search?query=Matrix", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handlers.SearchMovies(c); err != nil {
		t.Fatalf("SearchMovies() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	var results []MovieResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Title != "The Matrix" {
		t.Errorf("Title = %q, want %q", results[0].Title, "The Matrix")
	}
}

func TestHandlers_SearchMovies_MissingQuery(t *testing.T) {
	_, handlers := setupTestHandlers(t)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metadata/movie/search", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.SearchMovies(c)
	if err == nil {
		t.Error("Expected error for missing query")
	}

	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("Expected HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", httpErr.Code, http.StatusBadRequest)
	}
}

func TestHandlers_GetMovie(t *testing.T) {
	mockServer, handlers := setupTestHandlers(t)
	defer mockServer.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metadata/movie/603", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("603")

	if err := handlers.GetMovie(c); err != nil {
		t.Fatalf("GetMovie() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	var result MovieResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Title != "The Matrix" {
		t.Errorf("Title = %q, want %q", result.Title, "The Matrix")
	}
}

func TestHandlers_GetMovie_InvalidID(t *testing.T) {
	_, handlers := setupTestHandlers(t)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metadata/movie/invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("invalid")

	err := handlers.GetMovie(c)
	if err == nil {
		t.Error("Expected error for invalid id")
	}

	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("Expected HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", httpErr.Code, http.StatusBadRequest)
	}
}

func TestHandlers_SearchSeries(t *testing.T) {
	mockServer, handlers := setupTestHandlers(t)
	defer mockServer.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metadata/series/search?query=Breaking", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handlers.SearchSeries(c); err != nil {
		t.Fatalf("SearchSeries() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	var results []SeriesResult
	if err := json.Unmarshal(rec.Body.Bytes(), &results); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if results[0].Title != "Breaking Bad" {
		t.Errorf("Title = %q, want %q", results[0].Title, "Breaking Bad")
	}
}

func TestHandlers_GetSeriesByTMDB(t *testing.T) {
	mockServer, handlers := setupTestHandlers(t)
	defer mockServer.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metadata/series/tmdb/1396", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1396")

	if err := handlers.GetSeriesByTMDB(c); err != nil {
		t.Fatalf("GetSeriesByTMDB() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	var result SeriesResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result.Title != "Breaking Bad" {
		t.Errorf("Title = %q, want %q", result.Title, "Breaking Bad")
	}
	if result.TmdbID != 1396 {
		t.Errorf("TmdbID = %d, want %d", result.TmdbID, 1396)
	}
}

func TestHandlers_ClearCache(t *testing.T) {
	mockServer, handlers := setupTestHandlers(t)
	defer mockServer.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/metadata/cache", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handlers.ClearCache(c); err != nil {
		t.Fatalf("ClearCache() error = %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestHandlers_GetStatus(t *testing.T) {
	mockServer, handlers := setupTestHandlers(t)
	defer mockServer.Close()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metadata/status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handlers.GetStatus(c); err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response StatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	// TMDB should be configured in test
	if len(response.Movie) != 1 {
		t.Errorf("Expected 1 movie provider, got %d", len(response.Movie))
	}
	if !response.Movie[0].Configured {
		t.Error("Expected movie TMDB to be configured")
	}

	if len(response.Series) != 2 {
		t.Errorf("Expected 2 series providers, got %d", len(response.Series))
	}
}

func TestHandlers_NoProvidersConfigured(t *testing.T) {
	cfg := config.MetadataConfig{} // No API keys

	service := NewService(cfg, zerolog.Nop())
	artwork := NewArtworkDownloader(ArtworkConfig{
		BaseDir: t.TempDir(),
		Timeout: 5 * time.Second,
	}, zerolog.Nop())

	handlers := NewHandlers(service, artwork)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metadata/movie/search?query=test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handlers.SearchMovies(c)
	if err == nil {
		t.Error("Expected error for no providers configured")
	}

	httpErr, ok := err.(*echo.HTTPError)
	if !ok {
		t.Fatalf("Expected HTTPError, got %T", err)
	}
	if httpErr.Code != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want %d", httpErr.Code, http.StatusServiceUnavailable)
	}
}

func TestHandlers_RegisterRoutes(t *testing.T) {
	mockServer, handlers := setupTestHandlers(t)
	defer mockServer.Close()

	e := echo.New()
	g := e.Group("/api/v1/metadata")

	handlers.RegisterRoutes(g)

	// Verify routes are registered
	routes := e.Routes()
	expectedPaths := []string{
		"/api/v1/metadata/movie/search",
		"/api/v1/metadata/movie/:id",
		"/api/v1/metadata/movie/:id/artwork",
		"/api/v1/metadata/series/search",
		"/api/v1/metadata/series/tmdb/:id",
		"/api/v1/metadata/series/tvdb/:id",
		"/api/v1/metadata/series/:id/artwork",
		"/api/v1/metadata/cache",
		"/api/v1/metadata/status",
	}

	registeredPaths := make(map[string]bool)
	for _, route := range routes {
		registeredPaths[route.Path] = true
	}

	for _, path := range expectedPaths {
		if !registeredPaths[path] {
			t.Errorf("Expected route %s not registered", path)
		}
	}
}

package metadata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/metadata/tmdb"
)

func setupTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/search/movie":
			json.NewEncoder(w).Encode(tmdb.SearchMoviesResponse{
				Results: []tmdb.MovieResult{
					{ID: 603, Title: "The Matrix", ReleaseDate: "1999-03-30"},
				},
			})
		case r.URL.Path == "/movie/603":
			json.NewEncoder(w).Encode(tmdb.MovieDetails{
				ID: 603, Title: "The Matrix", ReleaseDate: "1999-03-30", Runtime: 136,
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
}

func TestService_SearchMovies(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	cfg := config.MetadataConfig{
		TMDB: config.TMDBConfig{
			APIKey:       "test-key",
			BaseURL:      server.URL,
			ImageBaseURL: "https://image.tmdb.org/t/p",
			Timeout:      5,
		},
	}

	svc := NewService(cfg, zerolog.Nop())

	// Test without year filter (year=0)
	results, err := svc.SearchMovies(context.Background(), "Matrix", 0)
	if err != nil {
		t.Fatalf("SearchMovies() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Title != "The Matrix" {
		t.Errorf("Title = %q, want %q", results[0].Title, "The Matrix")
	}
}

func TestService_SearchMovies_Caching(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(tmdb.SearchMoviesResponse{
			Results: []tmdb.MovieResult{
				{ID: 603, Title: "The Matrix"},
			},
		})
	}))
	defer server.Close()

	cfg := config.MetadataConfig{
		TMDB: config.TMDBConfig{
			APIKey:  "test-key",
			BaseURL: server.URL,
			Timeout: 5,
		},
	}

	svc := NewService(cfg, zerolog.Nop())

	// First call
	_, err := svc.SearchMovies(context.Background(), "Matrix", 0)
	if err != nil {
		t.Fatalf("First SearchMovies() error = %v", err)
	}

	// Second call (should be cached)
	_, err = svc.SearchMovies(context.Background(), "Matrix", 0)
	if err != nil {
		t.Fatalf("Second SearchMovies() error = %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 API call, got %d", callCount)
	}
}

func TestService_GetMovie(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	cfg := config.MetadataConfig{
		TMDB: config.TMDBConfig{
			APIKey:  "test-key",
			BaseURL: server.URL,
			Timeout: 5,
		},
	}

	svc := NewService(cfg, zerolog.Nop())

	result, err := svc.GetMovie(context.Background(), 603)
	if err != nil {
		t.Fatalf("GetMovie() error = %v", err)
	}

	if result.Title != "The Matrix" {
		t.Errorf("Title = %q, want %q", result.Title, "The Matrix")
	}
	if result.Runtime != 136 {
		t.Errorf("Runtime = %d, want %d", result.Runtime, 136)
	}
}

func TestService_SearchSeries(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	cfg := config.MetadataConfig{
		TMDB: config.TMDBConfig{
			APIKey:  "test-key",
			BaseURL: server.URL,
			Timeout: 5,
		},
	}

	svc := NewService(cfg, zerolog.Nop())

	results, err := svc.SearchSeries(context.Background(), "Breaking Bad")
	if err != nil {
		t.Fatalf("SearchSeries() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Title != "Breaking Bad" {
		t.Errorf("Title = %q, want %q", results[0].Title, "Breaking Bad")
	}
}

func TestService_GetSeriesByTMDB(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	cfg := config.MetadataConfig{
		TMDB: config.TMDBConfig{
			APIKey:  "test-key",
			BaseURL: server.URL,
			Timeout: 5,
		},
	}

	svc := NewService(cfg, zerolog.Nop())

	result, err := svc.GetSeriesByTMDB(context.Background(), 1396)
	if err != nil {
		t.Fatalf("GetSeriesByTMDB() error = %v", err)
	}

	if result.Title != "Breaking Bad" {
		t.Errorf("Title = %q, want %q", result.Title, "Breaking Bad")
	}
	if result.TmdbID != 1396 {
		t.Errorf("TmdbID = %d, want %d", result.TmdbID, 1396)
	}
}

func TestService_NoProviderConfigured(t *testing.T) {
	cfg := config.MetadataConfig{} // No API keys

	svc := NewService(cfg, zerolog.Nop())

	_, err := svc.SearchMovies(context.Background(), "Matrix", 0)
	if err != ErrNoProvidersConfigured {
		t.Errorf("SearchMovies() error = %v, want %v", err, ErrNoProvidersConfigured)
	}

	_, err = svc.SearchSeries(context.Background(), "Breaking Bad")
	if err != ErrNoProvidersConfigured {
		t.Errorf("SearchSeries() error = %v, want %v", err, ErrNoProvidersConfigured)
	}
}

func TestService_HasProviders(t *testing.T) {
	tests := []struct {
		name            string
		tmdbKey         string
		tvdbKey         string
		wantMovie       bool
		wantSeries      bool
	}{
		{"no providers", "", "", false, false},
		{"tmdb only", "key", "", true, true},
		{"tvdb only", "", "key", false, true},
		{"both", "key", "key", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.MetadataConfig{
				TMDB: config.TMDBConfig{APIKey: tt.tmdbKey},
				TVDB: config.TVDBConfig{APIKey: tt.tvdbKey},
			}
			svc := NewService(cfg, zerolog.Nop())

			if got := svc.HasMovieProvider(); got != tt.wantMovie {
				t.Errorf("HasMovieProvider() = %v, want %v", got, tt.wantMovie)
			}
			if got := svc.HasSeriesProvider(); got != tt.wantSeries {
				t.Errorf("HasSeriesProvider() = %v, want %v", got, tt.wantSeries)
			}
		})
	}
}

func TestService_ClearCache(t *testing.T) {
	server := setupTestServer(t)
	defer server.Close()

	cfg := config.MetadataConfig{
		TMDB: config.TMDBConfig{
			APIKey:  "test-key",
			BaseURL: server.URL,
			Timeout: 5,
		},
	}

	svc := NewService(cfg, zerolog.Nop())

	// Populate cache
	_, _ = svc.SearchMovies(context.Background(), "Matrix", 0)

	// Clear cache
	svc.ClearCache()

	// Verify cache is empty (would need to expose cache for proper test)
	// For now, just verify it doesn't panic
}

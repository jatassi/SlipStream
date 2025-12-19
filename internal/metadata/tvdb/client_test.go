package tvdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
)

func newTestClient(server *httptest.Server) *Client {
	cfg := config.TVDBConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Timeout: 5,
	}
	client := NewClient(cfg, zerolog.Nop())
	// Pre-set a valid token to skip authentication in tests
	client.token = "test-token"
	client.tokenExpiry = time.Now().Add(24 * time.Hour)
	return client
}

func TestClient_Name(t *testing.T) {
	client := NewClient(config.TVDBConfig{}, zerolog.Nop())
	if client.Name() != "tvdb" {
		t.Errorf("Name() = %q, want %q", client.Name(), "tvdb")
	}
}

func TestClient_IsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
		want   bool
	}{
		{"with key", "abc123", true},
		{"without key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(config.TVDBConfig{APIKey: tt.apiKey}, zerolog.Nop())
			if got := client.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_SearchSeries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		query := r.URL.Query().Get("query")
		if query != "Breaking Bad" {
			t.Errorf("unexpected query: %s", query)
		}

		response := SearchResponse{
			Status: "success",
			Data: []SearchResult{
				{
					TvdbID:   "81189",
					Name:     "Breaking Bad",
					Type:     "series",
					Year:     "2008",
					Overview: "A high school chemistry teacher diagnosed with lung cancer.",
					Status:   "Ended",
					ImageURL: "https://artworks.thetvdb.com/poster.jpg",
					RemoteIDs: []RemoteID{
						{ID: "tt0903747", SourceName: "IMDB"},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := newTestClient(server)
	results, err := client.SearchSeries(context.Background(), "Breaking Bad")
	if err != nil {
		t.Fatalf("SearchSeries() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("SearchSeries() returned %d results, want 1", len(results))
	}

	if results[0].Title != "Breaking Bad" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "Breaking Bad")
	}
	if results[0].Year != 2008 {
		t.Errorf("results[0].Year = %d, want %d", results[0].Year, 2008)
	}
	if results[0].TvdbID != 81189 {
		t.Errorf("results[0].TvdbID = %d, want %d", results[0].TvdbID, 81189)
	}
	if results[0].Status != "ended" {
		t.Errorf("results[0].Status = %q, want %q", results[0].Status, "ended")
	}
	if results[0].ImdbID != "tt0903747" {
		t.Errorf("results[0].ImdbID = %q, want %q", results[0].ImdbID, "tt0903747")
	}
}

func TestClient_SearchSeries_NoAPIKey(t *testing.T) {
	client := NewClient(config.TVDBConfig{}, zerolog.Nop())
	_, err := client.SearchSeries(context.Background(), "Breaking Bad")
	if err != ErrAPIKeyMissing {
		t.Errorf("SearchSeries() error = %v, want %v", err, ErrAPIKeyMissing)
	}
}

func TestClient_GetSeries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/series/81189/extended" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := SeriesResponse{
			Status: "success",
			Data: SeriesDetail{
				ID:             81189,
				Name:           "Breaking Bad",
				Year:           "2008",
				Overview:       "A high school chemistry teacher diagnosed with lung cancer.",
				Image:          "https://artworks.thetvdb.com/poster.jpg",
				AverageRuntime: 47,
				Status: SeriesStatus{
					Name: "Ended",
				},
				Genres: []Genre{
					{ID: 1, Name: "Drama"},
					{ID: 2, Name: "Thriller"},
				},
				RemoteIDs: []SeriesRemoteID{
					{ID: "tt0903747", SourceName: "IMDB"},
					{ID: "1396", SourceName: "TheMovieDB.com"},
				},
				Artworks: []Artwork{
					{Type: 1, Image: "https://artworks.thetvdb.com/poster.jpg"},
					{Type: 3, Image: "https://artworks.thetvdb.com/backdrop.jpg"},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := newTestClient(server)
	result, err := client.GetSeries(context.Background(), 81189)
	if err != nil {
		t.Fatalf("GetSeries() error = %v", err)
	}

	if result.Title != "Breaking Bad" {
		t.Errorf("Title = %q, want %q", result.Title, "Breaking Bad")
	}
	if result.Year != 2008 {
		t.Errorf("Year = %d, want %d", result.Year, 2008)
	}
	if result.TvdbID != 81189 {
		t.Errorf("TvdbID = %d, want %d", result.TvdbID, 81189)
	}
	if result.TmdbID != 1396 {
		t.Errorf("TmdbID = %d, want %d", result.TmdbID, 1396)
	}
	if result.ImdbID != "tt0903747" {
		t.Errorf("ImdbID = %q, want %q", result.ImdbID, "tt0903747")
	}
	if result.Status != "ended" {
		t.Errorf("Status = %q, want %q", result.Status, "ended")
	}
	if result.Runtime != 47 {
		t.Errorf("Runtime = %d, want %d", result.Runtime, 47)
	}
	if len(result.Genres) != 2 {
		t.Errorf("Genres = %d, want 2", len(result.Genres))
	}
	if result.BackdropURL != "https://artworks.thetvdb.com/backdrop.jpg" {
		t.Errorf("BackdropURL = %q, want backdrop URL", result.BackdropURL)
	}
}

func TestClient_GetSeries_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Status:  "failure",
			Message: "Not found",
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.GetSeries(context.Background(), 99999999)
	if err != ErrSeriesNotFound {
		t.Errorf("GetSeries() error = %v, want %v", err, ErrSeriesNotFound)
	}
}

func TestClient_SearchMovies_ReturnsEmpty(t *testing.T) {
	client := NewClient(config.TVDBConfig{APIKey: "test"}, zerolog.Nop())
	results, err := client.SearchMovies(context.Background(), "Matrix")
	if err != nil {
		t.Errorf("SearchMovies() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchMovies() returned %d results, want 0", len(results))
	}
}

func TestClient_GetMovie_ReturnsError(t *testing.T) {
	client := NewClient(config.TVDBConfig{APIKey: "test"}, zerolog.Nop())
	_, err := client.GetMovie(context.Background(), 123)
	if err == nil {
		t.Error("GetMovie() should return error for TVDB")
	}
}

func TestClient_Authenticate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			response := LoginResponse{
				Status: "success",
			}
			response.Data.Token = "new-test-token"
			json.NewEncoder(w).Encode(response)
			return
		}

		// Check token is sent
		auth := r.Header.Get("Authorization")
		if auth != "Bearer new-test-token" {
			t.Errorf("Authorization = %q, want Bearer token", auth)
		}

		response := SearchResponse{
			Status: "success",
			Data:   []SearchResult{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := config.TVDBConfig{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Timeout: 5,
	}
	client := NewClient(cfg, zerolog.Nop())

	// First call should trigger authentication
	_, err := client.SearchSeries(context.Background(), "test")
	if err != nil {
		t.Fatalf("SearchSeries() error = %v", err)
	}

	// Check token was stored
	if client.token != "new-test-token" {
		t.Errorf("token = %q, want %q", client.token, "new-test-token")
	}
}

func TestClient_AuthenticationFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{
			Status:  "failure",
			Message: "Invalid API key",
		})
	}))
	defer server.Close()

	cfg := config.TVDBConfig{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
		Timeout: 5,
	}
	client := NewClient(cfg, zerolog.Nop())

	_, err := client.SearchSeries(context.Background(), "test")
	if err != ErrAuthFailed {
		t.Errorf("SearchSeries() error = %v, want %v", err, ErrAuthFailed)
	}
}

func TestClient_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.SearchSeries(context.Background(), "test")
	if err != ErrRateLimited {
		t.Errorf("SearchSeries() error = %v, want %v", err, ErrRateLimited)
	}
}

func TestClient_StatusMapping(t *testing.T) {
	tests := []struct {
		tvdbStatus string
		wantStatus string
	}{
		{"Ended", "ended"},
		{"Upcoming", "upcoming"},
		{"Continuing", "continuing"},
		{"", "continuing"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.tvdbStatus, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(SeriesResponse{
					Status: "success",
					Data: SeriesDetail{
						ID:   1,
						Name: "Test",
						Status: SeriesStatus{
							Name: tt.tvdbStatus,
						},
					},
				})
			}))
			defer server.Close()

			client := newTestClient(server)
			result, err := client.GetSeries(context.Background(), 1)
			if err != nil {
				t.Fatalf("GetSeries() error = %v", err)
			}

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", result.Status, tt.wantStatus)
			}
		})
	}
}

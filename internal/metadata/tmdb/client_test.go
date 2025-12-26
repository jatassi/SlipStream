package tmdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/config"
)

func newTestClient(server *httptest.Server) *Client {
	cfg := config.TMDBConfig{
		APIKey:       "test-api-key",
		BaseURL:      server.URL,
		ImageBaseURL: "https://image.tmdb.org/t/p",
		Timeout:      5,
	}
	return NewClient(cfg, zerolog.Nop())
}

func TestClient_Name(t *testing.T) {
	client := NewClient(config.TMDBConfig{}, zerolog.Nop())
	if client.Name() != "tmdb" {
		t.Errorf("Name() = %q, want %q", client.Name(), "tmdb")
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
			client := NewClient(config.TMDBConfig{APIKey: tt.apiKey}, zerolog.Nop())
			if got := client.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_SearchMovies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/movie" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		query := r.URL.Query().Get("query")
		if query != "Matrix" {
			t.Errorf("unexpected query: %s", query)
		}

		response := SearchMoviesResponse{
			Page:         1,
			TotalResults: 2,
			TotalPages:   1,
			Results: []MovieResult{
				{
					ID:          603,
					Title:       "The Matrix",
					Overview:    "A computer hacker learns about the true nature of reality.",
					ReleaseDate: "1999-03-30",
				},
				{
					ID:          604,
					Title:       "The Matrix Reloaded",
					Overview:    "Neo and the rebel leaders continue to fight.",
					ReleaseDate: "2003-05-15",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := newTestClient(server)
	// Test without year (year=0 means no filter)
	results, err := client.SearchMovies(context.Background(), "Matrix", 0)
	if err != nil {
		t.Fatalf("SearchMovies() error = %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("SearchMovies() returned %d results, want 2", len(results))
	}

	if results[0].Title != "The Matrix" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "The Matrix")
	}
	if results[0].Year != 1999 {
		t.Errorf("results[0].Year = %d, want %d", results[0].Year, 1999)
	}
	if results[0].ID != 603 {
		t.Errorf("results[0].ID = %d, want %d", results[0].ID, 603)
	}
}

func TestClient_SearchMovies_WithYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/movie" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		query := r.URL.Query().Get("query")
		if query != "Matrix" {
			t.Errorf("unexpected query: %s", query)
		}

		year := r.URL.Query().Get("year")
		if year != "1999" {
			t.Errorf("unexpected year: %s, want 1999", year)
		}

		response := SearchMoviesResponse{
			Page:         1,
			TotalResults: 1,
			TotalPages:   1,
			Results: []MovieResult{
				{
					ID:          603,
					Title:       "The Matrix",
					Overview:    "A computer hacker learns about the true nature of reality.",
					ReleaseDate: "1999-03-30",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := newTestClient(server)
	results, err := client.SearchMovies(context.Background(), "Matrix", 1999)
	if err != nil {
		t.Fatalf("SearchMovies() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("SearchMovies() returned %d results, want 1", len(results))
	}

	if results[0].Title != "The Matrix" {
		t.Errorf("results[0].Title = %q, want %q", results[0].Title, "The Matrix")
	}
}

func TestClient_SearchMovies_NoAPIKey(t *testing.T) {
	client := NewClient(config.TMDBConfig{}, zerolog.Nop())
	_, err := client.SearchMovies(context.Background(), "Matrix", 0)
	if err != ErrAPIKeyMissing {
		t.Errorf("SearchMovies() error = %v, want %v", err, ErrAPIKeyMissing)
	}
}

func TestClient_GetMovie(t *testing.T) {
	poster := "/poster.jpg"
	backdrop := "/backdrop.jpg"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/movie/603" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := MovieDetails{
			ID:          603,
			Title:       "The Matrix",
			Overview:    "A computer hacker learns about the true nature of reality.",
			ReleaseDate: "1999-03-30",
			Runtime:     136,
			ImdbID:      "tt0133093",
			PosterPath:  &poster,
			BackdropPath: &backdrop,
			Genres: []Genre{
				{ID: 28, Name: "Action"},
				{ID: 878, Name: "Science Fiction"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := newTestClient(server)
	result, err := client.GetMovie(context.Background(), 603)
	if err != nil {
		t.Fatalf("GetMovie() error = %v", err)
	}

	if result.Title != "The Matrix" {
		t.Errorf("Title = %q, want %q", result.Title, "The Matrix")
	}
	if result.Year != 1999 {
		t.Errorf("Year = %d, want %d", result.Year, 1999)
	}
	if result.Runtime != 136 {
		t.Errorf("Runtime = %d, want %d", result.Runtime, 136)
	}
	if result.ImdbID != "tt0133093" {
		t.Errorf("ImdbID = %q, want %q", result.ImdbID, "tt0133093")
	}
	if len(result.Genres) != 2 {
		t.Errorf("Genres = %d, want 2", len(result.Genres))
	}
	if result.PosterURL == "" {
		t.Error("PosterURL should not be empty")
	}
}

func TestClient_GetMovie_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			StatusCode:    34,
			StatusMessage: "The resource you requested could not be found.",
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.GetMovie(context.Background(), 99999999)
	if err != ErrMovieNotFound {
		t.Errorf("GetMovie() error = %v, want %v", err, ErrMovieNotFound)
	}
}

func TestClient_SearchSeries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/tv" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := SearchTVResponse{
			Page:         1,
			TotalResults: 1,
			TotalPages:   1,
			Results: []TVResult{
				{
					ID:           1396,
					Name:         "Breaking Bad",
					Overview:     "A high school chemistry teacher diagnosed with lung cancer.",
					FirstAirDate: "2008-01-20",
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
}

func TestClient_GetSeries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tv/1396" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		response := TVDetails{
			ID:               1396,
			Name:             "Breaking Bad",
			Overview:         "A high school chemistry teacher diagnosed with lung cancer.",
			FirstAirDate:     "2008-01-20",
			Status:           "Ended",
			NumberOfSeasons:  5,
			NumberOfEpisodes: 62,
			EpisodeRunTime:   []int{45, 47},
			Genres: []Genre{
				{ID: 18, Name: "Drama"},
			},
			ExternalIDs: &ExternalIDs{
				ImdbID: "tt0903747",
				TvdbID: 81189,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := newTestClient(server)
	result, err := client.GetSeries(context.Background(), 1396)
	if err != nil {
		t.Fatalf("GetSeries() error = %v", err)
	}

	if result.Title != "Breaking Bad" {
		t.Errorf("Title = %q, want %q", result.Title, "Breaking Bad")
	}
	if result.Year != 2008 {
		t.Errorf("Year = %d, want %d", result.Year, 2008)
	}
	if result.Status != "ended" {
		t.Errorf("Status = %q, want %q", result.Status, "ended")
	}
	if result.ImdbID != "tt0903747" {
		t.Errorf("ImdbID = %q, want %q", result.ImdbID, "tt0903747")
	}
	if result.TvdbID != 81189 {
		t.Errorf("TvdbID = %d, want %d", result.TvdbID, 81189)
	}
	if result.Runtime != 45 {
		t.Errorf("Runtime = %d, want %d", result.Runtime, 45)
	}
}

func TestClient_GetImageURL(t *testing.T) {
	client := NewClient(config.TMDBConfig{
		ImageBaseURL: "https://image.tmdb.org/t/p",
	}, zerolog.Nop())

	tests := []struct {
		path string
		size string
		want string
	}{
		{"/abc.jpg", "w500", "https://image.tmdb.org/t/p/w500/abc.jpg"},
		{"/poster.jpg", "original", "https://image.tmdb.org/t/p/original/poster.jpg"},
		{"", "w500", ""},
	}

	for _, tt := range tests {
		got := client.GetImageURL(tt.path, tt.size)
		if got != tt.want {
			t.Errorf("GetImageURL(%q, %q) = %q, want %q", tt.path, tt.size, got, tt.want)
		}
	}
}

func TestClient_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(ErrorResponse{
			StatusCode:    25,
			StatusMessage: "Your request count is over the allowed limit.",
		})
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.SearchMovies(context.Background(), "test", 0)
	if err != ErrRateLimited {
		t.Errorf("SearchMovies() error = %v, want %v", err, ErrRateLimited)
	}
}

func TestClient_StatusMapping(t *testing.T) {
	tests := []struct {
		tmdbStatus string
		wantStatus string
	}{
		{"Ended", "ended"},
		{"Canceled", "ended"},
		{"Returning Series", "continuing"},
		{"In Production", "continuing"},
		{"Planned", "upcoming"},
		{"Unknown", "continuing"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.tmdbStatus, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(TVDetails{
					ID:     1,
					Name:   "Test",
					Status: tt.tmdbStatus,
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

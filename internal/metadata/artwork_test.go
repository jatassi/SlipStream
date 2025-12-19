package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestArtworkDownloader_Download(t *testing.T) {
	// Create test server that returns image data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.Write([]byte("fake image data"))
	}))
	defer server.Close()

	// Create temp directory
	tempDir := t.TempDir()

	cfg := ArtworkConfig{
		BaseDir: tempDir,
		Timeout: 5 * time.Second,
	}

	downloader := NewArtworkDownloader(cfg, zerolog.Nop())

	path, err := downloader.Download(context.Background(), server.URL+"/poster.jpg", MediaTypeMovie, 603, ArtworkTypePoster)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Downloaded file does not exist: %s", path)
	}

	// Verify content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != "fake image data" {
		t.Errorf("File content = %q, want %q", string(content), "fake image data")
	}

	// Verify path format
	expectedPath := filepath.Join(tempDir, "movie", "603_poster.jpg")
	if path != expectedPath {
		t.Errorf("Path = %q, want %q", path, expectedPath)
	}
}

func TestArtworkDownloader_Download_InvalidURL(t *testing.T) {
	tempDir := t.TempDir()
	cfg := ArtworkConfig{BaseDir: tempDir, Timeout: 5 * time.Second}
	downloader := NewArtworkDownloader(cfg, zerolog.Nop())

	_, err := downloader.Download(context.Background(), "", MediaTypeMovie, 1, ArtworkTypePoster)
	if err != ErrInvalidURL {
		t.Errorf("Download() error = %v, want %v", err, ErrInvalidURL)
	}
}

func TestArtworkDownloader_Download_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	cfg := ArtworkConfig{BaseDir: tempDir, Timeout: 5 * time.Second}
	downloader := NewArtworkDownloader(cfg, zerolog.Nop())

	_, err := downloader.Download(context.Background(), server.URL+"/poster.jpg", MediaTypeMovie, 1, ArtworkTypePoster)
	if err == nil {
		t.Error("Download() expected error for server error")
	}
}

func TestArtworkDownloader_Download_Extensions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("image"))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	cfg := ArtworkConfig{BaseDir: tempDir, Timeout: 5 * time.Second}
	downloader := NewArtworkDownloader(cfg, zerolog.Nop())

	tests := []struct {
		url     string
		wantExt string
	}{
		{server.URL + "/image.jpg", ".jpg"},
		{server.URL + "/image.jpeg", ".jpeg"},
		{server.URL + "/image.png", ".png"},
		{server.URL + "/image.webp", ".webp"},
		{server.URL + "/image", ".jpg"}, // Default
		{server.URL + "/image?query=1", ".jpg"}, // With query string
	}

	for i, tt := range tests {
		path, err := downloader.Download(context.Background(), tt.url, MediaTypeMovie, i+100, ArtworkTypePoster)
		if err != nil {
			t.Errorf("Download(%q) error = %v", tt.url, err)
			continue
		}

		ext := filepath.Ext(path)
		if ext != tt.wantExt {
			t.Errorf("Download(%q) ext = %q, want %q", tt.url, ext, tt.wantExt)
		}
	}
}

func TestArtworkDownloader_DownloadMovieArtwork(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte("image"))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	cfg := ArtworkConfig{BaseDir: tempDir, Timeout: 5 * time.Second}
	downloader := NewArtworkDownloader(cfg, zerolog.Nop())

	movie := &MovieResult{
		ID:          603,
		Title:       "The Matrix",
		PosterURL:   server.URL + "/poster.jpg",
		BackdropURL: server.URL + "/backdrop.jpg",
	}

	err := downloader.DownloadMovieArtwork(context.Background(), movie)
	if err != nil {
		t.Fatalf("DownloadMovieArtwork() error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 downloads, got %d", callCount)
	}

	// Verify files exist
	posterPath := filepath.Join(tempDir, "movie", "603_poster.jpg")
	backdropPath := filepath.Join(tempDir, "movie", "603_backdrop.jpg")

	if _, err := os.Stat(posterPath); os.IsNotExist(err) {
		t.Errorf("Poster file does not exist: %s", posterPath)
	}
	if _, err := os.Stat(backdropPath); os.IsNotExist(err) {
		t.Errorf("Backdrop file does not exist: %s", backdropPath)
	}
}

func TestArtworkDownloader_DownloadMovieArtwork_NilMovie(t *testing.T) {
	tempDir := t.TempDir()
	cfg := ArtworkConfig{BaseDir: tempDir, Timeout: 5 * time.Second}
	downloader := NewArtworkDownloader(cfg, zerolog.Nop())

	err := downloader.DownloadMovieArtwork(context.Background(), nil)
	if err != ErrInvalidMediaType {
		t.Errorf("DownloadMovieArtwork() error = %v, want %v", err, ErrInvalidMediaType)
	}
}

func TestArtworkDownloader_DownloadSeriesArtwork(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte("image"))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	cfg := ArtworkConfig{BaseDir: tempDir, Timeout: 5 * time.Second}
	downloader := NewArtworkDownloader(cfg, zerolog.Nop())

	series := &SeriesResult{
		ID:          1396,
		Title:       "Breaking Bad",
		PosterURL:   server.URL + "/poster.jpg",
		BackdropURL: server.URL + "/backdrop.jpg",
	}

	err := downloader.DownloadSeriesArtwork(context.Background(), series)
	if err != nil {
		t.Fatalf("DownloadSeriesArtwork() error = %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected 2 downloads, got %d", callCount)
	}

	// Verify files exist
	posterPath := filepath.Join(tempDir, "series", "1396_poster.jpg")
	backdropPath := filepath.Join(tempDir, "series", "1396_backdrop.jpg")

	if _, err := os.Stat(posterPath); os.IsNotExist(err) {
		t.Errorf("Poster file does not exist: %s", posterPath)
	}
	if _, err := os.Stat(backdropPath); os.IsNotExist(err) {
		t.Errorf("Backdrop file does not exist: %s", backdropPath)
	}
}

func TestArtworkDownloader_GetArtworkPath(t *testing.T) {
	tempDir := t.TempDir()
	cfg := ArtworkConfig{BaseDir: tempDir, Timeout: 5 * time.Second}
	downloader := NewArtworkDownloader(cfg, zerolog.Nop())

	// Create a test file
	dir := filepath.Join(tempDir, "movie")
	os.MkdirAll(dir, 0755)
	testFile := filepath.Join(dir, "603_poster.jpg")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Test finding existing file
	path := downloader.GetArtworkPath(MediaTypeMovie, 603, ArtworkTypePoster)
	if path != testFile {
		t.Errorf("GetArtworkPath() = %q, want %q", path, testFile)
	}

	// Test non-existent file
	path = downloader.GetArtworkPath(MediaTypeMovie, 999, ArtworkTypePoster)
	if path != "" {
		t.Errorf("GetArtworkPath() = %q, want empty string", path)
	}
}

func TestArtworkDownloader_HasArtwork(t *testing.T) {
	tempDir := t.TempDir()
	cfg := ArtworkConfig{BaseDir: tempDir, Timeout: 5 * time.Second}
	downloader := NewArtworkDownloader(cfg, zerolog.Nop())

	// Create a test file
	dir := filepath.Join(tempDir, "movie")
	os.MkdirAll(dir, 0755)
	testFile := filepath.Join(dir, "603_poster.jpg")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Test existing file
	if !downloader.HasArtwork(MediaTypeMovie, 603, ArtworkTypePoster) {
		t.Error("HasArtwork() = false, want true")
	}

	// Test non-existent file
	if downloader.HasArtwork(MediaTypeMovie, 999, ArtworkTypePoster) {
		t.Error("HasArtwork() = true, want false")
	}
}

func TestArtworkDownloader_DeleteArtwork(t *testing.T) {
	tempDir := t.TempDir()
	cfg := ArtworkConfig{BaseDir: tempDir, Timeout: 5 * time.Second}
	downloader := NewArtworkDownloader(cfg, zerolog.Nop())

	// Create test files
	dir := filepath.Join(tempDir, "movie")
	os.MkdirAll(dir, 0755)
	posterFile := filepath.Join(dir, "603_poster.jpg")
	backdropFile := filepath.Join(dir, "603_backdrop.jpg")
	os.WriteFile(posterFile, []byte("poster"), 0644)
	os.WriteFile(backdropFile, []byte("backdrop"), 0644)

	// Delete artwork
	err := downloader.DeleteArtwork(MediaTypeMovie, 603)
	if err != nil {
		t.Fatalf("DeleteArtwork() error = %v", err)
	}

	// Verify files are deleted
	if _, err := os.Stat(posterFile); !os.IsNotExist(err) {
		t.Error("Poster file should be deleted")
	}
	if _, err := os.Stat(backdropFile); !os.IsNotExist(err) {
		t.Error("Backdrop file should be deleted")
	}
}

func TestArtworkDownloader_getExtension(t *testing.T) {
	downloader := &ArtworkDownloader{}

	tests := []struct {
		url  string
		want string
	}{
		{"https://example.com/image.jpg", ".jpg"},
		{"https://example.com/image.jpeg", ".jpeg"},
		{"https://example.com/image.png", ".png"},
		{"https://example.com/image.webp", ".webp"},
		{"https://example.com/image.gif", ".gif"},
		{"https://example.com/image.JPG", ".jpg"}, // Case insensitive
		{"https://example.com/image.PNG", ".png"},
		{"https://example.com/image", ""},
		{"https://example.com/image.unknown", ""},
		{"https://example.com/image.jpg?size=500", ".jpg"}, // With query string
		{"", ""},
	}

	for _, tt := range tests {
		got := downloader.getExtension(tt.url)
		if got != tt.want {
			t.Errorf("getExtension(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

func TestDefaultArtworkConfig(t *testing.T) {
	cfg := DefaultArtworkConfig()

	if cfg.BaseDir != "data/artwork" {
		t.Errorf("BaseDir = %q, want %q", cfg.BaseDir, "data/artwork")
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 30*time.Second)
	}
}

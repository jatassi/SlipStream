package organizer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func newTestService() *Service {
	logger := zerolog.New(zerolog.NewTestWriter(nil)).Level(zerolog.Disabled)
	return NewService(DefaultNamingConfig(), logger)
}

func TestNewService(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultNamingConfig()

	service := NewService(config, logger)

	if service == nil {
		t.Fatal("NewService() returned nil")
	}

	if service.config.MovieFolderFormat != config.MovieFolderFormat {
		t.Error("NewService() did not set config correctly")
	}
}

func TestService_GetConfig(t *testing.T) {
	service := newTestService()
	config := service.GetConfig()

	if config.MovieFolderFormat != DefaultNamingConfig().MovieFolderFormat {
		t.Error("GetConfig() returned unexpected config")
	}
}

func TestService_SetConfig(t *testing.T) {
	service := newTestService()

	newConfig := NamingConfig{
		MovieFolderFormat: "{Title} - {Year}",
	}

	service.SetConfig(newConfig)

	if service.config.MovieFolderFormat != "{Title} - {Year}" {
		t.Error("SetConfig() did not update config")
	}
}

func TestService_GenerateMoviePath(t *testing.T) {
	service := newTestService()

	tests := []struct {
		name     string
		rootPath string
		tokens   MovieTokens
		want     string
	}{
		{
			name:     "standard movie",
			rootPath: "/movies",
			tokens:   MovieTokens{Title: "The Matrix", Year: 1999},
			want:     filepath.Join("/movies", "The Matrix (1999)"),
		},
		{
			name:     "movie without year",
			rootPath: "/movies",
			tokens:   MovieTokens{Title: "Unknown"},
			want:     filepath.Join("/movies", "Unknown"),
		},
		{
			name:     "windows path",
			rootPath: "C:\\Movies",
			tokens:   MovieTokens{Title: "Inception", Year: 2010},
			want:     filepath.Join("C:\\Movies", "Inception (2010)"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.GenerateMoviePath(tt.rootPath, tt.tokens)
			if got != tt.want {
				t.Errorf("GenerateMoviePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestService_GenerateMovieFilename(t *testing.T) {
	service := newTestService()

	tokens := MovieTokens{Title: "The Matrix", Year: 1999}
	got := service.GenerateMovieFilename(tokens)
	want := "The Matrix (1999)"

	if got != want {
		t.Errorf("GenerateMovieFilename() = %q, want %q", got, want)
	}
}

func TestService_GenerateSeriesPath(t *testing.T) {
	service := newTestService()

	tests := []struct {
		name     string
		rootPath string
		tokens   SeriesTokens
		want     string
	}{
		{
			name:     "standard series",
			rootPath: "/tv",
			tokens:   SeriesTokens{SeriesTitle: "Breaking Bad"},
			want:     filepath.Join("/tv", "Breaking Bad"),
		},
		{
			name:     "series with special chars",
			rootPath: "/tv",
			tokens:   SeriesTokens{SeriesTitle: "Marvel's Agents of S.H.I.E.L.D."},
			want:     filepath.Join("/tv", "Marvel's Agents of S.H.I.E.L.D."),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.GenerateSeriesPath(tt.rootPath, tt.tokens)
			if got != tt.want {
				t.Errorf("GenerateSeriesPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestService_GenerateSeasonPath(t *testing.T) {
	service := newTestService()

	tests := []struct {
		seriesPath   string
		seasonNumber int
		want         string
	}{
		{"/tv/Breaking Bad", 1, filepath.Join("/tv/Breaking Bad", "Season 01")},
		{"/tv/Breaking Bad", 5, filepath.Join("/tv/Breaking Bad", "Season 05")},
		{"/tv/Breaking Bad", 10, filepath.Join("/tv/Breaking Bad", "Season 10")},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := service.GenerateSeasonPath(tt.seriesPath, tt.seasonNumber)
			if got != tt.want {
				t.Errorf("GenerateSeasonPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestService_GenerateEpisodeFilename(t *testing.T) {
	service := newTestService()

	tests := []struct {
		name   string
		tokens EpisodeTokens
		want   string
	}{
		{
			name: "standard episode",
			tokens: EpisodeTokens{
				SeriesTitle:   "Breaking Bad",
				SeasonNumber:  1,
				EpisodeNumber: 1,
				EpisodeTitle:  "Pilot",
			},
			want: "Breaking Bad - S01E01 - Pilot",
		},
		{
			name: "multi-episode",
			tokens: EpisodeTokens{
				SeriesTitle:   "Game of Thrones",
				SeasonNumber:  1,
				EpisodeNumber: 1,
				EndEpisode:    2,
				EpisodeTitle:  "Winter Is Coming",
			},
			want: "Game of Thrones - S01E01-E02 - Winter Is Coming",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.GenerateEpisodeFilename(tt.tokens)
			if got != tt.want {
				t.Errorf("GenerateEpisodeFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestService_PreviewMovieRename(t *testing.T) {
	service := newTestService()

	tokens := MovieTokens{Title: "The Matrix", Year: 1999}
	folder, filename := service.PreviewMovieRename(tokens)

	if folder != "The Matrix (1999)" {
		t.Errorf("PreviewMovieRename() folder = %q, want %q", folder, "The Matrix (1999)")
	}

	if filename != "The Matrix (1999)" {
		t.Errorf("PreviewMovieRename() filename = %q, want %q", filename, "The Matrix (1999)")
	}
}

func TestService_PreviewEpisodeRename(t *testing.T) {
	service := newTestService()

	tokens := EpisodeTokens{
		SeriesTitle:   "Breaking Bad",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		EpisodeTitle:  "Pilot",
	}

	seasonFolder, filename := service.PreviewEpisodeRename(tokens, 1)

	if seasonFolder != "Season 01" {
		t.Errorf("PreviewEpisodeRename() seasonFolder = %q, want %q", seasonFolder, "Season 01")
	}

	if filename != "Breaking Bad - S01E01 - Pilot" {
		t.Errorf("PreviewEpisodeRename() filename = %q, want %q", filename, "Breaking Bad - S01E01 - Pilot")
	}
}

// File operation tests require a temp directory
func TestService_MoveFile(t *testing.T) {
	service := newTestService()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "organizer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(srcPath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Move to new location
	destPath := filepath.Join(tmpDir, "subdir", "dest.txt")
	if err := service.MoveFile(srcPath, destPath); err != nil {
		t.Fatalf("MoveFile() error = %v", err)
	}

	// Verify source is gone
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("MoveFile() did not remove source file")
	}

	// Verify dest exists
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("MoveFile() dest file not found: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read dest file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("MoveFile() content = %q, want %q", string(content), "test content")
	}
}

func TestService_CopyFile(t *testing.T) {
	service := newTestService()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "organizer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "source.txt")
	if err := os.WriteFile(srcPath, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy to new location
	destPath := filepath.Join(tmpDir, "subdir", "dest.txt")
	if err := service.CopyFile(srcPath, destPath); err != nil {
		t.Fatalf("CopyFile() error = %v", err)
	}

	// Verify source still exists
	if _, err := os.Stat(srcPath); err != nil {
		t.Error("CopyFile() removed source file")
	}

	// Verify dest exists
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("CopyFile() dest file not found: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read dest file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("CopyFile() content = %q, want %q", string(content), "test content")
	}
}

func TestService_RenameFile(t *testing.T) {
	service := newTestService()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "organizer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "old_name.mkv")
	if err := os.WriteFile(srcPath, []byte("video content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Rename file
	newPath, err := service.RenameFile(srcPath, "new_name")
	if err != nil {
		t.Fatalf("RenameFile() error = %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "new_name.mkv")
	if newPath != expectedPath {
		t.Errorf("RenameFile() = %q, want %q", newPath, expectedPath)
	}

	// Verify old file is gone
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Error("RenameFile() did not remove old file")
	}

	// Verify new file exists
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("RenameFile() new file not found: %v", err)
	}
}

func TestService_RenameFile_SameName(t *testing.T) {
	service := newTestService()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "organizer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcPath := filepath.Join(tmpDir, "same_name.mkv")
	if err := os.WriteFile(srcPath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Rename to same name (should be no-op)
	newPath, err := service.RenameFile(srcPath, "same_name")
	if err != nil {
		t.Fatalf("RenameFile() error = %v", err)
	}

	if newPath != srcPath {
		t.Errorf("RenameFile() same name = %q, want %q", newPath, srcPath)
	}
}

func TestService_OrganizeMovie(t *testing.T) {
	service := newTestService()

	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "organizer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	downloadDir := filepath.Join(tmpDir, "downloads")
	movieDir := filepath.Join(tmpDir, "movies")
	os.MkdirAll(downloadDir, 0755)

	// Create source file
	srcPath := filepath.Join(downloadDir, "The.Matrix.1999.1080p.BluRay.mkv")
	if err := os.WriteFile(srcPath, []byte("movie content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	tokens := MovieTokens{Title: "The Matrix", Year: 1999}
	destPath, err := service.OrganizeMovie(srcPath, movieDir, tokens)
	if err != nil {
		t.Fatalf("OrganizeMovie() error = %v", err)
	}

	expectedPath := filepath.Join(movieDir, "The Matrix (1999)", "The Matrix (1999).mkv")
	if destPath != expectedPath {
		t.Errorf("OrganizeMovie() = %q, want %q", destPath, expectedPath)
	}

	// Verify file exists at destination
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("OrganizeMovie() dest file not found: %v", err)
	}
}

func TestService_OrganizeEpisode(t *testing.T) {
	service := newTestService()

	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "organizer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	downloadDir := filepath.Join(tmpDir, "downloads")
	seriesDir := filepath.Join(tmpDir, "tv", "Breaking Bad")
	os.MkdirAll(downloadDir, 0755)

	// Create source file
	srcPath := filepath.Join(downloadDir, "Breaking.Bad.S01E01.720p.mkv")
	if err := os.WriteFile(srcPath, []byte("episode content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	tokens := EpisodeTokens{
		SeriesTitle:   "Breaking Bad",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		EpisodeTitle:  "Pilot",
	}

	destPath, err := service.OrganizeEpisode(srcPath, seriesDir, 1, tokens)
	if err != nil {
		t.Fatalf("OrganizeEpisode() error = %v", err)
	}

	expectedPath := filepath.Join(seriesDir, "Season 01", "Breaking Bad - S01E01 - Pilot.mkv")
	if destPath != expectedPath {
		t.Errorf("OrganizeEpisode() = %q, want %q", destPath, expectedPath)
	}

	// Verify file exists at destination
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("OrganizeEpisode() dest file not found: %v", err)
	}
}

func TestService_CleanEmptyFolders(t *testing.T) {
	service := newTestService()

	// Create temp directory structure
	tmpDir, err := os.MkdirTemp("", "organizer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some empty directories
	emptyDir1 := filepath.Join(tmpDir, "empty1")
	emptyDir2 := filepath.Join(tmpDir, "empty2", "nested")
	nonEmptyDir := filepath.Join(tmpDir, "notempty")

	os.MkdirAll(emptyDir1, 0755)
	os.MkdirAll(emptyDir2, 0755)
	os.MkdirAll(nonEmptyDir, 0755)

	// Add a file to nonEmptyDir
	os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("content"), 0644)

	// Clean empty folders
	if err := service.CleanEmptyFolders(tmpDir); err != nil {
		t.Fatalf("CleanEmptyFolders() error = %v", err)
	}

	// Verify empty directories are removed
	if _, err := os.Stat(emptyDir1); !os.IsNotExist(err) {
		t.Error("CleanEmptyFolders() did not remove empty1")
	}

	// The nested empty directory should be removed
	if _, err := os.Stat(emptyDir2); !os.IsNotExist(err) {
		t.Error("CleanEmptyFolders() did not remove nested empty dir")
	}

	// Non-empty directory should remain
	if _, err := os.Stat(nonEmptyDir); err != nil {
		t.Error("CleanEmptyFolders() removed non-empty directory")
	}
}

func TestService_MoveFile_NonExistentSource(t *testing.T) {
	service := newTestService()

	tmpDir, err := os.MkdirTemp("", "organizer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "nonexistent.mkv")
	destPath := filepath.Join(tmpDir, "dest.mkv")

	err = service.MoveFile(srcPath, destPath)
	if err == nil {
		t.Error("MoveFile() with non-existent source should return error")
	}
}

func TestService_CopyFile_NonExistentSource(t *testing.T) {
	service := newTestService()

	tmpDir, err := os.MkdirTemp("", "organizer_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "nonexistent.mkv")
	destPath := filepath.Join(tmpDir, "dest.mkv")

	err = service.CopyFile(srcPath, destPath)
	if err == nil {
		t.Error("CopyFile() with non-existent source should return error")
	}
}

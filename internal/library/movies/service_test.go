package movies

import (
	"context"
	"testing"

	"github.com/slipstream/slipstream/internal/testutil"
)

func TestMovieService_Create(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	input := CreateMovieInput{
		Title:     "The Matrix",
		Year:      1999,
		TmdbID:    603,
		ImdbID:    "tt0133093",
		Overview:  "A computer hacker learns about the true nature of reality.",
		Runtime:   136,
		Monitored: true,
	}

	movie, err := service.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if movie.ID == 0 {
		t.Error("Create() movie.ID = 0, want non-zero")
	}
	if movie.Title != input.Title {
		t.Errorf("Create() movie.Title = %q, want %q", movie.Title, input.Title)
	}
	if movie.Year != input.Year {
		t.Errorf("Create() movie.Year = %d, want %d", movie.Year, input.Year)
	}
	if movie.TmdbID != input.TmdbID {
		t.Errorf("Create() movie.TmdbID = %d, want %d", movie.TmdbID, input.TmdbID)
	}
	if movie.Status != "unreleased" {
		t.Errorf("Create() movie.Status = %q, want %q", movie.Status, "unreleased")
	}
	if !movie.Monitored {
		t.Error("Create() movie.Monitored = false, want true")
	}
}

func TestMovieService_Create_EmptyTitle(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	input := CreateMovieInput{
		Title: "",
	}

	_, err := service.Create(ctx, input)
	if err != ErrInvalidMovie {
		t.Errorf("Create() with empty title error = %v, want %v", err, ErrInvalidMovie)
	}
}

func TestMovieService_Create_DuplicateTmdbID(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create first movie
	input := CreateMovieInput{
		Title:  "The Matrix",
		TmdbID: 603,
	}

	_, err := service.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create() first movie error = %v", err)
	}

	// Try to create duplicate
	input2 := CreateMovieInput{
		Title:  "The Matrix Duplicate",
		TmdbID: 603, // Same TMDB ID
	}

	_, err = service.Create(ctx, input2)
	if err != ErrDuplicateTmdbID {
		t.Errorf("Create() duplicate TMDB ID error = %v, want %v", err, ErrDuplicateTmdbID)
	}
}

func TestMovieService_Get(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create a movie first
	input := CreateMovieInput{
		Title:    "Inception",
		Year:     2010,
		Overview: "A thief who steals corporate secrets through dream-sharing technology.",
	}

	created, err := service.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get the movie
	movie, err := service.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if movie.ID != created.ID {
		t.Errorf("Get() movie.ID = %d, want %d", movie.ID, created.ID)
	}
	if movie.Title != input.Title {
		t.Errorf("Get() movie.Title = %q, want %q", movie.Title, input.Title)
	}
}

func TestMovieService_Get_NotFound(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	_, err := service.Get(ctx, 99999)
	if err != ErrMovieNotFound {
		t.Errorf("Get() non-existent movie error = %v, want %v", err, ErrMovieNotFound)
	}
}

func TestMovieService_GetByTmdbID(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	input := CreateMovieInput{
		Title:  "Interstellar",
		Year:   2014,
		TmdbID: 157336,
	}

	created, err := service.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	movie, err := service.GetByTmdbID(ctx, 157336)
	if err != nil {
		t.Fatalf("GetByTmdbID() error = %v", err)
	}

	if movie.ID != created.ID {
		t.Errorf("GetByTmdbID() movie.ID = %d, want %d", movie.ID, created.ID)
	}
}

func TestMovieService_List(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create multiple movies
	movies := []CreateMovieInput{
		{Title: "Movie 1", Year: 2020},
		{Title: "Movie 2", Year: 2021},
		{Title: "Movie 3", Year: 2022},
	}

	for _, input := range movies {
		_, err := service.Create(ctx, input)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// List all
	list, err := service.List(ctx, ListMoviesOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("List() returned %d movies, want 3", len(list))
	}
}

func TestMovieService_List_Pagination(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create 5 movies
	for i := 1; i <= 5; i++ {
		_, err := service.Create(ctx, CreateMovieInput{Title: "Movie", Year: 2020 + i})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Get first page
	list, err := service.List(ctx, ListMoviesOptions{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(list) != 2 {
		t.Errorf("List() page 1 returned %d movies, want 2", len(list))
	}

	// Get second page
	list2, err := service.List(ctx, ListMoviesOptions{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("List() page 2 error = %v", err)
	}

	if len(list2) != 2 {
		t.Errorf("List() page 2 returned %d movies, want 2", len(list2))
	}
}

func TestMovieService_List_Search(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create movies
	_, _ = service.Create(ctx, CreateMovieInput{Title: "The Matrix", Year: 1999})
	_, _ = service.Create(ctx, CreateMovieInput{Title: "Matrix Reloaded", Year: 2003})
	_, _ = service.Create(ctx, CreateMovieInput{Title: "Inception", Year: 2010})

	// Search for "Matrix"
	list, err := service.List(ctx, ListMoviesOptions{Search: "Matrix"})
	if err != nil {
		t.Fatalf("List() search error = %v", err)
	}

	if len(list) != 2 {
		t.Errorf("List() search returned %d movies, want 2", len(list))
	}
}

func TestMovieService_Update(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create a movie
	created, err := service.Create(ctx, CreateMovieInput{
		Title:     "Original Title",
		Year:      2020,
		Monitored: false,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update the movie
	newTitle := "Updated Title"
	newYear := 2021
	monitored := true

	updated, err := service.Update(ctx, created.ID, UpdateMovieInput{
		Title:     &newTitle,
		Year:      &newYear,
		Monitored: &monitored,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("Update() movie.Title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Year != newYear {
		t.Errorf("Update() movie.Year = %d, want %d", updated.Year, newYear)
	}
	if !updated.Monitored {
		t.Error("Update() movie.Monitored = false, want true")
	}
}

func TestMovieService_Update_PartialUpdate(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create a movie with multiple fields
	created, err := service.Create(ctx, CreateMovieInput{
		Title:    "Test Movie",
		Year:     2020,
		Overview: "Original overview",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update only the title
	newTitle := "New Title"
	updated, err := service.Update(ctx, created.ID, UpdateMovieInput{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("Update() title = %q, want %q", updated.Title, newTitle)
	}
	// Other fields should remain unchanged
	if updated.Year != 2020 {
		t.Errorf("Update() year changed to %d, want 2020", updated.Year)
	}
	if updated.Overview != "Original overview" {
		t.Errorf("Update() overview changed to %q", updated.Overview)
	}
}

func TestMovieService_Update_NotFound(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	newTitle := "Test"
	_, err := service.Update(ctx, 99999, UpdateMovieInput{Title: &newTitle})
	if err != ErrMovieNotFound {
		t.Errorf("Update() non-existent movie error = %v, want %v", err, ErrMovieNotFound)
	}
}

func TestMovieService_Delete(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create a movie
	created, err := service.Create(ctx, CreateMovieInput{Title: "To Delete", Year: 2020})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete the movie
	err = service.Delete(ctx, created.ID, false)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err = service.Get(ctx, created.ID)
	if err != ErrMovieNotFound {
		t.Errorf("Get() after delete error = %v, want %v", err, ErrMovieNotFound)
	}
}

func TestMovieService_Delete_NotFound(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	err := service.Delete(ctx, 99999, false)
	if err != ErrMovieNotFound {
		t.Errorf("Delete() non-existent movie error = %v, want %v", err, ErrMovieNotFound)
	}
}

func TestMovieService_AddFile(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create a movie
	movie, err := service.Create(ctx, CreateMovieInput{Title: "Test Movie", Year: 2020})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Add a file
	fileInput := CreateMovieFileInput{
		Path:       "/movies/Test Movie (2020)/Test Movie (2020).mkv",
		Size:       1500000000,
		Quality:    "Bluray-1080p",
		VideoCodec: "x264",
		Resolution: "1920x1080",
	}

	file, err := service.AddFile(ctx, movie.ID, fileInput)
	if err != nil {
		t.Fatalf("AddFile() error = %v", err)
	}

	if file.ID == 0 {
		t.Error("AddFile() file.ID = 0, want non-zero")
	}
	if file.Path != fileInput.Path {
		t.Errorf("AddFile() file.Path = %q, want %q", file.Path, fileInput.Path)
	}
	if file.Size != fileInput.Size {
		t.Errorf("AddFile() file.Size = %d, want %d", file.Size, fileInput.Size)
	}

	// Verify movie now has file
	updated, err := service.Get(ctx, movie.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if updated.Status != "available" {
		t.Errorf("Movie status = %q, want %q", updated.Status, "available")
	}
}

func TestMovieService_GetFiles(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create a movie
	movie, _ := service.Create(ctx, CreateMovieInput{Title: "Test", Year: 2020})

	// Add multiple files
	_, _ = service.AddFile(ctx, movie.ID, CreateMovieFileInput{Path: "/path1.mkv", Size: 1000})
	_, _ = service.AddFile(ctx, movie.ID, CreateMovieFileInput{Path: "/path2.mkv", Size: 2000})

	files, err := service.GetFiles(ctx, movie.ID)
	if err != nil {
		t.Fatalf("GetFiles() error = %v", err)
	}

	if len(files) != 2 {
		t.Errorf("GetFiles() returned %d files, want 2", len(files))
	}
}

func TestMovieService_RemoveFile(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Create a movie and add a file
	movie, _ := service.Create(ctx, CreateMovieInput{Title: "Test", Year: 2020})
	file, _ := service.AddFile(ctx, movie.ID, CreateMovieFileInput{Path: "/path.mkv", Size: 1000})

	// Remove the file
	err := service.RemoveFile(ctx, file.ID)
	if err != nil {
		t.Fatalf("RemoveFile() error = %v", err)
	}

	// Verify file is gone
	files, _ := service.GetFiles(ctx, movie.ID)
	if len(files) != 0 {
		t.Errorf("GetFiles() after remove returned %d files, want 0", len(files))
	}

	// Verify movie status is back to missing
	updated, _ := service.Get(ctx, movie.ID)
	if updated.Status != "missing" {
		t.Errorf("Movie status after file removal = %q, want %q", updated.Status, "missing")
	}
}

func TestMovieService_RemoveFile_NotFound(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	err := service.RemoveFile(ctx, 99999)
	if err != ErrMovieFileNotFound {
		t.Errorf("RemoveFile() non-existent file error = %v, want %v", err, ErrMovieFileNotFound)
	}
}

func TestMovieService_Count(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, tdb.Logger)
	ctx := context.Background()

	// Initially should be 0
	count, err := service.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() initially = %d, want 0", count)
	}

	// Add some movies
	_, _ = service.Create(ctx, CreateMovieInput{Title: "Movie 1"})
	_, _ = service.Create(ctx, CreateMovieInput{Title: "Movie 2"})
	_, _ = service.Create(ctx, CreateMovieInput{Title: "Movie 3"})

	count, err = service.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

func TestGenerateSortTitle(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"The Matrix", "Matrix"},
		{"A Beautiful Mind", "Beautiful Mind"},
		{"An American Werewolf", "American Werewolf"},
		{"Inception", "Inception"},
		{"The", "The"},         // Too short to strip
		{"A", "A"},             // Too short to strip
		{"Theatre", "Theatre"}, // "The" is part of word, not prefix
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := generateSortTitle(tt.title)
			if got != tt.want {
				t.Errorf("generateSortTitle(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}

func TestGenerateMoviePath(t *testing.T) {
	tests := []struct {
		rootPath string
		title    string
		year     int
		want     string
	}{
		{"/movies", "The Matrix", 1999, "/movies/The Matrix (1999)"},
		{"/movies", "Unknown", 0, "/movies/Unknown"},
		// Uses forward slash as separator for cross-platform consistency
		{"C:\\Movies", "Inception", 2010, "C:\\Movies/Inception (2010)"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			got := GenerateMoviePath(tt.rootPath, tt.title, tt.year)
			if got != tt.want {
				t.Errorf("GenerateMoviePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

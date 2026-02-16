package tv

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/slipstream/slipstream/internal/testutil"
)

func TestTVService_CreateSeries(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	input := CreateSeriesInput{
		Title:        "Breaking Bad",
		Year:         2008,
		TvdbID:       81189,
		TmdbID:       1396,
		Overview:     "A high school chemistry teacher turned methamphetamine manufacturer.",
		Monitored:    true,
		SeasonFolder: true,
	}

	series, err := service.CreateSeries(ctx, &input)
	if err != nil {
		t.Fatalf("CreateSeries() error = %v", err)
	}

	if series.ID == 0 {
		t.Error("CreateSeries() series.ID = 0, want non-zero")
	}
	if series.Title != input.Title {
		t.Errorf("CreateSeries() series.Title = %q, want %q", series.Title, input.Title)
	}
	if series.Year != input.Year {
		t.Errorf("CreateSeries() series.Year = %d, want %d", series.Year, input.Year)
	}
	if series.TvdbID != input.TvdbID {
		t.Errorf("CreateSeries() series.TvdbID = %d, want %d", series.TvdbID, input.TvdbID)
	}
	if !series.Monitored {
		t.Error("CreateSeries() series.Monitored = false, want true")
	}
	if !series.SeasonFolder {
		t.Error("CreateSeries() series.SeasonFolder = false, want true")
	}
	if series.ProductionStatus != "continuing" {
		t.Errorf("CreateSeries() series.ProductionStatus = %q, want %q", series.ProductionStatus, "continuing")
	}
}

func TestTVService_CreateSeries_WithSeasons(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	airDate := time.Date(2008, 1, 20, 0, 0, 0, 0, time.UTC)

	input := CreateSeriesInput{
		Title:     "Breaking Bad",
		Year:      2008,
		Monitored: true,
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Monitored:    true,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Pilot", AirDate: &airDate, Monitored: true},
					{EpisodeNumber: 2, Title: "Cat's in the Bag...", Monitored: true},
				},
			},
			{
				SeasonNumber: 2,
				Monitored:    true,
			},
		},
	}

	series, err := service.CreateSeries(ctx, &input)
	if err != nil {
		t.Fatalf("CreateSeries() error = %v", err)
	}

	// Verify seasons were created
	seasons, err := service.ListSeasons(ctx, series.ID)
	if err != nil {
		t.Fatalf("ListSeasons() error = %v", err)
	}

	if len(seasons) != 2 {
		t.Errorf("ListSeasons() returned %d seasons, want 2", len(seasons))
	}

	// Verify episodes were created
	episodes, err := service.ListEpisodes(ctx, series.ID, testutil.IntPtr(1))
	if err != nil {
		t.Fatalf("ListEpisodes() error = %v", err)
	}

	if len(episodes) != 2 {
		t.Errorf("ListEpisodes() for season 1 returned %d episodes, want 2", len(episodes))
	}
}

func TestTVService_CreateSeries_EmptyTitle(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	input := CreateSeriesInput{Title: ""}

	_, err := service.CreateSeries(ctx, &input)
	if !errors.Is(err, ErrInvalidSeries) {
		t.Errorf("CreateSeries() with empty title error = %v, want %v", err, ErrInvalidSeries)
	}
}

func TestTVService_CreateSeries_DuplicateTvdbID(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	// Create first series
	_, err := service.CreateSeries(ctx, &CreateSeriesInput{Title: "Breaking Bad", TvdbID: 81189})
	if err != nil {
		t.Fatalf("CreateSeries() first error = %v", err)
	}

	// Try to create duplicate
	_, err = service.CreateSeries(ctx, &CreateSeriesInput{Title: "Breaking Bad Copy", TvdbID: 81189})
	if !errors.Is(err, ErrDuplicateTvdbID) {
		t.Errorf("CreateSeries() duplicate TVDB ID error = %v, want %v", err, ErrDuplicateTvdbID)
	}
}

func TestTVService_GetSeries(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	created, err := service.CreateSeries(ctx, &CreateSeriesInput{
		Title:    "The Office",
		Year:     2005,
		Overview: "A mockumentary on a group of typical office workers.",
	})
	if err != nil {
		t.Fatalf("CreateSeries() error = %v", err)
	}

	series, err := service.GetSeries(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetSeries() error = %v", err)
	}

	if series.ID != created.ID {
		t.Errorf("GetSeries() series.ID = %d, want %d", series.ID, created.ID)
	}
	if series.Title != "The Office" {
		t.Errorf("GetSeries() series.Title = %q, want %q", series.Title, "The Office")
	}
}

func TestTVService_GetSeries_NotFound(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	_, err := service.GetSeries(ctx, 99999)
	if !errors.Is(err, ErrSeriesNotFound) {
		t.Errorf("GetSeries() non-existent error = %v, want %v", err, ErrSeriesNotFound)
	}
}

func TestTVService_GetSeriesByTvdbID(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	created, _ := service.CreateSeries(ctx, &CreateSeriesInput{Title: "Test", TvdbID: 12345})

	series, err := service.GetSeriesByTvdbID(ctx, 12345)
	if err != nil {
		t.Fatalf("GetSeriesByTvdbID() error = %v", err)
	}

	if series.ID != created.ID {
		t.Errorf("GetSeriesByTvdbID() series.ID = %d, want %d", series.ID, created.ID)
	}
}

func TestTVService_ListSeries(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	// Create multiple series
	_, _ = service.CreateSeries(ctx, &CreateSeriesInput{Title: "Series 1", Year: 2020})
	_, _ = service.CreateSeries(ctx, &CreateSeriesInput{Title: "Series 2", Year: 2021})
	_, _ = service.CreateSeries(ctx, &CreateSeriesInput{Title: "Series 3", Year: 2022})

	list, err := service.ListSeries(ctx, ListSeriesOptions{})
	if err != nil {
		t.Fatalf("ListSeries() error = %v", err)
	}

	if len(list) != 3 {
		t.Errorf("ListSeries() returned %d series, want 3", len(list))
	}
}

func TestTVService_ListSeries_Search(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	_, _ = service.CreateSeries(ctx, &CreateSeriesInput{Title: "Breaking Bad"})
	_, _ = service.CreateSeries(ctx, &CreateSeriesInput{Title: "Better Call Saul"})
	_, _ = service.CreateSeries(ctx, &CreateSeriesInput{Title: "The Office"})

	list, err := service.ListSeries(ctx, ListSeriesOptions{Search: "Breaking"})
	if err != nil {
		t.Fatalf("ListSeries() search error = %v", err)
	}

	if len(list) != 1 {
		t.Errorf("ListSeries() search returned %d series, want 1", len(list))
	}
}

func TestTVService_UpdateSeries(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	created, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title:     "Original Title",
		Year:      2020,
		Monitored: false,
	})

	newTitle := "Updated Title"
	newYear := 2021
	monitored := true

	updated, err := service.UpdateSeries(ctx, created.ID, &UpdateSeriesInput{
		Title:     &newTitle,
		Year:      &newYear,
		Monitored: &monitored,
	})
	if err != nil {
		t.Fatalf("UpdateSeries() error = %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("UpdateSeries() title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Year != newYear {
		t.Errorf("UpdateSeries() year = %d, want %d", updated.Year, newYear)
	}
	if !updated.Monitored {
		t.Error("UpdateSeries() monitored = false, want true")
	}
}

func TestTVService_UpdateSeries_NotFound(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	newTitle := "Test"
	_, err := service.UpdateSeries(ctx, 99999, &UpdateSeriesInput{Title: &newTitle})
	if !errors.Is(err, ErrSeriesNotFound) {
		t.Errorf("UpdateSeries() non-existent error = %v, want %v", err, ErrSeriesNotFound)
	}
}

func TestTVService_DeleteSeries(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	created, _ := service.CreateSeries(ctx, &CreateSeriesInput{Title: "To Delete"})

	err := service.DeleteSeries(ctx, created.ID, false)
	if err != nil {
		t.Fatalf("DeleteSeries() error = %v", err)
	}

	_, err = service.GetSeries(ctx, created.ID)
	if !errors.Is(err, ErrSeriesNotFound) {
		t.Errorf("GetSeries() after delete error = %v, want %v", err, ErrSeriesNotFound)
	}
}

func TestTVService_DeleteSeries_NotFound(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	err := service.DeleteSeries(ctx, 99999, false)
	if !errors.Is(err, ErrSeriesNotFound) {
		t.Errorf("DeleteSeries() non-existent error = %v, want %v", err, ErrSeriesNotFound)
	}
}

func TestTVService_ListSeasons(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{SeasonNumber: 1, Monitored: true},
			{SeasonNumber: 2, Monitored: true},
			{SeasonNumber: 3, Monitored: false},
		},
	})

	seasons, err := service.ListSeasons(ctx, series.ID)
	if err != nil {
		t.Fatalf("ListSeasons() error = %v", err)
	}

	if len(seasons) != 3 {
		t.Errorf("ListSeasons() returned %d seasons, want 3", len(seasons))
	}
}

func TestTVService_UpdateSeasonMonitored(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{SeasonNumber: 1, Monitored: true},
		},
	})

	updated, err := service.UpdateSeasonMonitored(ctx, series.ID, 1, false)
	if err != nil {
		t.Fatalf("UpdateSeasonMonitored() error = %v", err)
	}

	if updated.Monitored {
		t.Error("UpdateSeasonMonitored() monitored = true, want false")
	}
}

func TestTVService_UpdateSeasonMonitored_NotFound(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{Title: "Test"})

	_, err := service.UpdateSeasonMonitored(ctx, series.ID, 99, false)
	if !errors.Is(err, ErrSeasonNotFound) {
		t.Errorf("UpdateSeasonMonitored() non-existent error = %v, want %v", err, ErrSeasonNotFound)
	}
}

func TestTVService_ListEpisodes(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test Series",
		Seasons: []SeasonInput{
			{
				SeasonNumber: 1,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "Ep 1"},
					{EpisodeNumber: 2, Title: "Ep 2"},
				},
			},
			{
				SeasonNumber: 2,
				Episodes: []EpisodeInput{
					{EpisodeNumber: 1, Title: "S2 Ep 1"},
				},
			},
		},
	})

	// List all episodes
	allEpisodes, err := service.ListEpisodes(ctx, series.ID, nil)
	if err != nil {
		t.Fatalf("ListEpisodes() all error = %v", err)
	}

	if len(allEpisodes) != 3 {
		t.Errorf("ListEpisodes() all returned %d episodes, want 3", len(allEpisodes))
	}

	// List season 1 episodes only
	s1Episodes, err := service.ListEpisodes(ctx, series.ID, testutil.IntPtr(1))
	if err != nil {
		t.Fatalf("ListEpisodes() season 1 error = %v", err)
	}

	if len(s1Episodes) != 2 {
		t.Errorf("ListEpisodes() season 1 returned %d episodes, want 2", len(s1Episodes))
	}
}

func TestTVService_GetEpisode(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test",
		Seasons: []SeasonInput{
			{SeasonNumber: 1, Episodes: []EpisodeInput{{EpisodeNumber: 1, Title: "Pilot"}}},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)
	if len(episodes) == 0 {
		t.Fatal("No episodes created")
	}

	episode, err := service.GetEpisode(ctx, episodes[0].ID)
	if err != nil {
		t.Fatalf("GetEpisode() error = %v", err)
	}

	if episode.Title != "Pilot" {
		t.Errorf("GetEpisode() title = %q, want %q", episode.Title, "Pilot")
	}
}

func TestTVService_GetEpisode_NotFound(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	_, err := service.GetEpisode(ctx, 99999)
	if !errors.Is(err, ErrEpisodeNotFound) {
		t.Errorf("GetEpisode() non-existent error = %v, want %v", err, ErrEpisodeNotFound)
	}
}

func TestTVService_UpdateEpisode(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test",
		Seasons: []SeasonInput{
			{SeasonNumber: 1, Episodes: []EpisodeInput{{EpisodeNumber: 1, Title: "Original"}}},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)
	if len(episodes) == 0 {
		t.Fatal("No episodes created")
	}

	newTitle := "Updated Title"
	monitored := false

	updated, err := service.UpdateEpisode(ctx, episodes[0].ID, UpdateEpisodeInput{
		Title:     &newTitle,
		Monitored: &monitored,
	})
	if err != nil {
		t.Fatalf("UpdateEpisode() error = %v", err)
	}

	if updated.Title != newTitle {
		t.Errorf("UpdateEpisode() title = %q, want %q", updated.Title, newTitle)
	}
	if updated.Monitored {
		t.Error("UpdateEpisode() monitored = true, want false")
	}
}

func TestTVService_AddEpisodeFile(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test",
		Seasons: []SeasonInput{
			{SeasonNumber: 1, Episodes: []EpisodeInput{{EpisodeNumber: 1, Title: "Pilot"}}},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)
	if len(episodes) == 0 {
		t.Fatal("No episodes created")
	}

	fileInput := CreateEpisodeFileInput{
		Path:       "/tv/Test/Season 01/Test - S01E01 - Pilot.mkv",
		Size:       500000000,
		Quality:    "HDTV-720p",
		VideoCodec: "x264",
	}

	file, err := service.AddEpisodeFile(ctx, episodes[0].ID, &fileInput)
	if err != nil {
		t.Fatalf("AddEpisodeFile() error = %v", err)
	}

	if file.ID == 0 {
		t.Error("AddEpisodeFile() file.ID = 0, want non-zero")
	}
	if file.Path != fileInput.Path {
		t.Errorf("AddEpisodeFile() path = %q, want %q", file.Path, fileInput.Path)
	}

	// Verify episode now has file
	episode, _ := service.GetEpisode(ctx, episodes[0].ID)
	if episode.EpisodeFile == nil {
		t.Error("Episode should have EpisodeFile set after adding file")
	}
}

func TestTVService_RemoveEpisodeFile(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	series, _ := service.CreateSeries(ctx, &CreateSeriesInput{
		Title: "Test",
		Seasons: []SeasonInput{
			{SeasonNumber: 1, Episodes: []EpisodeInput{{EpisodeNumber: 1, Title: "Pilot"}}},
		},
	})

	episodes, _ := service.ListEpisodes(ctx, series.ID, nil)
	file, _ := service.AddEpisodeFile(ctx, episodes[0].ID, &CreateEpisodeFileInput{Path: "/path.mkv", Size: 1000})

	err := service.RemoveEpisodeFile(ctx, file.ID)
	if err != nil {
		t.Fatalf("RemoveEpisodeFile() error = %v", err)
	}

	// Verify file is gone
	episode, _ := service.GetEpisode(ctx, episodes[0].ID)
	if episode.EpisodeFile != nil {
		t.Error("Episode should have no EpisodeFile after removing file")
	}
}

func TestTVService_RemoveEpisodeFile_NotFound(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	err := service.RemoveEpisodeFile(ctx, 99999)
	if !errors.Is(err, ErrEpisodeFileNotFound) {
		t.Errorf("RemoveEpisodeFile() non-existent error = %v, want %v", err, ErrEpisodeFileNotFound)
	}
}

func TestTVService_Count(t *testing.T) {
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	service := NewService(tdb.Conn, nil, &tdb.Logger)
	ctx := context.Background()

	count, _ := service.Count(ctx)
	if count != 0 {
		t.Errorf("Count() initially = %d, want 0", count)
	}

	_, _ = service.CreateSeries(ctx, &CreateSeriesInput{Title: "Series 1"})
	_, _ = service.CreateSeries(ctx, &CreateSeriesInput{Title: "Series 2"})

	count, _ = service.Count(ctx)
	if count != 2 {
		t.Errorf("Count() = %d, want 2", count)
	}
}

func TestGenerateSortTitle(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"The Office", "Office"},
		{"A Series", "Series"},
		{"An Adventure", "Adventure"},
		{"Breaking Bad", "Breaking Bad"},
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

func TestGenerateSeriesPath(t *testing.T) {
	got := GenerateSeriesPath("/tv", "Breaking Bad")
	want := "/tv/Breaking Bad"

	if got != want {
		t.Errorf("GenerateSeriesPath() = %q, want %q", got, want)
	}
}

func TestGenerateSeasonPath(t *testing.T) {
	got := GenerateSeasonPath("/tv/Breaking Bad", 1)
	want := "/tv/Breaking Bad/Season 01"

	if got != want {
		t.Errorf("GenerateSeasonPath() = %q, want %q", got, want)
	}
}

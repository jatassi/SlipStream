package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/notification/types"
)

func newTestMovie() *types.MediaInfo {
	return &types.MediaInfo{
		ID:         1,
		Title:      "The Matrix",
		Year:       1999,
		TMDbID:     603,
		IMDbID:     "tt0133093",
		TraktID:    481,
		Overview:   "A computer hacker learns about the true nature of reality.",
		PosterURL:  "https://image.tmdb.org/t/p/poster.jpg",
		FanartURL:  "https://image.tmdb.org/t/p/fanart.jpg",
		TrailerURL: "https://youtube.com/watch?v=trailer",
		WebsiteURL: "https://thematrix.com",
		Genres:     []string{"Action", "Sci-Fi"},
		Tags:       []int64{1, 2},
		Rating:     8.7,
	}
}

func newTestEpisode() *types.EpisodeInfo {
	return &types.EpisodeInfo{
		SeriesID:      1,
		SeriesTitle:   "Breaking Bad",
		SeasonNumber:  5,
		EpisodeNumber: 16,
		EpisodeTitle:  "Felina",
		AirDate:       "2013-09-29",
	}
}

func newTestSeries() types.SeriesInfo {
	return types.SeriesInfo{
		MediaInfo: types.MediaInfo{
			ID:         1,
			Title:      "Breaking Bad",
			Year:       2008,
			TMDbID:     1396,
			IMDbID:     "tt0903747",
			Overview:   "A high school chemistry teacher turned methamphetamine manufacturer.",
			PosterURL:  "https://image.tmdb.org/t/p/poster.jpg",
			FanartURL:  "https://image.tmdb.org/t/p/fanart.jpg",
			TrailerURL: "https://youtube.com/watch?v=trailer",
			Genres:     []string{"Drama", "Crime"},
			Tags:       []int64{1},
			Rating:     9.5,
		},
		TVDbID:  81189,
		TraktID: 1388,
	}
}

func newTestRelease() types.ReleaseInfo {
	return types.ReleaseInfo{
		ReleaseName:       "The.Matrix.1999.2160p.UHD.BluRay.x265-GROUP",
		Quality:           "Bluray-2160p",
		QualityVersion:    1,
		Size:              45000000000,
		Indexer:           "TestIndexer",
		ReleaseGroup:      "GROUP",
		SceneName:         "The.Matrix.1999.2160p.UHD.BluRay.x265-GROUP",
		IndexerFlags:      []string{"freeleech"},
		CustomFormats:     []types.CustomFormat{{ID: 1, Name: "HDR"}, {ID: 2, Name: "DV"}},
		CustomFormatScore: 150,
		Languages:         []string{"English", "French"},
	}
}

func newTestDownloadClient() types.DownloadClientInfo {
	return types.DownloadClientInfo{
		ID:         1,
		Name:       "qBittorrent",
		Type:       "qbittorrent",
		DownloadID: "abc123",
	}
}

func newTestMediaInfo() *types.MediaFileInfo {
	return &types.MediaFileInfo{
		VideoCodec:        "x265",
		VideoBitrate:      50000000,
		VideoResolution:   "2160p",
		VideoDynamicRange: "HDR10",
		AudioCodec:        "TrueHD Atmos",
		AudioBitrate:      5000000,
		AudioChannels:     "7.1",
		AudioLanguages:    []string{"English", "French"},
		Subtitles:         []string{"English", "Spanish"},
		Runtime:           136,
		ScanType:          "Progressive",
	}
}

type capturedRequest struct {
	Payload Payload
	Headers http.Header
	Method  string
}

func setupTestServer(t *testing.T, captured *capturedRequest) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Method = r.Method
		captured.Headers = r.Header
		if err := json.NewDecoder(r.Body).Decode(&captured.Payload); err != nil {
			t.Errorf("failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
}

func TestNotifier_Type(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())
	if n.Type() != types.NotifierWebhook {
		t.Errorf("expected type %s, got %s", types.NotifierWebhook, n.Type())
	}
}

func TestNotifier_Name(t *testing.T) {
	n := New("my-webhook", Settings{}, nil, zerolog.Nop())
	if n.Name() != "my-webhook" {
		t.Errorf("expected name 'my-webhook', got %s", n.Name())
	}
}

func TestNotifier_DefaultMethod(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())
	if n.settings.Method != "POST" {
		t.Errorf("expected default method POST, got %s", n.settings.Method)
	}
}

func TestNotifier_Test(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{
		URL:            server.URL,
		ApplicationURL: "http://localhost:8080",
	}, http.DefaultClient, zerolog.Nop())

	if err := n.Test(context.Background()); err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if captured.Payload.EventType != "test" {
		t.Errorf("expected event type 'test', got %s", captured.Payload.EventType)
	}
	if captured.Payload.InstanceName != "SlipStream" {
		t.Errorf("expected instance name 'SlipStream', got %s", captured.Payload.InstanceName)
	}
	if captured.Payload.ApplicationURL != "http://localhost:8080" {
		t.Errorf("expected application URL, got %s", captured.Payload.ApplicationURL)
	}
	if captured.Payload.Message != "Test notification from SlipStream" {
		t.Errorf("expected test message, got %s", captured.Payload.Message)
	}
}

func TestNotifier_CustomMethod(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{
		URL:    server.URL,
		Method: "PUT",
	}, http.DefaultClient, zerolog.Nop())

	if err := n.Test(context.Background()); err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if captured.Method != "PUT" {
		t.Errorf("expected method PUT, got %s", captured.Method)
	}
}

func TestNotifier_BasicAuth(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{
		URL:      server.URL,
		Username: "testuser",
		Password: "testpass",
	}, http.DefaultClient, zerolog.Nop())

	if err := n.Test(context.Background()); err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	auth := captured.Headers.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		t.Errorf("expected Basic auth header, got %s", auth)
	}
}

func TestNotifier_CustomHeaders(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{
		URL: server.URL,
		Headers: map[string]string{
			"X-Custom-Header": "custom-value",
			"X-API-Key":       "secret-key",
		},
	}, http.DefaultClient, zerolog.Nop())

	if err := n.Test(context.Background()); err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if captured.Headers.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("expected custom header, got %s", captured.Headers.Get("X-Custom-Header"))
	}
	if captured.Headers.Get("X-API-Key") != "secret-key" {
		t.Errorf("expected API key header, got %s", captured.Headers.Get("X-API-Key"))
	}
}

func TestNotifier_OnGrab_Movie(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{
		URL:            server.URL,
		ApplicationURL: "http://localhost:8080",
	}, http.DefaultClient, zerolog.Nop())

	event := types.GrabEvent{
		Movie:          newTestMovie(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		DownloadID:     "abc123",
		GrabbedAt:      time.Now(),
	}

	if err := n.OnGrab(context.Background(), event); err != nil {
		t.Fatalf("OnGrab() error = %v", err)
	}

	if captured.Payload.EventType != "grab" {
		t.Errorf("expected event type 'grab', got %s", captured.Payload.EventType)
	}
	if captured.Payload.ApplicationURL != "http://localhost:8080" {
		t.Errorf("expected application URL, got %s", captured.Payload.ApplicationURL)
	}
	if captured.Payload.DownloadID != "abc123" {
		t.Errorf("expected download ID, got %s", captured.Payload.DownloadID)
	}

	// Check movie
	if captured.Payload.Movie == nil {
		t.Fatal("expected movie in payload")
	}
	if captured.Payload.Movie.Title != "The Matrix" {
		t.Errorf("expected movie title, got %s", captured.Payload.Movie.Title)
	}
	if captured.Payload.Movie.Year != 1999 {
		t.Errorf("expected movie year, got %d", captured.Payload.Movie.Year)
	}
	if captured.Payload.Movie.TMDbID != 603 {
		t.Errorf("expected TMDb ID, got %d", captured.Payload.Movie.TMDbID)
	}
	if captured.Payload.Movie.IMDbID != "tt0133093" {
		t.Errorf("expected IMDb ID, got %s", captured.Payload.Movie.IMDbID)
	}
	if captured.Payload.Movie.TraktID != 481 {
		t.Errorf("expected Trakt ID, got %d", captured.Payload.Movie.TraktID)
	}
	if captured.Payload.Movie.Rating != 8.7 {
		t.Errorf("expected rating, got %f", captured.Payload.Movie.Rating)
	}
	if len(captured.Payload.Movie.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(captured.Payload.Movie.Tags))
	}
	if len(captured.Payload.Movie.Images) != 2 {
		t.Errorf("expected 2 images (poster, fanart), got %d", len(captured.Payload.Movie.Images))
	}
	if captured.Payload.Movie.TrailerURL != "https://youtube.com/watch?v=trailer" {
		t.Errorf("expected trailer URL, got %s", captured.Payload.Movie.TrailerURL)
	}
	if captured.Payload.Movie.WebsiteURL != "https://thematrix.com" {
		t.Errorf("expected website URL, got %s", captured.Payload.Movie.WebsiteURL)
	}

	// Check release
	if captured.Payload.Release == nil {
		t.Fatal("expected release in payload")
	}
	if captured.Payload.Release.ReleaseName != event.Release.ReleaseName {
		t.Errorf("expected release name, got %s", captured.Payload.Release.ReleaseName)
	}
	if captured.Payload.Release.Quality != "Bluray-2160p" {
		t.Errorf("expected quality, got %s", captured.Payload.Release.Quality)
	}

	// Check download client
	if captured.Payload.DownloadClient == nil {
		t.Fatal("expected download client in payload")
	}
	if captured.Payload.DownloadClient.Name != "qBittorrent" {
		t.Errorf("expected download client name, got %s", captured.Payload.DownloadClient.Name)
	}

	// Check custom formats
	if len(captured.Payload.CustomFormats) != 2 {
		t.Errorf("expected 2 custom formats, got %d", len(captured.Payload.CustomFormats))
	}
	if captured.Payload.CustomFormats[0].Name != "HDR" {
		t.Errorf("expected custom format name, got %s", captured.Payload.CustomFormats[0].Name)
	}

	// Check languages
	if len(captured.Payload.Languages) != 2 {
		t.Errorf("expected 2 languages, got %d", len(captured.Payload.Languages))
	}
}

func TestNotifier_OnGrab_Episode(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.GrabEvent{
		Episode:        newTestEpisode(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	if err := n.OnGrab(context.Background(), event); err != nil {
		t.Fatalf("OnGrab() error = %v", err)
	}

	if captured.Payload.Episode == nil {
		t.Fatal("expected episode in payload")
	}
	if captured.Payload.Episode.SeriesTitle != "Breaking Bad" {
		t.Errorf("expected series title, got %s", captured.Payload.Episode.SeriesTitle)
	}
	if captured.Payload.Episode.SeasonNumber != 5 {
		t.Errorf("expected season number, got %d", captured.Payload.Episode.SeasonNumber)
	}
	if captured.Payload.Episode.EpisodeNumber != 16 {
		t.Errorf("expected episode number, got %d", captured.Payload.Episode.EpisodeNumber)
	}
	if captured.Payload.Movie != nil {
		t.Error("expected no movie in episode grab")
	}
}

func TestNotifier_OnDownload_Movie(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{
		URL:            server.URL,
		ApplicationURL: "http://localhost:8080",
	}, http.DefaultClient, zerolog.Nop())

	event := types.DownloadEvent{
		Movie:             newTestMovie(),
		Quality:           "Bluray-2160p",
		SourcePath:        "/downloads/movie.mkv",
		DestinationPath:   "/movies/The Matrix (1999)/movie.mkv",
		ReleaseName:       "The.Matrix.1999.2160p.UHD.BluRay.x265-GROUP",
		ReleaseGroup:      "GROUP",
		DownloadID:        "abc123",
		DownloadClient:    "qBittorrent",
		CustomFormats:     []types.CustomFormat{{ID: 1, Name: "HDR"}},
		CustomFormatScore: 100,
		Languages:         []string{"English"},
		MediaInfo:         newTestMediaInfo(),
		ImportedAt:        time.Now(),
	}

	if err := n.OnDownload(context.Background(), event); err != nil {
		t.Fatalf("OnDownload() error = %v", err)
	}

	if captured.Payload.EventType != "download" {
		t.Errorf("expected event type 'download', got %s", captured.Payload.EventType)
	}
	if captured.Payload.Quality != "Bluray-2160p" {
		t.Errorf("expected quality, got %s", captured.Payload.Quality)
	}
	if captured.Payload.SourcePath != event.SourcePath {
		t.Errorf("expected source path, got %s", captured.Payload.SourcePath)
	}
	if captured.Payload.DestinationPath != event.DestinationPath {
		t.Errorf("expected destination path, got %s", captured.Payload.DestinationPath)
	}
	if captured.Payload.IsUpgrade {
		t.Error("expected IsUpgrade to be false for download event")
	}
	if captured.Payload.DownloadID != "abc123" {
		t.Errorf("expected download ID, got %s", captured.Payload.DownloadID)
	}

	// Check media info
	if captured.Payload.MediaInfo == nil {
		t.Fatal("expected media info in payload")
	}
	if captured.Payload.MediaInfo.VideoCodec != "x265" {
		t.Errorf("expected video codec, got %s", captured.Payload.MediaInfo.VideoCodec)
	}
	if captured.Payload.MediaInfo.VideoResolution != "2160p" {
		t.Errorf("expected video resolution, got %s", captured.Payload.MediaInfo.VideoResolution)
	}
	if captured.Payload.MediaInfo.VideoDynamicRange != "HDR10" {
		t.Errorf("expected dynamic range, got %s", captured.Payload.MediaInfo.VideoDynamicRange)
	}
	if captured.Payload.MediaInfo.AudioCodec != "TrueHD Atmos" {
		t.Errorf("expected audio codec, got %s", captured.Payload.MediaInfo.AudioCodec)
	}
	if captured.Payload.MediaInfo.AudioChannels != "7.1" {
		t.Errorf("expected audio channels, got %s", captured.Payload.MediaInfo.AudioChannels)
	}
	if len(captured.Payload.MediaInfo.AudioLanguages) != 2 {
		t.Errorf("expected 2 audio languages, got %d", len(captured.Payload.MediaInfo.AudioLanguages))
	}
	if len(captured.Payload.MediaInfo.Subtitles) != 2 {
		t.Errorf("expected 2 subtitles, got %d", len(captured.Payload.MediaInfo.Subtitles))
	}
}

func TestNotifier_OnUpgrade(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.UpgradeEvent{
		Movie:             newTestMovie(),
		OldQuality:        "Bluray-1080p",
		NewQuality:        "Bluray-2160p",
		OldPath:           "/movies/old.mkv",
		NewPath:           "/movies/new.mkv",
		ReleaseGroup:      "GROUP",
		CustomFormats:     []types.CustomFormat{{ID: 1, Name: "HDR"}},
		CustomFormatScore: 100,
		Languages:         []string{"English"},
		MediaInfo:         newTestMediaInfo(),
		UpgradedAt:        time.Now(),
	}

	if err := n.OnUpgrade(context.Background(), event); err != nil {
		t.Fatalf("OnUpgrade() error = %v", err)
	}

	if captured.Payload.EventType != "upgrade" {
		t.Errorf("expected event type 'upgrade', got %s", captured.Payload.EventType)
	}
	if captured.Payload.OldQuality != "Bluray-1080p" {
		t.Errorf("expected old quality, got %s", captured.Payload.OldQuality)
	}
	if captured.Payload.Quality != "Bluray-2160p" {
		t.Errorf("expected new quality, got %s", captured.Payload.Quality)
	}
	if !captured.Payload.IsUpgrade {
		t.Error("expected IsUpgrade to be true")
	}
}

func TestNotifier_OnMovieAdded(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.MovieAddedEvent{
		Movie:   *newTestMovie(),
		AddedAt: time.Now(),
	}

	if err := n.OnMovieAdded(context.Background(), event); err != nil {
		t.Fatalf("OnMovieAdded() error = %v", err)
	}

	if captured.Payload.EventType != "movieAdded" {
		t.Errorf("expected event type 'movieAdded', got %s", captured.Payload.EventType)
	}
	if captured.Payload.Movie == nil {
		t.Fatal("expected movie in payload")
	}
	if captured.Payload.Movie.Title != "The Matrix" {
		t.Errorf("expected movie title, got %s", captured.Payload.Movie.Title)
	}
}

func TestNotifier_OnMovieDeleted(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.MovieDeletedEvent{
		Movie:        *newTestMovie(),
		DeletedFiles: true,
		DeletedAt:    time.Now(),
	}

	if err := n.OnMovieDeleted(context.Background(), event); err != nil {
		t.Fatalf("OnMovieDeleted() error = %v", err)
	}

	if captured.Payload.EventType != "movieDeleted" {
		t.Errorf("expected event type 'movieDeleted', got %s", captured.Payload.EventType)
	}
	if !captured.Payload.DeletedFiles {
		t.Error("expected DeletedFiles to be true")
	}
}

func TestNotifier_OnSeriesAdded(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.SeriesAddedEvent{
		Series:  newTestSeries(),
		AddedAt: time.Now(),
	}

	if err := n.OnSeriesAdded(context.Background(), event); err != nil {
		t.Fatalf("OnSeriesAdded() error = %v", err)
	}

	if captured.Payload.EventType != "seriesAdded" {
		t.Errorf("expected event type 'seriesAdded', got %s", captured.Payload.EventType)
	}
	if captured.Payload.Series == nil {
		t.Fatal("expected series in payload")
	}
	if captured.Payload.Series.Title != "Breaking Bad" {
		t.Errorf("expected series title, got %s", captured.Payload.Series.Title)
	}
	if captured.Payload.Series.TVDbID != 81189 {
		t.Errorf("expected TVDb ID, got %d", captured.Payload.Series.TVDbID)
	}
	if captured.Payload.Series.TraktID != 1388 {
		t.Errorf("expected Trakt ID, got %d", captured.Payload.Series.TraktID)
	}
	if len(captured.Payload.Series.Images) != 2 {
		t.Errorf("expected 2 images, got %d", len(captured.Payload.Series.Images))
	}
}

func TestNotifier_OnSeriesDeleted(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.SeriesDeletedEvent{
		Series:       newTestSeries(),
		DeletedFiles: true,
		DeletedAt:    time.Now(),
	}

	if err := n.OnSeriesDeleted(context.Background(), event); err != nil {
		t.Fatalf("OnSeriesDeleted() error = %v", err)
	}

	if captured.Payload.EventType != "seriesDeleted" {
		t.Errorf("expected event type 'seriesDeleted', got %s", captured.Payload.EventType)
	}
	if !captured.Payload.DeletedFiles {
		t.Error("expected DeletedFiles to be true")
	}
}

func TestNotifier_OnHealthIssue(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "error",
		Message:   "Connection failed",
		WikiURL:   "https://wiki.example.com/indexer-error",
		OccuredAt: time.Now(),
	}

	if err := n.OnHealthIssue(context.Background(), event); err != nil {
		t.Fatalf("OnHealthIssue() error = %v", err)
	}

	if captured.Payload.EventType != "healthIssue" {
		t.Errorf("expected event type 'healthIssue', got %s", captured.Payload.EventType)
	}
	if captured.Payload.Health == nil {
		t.Fatal("expected health in payload")
	}
	if captured.Payload.Health.Source != "Indexer" {
		t.Errorf("expected source, got %s", captured.Payload.Health.Source)
	}
	if captured.Payload.Health.Type != "error" {
		t.Errorf("expected type, got %s", captured.Payload.Health.Type)
	}
	if captured.Payload.Health.Message != "Connection failed" {
		t.Errorf("expected message, got %s", captured.Payload.Health.Message)
	}
	if captured.Payload.Health.WikiURL != "https://wiki.example.com/indexer-error" {
		t.Errorf("expected wiki URL, got %s", captured.Payload.Health.WikiURL)
	}
}

func TestNotifier_OnHealthRestored(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "warning",
		Message:   "Connection restored",
		OccuredAt: time.Now(),
	}

	if err := n.OnHealthRestored(context.Background(), event); err != nil {
		t.Fatalf("OnHealthRestored() error = %v", err)
	}

	if captured.Payload.EventType != "healthRestored" {
		t.Errorf("expected event type 'healthRestored', got %s", captured.Payload.EventType)
	}
	if captured.Payload.Health.WikiURL != "" {
		t.Errorf("expected no wiki URL for restored, got %s", captured.Payload.Health.WikiURL)
	}
}

func TestNotifier_OnApplicationUpdate(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.AppUpdateEvent{
		PreviousVersion: "1.0.0",
		NewVersion:      "1.1.0",
		UpdatedAt:       time.Now(),
	}

	if err := n.OnApplicationUpdate(context.Background(), event); err != nil {
		t.Fatalf("OnApplicationUpdate() error = %v", err)
	}

	if captured.Payload.EventType != "applicationUpdate" {
		t.Errorf("expected event type 'applicationUpdate', got %s", captured.Payload.EventType)
	}
	if captured.Payload.PreviousVersion != "1.0.0" {
		t.Errorf("expected previous version, got %s", captured.Payload.PreviousVersion)
	}
	if captured.Payload.NewVersion != "1.1.0" {
		t.Errorf("expected new version, got %s", captured.Payload.NewVersion)
	}
}

func TestNotifier_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	err := n.Test(context.Background())
	if err == nil {
		t.Error("expected error for HTTP 400 response")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to contain status code, got %v", err)
	}
}

func TestNotifier_ContentType(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	if err := n.Test(context.Background()); err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	ct := captured.Headers.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

func TestNotifier_MapMovieImages(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())

	movie := newTestMovie()
	mapped := n.mapMediaInfo(movie)

	if len(mapped.Images) != 2 {
		t.Errorf("expected 2 images, got %d", len(mapped.Images))
	}

	hasPoster := false
	hasFanart := false
	for _, img := range mapped.Images {
		if img.CoverType == "poster" {
			hasPoster = true
			if img.URL != movie.PosterURL {
				t.Errorf("expected poster URL, got %s", img.URL)
			}
		}
		if img.CoverType == "fanart" {
			hasFanart = true
			if img.URL != movie.FanartURL {
				t.Errorf("expected fanart URL, got %s", img.URL)
			}
		}
	}

	if !hasPoster {
		t.Error("expected poster image")
	}
	if !hasFanart {
		t.Error("expected fanart image")
	}
}

func TestNotifier_MapSeriesImages(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())

	series := newTestSeries()
	mapped := n.mapSeriesInfo(&series)

	if len(mapped.Images) != 2 {
		t.Errorf("expected 2 images, got %d", len(mapped.Images))
	}
}

func TestNotifier_MapCustomFormats(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())

	formats := []types.CustomFormat{
		{ID: 1, Name: "HDR"},
		{ID: 2, Name: "DV"},
		{ID: 3, Name: "Atmos"},
	}

	mapped := n.mapCustomFormats(formats)

	if len(mapped) != 3 {
		t.Errorf("expected 3 custom formats, got %d", len(mapped))
	}
	if mapped[0].ID != 1 || mapped[0].Name != "HDR" {
		t.Errorf("expected first format HDR, got %v", mapped[0])
	}
}

func TestNotifier_MapCustomFormats_Empty(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())

	mapped := n.mapCustomFormats(nil)

	if mapped != nil {
		t.Error("expected nil for empty custom formats")
	}
}

func TestNotifier_MapMediaFileInfo_Nil(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())

	mapped := n.mapMediaFileInfo(nil)

	if mapped != nil {
		t.Error("expected nil for nil media info")
	}
}

func TestNotifier_MapEpisode_Nil(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())

	mapped := n.mapEpisode(nil)

	if mapped != nil {
		t.Error("expected nil for nil episode")
	}
}

func TestNotifier_MapMovie_Nil(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())

	mapped := n.mapMovie(nil)

	if mapped != nil {
		t.Error("expected nil for nil movie")
	}
}

func TestNotifier_Timestamp(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{URL: server.URL}, http.DefaultClient, zerolog.Nop())

	eventTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	event := types.GrabEvent{
		Movie:          newTestMovie(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      eventTime,
	}

	if err := n.OnGrab(context.Background(), event); err != nil {
		t.Fatalf("OnGrab() error = %v", err)
	}

	if !captured.Payload.Timestamp.Equal(eventTime) {
		t.Errorf("expected timestamp %v, got %v", eventTime, captured.Payload.Timestamp)
	}
}

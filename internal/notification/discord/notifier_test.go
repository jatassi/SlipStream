package discord

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

func newTestLogger() *zerolog.Logger {
	logger := zerolog.Nop()
	return &logger
}

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
	Payload WebhookPayload
	Headers http.Header
}

func setupTestServer(t *testing.T, captured *capturedRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		captured.Headers = r.Header
		if err := json.NewDecoder(r.Body).Decode(&captured.Payload); err != nil {
			t.Errorf("failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
}

func TestNotifier_Type(t *testing.T) {
	n := New("test", &Settings{}, nil, newTestLogger())
	if n.Type() != types.NotifierDiscord {
		t.Errorf("expected type %s, got %s", types.NotifierDiscord, n.Type())
	}
}

func TestNotifier_Name(t *testing.T) {
	n := New("my-notifier", &Settings{}, nil, newTestLogger())
	if n.Name() != "my-notifier" {
		t.Errorf("expected name 'my-notifier', got %s", n.Name())
	}
}

func TestNotifier_DefaultFields(t *testing.T) {
	n := New("test", &Settings{}, nil, newTestLogger())
	if len(n.settings.GrabFields) != len(DefaultGrabFields) {
		t.Errorf("expected %d grab fields, got %d", len(DefaultGrabFields), len(n.settings.GrabFields))
	}
	if len(n.settings.ImportFields) != len(DefaultImportFields) {
		t.Errorf("expected %d import fields, got %d", len(DefaultImportFields), len(n.settings.ImportFields))
	}
}

func TestNotifier_Test(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{
		WebhookURL: server.URL,
		Username:   "TestBot",
		AvatarURL:  "https://example.com/avatar.png",
	}, http.DefaultClient, newTestLogger())

	if err := n.Test(context.Background()); err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if captured.Payload.Username != "TestBot" {
		t.Errorf("expected username 'TestBot', got %s", captured.Payload.Username)
	}
	if captured.Payload.AvatarURL != "https://example.com/avatar.png" {
		t.Errorf("expected avatar URL, got %s", captured.Payload.AvatarURL)
	}
	if len(captured.Payload.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(captured.Payload.Embeds))
	}
	if captured.Payload.Embeds[0].Title != "SlipStream Test Notification" {
		t.Errorf("expected test title, got %s", captured.Payload.Embeds[0].Title)
	}
	if captured.Payload.Embeds[0].Color != ColorInfo {
		t.Errorf("expected color %d, got %d", ColorInfo, captured.Payload.Embeds[0].Color)
	}
}

func TestNotifier_OnGrab_Movie(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{
		WebhookURL: server.URL,
		GrabFields: []FieldType{FieldQuality, FieldIndexer, FieldDownloadClient, FieldSize, FieldReleaseGroup, FieldCustomFormats, FieldLanguages, FieldLinks, FieldPoster, FieldFanart},
	}, http.DefaultClient, newTestLogger())

	event := types.GrabEvent{
		Movie:          newTestMovie(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		DownloadID:     "abc123",
		GrabbedAt:      time.Now(),
	}

	if err := n.OnGrab(context.Background(), &event); err != nil {
		t.Fatalf("OnGrab() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if !strings.Contains(embed.Title, "Movie Grabbed") {
		t.Errorf("expected title to contain 'Movie Grabbed', got %s", embed.Title)
	}
	if !strings.Contains(embed.Title, "The Matrix") {
		t.Errorf("expected title to contain movie name, got %s", embed.Title)
	}
	if !strings.Contains(embed.Title, "1999") {
		t.Errorf("expected title to contain year, got %s", embed.Title)
	}
	if !strings.Contains(embed.Description, event.Release.ReleaseName) {
		t.Errorf("expected description to contain release name")
	}
	if embed.Color != ColorDefault {
		t.Errorf("expected color %d, got %d", ColorDefault, embed.Color)
	}
	if embed.Thumbnail == nil || embed.Thumbnail.URL != event.Movie.PosterURL {
		t.Error("expected thumbnail with poster URL")
	}
	if embed.Image == nil || embed.Image.URL != event.Movie.FanartURL {
		t.Error("expected image with fanart URL")
	}

	fieldNames := make(map[string]string)
	for _, f := range embed.Fields {
		fieldNames[f.Name] = f.Value
	}

	if fieldNames["Quality"] != "Bluray-2160p" {
		t.Errorf("expected quality field, got %s", fieldNames["Quality"])
	}
	if fieldNames["Indexer"] != "TestIndexer" {
		t.Errorf("expected indexer field, got %s", fieldNames["Indexer"])
	}
	if fieldNames["Download Client"] != "qBittorrent" {
		t.Errorf("expected download client field, got %s", fieldNames["Download Client"])
	}
	if !strings.Contains(fieldNames["Size"], "41.9") {
		t.Errorf("expected size field with formatted size, got %s", fieldNames["Size"])
	}
	if fieldNames["Release Group"] != "GROUP" {
		t.Errorf("expected release group field, got %s", fieldNames["Release Group"])
	}
	if !strings.Contains(fieldNames["Custom Formats"], "HDR") {
		t.Errorf("expected custom formats to contain HDR, got %s", fieldNames["Custom Formats"])
	}
	if !strings.Contains(fieldNames["Custom Formats"], "150") {
		t.Errorf("expected custom formats score, got %s", fieldNames["Custom Formats"])
	}
	if !strings.Contains(fieldNames["Languages"], "English") {
		t.Errorf("expected languages field, got %s", fieldNames["Languages"])
	}
	if !strings.Contains(fieldNames["Links"], "TMDb") {
		t.Errorf("expected TMDb link, got %s", fieldNames["Links"])
	}
	if !strings.Contains(fieldNames["Links"], "IMDb") {
		t.Errorf("expected IMDb link, got %s", fieldNames["Links"])
	}
	if !strings.Contains(fieldNames["Links"], "Trakt") {
		t.Errorf("expected Trakt link, got %s", fieldNames["Links"])
	}
	if !strings.Contains(fieldNames["Links"], "Trailer") {
		t.Errorf("expected Trailer link, got %s", fieldNames["Links"])
	}
}

func TestNotifier_OnGrab_Episode(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{WebhookURL: server.URL}, http.DefaultClient, newTestLogger())

	event := types.GrabEvent{
		Episode:        newTestEpisode(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	if err := n.OnGrab(context.Background(), &event); err != nil {
		t.Fatalf("OnGrab() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if !strings.Contains(embed.Title, "Episode Grabbed") {
		t.Errorf("expected title to contain 'Episode Grabbed', got %s", embed.Title)
	}
	if !strings.Contains(embed.Title, "Breaking Bad") {
		t.Errorf("expected title to contain series name, got %s", embed.Title)
	}
	if !strings.Contains(embed.Title, "S05E16") {
		t.Errorf("expected title to contain season/episode, got %s", embed.Title)
	}
}

func TestNotifier_OnImport_Movie(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{
		WebhookURL:   server.URL,
		ImportFields: []FieldType{FieldQuality, FieldReleaseGroup, FieldCustomFormats, FieldLanguages, FieldMediaInfo, FieldLinks, FieldPoster},
	}, http.DefaultClient, newTestLogger())

	event := types.ImportEvent{
		Movie:             newTestMovie(),
		Quality:           "Bluray-2160p",
		SourcePath:        "/downloads/movie.mkv",
		DestinationPath:   "/movies/The Matrix (1999)/movie.mkv",
		ReleaseName:       "The.Matrix.1999.2160p.UHD.BluRay.x265-GROUP",
		ReleaseGroup:      "GROUP",
		DownloadID:        "abc123",
		DownloadClient:    "qBittorrent",
		CustomFormats:     []types.CustomFormat{{ID: 1, Name: "HDR"}, {ID: 2, Name: "DV"}},
		CustomFormatScore: 150,
		Languages:         []string{"English", "French"},
		MediaInfo:         newTestMediaInfo(),
		ImportedAt:        time.Now(),
	}

	if err := n.OnImport(context.Background(), &event); err != nil {
		t.Fatalf("OnImport() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if !strings.Contains(embed.Title, "Movie Downloaded") {
		t.Errorf("expected title to contain 'Movie Downloaded', got %s", embed.Title)
	}
	if embed.Color != ColorSuccess {
		t.Errorf("expected color %d, got %d", ColorSuccess, embed.Color)
	}

	fieldNames := make(map[string]string)
	for _, f := range embed.Fields {
		fieldNames[f.Name] = f.Value
	}

	if !strings.Contains(fieldNames["Media Info"], "2160p") {
		t.Errorf("expected media info to contain resolution, got %s", fieldNames["Media Info"])
	}
	if !strings.Contains(fieldNames["Media Info"], "x265") {
		t.Errorf("expected media info to contain codec, got %s", fieldNames["Media Info"])
	}
	if !strings.Contains(fieldNames["Media Info"], "TrueHD Atmos") {
		t.Errorf("expected media info to contain audio codec, got %s", fieldNames["Media Info"])
	}
}

func TestNotifier_OnImport_Episode(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{WebhookURL: server.URL}, http.DefaultClient, newTestLogger())

	event := types.ImportEvent{
		Episode:    newTestEpisode(),
		Quality:    "HDTV-1080p",
		ImportedAt: time.Now(),
	}

	if err := n.OnImport(context.Background(), &event); err != nil {
		t.Fatalf("OnImport() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if !strings.Contains(embed.Title, "Episode Downloaded") {
		t.Errorf("expected title to contain 'Episode Downloaded', got %s", embed.Title)
	}
	if !strings.Contains(embed.Title, "S05E16") {
		t.Errorf("expected title to contain season/episode, got %s", embed.Title)
	}
}

func TestNotifier_OnUpgrade_Movie(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{
		WebhookURL:   server.URL,
		ImportFields: []FieldType{FieldQuality, FieldReleaseGroup, FieldCustomFormats, FieldLanguages, FieldMediaInfo, FieldLinks},
	}, http.DefaultClient, newTestLogger())

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

	if err := n.OnUpgrade(context.Background(), &event); err != nil {
		t.Fatalf("OnUpgrade() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if !strings.Contains(embed.Title, "Movie Upgraded") {
		t.Errorf("expected title to contain 'Movie Upgraded', got %s", embed.Title)
	}
	if embed.Color != ColorInfo {
		t.Errorf("expected color %d, got %d", ColorInfo, embed.Color)
	}

	fieldNames := make(map[string]string)
	for _, f := range embed.Fields {
		fieldNames[f.Name] = f.Value
	}

	if fieldNames["Old Quality"] != "Bluray-1080p" {
		t.Errorf("expected old quality, got %s", fieldNames["Old Quality"])
	}
	if fieldNames["New Quality"] != "Bluray-2160p" {
		t.Errorf("expected new quality, got %s", fieldNames["New Quality"])
	}
}

func TestNotifier_OnUpgrade_Episode(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{WebhookURL: server.URL}, http.DefaultClient, newTestLogger())

	event := types.UpgradeEvent{
		Episode:    newTestEpisode(),
		OldQuality: "HDTV-720p",
		NewQuality: "Bluray-1080p",
		UpgradedAt: time.Now(),
	}

	if err := n.OnUpgrade(context.Background(), &event); err != nil {
		t.Fatalf("OnUpgrade() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if !strings.Contains(embed.Title, "Episode Upgraded") {
		t.Errorf("expected title to contain 'Episode Upgraded', got %s", embed.Title)
	}
}

func TestNotifier_OnMovieAdded(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{
		WebhookURL:   server.URL,
		ImportFields: []FieldType{FieldOverview, FieldRating, FieldGenres, FieldLinks, FieldPoster, FieldFanart},
	}, http.DefaultClient, newTestLogger())

	event := types.MovieAddedEvent{
		Movie:   *newTestMovie(),
		AddedAt: time.Now(),
	}

	if err := n.OnMovieAdded(context.Background(), &event); err != nil {
		t.Fatalf("OnMovieAdded() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if !strings.Contains(embed.Title, "Movie Added") {
		t.Errorf("expected title to contain 'Movie Added', got %s", embed.Title)
	}
	if !strings.Contains(embed.Title, "The Matrix") {
		t.Errorf("expected title to contain movie name, got %s", embed.Title)
	}
	if embed.Color != ColorSuccess {
		t.Errorf("expected color %d, got %d", ColorSuccess, embed.Color)
	}
	if !strings.Contains(embed.Description, "computer hacker") {
		t.Errorf("expected description to contain overview, got %s", embed.Description)
	}
	if embed.Thumbnail == nil {
		t.Error("expected thumbnail")
	}
	if embed.Image == nil {
		t.Error("expected image")
	}

	fieldNames := make(map[string]string)
	for _, f := range embed.Fields {
		fieldNames[f.Name] = f.Value
	}

	if !strings.Contains(fieldNames["Rating"], "8.7") {
		t.Errorf("expected rating, got %s", fieldNames["Rating"])
	}
	if !strings.Contains(fieldNames["Genres"], "Action") {
		t.Errorf("expected genres, got %s", fieldNames["Genres"])
	}
}

func TestNotifier_OnMovieDeleted(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{WebhookURL: server.URL}, http.DefaultClient, newTestLogger())

	event := types.MovieDeletedEvent{
		Movie:        *newTestMovie(),
		DeletedFiles: true,
		DeletedAt:    time.Now(),
	}

	if err := n.OnMovieDeleted(context.Background(), &event); err != nil {
		t.Fatalf("OnMovieDeleted() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if !strings.Contains(embed.Title, "Movie Deleted") {
		t.Errorf("expected title to contain 'Movie Deleted', got %s", embed.Title)
	}
	if embed.Color != ColorDanger {
		t.Errorf("expected color %d, got %d", ColorDanger, embed.Color)
	}
	if !strings.Contains(embed.Description, "files deleted") {
		t.Errorf("expected description to mention files deleted, got %s", embed.Description)
	}
}

func TestNotifier_OnMovieDeleted_NoFiles(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{WebhookURL: server.URL}, http.DefaultClient, newTestLogger())

	event := types.MovieDeletedEvent{
		Movie:        *newTestMovie(),
		DeletedFiles: false,
		DeletedAt:    time.Now(),
	}

	if err := n.OnMovieDeleted(context.Background(), &event); err != nil {
		t.Fatalf("OnMovieDeleted() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if strings.Contains(embed.Description, "files deleted") {
		t.Errorf("expected description NOT to mention files deleted, got %s", embed.Description)
	}
}

func TestNotifier_OnSeriesAdded(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{
		WebhookURL:   server.URL,
		ImportFields: []FieldType{FieldOverview, FieldRating, FieldGenres, FieldLinks, FieldPoster},
	}, http.DefaultClient, newTestLogger())

	event := types.SeriesAddedEvent{
		Series:  newTestSeries(),
		AddedAt: time.Now(),
	}

	if err := n.OnSeriesAdded(context.Background(), &event); err != nil {
		t.Fatalf("OnSeriesAdded() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if !strings.Contains(embed.Title, "Series Added") {
		t.Errorf("expected title to contain 'Series Added', got %s", embed.Title)
	}
	if !strings.Contains(embed.Title, "Breaking Bad") {
		t.Errorf("expected title to contain series name, got %s", embed.Title)
	}
	if embed.Color != ColorSuccess {
		t.Errorf("expected color %d, got %d", ColorSuccess, embed.Color)
	}

	fieldNames := make(map[string]string)
	for _, f := range embed.Fields {
		fieldNames[f.Name] = f.Value
	}

	if !strings.Contains(fieldNames["Links"], "TVDb") {
		t.Errorf("expected TVDb link for series, got %s", fieldNames["Links"])
	}
	if !strings.Contains(fieldNames["Links"], "Trakt") {
		t.Errorf("expected Trakt link for series, got %s", fieldNames["Links"])
	}
}

func TestNotifier_OnSeriesDeleted(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{WebhookURL: server.URL}, http.DefaultClient, newTestLogger())

	event := types.SeriesDeletedEvent{
		Series:       newTestSeries(),
		DeletedFiles: true,
		DeletedAt:    time.Now(),
	}

	if err := n.OnSeriesDeleted(context.Background(), &event); err != nil {
		t.Fatalf("OnSeriesDeleted() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if !strings.Contains(embed.Title, "Series Deleted") {
		t.Errorf("expected title to contain 'Series Deleted', got %s", embed.Title)
	}
	if embed.Color != ColorDanger {
		t.Errorf("expected color %d, got %d", ColorDanger, embed.Color)
	}
	if !strings.Contains(embed.Description, "files deleted") {
		t.Errorf("expected description to mention files deleted, got %s", embed.Description)
	}
}

func TestNotifier_OnHealthIssue_Warning(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{WebhookURL: server.URL}, http.DefaultClient, newTestLogger())

	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "warning",
		Message:   "Indexer is unreachable",
		WikiURL:   "https://wiki.example.com/health",
		OccuredAt: time.Now(),
	}

	if err := n.OnHealthIssue(context.Background(), &event); err != nil {
		t.Fatalf("OnHealthIssue() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if embed.Title != "Health Issue" {
		t.Errorf("expected title 'Health Issue', got %s", embed.Title)
	}
	if embed.Color != ColorWarning {
		t.Errorf("expected color %d for warning, got %d", ColorWarning, embed.Color)
	}
	if embed.Description != event.Message {
		t.Errorf("expected message in description, got %s", embed.Description)
	}

	fieldNames := make(map[string]string)
	for _, f := range embed.Fields {
		fieldNames[f.Name] = f.Value
	}

	if fieldNames["Source"] != "Indexer" {
		t.Errorf("expected source field, got %s", fieldNames["Source"])
	}
	if fieldNames["Type"] != "warning" {
		t.Errorf("expected type field, got %s", fieldNames["Type"])
	}
}

func TestNotifier_OnHealthIssue_Error(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{WebhookURL: server.URL}, http.DefaultClient, newTestLogger())

	event := types.HealthEvent{
		Source:    "Database",
		Type:      "error",
		Message:   "Database connection failed",
		OccuredAt: time.Now(),
	}

	if err := n.OnHealthIssue(context.Background(), &event); err != nil {
		t.Fatalf("OnHealthIssue() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if embed.Color != ColorDanger {
		t.Errorf("expected color %d for error, got %d", ColorDanger, embed.Color)
	}
}

func TestNotifier_OnHealthRestored(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{WebhookURL: server.URL}, http.DefaultClient, newTestLogger())

	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "warning",
		Message:   "Indexer is now reachable",
		OccuredAt: time.Now(),
	}

	if err := n.OnHealthRestored(context.Background(), &event); err != nil {
		t.Fatalf("OnHealthRestored() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if embed.Title != "Health Issue Resolved" {
		t.Errorf("expected title 'Health Issue Resolved', got %s", embed.Title)
	}
	if embed.Color != ColorSuccess {
		t.Errorf("expected color %d, got %d", ColorSuccess, embed.Color)
	}
}

func TestNotifier_OnApplicationUpdate(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{WebhookURL: server.URL}, http.DefaultClient, newTestLogger())

	event := types.AppUpdateEvent{
		PreviousVersion: "1.0.0",
		NewVersion:      "1.1.0",
		UpdatedAt:       time.Now(),
	}

	if err := n.OnApplicationUpdate(context.Background(), &event); err != nil {
		t.Fatalf("OnApplicationUpdate() error = %v", err)
	}

	embed := captured.Payload.Embeds[0]
	if embed.Title != "Application Updated" {
		t.Errorf("expected title 'Application Updated', got %s", embed.Title)
	}
	if embed.Color != ColorInfo {
		t.Errorf("expected color %d, got %d", ColorInfo, embed.Color)
	}

	fieldNames := make(map[string]string)
	for _, f := range embed.Fields {
		fieldNames[f.Name] = f.Value
	}

	if fieldNames["Previous Version"] != "1.0.0" {
		t.Errorf("expected previous version, got %s", fieldNames["Previous Version"])
	}
	if fieldNames["New Version"] != "1.1.0" {
		t.Errorf("expected new version, got %s", fieldNames["New Version"])
	}
}

func TestNotifier_CustomSettings(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", &Settings{
		WebhookURL: server.URL,
		Username:   "CustomBot",
		AvatarURL:  "https://example.com/custom.png",
		Author:     "CustomAuthor",
	}, http.DefaultClient, newTestLogger())

	// Use OnGrab to test custom settings since Test() doesn't use buildPayload()
	event := types.GrabEvent{
		Movie:   newTestMovie(),
		Release: newTestRelease(),
	}
	if err := n.OnGrab(context.Background(), &event); err != nil {
		t.Fatalf("OnGrab() error = %v", err)
	}

	if captured.Payload.Username != "CustomBot" {
		t.Errorf("expected custom username, got %s", captured.Payload.Username)
	}
	if captured.Payload.AvatarURL != "https://example.com/custom.png" {
		t.Errorf("expected custom avatar URL, got %s", captured.Payload.AvatarURL)
	}
	if len(captured.Payload.Embeds) == 0 {
		t.Fatal("expected at least one embed")
	}
	if captured.Payload.Embeds[0].Author == nil {
		t.Fatal("expected author to be set")
	}
	if captured.Payload.Embeds[0].Author.Name != "CustomAuthor" {
		t.Errorf("expected custom author, got %s", captured.Payload.Embeds[0].Author.Name)
	}
}

func TestNotifier_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	n := New("test", &Settings{WebhookURL: server.URL}, http.DefaultClient, newTestLogger())

	err := n.Test(context.Background())
	if err == nil {
		t.Error("expected error for HTTP 400 response")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to contain status code, got %v", err)
	}
}

func TestNotifier_ConfigurableFields(t *testing.T) {
	tests := []struct {
		name        string
		grabFields  []FieldType
		expectField string
		dontExpect  string
	}{
		{
			name:        "only quality",
			grabFields:  []FieldType{FieldQuality},
			expectField: "Quality",
			dontExpect:  "Indexer",
		},
		{
			name:        "only indexer",
			grabFields:  []FieldType{FieldIndexer},
			expectField: "Indexer",
			dontExpect:  "Quality",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured capturedRequest
			server := setupTestServer(t, &captured)
			defer server.Close()

			n := New("test", &Settings{
				WebhookURL: server.URL,
				GrabFields: tt.grabFields,
			}, http.DefaultClient, newTestLogger())

			event := types.GrabEvent{
				Movie:          newTestMovie(),
				Release:        newTestRelease(),
				DownloadClient: newTestDownloadClient(),
				GrabbedAt:      time.Now(),
			}

			if err := n.OnGrab(context.Background(), &event); err != nil {
				t.Fatalf("OnGrab() error = %v", err)
			}

			fieldNames := make(map[string]bool)
			for _, f := range captured.Payload.Embeds[0].Fields {
				fieldNames[f.Name] = true
			}

			if !fieldNames[tt.expectField] {
				t.Errorf("expected field %s to be present", tt.expectField)
			}
			if fieldNames[tt.dontExpect] {
				t.Errorf("expected field %s NOT to be present", tt.dontExpect)
			}
		})
	}
}

func TestNotifier_BuildLinks(t *testing.T) {
	n := New("test", &Settings{}, http.DefaultClient, newTestLogger())

	movie := newTestMovie()
	links := n.buildLinks(movie)

	if !strings.Contains(links, "TMDb") {
		t.Error("expected TMDb link")
	}
	if !strings.Contains(links, "IMDb") {
		t.Error("expected IMDb link")
	}
	if !strings.Contains(links, "Trakt") {
		t.Error("expected Trakt link")
	}
	if !strings.Contains(links, "Trailer") {
		t.Error("expected Trailer link")
	}
	if !strings.Contains(links, "Website") {
		t.Error("expected Website link")
	}
	if !strings.Contains(links, "themoviedb.org/movie/603") {
		t.Errorf("expected correct TMDb URL, got %s", links)
	}
	if !strings.Contains(links, "imdb.com/title/tt0133093") {
		t.Errorf("expected correct IMDb URL, got %s", links)
	}
}

func TestNotifier_BuildSeriesLinks(t *testing.T) {
	n := New("test", &Settings{}, http.DefaultClient, newTestLogger())

	series := newTestSeries()
	links := n.buildSeriesLinks(&series)

	if !strings.Contains(links, "TMDb") {
		t.Error("expected TMDb link")
	}
	if !strings.Contains(links, "TVDb") {
		t.Error("expected TVDb link")
	}
	if !strings.Contains(links, "Trakt") {
		t.Error("expected Trakt link")
	}
	if !strings.Contains(links, "themoviedb.org/tv/1396") {
		t.Errorf("expected correct TMDb TV URL, got %s", links)
	}
	if !strings.Contains(links, "thetvdb.com/series/81189") {
		t.Errorf("expected correct TVDb URL, got %s", links)
	}
}

func TestNotifier_FormatMediaInfo(t *testing.T) {
	n := New("test", &Settings{}, http.DefaultClient, newTestLogger())

	mi := newTestMediaInfo()
	formatted := n.formatMediaInfo(mi)

	if !strings.Contains(formatted, "2160p") {
		t.Errorf("expected resolution in media info, got %s", formatted)
	}
	if !strings.Contains(formatted, "x265") {
		t.Errorf("expected video codec in media info, got %s", formatted)
	}
	if !strings.Contains(formatted, "HDR10") {
		t.Errorf("expected dynamic range in media info, got %s", formatted)
	}
	if !strings.Contains(formatted, "TrueHD Atmos") {
		t.Errorf("expected audio codec in media info, got %s", formatted)
	}
	if !strings.Contains(formatted, "7.1") {
		t.Errorf("expected audio channels in media info, got %s", formatted)
	}
}

func TestNotifier_FormatCustomFormats(t *testing.T) {
	n := New("test", &Settings{}, http.DefaultClient, newTestLogger())

	formats := []types.CustomFormat{{ID: 1, Name: "HDR"}, {ID: 2, Name: "DV"}, {ID: 3, Name: "Atmos"}}
	formatted := n.formatCustomFormats(formats)

	if formatted != "HDR, DV, Atmos" {
		t.Errorf("expected 'HDR, DV, Atmos', got %s", formatted)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is..."},
		{"exactly10!", 10, "exactly10!"},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{45000000000, "41.9 GB"},
	}

	for _, tt := range tests {
		result := formatSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, result, tt.expected)
		}
	}
}

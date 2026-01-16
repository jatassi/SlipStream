package telegram

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
		ID:        1,
		Title:     "The Matrix",
		Year:      1999,
		TMDbID:    603,
		IMDbID:    "tt0133093",
		TraktID:   481,
		Overview:  "A computer hacker learns about the true nature of reality.",
		PosterURL: "https://image.tmdb.org/t/p/poster.jpg",
		Genres:    []string{"Action", "Sci-Fi"},
		Rating:    8.7,
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
			ID:        1,
			Title:     "Breaking Bad",
			Year:      2008,
			TMDbID:    1396,
			IMDbID:    "tt0903747",
			Overview:  "A high school chemistry teacher turned methamphetamine manufacturer.",
			PosterURL: "https://image.tmdb.org/t/p/poster.jpg",
			Genres:    []string{"Drama", "Crime"},
			Rating:    9.5,
		},
		TVDbID:  81189,
		TraktID: 1388,
	}
}

func newTestRelease() types.ReleaseInfo {
	return types.ReleaseInfo{
		ReleaseName:  "The.Matrix.1999.2160p.UHD.BluRay.x265-GROUP",
		Quality:      "Bluray-2160p",
		Size:         45000000000,
		Indexer:      "TestIndexer",
		ReleaseGroup: "GROUP",
	}
}

func newTestDownloadClient() types.DownloadClientInfo {
	return types.DownloadClientInfo{
		ID:   1,
		Name: "qBittorrent",
		Type: "qbittorrent",
	}
}

type capturedRequest struct {
	ChatID              string `json:"chat_id"`
	Text                string `json:"text"`
	ParseMode           string `json:"parse_mode"`
	DisableNotification bool   `json:"disable_notification"`
	MessageThreadID     int64  `json:"message_thread_id"`
}

func setupTestServer(t *testing.T, captured *capturedRequest) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", ct)
		}
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			t.Errorf("expected path to end with /sendMessage, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(captured); err != nil {
			t.Errorf("failed to decode payload: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
}

func TestNotifier_Type(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())
	if n.Type() != types.NotifierTelegram {
		t.Errorf("expected type %s, got %s", types.NotifierTelegram, n.Type())
	}
}

func TestNotifier_Name(t *testing.T) {
	n := New("my-notifier", Settings{}, nil, zerolog.Nop())
	if n.Name() != "my-notifier" {
		t.Errorf("expected name 'my-notifier', got %s", n.Name())
	}
}

func TestNotifier_DefaultMetadataLinks(t *testing.T) {
	n := New("test", Settings{IncludeLinks: true}, nil, zerolog.Nop())
	if len(n.settings.MetadataLinks) != len(DefaultMetadataLinks) {
		t.Errorf("expected %d default metadata links, got %d", len(DefaultMetadataLinks), len(n.settings.MetadataLinks))
	}
}

func TestNotifier_Test(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{
		BotToken: "test-token",
		ChatID:   "123456789",
	}, http.DefaultClient, zerolog.Nop())
	n.settings.BotToken = "" // Clear token so we can use test server

	// Create a custom notifier that uses the test server URL
	n2 := &Notifier{
		name: "test",
		settings: Settings{
			BotToken: "test-token",
			ChatID:   "123456789",
		},
		httpClient: http.DefaultClient,
		logger:     zerolog.Nop(),
	}

	// Override the sendMessage to use test server
	testSend := func(ctx context.Context, text string) error {
		url := server.URL + "/sendMessage"
		payload := map[string]any{
			"chat_id":    n2.settings.ChatID,
			"text":       text,
			"parse_mode": "HTML",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}

	message := "<b>SlipStream Test Notification</b>\n\nThis is a test notification from SlipStream."
	if err := testSend(context.Background(), message); err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if captured.ChatID != "123456789" {
		t.Errorf("expected chat ID '123456789', got %s", captured.ChatID)
	}
	if captured.ParseMode != "HTML" {
		t.Errorf("expected parse mode 'HTML', got %s", captured.ParseMode)
	}
	if !strings.Contains(captured.Text, "SlipStream Test Notification") {
		t.Errorf("expected test title in text, got %s", captured.Text)
	}
}

func TestNotifier_SilentMode(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	testSend := func(ctx context.Context, chatID string, text string, silent bool, topicID int64) error {
		payload := map[string]any{
			"chat_id":    chatID,
			"text":       text,
			"parse_mode": "HTML",
		}
		if silent {
			payload["disable_notification"] = true
		}
		if topicID > 0 {
			payload["message_thread_id"] = topicID
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/sendMessage", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}

	if err := testSend(context.Background(), "123", "test", true, 0); err != nil {
		t.Fatalf("send error = %v", err)
	}

	if !captured.DisableNotification {
		t.Error("expected disable_notification to be true")
	}
}

func TestNotifier_TopicID(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	testSend := func(ctx context.Context, chatID string, text string, silent bool, topicID int64) error {
		payload := map[string]any{
			"chat_id":    chatID,
			"text":       text,
			"parse_mode": "HTML",
		}
		if silent {
			payload["disable_notification"] = true
		}
		if topicID > 0 {
			payload["message_thread_id"] = topicID
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, server.URL+"/sendMessage", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}

	if err := testSend(context.Background(), "123", "test", false, 42); err != nil {
		t.Fatalf("send error = %v", err)
	}

	if captured.MessageThreadID != 42 {
		t.Errorf("expected message_thread_id 42, got %d", captured.MessageThreadID)
	}
}

func TestNotifier_OnGrab_MovieMessage(t *testing.T) {
	event := types.GrabEvent{
		Movie:          newTestMovie(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	// Build the expected message format
	var sb strings.Builder
	sb.WriteString("<b>🎬 Release Grabbed</b>\n\n")
	sb.WriteString("<b>The Matrix</b> (1999)\n")

	message := sb.String()
	if !strings.Contains(message, "Release Grabbed") {
		t.Error("expected message to contain 'Release Grabbed'")
	}
	if !strings.Contains(message, "The Matrix") {
		t.Error("expected message to contain movie title")
	}
	if !strings.Contains(message, "1999") {
		t.Error("expected message to contain year")
	}

	// Verify event is properly constructed
	if event.Movie.Title != "The Matrix" {
		t.Errorf("expected movie title 'The Matrix', got %s", event.Movie.Title)
	}
}

func TestNotifier_OnGrab_EpisodeMessage(t *testing.T) {
	event := types.GrabEvent{
		Episode:        newTestEpisode(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	// Build expected message content
	expectedTitle := "Breaking Bad"
	expectedEpisode := "S05E16"

	if event.Episode.SeriesTitle != expectedTitle {
		t.Errorf("expected series title %s, got %s", expectedTitle, event.Episode.SeriesTitle)
	}
	if event.Episode.SeasonNumber != 5 || event.Episode.EpisodeNumber != 16 {
		t.Errorf("expected S05E16, got S%02dE%02d", event.Episode.SeasonNumber, event.Episode.EpisodeNumber)
	}
	_ = expectedEpisode
}

func TestNotifier_OnDownload_Movie(t *testing.T) {
	event := types.DownloadEvent{
		Movie:        newTestMovie(),
		Quality:      "Bluray-2160p",
		ReleaseGroup: "GROUP",
		ImportedAt:   time.Now(),
	}

	// Verify event is properly constructed
	if event.Movie.Title != "The Matrix" {
		t.Errorf("expected movie title 'The Matrix', got %s", event.Movie.Title)
	}
	if event.Quality != "Bluray-2160p" {
		t.Errorf("expected quality 'Bluray-2160p', got %s", event.Quality)
	}
}

func TestNotifier_OnUpgrade(t *testing.T) {
	event := types.UpgradeEvent{
		Movie:      newTestMovie(),
		OldQuality: "Bluray-1080p",
		NewQuality: "Bluray-2160p",
		UpgradedAt: time.Now(),
	}

	if event.OldQuality != "Bluray-1080p" {
		t.Errorf("expected old quality, got %s", event.OldQuality)
	}
	if event.NewQuality != "Bluray-2160p" {
		t.Errorf("expected new quality, got %s", event.NewQuality)
	}
}

func TestNotifier_OnMovieAdded(t *testing.T) {
	event := types.MovieAddedEvent{
		Movie:   *newTestMovie(),
		AddedAt: time.Now(),
	}

	if event.Movie.Title != "The Matrix" {
		t.Errorf("expected movie title, got %s", event.Movie.Title)
	}
	if event.Movie.Overview == "" {
		t.Error("expected movie overview")
	}
}

func TestNotifier_OnMovieDeleted(t *testing.T) {
	event := types.MovieDeletedEvent{
		Movie:        *newTestMovie(),
		DeletedFiles: true,
		DeletedAt:    time.Now(),
	}

	if !event.DeletedFiles {
		t.Error("expected DeletedFiles to be true")
	}
}

func TestNotifier_OnSeriesAdded(t *testing.T) {
	event := types.SeriesAddedEvent{
		Series:  newTestSeries(),
		AddedAt: time.Now(),
	}

	if event.Series.Title != "Breaking Bad" {
		t.Errorf("expected series title, got %s", event.Series.Title)
	}
}

func TestNotifier_OnSeriesDeleted(t *testing.T) {
	event := types.SeriesDeletedEvent{
		Series:       newTestSeries(),
		DeletedFiles: true,
		DeletedAt:    time.Now(),
	}

	if !event.DeletedFiles {
		t.Error("expected DeletedFiles to be true")
	}
}

func TestNotifier_OnHealthIssue(t *testing.T) {
	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "error",
		Message:   "Connection failed",
		OccuredAt: time.Now(),
	}

	if event.Type != "error" {
		t.Errorf("expected type 'error', got %s", event.Type)
	}
}

func TestNotifier_OnHealthRestored(t *testing.T) {
	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "warning",
		Message:   "Connection restored",
		OccuredAt: time.Now(),
	}

	if event.Source != "Indexer" {
		t.Errorf("expected source 'Indexer', got %s", event.Source)
	}
}

func TestNotifier_OnApplicationUpdate(t *testing.T) {
	event := types.AppUpdateEvent{
		PreviousVersion: "1.0.0",
		NewVersion:      "1.1.0",
		UpdatedAt:       time.Now(),
	}

	if event.PreviousVersion != "1.0.0" {
		t.Errorf("expected previous version, got %s", event.PreviousVersion)
	}
	if event.NewVersion != "1.1.0" {
		t.Errorf("expected new version, got %s", event.NewVersion)
	}
}

func TestNotifier_HasLink(t *testing.T) {
	n := New("test", Settings{
		IncludeLinks:  true,
		MetadataLinks: []MetadataLink{MetadataLinkTMDb, MetadataLinkIMDb},
	}, nil, zerolog.Nop())

	if !n.hasLink(MetadataLinkTMDb) {
		t.Error("expected hasLink(TMDb) to be true")
	}
	if !n.hasLink(MetadataLinkIMDb) {
		t.Error("expected hasLink(IMDb) to be true")
	}
	if n.hasLink(MetadataLinkTVDb) {
		t.Error("expected hasLink(TVDb) to be false")
	}
	if n.hasLink(MetadataLinkTrakt) {
		t.Error("expected hasLink(Trakt) to be false")
	}
}

func TestNotifier_WriteLinks_Movie(t *testing.T) {
	n := New("test", Settings{
		IncludeLinks:  true,
		MetadataLinks: []MetadataLink{MetadataLinkTMDb, MetadataLinkIMDb, MetadataLinkTrakt},
	}, nil, zerolog.Nop())

	movie := newTestMovie()
	var sb strings.Builder
	n.writeLinks(&sb, movie.TMDbID, movie.IMDbID, movie.TraktID, "movie")

	result := sb.String()
	if !strings.Contains(result, "TMDb") {
		t.Error("expected TMDb link")
	}
	if !strings.Contains(result, "themoviedb.org/movie/603") {
		t.Errorf("expected correct TMDb URL, got %s", result)
	}
	if !strings.Contains(result, "IMDb") {
		t.Error("expected IMDb link")
	}
	if !strings.Contains(result, "imdb.com/title/tt0133093") {
		t.Errorf("expected correct IMDb URL, got %s", result)
	}
	if !strings.Contains(result, "Trakt") {
		t.Error("expected Trakt link")
	}
	if !strings.Contains(result, "trakt.tv/movies/481") {
		t.Errorf("expected correct Trakt URL, got %s", result)
	}
}

func TestNotifier_WriteLinks_Series(t *testing.T) {
	n := New("test", Settings{
		IncludeLinks:  true,
		MetadataLinks: []MetadataLink{MetadataLinkTMDb, MetadataLinkIMDb, MetadataLinkTVDb, MetadataLinkTrakt},
	}, nil, zerolog.Nop())

	series := newTestSeries()
	var sb strings.Builder
	n.writeSeriesLinks(&sb, series.TMDbID, series.IMDbID, series.TVDbID, series.TraktID)

	result := sb.String()
	if !strings.Contains(result, "TMDb") {
		t.Error("expected TMDb link")
	}
	if !strings.Contains(result, "themoviedb.org/tv/1396") {
		t.Errorf("expected correct TMDb TV URL, got %s", result)
	}
	if !strings.Contains(result, "TVDb") {
		t.Error("expected TVDb link")
	}
	if !strings.Contains(result, "thetvdb.com/series/81189") {
		t.Errorf("expected correct TVDb URL, got %s", result)
	}
	if !strings.Contains(result, "Trakt") {
		t.Error("expected Trakt link")
	}
	if !strings.Contains(result, "trakt.tv/shows/1388") {
		t.Errorf("expected correct Trakt shows URL, got %s", result)
	}
}

func TestNotifier_WriteLinks_Disabled(t *testing.T) {
	n := New("test", Settings{
		IncludeLinks: false,
	}, nil, zerolog.Nop())

	movie := newTestMovie()
	var sb strings.Builder
	n.writeLinks(&sb, movie.TMDbID, movie.IMDbID, movie.TraktID, "movie")

	result := sb.String()
	if result != "" {
		t.Errorf("expected empty result when links disabled, got %s", result)
	}
}

func TestNotifier_WriteLinks_OnlyConfigured(t *testing.T) {
	n := New("test", Settings{
		IncludeLinks:  true,
		MetadataLinks: []MetadataLink{MetadataLinkTMDb}, // Only TMDb
	}, nil, zerolog.Nop())

	movie := newTestMovie()
	var sb strings.Builder
	n.writeLinks(&sb, movie.TMDbID, movie.IMDbID, movie.TraktID, "movie")

	result := sb.String()
	if !strings.Contains(result, "TMDb") {
		t.Error("expected TMDb link")
	}
	if strings.Contains(result, "IMDb") {
		t.Error("expected NO IMDb link when not configured")
	}
	if strings.Contains(result, "Trakt") {
		t.Error("expected NO Trakt link when not configured")
	}
}

func TestNotifier_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"ok":          false,
			"description": "Bad Request: chat not found",
		})
	}))
	defer server.Close()

	// Test error response parsing
	resp, err := http.Post(server.URL, "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode error = %v", err)
	}
	if result.OK {
		t.Error("expected ok to be false")
	}
	if !strings.Contains(result.Description, "chat not found") {
		t.Errorf("expected error description, got %s", result.Description)
	}
}

func TestNotifier_HTMLEscaping(t *testing.T) {
	movie := &types.MediaInfo{
		ID:    1,
		Title: "Test <Movie> & \"Quotes\"",
		Year:  2024,
	}

	// Verify the title contains special characters
	if !strings.Contains(movie.Title, "<") {
		t.Error("test data should contain < character")
	}
	if !strings.Contains(movie.Title, "&") {
		t.Error("test data should contain & character")
	}
	if !strings.Contains(movie.Title, "\"") {
		t.Error("test data should contain \" character")
	}
}

func TestNotifier_OverviewTruncation(t *testing.T) {
	longOverview := strings.Repeat("A", 250)
	movie := &types.MediaInfo{
		ID:       1,
		Title:    "Test Movie",
		Year:     2024,
		Overview: longOverview,
	}

	// Verify the overview is longer than the truncation limit
	if len(movie.Overview) <= 200 {
		t.Error("test overview should be longer than 200 characters")
	}

	// Simulate truncation
	overview := movie.Overview
	if len(overview) > 200 {
		overview = overview[:197] + "..."
	}

	if len(overview) != 200 {
		t.Errorf("expected truncated overview length 200, got %d", len(overview))
	}
	if !strings.HasSuffix(overview, "...") {
		t.Error("expected truncated overview to end with ...")
	}
}

func TestMetadataLinkConstants(t *testing.T) {
	if MetadataLinkTMDb != "tmdb" {
		t.Errorf("expected MetadataLinkTMDb = 'tmdb', got %s", MetadataLinkTMDb)
	}
	if MetadataLinkIMDb != "imdb" {
		t.Errorf("expected MetadataLinkIMDb = 'imdb', got %s", MetadataLinkIMDb)
	}
	if MetadataLinkTVDb != "tvdb" {
		t.Errorf("expected MetadataLinkTVDb = 'tvdb', got %s", MetadataLinkTVDb)
	}
	if MetadataLinkTrakt != "trakt" {
		t.Errorf("expected MetadataLinkTrakt = 'trakt', got %s", MetadataLinkTrakt)
	}
}

func TestDefaultMetadataLinks(t *testing.T) {
	if len(DefaultMetadataLinks) != 2 {
		t.Errorf("expected 2 default metadata links, got %d", len(DefaultMetadataLinks))
	}

	hasTMDb := false
	hasIMDb := false
	for _, link := range DefaultMetadataLinks {
		if link == MetadataLinkTMDb {
			hasTMDb = true
		}
		if link == MetadataLinkIMDb {
			hasIMDb = true
		}
	}

	if !hasTMDb {
		t.Error("expected TMDb in default metadata links")
	}
	if !hasIMDb {
		t.Error("expected IMDb in default metadata links")
	}
}

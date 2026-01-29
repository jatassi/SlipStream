package slack

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
			Overview:  "A high school chemistry teacher turned methamphetamine manufacturer.",
			PosterURL: "https://image.tmdb.org/t/p/poster.jpg",
			Genres:    []string{"Drama", "Crime"},
			Rating:    9.5,
		},
		TVDbID: 81189,
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
	Payload Payload
	Headers http.Header
}

func setupTestServer(t *testing.T, captured *capturedRequest) *httptest.Server {
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
		w.WriteHeader(http.StatusOK)
	}))
}

func TestNotifier_Type(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())
	if n.Type() != types.NotifierSlack {
		t.Errorf("expected type %s, got %s", types.NotifierSlack, n.Type())
	}
}

func TestNotifier_Name(t *testing.T) {
	n := New("my-notifier", Settings{}, nil, zerolog.Nop())
	if n.Name() != "my-notifier" {
		t.Errorf("expected name 'my-notifier', got %s", n.Name())
	}
}

func TestNotifier_Test(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{
		WebhookURL: server.URL,
		Username:   "TestBot",
		IconEmoji:  ":robot:",
		Channel:    "#notifications",
	}, http.DefaultClient, zerolog.Nop())

	if err := n.Test(context.Background()); err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if captured.Payload.Username != "TestBot" {
		t.Errorf("expected username 'TestBot', got %s", captured.Payload.Username)
	}
	if captured.Payload.IconEmoji != ":robot:" {
		t.Errorf("expected icon emoji, got %s", captured.Payload.IconEmoji)
	}
	if captured.Payload.Channel != "#notifications" {
		t.Errorf("expected channel, got %s", captured.Payload.Channel)
	}
	if len(captured.Payload.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(captured.Payload.Attachments))
	}
	if captured.Payload.Attachments[0].Title != "SlipStream Test Notification" {
		t.Errorf("expected test title, got %s", captured.Payload.Attachments[0].Title)
	}
	if captured.Payload.Attachments[0].Color != ColorGood {
		t.Errorf("expected color %s, got %s", ColorGood, captured.Payload.Attachments[0].Color)
	}
}

func TestNotifier_IconURL(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{
		WebhookURL: server.URL,
		IconURL:    "https://example.com/icon.png",
		IconEmoji:  ":robot:", // Should be ignored when IconURL is set
	}, http.DefaultClient, zerolog.Nop())

	// Use OnGrab instead of Test() because OnGrab uses buildPayload which handles IconURL properly
	event := types.GrabEvent{
		Movie:          newTestMovie(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	if err := n.OnGrab(context.Background(), event); err != nil {
		t.Fatalf("OnGrab() error = %v", err)
	}

	if captured.Payload.IconURL != "https://example.com/icon.png" {
		t.Errorf("expected icon URL, got %s", captured.Payload.IconURL)
	}
	if captured.Payload.IconEmoji != "" {
		t.Errorf("expected no icon emoji when IconURL is set, got %s", captured.Payload.IconEmoji)
	}
}

func TestNotifier_DefaultUsername(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	if err := n.Test(context.Background()); err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if captured.Payload.Username != "SlipStream" {
		t.Errorf("expected default username 'SlipStream', got %s", captured.Payload.Username)
	}
}

func TestNotifier_OnGrab_Movie(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.GrabEvent{
		Movie:          newTestMovie(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	if err := n.OnGrab(context.Background(), event); err != nil {
		t.Fatalf("OnGrab() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if !strings.Contains(attachment.Title, "Movie Grabbed") {
		t.Errorf("expected title to contain 'Movie Grabbed', got %s", attachment.Title)
	}
	if !strings.Contains(attachment.Title, "The Matrix") {
		t.Errorf("expected title to contain movie name, got %s", attachment.Title)
	}
	if !strings.Contains(attachment.Title, "1999") {
		t.Errorf("expected title to contain year, got %s", attachment.Title)
	}
	if !strings.Contains(attachment.Text, event.Release.ReleaseName) {
		t.Errorf("expected text to contain release name")
	}
	if attachment.Color != "#7289DA" {
		t.Errorf("expected Discord blurple color, got %s", attachment.Color)
	}

	fieldMap := make(map[string]string)
	for _, f := range attachment.Fields {
		fieldMap[f.Title] = f.Value
	}

	if fieldMap["Quality"] != "Bluray-2160p" {
		t.Errorf("expected quality field, got %s", fieldMap["Quality"])
	}
	if fieldMap["Indexer"] != "TestIndexer" {
		t.Errorf("expected indexer field, got %s", fieldMap["Indexer"])
	}
	if fieldMap["Client"] != "qBittorrent" {
		t.Errorf("expected client field, got %s", fieldMap["Client"])
	}
	if fieldMap["Group"] != "GROUP" {
		t.Errorf("expected group field, got %s", fieldMap["Group"])
	}
}

func TestNotifier_OnGrab_Episode(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.GrabEvent{
		Episode:        newTestEpisode(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	if err := n.OnGrab(context.Background(), event); err != nil {
		t.Fatalf("OnGrab() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if !strings.Contains(attachment.Title, "Episode Grabbed") {
		t.Errorf("expected title to contain 'Episode Grabbed', got %s", attachment.Title)
	}
	if !strings.Contains(attachment.Title, "Breaking Bad") {
		t.Errorf("expected title to contain series name, got %s", attachment.Title)
	}
	if !strings.Contains(attachment.Title, "S05E16") {
		t.Errorf("expected title to contain season/episode, got %s", attachment.Title)
	}
}

func TestNotifier_OnImport_Movie(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.ImportEvent{
		Movie:        newTestMovie(),
		Quality:      "Bluray-2160p",
		ReleaseGroup: "GROUP",
		ImportedAt:   time.Now(),
	}

	if err := n.OnImport(context.Background(), event); err != nil {
		t.Fatalf("OnImport() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if !strings.Contains(attachment.Title, "Movie Downloaded") {
		t.Errorf("expected title to contain 'Movie Downloaded', got %s", attachment.Title)
	}
	if attachment.Color != ColorGood {
		t.Errorf("expected color %s, got %s", ColorGood, attachment.Color)
	}

	fieldMap := make(map[string]string)
	for _, f := range attachment.Fields {
		fieldMap[f.Title] = f.Value
	}

	if fieldMap["Quality"] != "Bluray-2160p" {
		t.Errorf("expected quality field, got %s", fieldMap["Quality"])
	}
	if fieldMap["Group"] != "GROUP" {
		t.Errorf("expected group field, got %s", fieldMap["Group"])
	}
}

func TestNotifier_OnImport_Episode(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.ImportEvent{
		Episode:    newTestEpisode(),
		Quality:    "HDTV-1080p",
		ImportedAt: time.Now(),
	}

	if err := n.OnImport(context.Background(), event); err != nil {
		t.Fatalf("OnImport() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if !strings.Contains(attachment.Title, "Episode Downloaded") {
		t.Errorf("expected title to contain 'Episode Downloaded', got %s", attachment.Title)
	}
	if !strings.Contains(attachment.Title, "S05E16") {
		t.Errorf("expected title to contain season/episode, got %s", attachment.Title)
	}
}

func TestNotifier_OnUpgrade_Movie(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.UpgradeEvent{
		Movie:      newTestMovie(),
		OldQuality: "Bluray-1080p",
		NewQuality: "Bluray-2160p",
		UpgradedAt: time.Now(),
	}

	if err := n.OnUpgrade(context.Background(), event); err != nil {
		t.Fatalf("OnUpgrade() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if !strings.Contains(attachment.Title, "Movie Upgraded") {
		t.Errorf("expected title to contain 'Movie Upgraded', got %s", attachment.Title)
	}
	if attachment.Color != ColorGood {
		t.Errorf("expected color %s, got %s", ColorGood, attachment.Color)
	}

	fieldMap := make(map[string]string)
	for _, f := range attachment.Fields {
		fieldMap[f.Title] = f.Value
	}

	if fieldMap["Old Quality"] != "Bluray-1080p" {
		t.Errorf("expected old quality field, got %s", fieldMap["Old Quality"])
	}
	if fieldMap["New Quality"] != "Bluray-2160p" {
		t.Errorf("expected new quality field, got %s", fieldMap["New Quality"])
	}
}

func TestNotifier_OnUpgrade_Episode(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.UpgradeEvent{
		Episode:    newTestEpisode(),
		OldQuality: "HDTV-720p",
		NewQuality: "Bluray-1080p",
		UpgradedAt: time.Now(),
	}

	if err := n.OnUpgrade(context.Background(), event); err != nil {
		t.Fatalf("OnUpgrade() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if !strings.Contains(attachment.Title, "Episode Upgraded") {
		t.Errorf("expected title to contain 'Episode Upgraded', got %s", attachment.Title)
	}
}

func TestNotifier_OnMovieAdded(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.MovieAddedEvent{
		Movie:   *newTestMovie(),
		AddedAt: time.Now(),
	}

	if err := n.OnMovieAdded(context.Background(), event); err != nil {
		t.Fatalf("OnMovieAdded() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if !strings.Contains(attachment.Title, "Movie Added") {
		t.Errorf("expected title to contain 'Movie Added', got %s", attachment.Title)
	}
	if !strings.Contains(attachment.Title, "The Matrix") {
		t.Errorf("expected title to contain movie name, got %s", attachment.Title)
	}
	if attachment.Color != ColorGood {
		t.Errorf("expected color %s, got %s", ColorGood, attachment.Color)
	}
	if !strings.Contains(attachment.Text, "computer hacker") {
		t.Errorf("expected overview in text, got %s", attachment.Text)
	}
	if attachment.ThumbURL != event.Movie.PosterURL {
		t.Errorf("expected thumb URL, got %s", attachment.ThumbURL)
	}
}

func TestNotifier_OnMovieDeleted(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.MovieDeletedEvent{
		Movie:        *newTestMovie(),
		DeletedFiles: true,
		DeletedAt:    time.Now(),
	}

	if err := n.OnMovieDeleted(context.Background(), event); err != nil {
		t.Fatalf("OnMovieDeleted() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if !strings.Contains(attachment.Title, "Movie Deleted") {
		t.Errorf("expected title to contain 'Movie Deleted', got %s", attachment.Title)
	}
	if attachment.Color != ColorDanger {
		t.Errorf("expected color %s, got %s", ColorDanger, attachment.Color)
	}
	if !strings.Contains(attachment.Text, "files deleted") {
		t.Errorf("expected text to mention files deleted, got %s", attachment.Text)
	}
}

func TestNotifier_OnMovieDeleted_NoFiles(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.MovieDeletedEvent{
		Movie:        *newTestMovie(),
		DeletedFiles: false,
		DeletedAt:    time.Now(),
	}

	if err := n.OnMovieDeleted(context.Background(), event); err != nil {
		t.Fatalf("OnMovieDeleted() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if strings.Contains(attachment.Text, "files deleted") {
		t.Errorf("expected text NOT to mention files deleted, got %s", attachment.Text)
	}
}

func TestNotifier_OnSeriesAdded(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.SeriesAddedEvent{
		Series:  newTestSeries(),
		AddedAt: time.Now(),
	}

	if err := n.OnSeriesAdded(context.Background(), event); err != nil {
		t.Fatalf("OnSeriesAdded() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if !strings.Contains(attachment.Title, "Series Added") {
		t.Errorf("expected title to contain 'Series Added', got %s", attachment.Title)
	}
	if !strings.Contains(attachment.Title, "Breaking Bad") {
		t.Errorf("expected title to contain series name, got %s", attachment.Title)
	}
	if attachment.Color != ColorGood {
		t.Errorf("expected color %s, got %s", ColorGood, attachment.Color)
	}
}

func TestNotifier_OnSeriesDeleted(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.SeriesDeletedEvent{
		Series:       newTestSeries(),
		DeletedFiles: true,
		DeletedAt:    time.Now(),
	}

	if err := n.OnSeriesDeleted(context.Background(), event); err != nil {
		t.Fatalf("OnSeriesDeleted() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if !strings.Contains(attachment.Title, "Series Deleted") {
		t.Errorf("expected title to contain 'Series Deleted', got %s", attachment.Title)
	}
	if attachment.Color != ColorDanger {
		t.Errorf("expected color %s, got %s", ColorDanger, attachment.Color)
	}
	if !strings.Contains(attachment.Text, "files deleted") {
		t.Errorf("expected text to mention files deleted, got %s", attachment.Text)
	}
}

func TestNotifier_OnHealthIssue_Warning(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "warning",
		Message:   "Indexer is unreachable",
		OccuredAt: time.Now(),
	}

	if err := n.OnHealthIssue(context.Background(), event); err != nil {
		t.Fatalf("OnHealthIssue() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if attachment.Title != "Health Issue" {
		t.Errorf("expected title 'Health Issue', got %s", attachment.Title)
	}
	if attachment.Color != ColorWarning {
		t.Errorf("expected color %s for warning, got %s", ColorWarning, attachment.Color)
	}
	if attachment.Text != event.Message {
		t.Errorf("expected message in text, got %s", attachment.Text)
	}

	fieldMap := make(map[string]string)
	for _, f := range attachment.Fields {
		fieldMap[f.Title] = f.Value
	}

	if fieldMap["Source"] != "Indexer" {
		t.Errorf("expected source field, got %s", fieldMap["Source"])
	}
	if fieldMap["Type"] != "warning" {
		t.Errorf("expected type field, got %s", fieldMap["Type"])
	}
}

func TestNotifier_OnHealthIssue_Error(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.HealthEvent{
		Source:    "Database",
		Type:      "error",
		Message:   "Database connection failed",
		OccuredAt: time.Now(),
	}

	if err := n.OnHealthIssue(context.Background(), event); err != nil {
		t.Fatalf("OnHealthIssue() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if attachment.Color != ColorDanger {
		t.Errorf("expected color %s for error, got %s", ColorDanger, attachment.Color)
	}
}

func TestNotifier_OnHealthRestored(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "warning",
		Message:   "Indexer is now reachable",
		OccuredAt: time.Now(),
	}

	if err := n.OnHealthRestored(context.Background(), event); err != nil {
		t.Fatalf("OnHealthRestored() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if attachment.Title != "Health Issue Resolved" {
		t.Errorf("expected title 'Health Issue Resolved', got %s", attachment.Title)
	}
	if attachment.Color != ColorGood {
		t.Errorf("expected color %s, got %s", ColorGood, attachment.Color)
	}
}

func TestNotifier_OnApplicationUpdate(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.AppUpdateEvent{
		PreviousVersion: "1.0.0",
		NewVersion:      "1.1.0",
		UpdatedAt:       time.Now(),
	}

	if err := n.OnApplicationUpdate(context.Background(), event); err != nil {
		t.Fatalf("OnApplicationUpdate() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if attachment.Title != "Application Updated" {
		t.Errorf("expected title 'Application Updated', got %s", attachment.Title)
	}
	if attachment.Color != ColorGood {
		t.Errorf("expected color %s, got %s", ColorGood, attachment.Color)
	}

	fieldMap := make(map[string]string)
	for _, f := range attachment.Fields {
		fieldMap[f.Title] = f.Value
	}

	if fieldMap["Previous Version"] != "1.0.0" {
		t.Errorf("expected previous version, got %s", fieldMap["Previous Version"])
	}
	if fieldMap["New Version"] != "1.1.0" {
		t.Errorf("expected new version, got %s", fieldMap["New Version"])
	}
}

func TestNotifier_Fallback(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	event := types.GrabEvent{
		Movie:          newTestMovie(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	if err := n.OnGrab(context.Background(), event); err != nil {
		t.Fatalf("OnGrab() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if attachment.Fallback == "" {
		t.Error("expected fallback text to be set")
	}
	if !strings.Contains(attachment.Fallback, "Movie Grabbed") {
		t.Errorf("expected fallback to contain title, got %s", attachment.Fallback)
	}
}

func TestNotifier_Footer(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	// Use OnGrab instead of Test() because OnGrab uses buildPayload which sets FooterIcon
	event := types.GrabEvent{
		Movie:          newTestMovie(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	if err := n.OnGrab(context.Background(), event); err != nil {
		t.Fatalf("OnGrab() error = %v", err)
	}

	attachment := captured.Payload.Attachments[0]
	if attachment.Footer != "SlipStream" {
		t.Errorf("expected footer 'SlipStream', got %s", attachment.Footer)
	}
	if attachment.FooterIcon == "" {
		t.Error("expected footer icon to be set")
	}
}

func TestNotifier_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

	err := n.Test(context.Background())
	if err == nil {
		t.Error("expected error for HTTP 400 response")
	}
	if !strings.Contains(err.Error(), "400") {
		t.Errorf("expected error to contain status code, got %v", err)
	}
}

func TestNotifier_Timestamp(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	n := New("test", Settings{WebhookURL: server.URL}, http.DefaultClient, zerolog.Nop())

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

	attachment := captured.Payload.Attachments[0]
	if attachment.Ts != eventTime.Unix() {
		t.Errorf("expected timestamp %d, got %d", eventTime.Unix(), attachment.Ts)
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

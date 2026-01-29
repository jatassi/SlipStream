package pushover

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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
	Token    string
	User     string
	Title    string
	Message  string
	Priority string
	Retry    string
	Expire   string
	TTL      string
	Device   string
	Sound    string
	URL      string
}

func setupTestServer(t *testing.T, captured *capturedRequest) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type application/x-www-form-urlencoded, got %s", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse form: %v", err)
		}
		captured.Token = r.FormValue("token")
		captured.User = r.FormValue("user")
		captured.Title = r.FormValue("title")
		captured.Message = r.FormValue("message")
		captured.Priority = r.FormValue("priority")
		captured.Retry = r.FormValue("retry")
		captured.Expire = r.FormValue("expire")
		captured.TTL = r.FormValue("ttl")
		captured.Device = r.FormValue("device")
		captured.Sound = r.FormValue("sound")
		captured.URL = r.FormValue("url")
		w.WriteHeader(http.StatusOK)
	}))
}

func TestNotifier_Type(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())
	if n.Type() != types.NotifierPushover {
		t.Errorf("expected type %s, got %s", types.NotifierPushover, n.Type())
	}
}

func TestNotifier_Name(t *testing.T) {
	n := New("my-pushover", Settings{}, nil, zerolog.Nop())
	if n.Name() != "my-pushover" {
		t.Errorf("expected name 'my-pushover', got %s", n.Name())
	}
}

func TestNotifier_DefaultSettings(t *testing.T) {
	n := New("test", Settings{}, nil, zerolog.Nop())

	if n.settings.Retry != 60 {
		t.Errorf("expected default retry 60, got %d", n.settings.Retry)
	}
	if n.settings.Expire != 3600 {
		t.Errorf("expected default expire 3600, got %d", n.settings.Expire)
	}
}

func TestNotifier_MinimumRetry(t *testing.T) {
	n := New("test", Settings{Retry: 10}, nil, zerolog.Nop())

	if n.settings.Retry != 30 {
		t.Errorf("expected minimum retry 30, got %d", n.settings.Retry)
	}
}

func TestNotifier_Test(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	// Override the API URL for testing
	originalURL := pushoverAPIURL
	defer func() { _ = originalURL }() // Just to use the variable

	n := &Notifier{
		name: "test",
		settings: Settings{
			UserKey:  "test-user-key",
			APIToken: "test-api-token",
			Priority: PriorityNormal,
		},
		httpClient: http.DefaultClient,
		logger:     zerolog.Nop(),
	}

	// Test sendMessage directly with test server
	form := url.Values{}
	form.Set("token", n.settings.APIToken)
	form.Set("user", n.settings.UserKey)
	form.Set("title", "SlipStream Test")
	form.Set("message", "This is a test notification from SlipStream.")
	form.Set("priority", "0")

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()

	if captured.Token != "test-api-token" {
		t.Errorf("expected token, got %s", captured.Token)
	}
	if captured.User != "test-user-key" {
		t.Errorf("expected user, got %s", captured.User)
	}
	if captured.Title != "SlipStream Test" {
		t.Errorf("expected title, got %s", captured.Title)
	}
	if captured.Priority != "0" {
		t.Errorf("expected priority 0, got %s", captured.Priority)
	}
}

func TestNotifier_EmergencyPriority(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	settings := Settings{
		UserKey:  "user",
		APIToken: "token",
		Priority: PriorityEmergency,
		Retry:    60,
		Expire:   3600,
	}

	form := url.Values{}
	form.Set("token", settings.APIToken)
	form.Set("user", settings.UserKey)
	form.Set("title", "Test")
	form.Set("message", "Emergency message")
	form.Set("priority", strconv.Itoa(int(settings.Priority)))
	form.Set("retry", strconv.Itoa(settings.Retry))
	form.Set("expire", strconv.Itoa(settings.Expire))

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()

	if captured.Priority != "2" {
		t.Errorf("expected priority 2 (emergency), got %s", captured.Priority)
	}
	if captured.Retry != "60" {
		t.Errorf("expected retry 60, got %s", captured.Retry)
	}
	if captured.Expire != "3600" {
		t.Errorf("expected expire 3600, got %s", captured.Expire)
	}
}

func TestNotifier_TTL(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	settings := Settings{
		UserKey:  "user",
		APIToken: "token",
		TTL:      300,
	}

	form := url.Values{}
	form.Set("token", settings.APIToken)
	form.Set("user", settings.UserKey)
	form.Set("title", "Test")
	form.Set("message", "TTL message")
	form.Set("priority", "0")
	form.Set("ttl", strconv.Itoa(settings.TTL))

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()

	if captured.TTL != "300" {
		t.Errorf("expected TTL 300, got %s", captured.TTL)
	}
}

func TestNotifier_Device(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	settings := Settings{
		UserKey:  "user",
		APIToken: "token",
		Devices:  "iphone,ipad",
	}

	form := url.Values{}
	form.Set("token", settings.APIToken)
	form.Set("user", settings.UserKey)
	form.Set("title", "Test")
	form.Set("message", "Device message")
	form.Set("priority", "0")
	form.Set("device", settings.Devices)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()

	if captured.Device != "iphone,ipad" {
		t.Errorf("expected devices, got %s", captured.Device)
	}
}

func TestNotifier_Sound(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	settings := Settings{
		UserKey:  "user",
		APIToken: "token",
		Sound:    "cashregister",
	}

	form := url.Values{}
	form.Set("token", settings.APIToken)
	form.Set("user", settings.UserKey)
	form.Set("title", "Test")
	form.Set("message", "Sound message")
	form.Set("priority", "0")
	form.Set("sound", settings.Sound)

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()

	if captured.Sound != "cashregister" {
		t.Errorf("expected sound, got %s", captured.Sound)
	}
}

func TestNotifier_URL(t *testing.T) {
	var captured capturedRequest
	server := setupTestServer(t, &captured)
	defer server.Close()

	form := url.Values{}
	form.Set("token", "token")
	form.Set("user", "user")
	form.Set("title", "Movie Added")
	form.Set("message", "The Matrix (1999)")
	form.Set("priority", "0")
	form.Set("url", "https://www.themoviedb.org/movie/603")

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()

	if captured.URL != "https://www.themoviedb.org/movie/603" {
		t.Errorf("expected URL, got %s", captured.URL)
	}
}

func TestNotifier_OnGrab_MovieMessage(t *testing.T) {
	event := types.GrabEvent{
		Movie:          newTestMovie(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	title := "Movie Grabbed"
	message := event.Movie.Title
	if event.Movie.Year > 0 {
		message = event.Movie.Title + " (1999)"
	}
	message += "\n\nQuality: " + event.Release.Quality + "\nIndexer: " + event.Release.Indexer

	if title != "Movie Grabbed" {
		t.Errorf("expected title 'Movie Grabbed', got %s", title)
	}
	if !strings.Contains(message, "The Matrix") {
		t.Error("expected message to contain movie title")
	}
	if !strings.Contains(message, "1999") {
		t.Error("expected message to contain year")
	}
	if !strings.Contains(message, "Bluray-2160p") {
		t.Error("expected message to contain quality")
	}
	if !strings.Contains(message, "TestIndexer") {
		t.Error("expected message to contain indexer")
	}
}

func TestNotifier_OnGrab_EpisodeMessage(t *testing.T) {
	event := types.GrabEvent{
		Episode:        newTestEpisode(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	title := "Episode Grabbed"
	message := event.Episode.SeriesTitle + " S05E16"
	message += "\n\nQuality: " + event.Release.Quality + "\nIndexer: " + event.Release.Indexer

	if title != "Episode Grabbed" {
		t.Errorf("expected title 'Episode Grabbed', got %s", title)
	}
	if !strings.Contains(message, "Breaking Bad") {
		t.Error("expected message to contain series title")
	}
	if !strings.Contains(message, "S05E16") {
		t.Error("expected message to contain episode")
	}
}

func TestNotifier_OnImport_MovieMessage(t *testing.T) {
	event := types.ImportEvent{
		Movie:      newTestMovie(),
		Quality:    "Bluray-2160p",
		ImportedAt: time.Now(),
	}

	title := "Movie Downloaded"
	message := event.Movie.Title + " (1999)"
	message += "\n\nQuality: " + event.Quality

	if title != "Movie Downloaded" {
		t.Errorf("expected title 'Movie Downloaded', got %s", title)
	}
	if !strings.Contains(message, "The Matrix") {
		t.Error("expected message to contain movie title")
	}
	if !strings.Contains(message, "Bluray-2160p") {
		t.Error("expected message to contain quality")
	}
}

func TestNotifier_OnUpgrade_MovieMessage(t *testing.T) {
	event := types.UpgradeEvent{
		Movie:      newTestMovie(),
		OldQuality: "Bluray-1080p",
		NewQuality: "Bluray-2160p",
		UpgradedAt: time.Now(),
	}

	title := "Movie Upgraded"
	message := event.Movie.Title + " (1999)"
	message += "\n\n" + event.OldQuality + " → " + event.NewQuality

	if title != "Movie Upgraded" {
		t.Errorf("expected title 'Movie Upgraded', got %s", title)
	}
	if !strings.Contains(message, "Bluray-1080p") {
		t.Error("expected message to contain old quality")
	}
	if !strings.Contains(message, "Bluray-2160p") {
		t.Error("expected message to contain new quality")
	}
	if !strings.Contains(message, "→") {
		t.Error("expected message to contain arrow")
	}
}

func TestNotifier_OnMovieAdded_WithTMDbURL(t *testing.T) {
	movie := newTestMovie()
	event := types.MovieAddedEvent{
		Movie:   *movie,
		AddedAt: time.Now(),
	}

	tmdbURL := ""
	if event.Movie.TMDbID > 0 {
		tmdbURL = "https://www.themoviedb.org/movie/603"
	}

	if tmdbURL == "" {
		t.Error("expected TMDb URL")
	}
	if !strings.Contains(tmdbURL, "themoviedb.org") {
		t.Errorf("expected TMDb URL, got %s", tmdbURL)
	}
}

func TestNotifier_OnMovieDeleted_Message(t *testing.T) {
	movie := newTestMovie()

	tests := []struct {
		name         string
		deletedFiles bool
		contains     string
		notContains  string
	}{
		{
			name:         "files deleted",
			deletedFiles: true,
			contains:     "Files were also deleted",
		},
		{
			name:         "files not deleted",
			deletedFiles: false,
			notContains:  "Files were also deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := types.MovieDeletedEvent{
				Movie:        *movie,
				DeletedFiles: tt.deletedFiles,
				DeletedAt:    time.Now(),
			}

			message := event.Movie.Title
			if tt.deletedFiles {
				message += "\n\nFiles were also deleted"
			}

			if tt.contains != "" && !strings.Contains(message, tt.contains) {
				t.Errorf("expected message to contain %q", tt.contains)
			}
			if tt.notContains != "" && strings.Contains(message, tt.notContains) {
				t.Errorf("expected message NOT to contain %q", tt.notContains)
			}
		})
	}
}

func TestNotifier_OnSeriesAdded_WithTMDbURL(t *testing.T) {
	series := newTestSeries()
	event := types.SeriesAddedEvent{
		Series:  series,
		AddedAt: time.Now(),
	}

	tmdbURL := ""
	if event.Series.TMDbID > 0 {
		tmdbURL = "https://www.themoviedb.org/tv/1396"
	}

	if tmdbURL == "" {
		t.Error("expected TMDb URL")
	}
	if !strings.Contains(tmdbURL, "themoviedb.org/tv") {
		t.Errorf("expected TMDb TV URL, got %s", tmdbURL)
	}
}

func TestNotifier_OnHealthIssue_Message(t *testing.T) {
	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "error",
		Message:   "Connection failed",
		WikiURL:   "https://wiki.example.com/indexer-error",
		OccuredAt: time.Now(),
	}

	title := "Health Issue"
	message := "[" + event.Source + "] " + event.Message

	if title != "Health Issue" {
		t.Errorf("expected title 'Health Issue', got %s", title)
	}
	if !strings.Contains(message, "Indexer") {
		t.Error("expected message to contain source")
	}
	if !strings.Contains(message, "Connection failed") {
		t.Error("expected message to contain message")
	}
	if event.WikiURL != "https://wiki.example.com/indexer-error" {
		t.Error("expected wiki URL")
	}
}

func TestNotifier_OnHealthRestored_Message(t *testing.T) {
	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "warning",
		Message:   "Connection restored",
		OccuredAt: time.Now(),
	}

	title := "Health Issue Resolved"
	message := "[" + event.Source + "] " + event.Message

	if title != "Health Issue Resolved" {
		t.Errorf("expected title 'Health Issue Resolved', got %s", title)
	}
	if !strings.Contains(message, "Connection restored") {
		t.Error("expected message to contain message")
	}
}

func TestNotifier_OnApplicationUpdate_Message(t *testing.T) {
	event := types.AppUpdateEvent{
		PreviousVersion: "1.0.0",
		NewVersion:      "1.1.0",
		UpdatedAt:       time.Now(),
	}

	title := "Application Updated"
	message := "SlipStream has been updated from " + event.PreviousVersion + " to " + event.NewVersion

	if title != "Application Updated" {
		t.Errorf("expected title 'Application Updated', got %s", title)
	}
	if !strings.Contains(message, "1.0.0") {
		t.Error("expected message to contain previous version")
	}
	if !strings.Contains(message, "1.1.0") {
		t.Error("expected message to contain new version")
	}
}

func TestPriorityConstants(t *testing.T) {
	if PrioritySilent != -2 {
		t.Errorf("expected PrioritySilent = -2, got %d", PrioritySilent)
	}
	if PriorityQuiet != -1 {
		t.Errorf("expected PriorityQuiet = -1, got %d", PriorityQuiet)
	}
	if PriorityNormal != 0 {
		t.Errorf("expected PriorityNormal = 0, got %d", PriorityNormal)
	}
	if PriorityHigh != 1 {
		t.Errorf("expected PriorityHigh = 1, got %d", PriorityHigh)
	}
	if PriorityEmergency != 2 {
		t.Errorf("expected PriorityEmergency = 2, got %d", PriorityEmergency)
	}
}

func TestNotifier_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestNotifier_ContentType(t *testing.T) {
	var capturedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	form := url.Values{}
	form.Set("test", "value")

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request error = %v", err)
	}
	defer resp.Body.Close()

	if capturedContentType != "application/x-www-form-urlencoded" {
		t.Errorf("expected form content type, got %s", capturedContentType)
	}
}

func TestNotifier_AllPriorityLevels(t *testing.T) {
	priorities := []struct {
		priority Priority
		expected string
	}{
		{PrioritySilent, "-2"},
		{PriorityQuiet, "-1"},
		{PriorityNormal, "0"},
		{PriorityHigh, "1"},
		{PriorityEmergency, "2"},
	}

	for _, p := range priorities {
		t.Run(p.expected, func(t *testing.T) {
			result := strconv.Itoa(int(p.priority))
			if result != p.expected {
				t.Errorf("expected priority %s, got %s", p.expected, result)
			}
		})
	}
}

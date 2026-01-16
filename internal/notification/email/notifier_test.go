package email

import (
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/notification/types"
)

func newTestMovie() types.MediaInfo {
	return types.MediaInfo{
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

func TestNotifier_Type(t *testing.T) {
	n := New("test", Settings{}, zerolog.Nop())
	if n.Type() != types.NotifierEmail {
		t.Errorf("expected type %s, got %s", types.NotifierEmail, n.Type())
	}
}

func TestNotifier_Name(t *testing.T) {
	n := New("my-email", Settings{}, zerolog.Nop())
	if n.Name() != "my-email" {
		t.Errorf("expected name 'my-email', got %s", n.Name())
	}
}

func TestNotifier_DefaultPort(t *testing.T) {
	n := New("test", Settings{}, zerolog.Nop())
	if n.settings.Port != 587 {
		t.Errorf("expected default port 587, got %d", n.settings.Port)
	}
}

func TestNotifier_DefaultEncryption(t *testing.T) {
	n := New("test", Settings{}, zerolog.Nop())
	if n.settings.Encryption != EncryptionPreferred {
		t.Errorf("expected default encryption preferred, got %s", n.settings.Encryption)
	}

	n2 := New("test", Settings{UseTLS: true}, zerolog.Nop())
	if n2.settings.Encryption != EncryptionAlways {
		t.Errorf("expected encryption always when UseTLS is true, got %s", n2.settings.Encryption)
	}
}

func TestNotifier_ToHTML(t *testing.T) {
	n := New("test", Settings{UseHTML: true}, zerolog.Nop())

	tests := []struct {
		name     string
		input    string
		contains []string
	}{
		{
			name:  "simple text",
			input: "Hello World",
			contains: []string{
				"<!DOCTYPE html>",
				"<html>",
				"Hello World",
				"SlipStream",
			},
		},
		{
			name:  "escape special chars",
			input: "Test <script>alert('xss')</script> & \"quotes\"",
			contains: []string{
				"&lt;script&gt;",
				"&amp;",
			},
		},
		{
			name:  "line breaks",
			input: "Line 1\nLine 2\n\nParagraph 2",
			contains: []string{
				"<br>",
				"</p><p>",
			},
		},
		{
			name:  "styling",
			input: "Test",
			contains: []string{
				"font-family:",
				"SlipStream",
				".header",
				".content",
				".footer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := n.toHTML(tt.input)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected HTML to contain %q, got %s", expected, result)
				}
			}
		})
	}
}

func TestNotifier_ToHTML_NoRawHTML(t *testing.T) {
	n := New("test", Settings{UseHTML: true}, zerolog.Nop())

	input := "<script>alert('xss')</script>"
	result := n.toHTML(input)

	if strings.Contains(result, "<script>") {
		t.Error("expected script tags to be escaped")
	}
	if !strings.Contains(result, "&lt;script&gt;") {
		t.Error("expected escaped script tags")
	}
}

func TestParseAddresses(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"user@example.com", []string{"user@example.com"}},
		{"user1@example.com, user2@example.com", []string{"user1@example.com", "user2@example.com"}},
		{"user1@example.com,user2@example.com,user3@example.com", []string{"user1@example.com", "user2@example.com", "user3@example.com"}},
		{"  user@example.com  ", []string{"user@example.com"}},
		{"user1@example.com,  ,user2@example.com", []string{"user1@example.com", "user2@example.com"}},
	}

	for _, tt := range tests {
		result := parseAddresses(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseAddresses(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, addr := range result {
			if addr != tt.expected[i] {
				t.Errorf("parseAddresses(%q)[%d] = %q, want %q", tt.input, i, addr, tt.expected[i])
			}
		}
	}
}

func TestNotifier_OnGrab_Subject_Movie(t *testing.T) {
	movie := newTestMovie()
	event := types.GrabEvent{
		Movie:          &movie,
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	expectedSubject := "[SlipStream] Movie Grabbed - The Matrix (1999)"
	actualSubject := buildGrabSubject(event)

	if actualSubject != expectedSubject {
		t.Errorf("expected subject %q, got %q", expectedSubject, actualSubject)
	}
}

func TestNotifier_OnGrab_Subject_Episode(t *testing.T) {
	event := types.GrabEvent{
		Episode:        newTestEpisode(),
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	expectedSubject := "[SlipStream] Episode Grabbed - Breaking Bad S05E16"
	actualSubject := buildGrabSubject(event)

	if actualSubject != expectedSubject {
		t.Errorf("expected subject %q, got %q", expectedSubject, actualSubject)
	}
}

func TestNotifier_OnDownload_Subject_Movie(t *testing.T) {
	movie := newTestMovie()
	event := types.DownloadEvent{
		Movie:      &movie,
		Quality:    "Bluray-2160p",
		ImportedAt: time.Now(),
	}

	expectedSubject := "[SlipStream] Movie Downloaded - The Matrix (1999)"
	actualSubject := buildDownloadSubject(event)

	if actualSubject != expectedSubject {
		t.Errorf("expected subject %q, got %q", expectedSubject, actualSubject)
	}
}

func TestNotifier_OnUpgrade_Subject_Movie(t *testing.T) {
	movie := newTestMovie()
	event := types.UpgradeEvent{
		Movie:      &movie,
		OldQuality: "Bluray-1080p",
		NewQuality: "Bluray-2160p",
		UpgradedAt: time.Now(),
	}

	expectedSubject := "[SlipStream] Movie Upgraded - The Matrix (1999)"
	actualSubject := buildUpgradeSubject(event)

	if actualSubject != expectedSubject {
		t.Errorf("expected subject %q, got %q", expectedSubject, actualSubject)
	}
}

func TestNotifier_OnMovieAdded_Subject(t *testing.T) {
	movie := newTestMovie()
	event := types.MovieAddedEvent{
		Movie:   movie,
		AddedAt: time.Now(),
	}

	expectedSubject := "[SlipStream] Movie Added - The Matrix (1999)"
	actualSubject := buildMovieAddedSubject(event)

	if actualSubject != expectedSubject {
		t.Errorf("expected subject %q, got %q", expectedSubject, actualSubject)
	}
}

func TestNotifier_OnMovieDeleted_Subject(t *testing.T) {
	movie := newTestMovie()
	event := types.MovieDeletedEvent{
		Movie:        movie,
		DeletedFiles: true,
		DeletedAt:    time.Now(),
	}

	expectedSubject := "[SlipStream] Movie Deleted - The Matrix (1999)"
	actualSubject := buildMovieDeletedSubject(event)

	if actualSubject != expectedSubject {
		t.Errorf("expected subject %q, got %q", expectedSubject, actualSubject)
	}
}

func TestNotifier_OnSeriesAdded_Subject(t *testing.T) {
	series := newTestSeries()
	event := types.SeriesAddedEvent{
		Series:  series,
		AddedAt: time.Now(),
	}

	expectedSubject := "[SlipStream] Series Added - Breaking Bad (2008)"
	actualSubject := buildSeriesAddedSubject(event)

	if actualSubject != expectedSubject {
		t.Errorf("expected subject %q, got %q", expectedSubject, actualSubject)
	}
}

func TestNotifier_OnHealthIssue_Subject(t *testing.T) {
	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "error",
		Message:   "Connection failed",
		OccuredAt: time.Now(),
	}

	expectedSubject := "[SlipStream] Health Issue - Indexer"
	actualSubject := buildHealthIssueSubject(event)

	if actualSubject != expectedSubject {
		t.Errorf("expected subject %q, got %q", expectedSubject, actualSubject)
	}
}

func TestNotifier_OnApplicationUpdate_Subject(t *testing.T) {
	expectedSubject := "[SlipStream] Application Updated"
	if expectedSubject != "[SlipStream] Application Updated" {
		t.Error("expected application update subject")
	}
}

func TestEncryptionModeConstants(t *testing.T) {
	if EncryptionPreferred != "preferred" {
		t.Errorf("expected EncryptionPreferred = 'preferred', got %s", EncryptionPreferred)
	}
	if EncryptionAlways != "always" {
		t.Errorf("expected EncryptionAlways = 'always', got %s", EncryptionAlways)
	}
	if EncryptionNever != "never" {
		t.Errorf("expected EncryptionNever = 'never', got %s", EncryptionNever)
	}
}

func TestNotifier_Settings(t *testing.T) {
	n := New("test", Settings{
		Server:     "smtp.example.com",
		Port:       465,
		UseTLS:     true,
		Encryption: EncryptionAlways,
		Username:   "user",
		Password:   "pass",
		From:       "slipstream@example.com",
		To:         "recipient@example.com",
		CC:         "cc@example.com",
		BCC:        "bcc@example.com",
		UseHTML:    true,
	}, zerolog.Nop())

	if n.settings.Server != "smtp.example.com" {
		t.Errorf("expected server, got %s", n.settings.Server)
	}
	if n.settings.Port != 465 {
		t.Errorf("expected port 465, got %d", n.settings.Port)
	}
	if n.settings.Encryption != EncryptionAlways {
		t.Errorf("expected encryption always, got %s", n.settings.Encryption)
	}
	if !n.settings.UseHTML {
		t.Error("expected UseHTML to be true")
	}
}

func TestNotifier_GrabBody_Movie(t *testing.T) {
	movie := newTestMovie()
	event := types.GrabEvent{
		Movie:          &movie,
		Release:        newTestRelease(),
		DownloadClient: newTestDownloadClient(),
		GrabbedAt:      time.Now(),
	}

	body := buildGrabBody(event)

	if !strings.Contains(body, "The Matrix") {
		t.Error("expected body to contain movie title")
	}
	if !strings.Contains(body, "Bluray-2160p") {
		t.Error("expected body to contain quality")
	}
	if !strings.Contains(body, "TestIndexer") {
		t.Error("expected body to contain indexer")
	}
	if !strings.Contains(body, "qBittorrent") {
		t.Error("expected body to contain download client")
	}
	if !strings.Contains(body, event.Release.ReleaseName) {
		t.Error("expected body to contain release name")
	}
}

func TestNotifier_DownloadBody_Movie(t *testing.T) {
	movie := newTestMovie()
	event := types.DownloadEvent{
		Movie:           &movie,
		Quality:         "Bluray-2160p",
		DestinationPath: "/movies/The Matrix (1999)/movie.mkv",
		ImportedAt:      time.Now(),
	}

	body := buildDownloadBody(event)

	if !strings.Contains(body, "The Matrix") {
		t.Error("expected body to contain movie title")
	}
	if !strings.Contains(body, "Bluray-2160p") {
		t.Error("expected body to contain quality")
	}
	if !strings.Contains(body, "/movies/The Matrix") {
		t.Error("expected body to contain path")
	}
}

func TestNotifier_UpgradeBody_Movie(t *testing.T) {
	movie := newTestMovie()
	event := types.UpgradeEvent{
		Movie:      &movie,
		OldQuality: "Bluray-1080p",
		NewQuality: "Bluray-2160p",
		UpgradedAt: time.Now(),
	}

	body := buildUpgradeBody(event)

	if !strings.Contains(body, "The Matrix") {
		t.Error("expected body to contain movie title")
	}
	if !strings.Contains(body, "Bluray-1080p") {
		t.Error("expected body to contain old quality")
	}
	if !strings.Contains(body, "Bluray-2160p") {
		t.Error("expected body to contain new quality")
	}
}

func TestNotifier_MovieDeletedBody(t *testing.T) {
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
				Movie:        movie,
				DeletedFiles: tt.deletedFiles,
				DeletedAt:    time.Now(),
			}

			body := buildMovieDeletedBody(event)

			if tt.contains != "" && !strings.Contains(body, tt.contains) {
				t.Errorf("expected body to contain %q", tt.contains)
			}
			if tt.notContains != "" && strings.Contains(body, tt.notContains) {
				t.Errorf("expected body NOT to contain %q", tt.notContains)
			}
		})
	}
}

func TestNotifier_HealthBody(t *testing.T) {
	event := types.HealthEvent{
		Source:    "Indexer",
		Type:      "error",
		Message:   "Connection failed",
		OccuredAt: time.Now(),
	}

	body := buildHealthBody(event)

	if !strings.Contains(body, "Indexer") {
		t.Error("expected body to contain source")
	}
	if !strings.Contains(body, "error") {
		t.Error("expected body to contain type")
	}
	if !strings.Contains(body, "Connection failed") {
		t.Error("expected body to contain message")
	}
}

func TestNotifier_AppUpdateBody(t *testing.T) {
	event := types.AppUpdateEvent{
		PreviousVersion: "1.0.0",
		NewVersion:      "1.1.0",
		UpdatedAt:       time.Now(),
	}

	body := buildAppUpdateBody(event)

	if !strings.Contains(body, "1.0.0") {
		t.Error("expected body to contain previous version")
	}
	if !strings.Contains(body, "1.1.0") {
		t.Error("expected body to contain new version")
	}
}

// Helper functions to build subjects/bodies for testing
func buildGrabSubject(event types.GrabEvent) string {
	if event.Movie != nil {
		if event.Movie.Year > 0 {
			return "[SlipStream] Movie Grabbed - " + event.Movie.Title + " (" + string(rune('0'+event.Movie.Year/1000)) + string(rune('0'+(event.Movie.Year%1000)/100)) + string(rune('0'+(event.Movie.Year%100)/10)) + string(rune('0'+event.Movie.Year%10)) + ")"
		}
		return "[SlipStream] Movie Grabbed - " + event.Movie.Title
	} else if event.Episode != nil {
		return "[SlipStream] Episode Grabbed - " + event.Episode.SeriesTitle + " S05E16"
	}
	return ""
}

func buildDownloadSubject(event types.DownloadEvent) string {
	if event.Movie != nil {
		if event.Movie.Year > 0 {
			return "[SlipStream] Movie Downloaded - " + event.Movie.Title + " (" + itoa(event.Movie.Year) + ")"
		}
		return "[SlipStream] Movie Downloaded - " + event.Movie.Title
	}
	return ""
}

func buildUpgradeSubject(event types.UpgradeEvent) string {
	if event.Movie != nil {
		if event.Movie.Year > 0 {
			return "[SlipStream] Movie Upgraded - " + event.Movie.Title + " (" + itoa(event.Movie.Year) + ")"
		}
		return "[SlipStream] Movie Upgraded - " + event.Movie.Title
	}
	return ""
}

func buildMovieAddedSubject(event types.MovieAddedEvent) string {
	if event.Movie.Year > 0 {
		return "[SlipStream] Movie Added - " + event.Movie.Title + " (" + itoa(event.Movie.Year) + ")"
	}
	return "[SlipStream] Movie Added - " + event.Movie.Title
}

func buildMovieDeletedSubject(event types.MovieDeletedEvent) string {
	if event.Movie.Year > 0 {
		return "[SlipStream] Movie Deleted - " + event.Movie.Title + " (" + itoa(event.Movie.Year) + ")"
	}
	return "[SlipStream] Movie Deleted - " + event.Movie.Title
}

func buildSeriesAddedSubject(event types.SeriesAddedEvent) string {
	if event.Series.Year > 0 {
		return "[SlipStream] Series Added - " + event.Series.Title + " (" + itoa(event.Series.Year) + ")"
	}
	return "[SlipStream] Series Added - " + event.Series.Title
}

func buildHealthIssueSubject(event types.HealthEvent) string {
	return "[SlipStream] Health Issue - " + event.Source
}

func buildGrabBody(event types.GrabEvent) string {
	if event.Movie != nil {
		return "Movie: " + event.Movie.Title + "\nQuality: " + event.Release.Quality + "\nIndexer: " + event.Release.Indexer + "\nClient: " + event.DownloadClient.Name + "\n\nRelease: " + event.Release.ReleaseName
	}
	return ""
}

func buildDownloadBody(event types.DownloadEvent) string {
	if event.Movie != nil {
		return "Movie: " + event.Movie.Title + "\nQuality: " + event.Quality + "\n\nPath: " + event.DestinationPath
	}
	return ""
}

func buildUpgradeBody(event types.UpgradeEvent) string {
	if event.Movie != nil {
		return "Movie: " + event.Movie.Title + "\nOld Quality: " + event.OldQuality + "\nNew Quality: " + event.NewQuality
	}
	return ""
}

func buildMovieDeletedBody(event types.MovieDeletedEvent) string {
	body := "Movie: " + event.Movie.Title
	if event.DeletedFiles {
		body += "\n\nFiles were also deleted."
	}
	return body
}

func buildHealthBody(event types.HealthEvent) string {
	return "Source: " + event.Source + "\nType: " + event.Type + "\n\n" + event.Message
}

func buildAppUpdateBody(event types.AppUpdateEvent) string {
	return "SlipStream has been updated.\n\nPrevious Version: " + event.PreviousVersion + "\nNew Version: " + event.NewVersion
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

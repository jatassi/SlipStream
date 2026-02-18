package arrimport

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/downloader"
	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/notification"
	"github.com/slipstream/slipstream/internal/testutil"
)

// stubIndexerService implements IndexerImportService without needing cardigann.
type stubIndexerService struct {
	created []*indexer.IndexerDefinition
	nextID  int64
}

func (s *stubIndexerService) Create(_ context.Context, input *indexer.CreateIndexerInput) (*indexer.IndexerDefinition, error) {
	s.nextID++
	def := &indexer.IndexerDefinition{
		ID:           s.nextID,
		Name:         input.Name,
		DefinitionID: input.DefinitionID,
	}
	s.created = append(s.created, def)
	return def, nil
}

func (s *stubIndexerService) List(_ context.Context) ([]*indexer.IndexerDefinition, error) {
	return s.created, nil
}

// stubImportSettingsService implements ImportSettingsService backed by in-memory settings.
type stubImportSettingsService struct {
	settings *importer.ImportSettings
}

func newStubImportSettingsService() *stubImportSettingsService {
	s := importer.DefaultImportSettings()
	return &stubImportSettingsService{settings: &s}
}

func (s *stubImportSettingsService) GetSettings(_ context.Context) (*importer.ImportSettings, error) {
	return s.settings, nil
}

func (s *stubImportSettingsService) UpdateSettings(_ context.Context, settings *importer.ImportSettings) (*importer.ImportSettings, error) {
	s.settings = settings
	return settings, nil
}

// setupRadarrSourceDB creates a temporary Radarr-style SQLite DB with test data.
func setupRadarrSourceDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "radarr.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open source db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		t.Fatalf("set WAL mode: %v", err)
	}

	// Create required tables
	for _, ddl := range []string{
		`CREATE TABLE Movies (Id INTEGER PRIMARY KEY)`,
		`CREATE TABLE RootFolders (Id INTEGER, Path TEXT)`,
		`CREATE TABLE QualityDefinitions (Quality INTEGER, Title TEXT)`,
		`CREATE TABLE QualityProfiles (Id INTEGER, Name TEXT, Cutoff INTEGER, UpgradeAllowed INTEGER, Items TEXT)`,
		`CREATE TABLE DownloadClients (Id INTEGER, Name TEXT, Implementation TEXT, Settings TEXT,
			Enable INTEGER, Priority INTEGER, RemoveCompletedDownloads INTEGER, RemoveFailedDownloads INTEGER)`,
		`CREATE TABLE Indexers (Id INTEGER, Name TEXT, Implementation TEXT, Settings TEXT,
			EnableRss INTEGER, EnableAutomaticSearch INTEGER, EnableInteractiveSearch INTEGER, Priority INTEGER)`,
		`CREATE TABLE Notifications (Id INTEGER, Name TEXT, Implementation TEXT, Settings TEXT,
			OnGrab INTEGER, OnDownload INTEGER, OnUpgrade INTEGER,
			OnHealthIssue INTEGER, IncludeHealthWarnings INTEGER, OnHealthRestored INTEGER, OnApplicationUpdate INTEGER,
			OnMovieAdded INTEGER, OnMovieDelete INTEGER)`,
		`CREATE TABLE NamingConfig (Id INTEGER, RenameMovies INTEGER, ReplaceIllegalCharacters INTEGER,
			ColonReplacementFormat INTEGER, StandardMovieFormat TEXT, MovieFolderFormat TEXT)`,
	} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			t.Fatalf("create table: %v", err)
		}
	}

	// Insert test data — use CAST(? AS BLOB) for JSON fields so modernc.org/sqlite
	// returns them as []byte, which json.RawMessage ([]byte) can scan.
	execOrFatal := func(query string, args ...any) {
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("exec %q: %v", query[:40], err)
		}
	}

	dlSettings1 := []byte(`{"host":"seedbox.example.com","port":9091,"username":"user","password":"pass","useSsl":false,"movieCategory":"radarr","urlBase":"/transmission/"}`)
	execOrFatal(`INSERT INTO DownloadClients VALUES (1, 'My Transmission', 'Transmission', CAST(? AS BLOB), 1, 1, 1, 0)`, dlSettings1)

	dlSettings2 := []byte(`{"host":"localhost","port":8080}`)
	execOrFatal(`INSERT INTO DownloadClients VALUES (2, 'My Sabnzbd', 'Sabnzbd', CAST(? AS BLOB), 1, 2, 0, 0)`, dlSettings2)

	idxSettings := []byte(`{"baseUrl":"https://torznab.example.com","apiKey":"abc123","categories":[2000,5040]}`)
	execOrFatal(`INSERT INTO Indexers VALUES (1, 'My Torznab', 'Torznab', CAST(? AS BLOB), 1, 1, 1, 25)`, idxSettings)

	notifSettings := []byte(`{"webHookUrl":"https://discord.com/hook","username":"bot"}`)
	execOrFatal(`INSERT INTO Notifications VALUES (1, 'My Discord', 'Discord', CAST(? AS BLOB), 1, 1, 0, 1, 1, 0, 0, 1, 0)`, notifSettings)

	profileItems := []byte(`[
		{"quality":{"id":1,"name":"SDTV"},"items":[],"allowed":false},
		{"quality":{"id":9,"name":"HDTV-1080p"},"items":[],"allowed":true},
		{"id":1002,"name":"WEB 1080p","items":[
			{"quality":{"id":15,"name":"WEBRip-1080p"},"items":[],"allowed":true},
			{"quality":{"id":3,"name":"WEBDL-1080p"},"items":[],"allowed":true}
		],"allowed":true},
		{"quality":{"id":7,"name":"Bluray-1080p"},"items":[],"allowed":true}
	]`)
	execOrFatal(`INSERT INTO QualityProfiles VALUES (4, 'HD-1080p', 9, 1, CAST(? AS BLOB))`, profileItems)

	execOrFatal(`INSERT INTO NamingConfig VALUES (1, 1, 1, 4, '{Movie Title} ({Release Year})', '{Movie Title} ({Release Year})')`)

	return path
}

func TestConfigImportEndToEnd(t *testing.T) {
	// SlipStream destination DB
	tdb := testutil.NewTestDB(t)
	defer tdb.Close()

	// Source Radarr DB
	srcPath := setupRadarrSourceDB(t)

	logger := zerolog.Nop()

	// Real destination services
	dlSvc := downloader.NewService(tdb.Conn, &logger)
	notifSvc := notification.NewService(tdb.Conn, &logger)
	qualProfSvc := quality.NewService(tdb.Conn, &logger)

	// Stubs for services with heavy dependency chains
	idxSvc := &stubIndexerService{}
	importSettingsSvc := newStubImportSettingsService()

	// Build arrimport service (nil for media import dependencies — not needed for config import)
	svc := NewService(tdb.Conn, nil, nil, nil, nil, nil, nil, &logger)
	svc.SetConfigImportServices(dlSvc, idxSvc, notifSvc, qualProfSvc, importSettingsSvc)

	ctx := context.Background()

	// Connect to source
	if err := svc.Connect(ctx, ConnectionConfig{
		SourceType: SourceTypeRadarr,
		DBPath:     srcPath,
	}); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer svc.Disconnect(ctx) //nolint:errcheck // cleanup in test

	// --- Phase 1: GetConfigPreview ---
	preview, err := svc.GetConfigPreview(ctx)
	if err != nil {
		t.Fatalf("GetConfigPreview: %v", err)
	}

	// Download clients: 1 supported (Transmission), 1 unsupported (Sabnzbd)
	if len(preview.DownloadClients) != 2 {
		t.Fatalf("expected 2 download clients in preview, got %d", len(preview.DownloadClients))
	}
	assertPreviewItem(t, preview.DownloadClients, "My Transmission", previewStatusNew, "transmission")
	assertPreviewItem(t, preview.DownloadClients, "My Sabnzbd", previewStatusUnsupported, "")

	// Indexer
	if len(preview.Indexers) != 1 {
		t.Fatalf("expected 1 indexer in preview, got %d", len(preview.Indexers))
	}
	assertPreviewItem(t, preview.Indexers, "My Torznab", previewStatusNew, "torznab")

	// Notification
	if len(preview.Notifications) != 1 {
		t.Fatalf("expected 1 notification in preview, got %d", len(preview.Notifications))
	}
	assertPreviewItem(t, preview.Notifications, "My Discord", previewStatusNew, "discord")

	// Quality profile
	if len(preview.QualityProfiles) != 1 {
		t.Fatalf("expected 1 quality profile in preview, got %d", len(preview.QualityProfiles))
	}
	assertPreviewItem(t, preview.QualityProfiles, "HD-1080p", previewStatusNew, "quality_profile")

	// Naming config
	if preview.NamingConfig == nil {
		t.Fatal("expected naming config in preview")
	}
	if preview.NamingConfig.Status != "different" {
		t.Errorf("expected naming config status 'different', got %q", preview.NamingConfig.Status)
	}

	// --- Phase 2: ExecuteConfigImport ---
	selections := &ConfigImportSelections{
		DownloadClientIDs:  []int64{1, 2}, // includes unsupported Sabnzbd
		IndexerIDs:         []int64{1},
		NotificationIDs:    []int64{1},
		QualityProfileIDs:  []int64{4},
		ImportNamingConfig: true,
	}

	report, err := svc.ExecuteConfigImport(ctx, selections)
	if err != nil {
		t.Fatalf("ExecuteConfigImport: %v", err)
	}

	if report.DownloadClientsCreated != 1 {
		t.Errorf("download clients created = %d, want 1", report.DownloadClientsCreated)
	}
	if report.DownloadClientsSkipped != 1 {
		t.Errorf("download clients skipped = %d, want 1 (unsupported Sabnzbd)", report.DownloadClientsSkipped)
	}
	if report.IndexersCreated != 1 {
		t.Errorf("indexers created = %d, want 1", report.IndexersCreated)
	}
	if report.NotificationsCreated != 1 {
		t.Errorf("notifications created = %d, want 1", report.NotificationsCreated)
	}
	if report.QualityProfilesCreated != 1 {
		t.Errorf("quality profiles created = %d, want 1", report.QualityProfilesCreated)
	}
	if !report.NamingConfigImported {
		t.Error("naming config should have been imported")
	}
	if len(report.Errors) != 0 {
		t.Errorf("unexpected errors: %v", report.Errors)
	}

	// --- Phase 3: Verify created entities ---

	// Download client
	clients, err := dlSvc.List(ctx)
	if err != nil {
		t.Fatalf("list download clients: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1 download client, got %d", len(clients))
	}
	if clients[0].Name != "My Transmission" {
		t.Errorf("download client name = %q", clients[0].Name)
	}
	if clients[0].Type != "transmission" {
		t.Errorf("download client type = %q", clients[0].Type)
	}

	// Indexer (stub)
	if len(idxSvc.created) != 1 {
		t.Fatalf("expected 1 indexer created, got %d", len(idxSvc.created))
	}
	if idxSvc.created[0].Name != "My Torznab" {
		t.Errorf("indexer name = %q", idxSvc.created[0].Name)
	}
	if idxSvc.created[0].DefinitionID != "torznab" {
		t.Errorf("indexer definition = %q", idxSvc.created[0].DefinitionID)
	}

	// Notification
	notifs, err := notifSvc.List(ctx)
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notifs) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifs))
	}
	if notifs[0].Name != "My Discord" {
		t.Errorf("notification name = %q", notifs[0].Name)
	}
	if notifs[0].Type != notification.NotifierDiscord {
		t.Errorf("notification type = %q", notifs[0].Type)
	}
	// Verify settings key rename (webHookUrl → webhookUrl)
	var notifSettings map[string]any
	if err := json.Unmarshal(notifs[0].Settings, &notifSettings); err != nil {
		t.Fatalf("unmarshal notification settings: %v", err)
	}
	if _, ok := notifSettings["webHookUrl"]; ok {
		t.Error("webHookUrl should have been renamed to webhookUrl")
	}
	if notifSettings["webhookUrl"] != "https://discord.com/hook" {
		t.Errorf("webhookUrl = %v", notifSettings["webhookUrl"])
	}

	// Quality profile
	profiles, err := qualProfSvc.List(ctx)
	if err != nil {
		t.Fatalf("list quality profiles: %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("expected 1 quality profile, got %d", len(profiles))
	}
	if profiles[0].Name != "HD-1080p" {
		t.Errorf("quality profile name = %q", profiles[0].Name)
	}
	if !profiles[0].UpgradesEnabled {
		t.Error("quality profile should have upgrades enabled")
	}
	// Verify all 17 items present
	if len(profiles[0].Items) != 17 {
		t.Errorf("expected 17 quality items, got %d", len(profiles[0].Items))
	}

	// Naming config
	settings, err := importSettingsSvc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("get import settings: %v", err)
	}
	if !settings.RenameMovies {
		t.Error("rename movies should be true")
	}
	if settings.MovieFileFormat != "{Movie Title} ({Release Year})" {
		t.Errorf("movie file format = %q", settings.MovieFileFormat)
	}

	// --- Phase 4: Duplicate detection (re-import should skip) ---
	preview2, err := svc.GetConfigPreview(ctx)
	if err != nil {
		t.Fatalf("GetConfigPreview (round 2): %v", err)
	}
	assertPreviewItem(t, preview2.DownloadClients, "My Transmission", previewStatusDuplicate, "transmission")
	assertPreviewItem(t, preview2.Indexers, "My Torznab", previewStatusDuplicate, "torznab")
	assertPreviewItem(t, preview2.Notifications, "My Discord", previewStatusDuplicate, "discord")
	assertPreviewItem(t, preview2.QualityProfiles, "HD-1080p", previewStatusDuplicate, "quality_profile")

	// Re-import should skip all duplicates
	report2, err := svc.ExecuteConfigImport(ctx, selections)
	if err != nil {
		t.Fatalf("ExecuteConfigImport (round 2): %v", err)
	}
	if report2.DownloadClientsCreated != 0 {
		t.Errorf("round 2: download clients created = %d, want 0", report2.DownloadClientsCreated)
	}
	if report2.IndexersCreated != 0 {
		t.Errorf("round 2: indexers created = %d, want 0", report2.IndexersCreated)
	}
	if report2.NotificationsCreated != 0 {
		t.Errorf("round 2: notifications created = %d, want 0", report2.NotificationsCreated)
	}
	if report2.QualityProfilesCreated != 0 {
		t.Errorf("round 2: quality profiles created = %d, want 0", report2.QualityProfilesCreated)
	}
}

func assertPreviewItem(t *testing.T, items []ConfigPreviewItem, name, expectedStatus, expectedMapped string) {
	t.Helper()
	for _, item := range items {
		if item.SourceName == name {
			if item.Status != expectedStatus {
				t.Errorf("preview %q: status = %q, want %q", name, item.Status, expectedStatus)
			}
			if item.MappedType != expectedMapped {
				t.Errorf("preview %q: mappedType = %q, want %q", name, item.MappedType, expectedMapped)
			}
			return
		}
	}
	t.Errorf("preview item %q not found", name)
}

package arrimport

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/downloader"
	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/notification"
	"github.com/slipstream/slipstream/internal/progress"
)

const (
	previewStatusSkip        = "skip"
	previewStatusNew         = "new"
	previewStatusDuplicate   = "duplicate"
	previewStatusUnsupported = "unsupported"
	previewStatusIncomplete  = "incomplete"

	statusReasonRedacted = "credentials are redacted (API connection)"
)

// ModuleRegistry provides access to module adapters for arr import.
type ModuleRegistry interface {
	GetMovieArrAdapter() module.MovieArrImportAdapter
	GetTVArrAdapter() module.TVArrImportAdapter
}

// RootFolderService defines the interface for root folder operations.
type RootFolderService interface {
	List(ctx context.Context) ([]*rootfolder.RootFolder, error)
}

// QualityService defines the interface for quality profile operations.
type QualityService interface {
	List(ctx context.Context) ([]*quality.Profile, error)
}

// DownloadClientImportService defines the interface for creating download clients during config import.
type DownloadClientImportService interface {
	Create(ctx context.Context, input *downloader.CreateClientInput) (*downloader.DownloadClient, error)
	List(ctx context.Context) ([]*downloader.DownloadClient, error)
}

// IndexerImportService defines the interface for creating indexers during config import.
type IndexerImportService interface {
	Create(ctx context.Context, input *indexer.CreateIndexerInput) (*indexer.IndexerDefinition, error)
	List(ctx context.Context) ([]*indexer.IndexerDefinition, error)
}

// NotificationImportService defines the interface for creating notifications during config import.
type NotificationImportService interface {
	Create(ctx context.Context, input *notification.CreateInput) (*notification.Config, error)
	List(ctx context.Context) ([]notification.Config, error)
}

// QualityProfileImportService defines the interface for creating quality profiles during config import.
type QualityProfileImportService interface {
	Create(ctx context.Context, input *quality.CreateProfileInput) (*quality.Profile, error)
	List(ctx context.Context) ([]*quality.Profile, error)
}

// ImportSettingsService defines the interface for reading/updating import settings during config import.
type ImportSettingsService interface {
	GetSettings(ctx context.Context) (*importer.ImportSettings, error)
	UpdateSettings(ctx context.Context, settings *importer.ImportSettings) (*importer.ImportSettings, error)
}

// Service manages library imports from external sources.
type Service struct {
	db                *sql.DB
	reader            Reader
	sourceType        SourceType
	registry          ModuleRegistry
	rootFolderService RootFolderService
	qualityService    QualityService
	progressManager   *progress.Manager
	logger            *zerolog.Logger
	mu                sync.Mutex

	// Config import services (set via SetConfigImportServices)
	dlClientService       DownloadClientImportService
	indexerService        IndexerImportService
	notifService          NotificationImportService
	qualityProfService    QualityProfileImportService
	importSettingsService ImportSettingsService
}

// NewService creates a new library import service.
func NewService(
	db *sql.DB,
	registry ModuleRegistry,
	rootFolderService RootFolderService,
	qualityService QualityService,
	progressManager *progress.Manager,
	logger *zerolog.Logger,
) *Service {
	return &Service{
		db:                db,
		registry:          registry,
		rootFolderService: rootFolderService,
		qualityService:    qualityService,
		progressManager:   progressManager,
		logger:            logger,
	}
}

// SetConfigImportServices sets the services required for config import.
func (s *Service) SetConfigImportServices(
	dlClient DownloadClientImportService,
	idx IndexerImportService,
	notif NotificationImportService,
	qualityProf QualityProfileImportService,
	importSettings ImportSettingsService,
) {
	s.dlClientService = dlClient
	s.indexerService = idx
	s.notifService = notif
	s.qualityProfService = qualityProf
	s.importSettingsService = importSettings
}

// Connect establishes a connection to the source application.
func (s *Service) Connect(ctx context.Context, cfg ConnectionConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reader, err := NewReader(cfg)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}

	if err := reader.Validate(ctx); err != nil {
		return fmt.Errorf("failed to validate connection: %w", err)
	}

	s.reader = reader
	s.sourceType = cfg.SourceType
	s.logger.Info().Str("sourceType", string(cfg.SourceType)).Msg("connected to source")

	return nil
}

// GetSourceRootFolders retrieves the list of root folders from the source.
func (s *Service) GetSourceRootFolders(ctx context.Context) ([]SourceRootFolder, error) {
	s.mu.Lock()
	reader := s.reader
	s.mu.Unlock()

	if reader == nil {
		return nil, fmt.Errorf("not connected")
	}

	return reader.ReadRootFolders(ctx)
}

// GetSourceQualityProfiles retrieves the list of quality profiles from the source.
func (s *Service) GetSourceQualityProfiles(ctx context.Context) ([]SourceQualityProfile, error) {
	s.mu.Lock()
	reader := s.reader
	s.mu.Unlock()

	if reader == nil {
		return nil, fmt.Errorf("not connected")
	}

	return reader.ReadQualityProfiles(ctx)
}

// Preview generates a preview of what will be imported without making changes.
func (s *Service) Preview(ctx context.Context, mappings ImportMappings) (*ImportPreview, error) {
	s.mu.Lock()
	reader := s.reader
	sourceType := s.sourceType
	s.mu.Unlock()

	if reader == nil {
		return nil, fmt.Errorf("not connected")
	}

	preview := &ImportPreview{
		Movies:  []MoviePreview{},
		Series:  []SeriesPreview{},
		Summary: ImportSummary{},
	}

	adapted := &readerAdapter{inner: reader}

	if sourceType == SourceTypeRadarr {
		if err := s.previewMovies(ctx, adapted, preview); err != nil {
			return nil, err
		}
	}

	if sourceType == SourceTypeSonarr {
		if err := s.previewSeries(ctx, adapted, preview); err != nil {
			return nil, err
		}
	}

	return preview, nil
}

func (s *Service) previewMovies(ctx context.Context, reader module.ArrReader, preview *ImportPreview) error {
	adapter := s.registry.GetMovieArrAdapter()
	if adapter == nil {
		return fmt.Errorf("no movie arr import adapter registered")
	}
	items, err := adapter.PreviewMovies(ctx, reader)
	if err != nil {
		return fmt.Errorf("failed to preview movies: %w", err)
	}
	for i := range items {
		preview.Movies = append(preview.Movies, MoviePreview{
			Title:            items[i].Title,
			Year:             items[i].Year,
			TmdbID:           items[i].TmdbID,
			HasFile:          items[i].HasFile,
			Quality:          items[i].Quality,
			Monitored:        items[i].Monitored,
			QualityProfileID: items[i].QualityProfileID,
			PosterURL:        items[i].PosterURL,
			Status:           items[i].Status,
			SkipReason:       items[i].SkipReason,
		})
		preview.Summary.TotalMovies++
		switch items[i].Status {
		case "new":
			preview.Summary.NewMovies++
		case "duplicate":
			preview.Summary.DuplicateMovies++
		case "skip":
			preview.Summary.SkippedMovies++
		}
		if items[i].HasFile {
			preview.Summary.TotalFiles++
		}
	}
	return nil
}

func (s *Service) previewSeries(ctx context.Context, reader module.ArrReader, preview *ImportPreview) error {
	adapter := s.registry.GetTVArrAdapter()
	if adapter == nil {
		return fmt.Errorf("no TV arr import adapter registered")
	}
	items, err := adapter.PreviewSeries(ctx, reader)
	if err != nil {
		return fmt.Errorf("failed to preview series: %w", err)
	}
	for i := range items {
		preview.Series = append(preview.Series, SeriesPreview{
			Title:            items[i].Title,
			Year:             items[i].Year,
			TvdbID:           items[i].TvdbID,
			TmdbID:           items[i].TmdbID,
			EpisodeCount:     items[i].EpisodeCount,
			FileCount:        items[i].FileCount,
			Monitored:        items[i].Monitored,
			QualityProfileID: items[i].QualityProfileID,
			PosterURL:        items[i].PosterURL,
			Status:           items[i].Status,
			SkipReason:       items[i].SkipReason,
		})
		preview.Summary.TotalSeries++
		preview.Summary.TotalEpisodes += items[i].EpisodeCount
		preview.Summary.TotalFiles += items[i].FileCount
		switch items[i].Status {
		case "new":
			preview.Summary.NewSeries++
		case "duplicate":
			preview.Summary.DuplicateSeries++
		case "skip":
			preview.Summary.SkippedSeries++
		}
	}
	return nil
}

// Execute starts the import process asynchronously.
// The actual import is handled by an Executor running in a goroutine.
// Progress is tracked via the progress manager and broadcast over WebSocket.
func (s *Service) Execute(ctx context.Context, mappings ImportMappings) error {
	s.mu.Lock()
	if s.reader == nil {
		s.mu.Unlock()
		return fmt.Errorf("not connected")
	}
	reader := s.reader
	sourceType := s.sourceType
	s.mu.Unlock()

	executor := NewExecutor(s.db, reader, sourceType, s.registry, s.progressManager, s.logger)
	go executor.Run(context.Background(), mappings)

	return nil
}

// GetConfigPreview reads config entities from the source and generates a preview.
func (s *Service) GetConfigPreview(ctx context.Context) (*ConfigPreview, error) {
	s.mu.Lock()
	reader := s.reader
	sourceType := s.sourceType
	s.mu.Unlock()

	if reader == nil {
		return nil, fmt.Errorf("not connected")
	}

	preview := &ConfigPreview{
		DownloadClients: []ConfigPreviewItem{},
		Indexers:        []ConfigPreviewItem{},
		Notifications:   []ConfigPreviewItem{},
		QualityProfiles: []ConfigPreviewItem{},
		Warnings:        []string{},
	}

	existingClients, _ := s.dlClientService.List(ctx)
	existingIndexers, _ := s.indexerService.List(ctx)
	existingNotifs, _ := s.notifService.List(ctx)
	existingProfiles, _ := s.qualityProfService.List(ctx)

	clientNames := nameSet(existingClients, func(c *downloader.DownloadClient) string { return c.Name })
	indexerNames := nameSet(existingIndexers, func(i *indexer.IndexerDefinition) string { return i.Name })
	notifNames := nameSetVal(existingNotifs, func(n notification.Config) string { return n.Name })
	profileNames := nameSet(existingProfiles, func(p *quality.Profile) string { return p.Name })

	s.previewDownloadClients(ctx, reader, clientNames, preview)
	s.previewIndexers(ctx, reader, indexerNames, preview)
	s.previewNotifications(ctx, reader, notifNames, preview)
	s.previewQualityProfiles(ctx, reader, profileNames, preview)
	s.previewNamingConfig(ctx, reader, sourceType, preview)

	return preview, nil
}

func (s *Service) previewDownloadClients(ctx context.Context, reader Reader, existingNames map[string]bool, preview *ConfigPreview) {
	clients, err := reader.ReadDownloadClients(ctx)
	if err != nil {
		preview.Warnings = append(preview.Warnings, "failed to read download clients: "+err.Error())
		return
	}
	for _, c := range clients {
		item := ConfigPreviewItem{
			SourceID:   c.ID,
			SourceName: c.Name,
			SourceType: c.Implementation,
			MappedType: downloadClientTypeMap[c.Implementation],
		}
		switch {
		case item.MappedType == "":
			item.Status = previewStatusUnsupported
			item.StatusReason = c.Implementation + " is not supported"
		case existingNames[strings.ToLower(c.Name)]:
			item.Status = previewStatusDuplicate
		case hasRedactedCredentials(c.Settings):
			item.Status = previewStatusIncomplete
			item.StatusReason = statusReasonRedacted
		default:
			item.Status = previewStatusNew
		}
		preview.DownloadClients = append(preview.DownloadClients, item)
	}
}

func (s *Service) previewIndexers(ctx context.Context, reader Reader, existingNames map[string]bool, preview *ConfigPreview) {
	indexers, err := reader.ReadIndexers(ctx)
	if err != nil {
		preview.Warnings = append(preview.Warnings, "failed to read indexers: "+err.Error())
		return
	}
	for _, idx := range indexers {
		item := ConfigPreviewItem{
			SourceID:   idx.ID,
			SourceName: idx.Name,
			SourceType: idx.Implementation,
			MappedType: indexerImplementationToDefinitionID(idx.Implementation),
		}
		switch {
		case existingNames[strings.ToLower(idx.Name)]:
			item.Status = previewStatusDuplicate
		case hasRedactedCredentials(idx.Settings):
			item.Status = previewStatusIncomplete
			item.StatusReason = statusReasonRedacted
		default:
			item.Status = previewStatusNew
		}
		preview.Indexers = append(preview.Indexers, item)
	}
}

func (s *Service) previewNotifications(ctx context.Context, reader Reader, existingNames map[string]bool, preview *ConfigPreview) {
	notifs, err := reader.ReadNotifications(ctx)
	if err != nil {
		preview.Warnings = append(preview.Warnings, "failed to read notifications: "+err.Error())
		return
	}
	for _, n := range notifs {
		item := ConfigPreviewItem{
			SourceID:   n.ID,
			SourceName: n.Name,
			SourceType: n.Implementation,
		}
		mappedType, supported := notificationTypeMap[n.Implementation]
		if supported {
			item.MappedType = string(mappedType)
		}
		switch {
		case !supported:
			item.Status = previewStatusUnsupported
			item.StatusReason = n.Implementation + " is not supported"
		case existingNames[strings.ToLower(n.Name)]:
			item.Status = previewStatusDuplicate
		case hasRedactedCredentials(n.Settings):
			item.Status = previewStatusIncomplete
			item.StatusReason = statusReasonRedacted
		default:
			item.Status = previewStatusNew
		}
		preview.Notifications = append(preview.Notifications, item)
	}
}

func (s *Service) previewQualityProfiles(ctx context.Context, reader Reader, existingNames map[string]bool, preview *ConfigPreview) {
	profiles, err := reader.ReadQualityProfilesFull(ctx)
	if err != nil {
		preview.Warnings = append(preview.Warnings, "failed to read quality profiles: "+err.Error())
		return
	}
	for _, p := range profiles {
		item := ConfigPreviewItem{
			SourceID:   p.ID,
			SourceName: p.Name,
			SourceType: "quality_profile",
			MappedType: "quality_profile",
		}
		if existingNames[strings.ToLower(p.Name)] {
			item.Status = previewStatusDuplicate
		} else {
			item.Status = previewStatusNew
		}
		preview.QualityProfiles = append(preview.QualityProfiles, item)
	}
}

func (s *Service) previewNamingConfig(ctx context.Context, reader Reader, sourceType SourceType, preview *ConfigPreview) {
	nc, err := reader.ReadNamingConfig(ctx)
	if err != nil {
		preview.Warnings = append(preview.Warnings, "failed to read naming config: "+err.Error())
		return
	}
	currentSettings, settingsErr := s.importSettingsService.GetSettings(ctx)
	status := "different"
	if settingsErr == nil && namingConfigSame(nc, sourceType, currentSettings) {
		status = "same"
	}
	preview.NamingConfig = &NamingConfigPreview{
		Source: *nc,
		Status: status,
	}
}

// ExecuteConfigImport imports selected config entities from the source.
func (s *Service) ExecuteConfigImport(ctx context.Context, selections *ConfigImportSelections) (*ConfigImportReport, error) {
	s.mu.Lock()
	reader := s.reader
	sourceType := s.sourceType
	s.mu.Unlock()

	if reader == nil {
		return nil, fmt.Errorf("not connected")
	}

	report := newConfigImportReport()

	// Import download clients
	if len(selections.DownloadClientIDs) > 0 {
		s.importDownloadClients(ctx, reader, sourceType, selections.DownloadClientIDs, report)
	}

	// Import indexers
	if len(selections.IndexerIDs) > 0 {
		s.importIndexers(ctx, reader, selections.IndexerIDs, report)
	}

	// Import notifications
	if len(selections.NotificationIDs) > 0 {
		s.importNotifications(ctx, reader, sourceType, selections.NotificationIDs, report)
	}

	// Import quality profiles
	if len(selections.QualityProfileIDs) > 0 {
		s.importQualityProfiles(ctx, reader, sourceType, selections.QualityProfileIDs, report)
	}

	// Import naming config
	if selections.ImportNamingConfig {
		s.importNamingConfig(ctx, reader, sourceType, report)
	}

	return report, nil
}

func (s *Service) importDownloadClients(ctx context.Context, reader Reader, sourceType SourceType, selectedIDs []int64, report *ConfigImportReport) {
	clients, err := reader.ReadDownloadClients(ctx)
	if err != nil {
		report.Errors = append(report.Errors, "failed to read download clients: "+err.Error())
		return
	}

	existingClients, _ := s.dlClientService.List(ctx)
	existingNames := nameSet(existingClients, func(c *downloader.DownloadClient) string { return c.Name })

	for _, c := range clients {
		if !slices.Contains(selectedIDs, c.ID) {
			continue
		}

		mappedType := downloadClientTypeMap[c.Implementation]
		if mappedType == "" {
			report.DownloadClientsSkipped++
			report.Warnings = append(report.Warnings, fmt.Sprintf("download client %q: %s is not supported", c.Name, c.Implementation))
			continue
		}
		if existingNames[strings.ToLower(c.Name)] {
			report.DownloadClientsSkipped++
			continue
		}

		parsed := translateDownloadClientSettings(c.Settings, sourceType)
		report.Warnings = append(report.Warnings, parsed.Warnings...)

		cleanupMode := "leave"
		if c.RemoveCompletedDownloads {
			cleanupMode = "delete_after_import"
		}

		enabled := c.Enabled
		password := parsed.Password
		apiKey := parsed.APIKey
		if hasRedactedCredentials(c.Settings) {
			enabled = false
			report.Warnings = append(report.Warnings, fmt.Sprintf("download client %q: created with empty credentials and disabled (API connection)", c.Name))
			password = ""
			apiKey = ""
		}

		input := &downloader.CreateClientInput{
			Name:        c.Name,
			Type:        mappedType,
			Host:        parsed.Host,
			Port:        parsed.Port,
			Username:    parsed.Username,
			Password:    password,
			UseSSL:      parsed.UseSSL,
			APIKey:      apiKey,
			Category:    parsed.Category,
			URLBase:     parsed.URLBase,
			Priority:    c.Priority,
			Enabled:     enabled,
			CleanupMode: cleanupMode,
		}

		if _, err := s.dlClientService.Create(ctx, input); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to create download client %q: %v", c.Name, err))
		} else {
			report.DownloadClientsCreated++
		}
	}
}

func (s *Service) importIndexers(ctx context.Context, reader Reader, selectedIDs []int64, report *ConfigImportReport) {
	indexers, err := reader.ReadIndexers(ctx)
	if err != nil {
		report.Errors = append(report.Errors, "failed to read indexers: "+err.Error())
		return
	}

	existingIndexers, _ := s.indexerService.List(ctx)
	existingNames := nameSet(existingIndexers, func(i *indexer.IndexerDefinition) string { return i.Name })

	for _, idx := range indexers {
		if !slices.Contains(selectedIDs, idx.ID) {
			continue
		}
		if existingNames[strings.ToLower(idx.Name)] {
			report.IndexersSkipped++
			continue
		}

		defID := indexerImplementationToDefinitionID(idx.Implementation)
		translatedSettings, categories, warnings := translateIndexerSettings(idx.Settings)
		report.Warnings = append(report.Warnings, warnings...)

		enabled := true
		if hasRedactedCredentials(idx.Settings) {
			enabled = false
			report.Warnings = append(report.Warnings, fmt.Sprintf("indexer %q: created with empty credentials and disabled (API connection)", idx.Name))
		}

		rssEnabled := idx.EnableRss
		autoSearch := idx.EnableAutomaticSearch

		input := &indexer.CreateIndexerInput{
			Name:              idx.Name,
			DefinitionID:      defID,
			Settings:          translatedSettings,
			Categories:        categories,
			SupportsMovies:    true,
			SupportsTV:        true,
			Priority:          idx.Priority,
			Enabled:           enabled,
			AutoSearchEnabled: &autoSearch,
			RssEnabled:        &rssEnabled,
		}

		if _, err := s.indexerService.Create(ctx, input); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to create indexer %q: %v", idx.Name, err))
		} else {
			report.IndexersCreated++
		}
	}
}

func (s *Service) importNotifications(ctx context.Context, reader Reader, sourceType SourceType, selectedIDs []int64, report *ConfigImportReport) {
	notifs, err := reader.ReadNotifications(ctx)
	if err != nil {
		report.Errors = append(report.Errors, "failed to read notifications: "+err.Error())
		return
	}

	existingNotifs, _ := s.notifService.List(ctx)
	existingNames := nameSetVal(existingNotifs, func(n notification.Config) string { return n.Name })

	for _, n := range notifs {
		if !slices.Contains(selectedIDs, n.ID) {
			continue
		}

		mappedType, supported := notificationTypeMap[n.Implementation]
		if !supported {
			report.NotificationsSkipped++
			report.Warnings = append(report.Warnings, fmt.Sprintf("notification %q: %s is not supported", n.Name, n.Implementation))
			continue
		}
		if existingNames[strings.ToLower(n.Name)] {
			report.NotificationsSkipped++
			continue
		}

		translatedSettings, warnings := translateNotificationSettings(n.Implementation, n.Settings)
		report.Warnings = append(report.Warnings, warnings...)

		enabled := true
		if hasRedactedCredentials(n.Settings) {
			enabled = false
			report.Warnings = append(report.Warnings, fmt.Sprintf("notification %q: created disabled (redacted credentials)", n.Name))
		}

		toggles := map[string]bool{
			notification.EventGrab:           n.OnGrab,
			notification.EventImport:         n.OnDownload,
			notification.EventUpgrade:        n.OnUpgrade,
			notification.EventHealthIssue:    n.OnHealthIssue,
			notification.EventHealthRestored: n.OnHealthRestored,
			notification.EventAppUpdate:      n.OnApplicationUpdate,
		}

		// Map source-type-specific event fields
		switch sourceType {
		case SourceTypeSonarr:
			toggles[notification.EventSeriesAdded] = n.OnSeriesAdd
			toggles[notification.EventSeriesDeleted] = n.OnSeriesDelete
		case SourceTypeRadarr:
			toggles[notification.EventMovieAdded] = n.OnMovieAdded
			toggles[notification.EventMovieDeleted] = n.OnMovieDelete
		}

		input := &notification.CreateInput{
			Name:                  n.Name,
			Type:                  mappedType,
			Enabled:               enabled,
			Settings:              translatedSettings,
			EventToggles:          toggles,
			IncludeHealthWarnings: n.IncludeHealthWarnings,
		}

		if _, err := s.notifService.Create(ctx, input); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to create notification %q: %v", n.Name, err))
		} else {
			report.NotificationsCreated++
		}
	}
}

func (s *Service) importQualityProfiles(ctx context.Context, reader Reader, sourceType SourceType, selectedIDs []int64, report *ConfigImportReport) {
	profiles, err := reader.ReadQualityProfilesFull(ctx)
	if err != nil {
		report.Errors = append(report.Errors, "failed to read quality profiles: "+err.Error())
		return
	}

	existingProfiles, _ := s.qualityProfService.List(ctx)
	existingNames := nameSet(existingProfiles, func(p *quality.Profile) string { return p.Name })

	for _, p := range profiles {
		if !slices.Contains(selectedIDs, p.ID) {
			continue
		}
		if existingNames[strings.ToLower(p.Name)] {
			report.QualityProfilesSkipped++
			continue
		}

		items, cutoff, warnings := flattenQualityProfileItems(sourceType, p.Items, p.Cutoff)
		report.Warnings = append(report.Warnings, warnings...)

		// G1: UpgradesEnabled is *bool
		upgradesEnabled := p.UpgradeAllowed

		moduleType := "movie"
		if sourceType == SourceTypeSonarr {
			moduleType = "tv"
		}

		input := &quality.CreateProfileInput{
			Name:            p.Name,
			ModuleType:      moduleType,
			Cutoff:          cutoff,
			UpgradesEnabled: &upgradesEnabled,
			UpgradeStrategy: quality.StrategyBalanced,
			Items:           items,
		}

		if _, err := s.qualityProfService.Create(ctx, input); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("failed to create quality profile %q: %v", p.Name, err))
		} else {
			report.QualityProfilesCreated++
		}
	}
}

func (s *Service) importNamingConfig(ctx context.Context, reader Reader, sourceType SourceType, report *ConfigImportReport) {
	nc, err := reader.ReadNamingConfig(ctx)
	if err != nil {
		report.Errors = append(report.Errors, "failed to read naming config: "+err.Error())
		return
	}

	current, err := s.importSettingsService.GetSettings(ctx)
	if err != nil {
		report.Errors = append(report.Errors, "failed to get current import settings: "+err.Error())
		return
	}

	updated, warnings := translateNamingConfig(nc, sourceType, current)
	report.Warnings = append(report.Warnings, warnings...)

	if _, err := s.importSettingsService.UpdateSettings(ctx, updated); err != nil {
		report.Errors = append(report.Errors, "failed to update import settings: "+err.Error())
	} else {
		report.NamingConfigImported = true
	}
}

// nameSet builds a case-insensitive name lookup from a pointer slice.
func nameSet[T any](items []*T, getName func(*T) string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[strings.ToLower(getName(item))] = true
	}
	return m
}

// nameSetVal builds a case-insensitive name lookup from a value slice.
func nameSetVal[T any](items []T, getName func(T) string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[strings.ToLower(getName(item))] = true
	}
	return m
}

func namingConfigSame(src *SourceNamingConfig, sourceType SourceType, current *importer.ImportSettings) bool {
	translated, _ := translateNamingConfig(src, sourceType, current)
	translatedJSON, _ := json.Marshal(translated)
	currentJSON, _ := json.Marshal(current)
	return bytes.Equal(translatedJSON, currentJSON)
}

// Disconnect closes the connection to the source and clears session state.
func (s *Service) Disconnect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.reader != nil {
		if err := s.reader.Close(); err != nil {
			s.logger.Warn().Err(err).Msg("error closing reader")
		}
		s.reader = nil
	}

	s.sourceType = ""
	s.logger.Info().Msg("disconnected from source")

	return nil
}

// readerAdapter wraps an arrimport.Reader to implement module.ArrReader.
type readerAdapter struct{ inner Reader }

func (a *readerAdapter) ReadRootFolders(ctx context.Context) ([]module.ArrSourceRootFolder, error) {
	folders, err := a.inner.ReadRootFolders(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]module.ArrSourceRootFolder, len(folders))
	for i, f := range folders {
		result[i] = module.ArrSourceRootFolder{ID: f.ID, Path: f.Path}
	}
	return result, nil
}

func (a *readerAdapter) ReadQualityProfiles(ctx context.Context) ([]module.ArrSourceQualityProfile, error) {
	profiles, err := a.inner.ReadQualityProfiles(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]module.ArrSourceQualityProfile, len(profiles))
	for i, p := range profiles {
		result[i] = module.ArrSourceQualityProfile{ID: p.ID, Name: p.Name, InUse: p.InUse}
	}
	return result, nil
}

func (a *readerAdapter) ReadMovies(ctx context.Context) ([]module.ArrSourceMovie, error) {
	movies, err := a.inner.ReadMovies(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]module.ArrSourceMovie, len(movies))
	for i := range movies {
		result[i] = convertSourceMovie(&movies[i])
	}
	return result, nil
}

func (a *readerAdapter) ReadSeries(ctx context.Context) ([]module.ArrSourceSeries, error) {
	seriesList, err := a.inner.ReadSeries(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]module.ArrSourceSeries, len(seriesList))
	for i := range seriesList {
		result[i] = convertSourceSeries(&seriesList[i])
	}
	return result, nil
}

func (a *readerAdapter) ReadEpisodes(ctx context.Context, seriesID int64) ([]module.ArrSourceEpisode, error) {
	episodes, err := a.inner.ReadEpisodes(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	result := make([]module.ArrSourceEpisode, len(episodes))
	for i, ep := range episodes {
		result[i] = module.ArrSourceEpisode{
			ID:            ep.ID,
			SeriesID:      ep.SeriesID,
			SeasonNumber:  ep.SeasonNumber,
			EpisodeNumber: ep.EpisodeNumber,
			Title:         ep.Title,
			Overview:      ep.Overview,
			AirDateUtc:    ep.AirDateUtc,
			Monitored:     ep.Monitored,
			EpisodeFileID: ep.EpisodeFileID,
			HasFile:       ep.HasFile,
		}
	}
	return result, nil
}

func (a *readerAdapter) ReadEpisodeFiles(ctx context.Context, seriesID int64) ([]module.ArrSourceEpisodeFile, error) {
	files, err := a.inner.ReadEpisodeFiles(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	result := make([]module.ArrSourceEpisodeFile, len(files))
	for i := range files {
		result[i] = module.ArrSourceEpisodeFile{
			ID:               files[i].ID,
			SeriesID:         files[i].SeriesID,
			SeasonNumber:     files[i].SeasonNumber,
			RelativePath:     files[i].RelativePath,
			Size:             files[i].Size,
			QualityID:        files[i].QualityID,
			QualityName:      files[i].QualityName,
			VideoCodec:       files[i].VideoCodec,
			AudioCodec:       files[i].AudioCodec,
			Resolution:       files[i].Resolution,
			AudioChannels:    files[i].AudioChannels,
			DynamicRange:     files[i].DynamicRange,
			OriginalFilePath: files[i].OriginalFilePath,
			DateAdded:        files[i].DateAdded,
		}
	}
	return result, nil
}

func convertSourceMovie(m *SourceMovie) module.ArrSourceMovie {
	result := module.ArrSourceMovie{
		ID:               m.ID,
		Title:            m.Title,
		SortTitle:        m.SortTitle,
		Year:             m.Year,
		TmdbID:           m.TmdbID,
		ImdbID:           m.ImdbID,
		Overview:         m.Overview,
		Runtime:          m.Runtime,
		Path:             m.Path,
		RootFolderPath:   m.RootFolderPath,
		QualityProfileID: m.QualityProfileID,
		Monitored:        m.Monitored,
		Status:           m.Status,
		InCinemas:        m.InCinemas,
		PhysicalRelease:  m.PhysicalRelease,
		DigitalRelease:   m.DigitalRelease,
		Studio:           m.Studio,
		Certification:    m.Certification,
		Added:            m.Added,
		HasFile:          m.HasFile,
		PosterURL:        m.PosterURL,
	}
	if m.File != nil {
		result.File = &module.ArrSourceMovieFile{
			ID:               m.File.ID,
			Path:             m.File.Path,
			Size:             m.File.Size,
			QualityID:        m.File.QualityID,
			QualityName:      m.File.QualityName,
			VideoCodec:       m.File.VideoCodec,
			AudioCodec:       m.File.AudioCodec,
			Resolution:       m.File.Resolution,
			AudioChannels:    m.File.AudioChannels,
			DynamicRange:     m.File.DynamicRange,
			OriginalFilePath: m.File.OriginalFilePath,
			DateAdded:        m.File.DateAdded,
		}
	}
	return result
}

func convertSourceSeries(s *SourceSeries) module.ArrSourceSeries {
	result := module.ArrSourceSeries{
		ID:               s.ID,
		Title:            s.Title,
		SortTitle:        s.SortTitle,
		Year:             s.Year,
		TvdbID:           s.TvdbID,
		TmdbID:           s.TmdbID,
		ImdbID:           s.ImdbID,
		Overview:         s.Overview,
		Runtime:          s.Runtime,
		Path:             s.Path,
		RootFolderPath:   s.RootFolderPath,
		QualityProfileID: s.QualityProfileID,
		Monitored:        s.Monitored,
		SeasonFolder:     s.SeasonFolder,
		Status:           s.Status,
		Network:          s.Network,
		SeriesType:       s.SeriesType,
		Certification:    s.Certification,
		Added:            s.Added,
		PosterURL:        s.PosterURL,
	}
	if len(s.Seasons) > 0 {
		result.Seasons = make([]module.ArrSourceSeason, len(s.Seasons))
		for i, season := range s.Seasons {
			result.Seasons[i] = module.ArrSourceSeason{
				SeasonNumber: season.SeasonNumber,
				Monitored:    season.Monitored,
			}
		}
	}
	return result
}

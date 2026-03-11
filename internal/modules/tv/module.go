package tv

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/wire"
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	tvlib "github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/modules/shared"
)

var _ module.QualityDefinition = (*Descriptor)(nil)
var _ module.Migrator = (*Module)(nil)
var _ module.MonitoringPresets = (*Module)(nil)
var _ module.MonitoringCascader = (*Module)(nil)
var _ module.ReleaseDateResolver = (*Module)(nil)
var _ module.CalendarProvider = (*Module)(nil)
var _ module.WantedCollector = (*Module)(nil)
var _ module.NotificationEvents = (*Module)(nil)
var _ module.PortalProvisioner = (*Module)(nil)
var _ module.SlotSupport = (*Module)(nil)
var _ module.TVArrImportAdapter = (*Module)(nil)
var _ module.Module = (*Module)(nil)

// TV module notification event IDs.
const (
	EventTVAdded   = "tv:added"
	EventTVDeleted = "tv:deleted"
)

// Descriptor implements module.Descriptor for the TV module.
type Descriptor struct{}

func (d *Descriptor) ID() module.Type    { return module.TypeTV }
func (d *Descriptor) Name() string       { return "TV" }
func (d *Descriptor) PluralName() string { return "Series" }
func (d *Descriptor) Icon() string       { return "tv" }
func (d *Descriptor) ThemeColor() string { return "green" }

func (d *Descriptor) NodeSchema() module.NodeSchema {
	return module.NodeSchema{
		Levels: []module.NodeLevel{
			{
				Name:         string(module.EntitySeries),
				PluralName:   string(module.EntitySeries),
				IsRoot:       true,
				HasMonitored: true,
				Searchable:   true,
			},
			{
				Name:         string(module.EntitySeason),
				PluralName:   "seasons",
				HasMonitored: true,
				IsSpecial:    true,
			},
			{
				Name:                     string(module.EntityEpisode),
				PluralName:               "episodes",
				IsLeaf:                   true,
				HasMonitored:             true,
				Searchable:               true,
				SupportsMultiEntityFiles: true,
				FormatVariants:           []string{"standard", "daily", "anime"},
			},
		},
	}
}

func (d *Descriptor) EntityTypes() []module.EntityType {
	return []module.EntityType{module.EntitySeries, module.EntitySeason, module.EntityEpisode}
}

func (d *Descriptor) Wire() wire.ProviderSet {
	return wire.NewSet()
}

// QualityItems returns the video quality tiers for this module.
func (d *Descriptor) QualityItems() []module.QualityItem {
	return shared.VideoQualityItems()
}

func (d *Descriptor) ParseQuality(releaseTitle string) (*module.QualityResult, error) {
	return shared.ParseVideoQuality(releaseTitle)
}

func (d *Descriptor) ScoreQuality(item module.QualityItem) int {
	// Stub — delegates to existing scorer in Phase 5
	return item.Weight
}

func (d *Descriptor) IsUpgrade(current, candidate module.QualityItem, _ int64) (bool, error) {
	// Basic quality ordering: higher weight = higher quality.
	// Profile-aware cutoff logic is handled by the quality profile service.
	return candidate.Weight > current.Weight, nil
}

// Module is the top-level TV module satisfying module.Module.
type Module struct {
	descriptor        *Descriptor
	metadataProvider  *metadataProvider
	importHandler     *importHandler
	fileParser        *fileParser
	pathGenerator     *pathGenerator
	namingProvider    *namingProvider
	tvService         *tvlib.Service
	metadataSvc       *metadata.Service
	artworkDownloader *metadata.ArtworkDownloader
	rootFolderSvc     *rootfolder.Service
	qualitySvc        *quality.Service
	db                *sql.DB
	queries           *sqlc.Queries
	slotsService      module.ArrImportSlotsService
	logger            *zerolog.Logger
}

// NewModule creates a fully wired TV module.
func NewModule(db *sql.DB, metadataSvc *metadata.Service, tvSvc *tvlib.Service, rootFolderSvc *rootfolder.Service, artworkDl *metadata.ArtworkDownloader, qualitySvc *quality.Service, logger *zerolog.Logger) *Module {
	return &Module{
		descriptor:        &Descriptor{},
		metadataProvider:  newMetadataProvider(metadataSvc, tvSvc, logger),
		importHandler:     newImportHandler(tvSvc, rootFolderSvc, logger),
		fileParser:        newFileParser(tvSvc, rootFolderSvc, logger),
		pathGenerator:     &pathGenerator{tvSvc: tvSvc},
		namingProvider:    &namingProvider{},
		tvService:         tvSvc,
		metadataSvc:       metadataSvc,
		artworkDownloader: artworkDl,
		rootFolderSvc:     rootFolderSvc,
		qualitySvc:        qualitySvc,
		db:                db,
		queries:           sqlc.New(db),
		logger:            logger,
	}
}

// SetSlotsService sets the optional slots service for multi-version support during arr import.
func (m *Module) SetSlotsService(svc module.ArrImportSlotsService) {
	m.slotsService = svc
}

// --- Descriptor delegation ---

func (m *Module) ID() module.Type                  { return m.descriptor.ID() }
func (m *Module) Name() string                     { return m.descriptor.Name() }
func (m *Module) PluralName() string               { return m.descriptor.PluralName() }
func (m *Module) Icon() string                     { return m.descriptor.Icon() }
func (m *Module) ThemeColor() string               { return m.descriptor.ThemeColor() }
func (m *Module) NodeSchema() module.NodeSchema    { return m.descriptor.NodeSchema() }
func (m *Module) EntityTypes() []module.EntityType { return m.descriptor.EntityTypes() }
func (m *Module) Wire() wire.ProviderSet           { return m.descriptor.Wire() }

// --- QualityDefinition delegation ---

func (m *Module) QualityItems() []module.QualityItem { return m.descriptor.QualityItems() }
func (m *Module) ParseQuality(releaseTitle string) (*module.QualityResult, error) {
	return m.descriptor.ParseQuality(releaseTitle)
}
func (m *Module) ScoreQuality(item module.QualityItem) int { return m.descriptor.ScoreQuality(item) }
func (m *Module) IsUpgrade(current, candidate module.QualityItem, profileID int64) (bool, error) {
	return m.descriptor.IsUpgrade(current, candidate, profileID)
}

// --- MetadataProvider delegation ---

func (m *Module) Search(ctx context.Context, query string, opts module.SearchOptions) ([]module.SearchResult, error) {
	return m.metadataProvider.Search(ctx, query, opts)
}
func (m *Module) GetByID(ctx context.Context, externalID string) (*module.MediaMetadata, error) {
	return m.metadataProvider.GetByID(ctx, externalID)
}
func (m *Module) GetExtendedInfo(ctx context.Context, externalID string) (*module.ExtendedMetadata, error) {
	return m.metadataProvider.GetExtendedInfo(ctx, externalID)
}
func (m *Module) RefreshMetadata(ctx context.Context, entityID int64) (*module.RefreshResult, error) {
	return m.metadataProvider.RefreshMetadata(ctx, entityID)
}

// --- ImportHandler delegation ---

func (m *Module) MatchDownload(ctx context.Context, download *module.CompletedDownload) ([]module.MatchedEntity, error) {
	return m.importHandler.MatchDownload(ctx, download)
}
func (m *Module) ImportFile(ctx context.Context, filePath string, entity *module.MatchedEntity, qi *module.QualityInfo) (*module.ImportResult, error) {
	return m.importHandler.ImportFile(ctx, filePath, entity, qi)
}
func (m *Module) SupportsMultiFileDownload() bool { return m.importHandler.SupportsMultiFileDownload() }
func (m *Module) MatchIndividualFile(ctx context.Context, filePath string, parentEntity *module.MatchedEntity) (*module.MatchedEntity, error) {
	return m.importHandler.MatchIndividualFile(ctx, filePath, parentEntity)
}
func (m *Module) IsGroupImportReady(ctx context.Context, parentEntity *module.MatchedEntity, matchedFiles []module.MatchedEntity) bool {
	return m.importHandler.IsGroupImportReady(ctx, parentEntity, matchedFiles)
}
func (m *Module) MediaInfoFields() []module.MediaInfoFieldDecl {
	return m.importHandler.MediaInfoFields()
}

// --- PathGenerator delegation ---

func (m *Module) DefaultTemplates() map[string]string {
	return m.pathGenerator.DefaultTemplates()
}
func (m *Module) AvailableVariables(level string) []module.TemplateVariable {
	return m.pathGenerator.AvailableVariables(level)
}
func (m *Module) ResolveTemplate(template string, data map[string]any) (string, error) {
	return m.pathGenerator.ResolveTemplate(template, data)
}
func (m *Module) ConditionalSegments() []module.ConditionalSegment {
	return m.pathGenerator.ConditionalSegments()
}
func (m *Module) IsSpecialNode(ctx context.Context, entityType module.EntityType, entityID int64) (bool, error) {
	return m.pathGenerator.IsSpecialNode(ctx, entityType, entityID)
}

// --- NamingProvider delegation ---

func (m *Module) TokenContexts() []module.TokenContext {
	return m.namingProvider.TokenContexts()
}
func (m *Module) DefaultFileTemplates() map[string]string {
	return m.namingProvider.DefaultFileTemplates()
}
func (m *Module) FormatOptions() []module.FormatOption {
	return m.namingProvider.FormatOptions()
}

// --- FileParser delegation ---

func (m *Module) ParseFilename(filename string) (*module.ParseResult, error) {
	return m.fileParser.ParseFilename(filename)
}
func (m *Module) MatchToEntity(ctx context.Context, parseResult *module.ParseResult) (*module.MatchedEntity, error) {
	return m.fileParser.MatchToEntity(ctx, parseResult)
}
func (m *Module) TryMatch(filename string) (float64, *module.ParseResult) {
	return m.fileParser.TryMatch(filename)
}

// --- MockFactory: see mock_factory.go ---

// --- NotificationEvents ---

func (m *Module) DeclareEvents() []module.NotificationEvent {
	return []module.NotificationEvent{
		{ID: EventTVAdded, Label: "Series Added", Description: "When a series is added to the library"},
		{ID: EventTVDeleted, Label: "Series Deleted", Description: "When a series is removed"},
	}
}

// --- RouteProvider stub ---

func (m *Module) RegisterRoutes(_ module.RouteGroup) {}

// --- TaskProvider stub ---

func (m *Module) ScheduledTasks() []module.ScheduledTask { return nil }

// --- PortalProvisioner ---

func (m *Module) SupportedEntityTypes() []string {
	return []string{string(module.EntitySeries), string(module.EntitySeason), string(module.EntityEpisode)}
}

func (m *Module) ValidateRequest(_ context.Context, entityType string, externalIDs map[string]int64) error {
	switch entityType {
	case string(module.EntitySeries), string(module.EntitySeason), string(module.EntityEpisode):
	default:
		return fmt.Errorf("unsupported entity type %q for tv module", entityType)
	}
	if _, ok := externalIDs["tvdb"]; !ok {
		return errors.New("tvdb ID is required for tv requests")
	}
	return nil
}

func (m *Module) EnsureInLibrary(ctx context.Context, input *module.ProvisionInput) (int64, error) {
	tvdbID, ok := input.ExternalIDs["tvdb"]
	if !ok {
		return 0, errors.New("tvdb ID is required")
	}

	existing, err := m.tvService.GetSeriesByTvdbID(ctx, int(tvdbID))
	if err == nil && existing != nil {
		return m.applyMonitoringToExistingSeries(ctx, existing.ID, input)
	}

	rootFolderID, qualityProfileID, err := m.getDefaultSettings(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get default settings: %w", err)
	}

	if input.QualityProfileID != nil {
		qualityProfileID = *input.QualityProfileID
	}

	series, err := m.tvService.CreateSeries(ctx, &tvlib.CreateSeriesInput{
		Title:            input.Title,
		Year:             input.Year,
		TvdbID:           int(tvdbID),
		RootFolderID:     rootFolderID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
		AddedBy:          input.AddedBy,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create series: %w", err)
	}

	m.logger.Info().Int64("tvdbID", tvdbID).Int64("seriesID", series.ID).Str("title", input.Title).Msg("created series in library from request")

	if _, refreshErr := m.RefreshMetadata(ctx, series.ID); refreshErr != nil {
		m.logger.Warn().Err(refreshErr).Int64("seriesID", series.ID).Msg("failed to refresh series metadata, series created without episodes")
	}

	if applyErr := m.applyPortalRequestMonitoring(ctx, series.ID, input); applyErr != nil {
		m.logger.Warn().Err(applyErr).Int64("seriesID", series.ID).Msg("failed to apply portal request monitoring")
	}

	return series.ID, nil
}

func (m *Module) CheckAvailability(ctx context.Context, input *module.AvailabilityCheckInput) (*module.AvailabilityResult, error) {
	switch input.EntityType {
	case string(module.EntitySeries):
		return m.checkSeriesAvailability(ctx, input)
	case string(module.EntitySeason):
		return m.checkSeasonAvailability(ctx, input)
	case string(module.EntityEpisode):
		return m.checkEpisodeAvailability(ctx, input)
	default:
		return nil, fmt.Errorf("unsupported entity type %q", input.EntityType)
	}
}

func (m *Module) CheckRequestCompletion(ctx context.Context, input *module.RequestCompletionCheckInput) (*module.RequestCompletionResult, error) {
	switch input.RequestEntityType {
	case string(module.EntityEpisode):
		return &module.RequestCompletionResult{ShouldMarkAvailable: true}, nil
	case string(module.EntitySeason):
		return m.checkSeasonCompletion(ctx, input)
	case string(module.EntitySeries):
		return m.checkSeriesCompletion(ctx, input)
	default:
		return &module.RequestCompletionResult{}, nil
	}
}

// --- SlotSupport ---

func (m *Module) SlotEntityType() string {
	return string(module.EntityEpisode)
}

// --- provisioner helpers ---

func (m *Module) applyMonitoringToExistingSeries(ctx context.Context, seriesID int64, input *module.ProvisionInput) (int64, error) {
	m.logger.Debug().Int64("seriesID", seriesID).Msg("found existing series in library")
	if len(input.RequestedSeasons) > 0 {
		for _, sn := range input.RequestedSeasons {
			if _, err := m.tvService.UpdateSeasonMonitored(ctx, seriesID, int(sn), true); err != nil {
				m.logger.Warn().Err(err).Int64("seriesID", seriesID).Int64("seasonNumber", sn).Msg("failed to monitor requested season")
			}
		}
	}
	if input.MonitorFuture {
		m.applyMonitorFuture(ctx, seriesID)
	}
	return seriesID, nil
}

func (m *Module) applyPortalRequestMonitoring(ctx context.Context, seriesID int64, input *module.ProvisionInput) error {
	if len(input.RequestedSeasons) > 0 {
		if err := m.applyRequestedSeasonsMonitoring(ctx, seriesID, input.RequestedSeasons); err != nil {
			return err
		}
		if input.MonitorFuture {
			m.applyMonitorFuture(ctx, seriesID)
		}
	} else if input.MonitorFuture {
		if err := m.tvService.BulkMonitor(ctx, seriesID, tvlib.BulkMonitorInput{
			MonitorType:     tvlib.MonitorTypeFuture,
			IncludeSpecials: false,
		}); err != nil {
			return fmt.Errorf("failed to apply monitor future: %w", err)
		}
	}

	m.unmonitorSpecials(ctx, seriesID)
	return nil
}

func (m *Module) applyRequestedSeasonsMonitoring(ctx context.Context, seriesID int64, requestedSeasons []int64) error {
	seasons, err := m.tvService.ListSeasons(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("failed to get seasons: %w", err)
	}

	requestedSet := make(map[int64]bool)
	for _, sn := range requestedSeasons {
		requestedSet[sn] = true
	}

	for i := range seasons {
		season := &seasons[i]
		shouldMonitor := requestedSet[int64(season.SeasonNumber)]
		if season.Monitored != shouldMonitor {
			if _, err := m.tvService.UpdateSeasonMonitored(ctx, seriesID, season.SeasonNumber, shouldMonitor); err != nil {
				m.logger.Warn().Err(err).Int64("seriesID", seriesID).Int("seasonNumber", season.SeasonNumber).Bool("monitored", shouldMonitor).Msg("failed to update season monitoring")
			}
		}
	}
	return nil
}

func (m *Module) applyMonitorFuture(ctx context.Context, seriesID int64) {
	if err := m.queries.UpdateFutureEpisodesMonitored(ctx, sqlc.UpdateFutureEpisodesMonitoredParams{
		Monitored: true,
		SeriesID:  seriesID,
	}); err != nil {
		m.logger.Warn().Err(err).Int64("seriesID", seriesID).Msg("failed to monitor future episodes")
	}
	if err := m.queries.UpdateFutureSeasonsMonitored(ctx, sqlc.UpdateFutureSeasonsMonitoredParams{
		Monitored:  true,
		SeriesID:   seriesID,
		SeriesID_2: seriesID,
	}); err != nil {
		m.logger.Warn().Err(err).Int64("seriesID", seriesID).Msg("failed to monitor future seasons")
	}
}

func (m *Module) unmonitorSpecials(ctx context.Context, seriesID int64) {
	if _, err := m.tvService.UpdateSeasonMonitored(ctx, seriesID, 0, false); err != nil {
		if !errors.Is(err, tvlib.ErrSeasonNotFound) {
			m.logger.Warn().Err(err).Int64("seriesID", seriesID).Msg("failed to unmonitor specials")
		}
	}
}

func (m *Module) checkSeriesAvailability(ctx context.Context, input *module.AvailabilityCheckInput) (*module.AvailabilityResult, error) {
	tvdbID, ok := input.ExternalIDs["tvdb"]
	if !ok {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	series, err := m.queries.GetSeriesByTvdbID(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &module.AvailabilityResult{CanRequest: true}, nil
		}
		return nil, err
	}
	if series.ID == 0 {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	result := &module.AvailabilityResult{
		EntityID: &series.ID,
	}

	seasonAvail, err := m.getSeasonAvailability(ctx, series.ID)
	if err != nil {
		m.logger.Warn().Err(err).Int64("seriesID", series.ID).Msg("failed to get season availability")
	}

	hasAnyFiles := false
	allAvailable := true
	for _, sa := range seasonAvail {
		if sa.hasAnyFiles {
			hasAnyFiles = true
		}
		if !sa.available {
			allAvailable = false
		}
	}

	if hasAnyFiles {
		result.InLibrary = true
		result.CanRequest = !allAvailable
		result.IsComplete = allAvailable
	} else {
		result.CanRequest = true
	}

	result.Detail = seasonAvail
	return result, nil
}

func (m *Module) checkSeasonAvailability(ctx context.Context, input *module.AvailabilityCheckInput) (*module.AvailabilityResult, error) {
	tvdbID, ok := input.ExternalIDs["tvdb"]
	if !ok {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	if input.SeasonNumber == nil {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	series, err := m.queries.GetSeriesByTvdbID(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &module.AvailabilityResult{CanRequest: true}, nil
		}
		return nil, err
	}
	if series.ID == 0 {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	season, err := m.queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     series.ID,
		SeasonNumber: *input.SeasonNumber,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &module.AvailabilityResult{CanRequest: true}, nil
		}
		return nil, err
	}
	if season.ID == 0 {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	return &module.AvailabilityResult{
		InLibrary:  true,
		EntityID:   &season.ID,
		CanRequest: true,
	}, nil
}

func (m *Module) checkEpisodeAvailability(ctx context.Context, input *module.AvailabilityCheckInput) (*module.AvailabilityResult, error) {
	tvdbID, ok := input.ExternalIDs["tvdb"]
	if !ok {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	if input.SeasonNumber == nil || input.EpisodeNumber == nil {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	seriesID, found, err := m.lookupSeriesByTvdbID(ctx, tvdbID)
	if err != nil {
		return nil, err
	}
	if !found {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	episodeID, found, err := m.lookupEpisode(ctx, seriesID, *input.SeasonNumber, *input.EpisodeNumber)
	if err != nil {
		return nil, err
	}
	if !found {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	result := &module.AvailabilityResult{
		InLibrary:  true,
		EntityID:   &episodeID,
		CanRequest: true,
	}

	if m.episodeHasFile(ctx, episodeID) {
		result.CanRequest = false
		result.IsComplete = true
	}

	return result, nil
}

// lookupSeriesByTvdbID returns the series ID for a TVDB ID, or found=false if not in the library.
func (m *Module) lookupSeriesByTvdbID(ctx context.Context, tvdbID int64) (seriesID int64, found bool, err error) {
	series, err := m.queries.GetSeriesByTvdbID(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if series.ID == 0 {
		return 0, false, nil
	}
	return series.ID, true, nil
}

// lookupEpisode returns the episode ID for a series/season/episode number, or found=false if not in the library.
func (m *Module) lookupEpisode(ctx context.Context, seriesID, seasonNumber, episodeNumber int64) (episodeID int64, found bool, err error) {
	episode, err := m.queries.GetEpisodeByNumber(ctx, sqlc.GetEpisodeByNumberParams{
		SeriesID:      seriesID,
		SeasonNumber:  seasonNumber,
		EpisodeNumber: episodeNumber,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if episode.ID == 0 {
		return 0, false, nil
	}
	return episode.ID, true, nil
}

// episodeHasFile checks whether an episode has at least one file via slot assignments.
func (m *Module) episodeHasFile(ctx context.Context, episodeID int64) bool {
	slots, err := m.queries.ListEpisodeSlotAssignments(ctx, episodeID)
	if err != nil {
		return false
	}
	for _, a := range slots {
		if a.FileID.Valid {
			return true
		}
	}
	return false
}

type seasonAvailInfo struct {
	seasonNumber           int
	available              bool
	hasAnyFiles            bool
	airedEpisodesWithFiles int
	totalAiredEpisodes     int
	totalEpisodes          int
	monitored              bool
}

func (m *Module) getSeasonAvailability(ctx context.Context, seriesID int64) ([]seasonAvailInfo, error) {
	rows, err := m.queries.GetSeriesSeasonAvailabilitySummary(ctx, seriesID)
	if err != nil {
		return nil, err
	}

	result := make([]seasonAvailInfo, 0, len(rows))
	for _, row := range rows {
		airedEps := toInt(row.AiredEpisodes)
		airedWithFiles := toInt(row.AiredWithFiles)
		unairedMonitored := toInt(row.UnairedMonitored)
		totalEps := int(row.TotalEpisodes)
		unairedCount := totalEps - airedEps

		available := airedEps > 0 &&
			airedWithFiles == airedEps &&
			(unairedCount == 0 || unairedMonitored == unairedCount)

		result = append(result, seasonAvailInfo{
			seasonNumber:           int(row.SeasonNumber),
			available:              available,
			hasAnyFiles:            airedWithFiles > 0,
			airedEpisodesWithFiles: airedWithFiles,
			totalAiredEpisodes:     airedEps,
			totalEpisodes:          totalEps,
			monitored:              row.Monitored,
		})
	}

	return result, nil
}

func (m *Module) checkSeasonCompletion(ctx context.Context, input *module.RequestCompletionCheckInput) (*module.RequestCompletionResult, error) {
	if input.RequestEntityID == nil {
		return &module.RequestCompletionResult{}, nil
	}

	season, err := m.queries.GetSeason(ctx, *input.RequestEntityID)
	if err != nil {
		m.logger.Warn().Err(err).Int64("seasonID", *input.RequestEntityID).Msg("failed to look up season for completion check")
		return &module.RequestCompletionResult{}, nil
	}

	seasonAvail, err := m.getSeasonAvailability(ctx, season.SeriesID)
	if err != nil {
		m.logger.Warn().Err(err).Int64("seriesID", season.SeriesID).Msg("failed to get season availability for completion check")
		return &module.RequestCompletionResult{}, nil
	}

	for _, sa := range seasonAvail {
		if sa.seasonNumber == int(season.SeasonNumber) {
			return &module.RequestCompletionResult{ShouldMarkAvailable: sa.available}, nil
		}
	}

	return &module.RequestCompletionResult{}, nil
}

func (m *Module) checkSeriesCompletion(ctx context.Context, input *module.RequestCompletionCheckInput) (*module.RequestCompletionResult, error) {
	tvdbID, ok := input.RequestExternalIDs["tvdb"]
	if !ok {
		return &module.RequestCompletionResult{}, nil
	}

	series, err := m.queries.GetSeriesByTvdbID(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if err != nil {
		m.logger.Warn().Err(err).Int64("tvdbID", tvdbID).Msg("failed to look up series for completion check")
		return &module.RequestCompletionResult{}, nil
	}

	seasonAvail, err := m.getSeasonAvailability(ctx, series.ID)
	if err != nil {
		m.logger.Warn().Err(err).Int64("seriesID", series.ID).Msg("failed to get season availability for series completion check")
		return &module.RequestCompletionResult{}, nil
	}

	requestedSet := make(map[int]bool)
	for _, sn := range input.RequestedSeasons {
		requestedSet[int(sn)] = true
	}

	allComplete := true
	for _, sa := range seasonAvail {
		if sa.seasonNumber == 0 {
			continue // skip specials
		}
		if len(requestedSet) > 0 && !requestedSet[sa.seasonNumber] {
			continue // skip non-requested seasons
		}
		if !sa.available {
			allComplete = false
			break
		}
	}

	return &module.RequestCompletionResult{ShouldMarkAvailable: allComplete}, nil
}

func (m *Module) getDefaultSettings(ctx context.Context) (rootFolderID, qualityProfileID int64, err error) {
	rootFolderID = m.resolveRootFolderID(ctx)
	if rootFolderID == 0 {
		return 0, 0, fmt.Errorf("no tv root folder configured - please configure a root folder for tv content")
	}

	profiles, err := m.queries.ListQualityProfiles(ctx)
	if err != nil || len(profiles) == 0 {
		return 0, 0, errors.New("no quality profile configured")
	}
	qualityProfileID = profiles[0].ID

	return rootFolderID, qualityProfileID, nil
}

func (m *Module) resolveRootFolderID(ctx context.Context) int64 {
	if id := m.getMediaTypeSpecificRootFolder(ctx); id != 0 {
		return id
	}
	if id := m.getGenericRootFolder(ctx); id != 0 {
		return id
	}
	return m.getFirstAvailableRootFolder(ctx)
}

func (m *Module) getMediaTypeSpecificRootFolder(ctx context.Context) int64 {
	setting, err := m.queries.GetSetting(ctx, "requests_default_tv_root_folder_id")
	if err != nil || setting.Value == "" {
		return 0
	}
	v, parseErr := strconv.ParseInt(setting.Value, 10, 64)
	if parseErr != nil {
		return 0
	}
	return v
}

func (m *Module) getGenericRootFolder(ctx context.Context) int64 {
	setting, err := m.queries.GetSetting(ctx, "requests_default_root_folder_id")
	if err != nil || setting.Value == "" {
		return 0
	}
	v, parseErr := strconv.ParseInt(setting.Value, 10, 64)
	if parseErr != nil {
		return 0
	}
	rf, rfErr := m.queries.GetRootFolder(ctx, v)
	if rfErr != nil || rf.ModuleType != string(module.TypeTV) {
		return 0
	}
	return v
}

func (m *Module) getFirstAvailableRootFolder(ctx context.Context) int64 {
	rootFolders, err := m.queries.ListRootFoldersByMediaType(ctx, string(module.TypeTV))
	if err != nil || len(rootFolders) == 0 {
		return 0
	}
	return rootFolders[0].ID
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int64:
		return int(val)
	case float64:
		return int(val)
	case int:
		return val
	default:
		return 0
	}
}

package movie

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/wire"
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
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
var _ module.MovieArrImportAdapter = (*Module)(nil)
var _ module.Module = (*Module)(nil)

// Movie module notification event IDs.
const (
	EventMovieAdded   = "movie:added"
	EventMovieDeleted = "movie:deleted"
)

// Descriptor implements module.Descriptor for the Movie module.
type Descriptor struct{}

func (d *Descriptor) ID() module.Type    { return module.TypeMovie }
func (d *Descriptor) Name() string       { return "Movies" }
func (d *Descriptor) PluralName() string { return "Movies" }
func (d *Descriptor) Icon() string       { return "film" }
func (d *Descriptor) ThemeColor() string { return "blue" }

func (d *Descriptor) NodeSchema() module.NodeSchema {
	return module.NodeSchema{
		Levels: []module.NodeLevel{
			{
				Name:         string(module.EntityMovie),
				PluralName:   "movies",
				IsRoot:       true,
				IsLeaf:       true,
				HasMonitored: true,
				Searchable:   true,
			},
		},
	}
}

func (d *Descriptor) EntityTypes() []module.EntityType {
	return []module.EntityType{module.EntityMovie}
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

// Module is the top-level movie module satisfying module.Module.
type Module struct {
	descriptor        *Descriptor
	metadataProvider  *metadataProvider
	importHandler     *importHandler
	fileParser        *fileParser
	pathGenerator     *pathGenerator
	namingProvider    *namingProvider
	movieService      *movies.Service
	metadataSvc       *metadata.Service
	artworkDownloader *metadata.ArtworkDownloader
	rootFolderSvc     *rootfolder.Service
	db                *sql.DB
	queries           *sqlc.Queries
	slotsService      module.ArrImportSlotsService
	logger            *zerolog.Logger
}

// NewModule creates a fully wired movie module.
func NewModule(db *sql.DB, metadataSvc *metadata.Service, movieSvc *movies.Service, rootFolderSvc *rootfolder.Service, artworkDl *metadata.ArtworkDownloader, logger *zerolog.Logger) *Module {
	return &Module{
		descriptor:        &Descriptor{},
		metadataProvider:  newMetadataProvider(metadataSvc, movieSvc, logger),
		importHandler:     newImportHandler(movieSvc, rootFolderSvc, logger),
		fileParser:        newFileParser(movieSvc, rootFolderSvc, logger),
		pathGenerator:     &pathGenerator{},
		namingProvider:    &namingProvider{},
		movieService:      movieSvc,
		metadataSvc:       metadataSvc,
		artworkDownloader: artworkDl,
		rootFolderSvc:     rootFolderSvc,
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
		{ID: EventMovieAdded, Label: "Movie Added", Description: "When a movie is added to the library"},
		{ID: EventMovieDeleted, Label: "Movie Deleted", Description: "When a movie is removed"},
	}
}

// --- RouteProvider stub ---

func (m *Module) RegisterRoutes(_ module.RouteGroup) {}

// --- TaskProvider stub ---

func (m *Module) ScheduledTasks() []module.ScheduledTask { return nil }

// --- PortalProvisioner ---

func (m *Module) SupportedEntityTypes() []string {
	return []string{string(module.EntityMovie)}
}

func (m *Module) ValidateRequest(_ context.Context, entityType string, externalIDs map[string]int64) error {
	if entityType != string(module.EntityMovie) {
		return fmt.Errorf("unsupported entity type %q for movie module", entityType)
	}
	if _, ok := externalIDs["tmdb"]; !ok {
		return errors.New("tmdb ID is required for movie requests")
	}
	return nil
}

func (m *Module) EnsureInLibrary(ctx context.Context, input *module.ProvisionInput) (int64, error) {
	tmdbID, ok := input.ExternalIDs["tmdb"]
	if !ok {
		return 0, errors.New("tmdb ID is required")
	}

	existing, err := m.movieService.GetByTmdbID(ctx, int(tmdbID))
	if err == nil && existing != nil {
		m.logger.Debug().Int64("tmdbID", tmdbID).Int64("movieID", existing.ID).Msg("found existing movie in library")
		return existing.ID, nil
	}

	rootFolderID, qualityProfileID, err := m.getDefaultSettings(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get default settings: %w", err)
	}

	if input.QualityProfileID != nil {
		qualityProfileID = *input.QualityProfileID
	}

	movie, err := m.movieService.Create(ctx, &movies.CreateMovieInput{
		Title:            input.Title,
		Year:             input.Year,
		TmdbID:           int(tmdbID),
		RootFolderID:     rootFolderID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
		AddedBy:          input.AddedBy,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create movie: %w", err)
	}

	m.logger.Info().Int64("tmdbID", tmdbID).Int64("movieID", movie.ID).Str("title", input.Title).Msg("created movie in library from request")

	if _, refreshErr := m.RefreshMetadata(ctx, movie.ID); refreshErr != nil {
		m.logger.Warn().Err(refreshErr).Int64("movieID", movie.ID).Msg("failed to refresh movie metadata, movie created without full details")
	}

	return movie.ID, nil
}

func (m *Module) CheckAvailability(ctx context.Context, input *module.AvailabilityCheckInput) (*module.AvailabilityResult, error) {
	tmdbID, ok := input.ExternalIDs["tmdb"]
	if !ok {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	movieRow, err := m.queries.GetMovieByTmdbID(ctx, sql.NullInt64{Int64: tmdbID, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &module.AvailabilityResult{CanRequest: true}, nil
		}
		return nil, err
	}

	if movieRow.ID == 0 {
		return &module.AvailabilityResult{CanRequest: true}, nil
	}

	result := &module.AvailabilityResult{
		InLibrary:  true,
		EntityID:   &movieRow.ID,
		CanRequest: false,
	}

	fileCount, err := m.queries.CountMovieFiles(ctx, movieRow.ID)
	if err != nil {
		return nil, err
	}
	if fileCount == 0 {
		result.InLibrary = false
		result.CanRequest = true
	}

	return result, nil
}

func (m *Module) CheckRequestCompletion(_ context.Context, input *module.RequestCompletionCheckInput) (*module.RequestCompletionResult, error) {
	return &module.RequestCompletionResult{
		ShouldMarkAvailable: input.AvailableEntityType == string(module.EntityMovie),
	}, nil
}

// --- SlotSupport ---

func (m *Module) SlotEntityType() string {
	return string(module.EntityMovie)
}

// --- provisioner helpers ---

func (m *Module) getDefaultSettings(ctx context.Context) (rootFolderID, qualityProfileID int64, err error) {
	rootFolderID = m.resolveRootFolderID(ctx)
	if rootFolderID == 0 {
		return 0, 0, fmt.Errorf("no movie root folder configured - please configure a root folder for movie content")
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
	setting, err := m.queries.GetSetting(ctx, "requests_default_movie_root_folder_id")
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
	if rfErr != nil || rf.ModuleType != string(module.TypeMovie) {
		return 0
	}
	return v
}

func (m *Module) getFirstAvailableRootFolder(ctx context.Context) int64 {
	rootFolders, err := m.queries.ListRootFoldersByMediaType(ctx, string(module.TypeMovie))
	if err != nil || len(rootFolders) == 0 {
		return 0
	}
	return rootFolders[0].ID
}

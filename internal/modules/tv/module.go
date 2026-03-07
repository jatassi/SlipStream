package tv

import (
	"context"
	"database/sql"

	"github.com/google/wire"
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	tvlib "github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/modules/shared"
)

var _ module.QualityDefinition = (*Descriptor)(nil)
var _ module.MonitoringPresets = (*Module)(nil)
var _ module.ReleaseDateResolver = (*Module)(nil)
var _ module.CalendarProvider = (*Module)(nil)
var _ module.WantedCollector = (*Module)(nil)
var _ module.NotificationEvents = (*Module)(nil)

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
				Name:         "series",
				PluralName:   "series",
				IsRoot:       true,
				HasMonitored: true,
				Searchable:   true,
			},
			{
				Name:         "season",
				PluralName:   "seasons",
				HasMonitored: true,
				IsSpecial:    true,
			},
			{
				Name:                     "episode",
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

func (d *Descriptor) ParseQuality(_ string) (*module.QualityResult, error) {
	// Stub — delegates to existing parser in Phase 5
	return &module.QualityResult{}, nil
}

func (d *Descriptor) ScoreQuality(item module.QualityItem) int {
	// Stub — delegates to existing scorer in Phase 5
	return item.Weight
}

func (d *Descriptor) IsUpgrade(current, candidate module.QualityItem, profileID int64) (bool, error) {
	// Stub — upgrade logic lives on Profile, not the module.
	return false, nil
}

// Module is the top-level TV module satisfying module.Module.
type Module struct {
	descriptor       *Descriptor
	metadataProvider *metadataProvider
	importHandler    *importHandler
	fileParser       *fileParser
	pathGenerator    *pathGenerator
	namingProvider   *namingProvider
	tvService        *tvlib.Service
	queries          *sqlc.Queries
	logger           *zerolog.Logger
}

// NewModule creates a fully wired TV module.
func NewModule(db *sql.DB, metadataSvc *metadata.Service, tvSvc *tvlib.Service, rootFolderSvc *rootfolder.Service, logger *zerolog.Logger) *Module {
	return &Module{
		descriptor:       &Descriptor{},
		metadataProvider: newMetadataProvider(metadataSvc, tvSvc, logger),
		importHandler:    newImportHandler(tvSvc, rootFolderSvc, logger),
		fileParser:       newFileParser(tvSvc, rootFolderSvc, logger),
		pathGenerator:    &pathGenerator{tvSvc: tvSvc},
		namingProvider:   &namingProvider{},
		tvService:        tvSvc,
		queries:          sqlc.New(db),
		logger:           logger,
	}
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

// --- MockFactory stubs ---

func (m *Module) CreateMockMetadataProvider() module.MetadataProvider { return nil }
func (m *Module) CreateSampleLibraryData(_ context.Context) error     { return nil }
func (m *Module) CreateTestRootFolders(_ context.Context) error       { return nil }

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

package movie

import (
	"context"
	"time"

	"github.com/google/wire"
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/modules/shared"
)

var _ module.QualityDefinition = (*Descriptor)(nil)

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
				Name:         "movie",
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

// Module is the top-level movie module satisfying module.Module.
type Module struct {
	descriptor       *Descriptor
	metadataProvider *metadataProvider
}

// NewModule creates a fully wired movie module.
func NewModule(metadataSvc *metadata.Service, movieSvc *movies.Service, logger *zerolog.Logger) *Module {
	return &Module{
		descriptor:       &Descriptor{},
		metadataProvider: newMetadataProvider(metadataSvc, movieSvc, logger),
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

// --- SearchStrategy stubs ---

func (m *Module) Categories() []int { return nil }
func (m *Module) FilterRelease(_ module.Release, _ module.SearchableItem) (reject bool, reason string) {
	return false, ""
}
func (m *Module) TitlesMatch(_, _ string) bool { return false }
func (m *Module) BuildSearchCriteria(_ module.SearchableItem) module.SearchCriteria {
	return module.SearchCriteria{}
}
func (m *Module) IsGroupSearchEligible(_ module.EntityType, _ int64) bool { return false }
func (m *Module) SuppressChildSearches(_ module.EntityType, _ int64, _ module.Release) []int64 {
	return nil
}

// --- ImportHandler stubs ---

func (m *Module) MatchDownload(_ context.Context, _ module.CompletedDownload) ([]module.MatchedEntity, error) {
	return nil, nil
}
func (m *Module) ImportFile(_ context.Context, _ string, _ module.MatchedEntity, _ module.QualityInfo) (*module.ImportResult, error) {
	return &module.ImportResult{}, nil
}
func (m *Module) SupportsMultiFileDownload() bool { return false }
func (m *Module) MatchIndividualFile(_ context.Context, _ string, _ module.MatchedEntity) (*module.MatchedEntity, error) {
	return &module.MatchedEntity{}, nil
}
func (m *Module) IsGroupImportReady(_ context.Context, _ module.MatchedEntity, _ []module.MatchedEntity) bool {
	return false
}
func (m *Module) MediaInfoFields() []module.MediaInfoFieldDecl { return nil }

// --- PathGenerator stubs ---

func (m *Module) DefaultTemplates() map[string]string                        { return nil }
func (m *Module) AvailableVariables(_ string) []module.TemplateVariable      { return nil }
func (m *Module) ResolveTemplate(_ string, _ map[string]any) (string, error) { return "", nil }
func (m *Module) ConditionalSegments() []module.ConditionalSegment           { return nil }
func (m *Module) IsSpecialNode(_ module.EntityType, _ int64) bool            { return false }

// --- NamingProvider stubs ---

func (m *Module) TokenContexts() []module.TokenContext    { return nil }
func (m *Module) DefaultFileTemplates() map[string]string { return nil }
func (m *Module) FormatOptions() []module.FormatOption    { return nil }

// --- CalendarProvider stub ---

func (m *Module) GetItemsInDateRange(_ context.Context, _, _ time.Time) ([]module.CalendarItem, error) {
	return nil, nil
}

// --- WantedCollector stubs ---

func (m *Module) CollectMissing(_ context.Context) ([]module.SearchableItem, error) { return nil, nil }
func (m *Module) CollectUpgradable(_ context.Context) ([]module.SearchableItem, error) {
	return nil, nil
}

// --- MonitoringPresets stubs ---

func (m *Module) AvailablePresets() []module.MonitoringPreset            { return nil }
func (m *Module) ApplyPreset(_ context.Context, _ int64, _ string) error { return nil }

// --- FileParser stubs ---

func (m *Module) ParseFilename(_ string) (*module.ParseResult, error) {
	return &module.ParseResult{}, nil
}
func (m *Module) MatchToEntity(_ context.Context, _ *module.ParseResult) (*module.MatchedEntity, error) {
	return &module.MatchedEntity{}, nil
}
func (m *Module) TryMatch(_ string) (float64, *module.ParseResult) { return 0, nil }

// --- MockFactory stubs ---

func (m *Module) CreateMockMetadataProvider() module.MetadataProvider { return nil }
func (m *Module) CreateSampleLibraryData(_ context.Context) error     { return nil }
func (m *Module) CreateTestRootFolders(_ context.Context) error       { return nil }

// --- NotificationEvents stub ---

func (m *Module) DeclareEvents() []module.NotificationEvent { return nil }

// --- ReleaseDateResolver stubs ---

func (m *Module) ComputeAvailabilityDate(_ context.Context, _ int64) (*time.Time, error) {
	return &time.Time{}, nil
}
func (m *Module) CheckReleaseDateTransitions(_ context.Context) (int, error) { return 0, nil }

// --- RouteProvider stub ---

func (m *Module) RegisterRoutes(_ module.RouteGroup) {}

// --- TaskProvider stub ---

func (m *Module) ScheduledTasks() []module.ScheduledTask { return nil }

package module

import (
	"context"
	"time"
)

// MetadataProvider handles search and metadata retrieval for a module.
type MetadataProvider interface {
	Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error)
	GetByID(ctx context.Context, externalID string) (*MediaMetadata, error)
	GetExtendedInfo(ctx context.Context, externalID string) (*ExtendedMetadata, error)
	RefreshMetadata(ctx context.Context, entityID int64) (*RefreshResult, error)
}

// SearchStrategy defines search behavior for a module.
type SearchStrategy interface {
	Categories() []int
	FilterRelease(release Release, item SearchableItem) (reject bool, reason string)
	TitlesMatch(releaseTitle, mediaTitle string) bool
	BuildSearchCriteria(item SearchableItem) SearchCriteria
	IsGroupSearchEligible(parentEntityType EntityType, parentID int64) bool
	SuppressChildSearches(parentEntityType EntityType, parentID int64, grabbedRelease Release) []int64
}

// ImportHandler handles download completion and file import.
type ImportHandler interface {
	MatchDownload(ctx context.Context, download CompletedDownload) ([]MatchedEntity, error)
	ImportFile(ctx context.Context, filePath string, entity MatchedEntity, qualityInfo QualityInfo) (*ImportResult, error)
	SupportsMultiFileDownload() bool
	MatchIndividualFile(ctx context.Context, filePath string, parentEntity MatchedEntity) (*MatchedEntity, error)
	IsGroupImportReady(ctx context.Context, parentEntity MatchedEntity, matchedFiles []MatchedEntity) bool
	MediaInfoFields() []MediaInfoFieldDecl
}

// PathGenerator declares folder path templates.
type PathGenerator interface {
	DefaultTemplates() map[string]string
	AvailableVariables(level string) []TemplateVariable
	ResolveTemplate(template string, data map[string]any) (string, error)
	ConditionalSegments() []ConditionalSegment
	IsSpecialNode(entityType EntityType, entityID int64) bool
}

// NamingProvider declares file naming configuration.
type NamingProvider interface {
	TokenContexts() []TokenContext
	DefaultFileTemplates() map[string]string
	FormatOptions() []FormatOption
}

// CalendarProvider returns items for the unified calendar.
type CalendarProvider interface {
	GetItemsInDateRange(ctx context.Context, start, end time.Time) ([]CalendarItem, error)
}

// QualityDefinition declares quality tiers and scoring.
type QualityDefinition interface {
	QualityItems() []QualityItem
	ParseQuality(releaseTitle string) (*QualityResult, error)
	ScoreQuality(item QualityItem) int
	IsUpgrade(current, candidate QualityItem, profileID int64) (bool, error)
}

// WantedCollector collects missing/upgradable items.
type WantedCollector interface {
	CollectMissing(ctx context.Context) ([]SearchableItem, error)
	CollectUpgradable(ctx context.Context) ([]SearchableItem, error)
}

// MonitoringPresets defines monitoring strategies.
type MonitoringPresets interface {
	AvailablePresets() []MonitoringPreset
	ApplyPreset(ctx context.Context, rootEntityID int64, presetID string) error
}

// FileParser parses filenames to extract module-specific identifiers.
type FileParser interface {
	ParseFilename(filename string) (*ParseResult, error)
	MatchToEntity(ctx context.Context, parseResult *ParseResult) (*MatchedEntity, error)
	TryMatch(filename string) (confidence float64, match *ParseResult)
}

// MockFactory creates mock data for dev mode.
type MockFactory interface {
	CreateMockMetadataProvider() MetadataProvider
	CreateSampleLibraryData(ctx context.Context) error
	CreateTestRootFolders(ctx context.Context) error
}

// NotificationEvents declares notification event catalog.
type NotificationEvents interface {
	DeclareEvents() []NotificationEvent
}

// ReleaseDateResolver computes availability dates.
type ReleaseDateResolver interface {
	ComputeAvailabilityDate(ctx context.Context, entityID int64) (*time.Time, error)
	CheckReleaseDateTransitions(ctx context.Context) (transitioned int, err error)
}

// RouteProvider registers module-specific API routes.
type RouteProvider interface {
	RegisterRoutes(group RouteGroup)
}

// TaskProvider declares scheduled tasks.
type TaskProvider interface {
	ScheduledTasks() []ScheduledTask
}

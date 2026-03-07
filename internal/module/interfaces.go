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

// SearchStrategy defines search behavior for a module. (spec §5.4, §5.5, §5.6)
type SearchStrategy interface {
	// Categories returns the Newznab/Torznab category ranges for this module.
	Categories() []int

	// DefaultSearchCategories returns the default subset of categories used for searches.
	DefaultSearchCategories() []int

	// FilterRelease determines whether a release should be rejected for the given item.
	// Returns (true, reason) to reject, (false, "") to accept.
	FilterRelease(ctx context.Context, release *ReleaseForFilter, item SearchableItem) (reject bool, reason string)

	// TitlesMatch checks whether a parsed release title matches a media title.
	TitlesMatch(releaseTitle, mediaTitle string) bool

	// BuildSearchCriteria constructs indexer search parameters for a wanted item.
	BuildSearchCriteria(item SearchableItem) SearchCriteria

	// IsGroupSearchEligible determines if a parent node should be searched as a group.
	// childIdentifier carries the season number for TV, 0 for flat modules like Movie.
	// forUpgrade distinguishes missing vs upgrade eligibility checks.
	IsGroupSearchEligible(ctx context.Context, parentEntityType EntityType, parentID int64, childIdentifier int, forUpgrade bool) bool

	// SuppressChildSearches returns entity IDs of children that should be suppressed
	// after a successful group search (e.g., all episodes in a grabbed season pack).
	SuppressChildSearches(parentEntityType EntityType, parentID int64, seasonNumber int) []int64
}

// ImportHandler handles download completion and file import.
type ImportHandler interface {
	MatchDownload(ctx context.Context, download *CompletedDownload) ([]MatchedEntity, error)
	ImportFile(ctx context.Context, filePath string, entity *MatchedEntity, qualityInfo *QualityInfo) (*ImportResult, error)
	SupportsMultiFileDownload() bool
	MatchIndividualFile(ctx context.Context, filePath string, parentEntity *MatchedEntity) (*MatchedEntity, error)
	IsGroupImportReady(ctx context.Context, parentEntity *MatchedEntity, matchedFiles []MatchedEntity) bool
	MediaInfoFields() []MediaInfoFieldDecl
}

// PathGenerator declares folder path templates.
type PathGenerator interface {
	DefaultTemplates() map[string]string
	AvailableVariables(level string) []TemplateVariable
	ResolveTemplate(template string, data map[string]any) (string, error)
	ConditionalSegments() []ConditionalSegment
	IsSpecialNode(ctx context.Context, entityType EntityType, entityID int64) (bool, error)
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
	ApplyPreset(ctx context.Context, rootEntityID int64, presetID string, options map[string]any) error
}

// FileParser parses filenames to extract module-specific identifiers.
type FileParser interface {
	ParseFilename(filename string) (*ParseResult, error)
	MatchToEntity(ctx context.Context, parseResult *ParseResult) (*MatchedEntity, error)
	TryMatch(filename string) (confidence float64, match *ParseResult)
}

// MockFactory creates mock data for dev mode. (§14.1)
type MockFactory interface {
	// CreateMockMetadataProvider returns a mock metadata provider for this module.
	// May return nil if the module reuses a shared mock provider (current movie/TV behavior).
	CreateMockMetadataProvider() MetadataProvider
	// CreateSampleLibraryData creates sample media entities in the dev database.
	CreateSampleLibraryData(ctx context.Context, mctx *MockContext) error
	// CreateTestRootFolders creates mock root folders with virtual filesystem entries.
	CreateTestRootFolders(ctx context.Context, mctx *MockContext) error
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

// Optional interfaces — modules may implement these for additional capabilities:
//   - PortalProvisioner (portal_provisioner.go): portal request support
//   - SlotSupport (slot_support.go): multi-version slot support
//   - ArrImportAdapter (optional_interfaces.go): generic import from external *arr apps
//   - MovieArrImportAdapter (arr_import.go): type-safe movie import from Radarr
//   - TVArrImportAdapter (arr_import.go): type-safe TV series import from Sonarr

package module

import "context"

// ProvisionInput contains the data needed to ensure a media item exists in the library.
type ProvisionInput struct {
	ExternalIDs      map[string]int64 // e.g., {"tmdb": 12345, "tvdb": 67890}
	Title            string
	Year             int
	QualityProfileID *int64 // User's assigned quality profile (nil = use default)
	AddedBy          *int64 // Portal user ID who triggered the add
	// Module-specific fields for TV-like modules
	RequestedSeasons []int64 // Specific seasons to monitor (empty = all)
	MonitorFuture    bool    // Monitor future/unaired content
}

// AvailabilityCheckInput contains the data needed to check if an entity is available.
type AvailabilityCheckInput struct {
	ModuleType           string
	EntityType           string // e.g., "movie", "series", "season", "episode"
	ExternalIDs          map[string]int64
	SeasonNumber         *int64
	EpisodeNumber        *int64
	UserQualityProfileID *int64
}

// AvailabilityResult describes the availability state of an entity in the library.
type AvailabilityResult struct {
	InLibrary  bool
	EntityID   *int64
	CanRequest bool
	IsComplete bool // All children available (for parent nodes)
	// Module-specific detail (opaque to framework, passed to frontend)
	Detail any
}

// RequestCompletionCheckInput contains the data for checking if a request should
// transition to "available" based on an entity becoming available.
type RequestCompletionCheckInput struct {
	// The entity that just became available
	AvailableEntityType string
	AvailableEntityID   int64
	// The request to check
	RequestEntityType  string
	RequestEntityID    *int64 // media_id on the request (may be nil)
	RequestExternalIDs map[string]int64
	RequestedSeasons   []int64
}

// RequestCompletionResult indicates whether a request should be marked available.
type RequestCompletionResult struct {
	ShouldMarkAvailable bool
}

// PortalProvisioner is an optional module interface for portal request support.
// Modules that implement this interface enable their media types to be requested
// through the external requests portal.
// (spec S11.1)
type PortalProvisioner interface {
	// EnsureInLibrary finds or creates the requested media in the library.
	// Returns the root entity ID (e.g., movie ID, series ID).
	EnsureInLibrary(ctx context.Context, input *ProvisionInput) (entityID int64, err error)

	// CheckAvailability checks whether an entity exists in the library and
	// whether it can still be requested.
	CheckAvailability(ctx context.Context, input *AvailabilityCheckInput) (*AvailabilityResult, error)

	// CheckRequestCompletion determines whether a request should transition to
	// "available" because an entity just became available. Called by the
	// StatusTracker when an entity's status changes to available.
	CheckRequestCompletion(ctx context.Context, input *RequestCompletionCheckInput) (*RequestCompletionResult, error)

	// ValidateRequest validates a request before creation (e.g., checks that
	// the external IDs are valid, the media type is supported).
	ValidateRequest(ctx context.Context, entityType string, externalIDs map[string]int64) error

	// SupportedEntityTypes returns the entity types this module handles.
	// E.g., movie module returns ["movie"], TV module returns ["series", "season", "episode"].
	// Used by the framework to route requests to the correct module.
	SupportedEntityTypes() []string
}

//go:build wireinject

package api

import (
	"github.com/google/wire"
	"github.com/rs/zerolog"

	authratelimit "github.com/slipstream/slipstream/internal/api/ratelimit"
	"github.com/slipstream/slipstream/internal/auth"
	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/availability"
	"github.com/slipstream/slipstream/internal/calendar"
	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database"
	"github.com/slipstream/slipstream/internal/decisioning"
	"github.com/slipstream/slipstream/internal/defaults"
	"github.com/slipstream/slipstream/internal/domain/contracts"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/filesystem"
	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/history"
	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/arrimport"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	"github.com/slipstream/slipstream/internal/indexer/ratelimit"
	"github.com/slipstream/slipstream/internal/indexer/search"
	"github.com/slipstream/slipstream/internal/indexer/status"
	"github.com/slipstream/slipstream/internal/library/librarymanager"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/organizer"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/mediainfo"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/missing"
	"github.com/slipstream/slipstream/internal/notification"
	"github.com/slipstream/slipstream/internal/notification/plex"
	"github.com/slipstream/slipstream/internal/portal/admin"
	"github.com/slipstream/slipstream/internal/portal/autoapprove"
	"github.com/slipstream/slipstream/internal/portal/invitations"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	portalnotifs "github.com/slipstream/slipstream/internal/portal/notifications"
	"github.com/slipstream/slipstream/internal/portal/provisioner"
	"github.com/slipstream/slipstream/internal/portal/quota"
	"github.com/slipstream/slipstream/internal/portal/requests"
	"github.com/slipstream/slipstream/internal/portal/users"
	"github.com/slipstream/slipstream/internal/preferences"
	"github.com/slipstream/slipstream/internal/progress"
	"github.com/slipstream/slipstream/internal/prowlarr"
	"github.com/slipstream/slipstream/internal/rsssync"
	"github.com/slipstream/slipstream/internal/websocket"
)

// BuildServices constructs all service dependencies via Wire.
// The generated code in wire_gen.go calls constructors in dependency order.
func BuildServices(
	dbManager *database.Manager,
	hub *websocket.Hub,
	cfg *config.Config,
	logger *zerolog.Logger,
) (*ServiceContainer, error) {
	wire.Build(
		// --- Infrastructure providers ---
		provideDB,
		provideQueries,

		// --- Config extraction providers ---
		provideMetadataConfig,
		provideArtworkConfig,
		provideAutoSearchConfig,
		provideRssSyncConfig,
		provideRateLimitConfig,
		provideNamingConfig,
		provideMediainfoConfig,
		provideImporterConfig,

		// --- Error-swallowing providers ---
		provideCardigannManager,
		provideScheduler,
		providePasskeyService,

		// --- Providers with complex/ambiguous params ---
		provideAuthService,
		providePlexClient,
		provideModeManager,
		provideFeedFetcher,
		provideSearchLimiter,

		// --- Adapter providers ---
		provideStatusChangeLogger,
		provideImportHistoryService,
		provideMovieLookup,
		provideEpisodeLookup,
		providePortalEnabledChecker,
		provideAdminLibraryChecker,

		// --- System service constructors ---
		health.NewService,
		defaults.NewService,
		calendar.NewService,
		availability.NewService,
		missing.NewService,
		preferences.NewService,
		history.NewService,
		progress.NewManager,

		// --- Library service constructors ---
		scanner.NewService,
		movies.NewService,
		tv.NewService,
		quality.NewService,
		slots.NewService,
		rootfolder.NewService,
		librarymanager.NewService,
		organizer.NewService,
		mediainfo.NewService,

		// --- Metadata service constructors ---
		metadata.NewService,
		metadata.NewArtworkDownloader,
		metadata.NewSQLNetworkLogoStore,

		// --- Filesystem service constructors ---
		filesystem.NewService,
		filesystem.NewStorageService,

		// --- Download service constructors ---
		downloader.NewService,
		downloader.NewQueueBroadcaster,

		// --- Search service constructors ---
		indexer.NewService,
		status.NewService,
		ratelimit.NewLimiter,
		search.NewService,
		search.NewRouter,
		grab.NewService,
		prowlarr.NewService,
		prowlarr.NewSearchAdapter,
		prowlarr.NewGrabProvider,
		decisioning.NewGrabLock,

		// --- Automation service constructors ---
		importer.NewService,
		arrimport.NewService,
		autosearch.NewService,
		autosearch.NewScheduledSearcher,
		autosearch.NewSettingsHandler,
		rsssync.NewService,
		rsssync.NewSettingsHandler,
		importer.NewSettingsHandlers,

		// --- Notification service constructors ---
		notification.NewService,
		plex.NewHandlers,

		// --- Portal service constructors ---
		users.NewService,
		invitations.NewService,
		quota.NewService,
		requests.NewService,
		requests.NewWatchersService,
		requests.NewStatusTracker,
		requests.NewLibraryChecker,
		requests.NewRequestSearcher,
		requests.NewEventBroadcaster,
		portalnotifs.NewService,
		autoapprove.NewService,
		provisioner.NewService,
		admin.NewSettingsHandlers,
		portalmw.NewAuthMiddleware,

		// --- Security service constructors ---
		authratelimit.NewAuthLimiter,

		// --- Interface bindings (concrete → interface) ---
		wire.Bind(new(contracts.Broadcaster), new(*websocket.Hub)),
		wire.Bind(new(contracts.HealthService), new(*health.Service)),
		wire.Bind(new(contracts.QueueTrigger), new(*downloader.QueueBroadcaster)),

		// Search interfaces
		wire.Bind(new(search.SearchService), new(*search.Router)),
		wire.Bind(new(search.ProwlarrSearcher), new(*prowlarr.SearchAdapter)),
		wire.Bind(new(search.ModeProvider), new(*prowlarr.ModeManager)),
		wire.Bind(new(grab.IndexerClientProvider), new(*prowlarr.GrabProvider)),
		wire.Bind(new(prowlarr.InternalIndexerProvider), new(*indexer.Service)),

		// Portal interfaces
		wire.Bind(new(requests.SeriesLookup), new(*tv.Service)),
		wire.Bind(new(requests.NotificationDispatcher), new(*portalnotifs.Service)),
		wire.Bind(new(requests.MediaProvisioner), new(*provisioner.Service)),
		wire.Bind(new(portalmw.TokenValidator), new(*auth.Service)),
		wire.Bind(new(portalmw.UserExistenceChecker), new(*users.Service)),

		// Metadata interfaces
		wire.Bind(new(metadata.NetworkLogoStore), new(*metadata.SQLNetworkLogoStore)),

		// Download/import interfaces
		wire.Bind(new(downloader.PortalStatusTracker), new(*requests.StatusTracker)),
		wire.Bind(new(grab.PortalStatusTracker), new(*requests.StatusTracker)),
		wire.Bind(new(importer.StatusTrackerService), new(*requests.StatusTracker)),

		// Slot interfaces
		wire.Bind(new(slots.RootFolderProvider), new(*rootfolder.Service)),

		// ArrImport interfaces
		wire.Bind(new(arrimport.MovieService), new(*movies.Service)),
		wire.Bind(new(arrimport.TVService), new(*tv.Service)),
		wire.Bind(new(arrimport.RootFolderService), new(*rootfolder.Service)),
		wire.Bind(new(arrimport.QualityService), new(*quality.Service)),
		wire.Bind(new(arrimport.SlotsService), new(*slots.Service)),

		// --- Group struct assembly ---
		wire.Struct(new(ServiceContainer), "*"),
		wire.Struct(new(SystemGroup), "Health", "Defaults", "Calendar", "Availability", "Missing", "Preferences", "History", "Progress"),
		wire.Struct(new(LibraryGroup), "*"),
		wire.Struct(new(MetadataGroup), "Service", "ArtworkDownloader", "NetworkLogoStore"),
		wire.Struct(new(FilesystemGroup), "*"),
		wire.Struct(new(DownloadGroup), "*"),
		wire.Struct(new(SearchGroup), "*"),
		wire.Struct(new(AutomationGroup), "*"),
		wire.Struct(new(NotificationGroup), "*"),
		wire.Struct(new(PortalGroup), "*"),
		wire.Struct(new(SecurityGroup), "*"),
		wire.Struct(new(SwitchableServices), "*"),
	)
	return nil, nil
}

package api

import (
	authratelimit "github.com/slipstream/slipstream/internal/api/ratelimit"
	"github.com/slipstream/slipstream/internal/arrimport"
	"github.com/slipstream/slipstream/internal/auth"
	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/availability"
	"github.com/slipstream/slipstream/internal/calendar"
	"github.com/slipstream/slipstream/internal/decisioning"
	"github.com/slipstream/slipstream/internal/defaults"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/filesystem"
	"github.com/slipstream/slipstream/internal/firewall"
	"github.com/slipstream/slipstream/internal/health"
	"github.com/slipstream/slipstream/internal/history"
	importer "github.com/slipstream/slipstream/internal/import"
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
	portalratelimit "github.com/slipstream/slipstream/internal/portal/ratelimit"
	"github.com/slipstream/slipstream/internal/portal/requests"
	"github.com/slipstream/slipstream/internal/portal/users"
	"github.com/slipstream/slipstream/internal/preferences"
	"github.com/slipstream/slipstream/internal/progress"
	"github.com/slipstream/slipstream/internal/prowlarr"
	"github.com/slipstream/slipstream/internal/rsssync"
	"github.com/slipstream/slipstream/internal/scheduler"
	"github.com/slipstream/slipstream/internal/update"
)

// LibraryGroup holds library management services.
type LibraryGroup struct {
	Scanner        *scanner.Service
	Movies         *movies.Service
	TV             *tv.Service
	Quality        *quality.Service
	Slots          *slots.Service
	RootFolder     *rootfolder.Service
	LibraryManager *librarymanager.Service
	Organizer      *organizer.Service
	Mediainfo      *mediainfo.Service
}

// MetadataGroup holds metadata and artwork services.
type MetadataGroup struct {
	Service           *metadata.Service
	ArtworkDownloader *metadata.ArtworkDownloader
	NetworkLogoStore  *metadata.SQLNetworkLogoStore
	RealTMDBClient    metadata.TMDBClient
	RealTVDBClient    metadata.TVDBClient
	RealOMDBClient    metadata.OMDBClient
}

// FilesystemGroup holds filesystem services.
type FilesystemGroup struct {
	Service *filesystem.Service
	Storage *filesystem.StorageService
}

// DownloadGroup holds download management services.
type DownloadGroup struct {
	Service          *downloader.Service
	QueueBroadcaster *downloader.QueueBroadcaster
}

// SearchGroup holds search and indexer services.
type SearchGroup struct {
	Indexer        *indexer.Service
	Search         *search.Service
	Status         *status.Service
	RateLimiter    *ratelimit.Limiter
	Grab           *grab.Service
	GrabLock       *decisioning.GrabLock
	Router         *search.Router
	Prowlarr       *prowlarr.Service
	ProwlarrMode   *prowlarr.ModeManager
	ProwlarrSearch *prowlarr.SearchAdapter
	ProwlarrGrab   *prowlarr.GrabProvider
}

// AutomationGroup holds automation and scheduled task services.
type AutomationGroup struct {
	Autosearch         *autosearch.Service
	ScheduledSearcher  *autosearch.ScheduledSearcher
	AutosearchSettings *autosearch.SettingsHandler
	RssSync            *rsssync.Service
	RssSyncSettings    *rsssync.SettingsHandler
	Import             *importer.Service
	ImportSettings     *importer.SettingsHandlers
	ArrImport          *arrimport.Service
	Scheduler          *scheduler.Scheduler
	FeedFetcher        *rsssync.FeedFetcher
}

// SystemGroup holds system-level services.
type SystemGroup struct {
	Health       *health.Service
	Defaults     *defaults.Service
	Calendar     *calendar.Service
	Availability *availability.Service
	Missing      *missing.Service
	Preferences  *preferences.Service
	History      *history.Service
	Progress     *progress.Manager
	Update       *update.Service
	Firewall     *firewall.Checker
	Logs         LogsProvider
}

// NotificationGroup holds notification services.
type NotificationGroup struct {
	Service      *notification.Service
	PlexClient   *plex.Client
	PlexHandlers *plex.Handlers
}

// PortalGroup holds external requests portal services.
type PortalGroup struct {
	Users               *users.Service
	Invitations         *invitations.Service
	Requests            *requests.Service
	Quota               *quota.Service
	Notifications       *portalnotifs.Service
	AutoApprove         *autoapprove.Service
	Auth                *auth.Service
	Passkey             *auth.PasskeyService
	AuthMiddleware      *portalmw.AuthMiddleware
	SearchLimiter       *portalratelimit.SearchLimiter
	RequestSearcher     *requests.RequestSearcher
	MediaProvisioner    *provisioner.Service
	Watchers            *requests.WatchersService
	StatusTracker       *requests.StatusTracker
	LibraryChecker      *requests.LibraryChecker
	AdminLibraryChecker *adminRequestLibraryCheckerAdapter
	AdminSettings       *admin.SettingsHandlers
}

// SecurityGroup holds security services.
type SecurityGroup struct {
	AuthLimiter *authratelimit.AuthLimiter
}

// ServiceContainer holds all service groups, assembled by Wire.
type ServiceContainer struct {
	System       SystemGroup
	Library      LibraryGroup
	Metadata     MetadataGroup
	Filesystem   FilesystemGroup
	Download     DownloadGroup
	Search       SearchGroup
	Automation   AutomationGroup
	Notification NotificationGroup
	Portal       PortalGroup
	Security     SecurityGroup
	Switchable   SwitchableServices
}

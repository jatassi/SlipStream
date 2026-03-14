package api

import (
	"context"
	"time"

	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/firewall"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/metadata/omdb"
	"github.com/slipstream/slipstream/internal/metadata/tmdb"
	"github.com/slipstream/slipstream/internal/metadata/tvdb"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/rsssync"
	"github.com/slipstream/slipstream/internal/scheduler/tasks"
	"github.com/slipstream/slipstream/internal/update"
)

// wireCircularDeps wires setter-based dependencies that form dependency cycles
// and therefore cannot be expressed as constructor parameters.
func wireCircularDeps(s *Server) {
	// Circular: Movies ↔ Slots (Movies deletes files via Slots, Slots deletes media via Movies/TV)
	s.library.Movies.SetFileDeleteHandler(s.library.Slots)
	s.library.TV.SetFileDeleteHandler(s.library.Slots)
	s.library.Slots.SetFileDeleter(&slotFileDeleterAdapter{
		movieSvc: s.library.Movies,
		tvSvc:    s.library.TV,
	})

	// Circular: Quality ↔ Import
	s.library.Quality.SetImportDecisionCleaner(s.automation.Import)

	// Module registry → services
	s.library.LibraryManager.SetRegistry(s.registry)
	s.library.Movies.SetRegistry(s.registry)
	s.library.TV.SetRegistry(s.registry)
	s.search.Search.SetRegistry(s.registry)
	s.automation.Autosearch.SetRegistry(s.registry)
	s.automation.ScheduledSearcher.SetRegistry(s.registry)
	s.automation.RssSync.SetRegistry(s.registry)
	s.download.Service.SetRegistry(s.registry)
	s.automation.Import.SetRegistry(s.registry)

	// Register module quality items with the quality service
	registerModuleQualities(s.registry, s.library.Quality)

	// Circular: LibraryManager ↔ Autosearch
	s.library.LibraryManager.SetAutosearchService(s.automation.Autosearch)
	s.automation.ScheduledSearcher.SetSeriesRefresher(s.library.LibraryManager)

	// Circular: Notification → many consumers
	s.system.Health.SetNotifier(s.notification.Service)
	s.search.Grab.SetNotificationService(&grabNotificationAdapter{
		svc:    s.notification.Service,
		movies: s.library.Movies,
		tv:     s.library.TV,
	})
	s.library.Movies.SetNotificationDispatcher(&movieNotificationAdapter{s.notification.Service})
	s.library.TV.SetNotificationDispatcher(&tvNotificationAdapter{s.notification.Service})
	s.automation.Import.SetNotificationDispatcher(&importNotificationAdapter{s.notification.Service})

	// ArrImport → multiple services (notification-dependent)
	s.automation.ArrImport.SetConfigImportServices(
		s.download.Service,
		s.search.Indexer,
		s.notification.Service,
		s.library.Quality,
		s.automation.Import,
	)

	// QueueBroadcaster → Import (cross-group)
	if s.download.QueueBroadcaster != nil {
		s.download.QueueBroadcaster.SetCompletionHandler(s.automation.Import)
	}

	// Portal: AutoApprove ↔ RequestSearcher
	s.portal.RequestSearcher.SetUserGetter(&portalUserQualityProfileAdapter{usersSvc: s.portal.Users})
	s.portal.RequestSearcher.SetDevMode(s.dbManager.IsDevMode)
	s.portal.RequestSearcher.SetRegistry(s.registry)
	s.portal.AutoApprove.SetRequestSearcher(s.portal.RequestSearcher)
	s.portal.AutoApprove.SetRegistry(s.registry)
	s.portal.Quota.SetRegistry(s.registry)
}

// wireLateBindings wires dependencies that are unavailable at construction time:
// lambda callbacks, configuration loading, scheduler tasks, and cleanup goroutines.
func wireLateBindings(s *Server) {
	s.loadSavedSettings()

	// WebSocket hub callbacks
	if s.hub != nil {
		s.hub.SetDevModeHandler(s.devMode.OnToggle)
	}

	// Token validator (depends on auth service)
	if s.hub != nil && s.portal.Auth != nil {
		s.hub.SetTokenValidator(func(token string) error {
			_, err := s.portal.Auth.ValidateAdminToken(token)
			return err
		})
	}

	// Cardigann cookie store (depends on status service created by Wire)
	if manager := s.search.Indexer.GetManager(); manager != nil {
		cookieStore := indexer.NewCookieStore(s.search.Status)
		manager.SetCookieStore(cookieStore)

		// Mode check func for Cardigann definition updates
		modeManager := s.search.ProwlarrMode
		manager.SetModeCheckFunc(func() bool {
			isSlipStream, err := modeManager.IsSlipStreamMode(context.Background())
			if err != nil {
				return true
			}
			return isSlipStream
		})
	}

	// Store real metadata clients for dev mode switching
	s.metadata.RealTMDBClient = tmdb.NewClient(s.cfg.Metadata.TMDB, s.logger)
	s.metadata.RealTVDBClient = tvdb.NewClient(s.cfg.Metadata.TVDB, s.logger)
	s.metadata.RealOMDBClient = omdb.NewClient(s.cfg.Metadata.OMDB, s.logger)

	// Initialize update service (depends on scheduler)
	s.system.Update = update.NewService(s.dbManager.Conn(), s.logger, s.restartChan)
	s.system.Update.SetBroadcaster(s.hub)
	s.system.Update.SetPort(s.cfg.Server.Port)

	// Initialize firewall checker
	s.system.Firewall = firewall.NewChecker()

	// Scheduler callbacks and task registration
	if s.automation.Scheduler != nil {
		s.automation.Scheduler.OnTaskStateChanged(schedulerBroadcaster(s.hub))
		s.registerSchedulerTasks()
	}

	// Settings handler scheduler bindings
	if s.automation.Scheduler != nil {
		s.automation.AutosearchSettings.SetScheduler(s.automation.Scheduler, s.automation.ScheduledSearcher, tasks.UpdateAutoSearchTask)
		s.automation.RssSyncSettings.SetScheduler(s.automation.Scheduler, s.automation.RssSync, tasks.UpdateRssSyncTask)
	}

	// Start cleanup goroutines
	s.portal.SearchLimiter.StartCleanup(5 * time.Minute)
	s.security.AuthLimiter.StartCleanup(5 * time.Minute)
}

// loadSavedSettings loads persisted settings into runtime config and caches.
func (s *Server) loadSavedSettings() {
	db := s.dbManager.Conn()
	queries := sqlc.New(db)
	ctx := context.Background()

	if err := autosearch.LoadSettingsIntoConfig(ctx, queries, &s.cfg.AutoSearch); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load autosearch settings, using defaults")
	}
	if err := rsssync.LoadSettingsIntoConfig(ctx, queries, &s.cfg.RssSync); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load RSS sync settings, using defaults")
	}
	if err := s.registry.LoadEnabledState(ctx, db); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to load module enabled state, all modules enabled by default")
	}
}

// registerSchedulerTasks registers all scheduler tasks.
func (s *Server) registerSchedulerTasks() {
	sched := s.automation.Scheduler
	logger := s.logger
	cfg := s.cfg

	if err := tasks.RegisterAvailabilityTask(sched, s.system.Availability); err != nil {
		logger.Error().Err(err).Msg("Failed to register availability task")
	}
	if err := tasks.RegisterAutoSearchTask(sched, s.automation.ScheduledSearcher, &cfg.AutoSearch); err != nil {
		logger.Error().Err(err).Msg("Failed to register autosearch task")
	}
	if err := tasks.RegisterRssSyncTask(sched, s.automation.RssSync, &cfg.RssSync); err != nil {
		logger.Error().Err(err).Msg("Failed to register RSS sync task")
	}
	s.registerLibraryDependentTasks(cfg, logger)

	if err := tasks.RegisterUpdateCheckTask(sched, s.system.Update, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register update check task")
	}

	queries := sqlc.New(s.dbManager.Conn())
	if err := tasks.RegisterPlexRefreshTask(sched, queries, s.notification.Service, s.notification.PlexClient, logger); err != nil {
		logger.Error().Err(err).Msg("Failed to register Plex refresh task")
	}
}

// registerModuleQualities registers quality items from each module with the quality service.
func registerModuleQualities(registry *module.Registry, qualitySvc *quality.Service) {
	for _, mod := range registry.All() {
		items := mod.QualityItems()
		if len(items) == 0 {
			continue
		}
		defs := make([]quality.QualityItemDef, len(items))
		for i, item := range items {
			defs[i] = quality.QualityItemDef{
				ID:         item.ID,
				Name:       item.Name,
				Source:     item.Source,
				Resolution: item.Resolution,
				Weight:     item.Weight,
			}
		}
		qualitySvc.RegisterModuleQualities(string(mod.ID()), defs)
	}
}

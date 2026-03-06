package api

import (
	"context"
	"database/sql"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/auth"
	"github.com/slipstream/slipstream/internal/config"
	"github.com/slipstream/slipstream/internal/database"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/domain/contracts"
	"github.com/slipstream/slipstream/internal/history"
	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/indexer/cardigann"
	"github.com/slipstream/slipstream/internal/indexer/ratelimit"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/organizer"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/mediainfo"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/module"
	moviemod "github.com/slipstream/slipstream/internal/modules/movie"
	tvmod "github.com/slipstream/slipstream/internal/modules/tv"
	"github.com/slipstream/slipstream/internal/notification/plex"
	"github.com/slipstream/slipstream/internal/portal/admin"
	portalmw "github.com/slipstream/slipstream/internal/portal/middleware"
	portalratelimit "github.com/slipstream/slipstream/internal/portal/ratelimit"
	"github.com/slipstream/slipstream/internal/portal/requests"
	"github.com/slipstream/slipstream/internal/prowlarr"
	"github.com/slipstream/slipstream/internal/rsssync"
	"github.com/slipstream/slipstream/internal/scheduler"
)

// --- Infrastructure providers ---

func provideDB(dbManager *database.Manager) *sql.DB {
	return dbManager.Conn()
}

func provideQueries(db *sql.DB) *sqlc.Queries {
	return sqlc.New(db)
}

// --- Config extraction providers ---

func provideMetadataConfig(cfg *config.Config) *config.MetadataConfig {
	return &cfg.Metadata
}

func provideArtworkConfig(cfg *config.Config) metadata.ArtworkConfig {
	dataDir := filepath.Dir(cfg.Database.Path)
	return metadata.ArtworkConfig{
		BaseDir: filepath.Join(dataDir, "artwork"),
		Timeout: 30 * time.Second,
	}
}

func provideAutoSearchConfig(cfg *config.Config) *config.AutoSearchConfig {
	return &cfg.AutoSearch
}

func provideRssSyncConfig(cfg *config.Config) *config.RssSyncConfig {
	return &cfg.RssSync
}

func provideRateLimitConfig() ratelimit.Config {
	return ratelimit.DefaultConfig()
}

func provideNamingConfig() *organizer.NamingConfig {
	nc := organizer.DefaultNamingConfig()
	return &nc
}

func provideMediainfoConfig() mediainfo.Config {
	return mediainfo.DefaultConfig()
}

func provideImporterConfig() importer.Config {
	return importer.DefaultConfig()
}

// --- Error-swallowing providers for services with non-fatal failures ---

func provideCardigannManager(cfg *config.Config, logger *zerolog.Logger) *cardigann.Manager {
	cardigannCfg := cardigann.ManagerConfig{
		Repository: cardigann.RepositoryConfig{
			BaseURL:        cfg.Indexer.Cardigann.RepositoryURL,
			Branch:         cfg.Indexer.Cardigann.Branch,
			Version:        cfg.Indexer.Cardigann.Version,
			RequestTimeout: cfg.Indexer.Cardigann.RequestTimeoutDuration(),
			UserAgent:      "SlipStream/1.0",
		},
		Cache: cardigann.CacheConfig{
			DefinitionsDir: cfg.Indexer.Cardigann.DefinitionsDir,
			CustomDir:      cfg.Indexer.Cardigann.CustomDir,
		},
		AutoUpdate:     cfg.Indexer.Cardigann.AutoUpdate,
		UpdateInterval: cfg.Indexer.Cardigann.UpdateIntervalDuration(),
	}
	manager, err := cardigann.NewManager(&cardigannCfg, logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize Cardigann manager")
		return nil
	}
	return manager
}

func provideScheduler(logger *zerolog.Logger) *scheduler.Scheduler {
	sched, err := scheduler.New(logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize scheduler")
		return nil
	}
	return sched
}

func providePasskeyService(queries *sqlc.Queries, cfg *config.Config) *auth.PasskeyService {
	svc, err := auth.NewPasskeyService(queries, auth.PasskeyConfig{
		RPDisplayName: cfg.Portal.WebAuthn.RPDisplayName,
		RPID:          cfg.Portal.WebAuthn.RPID,
		RPOrigins:     cfg.Portal.WebAuthn.RPOrigins,
	})
	if err != nil {
		return nil
	}
	return svc
}

// --- Providers with ambiguous or complex params ---

func provideAuthService(queries *sqlc.Queries, logger *zerolog.Logger, cfg *config.Config) (*auth.Service, error) {
	return auth.NewService(queries, logger, cfg.Portal.JWTSecret)
}

func providePlexClient(logger *zerolog.Logger) *plex.Client {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	return plex.NewClient(httpClient, logger, config.Version)
}

func provideModeManager(service *prowlarr.Service, dbManager *database.Manager) *prowlarr.ModeManager {
	return prowlarr.NewModeManager(service, dbManager.IsDevMode)
}

func provideFeedFetcher(
	indexerService *indexer.Service,
	prowlarrService *prowlarr.Service,
	modeManager *prowlarr.ModeManager,
	queries *sqlc.Queries,
	logger *zerolog.Logger,
) *rsssync.FeedFetcher {
	return rsssync.NewFeedFetcher(indexerService, prowlarrService, modeManager, queries, logger)
}

func provideSearchLimiter(queries *sqlc.Queries) *portalratelimit.SearchLimiter {
	return portalratelimit.NewSearchLimiter(func() int64 {
		if setting, err := queries.GetSetting(context.Background(), admin.SettingSearchRateLimit); err == nil && setting.Value != "" {
			if v, parseErr := strconv.ParseInt(setting.Value, 10, 64); parseErr == nil {
				return v
			}
		}
		return portalratelimit.DefaultRequestsPerMinute
	})
}

// --- Module providers ---

func provideRegistry(movieMod *moviemod.Module, tvMod *tvmod.Module) *module.Registry {
	reg := module.NewRegistry()
	reg.Register(movieMod)
	reg.Register(tvMod)
	return reg
}

// --- Adapter providers for interface bindings ---

func provideStatusChangeLogger(h *history.Service) contracts.StatusChangeLogger {
	return &statusChangeLoggerAdapter{svc: h}
}

func provideImportHistoryService(h *history.Service) importer.HistoryService {
	return &importHistoryAdapter{svc: h}
}

func provideMovieLookup(m *movies.Service) requests.MovieLookup {
	return &statusTrackerMovieLookup{movieSvc: m}
}

func provideEpisodeLookup(t *tv.Service) requests.EpisodeLookup {
	return &statusTrackerEpisodeLookup{tvSvc: t}
}

func providePortalEnabledChecker(q *sqlc.Queries) portalmw.PortalEnabledChecker {
	return &portalEnabledChecker{queries: q}
}

func provideAdminLibraryChecker(q *sqlc.Queries) *adminRequestLibraryCheckerAdapter {
	return &adminRequestLibraryCheckerAdapter{queries: q}
}

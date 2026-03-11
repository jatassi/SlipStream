package api

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader"
	downloadermock "github.com/slipstream/slipstream/internal/downloader/mock"
	fsmock "github.com/slipstream/slipstream/internal/filesystem/mock"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/metadata/mock"
	"github.com/slipstream/slipstream/internal/module"
	moviemod "github.com/slipstream/slipstream/internal/modules/movie"
	tvmod "github.com/slipstream/slipstream/internal/modules/tv"
	"github.com/slipstream/slipstream/internal/notification"
)

// DevModeManager handles all dev mode toggling logic: switching databases,
// creating mock services, and copying production data to dev databases.
type DevModeManager struct {
	library      *LibraryGroup
	metadata     *MetadataGroup
	search       *SearchGroup
	download     *DownloadGroup
	notification *NotificationGroup
	switchable   *SwitchableServices
	dbManager    *database.Manager
	logger       *zerolog.Logger
	registry     *module.Registry
}

// NewDevModeManager creates a new DevModeManager with references to the server's service groups.
func NewDevModeManager(library *LibraryGroup, meta *MetadataGroup, search *SearchGroup,
	download *DownloadGroup, notif *NotificationGroup, switchable *SwitchableServices,
	dbManager *database.Manager, logger *zerolog.Logger, registry *module.Registry) *DevModeManager {
	return &DevModeManager{
		library:      library,
		metadata:     meta,
		search:       search,
		download:     download,
		notification: notif,
		switchable:   switchable,
		dbManager:    dbManager,
		logger:       logger,
		registry:     registry,
	}
}

func (d *DevModeManager) prodAndDevQueries() (prodQueries, devQueries *sqlc.Queries) {
	return sqlc.New(d.dbManager.ProdConn()), sqlc.New(d.dbManager.Conn())
}

// OnToggle handles the dev mode toggle event from the WebSocket hub.
func (d *DevModeManager) OnToggle(enabled bool) error {
	if err := d.dbManager.SetDevMode(enabled); err != nil {
		return err
	}
	if enabled {
		if err := module.MigrateAll(d.dbManager.Conn(), d.registry); err != nil {
			d.logger.Error().Err(err).Msg("failed to run module migrations on dev database")
			return err
		}
		d.copyJWTSecretToDevDB()
		d.copySettingsToDevDB()
	}
	d.updateServicesDB()
	d.switchMetadataClients(enabled)
	d.switchIndexer(enabled)
	d.switchDownloadClient(enabled)
	d.switchNotification(enabled)
	d.switchRootFolders(enabled)
	if enabled {
		profileIDMapping := d.copyQualityProfilesToDevDB()
		d.copyPortalUsersToDevDB(profileIDMapping)
		d.copyPortalUserNotificationsToDevDB()
		d.setupDevModeSlots()
		d.populateMockMedia()
	}
	return nil
}

func (d *DevModeManager) switchMetadataClients(devMode bool) {
	if devMode {
		d.logger.Info().Msg("Switching to mock metadata providers")
		d.metadata.Service.SetClients(mock.NewTMDBClient(), mock.NewTVDBClient(), mock.NewOMDBClient())
	} else {
		d.logger.Info().Msg("Switching to real metadata providers")
		d.metadata.Service.SetClients(d.metadata.RealTMDBClient, d.metadata.RealTVDBClient, d.metadata.RealOMDBClient)
	}
}

// switchIndexer creates or removes the mock indexer based on dev mode.
func (d *DevModeManager) switchIndexer(devMode bool) {
	ctx := context.Background()

	if devMode {
		// Check if mock indexer already exists
		indexers, err := d.search.Indexer.List(ctx)
		if err != nil {
			d.logger.Error().Err(err).Msg("Failed to list indexers for dev mode")
			return
		}

		// Look for existing mock indexer
		for _, idx := range indexers {
			if idx.DefinitionID == indexer.MockDefinitionID {
				d.logger.Info().Int64("id", idx.ID).Msg("Mock indexer already exists")
				return
			}
		}

		// Create mock indexer
		_, err = d.search.Indexer.Create(ctx, &indexer.CreateIndexerInput{
			Name:           "Mock Indexer",
			DefinitionID:   indexer.MockDefinitionID,
			SupportsMovies: true,
			SupportsTV:     true,
			Enabled:        true,
			Priority:       1,
		})
		if err != nil {
			d.logger.Error().Err(err).Msg("Failed to create mock indexer")
			return
		}
		d.logger.Info().Msg("Created mock indexer for dev mode")
	} else {
		d.logger.Info().Msg("Dev mode disabled - mock indexer will remain until manually deleted")
	}
}

// updateServicesDB updates all services to use the current database connection.
// This must be called after switching databases (e.g., when toggling dev mode).
func (d *DevModeManager) updateServicesDB() {
	db := d.dbManager.Conn()
	d.switchable.UpdateAll(db)

	if err := d.switchable.Auth.SetDB(sqlc.New(db)); err != nil {
		d.logger.Error().Err(err).Msg("Failed to switch auth service database")
	}

	d.logger.Info().Msg("Updated all services with new database connection")
}

// switchDownloadClient creates or removes the mock download client based on dev mode.
func (d *DevModeManager) switchDownloadClient(devMode bool) {
	ctx := context.Background()

	if devMode {
		// Check if mock client already exists
		clients, err := d.download.Service.List(ctx)
		if err != nil {
			d.logger.Error().Err(err).Msg("Failed to list download clients for dev mode")
			return
		}

		// Look for existing mock client
		for _, c := range clients {
			if c.Type == "mock" {
				d.logger.Info().Int64("id", c.ID).Msg("Mock download client already exists")
				return
			}
		}

		// Create mock download client
		_, err = d.download.Service.Create(ctx, &downloader.CreateClientInput{
			Name:     "Mock Download Client",
			Type:     "mock",
			Host:     "localhost",
			Port:     9999,
			Enabled:  true,
			Priority: 1,
		})
		if err != nil {
			d.logger.Error().Err(err).Msg("Failed to create mock download client")
			return
		}
		d.logger.Info().Msg("Created mock download client for dev mode")
	} else {
		// Clear mock downloads when disabling dev mode
		downloadermock.GetInstance().Clear()
		d.logger.Info().Msg("Cleared mock downloads")
	}
}

// switchNotification creates mock notification based on dev mode.
func (d *DevModeManager) switchNotification(devMode bool) {
	ctx := context.Background()

	if devMode {
		// Check if mock notification already exists
		notifications, err := d.notification.Service.List(ctx)
		if err != nil {
			d.logger.Error().Err(err).Msg("Failed to list notifications for dev mode")
			return
		}

		// Look for existing mock notification
		for i := range notifications {
			n := &notifications[i]
			if n.Type == notification.NotifierMock {
				d.logger.Info().Int64("id", n.ID).Msg("Mock notification already exists")
				return
			}
		}

		// Create mock notification (subscribed to all events)
		_, err = d.notification.Service.Create(ctx, &notification.CreateInput{
			Name:    "Mock Notification",
			Type:    notification.NotifierMock,
			Enabled: true,
			EventToggles: map[string]bool{
				notification.EventGrab:           true,
				notification.EventImport:         true,
				notification.EventUpgrade:        true,
				moviemod.EventMovieAdded:         true,
				moviemod.EventMovieDeleted:       true,
				tvmod.EventTVAdded:               true,
				tvmod.EventTVDeleted:             true,
				notification.EventHealthIssue:    true,
				notification.EventHealthRestored: true,
				notification.EventAppUpdate:      true,
			},
		})
		if err != nil {
			d.logger.Error().Err(err).Msg("Failed to create mock notification")
			return
		}
		d.logger.Info().Msg("Created mock notification for dev mode")
	} else {
		d.logger.Info().Msg("Dev mode disabled - mock notification will remain until manually deleted")
	}
}

// mockRootFolderAdapter adapts rootfolder.Service to module.MockRootFolderCreator.
type mockRootFolderAdapter struct {
	service *rootfolder.Service
}

func (a *mockRootFolderAdapter) Create(ctx context.Context, path, name, mediaType string) (int64, error) {
	rf, err := a.service.Create(ctx, rootfolder.CreateRootFolderInput{
		Path:      path,
		Name:      name,
		MediaType: mediaType,
	})
	if err != nil {
		return 0, err
	}
	return rf.ID, nil
}

// switchRootFolders creates mock root folders via module MockFactory.
func (d *DevModeManager) switchRootFolders(devMode bool) {
	if !devMode {
		d.logger.Info().Msg("Dev mode disabled - mock root folders will remain until manually deleted")
		return
	}

	ctx := context.Background()
	fsmock.ResetInstance()

	mctx := d.buildMockContext(ctx)
	for _, mod := range d.registry.All() {
		if err := mod.CreateTestRootFolders(ctx, mctx); err != nil {
			d.logger.Error().Err(err).Str("module", string(mod.ID())).Msg("Failed to create test root folders")
		}
	}
}

// populateMockMedia creates mock media via module MockFactory.
func (d *DevModeManager) populateMockMedia() {
	ctx := context.Background()
	mctx := d.buildMockContext(ctx)
	for _, mod := range d.registry.All() {
		if err := mod.CreateSampleLibraryData(ctx, mctx); err != nil {
			d.logger.Error().Err(err).Str("module", string(mod.ID())).Msg("Failed to create sample library data")
		}
	}
}

func (d *DevModeManager) buildMockContext(ctx context.Context) *module.MockContext {
	profiles, _ := d.library.Quality.List(ctx)
	mockProfiles := make([]module.MockQualityProfile, len(profiles))
	for i, p := range profiles {
		mockProfiles[i] = module.MockQualityProfile{ID: p.ID, Name: p.Name}
	}
	var defaultProfileID int64
	if len(profiles) > 0 {
		defaultProfileID = profiles[0].ID
	}

	return &module.MockContext{
		DB:                d.dbManager.Conn(),
		Logger:            d.logger,
		RootFolderCreator: &mockRootFolderAdapter{service: d.library.RootFolder},
		QualityProfiles:   mockProfiles,
		DefaultProfileID:  defaultProfileID,
	}
}

// copyJWTSecretToDevDB copies the JWT secret from production to dev database.
// This ensures tokens issued in production mode remain valid in dev mode.
func (d *DevModeManager) copyJWTSecretToDevDB() {
	ctx := context.Background()

	prodQueries, devQueries := d.prodAndDevQueries()

	// Get JWT secret from production database
	setting, err := prodQueries.GetSetting(ctx, "portal_jwt_secret")
	if err != nil {
		d.logger.Debug().Err(err).Msg("No JWT secret in production database to copy")
		return
	}

	if setting.Value == "" {
		d.logger.Debug().Msg("Production JWT secret is empty, nothing to copy")
		return
	}

	// Copy to dev database
	_, err = devQueries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "portal_jwt_secret",
		Value: setting.Value,
	})
	if err != nil {
		d.logger.Error().Err(err).Msg("Failed to copy JWT secret to dev database")
		return
	}

	d.logger.Info().Msg("Copied JWT secret from production to dev database")
}

// copySettingsToDevDB copies application settings from production to dev database.
func (d *DevModeManager) copySettingsToDevDB() {
	ctx := context.Background()

	prodQueries, devQueries := d.prodAndDevQueries()

	settingKeys := []string{"server_port", "log_level", "api_key"}
	copied := 0

	for _, key := range settingKeys {
		setting, err := prodQueries.GetSetting(ctx, key)
		if err != nil {
			continue
		}

		if setting.Value == "" {
			continue
		}

		_, err = devQueries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   key,
			Value: setting.Value,
		})
		if err != nil {
			d.logger.Error().Err(err).Str("key", key).Msg("Failed to copy setting to dev database")
			continue
		}
		copied++
	}

	if copied > 0 {
		d.logger.Info().Int("count", copied).Msg("Copied settings to dev database")
	}
}

// copyPortalUsersToDevDB copies portal users from production to dev database.
// This preserves user IDs so that JWTs issued against prod DB work in dev mode.
// profileIDMapping maps production quality profile IDs to dev database IDs.
func (d *DevModeManager) copyPortalUsersToDevDB(profileIDMapping map[int64]int64) {
	ctx := context.Background()

	prodQueries, devQueries := d.prodAndDevQueries()

	// Get users from production database
	prodUsers, err := prodQueries.ListPortalUsers(ctx)
	if err != nil {
		d.logger.Error().Err(err).Msg("Failed to list production portal users")
		return
	}

	if len(prodUsers) == 0 {
		d.logger.Debug().Msg("No portal users in production database to copy")
		return
	}

	// Copy each user to dev database (skip if already exists)
	copied := 0
	for _, user := range prodUsers {
		_, err := devQueries.GetPortalUser(ctx, user.ID)
		if err == nil {
			continue
		}

		_, err = devQueries.CreatePortalUserWithID(ctx, sqlc.CreatePortalUserWithIDParams{
			ID:           user.ID,
			Username:     user.Username,
			PasswordHash: user.PasswordHash,
			AutoApprove:  user.AutoApprove,
			Enabled:      user.Enabled,
			IsAdmin:      user.IsAdmin,
		})
		if err != nil {
			d.logger.Error().Err(err).Str("username", user.Username).Msg("Failed to copy portal user")
			continue
		}

		d.copyUserModuleSettingsToDevDB(ctx, prodQueries, devQueries, user.ID, profileIDMapping)
		copied++
	}

	if copied > 0 {
		d.logger.Info().Int("count", copied).Msg("Copied portal users to dev database")
	}
}

func (d *DevModeManager) copyUserModuleSettingsToDevDB(ctx context.Context, prodQueries, devQueries *sqlc.Queries, userID int64, profileIDMapping map[int64]int64) {
	prodSettings, err := prodQueries.ListUserModuleSettings(ctx, userID)
	if err != nil {
		return
	}
	for _, ms := range prodSettings {
		qpID := ms.QualityProfileID
		if qpID.Valid {
			qpID = mapProfileID(qpID, profileIDMapping)
		}
		_, _ = devQueries.UpsertUserModuleSettings(ctx, sqlc.UpsertUserModuleSettingsParams{
			UserID:           userID,
			ModuleType:       ms.ModuleType,
			QuotaLimit:       ms.QuotaLimit,
			QuotaUsed:        ms.QuotaUsed,
			QualityProfileID: qpID,
			PeriodStart:      ms.PeriodStart,
		})
	}
}

// copyPortalUserNotificationsToDevDB copies portal user notification channels from production to dev database.
func (d *DevModeManager) copyPortalUserNotificationsToDevDB() {
	ctx := context.Background()

	prodQueries, devQueries := d.prodAndDevQueries()

	prodUsers, err := prodQueries.ListPortalUsers(ctx)
	if err != nil {
		d.logger.Error().Err(err).Msg("Failed to list production portal users for notification copy")
		return
	}

	copied := 0
	for _, user := range prodUsers {
		notifs, err := prodQueries.ListUserNotifications(ctx, user.ID)
		if err != nil {
			d.logger.Error().Err(err).Int64("user_id", user.ID).Msg("Failed to list user notifications")
			continue
		}

		for _, n := range notifs {
			_, err := devQueries.CreateUserNotification(ctx, sqlc.CreateUserNotificationParams{
				UserID:      n.UserID,
				Type:        n.Type,
				Name:        n.Name,
				Settings:    n.Settings,
				OnAvailable: n.OnAvailable,
				OnApproved:  n.OnApproved,
				OnDenied:    n.OnDenied,
				Enabled:     n.Enabled,
			})
			if err != nil {
				d.logger.Error().Err(err).Str("name", n.Name).Msg("Failed to copy user notification")
				continue
			}
			copied++
		}
	}

	if copied > 0 {
		d.logger.Info().Int("count", copied).Msg("Copied portal user notifications to dev database")
	}
}

// copyQualityProfilesToDevDB copies quality profiles from production to dev database.
// Returns a mapping of production profile IDs to dev profile IDs.
func (d *DevModeManager) copyQualityProfilesToDevDB() map[int64]int64 {
	ctx := context.Background()
	idMapping := make(map[int64]int64)

	// Check if dev database already has profiles
	devProfiles, err := d.library.Quality.List(ctx)
	if err != nil {
		d.logger.Error().Err(err).Msg("Failed to list dev quality profiles")
		return idMapping
	}
	if len(devProfiles) > 0 {
		d.logger.Info().Int("count", len(devProfiles)).Msg("Dev database already has quality profiles")
		return idMapping
	}

	prodQueries, devQueries := d.prodAndDevQueries()

	// Get profiles from production database
	prodRows, err := prodQueries.ListQualityProfiles(ctx)
	if err != nil {
		d.logger.Error().Err(err).Msg("Failed to list production quality profiles")
		return idMapping
	}

	if len(prodRows) == 0 {
		d.logger.Warn().Msg("No quality profiles in production database to copy")
		// Create default profiles in dev database
		for _, mt := range []string{"movie", "tv"} {
			if err := d.library.Quality.EnsureDefaults(ctx, mt); err != nil {
				d.logger.Warn().Err(err).Str("moduleType", mt).Msg("Failed to ensure default quality profiles")
			}
		}
		return idMapping
	}

	// Copy each profile to dev database and track ID mapping
	for _, row := range prodRows {
		newProfile, err := devQueries.CreateQualityProfile(ctx, sqlc.CreateQualityProfileParams{
			Name:                    row.Name,
			ModuleType:              row.ModuleType,
			Cutoff:                  row.Cutoff,
			Items:                   row.Items,
			HdrSettings:             row.HdrSettings,
			VideoCodecSettings:      row.VideoCodecSettings,
			AudioCodecSettings:      row.AudioCodecSettings,
			AudioChannelSettings:    row.AudioChannelSettings,
			UpgradesEnabled:         row.UpgradesEnabled,
			AllowAutoApprove:        row.AllowAutoApprove,
			UpgradeStrategy:         row.UpgradeStrategy,
			CutoffOverridesStrategy: row.CutoffOverridesStrategy,
		})
		if err != nil {
			d.logger.Error().Err(err).Str("name", row.Name).Msg("Failed to copy quality profile")
			continue
		}
		idMapping[row.ID] = newProfile.ID
		d.logger.Debug().Str("name", row.Name).Int64("prodID", row.ID).Int64("devID", newProfile.ID).Msg("Copied quality profile to dev database")
	}

	d.logger.Info().Int("count", len(prodRows)).Msg("Copied quality profiles to dev database")
	return idMapping
}

// setupDevModeSlots configures version slots for developer mode testing.
// Assigns quality profiles to slots and enables them so the dry run feature works.
func (d *DevModeManager) setupDevModeSlots() {
	ctx := context.Background()

	// Get all slots
	slotList, err := d.library.Slots.List(ctx)
	if err != nil {
		d.logger.Error().Err(err).Msg("Failed to list slots for dev mode setup")
		return
	}

	// Get all quality profiles
	profiles, err := d.library.Quality.List(ctx)
	if err != nil {
		d.logger.Error().Err(err).Msg("Failed to list quality profiles for dev mode setup")
		return
	}

	if len(profiles) == 0 {
		d.logger.Warn().Msg("No quality profiles available for dev mode slot setup")
		return
	}

	// Assign profiles to slots and enable them
	// Use different profiles for each slot if available
	for i, slot := range slotList {
		profileIdx := i % len(profiles)
		profileID := profiles[profileIdx].ID

		input := slots.UpdateSlotInput{
			Name:         slot.Name,
			Enabled:      true,
			DisplayOrder: slot.DisplayOrder,
		}
		input.QualityProfileID = &profileID

		_, err := d.library.Slots.Update(ctx, slot.ID, input)
		if err != nil {
			d.logger.Error().Err(err).Int64("slotId", slot.ID).Msg("Failed to update slot for dev mode")
			continue
		}
		d.logger.Debug().Int64("slotId", slot.ID).Int64("profileId", profileID).Msg("Configured slot for dev mode")
	}

	d.logger.Info().Int("count", len(slotList)).Msg("Configured slots for dev mode")
}

func mapProfileID(id sql.NullInt64, mapping map[int64]int64) sql.NullInt64 {
	if id.Valid {
		if newID, ok := mapping[id.Int64]; ok {
			return sql.NullInt64{Int64: newID, Valid: true}
		}
	}
	return sql.NullInt64{}
}

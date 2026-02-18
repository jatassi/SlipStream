package api

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader"
	downloadermock "github.com/slipstream/slipstream/internal/downloader/mock"
	fsmock "github.com/slipstream/slipstream/internal/filesystem/mock"
	"github.com/slipstream/slipstream/internal/indexer"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/metadata/mock"
	"github.com/slipstream/slipstream/internal/notification"
)

func (s *Server) switchMetadataClients(devMode bool) {
	if devMode {
		s.logger.Info().Msg("Switching to mock metadata providers")
		s.metadataService.SetClients(mock.NewTMDBClient(), mock.NewTVDBClient(), mock.NewOMDBClient())
	} else {
		s.logger.Info().Msg("Switching to real metadata providers")
		s.metadataService.SetClients(s.realTMDBClient, s.realTVDBClient, s.realOMDBClient)
	}
}

// switchIndexer creates or removes the mock indexer based on dev mode.
func (s *Server) switchIndexer(devMode bool) {
	ctx := context.Background()

	if devMode {
		// Check if mock indexer already exists
		indexers, err := s.indexerService.List(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to list indexers for dev mode")
			return
		}

		// Look for existing mock indexer
		for _, idx := range indexers {
			if idx.DefinitionID == indexer.MockDefinitionID {
				s.logger.Info().Int64("id", idx.ID).Msg("Mock indexer already exists")
				return
			}
		}

		// Create mock indexer
		_, err = s.indexerService.Create(ctx, &indexer.CreateIndexerInput{
			Name:           "Mock Indexer",
			DefinitionID:   indexer.MockDefinitionID,
			SupportsMovies: true,
			SupportsTV:     true,
			Enabled:        true,
			Priority:       1,
		})
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to create mock indexer")
			return
		}
		s.logger.Info().Msg("Created mock indexer for dev mode")
	} else {
		s.logger.Info().Msg("Dev mode disabled - mock indexer will remain until manually deleted")
	}
}

// updateServicesDB updates all services to use the current database connection.
// This must be called after switching databases (e.g., when toggling dev mode).
func (s *Server) updateServicesDB() {
	db := s.dbManager.Conn()
	s.downloaderService.SetDB(db)
	s.notificationService.SetDB(db)
	s.indexerService.SetDB(db)
	s.grabService.SetDB(db)
	s.rateLimiter.SetDB(db)
	s.rootFolderService.SetDB(db)
	s.movieService.SetDB(db)
	s.tvService.SetDB(db)
	s.qualityService.SetDB(db)
	s.libraryManagerService.SetDB(db)
	s.historyService.SetDB(db)
	s.slotsService.SetDB(db)
	s.autosearchService.SetDB(db)
	s.rssSyncService.SetDB(sqlc.New(db))
	s.rssSyncSettingsHandler.SetDB(sqlc.New(db))
	s.importService.SetDB(db)
	s.importSettingsHandlers.SetDB(db)
	s.defaultsService.SetDB(db)
	s.preferencesService.SetDB(db)
	s.calendarService.SetDB(db)
	s.networkLogoStore.SetDB(db)
	s.availabilityService.SetDB(db)
	s.missingService.SetDB(db)
	s.statusService.SetDB(db)
	s.prowlarrService.SetDB(db)

	// Update portal services
	queries := sqlc.New(db)
	s.portalUsersService.SetDB(queries)
	s.portalInvitationsService.SetDB(queries)
	s.portalQuotaService.SetDB(queries)
	s.portalNotificationsService.SetDB(queries)
	s.portalAutoApproveService.SetDB(queries)
	s.portalRequestsService.SetDB(queries)
	s.portalAuthService.SetDB(queries)
	if s.portalPasskeyService != nil {
		s.portalPasskeyService.SetDB(queries)
	}
	s.portalMediaProvisioner.SetDB(db)
	s.portalRequestSearcher.SetDB(db)
	s.portalWatchersService.SetDB(db)
	s.portalStatusTracker.SetDB(db)
	s.portalLibraryChecker.SetDB(db)
	s.adminSettingsHandlers.SetDB(queries)

	s.logger.Info().Msg("Updated all services with new database connection")
}

// switchDownloadClient creates or removes the mock download client based on dev mode.
func (s *Server) switchDownloadClient(devMode bool) {
	ctx := context.Background()

	if devMode {
		// Check if mock client already exists
		clients, err := s.downloaderService.List(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to list download clients for dev mode")
			return
		}

		// Look for existing mock client
		for _, c := range clients {
			if c.Type == "mock" {
				s.logger.Info().Int64("id", c.ID).Msg("Mock download client already exists")
				return
			}
		}

		// Create mock download client
		_, err = s.downloaderService.Create(ctx, &downloader.CreateClientInput{
			Name:     "Mock Download Client",
			Type:     "mock",
			Host:     "localhost",
			Port:     9999,
			Enabled:  true,
			Priority: 1,
		})
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to create mock download client")
			return
		}
		s.logger.Info().Msg("Created mock download client for dev mode")
	} else {
		// Clear mock downloads when disabling dev mode
		downloadermock.GetInstance().Clear()
		s.logger.Info().Msg("Cleared mock downloads")
	}
}

// switchNotification creates mock notification based on dev mode.
func (s *Server) switchNotification(devMode bool) {
	ctx := context.Background()

	if devMode {
		// Check if mock notification already exists
		notifications, err := s.notificationService.List(ctx)
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to list notifications for dev mode")
			return
		}

		// Look for existing mock notification
		for i := range notifications {
			n := &notifications[i]
			if n.Type == notification.NotifierMock {
				s.logger.Info().Int64("id", n.ID).Msg("Mock notification already exists")
				return
			}
		}

		// Create mock notification (subscribed to all events)
		_, err = s.notificationService.Create(ctx, &notification.CreateInput{
			Name:             "Mock Notification",
			Type:             notification.NotifierMock,
			Enabled:          true,
			OnGrab:           true,
			OnImport:         true,
			OnUpgrade:        true,
			OnMovieAdded:     true,
			OnMovieDeleted:   true,
			OnSeriesAdded:    true,
			OnSeriesDeleted:  true,
			OnHealthIssue:    true,
			OnHealthRestored: true,
			OnAppUpdate:      true,
		})
		if err != nil {
			s.logger.Error().Err(err).Msg("Failed to create mock notification")
			return
		}
		s.logger.Info().Msg("Created mock notification for dev mode")
	} else {
		s.logger.Info().Msg("Dev mode disabled - mock notification will remain until manually deleted")
	}
}

// switchRootFolders creates mock root folders based on dev mode.
func (s *Server) switchRootFolders(devMode bool) {
	if !devMode {
		s.logger.Info().Msg("Dev mode disabled - mock root folders will remain until manually deleted")
		return
	}

	ctx := context.Background()
	fsmock.ResetInstance()

	folders, err := s.rootFolderService.List(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list root folders for dev mode")
		return
	}

	hasMovieRoot, hasTVRoot := s.checkExistingMockFolders(folders)
	if !hasMovieRoot {
		s.createMockMovieFolder(ctx)
	}
	if !hasTVRoot {
		s.createMockTVFolder(ctx)
	}
}

func (s *Server) checkExistingMockFolders(folders []*rootfolder.RootFolder) (hasMovieRoot, hasTVRoot bool) {
	for _, f := range folders {
		if f.Path == fsmock.MockMoviesPath {
			hasMovieRoot = true
		}
		if f.Path == fsmock.MockTVPath {
			hasTVRoot = true
		}
	}
	return
}

func (s *Server) createMockMovieFolder(ctx context.Context) {
	_, err := s.rootFolderService.Create(ctx, rootfolder.CreateRootFolderInput{
		Path:      fsmock.MockMoviesPath,
		Name:      "Mock Movies",
		MediaType: mediaTypeMovie,
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create mock movies root folder")
	} else {
		s.logger.Info().Str("path", fsmock.MockMoviesPath).Msg("Created mock movies root folder")
	}
}

func (s *Server) createMockTVFolder(ctx context.Context) {
	_, err := s.rootFolderService.Create(ctx, rootfolder.CreateRootFolderInput{
		Path:      fsmock.MockTVPath,
		Name:      "Mock TV",
		MediaType: "tv",
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to create mock TV root folder")
	} else {
		s.logger.Info().Str("path", fsmock.MockTVPath).Msg("Created mock TV root folder")
	}
}

// copyJWTSecretToDevDB copies the JWT secret from production to dev database.
// This ensures tokens issued in production mode remain valid in dev mode.
func (s *Server) copyJWTSecretToDevDB() {
	ctx := context.Background()

	// Get JWT secret from production database
	prodQueries := sqlc.New(s.dbManager.ProdConn())
	setting, err := prodQueries.GetSetting(ctx, "portal_jwt_secret")
	if err != nil {
		s.logger.Debug().Err(err).Msg("No JWT secret in production database to copy")
		return
	}

	if setting.Value == "" {
		s.logger.Debug().Msg("Production JWT secret is empty, nothing to copy")
		return
	}

	// Copy to dev database
	devQueries := sqlc.New(s.dbManager.Conn())
	_, err = devQueries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   "portal_jwt_secret",
		Value: setting.Value,
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to copy JWT secret to dev database")
		return
	}

	s.logger.Info().Msg("Copied JWT secret from production to dev database")
}

// copySettingsToDevDB copies application settings from production to dev database.
func (s *Server) copySettingsToDevDB() {
	ctx := context.Background()

	prodQueries := sqlc.New(s.dbManager.ProdConn())
	devQueries := sqlc.New(s.dbManager.Conn())

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
			s.logger.Error().Err(err).Str("key", key).Msg("Failed to copy setting to dev database")
			continue
		}
		copied++
	}

	if copied > 0 {
		s.logger.Info().Int("count", copied).Msg("Copied settings to dev database")
	}
}

// copyPortalUsersToDevDB copies portal users from production to dev database.
// This preserves user IDs so that JWTs issued against prod DB work in dev mode.
// profileIDMapping maps production quality profile IDs to dev database IDs.
func (s *Server) copyPortalUsersToDevDB(profileIDMapping map[int64]int64) {
	ctx := context.Background()

	// Get users from production database
	prodQueries := sqlc.New(s.dbManager.ProdConn())
	prodUsers, err := prodQueries.ListPortalUsers(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list production portal users")
		return
	}

	if len(prodUsers) == 0 {
		s.logger.Debug().Msg("No portal users in production database to copy")
		return
	}

	// Copy each user to dev database (skip if already exists)
	devQueries := sqlc.New(s.dbManager.Conn())
	copied := 0
	for _, user := range prodUsers {
		// Check if user already exists in dev DB
		_, err := devQueries.GetPortalUser(ctx, user.ID)
		if err == nil {
			continue // User already exists
		}

		// Map quality profile ID using the provided mapping
		var qualityProfileID sql.NullInt64
		if user.QualityProfileID.Valid {
			if newID, ok := profileIDMapping[user.QualityProfileID.Int64]; ok {
				qualityProfileID = sql.NullInt64{Int64: newID, Valid: true}
			}
		}

		_, err = devQueries.CreatePortalUserWithID(ctx, sqlc.CreatePortalUserWithIDParams{
			ID:               user.ID,
			Username:         user.Username,
			PasswordHash:     user.PasswordHash,
			DisplayName:      user.DisplayName,
			QualityProfileID: qualityProfileID,
			AutoApprove:      user.AutoApprove,
			Enabled:          user.Enabled,
			IsAdmin:          user.IsAdmin,
		})
		if err != nil {
			s.logger.Error().Err(err).Str("username", user.Username).Msg("Failed to copy portal user")
			continue
		}
		copied++
	}

	if copied > 0 {
		s.logger.Info().Int("count", copied).Msg("Copied portal users to dev database")
	}
}

// copyPortalUserNotificationsToDevDB copies portal user notification channels from production to dev database.
func (s *Server) copyPortalUserNotificationsToDevDB() {
	ctx := context.Background()

	prodQueries := sqlc.New(s.dbManager.ProdConn())
	devQueries := sqlc.New(s.dbManager.Conn())

	prodUsers, err := prodQueries.ListPortalUsers(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list production portal users for notification copy")
		return
	}

	copied := 0
	for _, user := range prodUsers {
		notifs, err := prodQueries.ListUserNotifications(ctx, user.ID)
		if err != nil {
			s.logger.Error().Err(err).Int64("user_id", user.ID).Msg("Failed to list user notifications")
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
				s.logger.Error().Err(err).Str("name", n.Name).Msg("Failed to copy user notification")
				continue
			}
			copied++
		}
	}

	if copied > 0 {
		s.logger.Info().Int("count", copied).Msg("Copied portal user notifications to dev database")
	}
}

// copyQualityProfilesToDevDB copies quality profiles from production to dev database.
// Returns a mapping of production profile IDs to dev profile IDs.
func (s *Server) copyQualityProfilesToDevDB() map[int64]int64 {
	ctx := context.Background()
	idMapping := make(map[int64]int64)

	// Check if dev database already has profiles
	devProfiles, err := s.qualityService.List(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list dev quality profiles")
		return idMapping
	}
	if len(devProfiles) > 0 {
		s.logger.Info().Int("count", len(devProfiles)).Msg("Dev database already has quality profiles")
		return idMapping
	}

	// Get profiles from production database
	prodQueries := sqlc.New(s.dbManager.ProdConn())
	prodRows, err := prodQueries.ListQualityProfiles(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list production quality profiles")
		return idMapping
	}

	if len(prodRows) == 0 {
		s.logger.Warn().Msg("No quality profiles in production database to copy")
		// Create default profiles in dev database
		if err := s.qualityService.EnsureDefaults(ctx); err != nil {
			s.logger.Error().Err(err).Msg("Failed to create default quality profiles")
		}
		return idMapping
	}

	// Copy each profile to dev database and track ID mapping
	devQueries := sqlc.New(s.dbManager.Conn())
	for _, row := range prodRows {
		newProfile, err := devQueries.CreateQualityProfile(ctx, sqlc.CreateQualityProfileParams{
			Name:                 row.Name,
			Cutoff:               row.Cutoff,
			Items:                row.Items,
			HdrSettings:          row.HdrSettings,
			VideoCodecSettings:   row.VideoCodecSettings,
			AudioCodecSettings:   row.AudioCodecSettings,
			AudioChannelSettings: row.AudioChannelSettings,
			UpgradesEnabled:      row.UpgradesEnabled,
			AllowAutoApprove:     row.AllowAutoApprove,
		})
		if err != nil {
			s.logger.Error().Err(err).Str("name", row.Name).Msg("Failed to copy quality profile")
			continue
		}
		idMapping[row.ID] = newProfile.ID
		s.logger.Debug().Str("name", row.Name).Int64("prodID", row.ID).Int64("devID", newProfile.ID).Msg("Copied quality profile to dev database")
	}

	s.logger.Info().Int("count", len(prodRows)).Msg("Copied quality profiles to dev database")
	return idMapping
}

// setupDevModeSlots configures version slots for developer mode testing.
// Assigns quality profiles to slots and enables them so the dry run feature works.
func (s *Server) setupDevModeSlots() {
	ctx := context.Background()

	// Get all slots
	slotList, err := s.slotsService.List(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list slots for dev mode setup")
		return
	}

	// Get all quality profiles
	profiles, err := s.qualityService.List(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list quality profiles for dev mode setup")
		return
	}

	if len(profiles) == 0 {
		s.logger.Warn().Msg("No quality profiles available for dev mode slot setup")
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

		_, err := s.slotsService.Update(ctx, slot.ID, input)
		if err != nil {
			s.logger.Error().Err(err).Int64("slotId", slot.ID).Msg("Failed to update slot for dev mode")
			continue
		}
		s.logger.Debug().Int64("slotId", slot.ID).Int64("profileId", profileID).Msg("Configured slot for dev mode")
	}

	s.logger.Info().Int("count", len(slotList)).Msg("Configured slots for dev mode")
}

// populateMockMedia creates mock movies and series in the dev database.
func (s *Server) populateMockMedia() {
	ctx := context.Background()

	// Get the mock root folders
	folders, err := s.rootFolderService.List(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to list root folders for mock media")
		return
	}

	var movieRootID, tvRootID int64
	for _, f := range folders {
		if f.Path == fsmock.MockMoviesPath {
			movieRootID = f.ID
		}
		if f.Path == fsmock.MockTVPath {
			tvRootID = f.ID
		}
	}

	if movieRootID == 0 || tvRootID == 0 {
		s.logger.Warn().Msg("Mock root folders not found, skipping media population")
		return
	}

	// Get a default quality profile
	profiles, err := s.qualityService.List(ctx)
	if err != nil || len(profiles) == 0 {
		s.logger.Warn().Msg("No quality profiles available for mock media")
		return
	}
	defaultProfileID := profiles[0].ID

	// Check if we already have mock movies
	existingMovies, _ := s.movieService.List(ctx, movies.ListMoviesOptions{})
	if len(existingMovies) > 0 {
		s.logger.Info().Int("count", len(existingMovies)).Msg("Dev database already has movies")
		return
	}

	s.populateMockMovies(ctx, movieRootID, defaultProfileID)
	s.populateMockSeries(ctx, tvRootID, defaultProfileID)
}

func (s *Server) populateMockMovies(ctx context.Context, rootFolderID, qualityProfileID int64) {
	// Mock movies with TMDB IDs - metadata will be fetched from mock provider
	mockMovieIDs := []struct {
		tmdbID   int
		hasFiles bool
	}{
		{603, true},     // The Matrix
		{27205, true},   // Inception
		{438631, true},  // Dune
		{680, true},     // Pulp Fiction
		{550, true},     // Fight Club
		{693134, false}, // Dune: Part Two
		{872585, false}, // Oppenheimer
		{346698, false}, // Barbie
	}

	for _, m := range mockMovieIDs {
		// Fetch full metadata from mock provider
		movieMeta, err := s.metadataService.GetMovie(ctx, m.tmdbID)
		if err != nil {
			s.logger.Error().Err(err).Int("tmdbID", m.tmdbID).Msg("Failed to fetch mock movie metadata")
			continue
		}

		path := fsmock.MockMoviesPath + "/" + movieMeta.Title + " (" + itoa(movieMeta.Year) + ")"

		input := movies.CreateMovieInput{
			Title:            movieMeta.Title,
			Year:             movieMeta.Year,
			TmdbID:           movieMeta.ID, // ID is the TMDB ID
			ImdbID:           movieMeta.ImdbID,
			Overview:         movieMeta.Overview,
			Runtime:          movieMeta.Runtime,
			RootFolderID:     rootFolderID,
			QualityProfileID: qualityProfileID,
			Path:             path,
			Monitored:        true,
		}

		movie, err := s.movieService.Create(ctx, &input)
		if err != nil {
			s.logger.Error().Err(err).Str("title", movieMeta.Title).Msg("Failed to create mock movie")
			continue
		}

		// Download artwork from mock metadata URLs
		go func(meta *metadata.MovieResult) {
			if err := s.artworkDownloader.DownloadMovieArtwork(ctx, meta); err != nil {
				s.logger.Debug().Err(err).Str("title", meta.Title).Msg("Failed to download movie artwork")
			}
		}(movieMeta)

		// If the movie has files in the VFS, create file entries
		if m.hasFiles {
			s.createMockMovieFiles(ctx, movie.ID, path)
		}

		s.logger.Debug().Str("title", movieMeta.Title).Bool("hasFiles", m.hasFiles).Msg("Created mock movie")
	}

	s.logger.Info().Int("count", len(mockMovieIDs)).Msg("Populated mock movies")
}

func (s *Server) createMockMovieFiles(ctx context.Context, movieID int64, moviePath string) {
	vfs := fsmock.GetInstance()
	files, err := vfs.ListDirectory(moviePath)
	if err != nil {
		return
	}

	for _, f := range files {
		if f.Type != fsmock.FileTypeVideo {
			continue
		}

		qualityName := parseQualityFromFilename(f.Name)
		input := movies.CreateMovieFileInput{
			Path:    f.Path,
			Size:    f.Size,
			Quality: qualityName,
		}
		if q, ok := quality.GetQualityByName(qualityName); ok {
			qid := int64(q.ID)
			input.QualityID = &qid
		}

		_, err := s.movieService.AddFile(ctx, movieID, &input)
		if err != nil {
			s.logger.Debug().Err(err).Str("path", f.Path).Msg("Failed to create movie file")
		}
	}
}

func (s *Server) populateMockSeries(ctx context.Context, rootFolderID, qualityProfileID int64) {
	mockSeriesIDs := []struct {
		tvdbID   int
		hasFiles bool
	}{
		{81189, true},   // Breaking Bad
		{121361, true},  // Game of Thrones
		{305288, true},  // Stranger Things
		{361753, true},  // The Mandalorian
		{355567, false}, // The Boys
	}

	for _, s2 := range mockSeriesIDs {
		s.createOneMockSeries(ctx, s2.tvdbID, s2.hasFiles, rootFolderID, qualityProfileID)
	}

	s.logger.Info().Int("count", len(mockSeriesIDs)).Msg("Populated mock series")
}

func (s *Server) createOneMockSeries(ctx context.Context, tvdbID int, hasFiles bool, rootFolderID, qualityProfileID int64) {
	seriesMeta, err := s.metadataService.GetSeriesByTVDB(ctx, tvdbID)
	if err != nil {
		s.logger.Error().Err(err).Int("tvdbID", tvdbID).Msg("Failed to fetch mock series metadata")
		return
	}

	seasonsMeta, err := s.metadataService.GetSeriesSeasons(ctx, seriesMeta.TmdbID, seriesMeta.TvdbID)
	if err != nil {
		s.logger.Warn().Err(err).Str("title", seriesMeta.Title).Msg("Failed to fetch seasons, using empty")
		seasonsMeta = nil
	}

	seasons := s.convertSeasonsMetadata(seasonsMeta)
	path := fsmock.MockTVPath + "/" + seriesMeta.Title

	input := tv.CreateSeriesInput{
		Title:            seriesMeta.Title,
		Year:             seriesMeta.Year,
		TvdbID:           seriesMeta.TvdbID,
		TmdbID:           seriesMeta.TmdbID,
		ImdbID:           seriesMeta.ImdbID,
		Overview:         seriesMeta.Overview,
		Runtime:          seriesMeta.Runtime,
		Network:          seriesMeta.Network,
		NetworkLogoURL:   seriesMeta.NetworkLogoURL,
		RootFolderID:     rootFolderID,
		QualityProfileID: qualityProfileID,
		Path:             path,
		Monitored:        true,
		SeasonFolder:     true,
		Seasons:          seasons,
	}

	series, err := s.tvService.CreateSeries(ctx, &input)
	if err != nil {
		s.logger.Error().Err(err).Str("title", seriesMeta.Title).Msg("Failed to create mock series")
		return
	}

	go func(meta *metadata.SeriesResult) {
		if err := s.artworkDownloader.DownloadSeriesArtwork(ctx, meta); err != nil {
			s.logger.Debug().Err(err).Str("title", meta.Title).Msg("Failed to download series artwork")
		}
	}(seriesMeta)

	if hasFiles {
		s.createMockEpisodeFiles(ctx, series.ID, path, qualityProfileID)
	}

	s.logger.Debug().Str("title", seriesMeta.Title).Bool("hasFiles", hasFiles).Int("seasons", len(seasons)).Msg("Created mock series")
}

func (s *Server) convertSeasonsMetadata(seasonsMeta []metadata.SeasonResult) []tv.SeasonInput {
	var seasons []tv.SeasonInput
	for _, sm := range seasonsMeta {
		episodes := s.convertEpisodesMetadata(sm.Episodes)
		seasons = append(seasons, tv.SeasonInput{
			SeasonNumber: sm.SeasonNumber,
			Monitored:    true,
			Episodes:     episodes,
		})
	}
	return seasons
}

func (s *Server) convertEpisodesMetadata(episodesMeta []metadata.EpisodeResult) []tv.EpisodeInput {
	var episodes []tv.EpisodeInput
	for _, ep := range episodesMeta {
		var airDate *time.Time
		if ep.AirDate != "" {
			if t, err := time.Parse("2006-01-02", ep.AirDate); err == nil {
				airDate = &t
			}
		}
		episodes = append(episodes, tv.EpisodeInput{
			EpisodeNumber: ep.EpisodeNumber,
			Title:         ep.Title,
			Overview:      ep.Overview,
			AirDate:       airDate,
			Monitored:     true,
		})
	}
	return episodes
}

func (s *Server) createMockEpisodeFiles(ctx context.Context, seriesID int64, seriesPath string, qualityProfileID int64) {
	vfs := fsmock.GetInstance()
	seasonDirs, err := vfs.ListDirectory(seriesPath)
	if err != nil {
		return
	}

	episodes, err := s.tvService.ListEpisodes(ctx, seriesID, nil)
	if err != nil {
		return
	}

	var profile *quality.Profile
	if s.qualityService != nil {
		profile, _ = s.qualityService.Get(ctx, qualityProfileID)
	}

	queries := sqlc.New(s.dbManager.Conn())
	episodeMap := s.buildEpisodeMap(episodes)

	for _, seasonDir := range seasonDirs {
		s.processSeasonDirectory(ctx, queries, episodeMap, profile, vfs, seasonDir)
	}
}

func (s *Server) buildEpisodeMap(episodes []tv.Episode) map[string]int64 {
	episodeMap := make(map[string]int64)
	for _, ep := range episodes {
		key := itoa(ep.SeasonNumber) + ":" + itoa(ep.EpisodeNumber)
		episodeMap[key] = ep.ID
	}
	return episodeMap
}

func (s *Server) processSeasonDirectory(ctx context.Context, queries *sqlc.Queries, episodeMap map[string]int64, profile *quality.Profile, vfs *fsmock.VirtualFS, seasonDir *fsmock.VirtualFile) {
	if seasonDir.Type != fsmock.FileTypeDirectory {
		return
	}

	seasonNum := parseSeasonNumber(seasonDir.Name)
	if seasonNum == 0 {
		return
	}

	episodeFiles, err := vfs.ListDirectory(seasonDir.Path)
	if err != nil {
		return
	}

	for _, f := range episodeFiles {
		s.processEpisodeFile(ctx, queries, episodeMap, profile, f, seasonNum)
	}
}

func (s *Server) processEpisodeFile(ctx context.Context, queries *sqlc.Queries, episodeMap map[string]int64, profile *quality.Profile, f *fsmock.VirtualFile, seasonNum int) {
	if f.Type != fsmock.FileTypeVideo {
		return
	}

	epNum := parseEpisodeNumber(f.Name)
	if epNum == 0 {
		return
	}

	key := itoa(seasonNum) + ":" + itoa(epNum)
	episodeID, ok := episodeMap[key]
	if !ok {
		return
	}

	qualityName := parseQualityFromFilename(f.Name)
	qualityID := sql.NullInt64{}
	if q, ok := quality.GetQualityByName(qualityName); ok {
		qualityID = sql.NullInt64{Int64: int64(q.ID), Valid: true}
	}

	_, _ = queries.CreateEpisodeFile(ctx, sqlc.CreateEpisodeFileParams{
		EpisodeID: episodeID,
		Path:      f.Path,
		Size:      f.Size,
		Quality:   sql.NullString{String: qualityName, Valid: qualityName != ""},
		QualityID: qualityID,
	})

	episodeStatus := "available"
	if qualityID.Valid && profile != nil {
		episodeStatus = profile.StatusForQuality(int(qualityID.Int64))
	}
	_ = queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
		ID:     episodeID,
		Status: episodeStatus,
	})
}

func parseQualityFromFilename(filename string) string {
	filename = strings.ToLower(filename)
	if strings.Contains(filename, "2160p") {
		if strings.Contains(filename, "remux") {
			return "Remux-2160p"
		}
		return "Bluray-2160p"
	}
	if strings.Contains(filename, "1080p") {
		if strings.Contains(filename, "web") {
			return "WEBDL-1080p"
		}
		return "Bluray-1080p"
	}
	if strings.Contains(filename, "720p") {
		if strings.Contains(filename, "web") {
			return "WEBDL-720p"
		}
		return "Bluray-720p"
	}
	return "Unknown"
}

func parseSeasonNumber(name string) int {
	// Handle "Season 01", "Season 1", "S01", etc.
	name = strings.ToLower(name)
	name = strings.TrimPrefix(name, "season ")
	name = strings.TrimPrefix(name, "s")
	name = strings.TrimSpace(name)
	num, _ := strconv.Atoi(name)
	return num
}

func parseEpisodeNumber(filename string) int {
	filename = strings.ToLower(filename)
	for i := 1; i < len(filename)-1; i++ {
		if !isEpisodeMarker(filename, i) {
			continue
		}
		return extractEpisodeDigits(filename, i)
	}
	return 0
}

func isEpisodeMarker(filename string, i int) bool {
	if filename[i] != 'e' {
		return false
	}
	if filename[i-1] < '0' || filename[i-1] > '9' {
		return false
	}
	if i+1 >= len(filename) {
		return false
	}
	return filename[i+1] >= '0' && filename[i+1] <= '9'
}

func extractEpisodeDigits(filename string, startIdx int) int {
	numStr := ""
	for j := startIdx + 1; j < len(filename) && j < startIdx+4; j++ {
		if filename[j] >= '0' && filename[j] <= '9' {
			numStr += string(filename[j])
		} else {
			break
		}
	}
	num, _ := strconv.Atoi(numStr)
	return num
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

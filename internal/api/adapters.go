package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/arrimport"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/downloader"
	"github.com/slipstream/slipstream/internal/history"
	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	indexerTypes "github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/librarymanager"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/rootfolder"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/notification"
	"github.com/slipstream/slipstream/internal/portal/autoapprove"
	"github.com/slipstream/slipstream/internal/portal/requests"
	"github.com/slipstream/slipstream/internal/portal/users"
	"github.com/slipstream/slipstream/internal/websocket"
)

type portalAutoApproveAdapter struct {
	svc *autoapprove.Service
}

func (a *portalAutoApproveAdapter) ProcessAutoApprove(req *requests.Request, user *users.User) error {
	ctx := context.Background()
	_, err := a.svc.ProcessAutoApprove(ctx, req)
	return err
}

// portalRequestSearcherAdapter adapts requests.RequestSearcher to admin.RequestSearcher interface.
type portalRequestSearcherAdapter struct {
	searcher *requests.RequestSearcher
}

func (a *portalRequestSearcherAdapter) SearchForRequestAsync(requestID int64) {
	a.searcher.SearchForRequestAsync(context.Background(), requestID)
}

// portalMediaProvisionerAdapter implements requests.MediaProvisioner to find or create media in library.
type portalMediaProvisionerAdapter struct {
	queries        *sqlc.Queries
	movieService   *movies.Service
	tvService      *tv.Service
	libraryManager *librarymanager.Service
	logger         *zerolog.Logger
}

func (a *portalMediaProvisionerAdapter) EnsureMovieInLibrary(ctx context.Context, input *requests.MediaProvisionInput) (int64, error) {
	// First, try to find existing movie by TMDB ID
	existing, err := a.movieService.GetByTmdbID(ctx, int(input.TmdbID))
	if err == nil && existing != nil {
		a.logger.Debug().Int64("tmdbID", input.TmdbID).Int64("movieID", existing.ID).Msg("found existing movie in library")
		return existing.ID, nil
	}

	// Movie not in library - get default settings and create it
	rootFolderID, qualityProfileID, err := a.getDefaultSettings(ctx, mediaTypeMovie)
	if err != nil {
		return 0, fmt.Errorf("failed to get default settings: %w", err)
	}

	// Use user's quality profile if provided
	if input.QualityProfileID != nil {
		qualityProfileID = *input.QualityProfileID
		a.logger.Debug().Int64("qualityProfileID", qualityProfileID).Msg("using user's assigned quality profile for movie")
	}

	movie, err := a.movieService.Create(ctx, &movies.CreateMovieInput{
		Title:            input.Title,
		Year:             input.Year,
		TmdbID:           int(input.TmdbID),
		RootFolderID:     rootFolderID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
		AddedBy:          input.AddedBy,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create movie: %w", err)
	}

	a.logger.Info().Int64("tmdbID", input.TmdbID).Int64("movieID", movie.ID).Str("title", input.Title).Msg("created movie in library from request")
	return movie.ID, nil
}

func (a *portalMediaProvisionerAdapter) EnsureSeriesInLibrary(ctx context.Context, input *requests.MediaProvisionInput) (int64, error) {
	// First, try to find existing series by TVDB ID
	existing, err := a.tvService.GetSeriesByTvdbID(ctx, int(input.TvdbID))
	if err == nil && existing != nil {
		a.logger.Debug().Int64("tvdbID", input.TvdbID).Int64("seriesID", existing.ID).Msg("found existing series in library")
		return existing.ID, nil
	}

	// Series not in library - get default settings and create it
	rootFolderID, qualityProfileID, err := a.getDefaultSettings(ctx, "tv")
	if err != nil {
		return 0, fmt.Errorf("failed to get default settings: %w", err)
	}

	// Use user's quality profile if provided
	if input.QualityProfileID != nil {
		qualityProfileID = *input.QualityProfileID
		a.logger.Debug().Int64("qualityProfileID", qualityProfileID).Msg("using user's assigned quality profile for series")
	}

	series, err := a.tvService.CreateSeries(ctx, &tv.CreateSeriesInput{
		Title:            input.Title,
		TvdbID:           int(input.TvdbID),
		RootFolderID:     rootFolderID,
		QualityProfileID: qualityProfileID,
		Monitored:        true,
		AddedBy:          input.AddedBy,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create series: %w", err)
	}

	a.logger.Info().Int64("tvdbID", input.TvdbID).Int64("seriesID", series.ID).Str("title", input.Title).Msg("created series in library from request")

	// Fetch metadata including seasons and episodes
	if a.libraryManager != nil {
		if _, err := a.libraryManager.RefreshSeriesMetadata(ctx, series.ID); err != nil {
			a.logger.Warn().Err(err).Int64("seriesID", series.ID).Msg("failed to refresh series metadata, series created without episodes")
		} else {
			a.logger.Info().Int64("seriesID", series.ID).Msg("fetched series metadata with seasons and episodes")
		}
	}

	// Apply requested seasons monitoring if specific seasons were requested
	if len(input.RequestedSeasons) > 0 {
		if err := a.applyRequestedSeasonsMonitoring(ctx, series.ID, input.RequestedSeasons); err != nil {
			a.logger.Warn().Err(err).Int64("seriesID", series.ID).Msg("failed to apply requested seasons monitoring")
		}
	}

	return series.ID, nil
}

// applyRequestedSeasonsMonitoring unmonitors all seasons except the requested ones.
func (a *portalMediaProvisionerAdapter) applyRequestedSeasonsMonitoring(ctx context.Context, seriesID int64, requestedSeasons []int64) error {
	// Get all seasons for the series
	seasons, err := a.tvService.ListSeasons(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("failed to get seasons: %w", err)
	}

	// Build a set of requested season numbers for quick lookup
	requestedSet := make(map[int64]bool)
	for _, sn := range requestedSeasons {
		requestedSet[sn] = true
	}

	// Update monitoring for each season
	for i := range seasons {
		season := &seasons[i]
		shouldMonitor := requestedSet[int64(season.SeasonNumber)]

		if season.Monitored != shouldMonitor {
			if _, err := a.tvService.UpdateSeasonMonitored(ctx, seriesID, season.SeasonNumber, shouldMonitor); err != nil {
				a.logger.Warn().
					Err(err).
					Int64("seriesID", seriesID).
					Int("seasonNumber", season.SeasonNumber).
					Bool("monitored", shouldMonitor).
					Msg("failed to update season monitoring")
			} else {
				a.logger.Debug().
					Int64("seriesID", seriesID).
					Int("seasonNumber", season.SeasonNumber).
					Bool("monitored", shouldMonitor).
					Msg("updated season monitoring")
			}
		}
	}

	a.logger.Info().
		Int64("seriesID", seriesID).
		Interface("requestedSeasons", requestedSeasons).
		Msg("applied requested seasons monitoring")

	return nil
}

func (a *portalMediaProvisionerAdapter) getDefaultSettings(ctx context.Context, mediaType string) (rootFolderID, qualityProfileID int64, err error) {
	rootFolderID = a.resolveRootFolderID(ctx, mediaType)
	if rootFolderID == 0 {
		return 0, 0, fmt.Errorf("no %s root folder configured - please configure a root folder for %s content", mediaType, mediaType)
	}

	profiles, err := a.queries.ListQualityProfiles(ctx)
	if err != nil || len(profiles) == 0 {
		return 0, 0, errors.New("no quality profile configured")
	}
	qualityProfileID = profiles[0].ID

	return rootFolderID, qualityProfileID, nil
}

func (a *portalMediaProvisionerAdapter) resolveRootFolderID(ctx context.Context, mediaType string) int64 {
	if id := a.getMediaTypeSpecificRootFolder(ctx, mediaType); id != 0 {
		return id
	}
	if id := a.getGenericRootFolder(ctx, mediaType); id != 0 {
		return id
	}
	return a.getFirstAvailableRootFolder(ctx, mediaType)
}

func (a *portalMediaProvisionerAdapter) getMediaTypeSpecificRootFolder(ctx context.Context, mediaType string) int64 {
	settingKey := "requests_default_root_folder_id"
	switch mediaType {
	case "tv":
		settingKey = "requests_default_tv_root_folder_id"
	case mediaTypeMovie:
		settingKey = "requests_default_movie_root_folder_id"
	}

	setting, err := a.queries.GetSetting(ctx, settingKey)
	if err != nil || setting.Value == "" {
		return 0
	}
	v, parseErr := strconv.ParseInt(setting.Value, 10, 64)
	if parseErr != nil {
		return 0
	}
	return v
}

func (a *portalMediaProvisionerAdapter) getGenericRootFolder(ctx context.Context, mediaType string) int64 {
	setting, err := a.queries.GetSetting(ctx, "requests_default_root_folder_id")
	if err != nil || setting.Value == "" {
		return 0
	}
	v, parseErr := strconv.ParseInt(setting.Value, 10, 64)
	if parseErr != nil {
		return 0
	}
	rf, rfErr := a.queries.GetRootFolder(ctx, v)
	if rfErr != nil || rf.MediaType != mediaType {
		return 0
	}
	return v
}

func (a *portalMediaProvisionerAdapter) getFirstAvailableRootFolder(ctx context.Context, mediaType string) int64 {
	rootFolders, err := a.queries.ListRootFoldersByMediaType(ctx, mediaType)
	if err != nil || len(rootFolders) == 0 {
		return 0
	}
	return rootFolders[0].ID
}

func (a *portalMediaProvisionerAdapter) SetDB(db *sql.DB) {
	a.queries = sqlc.New(db)
}

// portalUserQualityProfileAdapter implements requests.UserQualityProfileGetter
type portalUserQualityProfileAdapter struct {
	usersSvc *users.Service
}

func (a *portalUserQualityProfileAdapter) GetQualityProfileID(ctx context.Context, userID int64) (*int64, error) {
	user, err := a.usersSvc.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user.QualityProfileID, nil
}

// portalQueueGetterAdapter adapts downloader.Service to requests.QueueGetter interface.
type portalQueueGetterAdapter struct {
	downloaderSvc *downloader.Service
}

func (a *portalQueueGetterAdapter) GetQueue(ctx context.Context) ([]requests.QueueItem, error) {
	resp, err := a.downloaderSvc.GetQueue(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]requests.QueueItem, len(resp.Items))
	for i := range resp.Items {
		item := &resp.Items[i]
		result[i] = requests.QueueItem{
			ID:             item.ID,
			ClientID:       item.ClientID,
			ClientName:     item.ClientName,
			Title:          item.Title,
			MediaType:      item.MediaType,
			Status:         item.Status,
			Progress:       item.Progress,
			Size:           item.Size,
			DownloadedSize: item.DownloadedSize,
			DownloadSpeed:  item.DownloadSpeed,
			ETA:            item.ETA,
			Season:         item.Season,
			Episode:        item.Episode,
			MovieID:        item.MovieID,
			SeriesID:       item.SeriesID,
			SeasonNumber:   item.SeasonNumber,
			IsSeasonPack:   item.IsSeasonPack,
		}
	}
	return result, nil
}

// portalMediaLookupAdapter adapts sqlc.Queries to requests.MediaLookup interface.
type portalMediaLookupAdapter struct {
	queries *sqlc.Queries
}

func (a *portalMediaLookupAdapter) GetMovieTmdbID(ctx context.Context, movieID int64) (*int64, error) {
	movie, err := a.queries.GetMovie(ctx, movieID)
	if err != nil {
		return nil, err
	}
	if !movie.TmdbID.Valid {
		return nil, sql.ErrNoRows
	}
	return &movie.TmdbID.Int64, nil
}

func (a *portalMediaLookupAdapter) GetSeriesTvdbID(ctx context.Context, seriesID int64) (*int64, error) {
	series, err := a.queries.GetSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	if !series.TvdbID.Valid {
		return nil, sql.ErrNoRows
	}
	return &series.TvdbID.Int64, nil
}

// portalEnabledChecker checks if the external requests portal is enabled.
type portalEnabledChecker struct {
	queries *sqlc.Queries
}

func (c *portalEnabledChecker) IsPortalEnabled(ctx context.Context) bool {
	setting, err := c.queries.GetSetting(ctx, "requests_portal_enabled")
	if err != nil {
		return true // Default to enabled if setting not found
	}
	return setting.Value != "0" && setting.Value != "false"
}

type importHistoryAdapter struct {
	svc *history.Service
}

// Create implements importer.HistoryService.
func (a *importHistoryAdapter) Create(ctx context.Context, input *importer.HistoryInput) error {
	_, err := a.svc.Create(ctx, &history.CreateInput{
		EventType: history.EventType(input.EventType),
		MediaType: history.MediaType(input.MediaType),
		MediaID:   input.MediaID,
		Source:    input.Source,
		Quality:   input.Quality,
		Data:      input.Data,
	})
	return err
}

// statusChangeLoggerAdapter adapts history.Service for status transition logging.
type statusChangeLoggerAdapter struct {
	svc *history.Service
}

func (a *statusChangeLoggerAdapter) LogStatusChanged(ctx context.Context, mediaType string, mediaID int64, from, to, reason string) error {
	return a.svc.LogStatusChanged(ctx, history.MediaType(mediaType), mediaID, history.StatusChangedData{
		From: from, To: to, Reason: reason,
	})
}

// slotFileDeleterAdapter adapts movie and TV services to slots.FileDeleter interface.
// Req 12.2.2: Delete files when disabling a slot with delete action.
type slotFileDeleterAdapter struct {
	movieSvc *movies.Service
	tvSvc    *tv.Service
}

// DeleteFile implements slots.FileDeleter.
func (a *slotFileDeleterAdapter) DeleteFile(ctx context.Context, mediaType string, fileID int64) error {
	switch mediaType {
	case mediaTypeMovie:
		return a.movieSvc.RemoveFile(ctx, fileID)
	case mediaTypeEpisode:
		return a.tvSvc.RemoveEpisodeFile(ctx, fileID)
	default:
		return nil
	}
}

// slotRootFolderAdapter adapts rootfolder.Service to slots.RootFolderProvider.
type slotRootFolderAdapter struct {
	rootFolderSvc *rootfolder.Service
}

// Get implements slots.RootFolderProvider.
func (a *slotRootFolderAdapter) Get(ctx context.Context, id int64) (*slots.RootFolder, error) {
	rf, err := a.rootFolderSvc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &slots.RootFolder{
		ID:        rf.ID,
		Path:      rf.Path,
		Name:      rf.Name,
		MediaType: rf.MediaType,
	}, nil
}

// grabNotificationAdapter adapts notification.Service to grab.NotificationService interface.
type grabNotificationAdapter struct {
	svc    *notification.Service
	movies *movies.Service
	tv     *tv.Service
}

// OnGrab implements grab.NotificationService.
func (a *grabNotificationAdapter) OnGrab(ctx context.Context, release *indexerTypes.ReleaseInfo, clientName string, clientID int64, downloadID string, slotID *int64, slotName string, media *grab.GrabMediaContext) {
	event := a.buildBaseGrabEvent(release, clientName, clientID, downloadID, slotID, slotName)
	a.populateMediaInfo(ctx, &event, media)
	a.svc.Dispatch(ctx, notification.EventGrab, event)
}

func (a *grabNotificationAdapter) buildBaseGrabEvent(release *indexerTypes.ReleaseInfo, clientName string, clientID int64, downloadID string, slotID *int64, slotName string) notification.GrabEvent {
	event := notification.GrabEvent{
		Release: notification.ReleaseInfo{
			ReleaseName: release.Title,
			Quality:     release.Quality,
			Size:        release.Size,
			Indexer:     release.IndexerName,
			Languages:   release.Languages,
		},
		DownloadClient: notification.DownloadClientInfo{
			ID:         clientID,
			Name:       clientName,
			DownloadID: downloadID,
		},
		GrabbedAt: time.Now(),
	}
	if slotID != nil {
		event.Slot = &notification.SlotInfo{
			ID:   *slotID,
			Name: slotName,
		}
	}
	return event
}

func (a *grabNotificationAdapter) populateMediaInfo(ctx context.Context, event *notification.GrabEvent, media *grab.GrabMediaContext) {
	if media == nil {
		return
	}
	switch media.MediaType {
	case mediaTypeMovie:
		a.populateMovieInfo(ctx, event, media)
	case mediaTypeEpisode:
		a.populateEpisodeInfo(ctx, event, media)
	case "season":
		a.populateSeasonInfo(ctx, event, media)
	}
}

func (a *grabNotificationAdapter) populateMovieInfo(ctx context.Context, event *notification.GrabEvent, media *grab.GrabMediaContext) {
	if media.MediaID <= 0 {
		return
	}
	movie, err := a.movies.Get(ctx, media.MediaID)
	if err != nil {
		return
	}
	event.Movie = &notification.MediaInfo{
		ID:    movie.ID,
		Title: movie.Title,
		Year:  movie.Year,
	}
}

func (a *grabNotificationAdapter) populateEpisodeInfo(ctx context.Context, event *notification.GrabEvent, media *grab.GrabMediaContext) {
	if media.MediaID <= 0 {
		return
	}
	ep := &notification.EpisodeInfo{SeasonNumber: media.SeasonNumber}
	if episode, err := a.tv.GetEpisode(ctx, media.MediaID); err == nil {
		ep.EpisodeNumber = episode.EpisodeNumber
		ep.EpisodeTitle = episode.Title
	}
	if media.SeriesID > 0 {
		ep.SeriesID = media.SeriesID
		if series, err := a.tv.GetSeries(ctx, media.SeriesID); err == nil {
			ep.SeriesTitle = series.Title
		}
	}
	event.Episode = ep
}

func (a *grabNotificationAdapter) populateSeasonInfo(ctx context.Context, event *notification.GrabEvent, media *grab.GrabMediaContext) {
	if media.SeriesID <= 0 {
		return
	}
	ep := &notification.EpisodeInfo{
		SeriesID:     media.SeriesID,
		SeasonNumber: media.SeasonNumber,
	}
	if series, err := a.tv.GetSeries(ctx, media.SeriesID); err == nil {
		ep.SeriesTitle = series.Title
	}
	event.Episode = ep
}

// movieNotificationAdapter adapts the notification service for movies.
type movieNotificationAdapter struct {
	svc *notification.Service
}

// DispatchMovieAdded implements movies.NotificationDispatcher.
func (a *movieNotificationAdapter) DispatchMovieAdded(ctx context.Context, movie *movies.MovieNotificationInfo, addedAt time.Time) {
	event := notification.MovieAddedEvent{
		Movie: notification.MediaInfo{
			ID:       movie.ID,
			Title:    movie.Title,
			Year:     movie.Year,
			TMDbID:   int64(movie.TmdbID),
			IMDbID:   movie.ImdbID,
			Overview: movie.Overview,
		},
		AddedAt: addedAt,
	}
	a.svc.DispatchMovieAdded(ctx, &event)
}

// DispatchMovieDeleted implements movies.NotificationDispatcher.
func (a *movieNotificationAdapter) DispatchMovieDeleted(ctx context.Context, movie *movies.MovieNotificationInfo, deletedFiles bool, deletedAt time.Time) {
	event := notification.MovieDeletedEvent{
		Movie: notification.MediaInfo{
			ID:       movie.ID,
			Title:    movie.Title,
			Year:     movie.Year,
			TMDbID:   int64(movie.TmdbID),
			IMDbID:   movie.ImdbID,
			Overview: movie.Overview,
		},
		DeletedFiles: deletedFiles,
		DeletedAt:    deletedAt,
	}
	a.svc.DispatchMovieDeleted(ctx, &event)
}

// tvNotificationAdapter adapts the notification service for TV series.
type tvNotificationAdapter struct {
	svc *notification.Service
}

// DispatchSeriesAdded implements tv.NotificationDispatcher.
func (a *tvNotificationAdapter) DispatchSeriesAdded(ctx context.Context, series *tv.SeriesNotificationInfo, addedAt time.Time) {
	event := notification.SeriesAddedEvent{
		Series: notification.SeriesInfo{
			MediaInfo: notification.MediaInfo{
				ID:       series.ID,
				Title:    series.Title,
				Year:     series.Year,
				TMDbID:   int64(series.TmdbID),
				IMDbID:   series.ImdbID,
				Overview: series.Overview,
			},
			TVDbID: int64(series.TvdbID),
		},
		AddedAt: addedAt,
	}
	a.svc.DispatchSeriesAdded(ctx, &event)
}

// DispatchSeriesDeleted implements tv.NotificationDispatcher.
func (a *tvNotificationAdapter) DispatchSeriesDeleted(ctx context.Context, series *tv.SeriesNotificationInfo, deletedFiles bool, deletedAt time.Time) {
	event := notification.SeriesDeletedEvent{
		Series: notification.SeriesInfo{
			MediaInfo: notification.MediaInfo{
				ID:       series.ID,
				Title:    series.Title,
				Year:     series.Year,
				TMDbID:   int64(series.TmdbID),
				IMDbID:   series.ImdbID,
				Overview: series.Overview,
			},
			TVDbID: int64(series.TvdbID),
		},
		DeletedFiles: deletedFiles,
		DeletedAt:    deletedAt,
	}
	a.svc.DispatchSeriesDeleted(ctx, &event)
}

// importNotificationAdapter adapts the notification service for imports.
type importNotificationAdapter struct {
	svc *notification.Service
}

// DispatchImport implements importer.NotificationDispatcher.
func (a *importNotificationAdapter) DispatchImport(ctx context.Context, event *importer.ImportNotificationEvent) {
	notifEvent := notification.ImportEvent{
		Quality:         event.Quality,
		SourcePath:      event.SourcePath,
		DestinationPath: event.DestinationPath,
		ReleaseName:     event.ReleaseName,
		ImportedAt:      time.Now(),
	}

	if event.MediaType == mediaTypeMovie && event.MovieID != nil {
		notifEvent.Movie = &notification.MediaInfo{
			ID:    *event.MovieID,
			Title: event.MovieTitle,
			Year:  event.MovieYear,
		}
	} else if event.MediaType == mediaTypeEpisode {
		ep := &notification.EpisodeInfo{
			SeriesTitle:   event.SeriesTitle,
			SeasonNumber:  event.SeasonNumber,
			EpisodeNumber: event.EpisodeNumber,
			EpisodeTitle:  event.EpisodeTitle,
		}
		if event.SeriesID != nil {
			ep.SeriesID = *event.SeriesID
		}
		notifEvent.Episode = ep
	}

	if event.SlotID != nil {
		notifEvent.Slot = &notification.SlotInfo{
			ID:   *event.SlotID,
			Name: event.SlotName,
		}
	}

	a.svc.DispatchDownload(ctx, &notifEvent)
}

// DispatchUpgrade implements importer.NotificationDispatcher.
func (a *importNotificationAdapter) DispatchUpgrade(ctx context.Context, event *importer.UpgradeNotificationEvent) {
	notifEvent := notification.UpgradeEvent{
		OldQuality:  event.OldQuality,
		NewQuality:  event.NewQuality,
		OldPath:     event.OldPath,
		NewPath:     event.NewPath,
		ReleaseName: event.ReleaseName,
		UpgradedAt:  time.Now(),
	}

	if event.MediaType == mediaTypeMovie && event.MovieID != nil {
		notifEvent.Movie = &notification.MediaInfo{
			ID:    *event.MovieID,
			Title: event.MovieTitle,
			Year:  event.MovieYear,
		}
	} else if event.MediaType == mediaTypeEpisode {
		ep := &notification.EpisodeInfo{
			SeriesTitle:   event.SeriesTitle,
			SeasonNumber:  event.SeasonNumber,
			EpisodeNumber: event.EpisodeNumber,
			EpisodeTitle:  event.EpisodeTitle,
		}
		if event.SeriesID != nil {
			ep.SeriesID = *event.SeriesID
		}
		notifEvent.Episode = ep
	}

	if event.SlotID != nil {
		notifEvent.Slot = &notification.SlotInfo{
			ID:   *event.SlotID,
			Name: event.SlotName,
		}
	}

	a.svc.DispatchUpgrade(ctx, &notifEvent)
}

type statusTrackerMovieLookup struct {
	movieSvc *movies.Service
}

func (l *statusTrackerMovieLookup) GetTmdbIDByMovieID(ctx context.Context, movieID int64) (int64, error) {
	movie, err := l.movieSvc.Get(ctx, movieID)
	if err != nil {
		return 0, err
	}
	return int64(movie.TmdbID), nil
}

// statusTrackerEpisodeLookup implements requests.EpisodeLookup
type statusTrackerEpisodeLookup struct {
	tvSvc *tv.Service
}

func (l *statusTrackerEpisodeLookup) GetEpisodeInfo(ctx context.Context, episodeID int64) (tvdbID int64, seasonNum, episodeNum int, err error) {
	episode, err := l.tvSvc.GetEpisode(ctx, episodeID)
	if err != nil {
		return 0, 0, 0, err
	}

	series, err := l.tvSvc.GetSeries(ctx, episode.SeriesID)
	if err != nil {
		return 0, 0, 0, err
	}

	return int64(series.TvdbID), episode.SeasonNumber, episode.EpisodeNumber, nil
}

// statusTrackerSeriesLookup implements requests.SeriesLookup
type statusTrackerSeriesLookup struct {
	tvSvc *tv.Service
}

func (l *statusTrackerSeriesLookup) GetSeriesIDByTvdbID(ctx context.Context, tvdbID int64) (int64, error) {
	return l.tvSvc.GetSeriesIDByTvdbID(ctx, tvdbID)
}

func (l *statusTrackerSeriesLookup) AreSeasonsComplete(ctx context.Context, seriesID int64, seasonNumbers []int64) (bool, error) {
	return l.tvSvc.AreSeasonsComplete(ctx, seriesID, seasonNumbers)
}

// arrImportRootFolderAdapter adapts rootfolder.Service to arrimport.RootFolderService
type arrImportRootFolderAdapter struct {
	svc *rootfolder.Service
}

func (a *arrImportRootFolderAdapter) List(ctx context.Context) ([]*arrimport.RootFolder, error) {
	folders, err := a.svc.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*arrimport.RootFolder, len(folders))
	for i, f := range folders {
		result[i] = &arrimport.RootFolder{
			ID:        f.ID,
			Name:      f.Name,
			Path:      f.Path,
			MediaType: f.MediaType,
		}
	}
	return result, nil
}

// arrImportQualityAdapter adapts quality.Service to arrimport.QualityService
type arrImportQualityAdapter struct {
	svc *quality.Service
}

func (a *arrImportQualityAdapter) List(ctx context.Context) ([]*arrimport.QualityProfile, error) {
	profiles, err := a.svc.List(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*arrimport.QualityProfile, len(profiles))
	for i, p := range profiles {
		result[i] = &arrimport.QualityProfile{
			ID:   p.ID,
			Name: p.Name,
		}
	}
	return result, nil
}

// arrImportHubAdapter adapts websocket.Hub to the BroadcastJSON interface
type arrImportHubAdapter struct {
	hub *websocket.Hub
}

func (a *arrImportHubAdapter) BroadcastJSON(v interface{}) {
	// The arrimport service expects to broadcast progress updates
	// Hub.Broadcast takes (msgType string, payload interface{})
	// We'll use a generic message type for arr import progress
	a.hub.Broadcast("arrImportProgress", v)
}

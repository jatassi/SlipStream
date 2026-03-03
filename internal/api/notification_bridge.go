package api

import (
	"context"
	"time"

	importer "github.com/slipstream/slipstream/internal/import"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	indexerTypes "github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/notification"
)

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
		IsSeasonPack: true,
	}
	if series, err := a.tv.GetSeries(ctx, media.SeriesID); err == nil {
		ep.SeriesTitle = series.Title
	}
	event.Episode = ep
}

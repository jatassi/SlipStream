package api

import (
	"context"
	"time"

	"github.com/slipstream/slipstream/internal/indexer/grab"
	indexerTypes "github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/notification"
)

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

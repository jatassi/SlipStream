package requests

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// MovieLookup provides movie information for status tracking.
type MovieLookup interface {
	GetTmdbIDByMovieID(ctx context.Context, movieID int64) (int64, error)
}

// EpisodeLookup provides episode information for status tracking.
type EpisodeLookup interface {
	GetEpisodeInfo(ctx context.Context, episodeID int64) (tvdbID int64, seasonNum, episodeNum int, err error)
}

// SeriesLookup provides series information for status tracking.
type SeriesLookup interface {
	GetSeriesIDByTvdbID(ctx context.Context, tvdbID int64) (int64, error)
	AreSeasonsComplete(ctx context.Context, seriesID int64, seasonNumbers []int64) (bool, error)
}

type StatusTracker struct {
	queries         *sqlc.Queries
	requestsService *Service
	watchersService *WatchersService
	notifDispatcher NotificationDispatcher
	movieLookup     MovieLookup
	episodeLookup   EpisodeLookup
	seriesLookup    SeriesLookup
	logger          zerolog.Logger
}

func NewStatusTracker(
	queries *sqlc.Queries,
	requestsService *Service,
	watchersService *WatchersService,
	logger zerolog.Logger,
) *StatusTracker {
	return &StatusTracker{
		queries:         queries,
		requestsService: requestsService,
		watchersService: watchersService,
		logger:          logger.With().Str("component", "portal-status-tracker").Logger(),
	}
}

func (t *StatusTracker) SetDB(db *sql.DB) {
	t.queries = sqlc.New(db)
}

func (t *StatusTracker) SetMovieLookup(ml MovieLookup) {
	t.movieLookup = ml
}

func (t *StatusTracker) SetEpisodeLookup(el EpisodeLookup) {
	t.episodeLookup = el
}

func (t *StatusTracker) SetSeriesLookup(sl SeriesLookup) {
	t.seriesLookup = sl
}

func (t *StatusTracker) SetNotificationDispatcher(nd NotificationDispatcher) {
	t.notifDispatcher = nd
}

func (t *StatusTracker) OnMovieAvailable(ctx context.Context, movieID int64) error {
	if t.movieLookup == nil {
		t.logger.Warn().Msg("movie lookup not configured, cannot update request status")
		return nil
	}

	tmdbID, err := t.movieLookup.GetTmdbIDByMovieID(ctx, movieID)
	if err != nil {
		t.logger.Debug().Err(err).Int64("movieID", movieID).Msg("failed to get tmdb ID for movie")
		return nil
	}

	if tmdbID == 0 {
		t.logger.Debug().Int64("movieID", movieID).Msg("movie has no tmdb ID, skipping request update")
		return nil
	}

	req, err := t.requestsService.GetByTmdbID(ctx, tmdbID, MediaTypeMovie)
	if err != nil {
		if errors.Is(err, ErrRequestNotFound) {
			t.logger.Debug().Int64("movieID", movieID).Int64("tmdbID", tmdbID).Msg("no request found for movie")
			return nil
		}
		return err
	}

	if req.Status != StatusDownloading && req.Status != StatusApproved {
		t.logger.Debug().Int64("requestID", req.ID).Str("status", req.Status).Msg("request not in downloading/approved status, skipping")
		return nil
	}

	if err := t.markAvailable(ctx, req); err != nil {
		t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to mark request as available")
	}

	return nil
}

func (t *StatusTracker) OnEpisodeAvailable(ctx context.Context, episodeID int64) error {
	if t.episodeLookup == nil {
		t.logger.Warn().Msg("episode lookup not configured, cannot update request status")
		return nil
	}

	tvdbID, seasonNum, episodeNum, err := t.episodeLookup.GetEpisodeInfo(ctx, episodeID)
	if err != nil {
		t.logger.Debug().Err(err).Int64("episodeID", episodeID).Msg("failed to get episode info")
		return nil
	}

	if tvdbID == 0 {
		t.logger.Debug().Int64("episodeID", episodeID).Msg("episode has no tvdb ID, skipping request update")
		return nil
	}

	// First, try to find an episode-level request
	req, err := t.requestsService.GetByTvdbIDAndEpisode(ctx, tvdbID, int64(seasonNum), int64(episodeNum))
	if err != nil && !errors.Is(err, ErrRequestNotFound) {
		return err
	}

	if req != nil {
		if req.Status == StatusDownloading || req.Status == StatusApproved {
			if err := t.markAvailable(ctx, req); err != nil {
				t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to mark request as available")
			}
		}
		return nil
	}

	// No episode-level request found, check for series requests with requestedSeasons
	if err := t.checkSeriesRequestsForSeason(ctx, tvdbID, int64(seasonNum)); err != nil {
		t.logger.Warn().Err(err).Int64("tvdbID", tvdbID).Int("seasonNum", seasonNum).Msg("failed to check series requests")
	}

	return nil
}

func (t *StatusTracker) checkSeriesRequestsForSeason(ctx context.Context, tvdbID int64, seasonNum int64) error {
	// First, find series requests directly by TVDB ID (works even without MediaID linked)
	requests, err := t.findSeriesRequestsByTvdbID(ctx, tvdbID, seasonNum)
	if err != nil {
		t.logger.Warn().Err(err).Int64("tvdbID", tvdbID).Msg("failed to find series requests by TVDB ID")
	}

	// Also check by MediaID if series lookup is configured
	if t.seriesLookup != nil {
		seriesID, err := t.seriesLookup.GetSeriesIDByTvdbID(ctx, tvdbID)
		if err == nil && seriesID > 0 {
			mediaIDRequests, err := t.findSeriesRequestsWithSeason(ctx, seriesID, seasonNum)
			if err == nil {
				// Merge requests, avoiding duplicates
				seen := make(map[int64]bool)
				for _, r := range requests {
					seen[r.ID] = true
				}
				for _, r := range mediaIDRequests {
					if !seen[r.ID] {
						requests = append(requests, r)
					}
				}
			}
		}
	}

	if len(requests) == 0 {
		return nil
	}

	// Get internal series ID for completion check
	var seriesID int64
	if t.seriesLookup != nil {
		seriesID, _ = t.seriesLookup.GetSeriesIDByTvdbID(ctx, tvdbID)
	}

	for _, req := range requests {
		// Check if all requested seasons are now complete
		if t.seriesLookup != nil && seriesID > 0 && len(req.RequestedSeasons) > 0 {
			allComplete, err := t.seriesLookup.AreSeasonsComplete(ctx, seriesID, req.RequestedSeasons)
			if err != nil {
				t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to check if seasons are complete")
				continue
			}

			if allComplete {
				if err := t.markAvailable(ctx, req); err != nil {
					t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to mark request as available")
				}
			} else {
				t.logger.Debug().
					Int64("requestID", req.ID).
					Interface("requestedSeasons", req.RequestedSeasons).
					Msg("not all requested seasons are complete yet")
			}
		} else if len(req.RequestedSeasons) == 0 {
			// Full series request without specific seasons - mark as available
			if err := t.markAvailable(ctx, req); err != nil {
				t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to mark request as available")
			}
		}
	}

	return nil
}

func (t *StatusTracker) findSeriesRequestsByTvdbID(ctx context.Context, tvdbID int64, seasonNum int64) ([]*Request, error) {
	rows, err := t.queries.ListActiveSeriesRequestsByTvdbID(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if err != nil {
		return nil, err
	}

	result := make([]*Request, 0)
	for _, row := range rows {
		req := toRequest(row)

		// Check if this request has requestedSeasons that include our season
		if len(req.RequestedSeasons) == 0 {
			// Full series request - include it
			result = append(result, req)
			continue
		}

		for _, s := range req.RequestedSeasons {
			if s == seasonNum {
				result = append(result, req)
				break
			}
		}
	}

	return result, nil
}

func (t *StatusTracker) findSeriesRequestsWithSeason(ctx context.Context, seriesID int64, seasonNum int64) ([]*Request, error) {
	// Get all series requests for this media
	rows, err := t.queries.ListRequestsByMediaID(ctx, sqlc.ListRequestsByMediaIDParams{
		MediaID:   sql.NullInt64{Int64: seriesID, Valid: true},
		MediaType: MediaTypeSeries,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*Request, 0)
	for _, row := range rows {
		if row.Status != StatusDownloading && row.Status != StatusApproved {
			continue
		}

		req := toRequest(row)

		// Check if this request has requestedSeasons that include our season
		if len(req.RequestedSeasons) == 0 {
			// No specific seasons requested, this is a full series request
			// Skip for now - full series requests should use OnSeriesComplete
			continue
		}

		for _, s := range req.RequestedSeasons {
			if s == seasonNum {
				result = append(result, req)
				break
			}
		}
	}

	return result, nil
}

func (t *StatusTracker) OnSeasonComplete(ctx context.Context, seriesID int64, seasonNumber int) error {
	requests, err := t.findSeasonRequests(ctx, seriesID, int64(seasonNumber))
	if err != nil {
		return err
	}

	for _, req := range requests {
		if err := t.markAvailable(ctx, req); err != nil {
			t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to mark request as available")
		}
	}

	return nil
}

func (t *StatusTracker) OnSeriesComplete(ctx context.Context, seriesID int64) error {
	requests, err := t.findRequestsByMediaID(ctx, seriesID, MediaTypeSeries)
	if err != nil {
		return err
	}

	for _, req := range requests {
		if err := t.markAvailable(ctx, req); err != nil {
			t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to mark request as available")
		}
	}

	return nil
}

func (t *StatusTracker) findRequestsByMediaID(ctx context.Context, mediaID int64, mediaType string) ([]*Request, error) {
	rows, err := t.queries.ListRequestsByMediaID(ctx, sqlc.ListRequestsByMediaIDParams{
		MediaID:   sql.NullInt64{Int64: mediaID, Valid: true},
		MediaType: mediaType,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*Request, 0, len(rows))
	for _, row := range rows {
		if row.Status == StatusDownloading || row.Status == StatusApproved {
			result = append(result, toRequest(row))
		}
	}

	return result, nil
}

func (t *StatusTracker) findSeasonRequests(ctx context.Context, seriesID int64, seasonNumber int64) ([]*Request, error) {
	rows, err := t.queries.ListRequestsByMediaIDAndSeason(ctx, sqlc.ListRequestsByMediaIDAndSeasonParams{
		MediaID:      sql.NullInt64{Int64: seriesID, Valid: true},
		MediaType:    MediaTypeSeason,
		SeasonNumber: sql.NullInt64{Int64: seasonNumber, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	result := make([]*Request, 0, len(rows))
	for _, row := range rows {
		if row.Status == StatusDownloading || row.Status == StatusApproved {
			result = append(result, toRequest(row))
		}
	}

	return result, nil
}

func (t *StatusTracker) markAvailable(ctx context.Context, req *Request) error {
	_, err := t.requestsService.UpdateStatus(ctx, req.ID, StatusAvailable)
	if err != nil {
		return err
	}

	t.logger.Info().
		Int64("requestID", req.ID).
		Int64("userID", req.UserID).
		Str("mediaType", req.MediaType).
		Str("title", req.Title).
		Msg("request marked as available")

	// Dispatch notification
	if t.notifDispatcher != nil {
		watcherIDs, _ := t.watchersService.GetWatcherUserIDs(ctx, req.ID)
		go t.notifDispatcher.NotifyRequestAvailable(context.Background(), req, watcherIDs)
	}

	return nil
}

func (t *StatusTracker) GetWatcherUserIDs(ctx context.Context, requestID int64) ([]int64, error) {
	return t.watchersService.GetWatcherUserIDs(ctx, requestID)
}

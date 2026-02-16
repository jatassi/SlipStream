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
	logger          *zerolog.Logger
}

func NewStatusTracker(
	queries *sqlc.Queries,
	requestsService *Service,
	watchersService *WatchersService,
	logger *zerolog.Logger,
) *StatusTracker {
	subLogger := logger.With().Str("component", "portal-status-tracker").Logger()
	return &StatusTracker{
		queries:         queries,
		requestsService: requestsService,
		watchersService: watchersService,
		logger:          &subLogger,
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

// OnDownloadStarted is called when a media item transitions to downloading status.
// It updates any linked request to downloading status.
func (t *StatusTracker) OnDownloadStarted(ctx context.Context, mediaType string, mediaID int64) error {
	// Direct lookup for movie/episode requests
	reqs, err := t.findRequestsByMediaID(ctx, mediaID, mediaType)
	if err != nil {
		return err
	}

	// For episodes, also look up series/season requests linked to the episode's series
	if mediaType == "episode" && t.episodeLookup != nil {
		tvdbID, seasonNum, _, lookupErr := t.episodeLookup.GetEpisodeInfo(ctx, mediaID)
		if lookupErr == nil && tvdbID > 0 {
			seriesReqs, _ := t.findSeriesRequestsByTvdbID(ctx, tvdbID, int64(seasonNum))
			reqs = append(reqs, seriesReqs...)
		}
	}

	for _, req := range reqs {
		if req.Status == StatusApproved {
			if _, err := t.requestsService.UpdateStatus(ctx, req.ID, StatusDownloading); err != nil {
				t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to mark request as downloading")
			} else {
				t.logger.Info().Int64("requestID", req.ID).Str("title", req.Title).Msg("request marked as downloading")
			}
		}
	}

	return nil
}

// OnDownloadFailed is called when a media item transitions to failed status.
// For movie/episode requests, it sets the request to failed.
// For season/series requests, it only sets failed if ALL linked episodes are failed.
func (t *StatusTracker) OnDownloadFailed(ctx context.Context, mediaType string, mediaID int64) error {
	reqs := t.collectFailedDownloadRequests(ctx, mediaType, mediaID)

	for _, req := range reqs {
		if !t.isDownloadingOrApproved(req) {
			continue
		}
		t.processFailedRequest(ctx, req)
	}

	return nil
}

func (t *StatusTracker) collectFailedDownloadRequests(ctx context.Context, mediaType string, mediaID int64) []*Request {
	reqs, err := t.findRequestsByMediaID(ctx, mediaID, mediaType)
	if err != nil {
		return nil
	}

	if mediaType == "episode" && t.episodeLookup != nil {
		seriesReqs := t.findSeriesRequestsForEpisode(ctx, mediaID)
		reqs = append(reqs, seriesReqs...)
	}

	return reqs
}

func (t *StatusTracker) findSeriesRequestsForEpisode(ctx context.Context, episodeID int64) []*Request {
	tvdbID, seasonNum, _, lookupErr := t.episodeLookup.GetEpisodeInfo(ctx, episodeID)
	if lookupErr != nil || tvdbID == 0 {
		return nil
	}

	seriesReqs, _ := t.findSeriesRequestsByTvdbID(ctx, tvdbID, int64(seasonNum))
	return seriesReqs
}

func (t *StatusTracker) isDownloadingOrApproved(req *Request) bool {
	return req.Status == StatusDownloading || req.Status == StatusApproved
}

func (t *StatusTracker) processFailedRequest(ctx context.Context, req *Request) {
	if req.MediaType == MediaTypeMovie || req.MediaType == MediaTypeEpisode {
		t.markRequestFailed(ctx, req, "request marked as failed")
		return
	}

	if t.shouldMarkSeriesRequestFailed(ctx, req) {
		t.markRequestFailed(ctx, req, "series request marked as failed (all episodes failed)")
	}
}

func (t *StatusTracker) shouldMarkSeriesRequestFailed(ctx context.Context, req *Request) bool {
	if t.seriesLookup == nil || req.MediaID == nil {
		return false
	}

	allFailed, err := t.areAllEpisodesFailed(ctx, *req.MediaID, req.RequestedSeasons)
	if err != nil {
		t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to check episode statuses")
		return false
	}

	return allFailed
}

func (t *StatusTracker) markRequestFailed(ctx context.Context, req *Request, message string) {
	if _, err := t.requestsService.UpdateStatus(ctx, req.ID, StatusFailed); err != nil {
		t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to mark request as failed")
	} else {
		t.logger.Info().Int64("requestID", req.ID).Str("title", req.Title).Msg(message)
	}
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

	tvdbID, seasonNum, episodeNum, err := t.getEpisodeIdentifiers(ctx, episodeID)
	if err != nil {
		return err
	}
	if tvdbID == 0 {
		return nil
	}

	if err := t.processEpisodeLevelRequest(ctx, tvdbID, seasonNum, episodeNum); err != nil {
		return err
	}

	t.processSeriesLevelRequests(ctx, tvdbID, seasonNum)
	return nil
}

func (t *StatusTracker) getEpisodeIdentifiers(ctx context.Context, episodeID int64) (tvdbID int64, seasonNum, episodeNum int, err error) {
	tvdbID, seasonNum, episodeNum, err = t.episodeLookup.GetEpisodeInfo(ctx, episodeID)
	if err != nil {
		t.logger.Debug().Err(err).Int64("episodeID", episodeID).Msg("failed to get episode info")
		return 0, 0, 0, err
	}

	if tvdbID == 0 {
		t.logger.Debug().Int64("episodeID", episodeID).Msg("episode has no tvdb ID, skipping request update")
	}

	return tvdbID, seasonNum, episodeNum, nil
}

func (t *StatusTracker) processEpisodeLevelRequest(ctx context.Context, tvdbID int64, seasonNum, episodeNum int) error {
	req, err := t.requestsService.GetByTvdbIDAndEpisode(ctx, tvdbID, int64(seasonNum), int64(episodeNum))
	if err != nil && !errors.Is(err, ErrRequestNotFound) {
		return err
	}

	if req == nil {
		return nil
	}

	if req.Status == StatusDownloading || req.Status == StatusApproved {
		if err := t.markAvailable(ctx, req); err != nil {
			t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to mark request as available")
		}
	}

	return nil
}

func (t *StatusTracker) processSeriesLevelRequests(ctx context.Context, tvdbID int64, seasonNum int) {
	t.checkSeriesRequestsForSeason(ctx, tvdbID, int64(seasonNum))
}

func (t *StatusTracker) checkSeriesRequestsForSeason(ctx context.Context, tvdbID, seasonNum int64) {
	requests := t.collectSeriesRequests(ctx, tvdbID, seasonNum)
	if len(requests) == 0 {
		return
	}

	seriesID := t.getSeriesIDForCompletion(ctx, tvdbID)

	for _, req := range requests {
		if err := t.processSeasonRequest(ctx, req, seriesID, seasonNum); err != nil {
			t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to process season request")
		}
	}
}

func (t *StatusTracker) collectSeriesRequests(ctx context.Context, tvdbID, seasonNum int64) []*Request {
	requests, err := t.findSeriesRequestsByTvdbID(ctx, tvdbID, seasonNum)
	if err != nil {
		t.logger.Warn().Err(err).Int64("tvdbID", tvdbID).Msg("failed to find series requests by TVDB ID")
	}

	if t.seriesLookup == nil {
		return requests
	}

	seriesID, err := t.seriesLookup.GetSeriesIDByTvdbID(ctx, tvdbID)
	if err != nil || seriesID == 0 {
		return requests
	}

	mediaIDRequests, err := t.findSeriesRequestsWithSeason(ctx, seriesID, seasonNum)
	if err != nil {
		return requests
	}

	return t.mergeRequests(requests, mediaIDRequests)
}

func (t *StatusTracker) mergeRequests(requests, mediaIDRequests []*Request) []*Request {
	seen := make(map[int64]bool)
	for _, r := range requests {
		seen[r.ID] = true
	}
	for _, r := range mediaIDRequests {
		if !seen[r.ID] {
			requests = append(requests, r)
		}
	}
	return requests
}

func (t *StatusTracker) getSeriesIDForCompletion(ctx context.Context, tvdbID int64) int64 {
	if t.seriesLookup == nil {
		return 0
	}
	seriesID, _ := t.seriesLookup.GetSeriesIDByTvdbID(ctx, tvdbID)
	return seriesID
}

func (t *StatusTracker) processSeasonRequest(ctx context.Context, req *Request, seriesID, _seasonNum int64) error {
	if !t.shouldProcessSeasonRequest(req, seriesID) {
		return nil
	}

	if len(req.RequestedSeasons) == 0 {
		return t.markAvailable(ctx, req)
	}

	allComplete, err := t.seriesLookup.AreSeasonsComplete(ctx, seriesID, req.RequestedSeasons)
	if err != nil {
		return err
	}

	if allComplete {
		return t.markAvailable(ctx, req)
	}

	t.logger.Debug().
		Int64("requestID", req.ID).
		Interface("requestedSeasons", req.RequestedSeasons).
		Msg("not all requested seasons are complete yet")

	return nil
}

func (t *StatusTracker) shouldProcessSeasonRequest(_req *Request, seriesID int64) bool {
	return t.seriesLookup != nil && seriesID > 0
}

func (t *StatusTracker) findSeriesRequestsByTvdbID(ctx context.Context, tvdbID, seasonNum int64) ([]*Request, error) {
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

func (t *StatusTracker) findSeriesRequestsWithSeason(ctx context.Context, seriesID, seasonNum int64) ([]*Request, error) {
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

func (t *StatusTracker) findSeasonRequests(ctx context.Context, seriesID, seasonNumber int64) ([]*Request, error) {
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

// areAllEpisodesFailed checks if all monitored episodes for a series (or specific seasons) are failed.
func (t *StatusTracker) areAllEpisodesFailed(ctx context.Context, seriesID int64, requestedSeasons []int64) (bool, error) {
	if len(requestedSeasons) == 0 {
		// Full series request - check all episodes
		count, err := t.queries.CountNonFailedMonitoredEpisodesBySeries(ctx, seriesID)
		if err != nil {
			return false, err
		}
		return count == 0, nil
	}

	// Check each requested season
	for _, seasonNum := range requestedSeasons {
		count, err := t.queries.CountNonFailedMonitoredEpisodesBySeason(ctx, sqlc.CountNonFailedMonitoredEpisodesBySeasonParams{
			SeriesID:     seriesID,
			SeasonNumber: seasonNum,
		})
		if err != nil {
			return false, err
		}
		if count > 0 {
			return false, nil
		}
	}

	return true, nil
}

func (t *StatusTracker) GetWatcherUserIDs(ctx context.Context, requestID int64) ([]int64, error) {
	return t.watchersService.GetWatcherUserIDs(ctx, requestID)
}

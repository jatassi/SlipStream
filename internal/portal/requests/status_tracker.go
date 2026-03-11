package requests

import (
	"context"
	"database/sql"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/module"
)

// ModuleProvisionerLookup provides access to module provisioners and schema info by entity type.
type ModuleProvisionerLookup interface {
	GetProvisionerForEntityType(entityType string) module.PortalProvisioner
	IsLeafEntityType(entityType string) bool
}

type StatusTracker struct {
	queries           *sqlc.Queries
	requestsService   *Service
	watchersService   *WatchersService
	notifDispatcher   NotificationDispatcher
	provisionerLookup ModuleProvisionerLookup
	logger            *zerolog.Logger
}

func NewStatusTracker(
	queries *sqlc.Queries,
	requestsService *Service,
	watchersService *WatchersService,
	logger *zerolog.Logger,
	provisionerLookup ModuleProvisionerLookup,
	notifDispatcher NotificationDispatcher,
) *StatusTracker {
	subLogger := logger.With().Str("component", "portal-status-tracker").Logger()
	return &StatusTracker{
		queries:           queries,
		requestsService:   requestsService,
		watchersService:   watchersService,
		logger:            &subLogger,
		provisionerLookup: provisionerLookup,
		notifDispatcher:   notifDispatcher,
	}
}

func (t *StatusTracker) SetDB(db *sql.DB) {
	t.queries = sqlc.New(db)
}

// OnDownloadStarted is called when a media item transitions to downloading status.
// It updates any linked request to downloading status.
func (t *StatusTracker) OnDownloadStarted(ctx context.Context, mediaType string, mediaID int64) error {
	reqs, err := t.findRequestsByMediaID(ctx, mediaID, mediaType)
	if err != nil {
		return err
	}

	// For leaf entities (e.g., episodes), also look up parent requests
	parentReqs := t.findParentRequests(ctx, mediaType, mediaID)
	reqs = append(reqs, parentReqs...)

	for _, req := range reqs {
		if req.Status == StatusApproved || req.Status == StatusSearching {
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
// For direct entity matches, it sets the request to failed.
// For parent requests, it delegates to the module provisioner to decide.
func (t *StatusTracker) OnDownloadFailed(ctx context.Context, mediaType string, mediaID int64) error {
	reqs, err := t.findRequestsByMediaID(ctx, mediaID, mediaType)
	if err != nil {
		return err
	}

	parentReqs := t.findParentRequests(ctx, mediaType, mediaID)
	reqs = append(reqs, parentReqs...)

	for _, req := range reqs {
		if !t.isDownloadingOrApproved(req) {
			continue
		}
		t.processFailedRequest(ctx, req, mediaType, mediaID)
	}

	return nil
}

// OnEntityAvailable is called when a media entity becomes available.
// It finds all requests that could be satisfied by this entity and processes them.
func (t *StatusTracker) OnEntityAvailable(ctx context.Context, moduleType, entityType string, entityID int64) error {
	// Find requests directly linked to this entity
	reqs := t.findDirectRequests(ctx, entityType, entityID)

	// For leaf entities, also find parent-level requests that may be satisfied
	provisioner := t.provisionerLookup.GetProvisionerForEntityType(entityType)
	parentReqs := t.findParentRequests(ctx, entityType, entityID)
	reqs = append(reqs, parentReqs...)

	for _, req := range reqs {
		t.processRequestCompletion(ctx, req, provisioner, entityType, entityID)
	}

	return nil
}

func (t *StatusTracker) processRequestCompletion(ctx context.Context, req *Request, provisioner module.PortalProvisioner, availableEntityType string, availableEntityID int64) {
	if !t.isDownloadingOrApproved(req) {
		return
	}

	// For direct entity matches (e.g., movie request + movie available), mark available immediately
	if req.MediaType == availableEntityType {
		t.tryMarkAvailable(ctx, req)
		return
	}

	// For parent requests (e.g., series request + episode available), delegate to module
	if provisioner == nil {
		return
	}

	input := t.buildCompletionInput(req, availableEntityType, availableEntityID)
	result, err := provisioner.CheckRequestCompletion(ctx, input)
	if err != nil {
		t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to check request completion")
		return
	}

	if result.ShouldMarkAvailable {
		t.tryMarkAvailable(ctx, req)
	}
}

func (t *StatusTracker) buildCompletionInput(req *Request, availableEntityType string, availableEntityID int64) *module.RequestCompletionCheckInput {
	input := &module.RequestCompletionCheckInput{
		AvailableEntityType: availableEntityType,
		AvailableEntityID:   availableEntityID,
		RequestEntityType:   req.MediaType,
		RequestEntityID:     req.MediaID,
		RequestedSeasons:    req.RequestedSeasons,
		RequestExternalIDs:  t.collectExternalIDs(req),
	}
	return input
}

func (t *StatusTracker) collectExternalIDs(req *Request) map[string]int64 {
	ids := make(map[string]int64)
	if req.TmdbID != nil {
		ids["tmdb"] = *req.TmdbID
	}
	if req.TvdbID != nil {
		ids["tvdb"] = *req.TvdbID
	}
	if len(ids) == 0 {
		return nil
	}
	return ids
}

func (t *StatusTracker) tryMarkAvailable(ctx context.Context, req *Request) {
	if err := t.markAvailable(ctx, req); err != nil {
		t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to mark request as available")
	}
}

func (t *StatusTracker) processFailedRequest(ctx context.Context, req *Request, _ string, _ int64) {
	// Leaf entity types (e.g., movie, episode) are marked failed immediately
	if t.provisionerLookup.IsLeafEntityType(req.MediaType) {
		t.markRequestFailed(ctx, req, "request marked as failed")
		return
	}

	// For parent requests, delegate to module provisioner to check if all children failed
	if req.MediaID == nil {
		return
	}

	provisioner := t.provisionerLookup.GetProvisionerForEntityType(req.MediaType)
	if provisioner == nil {
		return
	}

	allFailed, err := t.areAllEpisodesFailed(ctx, *req.MediaID, req.RequestedSeasons)
	if err != nil {
		t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to check episode statuses")
		return
	}

	if allFailed {
		t.markRequestFailed(ctx, req, "series request marked as failed (all episodes failed)")
	}
}

// findDirectRequests finds requests where media_id and media_type match the given entity.
func (t *StatusTracker) findDirectRequests(ctx context.Context, entityType string, entityID int64) []*Request {
	reqs, err := t.findRequestsByMediaID(ctx, entityID, entityType)
	if err != nil {
		t.logger.Debug().Err(err).Str("entityType", entityType).Int64("entityID", entityID).Msg("failed to find direct requests")
		return nil
	}
	return reqs
}

// findParentRequests finds parent-level requests that might be satisfied by a leaf entity.
// For episodes, this looks up the episode's series TVDB ID and finds series-level requests.
func (t *StatusTracker) findParentRequests(ctx context.Context, entityType string, entityID int64) []*Request {
	if entityType != "episode" {
		return nil
	}

	// Look up episode to get series ID, then look up series to get TVDB ID
	episode, err := t.queries.GetEpisode(ctx, entityID)
	if err != nil {
		t.logger.Debug().Err(err).Int64("entityID", entityID).Msg("failed to get episode for parent request lookup")
		return nil
	}

	series, err := t.queries.GetSeries(ctx, episode.SeriesID)
	if err != nil {
		t.logger.Debug().Err(err).Int64("seriesID", episode.SeriesID).Msg("failed to get series for parent request lookup")
		return nil
	}

	if !series.TvdbID.Valid || series.TvdbID.Int64 == 0 {
		return nil
	}

	tvdbID := series.TvdbID.Int64
	seasonNum := episode.SeasonNumber

	// Find series requests by TVDB ID that cover this season
	seriesReqs, err := t.findSeriesRequestsByTvdbID(ctx, tvdbID, seasonNum)
	if err != nil {
		t.logger.Debug().Err(err).Int64("tvdbID", tvdbID).Msg("failed to find series requests by TVDB ID")
		return nil
	}

	// Also find requests linked by media_id (series ID)
	mediaIDReqs, err := t.findSeriesRequestsWithSeason(ctx, series.ID, seasonNum)
	if err != nil {
		return seriesReqs
	}

	return t.mergeRequests(seriesReqs, mediaIDReqs)
}

func (t *StatusTracker) isDownloadingOrApproved(req *Request) bool {
	return req.Status == StatusDownloading || req.Status == StatusApproved || req.Status == StatusSearching
}

func (t *StatusTracker) markRequestFailed(ctx context.Context, req *Request, message string) {
	if _, err := t.requestsService.UpdateStatus(ctx, req.ID, StatusFailed); err != nil {
		t.logger.Warn().Err(err).Int64("requestID", req.ID).Msg("failed to mark request as failed")
	} else {
		t.logger.Info().Int64("requestID", req.ID).Str("title", req.Title).Msg(message)
	}
}

func (t *StatusTracker) findSeriesRequestsByTvdbID(ctx context.Context, tvdbID, seasonNum int64) ([]*Request, error) {
	rows, err := t.queries.ListActiveSeriesRequestsByTvdbID(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if err != nil {
		return nil, err
	}

	result := make([]*Request, 0)
	for _, row := range rows {
		req := toRequest(row)

		if len(req.RequestedSeasons) == 0 {
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
	rows, err := t.queries.ListRequestsByMediaID(ctx, sqlc.ListRequestsByMediaIDParams{
		MediaID:    sql.NullInt64{Int64: seriesID, Valid: true},
		EntityType: MediaTypeSeries,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*Request, 0)
	for _, row := range rows {
		if row.Status != StatusDownloading && row.Status != StatusApproved && row.Status != StatusSearching {
			continue
		}

		req := toRequest(row)

		if len(req.RequestedSeasons) == 0 {
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

func (t *StatusTracker) findRequestsByMediaID(ctx context.Context, mediaID int64, mediaType string) ([]*Request, error) {
	rows, err := t.queries.ListRequestsByMediaID(ctx, sqlc.ListRequestsByMediaIDParams{
		MediaID:    sql.NullInt64{Int64: mediaID, Valid: true},
		EntityType: mediaType,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*Request, 0, len(rows))
	for _, row := range rows {
		if row.Status == StatusDownloading || row.Status == StatusApproved || row.Status == StatusSearching {
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

	if t.notifDispatcher != nil {
		watcherIDs, _ := t.watchersService.GetWatcherUserIDs(ctx, req.ID)
		go t.notifDispatcher.NotifyRequestAvailable(context.Background(), req, watcherIDs)
	}

	return nil
}

// areAllEpisodesFailed checks if all monitored episodes for a series (or specific seasons) are failed.
func (t *StatusTracker) areAllEpisodesFailed(ctx context.Context, seriesID int64, requestedSeasons []int64) (bool, error) {
	if len(requestedSeasons) == 0 {
		count, err := t.queries.CountNonFailedMonitoredEpisodesBySeries(ctx, seriesID)
		if err != nil {
			return false, err
		}
		return count == 0, nil
	}

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

package requests

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/module"
)

var (
	ErrMediaNotInLibrary = errors.New("media not in library")
	ErrSearchFailed      = errors.New("search failed")
)

type SearchForRequestResult struct {
	Found      bool   `json:"found"`
	Downloaded bool   `json:"downloaded"`
	Error      string `json:"error,omitempty"`
}

type ModuleProvisioner interface {
	EnsureInLibrary(ctx context.Context, moduleType string, input *module.ProvisionInput) (int64, error)
}

type UserQualityProfileGetter interface {
	GetQualityProfileID(ctx context.Context, userID int64, mediaType string) (*int64, error)
}

type RequestSearcher struct {
	queries         *sqlc.Queries
	requestsService *Service
	autosearchSvc   *autosearch.Service
	provisioner     ModuleProvisioner
	userGetter      UserQualityProfileGetter
	isDevMode       func() bool
	logger          *zerolog.Logger
}

func NewRequestSearcher(
	queries *sqlc.Queries,
	requestsService *Service,
	autosearchSvc *autosearch.Service,
	provisioner ModuleProvisioner,
	logger *zerolog.Logger,
) *RequestSearcher {
	subLogger := logger.With().Str("component", "portal-request-searcher").Logger()
	return &RequestSearcher{
		queries:         queries,
		requestsService: requestsService,
		autosearchSvc:   autosearchSvc,
		provisioner:     provisioner,
		logger:          &subLogger,
	}
}

func (s *RequestSearcher) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)
}

func (s *RequestSearcher) SetUserGetter(getter UserQualityProfileGetter) {
	s.userGetter = getter
}

func (s *RequestSearcher) SetDevMode(fn func() bool) {
	s.isDevMode = fn
}

func (s *RequestSearcher) SearchForRequest(ctx context.Context, requestID int64) (*SearchForRequestResult, error) {
	s.logger.Info().Int64("requestID", requestID).Msg("SearchForRequest called")

	request, err := s.requestsService.Get(ctx, requestID)
	if err != nil {
		s.logger.Error().Err(err).Int64("requestID", requestID).Msg("failed to get request")
		return nil, err
	}

	s.logRequestDetails(requestID, request)

	if err := s.ensureRequestHasMediaID(ctx, requestID, request); err != nil {
		return nil, err
	}

	if _, err := s.requestsService.UpdateStatus(ctx, requestID, StatusSearching); err != nil {
		s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("failed to set request status to searching")
	}

	if s.isDevMode != nil && s.isDevMode() {
		time.Sleep(3 * time.Second)
	}

	result := s.executeSearch(ctx, requestID, request)

	if result.Downloaded {
		s.updateRequestStatusDownloading(ctx, requestID)
	} else {
		s.revertToApproved(ctx, requestID)
	}

	s.logSearchCompletion(requestID, result)
	return result, nil
}

func (s *RequestSearcher) revertToApproved(ctx context.Context, requestID int64) {
	current, err := s.requestsService.Get(ctx, requestID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("failed to get request for revert")
		return
	}
	if current.Status != StatusSearching {
		return
	}
	if _, err := s.requestsService.UpdateStatus(ctx, requestID, StatusApproved); err != nil {
		s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("failed to revert request status to approved")
	}
}

func (s *RequestSearcher) logRequestDetails(requestID int64, request *Request) {
	s.logger.Info().
		Int64("requestID", requestID).
		Str("title", request.Title).
		Str("mediaType", request.MediaType).
		Interface("mediaID", request.MediaID).
		Interface("tmdbID", request.TmdbID).
		Interface("tvdbID", request.TvdbID).
		Msg("got request details")
}

func (s *RequestSearcher) ensureRequestHasMediaID(ctx context.Context, requestID int64, request *Request) error {
	if request.MediaID != nil {
		return nil
	}

	s.logger.Info().Int64("requestID", requestID).Msg("MediaID is nil, attempting to ensure media in library")

	mediaID, err := s.ensureMediaInLibrary(ctx, request)
	if err != nil {
		s.logger.Error().Err(err).Int64("requestID", requestID).Msg("failed to ensure media in library")
		return fmt.Errorf("failed to provision media: %w", err)
	}

	s.logger.Info().Int64("requestID", requestID).Int64("mediaID", mediaID).Msg("ensureMediaInLibrary returned mediaID")

	if _, err := s.requestsService.LinkMedia(ctx, requestID, mediaID); err != nil {
		s.logger.Error().Err(err).Int64("requestID", requestID).Int64("mediaID", mediaID).Msg("failed to link request to media")
		return fmt.Errorf("failed to link request to media: %w", err)
	}

	request.MediaID = &mediaID
	s.logger.Info().Int64("requestID", requestID).Int64("mediaID", mediaID).Msg("linked request to media")
	return nil
}

func (s *RequestSearcher) executeSearch(ctx context.Context, requestID int64, request *Request) *SearchForRequestResult {
	result := &SearchForRequestResult{}

	switch request.MediaType {
	case MediaTypeMovie:
		s.searchMovie(ctx, requestID, request, result)
	case MediaTypeSeries:
		s.searchSeries(ctx, requestID, request, result)
	case MediaTypeSeason:
		s.searchSeason(ctx, requestID, request, result)
	case MediaTypeEpisode:
		s.searchEpisode(ctx, requestID, request, result)
	}

	return result
}

func (s *RequestSearcher) searchMovie(ctx context.Context, requestID int64, request *Request, result *SearchForRequestResult) {
	searchResult, err := s.autosearchSvc.SearchMovie(ctx, *request.MediaID, autosearch.SearchSourceRequest)
	if err != nil {
		result.Error = err.Error()
		s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("movie search failed")
		return
	}
	result.Found = searchResult.Found
	result.Downloaded = searchResult.Downloaded
}

func (s *RequestSearcher) searchSeries(ctx context.Context, requestID int64, request *Request, result *SearchForRequestResult) {
	if len(request.RequestedSeasons) > 0 {
		s.searchSpecificSeasons(ctx, requestID, request, result)
		return
	}

	batchResult, err := s.autosearchSvc.SearchSeries(ctx, *request.MediaID, autosearch.SearchSourceRequest)
	if err != nil {
		result.Error = err.Error()
		s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("series search failed")
		return
	}
	result.Found = batchResult.Found > 0
	result.Downloaded = batchResult.Downloaded > 0
}

func (s *RequestSearcher) searchSpecificSeasons(ctx context.Context, requestID int64, request *Request, result *SearchForRequestResult) {
	var totalFound, totalDownloaded int
	for _, seasonNum := range request.RequestedSeasons {
		batchResult, err := s.autosearchSvc.SearchSeason(ctx, *request.MediaID, int(seasonNum), autosearch.SearchSourceRequest)
		if err != nil {
			s.logger.Warn().Err(err).Int64("requestID", requestID).Int64("season", seasonNum).Msg("season search failed")
			continue
		}
		totalFound += batchResult.Found
		totalDownloaded += batchResult.Downloaded
	}
	result.Found = totalFound > 0
	result.Downloaded = totalDownloaded > 0
}

func (s *RequestSearcher) searchSeason(ctx context.Context, requestID int64, request *Request, result *SearchForRequestResult) {
	if request.SeasonNumber == nil {
		result.Error = "season number not specified"
		return
	}
	batchResult, err := s.autosearchSvc.SearchSeason(ctx, *request.MediaID, int(*request.SeasonNumber), autosearch.SearchSourceRequest)
	if err != nil {
		result.Error = err.Error()
		s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("season search failed")
		return
	}
	result.Found = batchResult.Found > 0
	result.Downloaded = batchResult.Downloaded > 0
}

func (s *RequestSearcher) searchEpisode(ctx context.Context, requestID int64, request *Request, result *SearchForRequestResult) {
	searchResult, err := s.autosearchSvc.SearchEpisode(ctx, *request.MediaID, autosearch.SearchSourceRequest)
	if err != nil {
		result.Error = err.Error()
		s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("episode search failed")
		return
	}
	result.Found = searchResult.Found
	result.Downloaded = searchResult.Downloaded
}

func (s *RequestSearcher) updateRequestStatusDownloading(ctx context.Context, requestID int64) {
	if _, err := s.requestsService.UpdateStatus(ctx, requestID, StatusDownloading); err != nil {
		s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("failed to update request status to downloading")
	}
}

func (s *RequestSearcher) logSearchCompletion(requestID int64, result *SearchForRequestResult) {
	s.logger.Info().
		Int64("requestID", requestID).
		Bool("found", result.Found).
		Bool("downloaded", result.Downloaded).
		Msg("search completed for request")
}

func (s *RequestSearcher) SearchForRequestAsync(ctx context.Context, requestID int64) {
	go func() {
		bgCtx := context.Background()
		if _, err := s.SearchForRequest(bgCtx, requestID); err != nil {
			s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("async search failed")
		}
	}()
}

func (s *RequestSearcher) ensureMediaInLibrary(ctx context.Context, request *Request) (int64, error) {
	if s.provisioner == nil {
		return 0, ErrMediaNotInLibrary
	}

	moduleType := s.getModuleType(request.MediaType)
	input := &module.ProvisionInput{
		Title:            request.Title,
		Year:             intFromPtr(request.Year),
		RequestedSeasons: request.RequestedSeasons,
		MonitorFuture:    isMonitorFuture(request.MonitorType),
		AddedBy:          &request.UserID,
	}

	if request.MediaType == MediaTypeSeason && request.SeasonNumber != nil {
		input.RequestedSeasons = []int64{*request.SeasonNumber}
	}

	if request.TmdbID != nil {
		input.ExternalIDs = map[string]int64{"tmdb": *request.TmdbID}
	}
	if request.TvdbID != nil {
		if input.ExternalIDs == nil {
			input.ExternalIDs = make(map[string]int64)
		}
		input.ExternalIDs["tvdb"] = *request.TvdbID
	}

	s.resolveQualityProfile(ctx, input, request.UserID, moduleType)
	return s.provisioner.EnsureInLibrary(ctx, moduleType, input)
}

func (s *RequestSearcher) resolveQualityProfile(ctx context.Context, input *module.ProvisionInput, userID int64, moduleType string) {
	if s.userGetter == nil {
		return
	}
	qpID, err := s.userGetter.GetQualityProfileID(ctx, userID, moduleType)
	if err != nil {
		s.logger.Warn().Err(err).Int64("userID", userID).Msg("failed to get user's quality profile, using default")
	} else if qpID != nil {
		input.QualityProfileID = qpID
		s.logger.Debug().Int64("userID", userID).Int64("qualityProfileID", *qpID).Msg("using user's assigned quality profile")
	}
}

func (s *RequestSearcher) getModuleType(mediaType string) string {
	switch mediaType {
	case MediaTypeMovie:
		return "movie"
	default:
		return "tv"
	}
}

func intFromPtr(p *int64) int {
	if p == nil {
		return 0
	}
	return int(*p)
}

func isMonitorFuture(monitorType *string) bool {
	return monitorType != nil && *monitorType == "future"
}

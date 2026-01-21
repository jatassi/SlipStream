package requests

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/autosearch"
	"github.com/slipstream/slipstream/internal/database/sqlc"
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

type MediaProvisionInput struct {
	TmdbID int64
	TvdbID int64
	Title  string
	Year   int
}

type MediaProvisioner interface {
	EnsureMovieInLibrary(ctx context.Context, input MediaProvisionInput) (int64, error)
	EnsureSeriesInLibrary(ctx context.Context, input MediaProvisionInput) (int64, error)
}

type RequestSearcher struct {
	queries          *sqlc.Queries
	requestsService  *Service
	autosearchSvc    *autosearch.Service
	mediaProvisioner MediaProvisioner
	logger           zerolog.Logger
}

func NewRequestSearcher(
	queries *sqlc.Queries,
	requestsService *Service,
	autosearchSvc *autosearch.Service,
	mediaProvisioner MediaProvisioner,
	logger zerolog.Logger,
) *RequestSearcher {
	return &RequestSearcher{
		queries:          queries,
		requestsService:  requestsService,
		autosearchSvc:    autosearchSvc,
		mediaProvisioner: mediaProvisioner,
		logger:           logger.With().Str("component", "portal-request-searcher").Logger(),
	}
}

func (s *RequestSearcher) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)
}

func (s *RequestSearcher) SearchForRequest(ctx context.Context, requestID int64) (*SearchForRequestResult, error) {
	s.logger.Info().Int64("requestID", requestID).Msg("SearchForRequest called")

	request, err := s.requestsService.Get(ctx, requestID)
	if err != nil {
		s.logger.Error().Err(err).Int64("requestID", requestID).Msg("failed to get request")
		return nil, err
	}

	s.logger.Info().
		Int64("requestID", requestID).
		Str("title", request.Title).
		Str("mediaType", request.MediaType).
		Interface("mediaID", request.MediaID).
		Interface("tmdbID", request.TmdbID).
		Interface("tvdbID", request.TvdbID).
		Msg("got request details")

	// If MediaID is not set, try to find or create the media in library
	if request.MediaID == nil {
		s.logger.Info().Int64("requestID", requestID).Msg("MediaID is nil, attempting to ensure media in library")

		mediaID, err := s.ensureMediaInLibrary(ctx, request)
		if err != nil {
			s.logger.Error().Err(err).Int64("requestID", requestID).Msg("failed to ensure media in library")
			return nil, fmt.Errorf("failed to provision media: %w", err)
		}

		s.logger.Info().Int64("requestID", requestID).Int64("mediaID", mediaID).Msg("ensureMediaInLibrary returned mediaID")

		// Link the request to the media
		if _, err := s.requestsService.LinkMedia(ctx, requestID, mediaID); err != nil {
			s.logger.Error().Err(err).Int64("requestID", requestID).Int64("mediaID", mediaID).Msg("failed to link request to media")
			return nil, fmt.Errorf("failed to link request to media: %w", err)
		}

		// Update local request object with the new mediaID
		request.MediaID = &mediaID
		s.logger.Info().Int64("requestID", requestID).Int64("mediaID", mediaID).Msg("linked request to media")
	}

	result := &SearchForRequestResult{}

	switch request.MediaType {
	case MediaTypeMovie:
		searchResult, err := s.autosearchSvc.SearchMovie(ctx, *request.MediaID, autosearch.SearchSourceRequest)
		if err != nil {
			result.Error = err.Error()
			s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("movie search failed")
			return result, nil
		}
		result.Found = searchResult.Found
		result.Downloaded = searchResult.Downloaded

	case MediaTypeSeries:
		// If specific seasons were requested, search for those seasons individually
		if len(request.RequestedSeasons) > 0 {
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
		} else {
			// Search all missing episodes in the series
			batchResult, err := s.autosearchSvc.SearchSeries(ctx, *request.MediaID, autosearch.SearchSourceRequest)
			if err != nil {
				result.Error = err.Error()
				s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("series search failed")
				return result, nil
			}
			result.Found = batchResult.Found > 0
			result.Downloaded = batchResult.Downloaded > 0
		}

	case MediaTypeSeason:
		if request.SeasonNumber == nil {
			return nil, errors.New("season number not specified")
		}
		batchResult, err := s.autosearchSvc.SearchSeason(ctx, *request.MediaID, int(*request.SeasonNumber), autosearch.SearchSourceRequest)
		if err != nil {
			result.Error = err.Error()
			s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("season search failed")
			return result, nil
		}
		result.Found = batchResult.Found > 0
		result.Downloaded = batchResult.Downloaded > 0

	case MediaTypeEpisode:
		searchResult, err := s.autosearchSvc.SearchEpisode(ctx, *request.MediaID, autosearch.SearchSourceRequest)
		if err != nil {
			result.Error = err.Error()
			s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("episode search failed")
			return result, nil
		}
		result.Found = searchResult.Found
		result.Downloaded = searchResult.Downloaded

	default:
		return nil, fmt.Errorf("unsupported media type: %s", request.MediaType)
	}

	if result.Downloaded {
		if _, err := s.requestsService.UpdateStatus(ctx, requestID, StatusDownloading); err != nil {
			s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("failed to update request status to downloading")
		}
	}

	s.logger.Info().
		Int64("requestID", requestID).
		Bool("found", result.Found).
		Bool("downloaded", result.Downloaded).
		Msg("search completed for request")

	return result, nil
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
	if s.mediaProvisioner == nil {
		return 0, ErrMediaNotInLibrary
	}

	year := 0
	if request.Year != nil {
		year = int(*request.Year)
	}

	input := MediaProvisionInput{
		Title: request.Title,
		Year:  year,
	}

	switch request.MediaType {
	case MediaTypeMovie:
		if request.TmdbID == nil {
			return 0, errors.New("movie request missing tmdbID")
		}
		input.TmdbID = *request.TmdbID
		return s.mediaProvisioner.EnsureMovieInLibrary(ctx, input)

	case MediaTypeSeries, MediaTypeSeason, MediaTypeEpisode:
		if request.TvdbID == nil {
			return 0, errors.New("series request missing tvdbID")
		}
		input.TvdbID = *request.TvdbID
		return s.mediaProvisioner.EnsureSeriesInLibrary(ctx, input)

	default:
		return 0, fmt.Errorf("unsupported media type: %s", request.MediaType)
	}
}

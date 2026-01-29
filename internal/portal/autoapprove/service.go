package autoapprove

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/portal/quota"
	"github.com/slipstream/slipstream/internal/portal/requests"
	"github.com/slipstream/slipstream/internal/portal/users"
)

var (
	ErrAutoApproveDisabled = errors.New("auto-approve is disabled")
	ErrQuotaExceeded       = errors.New("quota exceeded")
)

type AutoApproveResult struct {
	AutoApproved  bool `json:"autoApproved"`
	QuotaExceeded bool `json:"quotaExceeded"`
	SearchStarted bool `json:"searchStarted"`
}

type RequestSearcher interface {
	SearchForRequestAsync(ctx context.Context, requestID int64)
}

type Service struct {
	queries          *sqlc.Queries
	usersService     *users.Service
	qualityService   *quality.Service
	quotaService     *quota.Service
	requestsService  *requests.Service
	requestSearcher  RequestSearcher
	logger           zerolog.Logger
}

func NewService(
	queries *sqlc.Queries,
	usersService *users.Service,
	qualityService *quality.Service,
	quotaService *quota.Service,
	requestsService *requests.Service,
	logger zerolog.Logger,
) *Service {
	return &Service{
		queries:         queries,
		usersService:    usersService,
		qualityService:  qualityService,
		quotaService:    quotaService,
		requestsService: requestsService,
		logger:          logger.With().Str("component", "portal-autoapprove").Logger(),
	}
}

func (s *Service) SetDB(queries *sqlc.Queries) {
	s.queries = queries
}

func (s *Service) SetRequestSearcher(searcher RequestSearcher) {
	s.requestSearcher = searcher
}

func (s *Service) ShouldAutoApprove(ctx context.Context, userID int64, qualityProfileID *int64) (bool, error) {
	user, err := s.usersService.Get(ctx, userID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("userID", userID).Msg("failed to get user for auto-approve check")
		return false, err
	}

	if user.AutoApprove {
		s.logger.Debug().Int64("userID", userID).Msg("user has auto-approve enabled")
		return true, nil
	}

	if qualityProfileID != nil {
		profile, err := s.qualityService.Get(ctx, *qualityProfileID)
		if err != nil {
			s.logger.Warn().Err(err).Int64("profileID", *qualityProfileID).Msg("failed to get quality profile")
			return false, nil
		}
		if profile.AllowAutoApprove {
			s.logger.Debug().Int64("userID", userID).Int64("profileID", *qualityProfileID).Msg("quality profile allows auto-approve")
			return true, nil
		}
	}

	s.logger.Debug().Int64("userID", userID).Bool("userAutoApprove", user.AutoApprove).
		Bool("hasQualityProfile", qualityProfileID != nil).
		Msg("auto-approve not enabled for user")
	return false, nil
}

func (s *Service) ProcessAutoApprove(ctx context.Context, request *requests.Request) (*AutoApproveResult, error) {
	result := &AutoApproveResult{}
	s.logger.Debug().Int64("requestID", request.ID).Int64("userID", request.UserID).Msg("processing auto-approve")

	user, err := s.usersService.Get(ctx, request.UserID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("userID", request.UserID).Msg("failed to get user for auto-approve")
		return nil, err
	}

	shouldAutoApprove, err := s.ShouldAutoApprove(ctx, request.UserID, user.QualityProfileID)
	if err != nil {
		return nil, err
	}

	if !shouldAutoApprove {
		return result, nil
	}

	mediaType := s.getQuotaMediaType(request.MediaType)
	canConsume, err := s.quotaService.CheckQuota(ctx, request.UserID, mediaType)
	if err != nil {
		return nil, err
	}

	if !canConsume {
		result.QuotaExceeded = true
		s.logger.Info().
			Int64("requestID", request.ID).
			Int64("userID", request.UserID).
			Str("mediaType", request.MediaType).
			Msg("auto-approve blocked by quota")
		return result, nil
	}

	if err := s.quotaService.ConsumeQuota(ctx, request.UserID, mediaType); err != nil {
		return nil, err
	}

	if _, err := s.requestsService.AutoApprove(ctx, request.ID); err != nil {
		return nil, err
	}

	result.AutoApproved = true
	s.logger.Info().
		Int64("requestID", request.ID).
		Int64("userID", request.UserID).
		Str("mediaType", request.MediaType).
		Msg("request auto-approved")

	if s.requestSearcher != nil {
		s.requestSearcher.SearchForRequestAsync(ctx, request.ID)
		result.SearchStarted = true
	}

	return result, nil
}

func (s *Service) getQuotaMediaType(requestMediaType string) string {
	switch requestMediaType {
	case requests.MediaTypeMovie:
		return "movie"
	case requests.MediaTypeSeries, requests.MediaTypeSeason:
		return "season"
	case requests.MediaTypeEpisode:
		return "episode"
	default:
		return "movie"
	}
}

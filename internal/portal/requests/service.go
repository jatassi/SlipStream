package requests

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

var (
	ErrRequestNotFound    = errors.New("request not found")
	ErrAlreadyRequested   = errors.New("item already requested")
	ErrCannotCancel       = errors.New("cannot cancel request")
	ErrNotOwner           = errors.New("not the owner of this request")
	ErrInvalidStatus      = errors.New("invalid status transition")
	ErrInvalidMediaType   = errors.New("invalid media type")
)

const (
	MediaTypeMovie   = "movie"
	MediaTypeSeries  = "series"
	MediaTypeSeason  = "season"
	MediaTypeEpisode = "episode"

	StatusPending     = "pending"
	StatusApproved    = "approved"
	StatusDenied      = "denied"
	StatusDownloading = "downloading"
	StatusAvailable   = "available"
)

type Request struct {
	ID               int64      `json:"id"`
	UserID           int64      `json:"userId"`
	MediaType        string     `json:"mediaType"`
	TmdbID           *int64     `json:"tmdbId,omitempty"`
	TvdbID           *int64     `json:"tvdbId,omitempty"`
	Title            string     `json:"title"`
	Year             *int64     `json:"year,omitempty"`
	SeasonNumber     *int64     `json:"seasonNumber,omitempty"`
	EpisodeNumber    *int64     `json:"episodeNumber,omitempty"`
	Status           string     `json:"status"`
	MonitorType      *string    `json:"monitorType,omitempty"`
	DeniedReason     *string    `json:"deniedReason,omitempty"`
	ApprovedAt       *time.Time `json:"approvedAt,omitempty"`
	ApprovedBy       *int64     `json:"approvedBy,omitempty"`
	MediaID          *int64     `json:"mediaId,omitempty"`
	TargetSlotID     *int64     `json:"targetSlotId,omitempty"`
	PosterUrl        *string    `json:"posterUrl,omitempty"`
	RequestedSeasons []int64    `json:"requestedSeasons,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type CreateInput struct {
	MediaType        string
	TmdbID           *int64
	TvdbID           *int64
	Title            string
	Year             *int64
	SeasonNumber     *int64
	EpisodeNumber    *int64
	MonitorType      *string
	TargetSlotID     *int64
	PosterUrl        *string
	RequestedSeasons []int64
}

type ApprovalAction string

const (
	ApprovalActionOnly       ApprovalAction = "approve_only"
	ApprovalActionAutoSearch ApprovalAction = "auto_search"
	ApprovalActionManual     ApprovalAction = "manual_search"
)

type ListFilters struct {
	Status    *string
	MediaType *string
	UserID    *int64
}

// NotificationDispatcher dispatches notifications for request status changes
type NotificationDispatcher interface {
	NotifyRequestApproved(ctx context.Context, request *Request, watcherUserIDs []int64)
	NotifyRequestDenied(ctx context.Context, request *Request, watcherUserIDs []int64)
	NotifyRequestAvailable(ctx context.Context, request *Request, watcherUserIDs []int64)
}

type Service struct {
	queries              *sqlc.Queries
	logger               zerolog.Logger
	broadcaster          *EventBroadcaster
	notifDispatcher      NotificationDispatcher
	watchersService      *WatchersService
}

func NewService(queries *sqlc.Queries, logger zerolog.Logger) *Service {
	return &Service{
		queries: queries,
		logger:  logger.With().Str("component", "portal-requests").Logger(),
	}
}

func (s *Service) SetDB(queries *sqlc.Queries) {
	s.queries = queries
}

func (s *Service) SetBroadcaster(broadcaster *EventBroadcaster) {
	s.broadcaster = broadcaster
}

func (s *Service) SetNotificationDispatcher(dispatcher NotificationDispatcher) {
	s.notifDispatcher = dispatcher
}

func (s *Service) SetWatchersService(watchersSvc *WatchersService) {
	s.watchersService = watchersSvc
}

func (s *Service) getWatcherUserIDs(ctx context.Context, requestID int64) []int64 {
	if s.watchersService == nil {
		return nil
	}
	ids, err := s.watchersService.GetWatcherUserIDs(ctx, requestID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("requestID", requestID).Msg("failed to get watcher user IDs")
		return nil
	}
	return ids
}

func (s *Service) Create(ctx context.Context, userID int64, input CreateInput) (*Request, error) {
	if !isValidMediaType(input.MediaType) {
		return nil, ErrInvalidMediaType
	}

	existing, err := s.checkExistingRequest(ctx, input)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyRequested
	}

	req, err := s.queries.CreateRequest(ctx, sqlc.CreateRequestParams{
		UserID:           userID,
		MediaType:        input.MediaType,
		TmdbID:           toNullInt64(input.TmdbID),
		TvdbID:           toNullInt64(input.TvdbID),
		Title:            input.Title,
		Year:             toNullInt64(input.Year),
		SeasonNumber:     toNullInt64(input.SeasonNumber),
		EpisodeNumber:    toNullInt64(input.EpisodeNumber),
		Status:           StatusPending,
		MonitorType:      toNullString(input.MonitorType),
		TargetSlotID:     toNullInt64(input.TargetSlotID),
		PosterUrl:        toNullString(input.PosterUrl),
		RequestedSeasons: seasonsToJSON(input.RequestedSeasons),
	})
	if err != nil {
		s.logger.Error().Err(err).Int64("userID", userID).Str("title", input.Title).Msg("failed to create request")
		return nil, err
	}

	result := toRequest(req)
	if s.broadcaster != nil {
		s.broadcaster.BroadcastRequestCreated(result)
	}

	return result, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Request, error) {
	req, err := s.queries.GetRequest(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}
	return toRequest(req), nil
}

func (s *Service) GetByTmdbID(ctx context.Context, tmdbID int64, mediaType string) (*Request, error) {
	req, err := s.queries.GetRequestByTmdbID(ctx, sqlc.GetRequestByTmdbIDParams{
		TmdbID:    sql.NullInt64{Int64: tmdbID, Valid: true},
		MediaType: mediaType,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}
	return toRequest(req), nil
}

func (s *Service) GetByTvdbID(ctx context.Context, tvdbID int64, mediaType string) (*Request, error) {
	req, err := s.queries.GetRequestByTvdbID(ctx, sqlc.GetRequestByTvdbIDParams{
		TvdbID:    sql.NullInt64{Int64: tvdbID, Valid: true},
		MediaType: mediaType,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}
	return toRequest(req), nil
}

func (s *Service) GetByTvdbIDAndSeason(ctx context.Context, tvdbID int64, seasonNumber int64) (*Request, error) {
	req, err := s.queries.GetRequestByTvdbIDAndSeason(ctx, sqlc.GetRequestByTvdbIDAndSeasonParams{
		TvdbID:       sql.NullInt64{Int64: tvdbID, Valid: true},
		SeasonNumber: sql.NullInt64{Int64: seasonNumber, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}
	return toRequest(req), nil
}

func (s *Service) GetByTvdbIDAndEpisode(ctx context.Context, tvdbID int64, seasonNumber, episodeNumber int64) (*Request, error) {
	req, err := s.queries.GetRequestByTvdbIDAndEpisode(ctx, sqlc.GetRequestByTvdbIDAndEpisodeParams{
		TvdbID:        sql.NullInt64{Int64: tvdbID, Valid: true},
		SeasonNumber:  sql.NullInt64{Int64: seasonNumber, Valid: true},
		EpisodeNumber: sql.NullInt64{Int64: episodeNumber, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}
	return toRequest(req), nil
}

func (s *Service) List(ctx context.Context, filters ListFilters) ([]*Request, error) {
	var requests []*sqlc.Request
	var err error

	if filters.UserID != nil && filters.Status != nil {
		requests, err = s.queries.ListRequestsByUserAndStatus(ctx, sqlc.ListRequestsByUserAndStatusParams{
			UserID: *filters.UserID,
			Status: *filters.Status,
		})
	} else if filters.UserID != nil {
		requests, err = s.queries.ListRequestsByUser(ctx, *filters.UserID)
	} else if filters.Status != nil {
		requests, err = s.queries.ListRequestsByStatus(ctx, *filters.Status)
	} else if filters.MediaType != nil {
		requests, err = s.queries.ListRequestsByMediaType(ctx, *filters.MediaType)
	} else {
		requests, err = s.queries.ListRequests(ctx)
	}

	if err != nil {
		return nil, err
	}

	result := make([]*Request, len(requests))
	for i, r := range requests {
		result[i] = toRequest(r)
	}
	return result, nil
}

func (s *Service) ListByUser(ctx context.Context, userID int64) ([]*Request, error) {
	return s.List(ctx, ListFilters{UserID: &userID})
}

func (s *Service) ListPending(ctx context.Context) ([]*Request, error) {
	requests, err := s.queries.ListPendingRequests(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Request, len(requests))
	for i, r := range requests {
		result[i] = toRequest(r)
	}
	return result, nil
}

func (s *Service) Cancel(ctx context.Context, id int64, userID int64) error {
	req, err := s.queries.GetRequest(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRequestNotFound
		}
		return err
	}

	if req.UserID != userID {
		return ErrNotOwner
	}

	if req.Status != StatusPending {
		return ErrCannotCancel
	}

	if err := s.queries.DeleteRequest(ctx, id); err != nil {
		return err
	}

	if s.broadcaster != nil {
		s.broadcaster.BroadcastRequestDeleted(toRequest(req))
	}

	return nil
}

func (s *Service) Approve(ctx context.Context, id int64, approverID int64, _ ApprovalAction) (*Request, error) {
	req, err := s.queries.ApproveRequest(ctx, sqlc.ApproveRequestParams{
		ID:         id,
		ApprovedBy: sql.NullInt64{Int64: approverID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}

	result := toRequest(req)
	if s.broadcaster != nil {
		s.broadcaster.BroadcastRequestUpdated(result, StatusPending)
	}

	// Dispatch notification
	if s.notifDispatcher != nil {
		watcherIDs := s.getWatcherUserIDs(ctx, id)
		go s.notifDispatcher.NotifyRequestApproved(context.Background(), result, watcherIDs)
	}

	s.logger.Info().Int64("requestID", id).Int64("approverID", approverID).Msg("request approved")
	return result, nil
}

func (s *Service) AutoApprove(ctx context.Context, id int64) (*Request, error) {
	req, err := s.queries.ApproveRequest(ctx, sqlc.ApproveRequestParams{
		ID:         id,
		ApprovedBy: sql.NullInt64{Valid: false},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}

	result := toRequest(req)
	if s.broadcaster != nil {
		s.broadcaster.BroadcastRequestUpdated(result, StatusPending)
	}

	// Dispatch notification
	if s.notifDispatcher != nil {
		watcherIDs := s.getWatcherUserIDs(ctx, id)
		go s.notifDispatcher.NotifyRequestApproved(context.Background(), result, watcherIDs)
	}

	s.logger.Info().Int64("requestID", id).Msg("request auto-approved")
	return result, nil
}

func (s *Service) Deny(ctx context.Context, id int64, reason *string) (*Request, error) {
	req, err := s.queries.DenyRequest(ctx, sqlc.DenyRequestParams{
		ID:           id,
		DeniedReason: toNullString(reason),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}

	result := toRequest(req)
	if s.broadcaster != nil {
		s.broadcaster.BroadcastRequestUpdated(result, StatusPending)
	}

	// Dispatch notification
	if s.notifDispatcher != nil {
		watcherIDs := s.getWatcherUserIDs(ctx, id)
		go s.notifDispatcher.NotifyRequestDenied(context.Background(), result, watcherIDs)
	}

	s.logger.Info().Int64("requestID", id).Msg("request denied")
	return result, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id int64, status string) (*Request, error) {
	if !isValidStatus(status) {
		return nil, ErrInvalidStatus
	}

	existing, err := s.queries.GetRequest(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}
	previousStatus := existing.Status

	req, err := s.queries.UpdateRequestStatus(ctx, sqlc.UpdateRequestStatusParams{
		ID:     id,
		Status: status,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}

	result := toRequest(req)
	if s.broadcaster != nil {
		s.broadcaster.BroadcastRequestUpdated(result, previousStatus)
	}

	return result, nil
}

func (s *Service) LinkMedia(ctx context.Context, id int64, mediaID int64) (*Request, error) {
	req, err := s.queries.LinkRequestToMedia(ctx, sqlc.LinkRequestToMediaParams{
		ID:      id,
		MediaID: sql.NullInt64{Int64: mediaID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRequestNotFound
		}
		return nil, err
	}

	s.logger.Info().Int64("requestID", id).Int64("mediaID", mediaID).Msg("request linked to media")
	return toRequest(req), nil
}

func (s *Service) BatchApprove(ctx context.Context, ids []int64, approverID int64, action ApprovalAction) ([]*Request, error) {
	results := make([]*Request, 0, len(ids))
	for _, id := range ids {
		req, err := s.Approve(ctx, id, approverID, action)
		if err != nil {
			s.logger.Warn().Err(err).Int64("requestID", id).Msg("failed to approve request in batch")
			continue
		}
		results = append(results, req)
	}
	return results, nil
}

func (s *Service) BatchDeny(ctx context.Context, ids []int64, reason *string) ([]*Request, error) {
	results := make([]*Request, 0, len(ids))
	for _, id := range ids {
		req, err := s.Deny(ctx, id, reason)
		if err != nil {
			s.logger.Warn().Err(err).Int64("requestID", id).Msg("failed to deny request in batch")
			continue
		}
		results = append(results, req)
	}
	return results, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.queries.DeleteRequest(ctx, id)
}

func (s *Service) checkExistingRequest(ctx context.Context, input CreateInput) (*Request, error) {
	var req *sqlc.Request
	var err error

	switch input.MediaType {
	case MediaTypeMovie:
		if input.TmdbID != nil {
			req, err = s.queries.GetRequestByTmdbID(ctx, sqlc.GetRequestByTmdbIDParams{
				TmdbID:    sql.NullInt64{Int64: *input.TmdbID, Valid: true},
				MediaType: MediaTypeMovie,
			})
		}
	case MediaTypeSeries:
		if input.TvdbID != nil {
			req, err = s.queries.GetRequestByTvdbID(ctx, sqlc.GetRequestByTvdbIDParams{
				TvdbID:    sql.NullInt64{Int64: *input.TvdbID, Valid: true},
				MediaType: MediaTypeSeries,
			})
		}
	case MediaTypeSeason:
		if input.TvdbID != nil && input.SeasonNumber != nil {
			req, err = s.queries.GetRequestByTvdbIDAndSeason(ctx, sqlc.GetRequestByTvdbIDAndSeasonParams{
				TvdbID:       sql.NullInt64{Int64: *input.TvdbID, Valid: true},
				SeasonNumber: sql.NullInt64{Int64: *input.SeasonNumber, Valid: true},
			})
		}
	case MediaTypeEpisode:
		if input.TvdbID != nil && input.SeasonNumber != nil && input.EpisodeNumber != nil {
			req, err = s.queries.GetRequestByTvdbIDAndEpisode(ctx, sqlc.GetRequestByTvdbIDAndEpisodeParams{
				TvdbID:        sql.NullInt64{Int64: *input.TvdbID, Valid: true},
				SeasonNumber:  sql.NullInt64{Int64: *input.SeasonNumber, Valid: true},
				EpisodeNumber: sql.NullInt64{Int64: *input.EpisodeNumber, Valid: true},
			})
		}
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if req != nil {
		return toRequest(req), nil
	}
	return nil, nil
}

func isValidMediaType(mediaType string) bool {
	switch mediaType {
	case MediaTypeMovie, MediaTypeSeries, MediaTypeSeason, MediaTypeEpisode:
		return true
	}
	return false
}

func isValidStatus(status string) bool {
	switch status {
	case StatusPending, StatusApproved, StatusDenied, StatusDownloading, StatusAvailable:
		return true
	}
	return false
}

func toNullInt64(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func toNullString(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

func seasonsToJSON(seasons []int64) sql.NullString {
	if len(seasons) == 0 {
		return sql.NullString{}
	}
	data, err := json.Marshal(seasons)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(data), Valid: true}
}

func seasonsFromJSON(s sql.NullString) []int64 {
	if !s.Valid || s.String == "" {
		return nil
	}
	var seasons []int64
	if err := json.Unmarshal([]byte(s.String), &seasons); err != nil {
		return nil
	}
	return seasons
}

func toRequest(r *sqlc.Request) *Request {
	req := &Request{
		ID:        r.ID,
		UserID:    r.UserID,
		MediaType: r.MediaType,
		Title:     r.Title,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}

	if r.TmdbID.Valid {
		req.TmdbID = &r.TmdbID.Int64
	}
	if r.TvdbID.Valid {
		req.TvdbID = &r.TvdbID.Int64
	}
	if r.Year.Valid {
		req.Year = &r.Year.Int64
	}
	if r.SeasonNumber.Valid {
		req.SeasonNumber = &r.SeasonNumber.Int64
	}
	if r.EpisodeNumber.Valid {
		req.EpisodeNumber = &r.EpisodeNumber.Int64
	}
	if r.MonitorType.Valid {
		req.MonitorType = &r.MonitorType.String
	}
	if r.DeniedReason.Valid {
		req.DeniedReason = &r.DeniedReason.String
	}
	if r.ApprovedAt.Valid {
		req.ApprovedAt = &r.ApprovedAt.Time
	}
	if r.ApprovedBy.Valid {
		req.ApprovedBy = &r.ApprovedBy.Int64
	}
	if r.MediaID.Valid {
		req.MediaID = &r.MediaID.Int64
	}
	if r.TargetSlotID.Valid {
		req.TargetSlotID = &r.TargetSlotID.Int64
	}
	if r.PosterUrl.Valid {
		req.PosterUrl = &r.PosterUrl.String
	}
	req.RequestedSeasons = seasonsFromJSON(r.RequestedSeasons)

	return req
}

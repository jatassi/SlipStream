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
	ErrRequestNotFound  = errors.New("request not found")
	ErrAlreadyRequested = errors.New("item already requested")
	ErrCannotCancel     = errors.New("cannot cancel request")
	ErrNotOwner         = errors.New("not the owner of this request")
	ErrInvalidStatus    = errors.New("invalid status transition")
	ErrInvalidMediaType = errors.New("invalid media type")
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
	StatusFailed      = "failed"
	StatusAvailable   = "available"
)

type Request struct {
	ID               int64      `json:"id"`
	UserID           int64      `json:"userId"`
	MediaType        string     `json:"mediaType"`
	TmdbID           *int64     `json:"tmdbId"`
	TvdbID           *int64     `json:"tvdbId"`
	Title            string     `json:"title"`
	Year             *int64     `json:"year"`
	SeasonNumber     *int64     `json:"seasonNumber"`
	EpisodeNumber    *int64     `json:"episodeNumber"`
	Status           string     `json:"status"`
	MonitorType      *string    `json:"monitorType,omitempty"`
	DeniedReason     *string    `json:"deniedReason,omitempty"`
	ApprovedAt       *time.Time `json:"approvedAt,omitempty"`
	ApprovedBy       *int64     `json:"approvedBy,omitempty"`
	MediaID          *int64     `json:"mediaId"`
	TargetSlotID     *int64     `json:"targetSlotId"`
	PosterURL        *string    `json:"posterUrl,omitempty"`
	RequestedSeasons []int64    `json:"requestedSeasons"`
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
	PosterURL        *string
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
	queries         *sqlc.Queries
	logger          *zerolog.Logger
	broadcaster     *EventBroadcaster
	notifDispatcher NotificationDispatcher
	watchersService *WatchersService
}

func NewService(queries *sqlc.Queries, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "portal-requests").Logger()
	return &Service{
		queries: queries,
		logger:  &subLogger,
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

func (s *Service) Create(ctx context.Context, userID int64, input *CreateInput) (*Request, error) {
	if !isValidMediaType(input.MediaType) {
		return nil, ErrInvalidMediaType
	}

	existing, err := s.checkExistingRequest(ctx, input)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
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
		PosterUrl:        toNullString(input.PosterURL),
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

func (s *Service) GetByTvdbIDAndSeason(ctx context.Context, tvdbID, seasonNumber int64) (*Request, error) {
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

func (s *Service) GetByTvdbIDAndEpisode(ctx context.Context, tvdbID, seasonNumber, episodeNumber int64) (*Request, error) {
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

	switch {
	case filters.UserID != nil && filters.Status != nil:
		requests, err = s.queries.ListRequestsByUserAndStatus(ctx, sqlc.ListRequestsByUserAndStatusParams{
			UserID: *filters.UserID,
			Status: *filters.Status,
		})
	case filters.UserID != nil:
		requests, err = s.queries.ListRequestsByUser(ctx, *filters.UserID)
	case filters.Status != nil:
		requests, err = s.queries.ListRequestsByStatus(ctx, *filters.Status)
	case filters.MediaType != nil:
		requests, err = s.queries.ListRequestsByMediaType(ctx, *filters.MediaType)
	default:
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

func (s *Service) Cancel(ctx context.Context, id, userID int64) error {
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

func (s *Service) Approve(ctx context.Context, id, approverID int64, _ ApprovalAction) (*Request, error) {
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

func (s *Service) LinkMedia(ctx context.Context, id, mediaID int64) (*Request, error) {
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

func (s *Service) checkExistingRequest(ctx context.Context, input *CreateInput) (*Request, error) {
	var req *sqlc.Request
	var err error

	switch input.MediaType {
	case MediaTypeMovie:
		req, err = s.checkExistingMovie(ctx, input)
	case MediaTypeSeries:
		req, err = s.checkExistingSeries(ctx, input)
	case MediaTypeSeason:
		req, err = s.checkExistingSeason(ctx, input)
	case MediaTypeEpisode:
		req, err = s.checkExistingEpisode(ctx, input)
	}

	return s.handleExistingRequestResult(req, err)
}

func (s *Service) checkExistingMovie(ctx context.Context, input *CreateInput) (*sqlc.Request, error) {
	if input.TmdbID == nil {
		return nil, sql.ErrNoRows
	}
	return s.queries.GetRequestByTmdbID(ctx, sqlc.GetRequestByTmdbIDParams{
		TmdbID:    sql.NullInt64{Int64: *input.TmdbID, Valid: true},
		MediaType: MediaTypeMovie,
	})
}

func (s *Service) checkExistingSeries(ctx context.Context, input *CreateInput) (*sqlc.Request, error) {
	if input.TvdbID == nil {
		return nil, sql.ErrNoRows
	}

	// Same-type check first
	req, err := s.queries.GetRequestByTvdbID(ctx, sqlc.GetRequestByTvdbIDParams{
		TvdbID:    sql.NullInt64{Int64: *input.TvdbID, Valid: true},
		MediaType: MediaTypeSeries,
	})
	if err == nil && req != nil {
		return req, nil
	}

	// Cross-type check: if ALL requested seasons are already covered by existing requests
	if len(input.RequestedSeasons) > 0 {
		coveredReq, err := s.checkCrossTypeCoverage(ctx, *input.TvdbID, input.RequestedSeasons)
		if err == nil && coveredReq != nil {
			return coveredReq, nil
		}
	}

	return nil, sql.ErrNoRows
}

func (s *Service) checkExistingSeason(ctx context.Context, input *CreateInput) (*sqlc.Request, error) {
	if input.TvdbID == nil || input.SeasonNumber == nil {
		return nil, sql.ErrNoRows
	}

	// Same-type check first
	req, err := s.queries.GetRequestByTvdbIDAndSeason(ctx, sqlc.GetRequestByTvdbIDAndSeasonParams{
		TvdbID:       sql.NullInt64{Int64: *input.TvdbID, Valid: true},
		SeasonNumber: sql.NullInt64{Int64: *input.SeasonNumber, Valid: true},
	})
	if err == nil && req != nil {
		return req, nil
	}

	// Cross-type check: is this season covered by an existing series request?
	coveredReq, err := s.checkCrossTypeCoverage(ctx, *input.TvdbID, []int64{*input.SeasonNumber})
	if err == nil && coveredReq != nil {
		return coveredReq, nil
	}

	return nil, sql.ErrNoRows
}

func (s *Service) checkExistingEpisode(ctx context.Context, input *CreateInput) (*sqlc.Request, error) {
	if input.TvdbID == nil || input.SeasonNumber == nil || input.EpisodeNumber == nil {
		return nil, sql.ErrNoRows
	}
	return s.queries.GetRequestByTvdbIDAndEpisode(ctx, sqlc.GetRequestByTvdbIDAndEpisodeParams{
		TvdbID:        sql.NullInt64{Int64: *input.TvdbID, Valid: true},
		SeasonNumber:  sql.NullInt64{Int64: *input.SeasonNumber, Valid: true},
		EpisodeNumber: sql.NullInt64{Int64: *input.EpisodeNumber, Valid: true},
	})
}

func (s *Service) checkCrossTypeCoverage(ctx context.Context, tvdbID int64, requestedSeasons []int64) (*sqlc.Request, error) {
	reqs, err := s.queries.FindRequestsCoveringSeasons(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if err != nil {
		return nil, err
	}

	coveredSeasons, coveringReq := buildCoveredSeasonsMap(reqs)

	for _, sn := range requestedSeasons {
		if !coveredSeasons[sn] {
			return nil, sql.ErrNoRows
		}
	}

	if coveringReq != nil {
		return coveringReq, nil
	}

	return nil, sql.ErrNoRows
}

func buildCoveredSeasonsMap(reqs []*sqlc.Request) (map[int64]bool, *sqlc.Request) {
	coveredSeasons := make(map[int64]bool)
	var coveringReq *sqlc.Request
	for _, r := range reqs {
		if r.MediaType == MediaTypeSeries {
			for _, sn := range seasonsFromJSON(r.RequestedSeasons) {
				coveredSeasons[sn] = true
			}
		} else if r.MediaType == MediaTypeSeason && r.SeasonNumber.Valid {
			coveredSeasons[r.SeasonNumber.Int64] = true
		}
		if coveringReq == nil {
			coveringReq = r
		}
	}
	return coveredSeasons, coveringReq
}

func (s *Service) handleExistingRequestResult(req *sqlc.Request, err error) (*Request, error) {
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	if req == nil {
		return nil, sql.ErrNoRows
	}
	return toRequest(req), nil
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
	case StatusPending, StatusApproved, StatusDenied, StatusDownloading, StatusFailed, StatusAvailable:
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
		return []int64{}
	}
	var seasons []int64
	if err := json.Unmarshal([]byte(s.String), &seasons); err != nil {
		return []int64{}
	}
	if seasons == nil {
		return []int64{}
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

	assignNullableInt64(&req.TmdbID, r.TmdbID)
	assignNullableInt64(&req.TvdbID, r.TvdbID)
	assignNullableInt64(&req.Year, r.Year)
	assignNullableInt64(&req.SeasonNumber, r.SeasonNumber)
	assignNullableInt64(&req.EpisodeNumber, r.EpisodeNumber)
	assignNullableInt64(&req.ApprovedBy, r.ApprovedBy)
	assignNullableInt64(&req.MediaID, r.MediaID)
	assignNullableInt64(&req.TargetSlotID, r.TargetSlotID)

	assignNullableString(&req.MonitorType, r.MonitorType)
	assignNullableString(&req.DeniedReason, r.DeniedReason)
	assignNullableString(&req.PosterURL, r.PosterUrl)

	if r.ApprovedAt.Valid {
		req.ApprovedAt = &r.ApprovedAt.Time
	}

	req.RequestedSeasons = seasonsFromJSON(r.RequestedSeasons)

	return req
}

func assignNullableInt64(dest **int64, src sql.NullInt64) {
	if src.Valid {
		*dest = &src.Int64
	}
}

func assignNullableString(dest **string, src sql.NullString) {
	if src.Valid {
		*dest = &src.String
	}
}

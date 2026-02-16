package requests

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

var (
	ErrAlreadyWatching = errors.New("already watching this request")
	ErrNotWatching     = errors.New("not watching this request")
)

type Watcher struct {
	RequestID int64     `json:"requestId"`
	UserID    int64     `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
}

type WatchersService struct {
	queries *sqlc.Queries
	logger  *zerolog.Logger
}

func NewWatchersService(queries *sqlc.Queries, logger *zerolog.Logger) *WatchersService {
	subLogger := logger.With().Str("component", "portal-watchers").Logger()
	return &WatchersService{
		queries: queries,
		logger:  &subLogger,
	}
}

func (s *WatchersService) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)
}

func (s *WatchersService) Watch(ctx context.Context, requestID, userID int64) (*Watcher, error) {
	isWatching, err := s.queries.IsWatchingRequest(ctx, sqlc.IsWatchingRequestParams{
		RequestID: requestID,
		UserID:    userID,
	})
	if err != nil {
		return nil, err
	}
	if isWatching == 1 {
		return nil, ErrAlreadyWatching
	}

	watcher, err := s.queries.CreateRequestWatcher(ctx, sqlc.CreateRequestWatcherParams{
		RequestID: requestID,
		UserID:    userID,
	})
	if err != nil {
		s.logger.Error().Err(err).
			Int64("requestID", requestID).
			Int64("userID", userID).
			Msg("failed to create watcher")
		return nil, err
	}

	return toWatcher(watcher), nil
}

func (s *WatchersService) Unwatch(ctx context.Context, requestID, userID int64) error {
	isWatching, err := s.queries.IsWatchingRequest(ctx, sqlc.IsWatchingRequestParams{
		RequestID: requestID,
		UserID:    userID,
	})
	if err != nil {
		return err
	}
	if isWatching == 0 {
		return ErrNotWatching
	}

	return s.queries.DeleteRequestWatcher(ctx, sqlc.DeleteRequestWatcherParams{
		RequestID: requestID,
		UserID:    userID,
	})
}

func (s *WatchersService) GetWatchers(ctx context.Context, requestID int64) ([]*Watcher, error) {
	watchers, err := s.queries.ListRequestWatchers(ctx, requestID)
	if err != nil {
		return nil, err
	}

	result := make([]*Watcher, len(watchers))
	for i, w := range watchers {
		result[i] = toWatcher(w)
	}
	return result, nil
}

func (s *WatchersService) GetWatcherUserIDs(ctx context.Context, requestID int64) ([]int64, error) {
	watchers, err := s.queries.ListRequestWatchers(ctx, requestID)
	if err != nil {
		return nil, err
	}

	userIDs := make([]int64, len(watchers))
	for i, w := range watchers {
		userIDs[i] = w.UserID
	}
	return userIDs, nil
}

func (s *WatchersService) IsWatching(ctx context.Context, requestID, userID int64) (bool, error) {
	isWatching, err := s.queries.IsWatchingRequest(ctx, sqlc.IsWatchingRequestParams{
		RequestID: requestID,
		UserID:    userID,
	})
	if err != nil {
		return false, err
	}
	return isWatching == 1, nil
}

func (s *WatchersService) GetWatchedRequests(ctx context.Context, userID int64) ([]*Request, error) {
	requests, err := s.queries.ListUserWatchedRequests(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*Request, len(requests))
	for i, r := range requests {
		result[i] = toRequest(r)
	}
	return result, nil
}

func (s *WatchersService) CountWatchers(ctx context.Context, requestID int64) (int64, error) {
	return s.queries.CountRequestWatchers(ctx, requestID)
}

func (s *WatchersService) DeleteAllWatchers(ctx context.Context, requestID int64) error {
	return s.queries.DeleteRequestWatchersByRequest(ctx, requestID)
}

func (s *WatchersService) DeleteUserWatchers(ctx context.Context, userID int64) error {
	return s.queries.DeleteRequestWatchersByUser(ctx, userID)
}

func toWatcher(w *sqlc.RequestWatcher) *Watcher {
	return &Watcher{
		RequestID: w.RequestID,
		UserID:    w.UserID,
		CreatedAt: w.CreatedAt,
	}
}

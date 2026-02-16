package indexer

import (
	"context"
	"errors"
	"time"

	"github.com/slipstream/slipstream/internal/indexer/cardigann"
	"github.com/slipstream/slipstream/internal/indexer/status"
)

// statusCookieStore wraps the status service to implement cardigann.CookieStore.
type statusCookieStore struct {
	statusService *status.Service
}

// NewCookieStore creates a CookieStore that uses the status service for persistence.
func NewCookieStore(statusService *status.Service) cardigann.CookieStore {
	return &statusCookieStore{statusService: statusService}
}

func (s *statusCookieStore) GetCookies(ctx context.Context, indexerID int64) (string, error) {
	data, err := s.statusService.GetCookies(ctx, indexerID)
	if err != nil {
		if errors.Is(err, status.ErrNoCookies) {
			return "", nil
		}
		return "", err
	}
	return data.Cookies, nil
}

func (s *statusCookieStore) SaveCookies(ctx context.Context, indexerID int64, cookies string, expiresAt time.Time) error {
	return s.statusService.SaveCookies(ctx, indexerID, cookies, &expiresAt)
}

func (s *statusCookieStore) ClearCookies(ctx context.Context, indexerID int64) error {
	return s.statusService.ClearCookies(ctx, indexerID)
}

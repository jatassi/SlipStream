package identity

import (
	"context"
	"log/slog"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
)

// APIKey is the API key in the clear. It exists only in the one answer
// that shows it; the Library keeps its digest.
type APIKey struct{ token string }

func newAPIKey() APIKey { return APIKey{token: apiKeyPrefix + newToken()} }

func (k APIKey) Reveal() string { return k.token }

func (APIKey) LogValue() slog.Value { return slog.StringValue("[api key]") }

func (s *Service) APIKeyCreatedAt(ctx context.Context, _ Admin) (time.Time, error) {
	at, err := db.New(s.lib).GetAPIKeyCreatedAt(ctx)
	return at.Time(), err
}

// RegenerateAPIKey replaces the API key. The old one stops at this Write.
func (s *Service) RegenerateAPIKey(ctx context.Context, _ Admin) (APIKey, time.Time, error) {
	key := newAPIKey()
	var at time.Time
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		at = fx.now.Time()
		return q.ReplaceAPIKey(ctx, db.ReplaceAPIKeyParams{TokenHash: digest(key.token), CreatedAt: fx.now})
	})
	if err != nil {
		return APIKey{}, time.Time{}, err
	}
	return key, at, nil
}

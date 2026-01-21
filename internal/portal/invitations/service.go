package invitations

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

const (
	DefaultExpiryDuration = 7 * 24 * time.Hour // 7 days
	TokenLength           = 32
)

var (
	ErrInvitationNotFound = errors.New("invitation not found")
	ErrInvitationExpired  = errors.New("invitation has expired")
	ErrInvitationUsed     = errors.New("invitation has already been used")
	ErrInvalidUsername    = errors.New("invalid username")
)

type Invitation struct {
	ID        int64      `json:"id"`
	Username  string     `json:"username"`
	Token     string     `json:"token"`
	ExpiresAt time.Time  `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt"`
	CreatedAt time.Time  `json:"createdAt"`
	Status    string     `json:"status"`
}

type Service struct {
	queries *sqlc.Queries
	logger  zerolog.Logger
}

func NewService(queries *sqlc.Queries, logger zerolog.Logger) *Service {
	return &Service{
		queries: queries,
		logger:  logger.With().Str("component", "portal-invitations").Logger(),
	}
}

func (s *Service) SetDB(queries *sqlc.Queries) {
	s.queries = queries
}

func (s *Service) Create(ctx context.Context, username string) (*Invitation, error) {
	if username == "" {
		return nil, ErrInvalidUsername
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(DefaultExpiryDuration)

	inv, err := s.queries.CreatePortalInvitation(ctx, sqlc.CreatePortalInvitationParams{
		Username:  username,
		Token:     token,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		s.logger.Error().Err(err).Str("username", username).Msg("failed to create invitation")
		return nil, err
	}

	return toInvitation(inv), nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Invitation, error) {
	inv, err := s.queries.GetPortalInvitation(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}
	return toInvitation(inv), nil
}

func (s *Service) GetByToken(ctx context.Context, token string) (*Invitation, error) {
	inv, err := s.queries.GetPortalInvitationByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}
	return toInvitation(inv), nil
}

func (s *Service) GetByUsername(ctx context.Context, username string) (*Invitation, error) {
	inv, err := s.queries.GetPortalInvitationByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}
	return toInvitation(inv), nil
}

func (s *Service) Validate(ctx context.Context, token string) (*Invitation, error) {
	inv, err := s.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if inv.UsedAt != nil {
		return nil, ErrInvitationUsed
	}

	if time.Now().After(inv.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	return inv, nil
}

func (s *Service) MarkUsed(ctx context.Context, id int64) error {
	return s.queries.MarkPortalInvitationUsed(ctx, id)
}

func (s *Service) ResendLink(ctx context.Context, username string) (*Invitation, error) {
	existing, err := s.queries.GetPortalInvitationByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}

	newToken, err := generateToken()
	if err != nil {
		return nil, err
	}

	newExpiry := time.Now().Add(DefaultExpiryDuration)

	inv, err := s.queries.UpdatePortalInvitationToken(ctx, sqlc.UpdatePortalInvitationTokenParams{
		ID:        existing.ID,
		Token:     newToken,
		ExpiresAt: newExpiry,
	})
	if err != nil {
		return nil, err
	}

	return toInvitation(inv), nil
}

func (s *Service) ListPending(ctx context.Context) ([]*Invitation, error) {
	invs, err := s.queries.ListPendingPortalInvitations(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Invitation, len(invs))
	for i, inv := range invs {
		result[i] = toInvitation(inv)
	}
	return result, nil
}

func (s *Service) List(ctx context.Context) ([]*Invitation, error) {
	invs, err := s.queries.ListPortalInvitations(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*Invitation, len(invs))
	for i, inv := range invs {
		result[i] = toInvitation(inv)
	}
	return result, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.queries.DeletePortalInvitation(ctx, id)
}

func (s *Service) CleanupExpired(ctx context.Context) error {
	return s.queries.DeleteExpiredPortalInvitations(ctx)
}

func generateToken() (string, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func toInvitation(inv *sqlc.PortalInvitation) *Invitation {
	var usedAt *time.Time
	if inv.UsedAt.Valid {
		usedAt = &inv.UsedAt.Time
	}

	status := "pending"
	if usedAt != nil {
		status = "used"
	} else if time.Now().After(inv.ExpiresAt) {
		status = "expired"
	}

	return &Invitation{
		ID:        inv.ID,
		Username:  inv.Username,
		Token:     inv.Token,
		ExpiresAt: inv.ExpiresAt,
		UsedAt:    usedAt,
		CreatedAt: inv.CreatedAt,
		Status:    status,
	}
}

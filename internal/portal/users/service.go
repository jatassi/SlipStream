package users

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/portal"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrUsernameExists     = errors.New("username already exists")
	ErrInvalidUsername    = errors.New("invalid username")
	ErrPasswordRequired   = errors.New("password is required")
	ErrInvalidCredentials = portal.ErrInvalidCredentials
	ErrUserDisabled       = portal.ErrUserDisabled
)

type User struct {
	ID               int64     `json:"id"`
	Username         string    `json:"username"`
	DisplayName      string    `json:"displayName"`
	QualityProfileID *int64    `json:"qualityProfileId"`
	AutoApprove      bool      `json:"autoApprove"`
	IsAdmin          bool      `json:"isAdmin"`
	Enabled          bool      `json:"enabled"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Username    string
	Password    string
	DisplayName string
}

type CreateUserInput = CreateInput

type UpdateInput struct {
	Username    *string
	Password    *string
	DisplayName *string
}

type Service struct {
	queries *sqlc.Queries
	logger  zerolog.Logger
}

func NewService(queries *sqlc.Queries, logger zerolog.Logger) *Service {
	return &Service{
		queries: queries,
		logger:  logger.With().Str("component", "portal-users").Logger(),
	}
}

func (s *Service) SetDB(queries *sqlc.Queries) {
	s.queries = queries
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*User, error) {
	if input.Username == "" {
		return nil, ErrInvalidUsername
	}
	if input.Password == "" {
		return nil, ErrPasswordRequired
	}

	existing, err := s.queries.GetPortalUserByUsername(ctx, input.Username)
	if err == nil && existing != nil {
		return nil, ErrUsernameExists
	}

	hash, err := portal.HashPassword(input.Password)
	if err != nil {
		return nil, err
	}

	displayName := input.DisplayName
	if displayName == "" {
		displayName = input.Username
	}

	user, err := s.queries.CreatePortalUser(ctx, sqlc.CreatePortalUserParams{
		Username:     input.Username,
		PasswordHash: hash,
		DisplayName:  sql.NullString{String: displayName, Valid: true},
		AutoApprove:  0,
		Enabled:      1,
	})
	if err != nil {
		s.logger.Error().Err(err).Str("username", input.Username).Msg("failed to create user")
		return nil, err
	}

	return toUser(user), nil
}

func (s *Service) Get(ctx context.Context, id int64) (*User, error) {
	user, err := s.queries.GetPortalUser(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUser(user), nil
}

func (s *Service) GetByUsername(ctx context.Context, username string) (*User, error) {
	user, err := s.queries.GetPortalUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUser(user), nil
}

func (s *Service) GetDBUser(ctx context.Context, id int64) (*sqlc.PortalUser, error) {
	user, err := s.queries.GetPortalUser(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) GetDBUserByUsername(ctx context.Context, username string) (*sqlc.PortalUser, error) {
	user, err := s.queries.GetPortalUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) List(ctx context.Context) ([]*User, error) {
	users, err := s.queries.ListPortalUsers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*User, len(users))
	for i, u := range users {
		result[i] = toUser(u)
	}
	return result, nil
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*User, error) {
	existing, err := s.queries.GetPortalUser(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	username := existing.Username
	if input.Username != nil {
		if *input.Username == "" {
			return nil, ErrInvalidUsername
		}
		other, err := s.queries.GetPortalUserByUsername(ctx, *input.Username)
		if err == nil && other != nil && other.ID != id {
			return nil, ErrUsernameExists
		}
		username = *input.Username
	}

	displayName := existing.DisplayName
	if input.DisplayName != nil {
		displayName = sql.NullString{String: *input.DisplayName, Valid: *input.DisplayName != ""}
	}

	user, err := s.queries.UpdatePortalUser(ctx, sqlc.UpdatePortalUserParams{
		ID:          id,
		Username:    username,
		DisplayName: displayName,
	})
	if err != nil {
		return nil, err
	}

	if input.Password != nil && *input.Password != "" {
		hash, err := portal.HashPassword(*input.Password)
		if err != nil {
			return nil, err
		}
		if err := s.queries.UpdatePortalUserPassword(ctx, sqlc.UpdatePortalUserPasswordParams{
			ID:           id,
			PasswordHash: hash,
		}); err != nil {
			return nil, err
		}
	}

	return toUser(user), nil
}

func (s *Service) SetEnabled(ctx context.Context, id int64, enabled bool) (*User, error) {
	enabledInt := int64(0)
	if enabled {
		enabledInt = 1
	}

	user, err := s.queries.UpdatePortalUserEnabled(ctx, sqlc.UpdatePortalUserEnabledParams{
		ID:      id,
		Enabled: enabledInt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUser(user), nil
}

func (s *Service) SetQualityProfile(ctx context.Context, id int64, profileID *int64) (*User, error) {
	var qpID sql.NullInt64
	if profileID != nil {
		qpID = sql.NullInt64{Int64: *profileID, Valid: true}
	}

	user, err := s.queries.UpdatePortalUserQualityProfile(ctx, sqlc.UpdatePortalUserQualityProfileParams{
		ID:               id,
		QualityProfileID: qpID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUser(user), nil
}

func (s *Service) SetAutoApprove(ctx context.Context, id int64, enabled bool) (*User, error) {
	autoApprove := int64(0)
	if enabled {
		autoApprove = 1
	}

	user, err := s.queries.UpdatePortalUserAutoApprove(ctx, sqlc.UpdatePortalUserAutoApproveParams{
		ID:          id,
		AutoApprove: autoApprove,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUser(user), nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.queries.DeletePortalUser(ctx, id)
}

func (s *Service) GetAdmin(ctx context.Context) (*User, error) {
	user, err := s.queries.GetAdminUser(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUser(user), nil
}

func (s *Service) GetDBAdmin(ctx context.Context) (*sqlc.PortalUser, error) {
	user, err := s.queries.GetAdminUser(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) AdminExists(ctx context.Context) (bool, error) {
	count, err := s.queries.CountAdminUsers(ctx)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Service) CreateAdmin(ctx context.Context, password string) (*User, error) {
	exists, err := s.AdminExists(ctx)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("admin user already exists")
	}

	if password == "" {
		return nil, ErrPasswordRequired
	}

	hash, err := portal.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user, err := s.queries.CreateAdminUser(ctx, sqlc.CreateAdminUserParams{
		Username:     "Administrator",
		PasswordHash: hash,
		DisplayName:  sql.NullString{String: "Administrator", Valid: true},
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to create admin user")
		return nil, err
	}

	return toUser(user), nil
}

func (s *Service) ValidateCredentials(ctx context.Context, username, password string) (*sqlc.PortalUser, error) {
	user, err := s.queries.GetPortalUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, portal.ErrInvalidCredentials
		}
		return nil, err
	}

	if err := portal.ValidatePassword(user.PasswordHash, password); err != nil {
		return nil, err
	}

	if user.Enabled == 0 {
		return nil, portal.ErrUserDisabled
	}

	return user, nil
}

func (s *Service) Authenticate(ctx context.Context, username, password string) (*User, error) {
	user, err := s.queries.GetPortalUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	if err := portal.ValidatePassword(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.Enabled == 0 {
		return nil, ErrUserDisabled
	}

	return toUser(user), nil
}

func toUser(u *sqlc.PortalUser) *User {
	var qpID *int64
	if u.QualityProfileID.Valid {
		qpID = &u.QualityProfileID.Int64
	}

	displayName := ""
	if u.DisplayName.Valid {
		displayName = u.DisplayName.String
	}

	return &User{
		ID:               u.ID,
		Username:         u.Username,
		DisplayName:      displayName,
		QualityProfileID: qpID,
		AutoApprove:      u.AutoApprove == 1,
		IsAdmin:          u.IsAdmin == 1,
		Enabled:          u.Enabled == 1,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}

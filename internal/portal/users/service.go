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
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	AutoApprove bool      `json:"autoApprove"`
	IsAdmin     bool      `json:"isAdmin"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ModuleSettingInput struct {
	QualityProfileID *int64
}

type CreateInput struct {
	Username       string
	Password       string
	ModuleSettings map[string]*ModuleSettingInput
	AutoApprove    bool
}

type CreateUserInput = CreateInput

type UpdateInput struct {
	Username *string
	Password *string
}

type Service struct {
	queries *sqlc.Queries
	logger  *zerolog.Logger
}

func NewService(queries *sqlc.Queries, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "portal-users").Logger()
	return &Service{
		queries: queries,
		logger:  &subLogger,
	}
}

func (s *Service) SetDB(queries *sqlc.Queries) {
	s.queries = queries
}

func (s *Service) UserExists(ctx context.Context, userID int64) (bool, error) {
	user, err := s.queries.GetPortalUser(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return user.Enabled, nil
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

	user, err := s.queries.CreatePortalUser(ctx, sqlc.CreatePortalUserParams{
		Username:     input.Username,
		PasswordHash: hash,
		AutoApprove:  input.AutoApprove,
		Enabled:      true,
	})
	if err != nil {
		s.logger.Error().Err(err).Str("username", input.Username).Msg("failed to create user")
		return nil, err
	}

	if err := s.initModuleSettings(ctx, user.ID, input.ModuleSettings); err != nil {
		return nil, err
	}

	return toUser(user), nil
}

func (s *Service) initModuleSettings(ctx context.Context, userID int64, settings map[string]*ModuleSettingInput) error {
	for moduleType, ms := range settings {
		if ms == nil {
			continue
		}
		var qpID sql.NullInt64
		if ms.QualityProfileID != nil {
			qpID = sql.NullInt64{Int64: *ms.QualityProfileID, Valid: true}
		}
		_, err := s.queries.UpsertUserModuleSettings(ctx, sqlc.UpsertUserModuleSettingsParams{
			UserID:           userID,
			ModuleType:       moduleType,
			QualityProfileID: qpID,
			PeriodStart:      time.Now(),
		})
		if err != nil {
			s.logger.Error().Err(err).Str("moduleType", moduleType).Int64("userID", userID).Msg("failed to create module settings")
			return err
		}
	}
	return nil
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

	username, err := s.resolveUsername(ctx, id, existing.Username, input.Username)
	if err != nil {
		return nil, err
	}

	user, err := s.queries.UpdatePortalUser(ctx, sqlc.UpdatePortalUserParams{
		ID:       id,
		Username: username,
	})
	if err != nil {
		return nil, err
	}

	if err := s.updatePasswordIfProvided(ctx, id, input.Password); err != nil {
		return nil, err
	}

	return toUser(user), nil
}

func (s *Service) resolveUsername(ctx context.Context, userID int64, current string, newUsername *string) (string, error) {
	if newUsername == nil {
		return current, nil
	}
	if *newUsername == "" {
		return "", ErrInvalidUsername
	}
	other, err := s.queries.GetPortalUserByUsername(ctx, *newUsername)
	if err == nil && other != nil && other.ID != userID {
		return "", ErrUsernameExists
	}
	return *newUsername, nil
}

func (s *Service) updatePasswordIfProvided(ctx context.Context, userID int64, password *string) error {
	if password == nil || *password == "" {
		return nil
	}
	hash, err := portal.HashPassword(*password)
	if err != nil {
		return err
	}
	return s.queries.UpdatePortalUserPassword(ctx, sqlc.UpdatePortalUserPasswordParams{
		ID:           userID,
		PasswordHash: hash,
	})
}

func (s *Service) SetEnabled(ctx context.Context, id int64, enabled bool) (*User, error) {
	user, err := s.queries.UpdatePortalUserEnabled(ctx, sqlc.UpdatePortalUserEnabledParams{
		ID:      id,
		Enabled: enabled,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return toUser(user), nil
}

func (s *Service) GetModuleSettings(ctx context.Context, userID int64) ([]*sqlc.PortalUserModuleSetting, error) {
	return s.queries.ListUserModuleSettings(ctx, userID)
}

func (s *Service) SetModuleQualityProfile(ctx context.Context, userID int64, moduleType string, profileID *int64) error {
	_, err := s.queries.GetUserModuleSettings(ctx, sqlc.GetUserModuleSettingsParams{
		UserID:     userID,
		ModuleType: moduleType,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			var qpID sql.NullInt64
			if profileID != nil {
				qpID = sql.NullInt64{Int64: *profileID, Valid: true}
			}
			_, err = s.queries.UpsertUserModuleSettings(ctx, sqlc.UpsertUserModuleSettingsParams{
				UserID:           userID,
				ModuleType:       moduleType,
				QualityProfileID: qpID,
				PeriodStart:      time.Now(),
			})
			return err
		}
		return err
	}

	var qpID sql.NullInt64
	if profileID != nil {
		qpID = sql.NullInt64{Int64: *profileID, Valid: true}
	}
	_, err = s.queries.UpdateUserModuleQualityProfile(ctx, sqlc.UpdateUserModuleQualityProfileParams{
		QualityProfileID: qpID,
		UserID:           userID,
		ModuleType:       moduleType,
	})
	return err
}

func (s *Service) SetAutoApprove(ctx context.Context, id int64, enabled bool) (*User, error) {
	user, err := s.queries.UpdatePortalUserAutoApprove(ctx, sqlc.UpdatePortalUserAutoApproveParams{
		ID:          id,
		AutoApprove: enabled,
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

	if !user.Enabled {
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

	if !user.Enabled {
		return nil, ErrUserDisabled
	}

	return toUser(user), nil
}

func toUser(u *sqlc.PortalUser) *User {
	return &User{
		ID:          u.ID,
		Username:    u.Username,
		AutoApprove: u.AutoApprove,
		IsAdmin:     u.IsAdmin,
		Enabled:     u.Enabled,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

package quota

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

const (
	SettingDefaultQuotaPrefix = "requests_default_%s_quota"
	DefaultQuota              = 5
)

var (
	ErrQuotaExceeded = errors.New("quota exceeded")
	ErrQuotaNotFound = errors.New("quota not found")
)

type ModuleQuotaStatus struct {
	ModuleType  string    `json:"moduleType"`
	QuotaLimit  int64     `json:"quotaLimit"`
	QuotaUsed   int64     `json:"quotaUsed"`
	PeriodStart time.Time `json:"periodStart"`
}

type QuotaStatus struct {
	UserID  int64               `json:"userId"`
	Modules []ModuleQuotaStatus `json:"modules"`
}

type Service struct {
	queries *sqlc.Queries
	logger  *zerolog.Logger
}

func NewService(queries *sqlc.Queries, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "portal-quota").Logger()
	return &Service{
		queries: queries,
		logger:  &subLogger,
	}
}

func (s *Service) SetDB(queries *sqlc.Queries) {
	s.queries = queries
}

func (s *Service) CheckQuota(ctx context.Context, userID int64, moduleType string) (bool, error) {
	setting, err := s.queries.GetUserModuleSettings(ctx, sqlc.GetUserModuleSettingsParams{
		UserID:     userID,
		ModuleType: moduleType,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return true, nil
		}
		return false, err
	}

	if s.shouldResetQuota(setting.PeriodStart) {
		setting, err = s.resetModuleQuota(ctx, userID, moduleType)
		if err != nil {
			return false, err
		}
	}

	limit := s.effectiveLimit(ctx, setting, moduleType)
	if limit == 0 {
		return true, nil
	}
	return setting.QuotaUsed < limit, nil
}

func (s *Service) ConsumeQuota(ctx context.Context, userID int64, moduleType string) error {
	canConsume, err := s.CheckQuota(ctx, userID, moduleType)
	if err != nil {
		return err
	}
	if !canConsume {
		return ErrQuotaExceeded
	}

	if ensureErr := s.ensureModuleSettings(ctx, userID, moduleType); ensureErr != nil {
		return ensureErr
	}

	_, err = s.queries.IncrementUserModuleQuota(ctx, sqlc.IncrementUserModuleQuotaParams{
		UserID:     userID,
		ModuleType: moduleType,
	})
	return err
}

func (s *Service) GetQuotaStatus(ctx context.Context, userID int64) (*QuotaStatus, error) {
	settings, err := s.queries.ListUserModuleSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	modules := make([]ModuleQuotaStatus, 0, len(settings))
	for _, ms := range settings {
		if s.shouldResetQuota(ms.PeriodStart) {
			ms, err = s.resetModuleQuota(ctx, userID, ms.ModuleType)
			if err != nil {
				return nil, err
			}
		}

		limit := s.effectiveLimit(ctx, ms, ms.ModuleType)
		modules = append(modules, ModuleQuotaStatus{
			ModuleType:  ms.ModuleType,
			QuotaLimit:  limit,
			QuotaUsed:   ms.QuotaUsed,
			PeriodStart: ms.PeriodStart,
		})
	}

	return &QuotaStatus{
		UserID:  userID,
		Modules: modules,
	}, nil
}

func (s *Service) GetGlobalDefaults(ctx context.Context) map[string]int64 {
	defaults := map[string]int64{}

	for _, moduleType := range []string{"movie", "tv"} {
		key := fmt.Sprintf(SettingDefaultQuotaPrefix, moduleType)
		if setting, err := s.queries.GetSetting(ctx, key); err == nil {
			if v, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
				defaults[moduleType] = v
				continue
			}
		}
		defaults[moduleType] = DefaultQuota
	}

	return defaults
}

func (s *Service) SetGlobalDefault(ctx context.Context, moduleType string, limit int64) error {
	key := fmt.Sprintf(SettingDefaultQuotaPrefix, moduleType)
	_, err := s.queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   key,
		Value: strconv.FormatInt(limit, 10),
	})
	return err
}

func (s *Service) SetUserOverride(ctx context.Context, userID int64, moduleType string, limit int64) error {
	if err := s.ensureModuleSettings(ctx, userID, moduleType); err != nil {
		return err
	}

	_, err := s.queries.UpdateUserModuleQuotaLimit(ctx, sqlc.UpdateUserModuleQuotaLimitParams{
		QuotaLimit: sql.NullInt64{Int64: limit, Valid: true},
		UserID:     userID,
		ModuleType: moduleType,
	})
	return err
}

func (s *Service) ClearUserOverride(ctx context.Context, userID int64, moduleType string) error {
	_, err := s.queries.UpdateUserModuleQuotaLimit(ctx, sqlc.UpdateUserModuleQuotaLimitParams{
		QuotaLimit: sql.NullInt64{Valid: false},
		UserID:     userID,
		ModuleType: moduleType,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrQuotaNotFound
		}
		return err
	}
	return nil
}

func (s *Service) ResetAllQuotas(ctx context.Context) error {
	periodStart := s.getNextMondayMidnight()
	err := s.queries.ResetAllModuleQuotas(ctx, periodStart)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to reset all quotas")
		return err
	}
	s.logger.Info().Time("periodStart", periodStart).Msg("all quotas reset")
	return nil
}

func (s *Service) ensureModuleSettings(ctx context.Context, userID int64, moduleType string) error {
	_, err := s.queries.GetUserModuleSettings(ctx, sqlc.GetUserModuleSettingsParams{
		UserID:     userID,
		ModuleType: moduleType,
	})
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	periodStart := s.getNextMondayMidnight()
	_, err = s.queries.UpsertUserModuleSettings(ctx, sqlc.UpsertUserModuleSettingsParams{
		UserID:           userID,
		ModuleType:       moduleType,
		QuotaLimit:       sql.NullInt64{Valid: false},
		QuotaUsed:        0,
		QualityProfileID: sql.NullInt64{Valid: false},
		PeriodStart:      periodStart,
	})
	return err
}

func (s *Service) resetModuleQuota(ctx context.Context, userID int64, moduleType string) (*sqlc.PortalUserModuleSetting, error) {
	periodStart := s.getNextMondayMidnight()
	return s.queries.ResetUserModuleQuota(ctx, sqlc.ResetUserModuleQuotaParams{
		PeriodStart: periodStart,
		UserID:      userID,
		ModuleType:  moduleType,
	})
}

func (s *Service) effectiveLimit(ctx context.Context, setting *sqlc.PortalUserModuleSetting, moduleType string) int64 {
	if setting.QuotaLimit.Valid {
		return setting.QuotaLimit.Int64
	}
	return s.getDefaultQuota(ctx, moduleType)
}

func (s *Service) getDefaultQuota(ctx context.Context, moduleType string) int64 {
	key := fmt.Sprintf(SettingDefaultQuotaPrefix, moduleType)
	if setting, err := s.queries.GetSetting(ctx, key); err == nil {
		if v, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
			return v
		}
	}
	return DefaultQuota
}

func (s *Service) shouldResetQuota(periodStart time.Time) bool {
	return time.Now().After(periodStart)
}

func (s *Service) getNextMondayMidnight() time.Time {
	now := time.Now().Local()
	daysUntilMonday := (8 - int(now.Weekday())) % 7
	if daysUntilMonday == 0 && now.Hour() >= 0 {
		daysUntilMonday = 7
	}

	nextMonday := now.AddDate(0, 0, daysUntilMonday)
	return time.Date(nextMonday.Year(), nextMonday.Month(), nextMonday.Day(), 0, 0, 0, 0, time.Local)
}

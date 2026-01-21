package quota

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

const (
	SettingDefaultMovieQuota   = "requests_default_movie_quota"
	SettingDefaultSeasonQuota  = "requests_default_season_quota"
	SettingDefaultEpisodeQuota = "requests_default_episode_quota"

	DefaultMovieQuota   = 5
	DefaultSeasonQuota  = 3
	DefaultEpisodeQuota = 10
)

var (
	ErrQuotaExceeded = errors.New("quota exceeded")
	ErrQuotaNotFound = errors.New("quota not found")
)

type QuotaStatus struct {
	UserID        int64     `json:"userId"`
	MoviesLimit   int64     `json:"moviesLimit"`
	SeasonsLimit  int64     `json:"seasonsLimit"`
	EpisodesLimit int64     `json:"episodesLimit"`
	MoviesUsed    int64     `json:"moviesUsed"`
	SeasonsUsed   int64     `json:"seasonsUsed"`
	EpisodesUsed  int64     `json:"episodesUsed"`
	PeriodStart   time.Time `json:"periodStart"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type QuotaLimits struct {
	MoviesLimit   *int64
	SeasonsLimit  *int64
	EpisodesLimit *int64
}

type Service struct {
	queries *sqlc.Queries
	logger  zerolog.Logger
}

func NewService(queries *sqlc.Queries, logger zerolog.Logger) *Service {
	return &Service{
		queries: queries,
		logger:  logger.With().Str("component", "portal-quota").Logger(),
	}
}

func (s *Service) SetDB(queries *sqlc.Queries) {
	s.queries = queries
}

func (s *Service) GetUserQuota(ctx context.Context, userID int64) (*QuotaStatus, error) {
	quota, err := s.queries.GetUserQuota(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return s.initializeUserQuota(ctx, userID)
		}
		return nil, err
	}

	if s.shouldResetQuota(quota.PeriodStart) {
		return s.resetUserQuota(ctx, userID)
	}

	return s.toQuotaStatus(ctx, quota), nil
}

func (s *Service) CheckQuota(ctx context.Context, userID int64, mediaType string) (bool, error) {
	status, err := s.GetUserQuota(ctx, userID)
	if err != nil {
		return false, err
	}

	switch mediaType {
	case "movie":
		return status.MoviesUsed < status.MoviesLimit, nil
	case "season":
		return status.SeasonsUsed < status.SeasonsLimit, nil
	case "episode":
		return status.EpisodesUsed < status.EpisodesLimit, nil
	}

	return true, nil
}

func (s *Service) ConsumeQuota(ctx context.Context, userID int64, mediaType string) error {
	canConsume, err := s.CheckQuota(ctx, userID, mediaType)
	if err != nil {
		return err
	}
	if !canConsume {
		return ErrQuotaExceeded
	}

	switch mediaType {
	case "movie":
		_, err = s.queries.IncrementMovieQuota(ctx, userID)
	case "season":
		_, err = s.queries.IncrementSeasonQuota(ctx, userID)
	case "episode":
		_, err = s.queries.IncrementEpisodeQuota(ctx, userID)
	}

	return err
}

func (s *Service) ResetAllQuotas(ctx context.Context) error {
	periodStart := s.getNextMondayMidnight()
	err := s.queries.ResetAllQuotas(ctx, periodStart)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to reset all quotas")
		return err
	}
	s.logger.Info().Time("periodStart", periodStart).Msg("all quotas reset")
	return nil
}

func (s *Service) GetGlobalDefaults(ctx context.Context) (*QuotaLimits, error) {
	movieLimit := int64(DefaultMovieQuota)
	seasonLimit := int64(DefaultSeasonQuota)
	episodeLimit := int64(DefaultEpisodeQuota)

	if setting, err := s.queries.GetSetting(ctx, SettingDefaultMovieQuota); err == nil {
		if v, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
			movieLimit = v
		}
	}

	if setting, err := s.queries.GetSetting(ctx, SettingDefaultSeasonQuota); err == nil {
		if v, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
			seasonLimit = v
		}
	}

	if setting, err := s.queries.GetSetting(ctx, SettingDefaultEpisodeQuota); err == nil {
		if v, err := strconv.ParseInt(setting.Value, 10, 64); err == nil {
			episodeLimit = v
		}
	}

	return &QuotaLimits{
		MoviesLimit:   &movieLimit,
		SeasonsLimit:  &seasonLimit,
		EpisodesLimit: &episodeLimit,
	}, nil
}

func (s *Service) SetGlobalDefaults(ctx context.Context, limits QuotaLimits) error {
	if limits.MoviesLimit != nil {
		if _, err := s.queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   SettingDefaultMovieQuota,
			Value: strconv.FormatInt(*limits.MoviesLimit, 10),
		}); err != nil {
			return err
		}
	}

	if limits.SeasonsLimit != nil {
		if _, err := s.queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   SettingDefaultSeasonQuota,
			Value: strconv.FormatInt(*limits.SeasonsLimit, 10),
		}); err != nil {
			return err
		}
	}

	if limits.EpisodesLimit != nil {
		if _, err := s.queries.SetSetting(ctx, sqlc.SetSettingParams{
			Key:   SettingDefaultEpisodeQuota,
			Value: strconv.FormatInt(*limits.EpisodesLimit, 10),
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) SetUserOverride(ctx context.Context, userID int64, limits QuotaLimits) (*QuotaStatus, error) {
	existing, err := s.queries.GetUserQuota(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_, err = s.initializeUserQuota(ctx, userID)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	var moviesLimit, seasonsLimit, episodesLimit sql.NullInt64

	if existing != nil {
		moviesLimit = existing.MoviesLimit
		seasonsLimit = existing.SeasonsLimit
		episodesLimit = existing.EpisodesLimit
	}

	if limits.MoviesLimit != nil {
		moviesLimit = sql.NullInt64{Int64: *limits.MoviesLimit, Valid: true}
	}
	if limits.SeasonsLimit != nil {
		seasonsLimit = sql.NullInt64{Int64: *limits.SeasonsLimit, Valid: true}
	}
	if limits.EpisodesLimit != nil {
		episodesLimit = sql.NullInt64{Int64: *limits.EpisodesLimit, Valid: true}
	}

	quota, err := s.queries.UpdateUserQuotaLimits(ctx, sqlc.UpdateUserQuotaLimitsParams{
		UserID:        userID,
		MoviesLimit:   moviesLimit,
		SeasonsLimit:  seasonsLimit,
		EpisodesLimit: episodesLimit,
	})
	if err != nil {
		return nil, err
	}

	return s.toQuotaStatus(ctx, quota), nil
}

func (s *Service) ClearUserOverride(ctx context.Context, userID int64) (*QuotaStatus, error) {
	quota, err := s.queries.UpdateUserQuotaLimits(ctx, sqlc.UpdateUserQuotaLimitsParams{
		UserID:        userID,
		MoviesLimit:   sql.NullInt64{Valid: false},
		SeasonsLimit:  sql.NullInt64{Valid: false},
		EpisodesLimit: sql.NullInt64{Valid: false},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrQuotaNotFound
		}
		return nil, err
	}

	return s.toQuotaStatus(ctx, quota), nil
}

func (s *Service) initializeUserQuota(ctx context.Context, userID int64) (*QuotaStatus, error) {
	periodStart := s.getNextMondayMidnight()

	quota, err := s.queries.UpsertUserQuota(ctx, sqlc.UpsertUserQuotaParams{
		UserID:        userID,
		MoviesLimit:   sql.NullInt64{Valid: false},
		SeasonsLimit:  sql.NullInt64{Valid: false},
		EpisodesLimit: sql.NullInt64{Valid: false},
		MoviesUsed:    0,
		SeasonsUsed:   0,
		EpisodesUsed:  0,
		PeriodStart:   periodStart,
	})
	if err != nil {
		return nil, err
	}

	return s.toQuotaStatus(ctx, quota), nil
}

func (s *Service) resetUserQuota(ctx context.Context, userID int64) (*QuotaStatus, error) {
	periodStart := s.getNextMondayMidnight()

	quota, err := s.queries.ResetUserQuota(ctx, sqlc.ResetUserQuotaParams{
		UserID:      userID,
		PeriodStart: periodStart,
	})
	if err != nil {
		return nil, err
	}

	return s.toQuotaStatus(ctx, quota), nil
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

func (s *Service) toQuotaStatus(ctx context.Context, q *sqlc.UserQuota) *QuotaStatus {
	defaults, _ := s.GetGlobalDefaults(ctx)

	moviesLimit := *defaults.MoviesLimit
	seasonsLimit := *defaults.SeasonsLimit
	episodesLimit := *defaults.EpisodesLimit

	if q.MoviesLimit.Valid {
		moviesLimit = q.MoviesLimit.Int64
	}
	if q.SeasonsLimit.Valid {
		seasonsLimit = q.SeasonsLimit.Int64
	}
	if q.EpisodesLimit.Valid {
		episodesLimit = q.EpisodesLimit.Int64
	}

	return &QuotaStatus{
		UserID:        q.UserID,
		MoviesLimit:   moviesLimit,
		SeasonsLimit:  seasonsLimit,
		EpisodesLimit: episodesLimit,
		MoviesUsed:    q.MoviesUsed,
		SeasonsUsed:   q.SeasonsUsed,
		EpisodesUsed:  q.EpisodesUsed,
		PeriodStart:   q.PeriodStart,
		UpdatedAt:     q.UpdatedAt,
	}
}

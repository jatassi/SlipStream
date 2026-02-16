package preferences

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

const boolTrue = "true"

type Service struct {
	queries *sqlc.Queries
}

func NewService(queries *sqlc.Queries) *Service {
	return &Service{queries: queries}
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)
}

// GetAddFlowPreferences returns all add-flow related preferences
func (s *Service) GetAddFlowPreferences(ctx context.Context) (*AddFlowPreferences, error) {
	prefs := DefaultPreferences()

	if val, err := s.getString(ctx, KeyMovieSearchOnAdd); err == nil {
		prefs.MovieSearchOnAdd = val == boolTrue
	}

	if val, err := s.getString(ctx, KeySeriesSearchOnAdd); err == nil && ValidSeriesSearchOnAdd(val) {
		prefs.SeriesSearchOnAdd = SeriesSearchOnAdd(val)
	}

	if val, err := s.getString(ctx, KeySeriesMonitorOnAdd); err == nil && ValidSeriesMonitorOnAdd(val) {
		prefs.SeriesMonitorOnAdd = SeriesMonitorOnAdd(val)
	}

	if val, err := s.getString(ctx, KeySeriesIncludeSpecials); err == nil {
		prefs.SeriesIncludeSpecials = val == boolTrue
	}

	return &prefs, nil
}

// SetAddFlowPreferences updates all add-flow related preferences
func (s *Service) SetAddFlowPreferences(ctx context.Context, prefs AddFlowPreferences) error {
	if err := s.setString(ctx, KeyMovieSearchOnAdd, strconv.FormatBool(prefs.MovieSearchOnAdd)); err != nil {
		return err
	}

	if ValidSeriesSearchOnAdd(string(prefs.SeriesSearchOnAdd)) {
		if err := s.setString(ctx, KeySeriesSearchOnAdd, string(prefs.SeriesSearchOnAdd)); err != nil {
			return err
		}
	}

	if ValidSeriesMonitorOnAdd(string(prefs.SeriesMonitorOnAdd)) {
		if err := s.setString(ctx, KeySeriesMonitorOnAdd, string(prefs.SeriesMonitorOnAdd)); err != nil {
			return err
		}
	}

	if err := s.setString(ctx, KeySeriesIncludeSpecials, strconv.FormatBool(prefs.SeriesIncludeSpecials)); err != nil {
		return err
	}

	return nil
}

// GetMovieSearchOnAdd returns the movie search-on-add preference
func (s *Service) GetMovieSearchOnAdd(ctx context.Context) bool {
	val, err := s.getString(ctx, KeyMovieSearchOnAdd)
	if err != nil {
		return DefaultPreferences().MovieSearchOnAdd
	}
	return val == boolTrue
}

// SetMovieSearchOnAdd updates the movie search-on-add preference
func (s *Service) SetMovieSearchOnAdd(ctx context.Context, value bool) error {
	return s.setString(ctx, KeyMovieSearchOnAdd, strconv.FormatBool(value))
}

// GetSeriesSearchOnAdd returns the series search-on-add preference
func (s *Service) GetSeriesSearchOnAdd(ctx context.Context) SeriesSearchOnAdd {
	val, err := s.getString(ctx, KeySeriesSearchOnAdd)
	if err != nil || !ValidSeriesSearchOnAdd(val) {
		return DefaultPreferences().SeriesSearchOnAdd
	}
	return SeriesSearchOnAdd(val)
}

// SetSeriesSearchOnAdd updates the series search-on-add preference
func (s *Service) SetSeriesSearchOnAdd(ctx context.Context, value SeriesSearchOnAdd) error {
	if !ValidSeriesSearchOnAdd(string(value)) {
		return errors.New("invalid series search on add value")
	}
	return s.setString(ctx, KeySeriesSearchOnAdd, string(value))
}

// GetSeriesMonitorOnAdd returns the series monitor-on-add preference
func (s *Service) GetSeriesMonitorOnAdd(ctx context.Context) SeriesMonitorOnAdd {
	val, err := s.getString(ctx, KeySeriesMonitorOnAdd)
	if err != nil || !ValidSeriesMonitorOnAdd(val) {
		return DefaultPreferences().SeriesMonitorOnAdd
	}
	return SeriesMonitorOnAdd(val)
}

// SetSeriesMonitorOnAdd updates the series monitor-on-add preference
func (s *Service) SetSeriesMonitorOnAdd(ctx context.Context, value SeriesMonitorOnAdd) error {
	if !ValidSeriesMonitorOnAdd(string(value)) {
		return errors.New("invalid series monitor on add value")
	}
	return s.setString(ctx, KeySeriesMonitorOnAdd, string(value))
}

// GetSeriesIncludeSpecials returns the series include-specials preference
func (s *Service) GetSeriesIncludeSpecials(ctx context.Context) bool {
	val, err := s.getString(ctx, KeySeriesIncludeSpecials)
	if err != nil {
		return DefaultPreferences().SeriesIncludeSpecials
	}
	return val == boolTrue
}

// SetSeriesIncludeSpecials updates the series include-specials preference
func (s *Service) SetSeriesIncludeSpecials(ctx context.Context, value bool) error {
	return s.setString(ctx, KeySeriesIncludeSpecials, strconv.FormatBool(value))
}

func (s *Service) getString(ctx context.Context, key string) (string, error) {
	setting, err := s.queries.GetSetting(ctx, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
		return "", err
	}
	return setting.Value, nil
}

func (s *Service) setString(ctx context.Context, key, value string) error {
	_, err := s.queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   key,
		Value: value,
	})
	return err
}

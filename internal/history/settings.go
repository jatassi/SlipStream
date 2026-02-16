package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

const settingsKey = "history_retention"

// RetentionSettings contains history retention configuration.
type RetentionSettings struct {
	Enabled       bool `json:"enabled"`
	RetentionDays int  `json:"retentionDays"`
}

// DefaultRetentionSettings returns default retention settings.
func DefaultRetentionSettings() RetentionSettings {
	return RetentionSettings{
		Enabled:       true,
		RetentionDays: 365,
	}
}

// GetRetentionSettings loads retention settings from the database.
func (s *Service) GetRetentionSettings(ctx context.Context) (RetentionSettings, error) {
	row, err := s.queries.GetSetting(ctx, settingsKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DefaultRetentionSettings(), nil
		}
		return RetentionSettings{}, err
	}

	var settings RetentionSettings
	if err := json.Unmarshal([]byte(row.Value), &settings); err != nil {
		return DefaultRetentionSettings(), nil //nolint:nilerr // Invalid JSON, use defaults
	}
	return settings, nil
}

// SaveRetentionSettings saves retention settings to the database.
func (s *Service) SaveRetentionSettings(ctx context.Context, settings RetentionSettings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}

	_, err = s.queries.SetSetting(ctx, sqlc.SetSettingParams{
		Key:   settingsKey,
		Value: string(data),
	})
	return err
}

// CleanupOldEntries deletes history entries older than the configured retention period.
func (s *Service) CleanupOldEntries(ctx context.Context) error {
	settings, err := s.GetRetentionSettings(ctx)
	if err != nil {
		return err
	}

	if !settings.Enabled || settings.RetentionDays <= 0 {
		return nil
	}

	cutoff := sql.NullTime{
		Time:  time.Now().AddDate(0, 0, -settings.RetentionDays),
		Valid: true,
	}

	return s.queries.DeleteOldHistory(ctx, cutoff)
}

package module

import (
	"context"
	"database/sql"
	"fmt"
)

// IsModuleEnabled checks whether a module is enabled in settings.
// Returns true by default if no setting exists (all modules enabled out of the box).
func IsModuleEnabled(ctx context.Context, db *sql.DB, moduleType Type) (bool, error) {
	key := fmt.Sprintf("module.%s.enabled", moduleType)
	var value string
	err := db.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return value == "true", nil
}

// SetModuleEnabled sets the enabled state for a module.
func SetModuleEnabled(ctx context.Context, db *sql.DB, moduleType Type, enabled bool) error {
	key := fmt.Sprintf("module.%s.enabled", moduleType)
	value := "false"
	if enabled {
		value = "true"
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		key, value,
	)
	return err
}

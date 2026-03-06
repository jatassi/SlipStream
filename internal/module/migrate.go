package module

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

// MigrateModule runs migrations for a specific module using its own
// embedded migration files and version tracking table.
//
// Each module gets a separate goose version table (e.g., goose_movie_version,
// goose_tv_version) so modules can migrate independently.
//
// Uses goose.NewProvider() (instance-based API) to avoid global state issues.
// This is safe to call for multiple modules without concurrency concerns.
//
// The dir parameter specifies the subdirectory within the embedded FS that
// contains the .sql files (e.g., "migrations").
func MigrateModule(db *sql.DB, moduleType Type, migrations embed.FS, dir string) error {
	tableName := fmt.Sprintf("goose_%s_version", moduleType)

	subFS, err := fs.Sub(migrations, dir)
	if err != nil {
		return fmt.Errorf("sub filesystem for module %s migrations: %w", moduleType, err)
	}

	provider, err := goose.NewProvider(
		goose.DialectSQLite3,
		db,
		subFS,
		goose.WithTableName(tableName),
	)
	if err != nil {
		return fmt.Errorf("create migration provider for module %s: %w", moduleType, err)
	}

	if _, err := provider.Up(context.Background()); err != nil {
		return fmt.Errorf("migrate module %s: %w", moduleType, err)
	}

	return nil
}

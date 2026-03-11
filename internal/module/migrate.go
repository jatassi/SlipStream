package module

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

// Migrator is an optional interface that modules implement to declare their
// own embedded migration files. Each module gets a separate goose version
// table (e.g., goose_db_version_movie, goose_db_version_tv) so modules can
// migrate independently.
//
// Modules that have no module-specific migrations yet need not implement this
// interface — they will simply be skipped during MigrateAll.
type Migrator interface {
	// MigrationFS returns the embedded filesystem containing migration files
	// and the subdirectory within that FS where .sql files reside.
	MigrationFS() (migrations embed.FS, dir string)
}

// MigrateAll runs per-module migrations for every registered module that
// implements the Migrator interface. This should be called after framework
// migrations have completed.
func MigrateAll(db *sql.DB, registry *Registry) error {
	for _, mod := range registry.All() {
		migrator, ok := mod.(Migrator)
		if !ok {
			continue
		}
		migrations, dir := migrator.MigrationFS()
		if err := MigrateModule(db, mod.ID(), migrations, dir); err != nil {
			return err
		}
	}
	return nil
}

// MigrateModule runs migrations for a specific module using its own
// embedded migration files and version tracking table.
//
// Each module gets a separate goose version table (e.g., goose_db_version_movie,
// goose_db_version_tv) so modules can migrate independently.
//
// Uses goose.NewProvider() (instance-based API) to avoid global state issues.
// This is safe to call for multiple modules without concurrency concerns.
//
// The dir parameter specifies the subdirectory within the embedded FS that
// contains the .sql files (e.g., "migrations").
func MigrateModule(db *sql.DB, moduleType Type, migrations embed.FS, dir string) error {
	subFS, err := fs.Sub(migrations, dir)
	if err != nil {
		return fmt.Errorf("sub filesystem for module %s migrations: %w", moduleType, err)
	}

	// Check if there are any .sql migration files; skip if directory is empty.
	hasMigrations := false
	_ = fs.WalkDir(subFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && len(path) > 4 && path[len(path)-4:] == ".sql" {
			hasMigrations = true
			return fs.SkipAll
		}
		return nil
	})
	if !hasMigrations {
		return nil
	}

	tableName := fmt.Sprintf("goose_db_version_%s", moduleType)

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

package movie

import "embed"

//go:embed all:migrations
var moduleMigrations embed.FS

// MigrationFS returns the embedded migration files and the subdirectory
// containing them. Used by the framework to run per-module migrations
// with a module-specific goose version table.
func (m *Module) MigrationFS() (migrations embed.FS, dir string) {
	return moduleMigrations, "migrations"
}

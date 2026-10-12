package storage

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "PRAGMA foreign_keys = OFF")
	_ = goose.Up
	return err
}

package store

import "database/sql"

func Exec(db *sql.DB) { _, _ = db.ExecContext(nil, "") }

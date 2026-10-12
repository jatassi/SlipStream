package media

import "database/sql"

func plant(db *sql.DB) { _, _ = db.ExecContext(nil, "INSERT INTO goose_media VALUES (9)") }

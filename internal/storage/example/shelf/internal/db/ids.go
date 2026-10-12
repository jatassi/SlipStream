package db

// ShelfID is shelf.id. sqlc.yaml maps the column here with a bare go_type;
// the shelf package exports it as shelf.ID.
type ShelfID int64

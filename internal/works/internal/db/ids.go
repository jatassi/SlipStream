package db

import "github.com/jatassi/slipstream/internal/storage"

type (
	WorkID  struct{ storage.UUID }
	LevelID struct{ storage.UUID }
	ItemID  struct{ storage.UUID }
)

type TypeID string

type Key string

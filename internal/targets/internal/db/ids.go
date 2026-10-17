package db

import "github.com/jatassi/slipstream/internal/storage"

type (
	TargetID  struct{ storage.UUID }
	ProfileID struct{ storage.UUID }
)

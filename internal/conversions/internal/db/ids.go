package db

import "github.com/jatassi/slipstream/internal/storage"

type (
	ConversionID struct{ storage.UUID }
	BackfillID   struct{ storage.UUID }
)

// NudgeID is a conversions_nudge row's rowid, oldest first.
type NudgeID int64

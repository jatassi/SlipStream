package db

import "github.com/jatassi/slipstream/internal/storage"

type ClipID int64

type ClipPublicID struct{ storage.UUID }

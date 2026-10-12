package db

import "github.com/jatassi/slipstream/internal/storage"

type AccountID struct{ storage.UUID }

type SessionID struct{ storage.UUID }

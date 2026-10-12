package db

import "time"

type RootID int64

type legacy struct {
	ID      int64
	AddedAt time.Time
}

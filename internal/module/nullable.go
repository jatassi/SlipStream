package module

import (
	"database/sql"
	"time"
)

// NullStr returns the string value if valid, empty string otherwise.
func NullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// NullTime returns the time value if valid, zero time otherwise.
func NullTime(nt sql.NullTime) time.Time {
	if nt.Valid {
		return nt.Time
	}
	return time.Time{}
}

// NullInt64Ptr returns a pointer to the int64 value if valid, nil otherwise.
func NullInt64Ptr(ni sql.NullInt64) *int64 {
	if ni.Valid {
		v := ni.Int64
		return &v
	}
	return nil
}

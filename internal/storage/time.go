package storage

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Time is a UTC instant at microsecond precision, stored as INTEGER unix
// microseconds. Every *_at column maps to it (each sqlc.yaml's first
// override). Its field is unexported, so a local time cannot be built; Now
// and FromTime are the only ways in.
//
// The zero Time is SQL NULL, both ways: sqlc's column override cannot tell
// a nullable column apart, so nullability lives in the value. Two Times are
// equal exactly when == says so, because every Time is normalized.
type Time struct {
	t time.Time // zero, or UTC with no monotonic reading, truncated to the µs
}

// Now reads time.Now, which synctest fakes.
func Now() Time { return FromTime(time.Now()) }

// FromTime converts an instant in any zone to UTC. The zero time.Time stays
// the zero Time.
func FromTime(t time.Time) Time {
	if t.IsZero() {
		return Time{}
	}
	return Time{t: t.UTC().Truncate(time.Microsecond)}
}

func (t Time) Time() time.Time { return t.t }

func (t Time) IsZero() bool { return t.t.IsZero() }

func (t Time) Compare(u Time) int { return t.t.Compare(u.t) }

func (t Time) String() string { return t.t.Format(time.RFC3339Nano) }

// Value is NULL for the zero Time, else unix microseconds.
func (t Time) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.t.UnixMicro(), nil
}

// Scan accepts NULL (the zero Time) and int64, the only values a STRICT
// INTEGER column holds.
func (t *Time) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*t = Time{}
	case int64:
		*t = FromTime(time.UnixMicro(v))
	default:
		return fmt.Errorf("storage.Time: cannot scan %T", src)
	}
	return nil
}

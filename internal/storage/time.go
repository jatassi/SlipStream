package storage

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Time is a UTC instant at microsecond precision, stored as INTEGER unix
// microseconds.
//
// The zero Time is SQL NULL, both ways: sqlc's column override cannot tell
// a nullable column apart, so nullability lives in the value. Two Times are
// equal exactly when == says so, because every Time is normalized.
type Time struct {
	t time.Time
}

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

func (t Time) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.t.UnixMicro(), nil
}

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

package storage

import (
	"database/sql/driver"
	"fmt"
	"uuid"
)

// UUID is a UUIDv7 stored as a 16-byte BLOB (ADR 0021). Ids that cross
// Nodes or appear in URLs embed it in their own type, declared in the
// owner's internal/db/ids.go:
//
//	type TransferID struct{ storage.UUID }
//
// so Scan and Value are promoted and a TransferID is not a WorkID. Its byte
// order is its creation order, so ORDER BY on the column sorts by time.
//
// The zero UUID is SQL NULL, for the same reason as the zero Time.
type UUID struct {
	u uuid.UUID
}

func NewUUID() UUID { return UUID{u: uuid.NewV7()} }

func ParseUUID(s string) (UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return UUID{}, fmt.Errorf("storage.UUID: %w", err)
	}
	return UUID{u: u}, nil
}

func (u UUID) String() string { return u.u.String() }

func (u UUID) IsZero() bool { return u.u == uuid.Nil() }

func (u UUID) Compare(v UUID) int { return u.u.Compare(v.u) }

func (u UUID) Value() (driver.Value, error) {
	if u.IsZero() {
		return nil, nil
	}
	b := u.u
	return b[:], nil
}

// Scan accepts NULL and a 16-byte []byte, which it copies because the
// driver may reuse the buffer.
func (u *UUID) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*u = UUID{}
	case []byte:
		if len(v) != len(u.u) {
			return fmt.Errorf("storage.UUID: cannot scan %d bytes", len(v))
		}
		u.u = uuid.UUID(v)
	default:
		return fmt.Errorf("storage.UUID: cannot scan %T", src)
	}
	return nil
}

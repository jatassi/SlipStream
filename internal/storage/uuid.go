package storage

import (
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"sync"
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

// minted is the timestamp of the last id NewUUID minted, in uuid.NewV7's
// units of 1/4096 ms.
var minted struct {
	mu sync.Mutex
	at uint64
}

// NewUUID mints an id that sorts after every id minted before it in this
// process. uuid.NewV7 alone starts again from the clock when the clock
// steps back, as it does between synctest bubbles that run side by side.
func NewUUID() UUID {
	u := uuid.NewV7()
	hi := binary.BigEndian.Uint64(u[:8])
	minted.mu.Lock()
	at := max(hi>>16<<12|hi&0x0fff, minted.at+1)
	minted.at = at
	minted.mu.Unlock()
	binary.BigEndian.PutUint64(u[:8], at>>12<<16|hi&0xf000|at&0x0fff)
	return UUID{u: u}
}

func ParseUUID(s string) (UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return UUID{}, fmt.Errorf("storage.UUID: %w", err)
	}
	return UUID{u: u}, nil
}

// UUIDFromBytes is the UUID whose 16 bytes are b, as it crosses the link.
func UUIDFromBytes(b []byte) (UUID, error) {
	if len(b) != len(uuid.UUID{}) {
		return UUID{}, fmt.Errorf("storage.UUID: %d bytes, want 16", len(b))
	}
	return UUID{u: uuid.UUID(b)}, nil
}

func (u UUID) Bytes() []byte {
	b := u.u
	return b[:]
}

func (u UUID) String() string { return u.u.String() }

func (u UUID) IsZero() bool { return u.u == uuid.Nil() }

func (u UUID) Compare(v UUID) int { return u.u.Compare(v.u) }

// MarshalText is u's canonical string, so a UUID in stored JSON or on the
// wire reads as its id.
func (u UUID) MarshalText() ([]byte, error) { return []byte(u.String()), nil }

func (u *UUID) UnmarshalText(b []byte) error {
	v, err := ParseUUID(string(b))
	if err != nil {
		return err
	}
	*u = v
	return nil
}

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

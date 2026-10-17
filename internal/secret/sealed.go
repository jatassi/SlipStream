package secret

import (
	"crypto/cipher"
	"crypto/rand"
	"database/sql/driver"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"

	"golang.org/x/crypto/chacha20poly1305"
)

// Sealer seals and opens stored credentials under the key derived from
// library.key for that purpose. The Node builds one per open library.db
// and hands it to every owner of a sealed column.
type Sealer struct {
	aead cipher.AEAD
}

func (k *Keys) Sealer() *Sealer {
	aead, err := chacha20poly1305.NewX(k.derive("slipstream credential key"))
	if err != nil {
		panic(fmt.Sprintf("secret: build the sealer: %v", err))
	}
	return &Sealer{aead: aead}
}

const sealedVersion = 1

// Sealed is a credential as stored: a version byte, a 24-byte nonce, then
// the ciphertext and its tag. The zero Sealed is SQL NULL both ways, as
// storage.Time is, so a nullable *_sealed column needs no pointer.
type Sealed struct{ b []byte }

func (s Sealed) Set() bool { return len(s.b) > 0 }

func (s Sealed) Value() (driver.Value, error) {
	if !s.Set() {
		return nil, nil
	}
	return s.b, nil
}

func (s *Sealed) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*s = Sealed{}
	case []byte:
		*s = Sealed{b: slices.Clone(v)}
	default:
		return fmt.Errorf("secret.Sealed: cannot scan %T", src)
	}
	return nil
}

func (s Sealed) LogValue() slog.Value { return slog.StringValue("[sealed]") }

// Column is one sealed column, "<table>.<column>_sealed", declared once
// beside the queries that read it.
type Column struct{ qualified string }

var sealedColumn = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*_sealed$`)

// NewColumn panics unless qualified names a *_sealed column, so the
// associated data and the credential-columns lint agree on what one is.
func NewColumn(qualified string) Column {
	if !sealedColumn.MatchString(qualified) {
		panic(fmt.Sprintf("secret: %q is not <table>.<column>_sealed", qualified))
	}
	return Column{qualified: qualified}
}

// AD binds a sealed value to one row of one column: "<table>.<column>:<id>".
// It never uses SQLite's rowid, which VACUUM INTO may renumber.
type AD struct{ b []byte }

// Row binds to the row whose primary key id renders as id.
func (c Column) Row(id string) AD { return AD{b: []byte(c.qualified + ":" + id)} }

// Single binds to a one-row table's row, id 1.
func (c Column) Single() AD { return c.Row("1") }

func (s *Sealer) Seal(plaintext []byte, ad AD) Sealed {
	out := make([]byte, 1+chacha20poly1305.NonceSizeX, 1+chacha20poly1305.NonceSizeX+len(plaintext)+s.aead.Overhead())
	out[0] = sealedVersion
	_, _ = rand.Read(out[1:])
	return Sealed{b: s.aead.Seal(out, out[1:], plaintext, ad.b)}
}

var ErrUnsealable = errors.New("secret: the sealed value does not open under this key and row")

// Open fails with ErrUnsealable for a value moved to another row or column,
// sealed under another key, or of a version it doesn't know.
func (s *Sealer) Open(v Sealed, ad AD) ([]byte, error) {
	if len(v.b) < 1+chacha20poly1305.NonceSizeX || v.b[0] != sealedVersion {
		return nil, ErrUnsealable
	}
	nonce, ciphertext := v.b[1:1+chacha20poly1305.NonceSizeX], v.b[1+chacha20poly1305.NonceSizeX:]
	plaintext, err := s.aead.Open(nil, nonce, ciphertext, ad.b)
	if err != nil {
		return nil, ErrUnsealable
	}
	return plaintext, nil
}

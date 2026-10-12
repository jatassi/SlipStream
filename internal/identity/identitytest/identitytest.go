// Package identitytest gives tests a Member and fresh sessions, until
// Accounts builds the real way to add them.
package identitytest

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/netip"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// neverMatches has argon2id's shape and matches no PIN, so a Member added
// here signs in only through Session.
const neverMatches = "$argon2id$v=19$m=19456,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

// Member adds a Member Account with email and returns its id.
func Member(t testing.TB, lib *storage.DB[storage.Library], email string) identity.AccountID {
	t.Helper()
	id := identity.AccountID{UUID: storage.NewUUID()}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).InsertAccount(ctx, db.InsertAccountParams{
			ID:          id,
			Email:       email,
			DisplayName: "Member",
			Admin:       false,
			PinHash:     neverMatches,
			CreatedAt:   storage.Now(),
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// Session starts a fresh session for account and returns its token, the
// session cookie's value.
func Session(t testing.TB, lib *storage.DB[storage.Library], account identity.AccountID) string {
	t.Helper()
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	token := base64.RawURLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(token))
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		now := storage.Now()
		return db.New(tx).InsertSession(ctx, db.InsertSessionParams{
			ID:         identity.SessionID{UUID: storage.NewUUID()},
			TokenHash:  hash[:],
			AccountID:  account,
			Address:    netip.IPv6Loopback().String(),
			CreatedAt:  now,
			LastSeenAt: now,
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

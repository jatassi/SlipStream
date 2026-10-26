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
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/storage"
)

// neverMatches has argon2id's shape and matches no PIN, so a Member added
// here signs in only through Session.
const neverMatches = "$argon2id$v=19$m=19456,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

func Member(t testing.TB, lib *storage.DB[storage.Library], email string) identity.AccountID {
	t.Helper()
	return account(t, lib, email, false)
}

// Admin adds an enabled Admin, as a claim does, without a session or an
// API key.
func Admin(t testing.TB, lib *storage.DB[storage.Library], email string) identity.AccountID {
	t.Helper()
	return account(t, lib, email, true)
}

func account(t testing.TB, lib *storage.DB[storage.Library], email string, admin bool) identity.AccountID {
	t.Helper()
	id := identity.AccountID{UUID: storage.NewUUID()}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).InsertAccount(ctx, db.InsertAccountParams{
			ID:          id,
			Email:       email,
			DisplayName: "Member",
			Admin:       admin,
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
	return session(t, lib, account, time.Now(), false)
}

// LapsedSession is a fresh session whose last confirm is older than any
// re-confirm window.
func LapsedSession(t testing.TB, lib *storage.DB[storage.Library], account identity.AccountID) string {
	t.Helper()
	return session(t, lib, account, time.Now().Add(-25*time.Hour), false)
}

// PINRequiredSession is a session that must set a new PIN, as a passkey
// sign-in on a turned-off PIN starts.
func PINRequiredSession(t testing.TB, lib *storage.DB[storage.Library], account identity.AccountID) string {
	t.Helper()
	return session(t, lib, account, time.Now(), true)
}

func session(t testing.TB, lib *storage.DB[storage.Library], account identity.AccountID, confirmed time.Time, pinRequired bool) string {
	t.Helper()
	token := newToken()
	hash := sha256.Sum256([]byte(token))
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		now := storage.Now()
		return db.New(tx).InsertSession(ctx, db.InsertSessionParams{
			ID:          identity.SessionID{UUID: storage.NewUUID()},
			TokenHash:   hash[:],
			AccountID:   account,
			Address:     netip.IPv6Loopback().String(),
			CreatedAt:   now,
			LastSeenAt:  now,
			ConfirmedAt: storage.FromTime(confirmed),
			PinRequired: pinRequired,
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// APIKey replaces the claimed Node's API key with a fresh one and returns
// it, so a call that regenerates the key can't leave its caller holding a
// dead one.
func APIKey(t testing.TB, lib *storage.DB[storage.Library]) string {
	t.Helper()
	key := "ssk_" + newToken()
	hash := sha256.Sum256([]byte(key))
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).ReplaceAPIKey(ctx, db.ReplaceAPIKeyParams{TokenHash: hash[:], CreatedAt: storage.Now()})
	})
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

package identity_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestEverySignInSettingIsDeclaredSoundly(t *testing.T) {
	for _, d := range []interface{ Check() error }{
		identity.PINWaitAfter, identity.PINWaitLength, identity.PINOffAfter, identity.AddressFailures,
		identity.AddressWindow, identity.ReconfirmWindow, identity.SessionLifetime,
		identity.SignupLinkLifetime, identity.ResetLinkLifetime, identity.EmailLinkLifetime,
	} {
		if err := d.Check(); err != nil {
			t.Error(err)
		}
	}
}

func TestTheOverrideMigrationKeepsOnlyChangedSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	log := slog.New(slog.DiscardHandler)
	old, err := storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{identity.Schema().Through(2)}, nil, storage.DefaultTimings(), log)
	if err != nil {
		t.Fatal(err)
	}
	err = old.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE identity_settings SET pin_wait_after = 4, session_lifetime_seconds = 3600, address = 'https://media.example.com'")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	lib, err := storage.OpenLibrary(t.Context(), path, []storage.Schema[storage.Library]{identity.Schema()}, nil, storage.DefaultTimings(), log)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lib.Close() })
	var row struct {
		pinWaitAfter, pinWait, pinOffAfter, sessionLifetime *int64
		address                                             string
	}
	err = lib.QueryRowContext(t.Context(), "SELECT pin_wait_after, pin_wait_seconds, pin_off_after, session_lifetime_seconds, address FROM identity_settings").
		Scan(&row.pinWaitAfter, &row.pinWait, &row.pinOffAfter, &row.sessionLifetime, &row.address)
	if err != nil {
		t.Fatal(err)
	}
	if row.pinWaitAfter == nil || *row.pinWaitAfter != 4 || row.sessionLifetime == nil || *row.sessionLifetime != 3600 {
		t.Errorf("changed settings = %v, %v, want 4 and 3600 kept", row.pinWaitAfter, row.sessionLifetime)
	}
	if row.pinWait != nil || row.pinOffAfter != nil {
		t.Errorf("default settings = %v, %v, want null", row.pinWait, row.pinOffAfter)
	}
	if row.address != "https://media.example.com" {
		t.Errorf("address = %q, want it kept", row.address)
	}
}

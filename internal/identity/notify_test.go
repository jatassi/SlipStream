package identity_test

import (
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestASenderReadsTheAddressAndWhenAnAccountWasMade(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := t.Context()
		lib := storagetest.Library(t, secret.Schema(), identity.Schema())
		keys, err := secret.Open(ctx, lib, filepath.Join(t.TempDir(), secret.KeyFile))
		if err != nil {
			t.Fatal(err)
		}
		svc, err := identity.New(ctx, identity.Config{Library: lib, Pepper: keys.PINPepper(), Hub: identitytest.Hub(t)})
		if err != nil {
			t.Fatal(err)
		}
		if got, err := identity.ReadAddress(ctx, lib); err != nil || !got.IsZero() {
			t.Fatalf("before it is set the Address reads %q, %v; want none", got, err)
		}
		address, err := identity.ParseAddress("https://media.example.com")
		if err != nil {
			t.Fatal(err)
		}
		if err := svc.SetAddress(ctx, identity.Admin{}, address); err != nil {
			t.Fatal(err)
		}
		if got, err := identity.ReadAddress(ctx, lib); err != nil || got != address {
			t.Fatalf("the Address reads %q, %v; want %q", got, err, address)
		}

		time.Sleep(time.Hour)
		made := storage.Now()
		sam := identitytest.Admin(t, lib, "sam@example.test")
		if got, err := identity.CreatedAt(ctx, lib, sam); err != nil || got != made {
			t.Fatalf("Sam's Account was made at %v, %v; want %v", got, err, made)
		}
	})
}

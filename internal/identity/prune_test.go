package identity_test

import (
	"context"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// recorder records whose sessions changed and which streams ended.
type recorder struct {
	mu        sync.Mutex
	published []string
	ended     int
}

func (r *recorder) Publish(_ context.Context, account storage.UUID, _ identity.SessionsChanged) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.published = append(r.published, account.String())
}

func (r *recorder) EndSession(storage.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ended++
}

func TestPruneSessionsDeletesEveryAccountsIdleSessions(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, secret.Schema(), identity.Schema())
		keys, err := secret.Open(t.Context(), lib, filepath.Join(t.TempDir(), secret.KeyFile))
		if err != nil {
			t.Fatal(err)
		}
		events := &recorder{}
		svc, err := identity.New(t.Context(), lib, keys.PINPepper(), events, events)
		if err != nil {
			t.Fatal(err)
		}
		grace := identitytest.Member(t, lib, "grace@example.test")
		hopper := identitytest.Member(t, lib, "hopper@example.test")
		identitytest.Session(t, lib, grace)
		identitytest.Session(t, lib, grace)
		time.Sleep(30 * 24 * time.Hour)
		identitytest.Session(t, lib, hopper)
		time.Sleep(time.Second)

		if err := svc.PruneSessions(t.Context()); err != nil {
			t.Fatal(err)
		}
		var left []string
		rows, err := lib.QueryContext(t.Context(), "SELECT a.email FROM identity_session s JOIN identity_account a ON a.id = s.account_id")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var email string
			if err := rows.Scan(&email); err != nil {
				t.Fatal(err)
			}
			left = append(left, email)
		}
		if want := []string{"hopper@example.test"}; !slices.Equal(left, want) {
			t.Fatalf("sessions left for %q, want %q", left, want)
		}
		if want := []string{grace.String()}; events.ended != 2 || !slices.Equal(events.published, want) {
			t.Fatalf("ended %d streams and told %q, want 2 and %q", events.ended, events.published, want)
		}
	})
}

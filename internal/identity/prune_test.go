package identity_test

import (
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestPruneSessionsDeletesEveryAccountsIdleSessions(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := storagetest.Library(t, secret.Schema(), identity.Schema())
		keys, err := secret.Open(t.Context(), lib, filepath.Join(t.TempDir(), secret.KeyFile))
		if err != nil {
			t.Fatal(err)
		}
		hub := identitytest.Hub(t)
		svc, err := identity.New(t.Context(), identity.Config{Library: lib, Pepper: keys.PINPepper(), Hub: hub})
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

		var streams []*live.Subscription
		for _, id := range query[storage.UUID](t, lib, "SELECT id FROM identity_session WHERE account_id = ?", grace) {
			streams = append(streams, hub.Subscribe(live.Viewer{Account: grace.UUID, Session: id}, ""))
		}
		elsewhere := hub.Subscribe(live.Viewer{Account: grace.UUID, Session: storage.NewUUID()}, "")
		if err := svc.PruneSessions(t.Context()); err != nil {
			t.Fatal(err)
		}

		left := query[string](t, lib, "SELECT a.email FROM identity_session s JOIN identity_account a ON a.id = s.account_id")
		if want := []string{"hopper@example.test"}; !slices.Equal(left, want) {
			t.Fatalf("sessions left for %q, want %q", left, want)
		}
		if len(streams) != 2 {
			t.Fatalf("grace has %d sessions, want 2", len(streams))
		}
		for _, s := range streams {
			if got := next(t, s); got.Name != live.SessionEndedEvent.Name() || got.Data != (live.SessionEnded{Reason: live.Expired}) {
				t.Fatalf("stream got %s %+v, want the session ended as expired", got.Name, got.Data)
			}
		}
		if got := next(t, elsewhere); got.Name != identity.SessionsChangedEvent.Name() {
			t.Fatalf("grace's other stream got %s, want %s", got.Name, identity.SessionsChangedEvent.Name())
		}
	})
}

// next is s's next message past the resync a fresh subscription starts with.
func next(t *testing.T, s *live.Subscription) live.Message {
	t.Helper()
	for {
		m, err := s.Next(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if m.Name != live.ResyncEvent.Name() {
			return m
		}
	}
}

func query[T any](t *testing.T, lib *storage.DB[storage.Library], sql string, args ...any) []T {
	t.Helper()
	rows, err := lib.QueryContext(t.Context(), sql, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []T
	for rows.Next() {
		var v T
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

package nodes

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// TestAWriteOffForfeitsUntilTheOwnerHoldsNothing writes off seedbox's
// Downloading while seedbox holds 3 torrents and still has Downloading
// ticked: the same Write unticks it, and the forfeit stands while the
// owner counts torrents on seedbox, which drains them meanwhile. Once the
// owner counts none, settling deletes the forfeit.
func TestAWriteOffForfeitsUntilTheOwnerHoldsNothing(t *testing.T) {
	var torrents atomic.Int64
	torrents.Store(3)
	s, lib, seedbox := downloadingSeedbox(t, &torrents)
	writeOff := WriteOff{Node: seedbox, Roles: role.Of(role.Downloading)}
	drains := func(what string, want ...role.Name) {
		t.Helper()
		scope, err := s.scope(t.Context(), lib, seedbox)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(scope.Draining, want) || scope.Name != "seedbox" {
			t.Errorf("%s, seedbox's node scope drains %v as %q, want %v as seedbox", what, scope.Draining, scope.Name, want)
		}
	}

	out, err := s.Apply(t.Context(), writeOff)
	if err != nil {
		t.Fatal(err)
	}
	if out.Next[seedbox].Has(role.Downloading) || out.Forfeit[seedbox] != role.Of(role.Downloading) {
		t.Fatalf("the write-off stores %v and forfeits %v, want Downloading unticked and forfeited on seedbox", out.Next, out.Forfeit)
	}
	if st := s.Who(role.Downloading).State(seedbox); st != WritingOff || !s.WrittenOff(role.Downloading, seedbox) {
		t.Fatalf("after the write-off seedbox stands %v in Downloading, want writing-off and written off", st)
	}
	drains("writing off", role.Downloading)

	again, err := s.Apply(t.Context(), writeOff)
	if err != nil || !again.Same(mustSituation(t, s)) {
		t.Fatalf("writing off twice = %v, %v; want a change that stores nothing new", again, err)
	}

	torrents.Store(0)
	if err := s.settle(t.Context()); err != nil {
		t.Fatal(err)
	}
	if st := s.Who(role.Downloading).State(seedbox); st != NoState || s.WrittenOff(role.Downloading, seedbox) {
		t.Fatalf("once the owner holds nothing seedbox stands %v in Downloading, want none and nothing written off", st)
	}
	drains("settled")

	_, err = s.Apply(t.Context(), writeOff)
	if blocked, ok := errors.AsType[*BlockedError](err); !ok || len(blocked.Outcome.Checks) != 1 || blocked.Outcome.Checks[0].Reason != NothingHeld {
		t.Fatalf("writing off what seedbox no longer holds = %v, want blocked with nothing_held", err)
	}
}

// downloadingSeedbox is home with seedbox paired and ticked for
// Downloading, whose owner counts torrents on seedbox and none elsewhere.
func downloadingSeedbox(t *testing.T, torrents *atomic.Int64) (*Service, *storage.DB[storage.Library], ID) {
	t.Helper()
	seedbox := ID{UUID: storage.NewUUID()}
	downloading := Unowned(role.Downloading)
	downloading.Held = func(_ context.Context, _ storage.Reader[storage.Library], node ID) (Held, error) {
		if node != seedbox {
			return Held{}, nil
		}
		return Held{Units: torrents.Load(), Noun: Noun{"torrent", "torrents"}}, nil
	}
	owners := []Owner{downloading}
	for _, r := range role.Names() {
		if r != role.Downloading {
			owners = append(owners, Unowned(r))
		}
	}
	lib := storagetest.Library(t, Schema(), role.LibrarySchema())
	s, err := Open(t.Context(), Config{Library: lib, Name: "home", Link: noLink{}, Changed: unheard{}, Owners: owners, Log: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatal(err)
	}
	err = lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'seedbox', 0)", seedbox); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO nodes_role (node_id, role, assigned_at) VALUES (?, 'downloading', 0)", seedbox)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.reload(t.Context()); err != nil {
		t.Fatal(err)
	}
	return s, lib, seedbox
}

func mustSituation(t *testing.T, s *Service) Situation {
	t.Helper()
	sit, err := s.situation(t.Context(), s.cfg.Library, s.Board())
	if err != nil {
		t.Fatal(err)
	}
	return sit
}

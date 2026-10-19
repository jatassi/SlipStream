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
	writtenOffIn(t, lib, role.Downloading, seedbox)
	writtenOffIn(t, lib, role.Storage)

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

// TestRemovingANodeThatHoldsWorkNeedsAWriteOff removes seedbox while it
// seeds 3 torrents: it is refused until the removal writes them off, and
// the Controller is never removed.
func TestRemovingANodeThatHoldsWorkNeedsAWriteOff(t *testing.T) {
	var torrents atomic.Int64
	torrents.Store(3)
	s, lib, seedbox := downloadingSeedbox(t, &torrents)
	_, err := s.Apply(t.Context(), Remove{Node: seedbox})
	if blocked, ok := errors.AsType[*BlockedError](err); !ok || len(blocked.Outcome.Checks) != 1 || blocked.Outcome.Checks[0].Reason != HoldsWork {
		t.Fatalf("removing seedbox while it seeds = %v, want blocked with node_holds_work", err)
	}
	if _, err := s.Get(t.Context(), seedbox); err != nil {
		t.Fatalf("the refused removal took seedbox away: %v", err)
	}
	out, err := s.Apply(t.Context(), Remove{Node: seedbox, WriteOff: true})
	if err != nil {
		t.Fatal(err)
	}
	var said []string
	for _, e := range out.Effects {
		said = append(said, e.Words())
	}
	want := []string{
		"The 3 torrents seedbox holds for Downloading are written off. No files are deleted.",
		"seedbox is removed. It can join again with a new Join Code.",
	}
	if !slices.Equal(said, want) {
		t.Fatalf("the removal says %q, want %q", said, want)
	}
	if !s.WrittenOff(role.Downloading, seedbox) {
		t.Fatal("seedbox's torrents read as kept once it is removed with a write-off")
	}
	writtenOffIn(t, lib, role.Storage, seedbox)
	_, err = s.Apply(t.Context(), Remove{Node: s.Self(), WriteOff: true})
	if blocked, ok := errors.AsType[*BlockedError](err); !ok || len(blocked.Outcome.Checks) != 1 || blocked.Outcome.Checks[0].Reason != IsController {
		t.Fatalf("removing the Controller = %v, want blocked with node_is_controller", err)
	}
}

// writtenOffIn checks the library reads exactly want as written off in
// the Role named name.
func writtenOffIn(t *testing.T, lib *storage.DB[storage.Library], name role.Name, want ...ID) {
	t.Helper()
	got, err := WrittenOffIn(t.Context(), lib, name)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("the library reads %v written off in %s, want %v", got, name, want)
	}
	for _, id := range want {
		if !got[id] {
			t.Fatalf("the library reads %v written off in %s, want %v", got, name, want)
		}
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

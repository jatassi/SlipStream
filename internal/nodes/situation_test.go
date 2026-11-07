package nodes

import (
	"context"
	"log/slog"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type noLink struct{}

func (noLink) State(link.Key) link.State { return link.Down{} }

func (noLink) Subscribe(context.Context) <-chan link.Change { return nil }

func (noLink) Advertised() (link.Addresses, bool) { return link.Addresses{}, false }

func (noLink) Reset(link.Key) {}

func (noLink) Keep(link.Target) {}

func (noLink) Drop(link.Key) {}

func (noLink) Invite(context.Context, link.Target) error { return nil }
func (noLink) Stats(link.Key) (link.Stats, bool)         { return link.Stats{}, false }
func (noLink) LaneBytes(link.Key) link.Stats             { return link.Stats{} }

type unheard struct{}

func (unheard) Publish(context.Context, Changed) {}

// counted is an owner that holds n units of noun on on and nothing
// elsewhere.
func counted(r role.Name, on ID, n int64, noun Noun) Owner {
	o := Unowned(r)
	o.Held = func(_ context.Context, _ storage.Reader[storage.Library], node ID) (Held, error) {
		if node != on {
			return Held{}, nil
		}
		return Held{Units: n, Noun: noun}, nil
	}
	return o
}

// Plan's Seats come from the rows, the board and the owners: seedbox, out
// of step, holds 3 torrents that drain and 2 Indexers that move when it
// lets its Roles go.
func TestSeatsCarryTiedWorkAsHeldAndPortableWorkAsMoves(t *testing.T) {
	torrents, indexers := Noun{"torrent", "torrents"}, Noun{"Indexer", "Indexers"}
	seedbox := ID{UUID: storage.NewUUID()}
	owners := []Owner{counted(role.Downloading, seedbox, 3, torrents), counted(role.Searching, seedbox, 2, indexers)}
	for _, r := range role.Names() {
		if r != role.Downloading && r != role.Searching {
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
		_, err := tx.ExecContext(ctx, "INSERT INTO nodes_role (node_id, role, assigned_at) VALUES (?, 'searching', 0), (?, 'downloading', 0)", seedbox, seedbox)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.reload(t.Context()); err != nil {
		t.Fatal(err)
	}
	sit, err := s.situation(t.Context(), lib, s.Board())
	if err != nil {
		t.Fatal(err)
	}
	if len(sit.Seats) != 2 || sit.Controller != s.Self() {
		t.Fatalf("the situation has %d seats with Controller %s, want home and seedbox with home", len(sit.Seats), sit.Controller)
	}
	homeSeat, boxSeat := sit.Seats[0], sit.Seats[1]
	if homeSeat.ID == seedbox {
		homeSeat, boxSeat = boxSeat, homeSeat
	}
	if homeSeat.ID != s.Self() || homeSeat.Away || len(homeSeat.Held) != 0 || len(homeSeat.Moves) != 0 {
		t.Errorf("home's seat = %+v, want in step and holding nothing", homeSeat)
	}
	wantHeld := map[role.Name]Held{role.Downloading: {Units: 3, Noun: torrents}}
	wantMoves := map[role.Name]Held{role.Searching: {Units: 2, Noun: indexers}}
	if boxSeat.ID != seedbox || !boxSeat.Away || boxSeat.Assigned != role.Of(role.Searching, role.Downloading) ||
		!reflect.DeepEqual(boxSeat.Held, wantHeld) || !reflect.DeepEqual(boxSeat.Moves, wantMoves) {
		t.Errorf("seedbox's seat = %+v, want away, holding Searching and Downloading, Held %v, Moves %v", boxSeat, wantHeld, wantMoves)
	}

	out := Plan(sit, SetRoles{Basis: sit.Basis(), Want: Roles{seedbox: 0}})
	want := []Effect{
		LetsGo{Node: boxSeat.Named, Role: role.Searching},
		Drains{Node: boxSeat.Named, Role: role.Downloading, Held: wantHeld[role.Downloading]},
		Moves{Role: role.Searching, From: boxSeat.Named, To: homeSeat.Named, Held: wantMoves[role.Searching]},
	}
	if !reflect.DeepEqual(out.Effects, want) {
		t.Errorf("unticking seedbox's Roles plans %v, want %v", out.Effects, want)
	}
}

// Unticking Searching on seedbox, which has Indexers homed on it, moves
// them in the same Write to home, the Controller that keeps Searching.
func TestApplyMovesPortableWorkToTheHeir(t *testing.T) {
	seedbox := ID{UUID: storage.NewUUID()}
	searching := counted(role.Searching, seedbox, 2, Noun{"Indexer", "Indexers"})
	var moved [][2]ID
	searching.Unassigned = func(_ context.Context, _ *storage.Tx[storage.Library], node, heir ID) error {
		moved = append(moved, [2]ID{node, heir})
		return nil
	}
	owners := []Owner{searching}
	for _, r := range role.Names() {
		if r != role.Searching {
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
		_, err := tx.ExecContext(ctx, "INSERT INTO nodes_role (node_id, role, assigned_at) VALUES (?, 'searching', 0)", seedbox)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.reload(t.Context()); err != nil {
		t.Fatal(err)
	}
	board, err := s.RoleBoard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(t.Context(), SetRoles{Basis: board.Basis(), Want: Roles{seedbox: 0}}); err != nil {
		t.Fatal(err)
	}
	if want := [][2]ID{{seedbox, s.Self()}}; !reflect.DeepEqual(moved, want) {
		t.Fatalf("the Searching owner moved %v, want seedbox's Indexers to home", moved)
	}
}

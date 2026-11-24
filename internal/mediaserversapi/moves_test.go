package mediaserversapi_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	fakejellyfin "github.com/jatassi/slipstream/fakes/jellyfin"
	fakeplex "github.com/jatassi/slipstream/fakes/plex"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/mstest"
	wire "github.com/jatassi/slipstream/internal/providers/jellyfin"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

const (
	vault = "/media/Vault"
	// faust is a film already in Vault, so Den scans the folder a report
	// in Vault resolves to.
	faust = vault + "/Faust/Faust.mkv"
)

// moving is Den with the Shelves and Vault roots shown and scanned, each
// with its own folder: in one library, or apart, each in its own.
type moving struct {
	shelf
	vault  roots.ID
	flavor wire.Flavor
}

func newMoving(t *testing.T, kind string, apart bool, placed ...string) moving {
	t.Helper()
	r := newRig(t)
	srv := family()
	if apart {
		srv.Libraries = append(srv.Libraries, fakejellyfin.Library{Name: "Vault", Type: "homevideos", Folders: []string{vault}})
	} else {
		srv.Libraries[0].Folders = append(srv.Libraries[0].Folders, vault)
	}
	m := moving{shelf: shelf{rig: r}, flavor: wire.Jellyfin}
	if kind == "emby" {
		m.den, m.flavor = r.w.Emby(t, host, srv, placed...), wire.Emby
	} else {
		m.den = r.w.Jellyfin(t, host, srv, placed...)
	}
	draft := `{"kind":"` + kind + `","name":"Den","address":"` + address + `","reachingNodeId":"` + r.w.Seedbox.String() + `","apiKey":"` + denKey + `"}`
	status, body := r.o.Call(t, http.MethodPost, servers, draft)
	if status != http.StatusCreated {
		t.Fatalf("add Den = %d %s", status, body)
	}
	m.id = decode[serverOut](t, body).ID
	m.root, m.vault = r.w.Root(t, "Shelves").ID, r.w.Root(t, "Vault").ID
	mappings := `{"mappings":[{"rootId":"` + m.root.String() + `","shown":true,"serverFolder":"` + shelves + `"},` +
		`{"rootId":"` + m.vault.String() + `","shown":true,"serverFolder":"` + vault + `"}]}`
	if status, reply := m.o.Call(t, http.MethodPut, servers+"/"+m.id+"/mappings", mappings); status != http.StatusOK {
		t.Fatalf("save the mappings = %d %s", status, reply)
	}
	m.w.Scanned(t)
	settle(15 * time.Second)
	return m
}

func (m moving) server(t *testing.T) link.ServerID {
	t.Helper()
	u, err := storage.ParseUUID(m.id)
	if err != nil {
		t.Fatal(err)
	}
	return link.ServerID{UUID: u}
}

// copied copies the bytes at rel in Shelves to Vault, then tells the
// queue the file switched to Vault, as a Move's switch does.
func (m moving) copied(t *testing.T, file files.ID, rel storing.Rel) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(m.den.Media, "Shelves", filepath.FromSlash(rel.String())))
	if err != nil {
		t.Fatal(err)
	}
	place(t, m.den.Media, "Vault/"+rel.String())
	if err := os.WriteFile(filepath.Join(m.den.Media, "Vault", filepath.FromSlash(rel.String())), b, 0o600); err != nil {
		t.Fatal(err)
	}
	m.w.Tell(t, files.PathChange{Kind: files.PathMoved, File: file, Root: m.vault, FromRoot: m.root, Path: rel, From: rel})
}

// sourceGone removes the bytes at rel in Shelves and stamps the Move's
// carries, as a Move's cleanup does.
func (m moving) sourceGone(t *testing.T, file files.ID, rel storing.Rel) {
	t.Helper()
	if err := os.Remove(filepath.Join(m.den.Media, "Shelves", filepath.FromSlash(rel.String()))); err != nil {
		t.Fatal(err)
	}
	err := m.w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return m.w.Moves().SourcesGone(ctx, tx, m.root, []files.ID{file})
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (m moving) snapshotted(t *testing.T, file files.ID) map[link.ServerID]bool {
	t.Helper()
	var out map[link.ServerID]bool
	err := m.w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		out, err = m.w.Moves().Snapshotted(ctx, r, []files.ID{file}, m.root)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

const movingWords = "Saved every user's watch data before the Move. SlipStream sends this change once the Move finishes."

// unscanned clears Vault's scan, as for a folder Den never scanned.
func (m moving) unscanned(t *testing.T) {
	t.Helper()
	err := m.w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE mediaservers_mapping SET scanned_at = NULL WHERE root_id = ?", m.vault)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestAMoveToAnotherRootKeepsEveryUsersWatchData moves a file both of
// Den's users watched from Shelves to Vault: SlipStream saves their data
// before the copy, a disk pass keeps it, Den hears nothing while the
// source still exists, and once the Move removed it Den holds the new
// item with the same data and collection, also when Den never scanned
// Vault and scans it first.
func TestAMoveToAnotherRootKeepsEveryUsersWatchData(t *testing.T) {
	for _, tc := range []struct {
		name  string
		kind  string
		apart bool
		fresh bool
	}{
		{"Jellyfin, in one library", "jellyfin", false, false},
		{"Jellyfin, across libraries", "jellyfin", true, false},
		{"Emby, across libraries", "emby", true, false},
		{"Jellyfin, into a folder it never scanned", "jellyfin", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				placed := []string{p + "Trip/Trip.mkv", faust}
				if tc.fresh {
					placed = placed[:1]
				}
				m := newMoving(t, tc.kind, tc.apart, placed...)
				if tc.fresh {
					m.unscanned(t)
				}
				srv := m.server(t)
				v := m.viewerOf(t, m.flavor)
				v.watch("Trip/Trip.mkv")
				rel := mustRel(t, "Trip/Trip.mkv")
				trip := m.w.Place(m.root, rel)
				m.w.Moves().PrepareMove(t.Context(), []files.ID{trip}, m.root, m.vault)
				m.w.Prepare(t)
				expectWords(t, change(t, m.changes(t), "Trip/Trip.mkv"), "prepared", movingWords)
				if got := m.snapshotted(t, trip); !reflect.DeepEqual(got, map[link.ServerID]bool{srv: true}) {
					t.Errorf("Snapshotted = %v, want Den's snapshot fresh", got)
				}
				m.copied(t, trip, rel)
				settle(5 * time.Minute)
				expectReported(t, m.den)
				m.sourceGone(t, trip, rel)
				settle(5 * time.Minute)
				sent := change(t, m.changes(t), "Trip/Trip.mkv")
				if sent.State != "sent" || !strings.HasPrefix(sent.Words, "Sent the Move to ") || !strings.HasSuffix(sent.Words, "Vault on home. Kept every user's watch data.") {
					t.Errorf("the Move's change is %s: %q", sent.State, sent.Words)
				}
				expectReported(t, m.den, "Deleted "+p+"Trip/Trip.mkv", "Created "+vault+"/Trip/Trip.mkv")
				v.kept(vault + "/Trip/Trip.mkv")
			})
		})
	}
}

// TestPlayingReadsAPlayByItsPathAndNotOnAServerAway plays a file whose
// play Den started before SlipStream knew the file, then takes seedbox,
// Den's Reaching Node, away while the play stays open.
func TestPlayingReadsAPlayByItsPathAndNotOnAServerAway(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := newMoving(t, "jellyfin", true, p+"Trip/Trip.mkv")
		m.den.Cue(t, "jellyfin-play", map[string]any{"path": p + "Trip/Trip.mkv", "user": "ada", "player": "Living room", "decision": link.DecisionDirectPlay})
		settle(every)
		trip := m.w.Place(m.root, mustRel(t, "Trip/Trip.mkv"))
		playing := func() map[files.ID][]link.ServerID {
			var out map[files.ID][]link.ServerID
			err := m.w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
				var err error
				out, err = m.w.Moves().Playing(ctx, r, []files.ID{trip})
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			return out
		}
		if got := playing(); !slices.Equal(got[trip], []link.ServerID{m.server(t)}) {
			t.Errorf("Playing = %v, want Den playing the file", got)
		}
		if got := m.snapshotted(t, trip); got[m.server(t)] {
			t.Errorf("Snapshotted with no snapshot = %v", got)
		}
		m.w.Stop(m.w.Seedbox)
		m.w.Offline(m.w.Seedbox)
		synctest.Wait()
		if got := playing(); len(got) != 0 {
			t.Errorf("Playing with seedbox away = %v, want nothing", got)
		}
	})
}

// TestASnapshotAPlayFollowsIsStale snapshots a file for a Move, then
// plays it on Den: the snapshot no longer holds what Den holds.
func TestASnapshotAPlayFollowsIsStale(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := newMoving(t, "jellyfin", true, p+"Trip/Trip.mkv")
		trip := m.w.Place(m.root, mustRel(t, "Trip/Trip.mkv"))
		m.w.Moves().PrepareMove(t.Context(), []files.ID{trip}, m.root, m.vault)
		if got := m.snapshotted(t, trip); !got[m.server(t)] {
			t.Fatalf("Snapshotted after PrepareMove = %v", got)
		}
		m.den.Cue(t, "jellyfin-play", map[string]any{"path": p + "Trip/Trip.mkv", "user": "ada", "player": "Living room", "decision": link.DecisionDirectPlay})
		settle(every)
		if got := m.snapshotted(t, trip); got[m.server(t)] {
			t.Errorf("Snapshotted after a play = %v, want stale", got)
		}
		if strings.Contains(change(t, m.changes(t), "Trip/Trip.mkv").State, "sent") {
			t.Error("the Move's prepared change was sent")
		}
	})
}

// newPlexMoving is newShows with the Archive root shown beside Shows, as
// a second location of the Shows section.
func newPlexMoving(t *testing.T, placed ...string) (shelf, roots.ID, mstest.Plex) {
	t.Helper()
	r := plexRig{rig: newRig(t)}
	srv := windowsPlex
	srv.Sections = []fakeplex.Section{{Key: "2", Title: "Shows", Type: "show", Folders: []string{`C:\Media\Shows`, `C:\Media\Archive`}, Scanned: true}}
	r.plex = r.w.Plex(t, srv, fakeplextv.Account{Username: "plex-admin", PlexPass: true})
	for _, rel := range append([]string{"Shows/Harbor/Season 01/Harbor - S01E01.mkv", "Archive/Quay/Season 01/Quay - S01E01.mkv"}, placed...) {
		place(t, r.plex.Server.Media, rel)
	}
	r.signIn(t)
	s := shelf{rig: r.rig, id: r.addPlex(t, r.w.Seedbox.String()).ID, root: r.w.Root(t, "Shows").ID}
	archive := r.w.Root(t, "Archive").ID
	mappings := `{"mappings":[{"rootId":"` + s.root.String() + `","shown":true,"serverFolder":"C:\\Media\\Shows"},` +
		`{"rootId":"` + archive.String() + `","shown":true,"serverFolder":"C:\\Media\\Archive"}]}`
	if status, reply := s.o.Call(t, http.MethodPut, servers+"/"+s.id+"/mappings", mappings); status != http.StatusOK {
		t.Fatalf("save the mappings = %d %s", status, reply)
	}
	s.w.Scanned(t)
	settle(15 * time.Second)
	return s, archive, r.plex
}

// TestAPlexMoveIsSafeOnlyOncePlexHoldsTheCopy switches a watched episode
// to Archive before its copy is there: past every confirm timeout Plex is
// not safe and its vacated folder is never refreshed. Once the copy lands
// the next refresh of the destination finds it and Plex is safe.
func TestAPlexMoveIsSafeOnlyOncePlexHoldsTheCopy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const ep = "Voyage/Season 01/Voyage - S01E01.mkv"
		s, archive, px := newPlexMoving(t, "Shows/"+ep)
		plexWatch(t, px, ep)
		rel := mustRel(t, ep)
		f := s.w.Place(s.root, rel)
		s.w.Tell(t, files.PathChange{Kind: files.PathMoved, File: f, Root: archive, FromRoot: s.root, Path: rel, From: rel})
		u, err := storage.ParseUUID(s.id)
		if err != nil {
			t.Fatal(err)
		}
		plexID := link.ServerID{UUID: u}
		safe := func() map[link.ServerID]bool {
			var out map[link.ServerID]bool
			err := s.w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
				var err error
				out, err = s.w.Moves().Safe(ctx, r, s.root, []files.ID{f})
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			return out
		}
		settle(3 * mstest.Timings.MovedConfirm)
		if got := safe(); !reflect.DeepEqual(got, map[link.ServerID]bool{plexID: false}) {
			t.Errorf("Safe with no copy = %v, want Plex not safe", got)
		}
		if got := refreshed(t, px); slices.ContainsFunc(got, func(r string) bool { return strings.Contains(r, `C:\Media\Shows\Voyage`) }) {
			t.Errorf("Plex refreshed the vacated folder: %v", got)
		}
		place(t, px.Server.Media, "Archive/"+ep)
		settle(mstest.Timings.MovedConfirm)
		if got := safe(); !reflect.DeepEqual(got, map[link.ServerID]bool{plexID: true}) {
			t.Errorf("Safe once the copy landed = %v, want Plex safe", got)
		}
	})
}

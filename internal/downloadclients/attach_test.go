package downloadclients_test

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

func dotTorrent(t *testing.T, name string) (m bencode.Metainfo, raw []byte) {
	t.Helper()
	raw, err := bencode.Encode(map[string]any{
		"announce": "https://tracker.test/announce/secret",
		"announce-list": []any{
			[]any{"https://tracker.test/announce/secret"},
			[]any{"udp://backup.test:6969/announce"},
		},
		"info": map[string]any{
			"name": name, "piece length": int64(16384), "pieces": "aaaaaaaaaaaaaaaaaaaa", "private": int64(1),
			"files": []any{
				map[string]any{"length": int64(1000), "path": []any{name + ".mkv"}},
				map[string]any{"length": int64(24), "path": []any{"Sample", "sample.mkv"}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if m, err = bencode.DecodeMetainfo(raw); err != nil {
		t.Fatal(err)
	}
	return m, raw
}

func noneWrittenOff(role.NodeID) bool { return false }

func attach(t *testing.T, lib library, want dc.Want) (dc.ID, role.NodeID, error) {
	t.Helper()
	var id dc.ID
	var node role.NodeID
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		id, node, err = dc.Attach(ctx, tx, want, storage.Now())
		return err
	})
	return id, node, err
}

func mustAttach(t *testing.T, lib library, want dc.Want) (dc.ID, role.NodeID) {
	t.Helper()
	id, node, err := attach(t, lib, want)
	if err != nil {
		t.Fatal(err)
	}
	return id, node
}

func exec(t *testing.T, lib library, query string, args ...any) {
	t.Helper()
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, query, args...)
		return err
	})
}

func removeTorrent(t *testing.T, lib library, id dc.ID, reason dc.RemoveReason) {
	t.Helper()
	exec(t, lib, "UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = ? WHERE id = ?", storage.Now(), reason, id)
}

func partTorrent(t *testing.T, lib library, id dc.ID) {
	t.Helper()
	exec(t, lib, "UPDATE downloadclients_torrent SET parted_at = ?, freed = 0 WHERE id = ?", storage.Now(), id)
}

func intentOf(t *testing.T, lib library, node role.NodeID) []dc.ID {
	t.Helper()
	var ids []dc.ID
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		in, err := dc.Intent(ctx, r, node)
		for _, tor := range in.GetTorrents() {
			u, err := storage.UUIDFromBytes(tor.GetId())
			if err != nil {
				return err
			}
			ids = append(ids, dc.ID{UUID: u})
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

func TestAttachAddsATorrentFromTheMetainfo(t *testing.T) {
	lib := open(t)
	a := addNode(t, lib, "seedbox")
	m, raw := dotTorrent(t, "Orchard")
	id, node := mustAttach(t, lib, dc.Want{Node: a, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	if node != a {
		t.Fatalf("Attach put the Torrent on %s, want the routed Node %s", node, a)
	}
	var name, trackers, wanted, contentType string
	var size, private, order int64
	var stored []byte
	err := lib.QueryRowContext(t.Context(), `SELECT name, size, private, trackers, wanted, content_type, added_order, metainfo
		FROM downloadclients_torrent WHERE id = ?`, id).Scan(&name, &size, &private, &trackers, &wanted, &contentType, &order, &stored)
	if err != nil {
		t.Fatal(err)
	}
	if name != "Orchard" || size != 1024 || private != 1 || trackers != `["tracker.test","backup.test"]` || wanted != "started" ||
		contentType != string(contenttypetest.ID) || order != 1 || !bytes.Equal(stored, raw) {
		t.Errorf("the new Torrent is %q %d B private=%d trackers=%s %s %s order %d, want Orchard 1024 B private=1 "+
			`["tracker.test","backup.test"] started fixture order 1 with the .torrent's bytes`, name, size, private, trackers, wanted, contentType, order)
	}
	if got := intentOf(t, lib, a); !slices.Equal(got, []dc.ID{id}) {
		t.Errorf("seedbox's intent lists %v, want the new Torrent %s", got, id)
	}
}

// TestAttachPrefersTheRoutedNodeThenAnyLiveOne walks each Torrent a hash
// can already have against a Grab routed to a.
func TestAttachPrefersTheRoutedNodeThenAnyLiveOne(t *testing.T) {
	for _, c := range []struct {
		name string
		// held places the hash's Torrents before the Grab attaches, and
		// returns the one it must bind to, zero for a new Torrent on a.
		held       func(t *testing.T, lib library, a, b role.NodeID, want dc.Want) dc.ID
		writtenOff bool
		err        error
	}{
		{name: "live on the routed Node and, added earlier, on another", held: func(t *testing.T, lib library, a, b role.NodeID, want dc.Want) dc.ID {
			want.Node = b
			first, _ := mustAttach(t, lib, want)
			id := dc.ID{UUID: storage.NewUUID()}
			exec(t, lib, `INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers, content_type,
				wanted, added_order, added_at) SELECT ?, ?, info_hash, metainfo, name, size, private, trackers, content_type, wanted,
				added_order + 1, added_at FROM downloadclients_torrent WHERE id = ?`, id, a, first)
			return id
		}},
		{name: "live only on another Node", held: func(t *testing.T, lib library, _, b role.NodeID, want dc.Want) dc.ID {
			want.Node = b
			id, _ := mustAttach(t, lib, want)
			return id
		}},
		{name: "live only on a written-off Node", writtenOff: true, held: func(t *testing.T, lib library, _, b role.NodeID, want dc.Want) dc.ID {
			want.Node = b
			mustAttach(t, lib, want)
			return dc.ID{}
		}},
		{name: "Removing on the routed Node", err: dc.ErrRemoving, held: func(t *testing.T, lib library, _, _ role.NodeID, want dc.Want) dc.ID {
			id, _ := mustAttach(t, lib, want)
			removeTorrent(t, lib, id, dc.ByAdmin)
			return dc.ID{}
		}},
		{name: "Removing on another Node, live on none", held: func(t *testing.T, lib library, _, b role.NodeID, want dc.Want) dc.ID {
			want.Node = b
			id, _ := mustAttach(t, lib, want)
			removeTorrent(t, lib, id, dc.ByAdmin)
			return dc.ID{}
		}},
		{name: "parted on the routed Node", held: func(t *testing.T, lib library, _, _ role.NodeID, want dc.Want) dc.ID {
			id, _ := mustAttach(t, lib, want)
			removeTorrent(t, lib, id, dc.ByAdmin)
			partTorrent(t, lib, id)
			return dc.ID{}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			lib := open(t)
			a, b := addNode(t, lib, "home"), addNode(t, lib, "seedbox")
			m, raw := dotTorrent(t, "Orchard")
			want := dc.Want{Node: a, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff}
			bound := c.held(t, lib, a, b, want)
			want.WrittenOff = func(n role.NodeID) bool { return c.writtenOff && n == b }
			id, node, err := attach(t, lib, want)
			if c.err != nil {
				if !errors.Is(err, c.err) {
					t.Fatalf("Attach = %v, want %v", err, c.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			expectBound(t, lib, id, node, bound, a)
		})
	}
}

// expectBound checks Attach answered bound on its Node, or, when bound is
// zero, a new Torrent in a's intent.
func expectBound(t *testing.T, lib library, id dc.ID, node role.NodeID, bound dc.ID, a role.NodeID) {
	t.Helper()
	if bound.IsZero() {
		if node != a || !slices.Contains(intentOf(t, lib, a), id) {
			t.Fatalf("Attach bound %s on %s, want a new Torrent in home's intent", id, node)
		}
		return
	}
	var holder role.NodeID
	if err := lib.QueryRowContext(t.Context(), "SELECT node_id FROM downloadclients_torrent WHERE id = ?", bound).Scan(&holder); err != nil {
		t.Fatal(err)
	}
	if id != bound || node != holder {
		t.Fatalf("Attach bound %s on %s, want the held Torrent %s on %s", id, node, bound, holder)
	}
}

func TestANewTorrentStartsFromTheLatestRemovedTotals(t *testing.T) {
	lib := open(t)
	a, b := addNode(t, lib, "home"), addNode(t, lib, "seedbox")
	m, raw := dotTorrent(t, "Orchard")
	want := dc.Want{Node: b, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff}
	t0 := time.Date(2026, 10, 19, 20, 0, 0, 0, time.UTC)
	older, _ := mustAttach(t, lib, want)
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return dc.MergeTotals(ctx, tx, b, older, dc.Totals{Uploaded: 100, Downloaded: 1024, Active: time.Hour, At: t0})
	})
	removeTorrent(t, lib, older, dc.ByAdmin)
	partTorrent(t, lib, older)
	newer, _ := mustAttach(t, lib, want)
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return dc.MergeTotals(ctx, tx, b, newer, dc.Totals{Uploaded: 5000, Downloaded: 1024, Active: 2 * time.Hour, Seeding: time.Hour, At: t0.Add(time.Hour)})
	})
	removeTorrent(t, lib, newer, dc.ByAdmin)

	want.Node = a
	id, _ := mustAttach(t, lib, want)
	got := totalsOf(t, lib, id)
	if got.Uploaded != 5000 || got.Downloaded != 1024 || got.Active != 2*time.Hour || got.Seeding != time.Hour || !got.At.Equal(t0.Add(time.Hour)) {
		t.Errorf("the new Torrent's totals are %+v, want the latest removed Torrent's 5000 up, 1024 down, 2h active, 1h seeding", got)
	}
}

func outcomeOf(t *testing.T, lib library, id dc.ID) dc.Attached {
	t.Helper()
	var got map[dc.ID]dc.Attached
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = dc.Outcomes(ctx, r, []dc.ID{id})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return got[id]
}

func apply(t *testing.T, lib library, node role.NodeID, id dc.ID, r *dlv1.TorrentReport) {
	t.Helper()
	r.Id = id.Bytes()
	rep := &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{r}}}}
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return clients().Apply(ctx, tx, node, rep)
	})
}

func clients() *dc.Clients {
	return dc.New(dc.Config{Touch: func(...role.NodeID) {}, Changed: published{}})
}

type published struct{}

func (published) Publish(context.Context, dc.Changed) {}

var reportedAt = time.Date(2026, 10, 19, 21, 0, 0, 0, time.UTC)

func completed(files ...engine.File) *dlv1.TorrentReport {
	c := &dlv1.Completion{}
	for _, f := range files {
		c.Files = append(c.Files, &dlv1.File{Index: int32(f.Index), Path: f.Path, Size: f.Size}) //nolint:gosec // a test index fits
	}
	return &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_SEEDING, Complete: true, Completion: c}
}

func failed() *dlv1.TorrentReport {
	return &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_FAILED, Error: "disk on fire"}
}

// TestOutcomesFollowTheTorrentsState is what a Grab attached to a
// Torrent reads in each state: Gone over Failed over Completed, with the
// first completion's files kept through a removal.
func TestOutcomesFollowTheTorrentsState(t *testing.T) {
	feature := engine.File{Index: 0, Path: "Orchard/Orchard.mkv", Size: 1000}
	for _, c := range []struct {
		name  string
		to    func(t *testing.T, lib library, node role.NodeID, id dc.ID)
		want  dc.Outcome
		files []engine.File
	}{
		{"added, no report", func(*testing.T, library, role.NodeID, dc.ID) {}, dc.Pending{}, nil},
		{"complete", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, completed(feature))
		}, dc.Completed{}, []engine.File{feature}},
		{"a second completion keeps the first's files", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, completed(feature))
			apply(t, lib, node, id, completed(engine.File{Index: 3, Path: "other.mkv", Size: 1}))
		}, dc.Completed{}, []engine.File{feature}},
		{"failed", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, failed())
		}, dc.Failed{Error: "disk on fire"}, nil},
		{"failed, then complete", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, failed())
			apply(t, lib, node, id, completed(feature))
		}, dc.Completed{}, []engine.File{feature}},
		{"complete, then failed", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, completed(feature))
			apply(t, lib, node, id, failed())
		}, dc.Failed{Error: "disk on fire"}, []engine.File{feature}},
		{"a result from another Node", func(t *testing.T, lib library, _ role.NodeID, id dc.ID) {
			apply(t, lib, addNode(t, lib, "elsewhere"), id, completed(feature))
		}, dc.Pending{}, nil},
		{"removed by an Admin once complete", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, completed(feature))
			removeTorrent(t, lib, id, dc.ByAdmin)
		}, dc.Gone{Reason: dc.ByAdmin}, []engine.File{feature}},
		{"marked failed", func(t *testing.T, lib library, _ role.NodeID, id dc.ID) {
			removeTorrent(t, lib, id, dc.ByMarkFailed)
		}, dc.Gone{Reason: dc.ByMarkFailed}, nil},
		{"failed and removed", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, failed())
			removeTorrent(t, lib, id, dc.ByAdmin)
			partTorrent(t, lib, id)
		}, dc.Gone{Reason: dc.ByAdmin}, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			lib := open(t)
			node := addNode(t, lib, "seedbox")
			m, raw := dotTorrent(t, "Orchard")
			id, _ := mustAttach(t, lib, dc.Want{Node: node, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
			c.to(t, lib, node, id)
			got := outcomeOf(t, lib, id)
			if got.Outcome != c.want || got.InfoHash != m.InfoHash || !slices.Equal(got.Files, c.files) {
				t.Errorf("Outcomes = %+v, want %#v with files %v and hash %s", got, c.want, c.files, m.InfoHash)
			}
		})
	}
}

func TestAWriteOffPartsEveryTorrentOfTheNodeAndKeepsFiles(t *testing.T) {
	lib := open(t)
	box, home := addNode(t, lib, "seedbox"), addNode(t, lib, "home")
	attachNamed := func(node role.NodeID, name string) dc.ID {
		m, raw := dotTorrent(t, name)
		id, _ := mustAttach(t, lib, dc.Want{Node: node, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
		return id
	}
	live, removing, kept := attachNamed(box, "Orchard"), attachNamed(box, "Attic"), attachNamed(home, "Harbor")
	exec(t, lib, "UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = 'admin', delete_files = 1 WHERE id = ?", storage.Now(), removing)

	writeOff := func() int64 {
		var n int64
		write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			var err error
			n, err = dc.WriteOff(ctx, tx, box, storage.Now())
			return err
		})
		return n
	}
	if n := writeOff(); n != 2 {
		t.Fatalf("WriteOff parted %d Torrents, want seedbox's 2", n)
	}
	for _, id := range []dc.ID{live, removing} {
		if got := outcomeOf(t, lib, id).Outcome; got != (dc.Gone{Reason: dc.NodeWrittenOff}) {
			t.Errorf("a written-off Torrent reads %#v, want Gone for node_written_off", got)
		}
		var parted int64
		if err := lib.QueryRowContext(t.Context(), "SELECT count(*) FROM downloadclients_torrent WHERE id = ? AND parted_at IS NOT NULL", id).Scan(&parted); err != nil || parted != 1 {
			t.Errorf("a written-off Torrent waits on its Node (%v), want it parted at once", err)
		}
	}
	if got := intentOf(t, lib, box); len(got) != 0 {
		t.Errorf("seedbox's intent lists %v after its write-off, want nothing", got)
	}
	if got := intentOf(t, lib, home); !slices.Equal(got, []dc.ID{kept}) {
		t.Errorf("home's intent lists %v, want its own Torrent %s kept", got, kept)
	}
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		held, err := dc.DownloadingOwner.Held(ctx, r, box)
		if err == nil && held.Units != 0 {
			t.Errorf("seedbox holds %d torrents after its write-off, want none", held.Units)
		}
		holding, err2 := dc.Holding(ctx, r)
		if err2 == nil && !slices.Equal(holding, []role.NodeID{home}) {
			t.Errorf("the Nodes holding Torrents are %v, want only home", holding)
		}
		return errors.Join(err, err2)
	})
	if err != nil {
		t.Fatal(err)
	}
	if n := writeOff(); n != 0 {
		t.Errorf("a second WriteOff parted %d Torrents, want none", n)
	}
}

package downloadclients_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// moving answers that each source it names moves, as moves would.
type moving map[dc.ID]dc.MovingTo

func (m moving) Moving(_ context.Context, _ storage.Reader[storage.Library], ids []dc.ID) (map[dc.ID]dc.MovingTo, error) {
	out := map[dc.ID]dc.MovingTo{}
	for _, id := range ids {
		if to, ok := m[id]; ok {
			out[id] = to
		}
	}
	return out, nil
}

func (moving) MovesOf(context.Context, storage.Reader[storage.Library], []dc.ID) (map[dc.ID]dc.Move, error) {
	return nil, nil
}

func (moving) UsesClient(context.Context, storage.Reader[storage.Library], dc.ClientID) (int64, error) {
	return 0, nil
}

// move is the Orchard on seedbox mid-Move to den's built-in client, its
// arriving Torrent inserted.
type move struct {
	shelf
	den      role.NodeID
	arriving dc.ID
	c        *dc.Clients
}

func newMove(t *testing.T) move {
	t.Helper()
	m := move{shelf: newShelf(t)}
	m.den = addNode(t, m.lib, "den")
	write(t, m.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		m.arriving, err = dc.InsertArriving(ctx, tx, dc.Arriving{Unit: [16]byte(storage.NewUUID().Bytes()), From: m.orchard, Node: m.den})
		return err
	})
	m.c = dc.New(dc.Config{
		Library: m.lib, Touch: func(...role.NodeID) {}, Changed: &heard{}, Roster: &roster{}, Blocklist: &heard{}, Seeding: seeder{},
		Moving: moving{m.orchard: {Node: m.den}}, Uses: uses{}, Log: slog.New(slog.DiscardHandler),
		Remote: func(role.NodeID) downloadingv1connect.DownloadingServiceHandler { return &client{} },
		Live:   dc.LiveConfig{Watched: func() []role.NodeID { return nil }, Event: &admin{}, Sample: liveSample},
	})
	return m
}

// TestEveryAdminVerbRefusesAMovingSourceAndItsArrivingTorrent runs each
// action on the Orchard mid-Move and on its arriving Torrent, beside the
// Attic: both are refused moving, naming den, and stay as they were, and
// the Attic is acted on.
func TestEveryAdminVerbRefusesAMovingSourceAndItsArrivingTorrent(t *testing.T) {
	m := newMove(t)
	for _, kind := range []dc.ActionKind{dc.Pause, dc.Resume, dc.Announce, dc.Recheck, dc.Retry, dc.MarkFailed, dc.Remove} {
		results, _ := act(t, m.c, dc.Action{Kind: kind, IDs: []dc.ID{m.orchard, m.arriving, m.attic}})
		got := refusals(results)
		if got[0] == nil || got[0].Code != dc.RefusedMoving || got[0].Node != "den" || got[0].Client != "" {
			t.Errorf("%s on the moving source answered %+v, want refused moving to den", kind, got[0])
		}
		if got[1] == nil || got[1].Code != dc.RefusedMoving {
			t.Errorf("%s on the arriving Torrent answered %+v, want refused moving", kind, got[1])
		}
		if results[2].ID != m.attic || got[2] != nil && got[2].Code == dc.RefusedMoving {
			t.Errorf("%s on the Attic answered %+v, want it judged as without a Move", kind, results[2])
		}
		if wantedOf(t, m.lib, m.orchard) != "started" || wantedOf(t, m.lib, m.arriving) != "paused" {
			t.Fatalf("after %s the source is %s and its destination %s, want started and paused", kind, wantedOf(t, m.lib, m.orchard), wantedOf(t, m.lib, m.arriving))
		}
	}
}

// TestAGrabMidMoveAttachesToTheSource grabs the Orchard for den while it
// moves there: the Grab takes the seeding source, never the arriving
// Torrent, and den's intent holds nothing of the built-in arriving one.
func TestAGrabMidMoveAttachesToTheSource(t *testing.T) {
	m := newMove(t)
	meta, raw := dotTorrent(t, "Orchard")
	id, node := mustAttach(t, m.lib, dc.Want{Target: dc.Target{Node: m.den}, Metainfo: meta, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	if id != m.orchard || node != m.box {
		t.Fatalf("a Grab mid-Move attached to %s on %s, want the source on seedbox", id, node)
	}
	if got := intentWithDiscards(t, m.lib, m.den); len(got.GetTorrents()) != 0 {
		t.Errorf("den's intent holds %v before any start is asked, want nothing", got.GetTorrents())
	}
}

// TestTheHandoffReadsReportsInControllerTime has seedbox, its clock ten
// minutes ahead, report the watched Orchard's announces and its pause, and
// den echo a start ask, then a stale one, and seedbox forge den's arrival:
// every time stored is the Controller's, and only den's echo of the open
// ask counts.
func TestTheHandoffReadsReportsInControllerTime(t *testing.T) {
	m := newMove(t)
	write(t, m.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return m.c.WatchIn(ctx, tx, m.orchard, true)
	})
	before := storage.Now().Time()
	ahead := before.Add(10 * time.Minute)
	report := func(node role.NodeID, r *dlv1.TorrentReport) {
		t.Helper()
		write(t, m.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return m.c.Apply(ctx, tx, node, &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{r}}}})
		})
	}
	report(m.box, &dlv1.TorrentReport{Id: m.orchard.Bytes(), AtUnixMs: ahead.UnixMilli(), Trackers: []*dlv1.TrackerLine{{
		Status: dlv1.TrackerStatus_TRACKER_STATUS_WORKING, AnsweredAtUnixMs: ahead.Add(-time.Minute).UnixMilli(),
		NextAnnounceUnixMs: ahead.Add(20 * time.Minute).UnixMilli(),
	}}})
	h := handoff(t, m)
	if h.AnnouncedAt == nil || h.AnnouncedAt.After(before.Add(-time.Minute+time.Second)) || h.AnnouncedAt.Before(before.Add(-time.Minute-5*time.Second)) {
		t.Fatalf("the announce a minute before the report reads %v, want about a minute before %v on the Controller's clock", h.AnnouncedAt, before)
	}
	if h.AnnounceDueAt == nil || h.AnnounceDueAt.Sub(*h.AnnouncedAt).Round(time.Second) != 21*time.Minute {
		t.Errorf("the next announce reads %v, want 21 minutes after the last", h.AnnounceDueAt)
	}

	write(t, m.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, _, err := m.c.WantIn(ctx, tx, []dc.ID{m.orchard}, dc.Paused)
		return err
	})
	report(m.box, &dlv1.TorrentReport{Id: m.orchard.Bytes(), AtUnixMs: ahead.UnixMilli(), Hold: dlv1.Hold_HOLD_WANTED, Trackers: []*dlv1.TrackerLine{{
		Status: dlv1.TrackerStatus_TRACKER_STATUS_WORKING, AnsweredAtUnixMs: ahead.Add(-2 * time.Minute).UnixMilli(), Stopped: true,
	}}})
	h = handoff(t, m)
	if h.PausedAt == nil || h.PausedAt.Before(before) || h.PausedAt.After(storage.Now().Time()) {
		t.Fatalf("the pause reads %v, want the Controller's time of the report", h.PausedAt)
	}
	if h.AnnouncedAt.Before(before.Add(-time.Minute - 5*time.Second)) {
		t.Errorf("an older answer moved the last announce back to %v", h.AnnouncedAt)
	}

	asked := storage.Now().Time()
	write(t, m.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return m.c.StartIn(ctx, tx, m.arriving, m.orchard, asked, asked.Add(90*time.Second))
	})
	arrival := func(node role.NodeID, askedMs int64, result dlv1.ArrivalResult) {
		report(node, &dlv1.TorrentReport{Id: m.arriving.Bytes(), AtUnixMs: ahead.UnixMilli(), Arrival: &dlv1.Arrival{AskedUnixMs: askedMs, Result: result, Reason: "no_answer"}})
	}
	arrival(m.box, asked.UnixMilli(), dlv1.ArrivalResult_ARRIVAL_RESULT_STARTED)
	arrival(m.den, asked.Add(-time.Minute).UnixMilli(), dlv1.ArrivalResult_ARRIVAL_RESULT_STARTED)
	if h = handoff(t, m); h.Arrival != nil {
		t.Fatalf("a forged arrival from seedbox and a stale echo from den stored %s", *h.Arrival)
	}
	if h.StartAskedAt == nil || h.Final.Uploaded != 0 {
		t.Fatalf("the ask reads %+v, want it open", h)
	}
	arrival(m.den, asked.UnixMilli(), dlv1.ArrivalResult_ARRIVAL_RESULT_REFUSED)
	if h = handoff(t, m); h.Arrival == nil || *h.Arrival != dc.ArrivalRefused || h.ArrivalWhy != "no_answer" {
		t.Fatalf("den's refusal of the open ask reads %v %q, want refused no_answer", h.Arrival, h.ArrivalWhy)
	}
	if got := intentWithDiscards(t, m.lib, m.den).GetTorrents(); len(got) != 1 || !got[0].GetArriving() || got[0].GetStartAskedUnixMs() != asked.UnixMilli() ||
		got[0].GetStartWindowMs() <= 0 || got[0].GetStartWindowMs() > 90_000 {
		t.Errorf("den's intent holds %v, want the arriving Torrent with its ask and what is left of its window", got)
	}

	write(t, m.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return m.c.RollbackIn(ctx, tx, m.orchard, m.arriving)
	})
	if h = handoff(t, m); h.Arrival != nil || h.StartAskedAt != nil || h.PausedAt != nil || h.Wanted != dc.Started {
		t.Errorf("after the rollback the handoff reads %+v, want the source started and the ask gone", h)
	}
	if wantedOf(t, m.lib, m.arriving) != "paused" {
		t.Errorf("after the rollback the destination is %s, want paused", wantedOf(t, m.lib, m.arriving))
	}
}

func handoff(t *testing.T, m move) dc.Handoff {
	t.Helper()
	var got map[dc.ID]dc.Handoff
	err := m.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = dc.Handoffs(ctx, r, []dc.Pair{{From: m.orchard, To: &m.arriving}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return got[m.orchard]
}

// TestTheSwitchHandsTheHashToTheArrivingTorrent switches the Orchard to
// den: seedbox discards the source with its files, den's Torrent is live
// with the source's totals, and a Grab for the hash takes it.
func TestTheSwitchHandsTheHashToTheArrivingTorrent(t *testing.T) {
	m := newMove(t)
	apply(t, m.lib, m.box, m.orchard, &dlv1.TorrentReport{Totals: &dlv1.TotalsRow{Id: m.orchard.Bytes(), Uploaded: 4096, SeedingSeconds: 7200}})
	write(t, m.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return m.c.SwitchIn(ctx, tx, m.orchard, m.arriving)
	})
	discards := intentWithDiscards(t, m.lib, m.box).GetDiscards()
	if !slices.ContainsFunc(discards, func(d *dlv1.Discard) bool { return idFrom(d.GetId()) == m.orchard && d.GetDeleteFiles() }) {
		t.Fatalf("seedbox's intent discards %v, want the source with its files", discards)
	}
	var up, seeding int64
	if err := m.lib.QueryRowContext(t.Context(), "SELECT uploaded, seeding_seconds FROM downloadclients_torrent WHERE id = ?", m.arriving).Scan(&up, &seeding); err != nil {
		t.Fatal(err)
	}
	if up != 4096 || seeding != 7200 {
		t.Errorf("den's Torrent has %d uploaded and %d s seeding, want the source's 4096 and 7200", up, seeding)
	}
	meta, raw := dotTorrent(t, "Orchard")
	if id, _ := mustAttach(t, m.lib, dc.Want{Target: dc.Target{Node: m.den}, Metainfo: meta, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff}); id != m.arriving {
		t.Errorf("a Grab after the switch attached to %s, want den's Torrent", id)
	}
}

// TestALateReportLeavesASwitchedSourcesTotals switches the Orchard to den
// once seedbox counted 2 hours of seeding, then seedbox's last report and
// totals row for it land, twice each, with 2 seconds more: the switch
// carried the source's final totals, so both Torrents keep them.
func TestALateReportLeavesASwitchedSourcesTotals(t *testing.T) {
	m := newMove(t)
	apply(t, m.lib, m.box, m.orchard, &dlv1.TorrentReport{Totals: &dlv1.TotalsRow{Id: m.orchard.Bytes(), Uploaded: 4096, SeedingSeconds: 7200}})
	write(t, m.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return m.c.SwitchIn(ctx, tx, m.orchard, m.arriving)
	})
	late := &dlv1.TotalsRow{Id: m.orchard.Bytes(), Uploaded: 4200, SeedingSeconds: 7202}
	for range 2 {
		apply(t, m.lib, m.box, m.orchard, &dlv1.TorrentReport{Totals: late})
		write(t, m.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return m.c.Apply(ctx, tx, m.box, &dlv1.Report{Kind: &dlv1.Report_Totals{Totals: &dlv1.Totals{Rows: []*dlv1.TotalsRow{late}}}})
		})
	}
	for _, id := range []dc.ID{m.orchard, m.arriving} {
		if got := totalsOf(t, m.lib, id); got.Uploaded != 4096 || got.Seeding != 2*time.Hour {
			t.Errorf("Torrent %s has %d uploaded and %s seeding after the late reports, want the switch's 4096 and 2h0m0s", id, got.Uploaded, got.Seeding)
		}
	}
}

// TestTheMovesRebuildKeepsEveryTorrentAndHoldsAHashTwice migrates an
// adopted withheld Torrent through 00011's rebuild: it keeps its columns,
// may be removed moved, and a Move's arriving Torrent of the same hash on
// the same Node is no second holder.
func TestTheMovesRebuildKeepsEveryTorrentAndHoldsAHashTwice(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), storage.LibraryFile)
	openWith := func(clients storage.Schema[storage.Library]) library {
		t.Helper()
		lib, err := storage.OpenLibrary(ctx, path, []storage.Schema[storage.Library]{
			secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), health.Schema(), roots.Schema(), clients, indexers.Schema(),
		}, nil, storage.DefaultTimings(), slog.New(slog.DiscardHandler))
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	old := openWith(dc.Schema().Through(10))
	node, torrent := addNode(t, old, "Den"), storage.NewUUID()
	write(t, old, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers,
			adopted, withheld_at, wanted, added_order, added_at, uploaded, client_error_ms)
			VALUES (?, ?, randomblob(20), x'00', 'Heat.1995.1080p.BluRay', 1024, 1, '[]', 1, 5, 'started', 1, 0, 512, 300)`, torrent, node)
		return err
	})
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	lib := openWith(dc.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	var got string
	err := lib.QueryRowContext(ctx, `SELECT name || ' ' || uploaded || ' ' || adopted || ' ' || withheld_at || ' ' || client_error_ms || ' ' ||
		watch_announce || ' ' || coalesce(arriving_unit, 'none') FROM downloadclients_torrent WHERE id = ?`, torrent).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if want := "Heat.1995.1080p.BluRay 512 1 5 300 0 none"; got != want {
		t.Fatalf("after the migration the Torrent reads %q, want %q", got, want)
	}
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO downloadclients_torrent (id, node_id, info_hash, metainfo, name, size, private, trackers,
			adopted, wanted, added_order, added_at, arriving_unit)
			SELECT ?, node_id, info_hash, metainfo, name, size, private, trackers, 1, 'paused', 2, 0, randomblob(16)
			FROM downloadclients_torrent WHERE id = ?`, storage.NewUUID(), torrent)
		return err
	})
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE downloadclients_torrent SET removed_at = 9, removed_reason = 'moved' WHERE id = ?", torrent)
		return err
	})
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE downloadclients_torrent SET arrival = 'started' WHERE arriving_unit IS NOT NULL")
		return err
	})
	if err == nil {
		t.Fatal("a started arrival was stored with no ask")
	}
}

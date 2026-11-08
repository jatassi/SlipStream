package downloadclients_test

import (
	"context"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

func addTorrent(t *testing.T, lib library, node role.NodeID) dc.ID {
	t.Helper()
	id := dc.ID{UUID: storage.NewUUID()}
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO downloadclients_torrent
			(id, node_id, info_hash, metainfo, name, size, private, trackers, content_type, wanted, added_order, added_at)
			VALUES (?, ?, randomblob(20), x'00', 'Release', 1000, 1, '[]', 'movie', 'started', 1, 0)`, id, node)
		return err
	})
	return id
}

func totalsOf(t *testing.T, lib library, id dc.ID) dc.Totals {
	t.Helper()
	var tot dc.Totals
	var active, seeding int64
	var at storage.Time
	err := lib.QueryRowContext(t.Context(), "SELECT uploaded, downloaded, active_seconds, seeding_seconds, totals_at FROM downloadclients_torrent WHERE id = ?", id).
		Scan(&tot.Uploaded, &tot.Downloaded, &active, &seeding, &at)
	if err != nil {
		t.Fatal(err)
	}
	tot.Active, tot.Seeding, tot.At = time.Duration(active)*time.Second, time.Duration(seeding)*time.Second, at.Time()
	return tot
}

func TestTotalsNeverGoDown(t *testing.T) {
	lib := open(t)
	seedbox, home := addNode(t, lib, "seedbox"), addNode(t, lib, "home")
	id := addTorrent(t, lib, seedbox)
	t0 := time.Date(2026, 10, 19, 20, 0, 0, 0, time.UTC)
	merge := func(node role.NodeID, tot dc.Totals) {
		write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return dc.MergeTotals(ctx, tx, node, id, tot)
		})
	}

	merge(seedbox, dc.Totals{Uploaded: 900, Downloaded: 1000, Active: time.Hour, Seeding: 30 * time.Minute, At: t0})
	merge(seedbox, dc.Totals{Uploaded: 100, Downloaded: 1200, Active: time.Minute, Seeding: time.Hour, At: t0.Add(time.Minute)})
	want := dc.Totals{Uploaded: 900, Downloaded: 1200, Active: time.Hour, Seeding: time.Hour, At: t0.Add(time.Minute)}
	if got := totalsOf(t, lib, id); !got.At.Equal(want.At) || got.Uploaded != want.Uploaded || got.Downloaded != want.Downloaded || got.Active != want.Active || got.Seeding != want.Seeding {
		t.Errorf("totals after a rebuilt Node's lower report = %+v, want %+v", got, want)
	}

	merge(home, dc.Totals{Uploaded: 5000, At: t0.Add(time.Hour)})
	if got := totalsOf(t, lib, id); got.Uploaded != 900 {
		t.Errorf("uploaded after another Node reported the id = %d, want 900 kept", got.Uploaded)
	}
}

func TestRatio(t *testing.T) {
	for _, c := range []struct {
		name string
		tot  dc.Totals
		size int64
		want float64
	}{
		{"downloaded here", dc.Totals{Uploaded: 3000, Downloaded: 2000}, 1000, 1.5},
		{"came complete from elsewhere", dc.Totals{Uploaded: 500}, 1000, 0.5},
		{"a size of nothing", dc.Totals{}, 0, 0},
	} {
		if got := dc.Ratio(c.tot, c.size); got != c.want {
			t.Errorf("%s: Ratio = %v, want %v", c.name, got, c.want)
		}
	}
}

func clientTotals(rows ...*dlv1.ClientCounters) *dlv1.Report {
	return &dlv1.Report{Kind: &dlv1.Report_ClientTotals{ClientTotals: &dlv1.ClientTotals{Rows: rows}}}
}

func TestAnExternalClientsCountersAddOnlyTheirRiseWhileTracked(t *testing.T) {
	s := newMediaServer(t)
	id := s.send(t, "Orchard")
	counters := func(up, down, active, seeding int64) *dlv1.Report {
		return clientTotals(&dlv1.ClientCounters{Id: id.Bytes(), Uploaded: up, Downloaded: down, ActiveSeconds: active, SeedingSeconds: seeding})
	}
	expect := func(when string, up, down int64, active, seeding time.Duration) {
		t.Helper()
		got := totalsOf(t, s.lib, id)
		if got.Uploaded != up || got.Downloaded != down || got.Active != active || got.Seeding != seeding {
			t.Errorf("%s: totals = %d up, %d down, %s active, %s seeding; want %d, %d, %s, %s",
				when, got.Uploaded, got.Downloaded, got.Active, got.Seeding, up, down, active, seeding)
		}
	}

	s.report(t, s.node, counters(9000, 4000, 7200, 3600))
	expect("the first report, which counts what came before SlipStream tracked it", 0, 0, 0, 0)

	s.report(t, s.node, counters(9500, 4600, 7260, 3660))
	expect("a rise", 500, 600, time.Minute, time.Minute)

	s.report(t, s.node, counters(9500, 4600, 7260, 3660))
	expect("a replay", 500, 600, time.Minute, time.Minute)

	s.report(t, s.node, counters(100, 4700, 7320, 3720))
	expect("an uploaded counter that dropped as the torrent was added again", 500, 700, 2*time.Minute, 2*time.Minute)

	s.report(t, s.node, counters(400, 4700, 7320, 3720))
	expect("a rise from the new baseline", 800, 700, 2*time.Minute, 2*time.Minute)

	s.report(t, addNode(t, s.lib, "Seedbox"), counters(90000, 90000, 90000, 90000))
	expect("another Node's report", 800, 700, 2*time.Minute, 2*time.Minute)

	m, raw := dotTorrent(t, "Harbor")
	builtIn, _ := mustAttach(t, s.lib, dc.Want{Target: dc.Target{Node: s.node}, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	s.report(t, s.node, clientTotals(&dlv1.ClientCounters{Id: builtIn.Bytes(), Uploaded: 10, Downloaded: 10}))
	s.report(t, s.node, clientTotals(&dlv1.ClientCounters{Id: builtIn.Bytes(), Uploaded: 900, Downloaded: 900}))
	if got := totalsOf(t, s.lib, builtIn); got.Uploaded != 0 || got.Downloaded != 0 {
		t.Errorf("raw counters for a built-in Torrent left its totals %d up, %d down, want them unchanged", got.Uploaded, got.Downloaded)
	}
}

// TestTheAnnouncedSecondsANodeReportsFillItsTorrent reports a built-in
// Torrent's announced seconds in a state report and in totals batches: the
// row reads the most reported, a lower or another Node's report lowers
// nothing, the Node fetching it again starts from it, and a Torrent of the
// same hash added once it is removed inherits it.
func TestTheAnnouncedSecondsANodeReportsFillItsTorrent(t *testing.T) {
	s := newMediaServer(t)
	m, raw := dotTorrent(t, "Harbor")
	want := dc.Want{Target: dc.Target{Node: s.node}, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff}
	id, _ := mustAttach(t, s.lib, want)
	batch := func(seconds int64) *dlv1.Report {
		return &dlv1.Report{Kind: &dlv1.Report_Totals{Totals: &dlv1.Totals{Rows: []*dlv1.TotalsRow{{Id: id.Bytes(), AnnouncedSeconds: seconds}}}}}
	}
	expect := func(when string, want time.Duration) {
		t.Helper()
		if got := s.row(t, id).Totals.Announced; got != want {
			t.Errorf("%s the row reads %v announced, want %v", when, got, want)
		}
	}

	s.report(t, s.node, s.torrent(t, &dlv1.TorrentReport{Id: id.Bytes(), Engine: dlv1.EngineState_ENGINE_STATE_SEEDING, Complete: true,
		Totals: &dlv1.TotalsRow{Id: id.Bytes(), AnnouncedSeconds: 1800}}))
	expect("after a state report", 30*time.Minute)
	s.report(t, s.node, batch(5400))
	expect("after a totals batch", 90*time.Minute)
	s.report(t, s.node, batch(3600))
	s.report(t, addNode(t, s.lib, "Seedbox"), batch(90000))
	expect("after a lower report and another Node's", 90*time.Minute)

	svc := dc.ControlService{Library: s.lib, Nodes: caller{s.node}, Types: types{}}
	res, err := svc.Torrent(t.Context(), &dlv1.TorrentRequest{Id: id.Bytes()})
	if err != nil || res.GetBaseline().GetAnnouncedSeconds() != 5400 {
		t.Errorf("a Node that lost its local row fetches a baseline of %d announced seconds (%v), want 5400", res.GetBaseline().GetAnnouncedSeconds(), err)
	}

	removeTorrent(t, s.lib, id, dc.ByAdmin)
	partTorrent(t, s.lib, id)
	again, _ := mustAttach(t, s.lib, want)
	if got := s.row(t, again).Totals.Announced; got != 90*time.Minute {
		t.Errorf("a new Torrent of the removed one's hash starts from %v announced, want its 1h30m", got)
	}
}

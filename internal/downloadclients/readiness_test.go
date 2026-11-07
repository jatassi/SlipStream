package downloadclients_test

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

func readinessOf(t *testing.T, lib library, c *dc.Clients, node role.NodeID, ref source.Ref) source.Readiness {
	t.Helper()
	var got map[source.Ref]source.Readiness
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = c.ReadinessIn(ctx, r, node, []source.Ref{ref})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return got[ref]
}

func reported(engine dlv1.EngineState, complete bool) *dlv1.TorrentReport {
	return &dlv1.TorrentReport{AtUnixMs: reportedAt.UnixMilli(), Engine: engine, Complete: complete}
}

var (
	checking    = reported(dlv1.EngineState_ENGINE_STATE_CHECKING, false)
	downloading = reported(dlv1.EngineState_ENGINE_STATE_DOWNLOADING, false)
	seeding     = reported(dlv1.EngineState_ENGINE_STATE_SEEDING, true)
)

// TestReadinessFollowsTheTorrentsReportAndRemoval is a Transfer's
// source as the Controller reads it from the Torrent of its hash on its
// Node.
func TestReadinessFollowsTheTorrentsReportAndRemoval(t *testing.T) {
	for _, c := range []struct {
		name string
		to   func(t *testing.T, lib library, node role.NodeID, id dc.ID)
		want source.Readiness
	}{
		{"added, no report", func(*testing.T, library, role.NodeID, dc.ID) {}, source.Readiness{State: source.Unknown, Name: "Orchard"}},
		{"downloading", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, downloading)
		}, source.Readiness{State: source.Incomplete, Name: "Orchard"}},
		{"complete", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, seeding)
		}, source.Readiness{State: source.Ready, Name: "Orchard"}},
		{"checked once complete", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, seeding)
			apply(t, lib, node, id, checking)
		}, source.Readiness{State: source.Checking, Name: "Orchard"}},
		{"found incomplete by a check", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, seeding)
			apply(t, lib, node, id, checking)
			apply(t, lib, node, id, downloading)
		}, source.Readiness{State: source.Incomplete, Name: "Orchard"}},
		{"failed once complete", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, seeding)
			apply(t, lib, node, id, failed())
		}, source.Readiness{State: source.FailedSource, Name: "Orchard", Detail: "disk on fire"}},
		{"written off", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, seeding)
			removeTorrent(t, lib, id, dc.NodeWrittenOff)
		}, source.Readiness{State: source.WrittenOff, Name: "Orchard"}},
		{"removed by an Admin", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			apply(t, lib, node, id, seeding)
			removeTorrent(t, lib, id, dc.ByAdmin)
		}, source.Readiness{State: source.Unknown, Name: "Orchard"}},
		{"attached again after a write-off", func(t *testing.T, lib library, node role.NodeID, id dc.ID) {
			removeTorrent(t, lib, id, dc.NodeWrittenOff)
			partTorrent(t, lib, id)
			m, raw := dotTorrent(t, "Orchard")
			again, _ := mustAttach(t, lib, dc.Want{Node: node, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
			apply(t, lib, node, again, seeding)
		}, source.Readiness{State: source.Ready, Name: "Orchard"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			lib := open(t)
			node := addNode(t, lib, "seedbox")
			m, raw := dotTorrent(t, "Orchard")
			id, _ := mustAttach(t, lib, dc.Want{Node: node, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
			c.to(t, lib, node, id)
			if got := readinessOf(t, lib, clients(), node, source.Torrent{Hash: m.InfoHash, File: 0}); got != c.want {
				t.Errorf("ReadinessIn = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestReadinessOfAHashTheNodeNeverHeldIsUnknown(t *testing.T) {
	lib := open(t)
	box, home := addNode(t, lib, "seedbox"), addNode(t, lib, "home")
	m, raw := dotTorrent(t, "Orchard")
	id, _ := mustAttach(t, lib, dc.Want{Node: home, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	apply(t, lib, home, id, seeding)
	if got := readinessOf(t, lib, clients(), box, source.Torrent{Hash: m.InfoHash}); got != (source.Readiness{}) {
		t.Errorf("ReadinessIn on seedbox of a hash only home holds = %+v, want Unknown", got)
	}
}

// TestMovedClosesWhenAReportChangesReadiness applies the reports a
// check that ends and a download that completes send, each through
// Applied: Moved closes for each, so a reader of ReadinessIn reads again.
func TestMovedClosesWhenAReportChangesReadiness(t *testing.T) {
	lib := open(t)
	node := addNode(t, lib, "seedbox")
	m, raw := dotTorrent(t, "Orchard")
	id, _ := mustAttach(t, lib, dc.Want{Node: node, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	c := clients()
	ref := source.Torrent{Hash: m.InfoHash}
	for _, step := range []struct {
		name   string
		report *dlv1.TorrentReport
		want   source.ReadyState
	}{
		{"the check starts", checking, source.Checking},
		{"the check ends", downloading, source.Incomplete},
		{"the download completes", seeding, source.Ready},
	} {
		moved := c.Moved()
		step.report.Id = id.Bytes()
		write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return c.Apply(ctx, tx, node, &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{step.report}}}})
		})
		c.Applied(node, string(role.Downloading))
		select {
		case <-moved:
		default:
			t.Fatalf("Moved is open once %s, want it closed", step.name)
		}
		if got := readinessOf(t, lib, c, node, ref).State; got != step.want {
			t.Fatalf("readiness once %s is %v, want %v", step.name, got, step.want)
		}
	}
}

// TestAWriteOffMakesTheSourceWrittenOffAndMoves runs the real write-off
// pass: the source reads WrittenOff from the durable removal, after the
// roster's transient write-off is gone, and Moved closes.
func TestAWriteOffMakesTheSourceWrittenOffAndMoves(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		lib := open(t)
		box := addNode(t, lib, "seedbox")
		m, raw := dotTorrent(t, "Orchard")
		id, _ := mustAttach(t, lib, dc.Want{Node: box, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
		apply(t, lib, box, id, seeding)
		r := &roster{}
		c, _ := newClients(lib, r, nil, nil)
		ctx, stop := context.WithCancel(context.WithoutCancel(t.Context()))
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = c.Run(ctx)
		}()
		t.Cleanup(func() {
			stop()
			<-done
		})
		synctest.Wait()
		moved := c.Moved()

		r.writeOff(box)
		synctest.Wait()
		r.change(func() { delete(r.writtenOff, box) })
		synctest.Wait()
		select {
		case <-moved:
		default:
			t.Fatal("Moved is open after the write-off, want it closed")
		}
		if got := readinessOf(t, lib, c, box, source.Torrent{Hash: m.InfoHash}); got != (source.Readiness{State: source.WrittenOff, Name: "Orchard"}) {
			t.Errorf("ReadinessIn after the write-off = %+v, want written off", got)
		}
	})
}

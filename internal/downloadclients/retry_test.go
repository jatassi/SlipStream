package downloadclients_test

import (
	"context"
	"slices"
	"testing"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// retryAts is the Retry the intent carries for each of node's Torrents,
// 0 for none.
func retryAts(t *testing.T, lib library, node role.NodeID) map[dc.ID]int64 {
	t.Helper()
	out := map[dc.ID]int64{}
	err := lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		in, err := dc.Intent(ctx, r, node)
		for _, tor := range in.GetTorrents() {
			out[idFrom(tor.GetId())] = tor.GetRetryAtUnixMs()
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func rowOf(t *testing.T, c *dc.Clients, id dc.ID) dc.Row {
	t.Helper()
	l, err := c.List(t.Context(), dc.Filter{IDs: []dc.ID{id}})
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Rows) != 1 {
		t.Fatalf("listing %s found %d rows, want 1", id, len(l.Rows))
	}
	return l.Rows[0]
}

// checkedReport is what the Node reports once a Retry's check at found
// found of size bytes, held paused.
func checkedReport(at, found, size int64) *dlv1.TorrentReport {
	r := &dlv1.TorrentReport{
		AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Hold: dlv1.Hold_HOLD_WANTED,
		Done: found, Wanted: size, RetryCheckedUnixMs: at,
	}
	if found == size {
		r.Engine, r.Complete = dlv1.EngineState_ENGINE_STATE_SEEDING, true
	}
	return r
}

// applyTo has c apply r for id from node, as the Downloading Control
// does, and tells c it committed.
func applyTo(t *testing.T, lib library, c *dc.Clients, node role.NodeID, id dc.ID, r *dlv1.TorrentReport) {
	t.Helper()
	r.Id = id.Bytes()
	rep := &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{r}}}}
	write(t, lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return c.Apply(ctx, tx, node, rep)
	})
	c.Applied(node, string(role.Downloading))
}

// TestRetryChecksOnlyAFailedTorrentThatCompleted retries a seed that
// failed, a download that failed and a seed that didn't: only the failed
// seed is retried. Its Torrent is paused and the intent carries the Retry
// until the Node's report names its check, which finds all of its data,
// so it is started again. The same report again changes nothing.
func TestRetryChecksOnlyAFailedTorrentThatCompleted(t *testing.T) {
	s := newShelf(t)
	apply(t, s.lib, s.box, s.harbor, failed())
	apply(t, s.lib, s.box, s.orchard, failed())
	c, h := newClients(s.lib, &roster{}, nil, nil)
	if row := rowOf(t, c, s.harbor); !row.Retryable || rowOf(t, c, s.orchard).Retryable || rowOf(t, c, s.attic).Retryable {
		t.Fatalf("retryable reads %t, %t, %t for the failed seed, the failed download and the queued one; want only the failed seed",
			row.Retryable, rowOf(t, c, s.orchard).Retryable, rowOf(t, c, s.attic).Retryable)
	}

	results, _ := act(t, c, dc.Action{Kind: dc.Retry, IDs: []dc.ID{s.harbor, s.orchard, s.attic}})
	if got := refusals(results); got[0] != nil || got[1].Code != dc.RefusedNotRetryable || got[2].Code != dc.RefusedNotRetryable {
		t.Fatalf("Retry answered %+v, want the failed seed retried and the failed download and the queued one not_retryable", got)
	}
	if got := wantedOf(t, s.lib, s.harbor); got != "paused" {
		t.Fatalf("the retried Torrent reads %s, want paused while its Node checks it", got)
	}
	ats := retryAts(t, s.lib, s.box)
	at := ats[s.harbor]
	if at == 0 || ats[s.orchard] != 0 || ats[s.attic] != 0 {
		t.Fatalf("the intent carries the Retries %v, want one for the retried Torrent only", ats)
	}
	if got := h.touchedNodes(); !slices.Equal(got, []role.NodeID{s.box}) {
		t.Fatalf("Retry touched %v, want seedbox's intent", got)
	}

	size := rowOf(t, c, s.harbor).Size
	applyTo(t, s.lib, c, s.box, s.harbor, checkedReport(at, size, size))
	if got := wantedOf(t, s.lib, s.harbor); got != "started" {
		t.Errorf("once its check found all of its data, the retried Torrent reads %s, want started again", got)
	}
	row := rowOf(t, c, s.harbor)
	if row.Retry == nil || row.Retry.Found == nil || *row.Retry.Found != size || row.Pending != dc.Started || row.Retryable {
		t.Errorf("once its check found all of its data, the Torrent's row reads %+v with Retry %+v, want a resume pending and %d bytes found", row, row.Retry, size)
	}
	if got := retryAts(t, s.lib, s.box)[s.harbor]; got != 0 {
		t.Errorf("the intent still carries the settled Retry at %d, want none", got)
	}
	if got := h.touchedNodes(); len(got) != 2 {
		t.Errorf("the report that settled the Retry touched %v, want seedbox's intent again", got)
	}

	exec(t, s.lib, "UPDATE downloadclients_torrent SET wanted = 'paused' WHERE id = ?", s.harbor)
	applyTo(t, s.lib, c, s.box, s.harbor, checkedReport(at, size, size))
	if got := wantedOf(t, s.lib, s.harbor); got != "paused" {
		t.Errorf("the same report applied again set the Torrent %s, want the settled Retry to change nothing", got)
	}
}

// TestARetryThatFindsLessStaysPaused retries a failed seed whose check
// finds 40% of its data: it stays paused, and its row says what was
// found.
func TestARetryThatFindsLessStaysPaused(t *testing.T) {
	s := newShelf(t)
	apply(t, s.lib, s.box, s.harbor, failed())
	c, _ := newClients(s.lib, &roster{}, nil, nil)
	act(t, c, dc.Action{Kind: dc.Retry, IDs: []dc.ID{s.harbor}})
	at, size := retryAts(t, s.lib, s.box)[s.harbor], rowOf(t, c, s.harbor).Size
	found := size * 4 / 10

	applyTo(t, s.lib, c, s.box, s.harbor, checkedReport(at, found, size))
	if got := wantedOf(t, s.lib, s.harbor); got != "paused" {
		t.Errorf("once its check found %d of %d bytes, the retried Torrent reads %s, want paused", found, size, got)
	}
	row := rowOf(t, c, s.harbor)
	if row.Retry == nil || row.Retry.Found == nil || *row.Retry.Found != found || row.State != dc.StatePaused {
		t.Errorf("the row reads %+v with Retry %+v, want paused with %d bytes found", row, row.Retry, found)
	}
	if got := retryAts(t, s.lib, s.box)[s.harbor]; got != 0 {
		t.Errorf("the intent still carries the settled Retry at %d, want none", got)
	}
}

// TestARetryKeepsTheWantedStateTheTorrentHad retries a failed seed the
// Admin had paused, and a failed seed twice before its Node checks it:
// once all of its data is found each gets the wanted state it had before
// its first Retry.
func TestARetryKeepsTheWantedStateTheTorrentHad(t *testing.T) {
	s := newShelf(t)
	apply(t, s.lib, s.box, s.attic, completed())
	for _, id := range []dc.ID{s.harbor, s.attic} {
		apply(t, s.lib, s.box, id, failed())
	}
	c, _ := newClients(s.lib, &roster{}, nil, nil)
	act(t, c, dc.Action{Kind: dc.Pause, IDs: []dc.ID{s.attic}})
	act(t, c, dc.Action{Kind: dc.Retry, IDs: []dc.ID{s.harbor, s.attic}})
	act(t, c, dc.Action{Kind: dc.Retry, IDs: []dc.ID{s.harbor}})

	for id, want := range map[dc.ID]string{s.harbor: "started", s.attic: "paused"} {
		at, size := retryAts(t, s.lib, s.box)[id], rowOf(t, c, id).Size
		applyTo(t, s.lib, c, s.box, id, checkedReport(at, size, size))
		if got := wantedOf(t, s.lib, id); got != want {
			t.Errorf("once its check found all of its data, %s reads %s, want %s as before its Retry", id, got, want)
		}
	}
}

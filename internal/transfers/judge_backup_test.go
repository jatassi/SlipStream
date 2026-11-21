package transfers

import (
	"maps"
	"slices"
	"testing"
	"time"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

const backupName = "slipstream-20261114T040012Z-2.1.0-scheduled-7k3m.tar.zst.age"

// copyTo appends a Low copy of one Backup from the Controller's Backups
// folder to folder on dest, line order following n.
func (w *world) copyTo(n byte, dest role.NodeID, folder string) *world {
	w.s.transfers = append(w.s.transfers, Transfer{
		ID: tid(n), Purpose: Backup, Job: "copy", Source: home, Priority: Low, Created: t0.Add(time.Duration(n) * time.Second),
		Destination: BackupDestination{ID: [16]byte{n}, Node: dest, Folder: folder},
		Files:       []File{{Name: backupName, Source: source.Backup{Folder: "backups", Name: backupName}, Size: 900 * gb}},
	})
	return w
}

// fromHome appends a Normal import of one file from home into vps's root.
func (w *world) fromHome(n byte) *world { return w.add(n, home, onVps, gb) }

func states(j judgment) map[byte]string {
	out := map[byte]string{}
	for _, v := range j.views {
		out[v.ID.Bytes()[15]] = state(v)
	}
	return out
}

// TestALowCopyWaitsBehindNormalAndRunsAfterTheLimit: a copy queued first
// still waits behind a Normal Transfer on its pair, says when it next
// changes, and runs ahead of it once it has waited LowWait. A copy its
// receiver already runs isn't preempted by a newer Normal one.
func TestALowCopyWaitsBehindNormalAndRunsAfterTheLimit(t *testing.T) {
	w := newWorld().copyTo(1, vps, "/var/lib/slipstream/backups").fromHome(2)
	w.s.lowWait = 24 * time.Hour
	j := judge(w.s)
	same(t, "before the limit", states(j), map[byte]string{1: "queued 1", 2: "waiting connecting home route 1"})
	if want := t0.Add(time.Second + 24*time.Hour); !j.next.Equal(want) {
		t.Errorf("next is %v, want the copy's limit %v", j.next, want)
	}

	w.s.now = t0.Add(time.Second + 24*time.Hour)
	same(t, "at the limit", states(judge(w.s)), map[byte]string{1: "waiting connecting home route 1", 2: "queued 1"})

	w = newWorld().copyTo(1, vps, "/var/lib/slipstream/backups").fromHome(2).
		receiving(vps, &tv1.Receiving{TransferId: tid(1).Bytes(), Route: tv1.Route_ROUTE_RECEIVER_DIALS, Active: true, Moving: true})
	w.s.lowWait = 24 * time.Hour
	same(t, "a copy already running", states(judge(w.s)), map[byte]string{1: "copying route 1", 2: "queued 1"})
}

// TestNothingDispatchesWhileBlocked: while the library is Reconciling no
// Transfer is active on either end, each waits for Reconciling to finish,
// and every receiver still keeps its staging.
func TestNothingDispatchesWhileBlocked(t *testing.T) {
	w := newWorld().add(1, seedbox, onNas, gb).copyTo(2, vps, "/backups")
	w.s.paused = true
	j := judge(w.s)
	same(t, "states", states(j), map[byte]string{1: "waiting reconciling ", 2: "waiting reconciling "})
	if a := actives(j); len(a) > 0 {
		t.Errorf("active while Reconciling: %v", a)
	}
	for node, id := range map[role.NodeID]ID{nas: tid(1), vps: tid(2)} {
		if !slices.ContainsFunc(j.intents.of(node).GetKeep(), func(k *tv1.Kept) bool { got, _ := idOf(k.GetTransferId()); return got == id }) {
			t.Errorf("%s doesn't keep Transfer %d", names[node], id.Bytes()[15])
		}
	}
	if got := (View{State: Waiting, Wait: j.views[0].Wait}).Reason(names); got != "Waiting for SlipStream to finish Reconciling." {
		t.Errorf("the reason reads %q", got)
	}
}

// TestABackupTransferJudgesWithoutReadingTheIndex: a copy's record names
// its Node and folder, so judge activates it from the library snapshot
// alone, with no root and no space step, and the receiver's intent names
// the folder and the Backup name it renames the file to.
func TestABackupTransferJudgesWithoutReadingTheIndex(t *testing.T) {
	w := newWorld().copyTo(1, vps, `D:\Backups`)
	w.s.dests = nil
	j := judge(w.s)
	same(t, "states", states(j), map[byte]string{1: "waiting connecting home route 1"})
	recv := j.intents.of(vps).GetActive()
	if len(recv) != 1 {
		t.Fatalf("vps receives %d Transfers", len(recv))
	}
	got := recv[0].GetDestination().GetBackup()
	if got.GetFolder() != `D:\Backups` || got.GetName() != backupName {
		t.Errorf("vps receives into %q as %q", got.GetFolder(), got.GetName())
	}
	src := j.intents.of(home).GetActive()[0].GetFiles()[0].GetSource().GetBackup()
	if src.GetFolder() != "backups" || src.GetName() != backupName {
		t.Errorf("home sends %q from %q", src.GetName(), src.GetFolder())
	}
}

func same[K comparable, V comparable](t *testing.T, what string, got, want map[K]V) {
	t.Helper()
	if !maps.Equal(got, want) {
		t.Errorf("%s: got %v, want %v", what, got, want)
	}
}

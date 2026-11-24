package reconciling_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/reconciling"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// adoptionCases are the rows of the adoption table, each on a restore in
// place, with orchard held by vps-relay, and media-pc the Node another
// record names.
var adoptionCases = []struct {
	name string
	run  func(t *testing.T, w *world, relay, pc role.NodeID, orchard torrent)
}{
	{"kept", func(t *testing.T, w *world, relay, pc role.NodeID, orchard torrent) {
		id := w.record(relay, orchard)
		w.restore()
		w.reportAll(pc)
		w.reportAll(relay, orchard.entry())
		if got := w.only(orchard); got.id != id || got.node != relay || got.adopted || got.withheld {
			t.Errorf("orchard is recorded as %+v, want Torrent %s on vps-relay as it was", got, id)
		}
		if !w.lists(relay, orchard) || w.keeps(relay, orchard) {
			t.Errorf("vps-relay's intent lists orchard %t and keeps it unlisted %t, want listed only", w.lists(relay, orchard), w.keeps(relay, orchard))
		}
	}},
	{"moved after its Node reported without it", func(t *testing.T, w *world, relay, pc role.NodeID, orchard torrent) {
		id := w.record(pc, orchard)
		w.restore()
		w.reportAll(relay, orchard.entry())
		if got := w.only(orchard); got.node != pc || !w.keeps(relay, orchard) {
			t.Fatalf("before media-pc reports, orchard is recorded as %+v and kept by vps-relay %t, want on media-pc and kept", got, w.keeps(relay, orchard))
		}
		w.reportAll(pc)
		if got := w.only(orchard); got.id != id || got.node != relay || !got.live || got.adopted {
			t.Errorf("orchard is recorded as %+v, want Torrent %s moved to vps-relay", got, id)
		}
		if !w.lists(relay, orchard) || w.keeps(relay, orchard) || w.lists(pc, orchard) {
			t.Errorf("vps-relay lists orchard %t and keeps it unlisted %t, media-pc lists it %t; want vps-relay listing it alone",
				w.lists(relay, orchard), w.keeps(relay, orchard), w.lists(pc, orchard))
		}
	}},
	{"moved off a written-off Node with its Grab link", func(t *testing.T, w *world, relay, pc role.NodeID, orchard torrent) {
		id := w.record(pc, orchard)
		w.restore()
		w.writeOff(pc)
		w.reportAll(relay, orchard.entry())
		if got := w.only(orchard); got.id != id || got.node != relay || !got.live {
			t.Errorf("orchard is recorded as %+v, want Torrent %s live again on vps-relay, so its Grab still names it", got, id)
		}
	}},
	{"waits", func(t *testing.T, w *world, relay, pc role.NodeID, orchard torrent) {
		id := w.record(pc, orchard)
		w.restore()
		w.reportAll(relay, orchard.entry())
		if got := w.only(orchard); got.id != id || got.node != pc {
			t.Errorf("orchard is recorded as %+v, want still on media-pc", got)
		}
		if !w.keeps(relay, orchard) || w.lists(relay, orchard) || !w.pending(pc) {
			t.Errorf("vps-relay keeps orchard %t, lists it %t, media-pc keeps everything %t; want kept unlisted while media-pc keeps everything",
				w.keeps(relay, orchard), w.lists(relay, orchard), w.pending(pc))
		}
	}},
	{"a conflict, its Node reporting first", func(t *testing.T, w *world, relay, pc role.NodeID, orchard torrent) {
		id := w.record(pc, orchard)
		w.restore()
		w.reportAll(pc, orchard.entry())
		w.reportAll(relay, orchard.entry())
		conflicted(t, w, relay, pc, orchard, id)
	}},
	{"a conflict, its Node reporting last", func(t *testing.T, w *world, relay, pc role.NodeID, orchard torrent) {
		id := w.record(pc, orchard)
		w.restore()
		w.reportAll(relay, orchard.entry())
		w.reportAll(pc, orchard.entry())
		conflicted(t, w, relay, pc, orchard, id)
	}},
	{"adopted when unknown", func(t *testing.T, w *world, relay, _ role.NodeID, orchard torrent) {
		w.restore()
		w.exports(orchard)
		w.reportAll(relay, orchard.entry())
		got := w.only(orchard)
		if got.node != relay || !got.adopted || got.folder != "/downloads/"+orchard.name || !got.metainfo || got.seeding != 3600 {
			t.Errorf("orchard is recorded as %+v, want adopted on vps-relay in its folder with its metainfo and an hour's seeding", got)
		}
		if !w.lists(relay, orchard) {
			t.Error("vps-relay's intent doesn't list the adopted orchard")
		}
	}},
	{"stranded for an unknown hash with no resume data", func(t *testing.T, w *world, relay, _ role.NodeID, orchard torrent) {
		w.restore()
		w.exports(orchard)
		w.reportAll(relay, orchard.unsaved())
		if got := w.rows(orchard); len(got) != 0 {
			t.Errorf("orchard is recorded as %+v, want never adopted", got)
		}
		if !w.keeps(relay, orchard) || w.asked[orchard.hash] != 0 {
			t.Errorf("vps-relay keeps orchard %t, its metainfo was asked for %d times; want kept and never asked", w.keeps(relay, orchard), w.asked[orchard.hash])
		}
	}},
	{"not held after a restore goes on the review list", func(t *testing.T, w *world, relay, _ role.NodeID, orchard torrent) {
		id := w.record(relay, orchard)
		w.restore()
		w.reportAll(relay)
		if got := w.only(orchard); !got.withheld || w.review(id) != "not_held" {
			t.Errorf("orchard is recorded as %+v and on the review list as %q, want withheld and not_held", got, w.review(id))
		}
		if w.lists(relay, orchard) {
			t.Error("vps-relay's intent still lists orchard, which is on the review list")
		}
	}},
}

// TestEachAdoptionCase runs each row of the adoption table, and the not
// held row after a pairing.
func TestEachAdoptionCase(t *testing.T) {
	for _, c := range adoptionCases {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := newWorld(t)
				relay, pc := w.node("vps-relay"), w.node("media-pc")
				c.run(t, w, relay, pc, newTorrent(t, "Orchard"))
			})
		})
	}
	t.Run("not held after a pairing goes back into the intent", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			w := newWorld(t)
			relay := w.paired("vps-relay")
			orchard := newTorrent(t, "Orchard")
			id := w.record(relay, orchard)
			w.reportAll(relay)
			if got := w.only(orchard); got.withheld || w.review(id) != "" || !w.lists(relay, orchard) {
				t.Errorf("orchard is recorded as %+v, on the review list as %q, listed %t; want listed again and on no list", got, w.review(id), w.lists(relay, orchard))
			}
		})
	})
}

func conflicted(t *testing.T, w *world, relay, pc role.NodeID, orchard torrent, id dc.ID) {
	t.Helper()
	if got := w.only(orchard); got.id != id || got.node != pc {
		t.Errorf("orchard is recorded as %+v, want still on media-pc", got)
	}
	if !w.keeps(relay, orchard) || !w.lists(pc, orchard) || w.review(id) != "conflict" {
		t.Errorf("vps-relay keeps orchard %t, media-pc lists it %t, the review list has it as %q; want both keeping it as a conflict",
			w.keeps(relay, orchard), w.lists(pc, orchard), w.review(id))
	}
}

// TestANodeWithNoTorrentsIsNotGrantedDownloading pairs two Nodes: one
// whose first Holdings are empty is adopted with no Role, and one whose
// Holdings list a torrent gets Downloading in the Write that adopts it.
func TestANodeWithNoTorrentsIsNotGrantedDownloading(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		empty, relay := w.paired("laptop"), w.paired("vps-relay")
		orchard := newTorrent(t, "Orchard")
		w.exports(orchard)
		w.reportAll(empty)
		w.reportAll(relay, orchard.entry())
		if w.pending(empty) || len(w.roles(empty)) != 0 || w.granted(empty) != 0 {
			t.Errorf("laptop keeps everything %t with Roles %v granted at %d, want adopted with no Role", w.pending(empty), w.roles(empty), w.granted(empty))
		}
		if w.pending(relay) || !slices.Equal(w.roles(relay), []string{string(role.Downloading)}) || w.granted(relay) == 0 {
			t.Errorf("vps-relay keeps everything %t with Roles %v granted at %d, want adopted and granted Downloading", w.pending(relay), w.roles(relay), w.granted(relay))
		}
		if in := w.intent(relay); in.GetKeepAll() || in.GetHoldings() != nil {
			t.Errorf("vps-relay's intent keeps all %t and asks for Holdings %v after adoption, want neither", in.GetKeepAll(), in.GetHoldings())
		}
	})
}

// TestAHostileClaimMovesNothing has vps-relay claim a torrent recorded on
// media-pc: before media-pc reports, and after it reports holding it, the
// record stays on media-pc.
func TestAHostileClaimMovesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay, pc := w.node("vps-relay"), w.node("media-pc")
		orchard := newTorrent(t, "Orchard")
		id := w.record(pc, orchard)
		w.restore()
		w.exports(orchard)
		w.reportAll(relay, orchard.entry())
		if got := w.only(orchard); got.id != id || got.node != pc {
			t.Fatalf("after vps-relay's claim orchard is recorded as %+v, want on media-pc", got)
		}
		w.reportAll(pc, orchard.entry())
		if got := w.only(orchard); got.id != id || got.node != pc || !w.lists(pc, orchard) {
			t.Errorf("after media-pc reports it, orchard is recorded as %+v, listed by media-pc %t; want on media-pc", got, w.lists(pc, orchard))
		}
	})
}

// TestAMetainfoWhoseHashDiffersIsNotAdopted has vps-relay answer
// meadow's metainfo for orchard: nothing is adopted, and orchard is kept.
func TestAMetainfoWhoseHashDiffersIsNotAdopted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.paired("vps-relay")
		orchard, meadow := newTorrent(t, "Orchard"), newTorrent(t, "Meadow")
		w.meta[orchard.hash] = meadow.raw
		w.reportAll(relay, orchard.entry())
		if got := append(w.rows(orchard), w.rows(meadow)...); len(got) != 0 {
			t.Errorf("the library records %+v, want nothing adopted", got)
		}
		if !w.keeps(relay, orchard) {
			t.Error("vps-relay doesn't keep orchard")
		}
	})
}

// TestARefusedPageKeepsTheNodeKeeping sends a page past its round's page
// count: the Node keeps everything, says why, and a good page afterwards
// changes nothing.
func TestARefusedPageKeepsTheNodeKeeping(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.paired("vps-relay")
		orchard := newTorrent(t, "Orchard")
		w.exports(orchard)
		w.report(relay, w.round(relay), 3, 2, orchard.entry())
		w.reportAll(relay, orchard.entry())
		var refused string
		if err := w.lib.QueryRowContext(t.Context(), "SELECT refused FROM reconciling_node WHERE node_id = ?", relay).Scan(&refused); err != nil {
			t.Fatal(err)
		}
		if refused != "The Node sent page 4 of a list that has 2 pages." {
			t.Errorf("the refusal reads %q", refused)
		}
		if !w.pending(relay) || !w.intent(relay).GetKeepAll() || len(w.rows(orchard)) != 0 {
			t.Errorf("vps-relay keeps everything %t, its intent keeps all %t, orchard is recorded %+v; want kept, none adopted",
				w.pending(relay), w.intent(relay).GetKeepAll(), w.rows(orchard))
		}
	})
}

// TestAnOldRoundsPageIsDropped sends a page of a round the intent didn't
// ask for: it stores nothing, and the asked-for round still adopts.
func TestAnOldRoundsPageIsDropped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.paired("vps-relay")
		orchard, meadow := newTorrent(t, "Orchard"), newTorrent(t, "Meadow")
		w.exports(orchard, meadow)
		w.report(relay, w.round(relay)+1, 0, 1, meadow.entry())
		if !w.pending(relay) || len(w.rows(meadow)) != 0 {
			t.Fatalf("after another round's page vps-relay keeps everything %t and meadow is recorded %+v, want still waiting", w.pending(relay), w.rows(meadow))
		}
		w.reportAll(relay, orchard.entry())
		if w.pending(relay) || len(w.rows(orchard)) != 1 || len(w.rows(meadow)) != 0 {
			t.Errorf("vps-relay keeps everything %t, orchard %+v, meadow %+v; want orchard alone adopted", w.pending(relay), w.rows(orchard), w.rows(meadow))
		}
	})
}

// TestARepeatedPageCountsOnce sends the first of two pages twice: the
// round isn't in until the second arrives, and each torrent is adopted
// once.
func TestARepeatedPageCountsOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.paired("vps-relay")
		orchard, meadow := newTorrent(t, "Orchard"), newTorrent(t, "Meadow")
		w.exports(orchard, meadow)
		w.report(relay, w.round(relay), 0, 2, orchard.entry())
		w.report(relay, w.round(relay), 0, 2, orchard.entry())
		if !w.pending(relay) || len(w.rows(orchard)) != 0 {
			t.Fatalf("after page 1 twice vps-relay keeps everything %t and orchard is recorded %+v, want the round still open", w.pending(relay), w.rows(orchard))
		}
		w.report(relay, w.round(relay), 1, 2, meadow.entry())
		if w.pending(relay) || len(w.rows(orchard)) != 1 || len(w.rows(meadow)) != 1 {
			t.Errorf("vps-relay keeps everything %t, orchard %+v, meadow %+v; want each adopted once", w.pending(relay), w.rows(orchard), w.rows(meadow))
		}
	})
}

// TestAdoptionResumesAfterACrashMidPage adopts five torrents two a page.
// vps-relay can't answer for the second page's torrents, so the
// Controller stops with one page adopted; a fresh start adopts the rest,
// each once, and grants Downloading.
func TestAdoptionResumesAfterACrashMidPage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := idle(t, 2)
		relay := w.paired("vps-relay")
		trs := torrents(t, "Orchard", 5)
		w.exports(trs...)
		sorted := slices.Clone(trs)
		slices.SortFunc(sorted, func(a, b torrent) int { return slices.Compare(a.hash[:], b.hash[:]) })
		for _, tr := range sorted[2:4] {
			w.failing[tr.hash] = rpc.Errorf(rpc.CodeUnavailable, "the link is down")
		}
		ctx, crash := context.WithCancel(t.Context())
		w.start(ctx)
		for p := range 3 {
			w.report(relay, w.round(relay), uint32(p), 3, entries(trs[p*2:min(p*2+2, len(trs))])...)
		}
		adopted := 0
		for _, tr := range trs {
			adopted += len(w.rows(tr))
		}
		if adopted != 3 || !w.pending(relay) {
			t.Fatalf("before the crash %d torrents are adopted and vps-relay keeps everything %t, want the first page and the last", adopted, w.pending(relay))
		}
		crash()
		synctest.Wait()

		w.mu.Lock()
		clear(w.failing)
		w.mu.Unlock()
		w.start(t.Context())
		for _, tr := range trs {
			if got := w.rows(tr); len(got) != 1 || got[0].node != relay || !got[0].adopted {
				t.Errorf("%s is recorded as %+v, want adopted once on vps-relay", tr.name, got)
			}
		}
		if w.pending(relay) || w.granted(relay) == 0 {
			t.Errorf("vps-relay keeps everything %t, granted at %d; want adopted and granted Downloading", w.pending(relay), w.granted(relay))
		}
	})
}

// TestAPageOf5000AdoptsInBoundedWrites adopts 5,000 torrents in pages of
// 1,000: five page Writes and the one that grants, and neither query
// adoption reads the records with scans downloadclients_torrent.
func TestAPageOf5000AdoptsInBoundedWrites(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.paired("vps-relay")
		trs := torrents(t, "Orchard", 5000)
		w.exports(trs...)
		for p := range 5 {
			w.report(relay, w.round(relay), uint32(p), 5, entries(trs[p*1000:(p+1)*1000])...)
		}
		if w.pending(relay) || len(w.rows(trs[4999])) != 1 {
			t.Fatalf("vps-relay keeps everything %t, want its 5,000 torrents adopted", w.pending(relay))
		}
		if got := w.touched[relay]; got != 6 {
			t.Errorf("adoption touched vps-relay's intent %d times, want once for each of 5 page Writes and once for the grant", got)
		}
		scansNoTorrents(t, w, relay, trs[:1000])
	})
}

// TestTheEndNeedsEveryCondition begins Reconciling on a new machine with
// a root: it ends only once the folders are found, the Node is continued
// without, and the root is scanned.
func TestTheEndNeedsEveryCondition(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.node("vps-relay")
		root := storage.NewUUID()
		w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, "INSERT INTO roots_root (id, content_type, node_id, path, is_default) VALUES (?, ?, ?, '/media/fixture', 1)", root, contenttypetest.ID, relay)
			if err != nil {
				return err
			}
			return reconciling.Begin(ctx, tx, reconciling.BeginArgs{ID: "restore-1", Cause: reconciling.Restore, Backup: "nightly", TakenAt: time.Now(), NewMachine: true})
		})
		steps := []struct {
			what string
			sql  string
		}{
			{"the folders are found", "UPDATE reconciling_state SET rehome_saved_at = 1"},
			{"the root is scanned", "UPDATE reconciling_root SET saved_at = 1, scanned_at = 2"},
		}
		if got := w.waits(); !got.Rehome || len(got.Nodes) != 1 || len(got.Roots) != 1 {
			t.Fatalf("Reconciling waits on %+v, want the folders, vps-relay and the root", got)
		}
		for _, s := range steps {
			w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				_, err := tx.ExecContext(ctx, s.sql)
				return err
			})
			w.keeper.Wake()
			synctest.Wait()
			if w.phase() != reconciling.Reconciling {
				t.Fatalf("Reconciling ended once %s, want it waiting on vps-relay", s.what)
			}
		}
		w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return reconciling.ContinueWithout(ctx, tx, relay)
		})
		w.keeper.Wake()
		synctest.Wait()
		if w.phase() != reconciling.None {
			t.Fatalf("Reconciling still runs with %+v left", w.waits())
		}
		if !w.pending(relay) || !w.intent(relay).GetKeepAll() {
			t.Error("vps-relay, continued without, no longer keeps everything")
		}
	})
}

// scansNoTorrents fails t when either query adoption reads the records
// with scans downloadclients_torrent.
func scansNoTorrents(t *testing.T, w *world, node role.NodeID, trs []torrent) {
	t.Helper()
	hashes := make([]bencode.InfoHash, len(trs))
	for i, tr := range trs {
		hashes[i] = tr.hash
	}
	err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		plans := storagetest.Plan(t, r, func(r storage.Reader[storage.Library]) error {
			if _, err := dc.RecordedIn(ctx, r, node, hashes); err != nil {
				return err
			}
			_, err := dc.LiveOnNode(ctx, r, node, 0, len(trs))
			return err
		})
		if len(plans) != 2 {
			t.Fatalf("the two reads ran %d queries, want 2", len(plans))
		}
		for _, p := range plans {
			if slices.ContainsFunc(p.Lines, func(l string) bool { return strings.HasPrefix(l, "SCAN") && !strings.Contains(l, "json_each") }) {
				t.Errorf("%s scans every Torrent: %v", p.Query, p.Lines)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestAnExternalTorrentAdoptsWithItsHostsAndNoMetainfoWhenItCantExport
// has vps-relay's Transmission hold a torrent the library doesn't record.
// Transmission can't export its metainfo, so it is adopted on that client
// with its hosts and no metainfo, and stays off the review list, where
// Re-add would need the metainfo.
func TestAnExternalTorrentAdoptsWithItsHostsAndNoMetainfoWhenItCantExport(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.paired("vps-relay")
		client := storage.NewUUID()
		w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `INSERT INTO downloadclients_client (id, node_id, kind, name, address, address_key, label, is_on, settings_version,
				poll_state, poll_since, poll_problem, poll_settings_version, created_at, updated_at)
				VALUES (?, ?, 'transmission', 'Transmission', 'http://192.168.1.20:9091/transmission/rpc', 'a', 'tv', 1, 1, 'responding', 0, NULL, 1, 0, 0)`, client, relay)
			return err
		})
		orchard := newTorrent(t, "orchard")
		held := orchard.entry()
		held.ClientId = client.Bytes()
		w.reportAll(relay, held)
		got := w.only(orchard)
		var hosts, onClient string
		err := w.lib.QueryRowContext(t.Context(), "SELECT trackers, coalesce(hex(client_id), '') FROM downloadclients_torrent WHERE id = ?", got.id).Scan(&hosts, &onClient)
		if err != nil {
			t.Fatal(err)
		}
		if !got.adopted || got.metainfo || hosts != `["tracker.example"]` || onClient == "" || w.review(got.id) != "" || w.asked[orchard.hash] != 1 {
			t.Errorf("orchard is recorded as %+v with hosts %s on client %q, on the review list as %q, asked %d times; want adopted on Transmission with its host, no metainfo, off the list, asked once",
				got, hosts, onClient, w.review(got.id), w.asked[orchard.hash])
		}
	})
}

// TestARestoreAsksEachNodeForANewRound adopts a paired Node's Holdings,
// then begins Reconciling: its intent asks for a round it never sent, since
// a Worker sends each round once while it runs.
func TestARestoreAsksEachNodeForANewRound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWorld(t)
		relay := w.paired("vps-relay")
		sent := w.round(relay)
		w.reportAll(relay)
		if w.round(relay) != 0 {
			t.Fatal("vps-relay's Holdings weren't adopted")
		}
		w.restore()
		if again := w.round(relay); again == 0 || again == sent {
			t.Fatalf("after the restore vps-relay is asked for round %d, want one other than %d", again, sent)
		}
	})
}

// TestTheRestoredAndReplacedRowsGoOutOnce begins Reconciling on a new
// machine, then starts the Service twice: History records Backup restored
// and Controller replaced at the first start only.
func TestTheRestoredAndReplacedRowsGoOutOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := idle(t, 1000)
		w.node("home")
		taken := time.Now().Add(-time.Hour)
		w.write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return reconciling.Begin(ctx, tx, reconciling.BeginArgs{ID: "restore-1", Cause: reconciling.Restore, Backup: "nightly", TakenAt: taken, NewMachine: true})
		})
		first, stop := context.WithCancel(t.Context())
		w.start(first)
		restored, replaced := entrytest.Of(w.history, reconciling.RestoredRow), entrytest.Of(w.history, reconciling.ReplacedRow)
		if len(restored) != 1 || restored[0].Backup != "nightly" || !restored[0].TakenAt.Equal(taken) || len(replaced) != 1 || replaced[0].Controller != "home" {
			t.Fatalf("the first start recorded %+v and %+v, want the restore of nightly and the Controller home replaced", restored, replaced)
		}
		stop()
		synctest.Wait()
		w.start(t.Context())
		if n := w.history.Len(); n != 0 {
			t.Fatalf("the second start recorded %d rows, want none", n)
		}
	})
}

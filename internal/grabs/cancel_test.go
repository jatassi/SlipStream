package grabs_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

const canceledByKey = "This Grab was canceled through the API key."

func (p *pipeline) cancel(id grabs.GrabID, blocklist bool) error {
	return p.g.Cancel(p.t.Context(), identity.Admin{}, id, blocklist)
}

func (p *pipeline) canceled(id grabs.GrabID, at grabs.Stage) {
	p.t.Helper()
	v := p.view(id)
	if v.Stage != grabs.Failed || v.Failed.Stage != at || v.Failed.Code != grabs.Canceled || v.Failed.Message != canceledByKey || v.NeedsYou() {
		p.t.Fatalf("the Grab is %s with %+v, want failed at %s as canceled through the API key, needing no one", v.Stage, v.Failed, at)
	}
}

func (p *pipeline) count(query string, args ...any) int {
	p.t.Helper()
	var n int
	if err := p.lib.QueryRowContext(p.t.Context(), query, args...).Scan(&n); err != nil {
		p.t.Fatal(err)
	}
	return n
}

// TestCancelAtFetchingFailsTheGrabAndAFetchLandingAfterCommitsNothing
// cancels a Grab while its magnet link's metadata is on its way: the
// Grab fails canceled, and the metadata that lands after attaches no
// Torrent. A second cancel changes nothing.
func TestCancelAtFetchingFailsTheGrabAndAFetchLandingAfterCommitsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel := newRelease(t)
		p := newPipeline(t, answer{res: downloadclients.Resolved{Info: rel.info}})
		id := p.grab(rel.link.String(), nil)
		p.running()
		if v := p.view(id); v.Stage != grabs.Fetching || v.Metadata == nil {
			t.Fatalf("before the cancel the Grab is %s showing %+v, want fetching its metadata", v.Stage, v.Metadata)
		}

		for range 2 {
			if err := p.cancel(id, false); err != nil {
				t.Fatal(err)
			}
		}
		time.Sleep(2 * hold)

		p.canceled(id, grabs.Fetching)
		if n := len(p.swarm.asked()); n != 1 {
			t.Fatalf("the Node was asked for the metadata %d times, want once", n)
		}
		if n := p.count(`SELECT count(*) FROM downloadclients_torrent`); n != 0 {
			t.Fatalf("the late metadata added %d Torrents, want none", n)
		}
		if n := p.count(`SELECT count(*) FROM grabs_stage WHERE grab_id = ? AND stage = 'failed'`, id); n != 1 {
			t.Fatalf("the Grab entered failed %d times, want once", n)
		}
	})
}

// TestCancelingAnAutomaticGrabAtFetchingEndsWhatItsRunKeptForARerun
// cancels the Grab an automatic search took before its fetch: its run's
// sealed links go, since no rerun follows a cancel.
func TestCancelingAnAutomaticGrabAtFetchingEndsWhatItsRunKeptForARerun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tw := newTwinWorld(t, twin{sites: []twinSite{{name: "Better Site", priority: 10, releases: []searching.Release{listing(bluRay, 90, "a"), listing(webDL, 5, "b")}}}})
		tw.cfg.BeforeStep = func(ctx context.Context, s grabs.Stage) {
			if s == grabs.Fetching {
				<-ctx.Done()
			}
		}
		tw.g = tw.open()
		if got := tw.searchedAutomatically(t); got != "Better Site "+bluRay {
			t.Fatalf("the search grabbed %q", got)
		}
		kept := func() int {
			var n int
			if err := tw.lib.QueryRowContext(t.Context(), `SELECT count(*) FROM grabs_release WHERE link_sealed IS NOT NULL`).Scan(&n); err != nil {
				t.Fatal(err)
			}
			return n
		}
		if kept() == 0 {
			t.Fatal("the run kept no link for a rerun before the cancel")
		}
		runs, err := tw.g.Searches(t.Context(), identity.Admin{}, tw.work)
		if err != nil {
			t.Fatal(err)
		}
		if err := tw.g.Cancel(t.Context(), identity.Admin{}, runs[0].GrabID, false); err != nil {
			t.Fatal(err)
		}
		if n := kept(); n != 0 {
			t.Fatalf("the run keeps %d links after its Grab was canceled, want none", n)
		}
	})
}

// TestCancelAtDownloadingRemovesTheTorrentAndFailsItsOtherGrabs cancels
// one of two Grabs downloading one Torrent on the built-in client: the
// Torrent goes with reason canceled and its files, and the other Grab
// fails canceled by that.
func TestCancelAtDownloadingRemovesTheTorrentAndFailsItsOtherGrabs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPipeline(t)
		id := downloadingGrab(t, p, downloadingReport)
		other := storage.NewUUID()
		run(t, p.lib, sql(`INSERT INTO grabs_grab (id, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size,
			release_magnet, stage, torrent_id, created_at, updated_at)
			SELECT ?, work_id, item_id, edition, source, indexer_id, node_id, release_title, release_size, release_magnet, 'downloading', torrent_id, created_at, updated_at
			FROM grabs_grab`, other))

		if err := p.cancel(id, false); err != nil {
			t.Fatal(err)
		}
		p.running()
		time.Sleep(time.Second)

		p.canceled(id, grabs.Downloading)
		if n := p.count(`SELECT count(*) FROM downloadclients_torrent WHERE removed_reason = 'canceled' AND delete_files = 1`); n != 1 {
			t.Fatal("the Torrent wasn't removed with reason canceled and its files")
		}
		v := p.view(grabs.GrabID{UUID: other})
		if v.Stage != grabs.Failed || v.Failed.Code != grabs.Canceled || v.Failed.Message != "This Grab's torrent was removed when another Grab of it was canceled." {
			t.Fatalf("the other Grab of the Torrent is %s with %+v, want failed canceled", v.Stage, v.Failed)
		}
	})
}

// TestCancelAtDownloadingKeepsTheFilesOfATorrentSlipStreamAttached
// cancels a Grab of a torrent that was in Transmission before SlipStream
// attached it: SlipStream stops tracking it at once and leaves its files.
func TestCancelAtDownloadingKeepsTheFilesOfATorrentSlipStreamAttached(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSending(t, downloadingReport)
		run(t, s.lib, sql(`UPDATE downloadclients_torrent SET client_id = ?, attached = 1`, s.transmission))

		if err := s.cancel(s.grab, false); err != nil {
			t.Fatal(err)
		}

		s.canceled(s.grab, grabs.Downloading)
		if n := s.count(`SELECT count(*) FROM downloadclients_torrent WHERE removed_reason = 'canceled' AND parted_at IS NOT NULL AND delete_files = 0`); n != 1 {
			t.Fatal("the attached torrent wasn't parted at once with its files kept")
		}
	})
}

// TestCancelAtImportingEndsItsImportAndFailsTheGrabWithNothingPlaced
// cancels an importing Grab whose Import placed nothing: its Import ends
// as canceled, and the Grab fails canceled at importing.
func TestCancelAtImportingEndsItsImportAndFailsTheGrabWithNothingPlaced(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := inTransmission(t, grabs.Importing)

		if err := s.cancel(s.grab, false); err != nil {
			t.Fatal(err)
		}

		s.canceled(s.grab, grabs.Importing)
	})
}

// TestCancelRefusesAGrabPlacingItsFileOrFinished cancels a Grab at each
// stage a cancel can't stop: an importing one whose Import places a file
// is busy, saying which Node it waits for while that Node is away, and
// imported and failed are finished. Each is left as it was.
func TestCancelRefusesAGrabPlacingItsFileOrFinished(t *testing.T) {
	const waits = "This Grab is waiting for home to come back. It can't be canceled until it places its file."
	for _, c := range []struct {
		stage grabs.Stage
		away  bool
		busy  *grabs.GrabBusyError
	}{
		{grabs.Importing, false, &grabs.GrabBusyError{Stage: grabs.Importing}},
		{grabs.Importing, true, &grabs.GrabBusyError{Stage: grabs.Importing, Words: waits}},
		{grabs.Imported, false, nil},
		{grabs.Failed, false, nil},
	} {
		t.Run(fmt.Sprintf("%s away %v", c.stage, c.away), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s := inTransmission(t, grabs.Importing)
				s.imports.busy = true
				run(t, s.lib, sql(`UPDATE grabs_grab SET stage = ?, failed_stage = CASE WHEN ? = 'failed' THEN 'importing' END,
					failure_code = CASE WHEN ? = 'failed' THEN 'import_failed' END, failure_message = CASE WHEN ? = 'failed' THEN 'The disk is full.' END`,
					string(c.stage), string(c.stage), string(c.stage), string(c.stage)))
				if c.away {
					s.roster.takeAway(s.node, s.media)
				}

				if err := s.cancel(s.grab, true); !refusedAs(err, c.busy) {
					t.Fatalf("cancel = %v, want %+v, or ErrGrabFinished when nil", err, c.busy)
				}
				if v := s.view(s.grab); v.Stage != c.stage || s.count(`SELECT count(*) FROM grabs_blocklist`) != 0 {
					t.Fatalf("after the refused cancel the Grab is %s, want %s and nothing blocklisted", v.Stage, c.stage)
				}
			})
		})
	}
}

// refusedAs reports whether err is busy, or ErrGrabFinished when busy is
// nil.
func refusedAs(err error, busy *grabs.GrabBusyError) bool {
	if busy == nil {
		return errors.Is(err, grabs.ErrGrabFinished)
	}
	got, ok := errors.AsType[*grabs.GrabBusyError](err)
	return ok && *got == *busy
}

// TestAlsoBlocklistAddsAnEntryOnlyForAGrabWithAnInfoHash cancels with
// "Also blocklist" a Grab with a magnet link's info hash, which gets a
// by_hand entry the API key added, and one whose link names none, which
// gets none and isn't offered it.
func TestAlsoBlocklistAddsAnEntryOnlyForAGrabWithAnInfoHash(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPipeline(t)
		rel := newRelease(t)
		hashed, unhashed := p.grab(rel.link.String(), nil), p.grab("magnet:?dn=The.Attic.1080p.WEB", nil)
		for _, c := range []struct {
			id    grabs.GrabID
			offer bool
		}{{hashed, true}, {unhashed, false}} {
			if a := p.view(c.id).Actions; a != (grabs.Actions{Cancel: true, Blocklist: c.offer}) {
				t.Errorf("a fetching Grab offers %+v, want a cancel with Blocklist %v", a, c.offer)
			}
			if err := p.cancel(c.id, true); err != nil {
				t.Fatal(err)
			}
			p.canceled(c.id, grabs.Fetching)
		}

		if n := p.count(`SELECT count(*) FROM grabs_blocklist`); n != 1 {
			t.Fatalf("the Blocklist holds %d entries, want the hashed Grab's alone", n)
		}
		if n := p.count(`SELECT count(*) FROM grabs_blocklist WHERE grab_id = ? AND info_hash = ? AND cause = 'by_hand'
			AND added_by IS NULL AND added_by_name IS NULL`, hashed, rel.link.InfoHash[:]); n != 1 {
			t.Fatal("the entry isn't the magnet link's info hash, by hand, through the API key")
		}
	})
}

// TestCancelAtDownloadingRefusesWhileReconciling cancels a downloading
// Grab while a restored Controller is Reconciling: the cancel refuses and
// the Torrent and the Grab stay.
func TestCancelAtDownloadingRefusesWhileReconciling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPipeline(t)
		id := downloadingGrab(t, p, downloadingReport)
		p.reconciling.Store(true)

		if err := p.cancel(id, false); !errors.Is(err, grabs.ErrReconciling) {
			t.Fatalf("cancel while Reconciling = %v, want ErrReconciling", err)
		}
		if v := p.view(id); v.Stage != grabs.Downloading || p.count(`SELECT count(*) FROM downloadclients_torrent WHERE removed_at IS NULL`) != 1 {
			t.Fatalf("after the refused cancel the Grab is %s, want downloading its Torrent still", v.Stage)
		}
	})
}

// TestACanceledGrabLeavesItsCellMissingWithNothingNeedingTheAdmin cancels
// a Grab of a cell an earlier Grab failed, and reads the cells: the cell
// reads as if no Grab had it, and the Grab needs no one.
func TestACanceledGrabLeavesItsCellMissingWithNothingNeedingTheAdmin(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := newPipeline(t)
		rel := newRelease(t)
		earlier, id := p.grab("magnet:?dn=The.Attic.1080p.WEB", nil), p.grab(rel.link.String(), nil)
		target := storage.NewUUID()
		run(t, p.lib,
			sql(`INSERT INTO targets_target (id, name, created_at) VALUES (?, 'HD', 0)`, target),
			sql(`INSERT INTO grabs_target (grab_id, position, item_id, target_id) SELECT id, 0, item_id, ? FROM grabs_grab`, target),
			sql(`UPDATE grabs_grab SET stage = 'failed', failed_stage = 'fetching', failure_code = 'magnet_invalid', failure_message = 'The link names no info hash.',
				created_at = created_at - 10 WHERE id = ?`, earlier),
		)
		cells := func() map[files.CellKey]files.GrabCell {
			var got map[files.CellKey]files.GrabCell
			err := p.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
				var err error
				got, err = grabs.GrabCells{InStep: func(role.NodeID) bool { return true }, Router: byRoot{}}.GrabCells(ctx, r, works.Only([]works.WorkID{p.work}))
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			return got
		}
		if got := cells(); len(got) != 1 {
			t.Fatalf("before the cancel the cells read %+v, want the fetching Grab's", got)
		}

		if err := p.cancel(id, false); err != nil {
			t.Fatal(err)
		}

		if got := cells(); len(got) != 0 {
			t.Fatalf("after the cancel the cells read %+v, want none, as with no Grab", got)
		}
		if p.view(id).NeedsYou() {
			t.Fatal("the canceled Grab needs the Admin")
		}
	})
}

func TestADownloadingGrabOffersCancelBlocklistAndSendToAndAnImportingOneCancelAndBlocklist(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSending(t, downloadingReport)
		if a := s.view(s.grab).Actions; a != (grabs.Actions{Cancel: true, Blocklist: true, SendTo: true}) {
			t.Fatalf("a downloading Grab offers %+v, want all three", a)
		}
		run(t, s.lib, sql(`UPDATE grabs_grab SET stage = 'importing'`))
		if a := s.view(s.grab).Actions; a != (grabs.Actions{Cancel: true, Blocklist: true}) {
			t.Fatalf("an importing Grab offers %+v, want a cancel and Blocklist", a)
		}
	})
}

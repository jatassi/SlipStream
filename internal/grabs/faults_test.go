package grabs_test

import (
	"context"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/identity"
	rel "github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// removing is the swarm with its Torrent removals carried out by
// downloadclients, as a Controller's are.
type removing struct {
	*swarm
	clients *downloadclients.Clients
}

func (r removing) Moved() <-chan struct{} { return r.clients.Moved() }

func (r removing) Touched(ts ...downloadclients.Touched) { r.clients.Touched(ts...) }

func (r removing) RemoveIn(ctx context.Context, tx *storage.Tx[storage.Library], ids []downloadclients.ID, reason downloadclients.RemoveReason, deleteFiles bool) ([]downloadclients.Result, downloadclients.Touched, error) {
	return r.clients.RemoveIn(ctx, tx, ids, reason, deleteFiles)
}

// faulty is a pipeline that removes Torrents and reads its Blocklist.
type faulty struct {
	*pipeline
}

func newFaulty(t *testing.T, answers ...answer) *faulty {
	t.Helper()
	f := &faulty{pipeline: newPipeline(t, answers...)}
	goals, err := targets.Open(t.Context(), targets.Config{
		Library: f.lib, Works: f.cfg.Works, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: contenttypetest.New()},
		Changed: discard[targets.Changed]{}, Formats: formats.DefaultTimings(),
	})
	if err != nil {
		t.Fatal(err)
	}
	clients := downloadclients.New(downloadclients.Config{
		Library: f.lib, Touch: func(...role.NodeID) {}, Changed: discard[downloadclients.Changed]{},
		Uses: grabs.TorrentUses{}, Log: slog.New(slog.DiscardHandler),
	})
	f.cfg.Targets, f.cfg.Names = goals, map[works.TypeID]rel.Names{contenttypetest.ID: dated{}}
	f.cfg.Downloads = removing{swarm: f.swarm, clients: clients}
	f.copies = transfers.New(transfers.ControllerConfig{Library: f.lib})
	f.cfg.Transfers = f.copies
	f.g = grabs.New(f.cfg)
	return f
}

// downloading is a Grab at downloading whose Torrent is part done.
func (f *faulty) downloading() grabs.GrabID {
	f.t.Helper()
	id := downloadingGrab(f.t, f.pipeline, seedingReport)
	run(f.t, f.lib, sql(`UPDATE downloadclients_torrent SET report = ?, completion = NULL, completed_at = NULL`, downloadingReport))
	return id
}

// fault records kind on the Torrent, as its Node reports it.
func (f *faulty) fault(kind downloadclients.FaultKind) {
	f.t.Helper()
	run(f.t, f.lib, sql(`UPDATE downloadclients_torrent SET fault = ?, fault_at = ?`, string(kind), storage.Now()))
}

func (f *faulty) entries() []grabs.BlockEntry {
	f.t.Helper()
	synctest.Wait()
	page, err := f.g.Blocklist(f.t.Context(), identity.Admin{}, grabs.BlocklistQuery{})
	if err != nil {
		f.t.Fatal(err)
	}
	return page.Entries
}

// torrent is the one Torrent as its Grabs read it.
func (f *faulty) torrent() downloadclients.Attached {
	f.t.Helper()
	id := torrentOf(f.t, f.pipeline)
	var got map[downloadclients.ID]downloadclients.Attached
	err := f.lib.Read(f.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = downloadclients.Outcomes(ctx, r, []downloadclients.ID{id})
		return err
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return got[id]
}

// told is how many Torrents the Node is told to keep, and the delete-files
// flag of each it is told to remove.
func (f *faulty) told() (keeps int, removes []bool) {
	f.t.Helper()
	err := f.lib.Read(f.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		intent, err := downloadclients.Intent(ctx, r, f.node)
		if err != nil {
			return err
		}
		keeps = len(intent.Torrents)
		for _, d := range intent.Discards {
			removes = append(removes, d.DeleteFiles)
		}
		return nil
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return keeps, removes
}

func (f *faulty) failedAtDownloading(id grabs.GrabID, code grabs.Code) {
	f.t.Helper()
	if v := f.view(id); v.Stage != grabs.Failed || v.Failed.Stage != grabs.Downloading || v.Failed.Code != code {
		f.t.Errorf("the Grab is %s with %+v, want failed at downloading with %s", v.Stage, v.Failed, code)
	}
}

func (f *faulty) faultEntry(want grabs.BlockEntry) {
	f.t.Helper()
	got := f.entries()
	if len(got) != 1 {
		f.t.Fatalf("the Blocklist holds %d entries, want 1", len(got))
	}
	e := got[0]
	if e.InfoHash != want.InfoHash || e.Grab != want.Grab || e.Title != want.Title || e.Cause != want.Cause || e.Because != want.Because || e.By != nil || e.IndexerName != "Fixture Site" {
		f.t.Errorf("the entry is %+v, want %+v from Fixture Site with no actor", e, want)
	}
}

func isGoneForFault(a downloadclients.Attached) bool {
	gone, ok := a.Outcome.(downloadclients.Gone)
	return ok && gone.Reason == downloadclients.ReleaseFaulted
}

func TestAReleaseFaultOnADownloadingGrabBlocklistsRemovesTheTorrentWithItsCopyAndFailsEachDownloadingGrab(t *testing.T) {
	for _, c := range []struct {
		kind    downloadclients.FaultKind
		cause   grabs.BlockCause
		because string
		code    grabs.Code
	}{
		{downloadclients.FaultUnregistered, grabs.CauseUnregistered, "Fixture Site reported it unregistered.", grabs.ReleaseFault},
		{downloadclients.FaultNoPayload, grabs.CauseNoPayload, "It never downloaded any data.", grabs.ReleaseFault},
		{downloadclients.FaultEngineError, grabs.CauseEngineError, "home's built-in client couldn't read it.", grabs.DownloadFailed},
	} {
		t.Run(string(c.kind), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := newFaulty(t)
				first := f.downloading()
				time.Sleep(time.Second)
				newest := f.grab(newRelease(t).link.String(), nil)
				run(t, f.lib, sql(`UPDATE grabs_grab SET stage = 'downloading', release_title = 'The.Attic.1080p.WEB.REPACK',
					torrent_id = (SELECT id FROM downloadclients_torrent) WHERE id = ?`, newest))
				f.fault(c.kind)
				hash := f.torrent().InfoHash

				f.running()

				f.failedAtDownloading(first, c.code)
				f.failedAtDownloading(newest, c.code)
				f.faultEntry(grabs.BlockEntry{InfoHash: hash, Grab: newest, Title: "The.Attic.1080p.WEB.REPACK", Cause: c.cause, Because: c.because})
				if a := f.torrent(); !isGoneForFault(a) {
					t.Errorf("the Torrent reads %+v, want removed for release_fault", a.Outcome)
				}
				if keeps, removes := f.told(); keeps != 0 || len(removes) != 1 || !removes[0] {
					t.Errorf("the Node keeps %d Torrents and removes %v, want the one removed with its files", keeps, removes)
				}
			})
		})
	}
}

func TestAReleaseFaultPastDownloadingBlocklistsAndTheGrabCarriesOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFaulty(t)
		id := downloadingGrab(t, f.pipeline, seedingReport)
		run(t, f.lib, sql(`UPDATE grabs_grab SET stage = 'probing'`))
		f.fault(downloadclients.FaultUnregistered)
		f.beforeStep = func(ctx context.Context, s grabs.Stage) {
			if s == grabs.Probing {
				<-ctx.Done()
			}
		}

		f.running()

		f.faultEntry(grabs.BlockEntry{InfoHash: f.torrent().InfoHash, Grab: id, Title: "The.Attic.1080p.WEB", Cause: grabs.CauseUnregistered, Because: "Fixture Site reported it unregistered."})
		if v := f.view(id); v.Stage != grabs.Probing {
			t.Errorf("the Grab is %s, want still probing", v.Stage)
		}
		if a := f.torrent(); a.Outcome != (downloadclients.Completed{}) {
			t.Errorf("the Torrent reads %+v, want it complete", a.Outcome)
		}
		if keeps, removes := f.told(); keeps != 1 || len(removes) != 0 {
			t.Errorf("the Node keeps %d Torrents and removes %v, want it kept", keeps, removes)
		}
	})
}

func TestAReleaseFaultOnAnAttachedTorrentStopsTrackingItAndKeepsItsFiles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFaulty(t)
		id := f.downloading()
		client := storage.NewUUID()
		run(t, f.lib,
			sql(`INSERT INTO downloadclients_client (id, node_id, kind, name, address, address_key, label, is_on, settings_version,
				poll_state, poll_since, poll_settings_version, created_at, updated_at)
				VALUES (?, ?, 'transmission', 'Transmission', 'http://192.168.1.20:9091/transmission/rpc', 'a', 'tv-sonarr', 1, 1, 'responding', 0, 1, 0, 0)`, client, f.node),
			sql(`UPDATE downloadclients_torrent SET client_id = ?, attached = 1`, client),
		)
		f.fault(downloadclients.FaultUnregistered)
		hash := f.torrent().InfoHash

		f.running()

		f.failedAtDownloading(id, grabs.ReleaseFault)
		f.faultEntry(grabs.BlockEntry{InfoHash: hash, Grab: id, Title: "The.Attic.1080p.WEB", Cause: grabs.CauseUnregistered, Because: "Fixture Site reported it unregistered."})
		if e := f.entries()[0]; e.Client != "Transmission on home" {
			t.Errorf("the entry names the client %q, want Transmission on home", e.Client)
		}
		if a := f.torrent(); !isGoneForFault(a) {
			t.Errorf("the Torrent reads %+v, want removed for release_fault", a.Outcome)
		}
		if keeps, removes := f.told(); keeps != 0 || len(removes) != 0 {
			t.Errorf("the Node keeps %d Torrents and removes %v, want it untracked and nothing removed from its client", keeps, removes)
		}
	})
}

func TestAReleaseFaultWithNoUnfinishedGrabBlocklistsOnceAndTheTorrentKeepsSeeding(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFaulty(t)
		id := downloadingGrab(t, f.pipeline, seedingReport)
		run(t, f.lib, sql(`UPDATE grabs_grab SET stage = 'failed', failed_stage = 'probing', failure_code = 'probe_failed', failure_message = 'x'`))
		f.fault(downloadclients.FaultNoPayload)
		hash := f.torrent().InfoHash

		f.running()

		f.faultEntry(grabs.BlockEntry{InfoHash: hash, Grab: id, Title: "The.Attic.1080p.WEB", Cause: grabs.CauseNoPayload, Because: "It never downloaded any data."})
		if keeps, removes := f.told(); keeps != 1 || len(removes) != 0 {
			t.Errorf("the Node keeps %d Torrents and removes %v, want it seeding", keeps, removes)
		}

		if err := f.g.Unblock(t.Context(), identity.Admin{}, []bencode.InfoHash{hash}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Hour)
		if got := f.entries(); len(got) != 0 {
			t.Errorf("the Blocklist holds %+v after its entry was removed, want the settled fault never to add it again", got)
		}
	})
}

func TestAReleaseFaultKeepsAnEntryAlreadyListed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFaulty(t)
		id := f.downloading()
		f.fault(downloadclients.FaultUnregistered)
		hash := f.torrent().InfoHash
		grabstest.Blocked(t, f.lib, grabs.BlockEntry{InfoHash: hash, Title: "The.Attic.1080p.WEB", Size: 4096, Cause: grabs.CauseByHand, By: &grabs.Actor{Name: "Jackson"}})

		f.running()

		f.failedAtDownloading(id, grabs.ReleaseFault)
		if got := f.entries(); len(got) != 1 || got[0].Because != "Jackson blocklisted it when canceling its Grab." {
			t.Errorf("the Blocklist holds %+v, want Jackson's entry alone", got)
		}
	})
}

func TestAReleaseFaultWaitsWhileReconcilingAndSettlesOnceItEnds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFaulty(t)
		f.reconciling.Store(true)
		id := f.downloading()
		f.fault(downloadclients.FaultUnregistered)

		f.running()
		time.Sleep(time.Minute)

		if got := f.entries(); len(got) != 0 {
			t.Errorf("the Blocklist holds %+v while Reconciling, want none", got)
		}
		if v := f.view(id); v.Stage != grabs.Downloading {
			t.Errorf("the Grab is %s while Reconciling, want still downloading", v.Stage)
		}
		if a := f.torrent(); a.Outcome != (downloadclients.Pending{}) {
			t.Errorf("the Torrent reads %+v while Reconciling, want it kept", a.Outcome)
		}

		f.reconciling.Store(false)
		time.Sleep(time.Minute)

		f.failedAtDownloading(id, grabs.ReleaseFault)
		if got := f.entries(); len(got) != 1 || got[0].Cause != grabs.CauseUnregistered {
			t.Errorf("the Blocklist holds %+v once Reconciling ended, want the fault's entry", got)
		}
	})
}

func TestSetupFaultsAndNoMetadataNeverBlocklist(t *testing.T) {
	for _, r := range []struct {
		reason downloadclients.RemoveReason
		code   grabs.Code
	}{
		{downloadclients.ByClient, grabs.Removed},
		{downloadclients.ClientDeleted, grabs.Removed},
		{downloadclients.NodeWrittenOff, grabs.NodeWrittenOff},
	} {
		t.Run(string(r.reason), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				f := newFaulty(t)
				id := f.downloading()
				now := storage.Now()
				run(t, f.lib, sql(`UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = ?, parted_at = ?, freed = 0`, now, string(r.reason), now))

				f.running()

				f.failedAtDownloading(id, r.code)
				if got := f.entries(); len(got) != 0 {
					t.Errorf("the Blocklist holds %+v, want none", got)
				}
			})
		})
	}
	t.Run("engine failure", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			f := newFaulty(t)
			id := f.downloading()
			run(t, f.lib, sql(`UPDATE downloadclients_torrent SET report = '{"at":"2026-01-01T00:00:00Z","engine":"failed","error":"No space left on device"}'`))

			f.running()

			f.failedAtDownloading(id, grabs.DownloadFailed)
			if got := f.entries(); len(got) != 0 {
				t.Errorf("the Blocklist holds %+v, want none", got)
			}
		})
	})
	t.Run("no_metadata", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			magnetized := newRelease(t)
			f := newFaulty(t, answer{res: downloadclients.Unresolved{Reason: downloadclients.TimedOut, MostPeers: 2, TrackerMessage: "unregistered torrent"}})
			id := f.grab(magnetized.link.String(), &magnetized.link.InfoHash)

			f.running()

			f.failed(id, grabs.NoMetadata)
			if got := f.entries(); len(got) != 0 {
				t.Errorf("the Blocklist holds %+v, want none", got)
			}
		})
	})
}

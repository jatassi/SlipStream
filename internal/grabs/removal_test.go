package grabs_test

import (
	"context"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

var removals = []struct {
	reason downloadclients.RemoveReason
	code   grabs.Code
}{
	{downloadclients.ByAdmin, grabs.Removed},
	{downloadclients.ByMarkFailed, grabs.DownloadFailed},
	{downloadclients.NodeWrittenOff, grabs.NodeWrittenOff},
}

// Mark failed takes only a stalled download, so its Torrent stalls once
// complete, as after a recheck.
const (
	seedingReport = `{"at":"2026-01-01T00:00:00Z","engine":"seeding","complete":true,"done":4096}`
	stalledReport = `{"at":"2026-01-01T00:00:00Z","engine":"downloading","stalled":true,"done":4096}`
)

func downloadingGrab(t *testing.T, p *pipeline, report string) grabs.GrabID {
	t.Helper()
	rel := newRelease(t)
	id := p.grab(rel.link.String(), &rel.link.InfoHash)
	raw, err := rel.link.Torrent(rel.info)
	if err != nil {
		t.Fatal(err)
	}
	attachTorrent(t, p, raw)
	now := storage.Now()
	run(t, p.lib,
		sql(`UPDATE downloadclients_torrent SET report = ?, reported_at = ?, completed_at = ?,
			completion = '{"files":[{"Index":0,"Path":"The.Attic.1080p.WEB.nfo","Size":4096}]}'`, report, now, now),
		sql(`UPDATE grabs_grab SET stage = 'downloading', torrent_id = (SELECT id FROM downloadclients_torrent)`),
	)
	return id
}

func torrentOf(t *testing.T, p *pipeline) downloadclients.ID {
	t.Helper()
	var id downloadclients.ID
	err := p.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, `SELECT id FROM downloadclients_torrent`).Scan(&id)
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// TestADownloadingGrabFailsOnceItsTorrentIsParted parts a downloading
// Grab's complete Torrent for each reason before the Grabs read it, and
// fails the Grab at downloading with that reason's code rather than
// moving it on.
func TestADownloadingGrabFailsOnceItsTorrentIsParted(t *testing.T) {
	for _, r := range removals {
		t.Run(string(r.reason), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				p := newPipeline(t)
				id := downloadingGrab(t, p, seedingReport)
				now := storage.Now()
				run(t, p.lib, sql(`UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = ?, parted_at = ?, freed = 0`, now, string(r.reason), now))

				p.running()
				time.Sleep(time.Second)

				v := p.view(id)
				if v.Stage != grabs.Failed || v.Failed.Stage != grabs.Downloading || v.Failed.Code != r.code {
					t.Fatalf("the Grab is %s with %+v, want failed at downloading with %s", v.Stage, v.Failed, r.code)
				}
			})
		})
	}
}

// TestAReleaseFaultOrACancelFailsADownloadingGrabByItsCause parts a
// downloading Grab's Torrent for a release fault or another Grab's cancel,
// and fails the Grab by that cause. An engine error keeps download_failed.
func TestAReleaseFaultOrACancelFailsADownloadingGrabByItsCause(t *testing.T) {
	for _, c := range []struct {
		reason  downloadclients.RemoveReason
		fault   *string
		code    grabs.Code
		message string
	}{
		{downloadclients.ReleaseFaulted, new("unregistered"), grabs.ReleaseFault, "The tracker reports this Grab's torrent unregistered."},
		{downloadclients.ReleaseFaulted, new("no_payload"), grabs.ReleaseFault, "This Grab's torrent never downloaded any data."},
		{downloadclients.ReleaseFaulted, new("engine_error"), grabs.DownloadFailed, "The built-in client couldn't read this Grab's torrent."},
		{downloadclients.Canceled, nil, grabs.Canceled, "This Grab's torrent was removed when another Grab of it was canceled."},
	} {
		t.Run(c.message, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				p := newPipeline(t)
				id := downloadingGrab(t, p, seedingReport)
				now := storage.Now()
				run(t, p.lib, sql(`UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = ?, parted_at = ?, freed = 0,
					fault = ?, fault_at = CASE WHEN ? IS NULL THEN NULL ELSE ? END`, now, string(c.reason), now, c.fault, c.fault, now))

				p.running()
				time.Sleep(time.Second)

				v := p.view(id)
				if v.Stage != grabs.Failed || v.Failed.Code != c.code || v.Failed.Message != c.message {
					t.Fatalf("the Grab is %s with %+v, want failed with %s: %q", v.Stage, v.Failed, c.code, c.message)
				}
			})
		})
	}
}

var actions = map[downloadclients.ActionKind]struct {
	code   grabs.Code
	report string
}{
	downloadclients.Remove:     {grabs.Removed, seedingReport},
	downloadclients.MarkFailed: {grabs.DownloadFailed, stalledReport},
}

// actAsTheGrabSteps runs kind on a downloading Grab's Torrent just before
// the Grab's step at stage at, holding that step until the Grabs stop, and
// returns the Grab and the action's refusal.
func actAsTheGrabSteps(t *testing.T, kind downloadclients.ActionKind, report string, at grabs.Stage) (grabs.View, *downloadclients.Refusal) {
	t.Helper()
	p := newPipeline(t)
	id := downloadingGrab(t, p, report)
	torrent := torrentOf(t, p)
	clients := downloadclients.New(downloadclients.Config{
		Library: p.lib, Touch: func(...role.NodeID) {}, Changed: discard[downloadclients.Changed]{},
		Uses: grabs.TorrentUses{}, Log: slog.New(slog.DiscardHandler),
	})
	var refusal *downloadclients.Refusal
	p.beforeStep = func(ctx context.Context, s grabs.Stage) {
		if s != at {
			return
		}
		results, _, err := clients.Act(ctx, downloadclients.Action{Kind: kind, IDs: []downloadclients.ID{torrent}})
		if err != nil {
			t.Error(err)
			return
		}
		refusal = results[0].Refusal
		if at == grabs.Probing {
			<-ctx.Done()
		}
	}
	p.running()
	time.Sleep(time.Second)
	return p.view(id), refusal
}

// TestAnAdminRemovalBetweenAGrabsReadAndCommitFailsIt runs remove and
// mark-failed between a Grab's read of its complete Torrent and its commit
// to probing: the action lands and the commit fails the Grab at
// downloading.
func TestAnAdminRemovalBetweenAGrabsReadAndCommitFailsIt(t *testing.T) {
	for kind, k := range actions {
		t.Run(string(kind), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				v, refusal := actAsTheGrabSteps(t, kind, k.report, grabs.Downloading)
				if refusal != nil {
					t.Fatalf("%s between the Grab's read and its commit was refused %+v, want done", kind, refusal)
				}
				if v.Stage != grabs.Failed || v.Failed.Stage != grabs.Downloading || v.Failed.Code != k.code {
					t.Fatalf("the Grab is %s with %+v, want failed at downloading with %s", v.Stage, v.Failed, k.code)
				}
			})
		})
	}
}

// TestAnAdminRemovalOfAProbingGrabsTorrentIsRefused runs remove and
// mark-failed once the Grab probes: the action is refused as importing and
// leaves the Torrent and the Grab be.
func TestAnAdminRemovalOfAProbingGrabsTorrentIsRefused(t *testing.T) {
	for kind, k := range actions {
		t.Run(string(kind), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				v, refusal := actAsTheGrabSteps(t, kind, k.report, grabs.Probing)
				if refusal == nil || refusal.Code != downloadclients.RefusedImporting {
					t.Fatalf("%s of a probing Grab's Torrent answered %+v, want refused as importing", kind, refusal)
				}
				if v.Stage != grabs.Probing {
					t.Fatalf("the Grab is %s with %+v, want still probing", v.Stage, v.Failed)
				}
			})
		})
	}
}

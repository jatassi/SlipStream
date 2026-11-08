package grabs_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

var goneAtAnyStage = []struct {
	reason downloadclients.RemoveReason
	code   grabs.Code
	msg    string
}{
	{downloadclients.NodeWrittenOff, grabs.NodeWrittenOff, "The Admin wrote off what this Grab's Node downloads"},
	{downloadclients.ByClient, grabs.Removed, "Transmission removed the torrent before SlipStream imported it."},
	{downloadclients.ClientDeleted, grabs.Removed, "The Admin deleted Transmission before SlipStream imported this torrent."},
}

// inTransmission is a Grab at stage whose Torrent is in Transmission on
// home and complete.
func inTransmission(t *testing.T, stage grabs.Stage) *sending {
	t.Helper()
	s := newSending(t, seedingReport)
	now := storage.Now()
	run(t, s.lib,
		sql(`UPDATE downloadclients_torrent SET client_id = ?, attached = 0, completed_at = ?,
			completion = '{"files":[{"Index":0,"Path":"The.Attic.1080p.WEB.mkv","Size":4096}]}'`, s.transmission, now),
		sql(`UPDATE grabs_grab SET stage = ?, file_index = 0, probe = '{}', root_id = ?, path = 'The Attic.mkv'`, string(stage), s.root),
	)
	return s
}

// TestAGrabFailsAtAnyStageOnceItsTorrentIsGoneWithoutAnAdminsRemove parts
// a Grab's Torrent for each reason that no import can refuse, at each
// stage past fetching: the Grab fails at that stage with that reason's
// code, the two client reasons naming Transmission.
func TestAGrabFailsAtAnyStageOnceItsTorrentIsGoneWithoutAnAdminsRemove(t *testing.T) {
	for _, r := range goneAtAnyStage {
		for _, stage := range []grabs.Stage{grabs.Downloading, grabs.Probing, grabs.Transferring, grabs.Importing} {
			t.Run(string(r.reason)+"/"+string(stage), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					s := inTransmission(t, stage)
					now := storage.Now()
					run(t, s.lib, sql(`UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = ?, parted_at = ?`, now, string(r.reason), now))

					s.running()
					time.Sleep(time.Second)

					v := s.view(s.grab)
					if v.Stage != grabs.Failed || v.Failed.Stage != stage || v.Failed.Code != r.code || v.Failed.Message != r.msg {
						t.Fatalf("the Grab is %s with %+v, want failed at %s with %s: %q", v.Stage, v.Failed, stage, r.code, r.msg)
					}
				})
			})
		}
	}
}

// TestAnExternalTorrentsGoneSourceWaitsForItsVanishedReport fails the
// Transfer of a transferring Grab whose Torrent is in Transmission as its
// source gone: the Grab keeps transferring, since the Node sends the
// Vanished report first, and fails removed once that parts the Torrent.
func TestAnExternalTorrentsGoneSourceWaitsForItsVanishedReport(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := inTransmission(t, grabs.Transferring)
		var transfer transfers.ID
		run(t, s.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			var err error
			transfer, err = s.copies.Start(ctx, tx, transfers.Request{
				Purpose: transfers.Grab, Job: s.grab.String(), Source: s.node, Destination: transfers.Root{ID: s.root},
				Files: []transfers.FileRequest{{Name: "The Attic.mkv", Source: source.Torrent{Hash: s.hash}, Size: 4096}},
			})
			return err
		})
		run(t, s.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return s.copies.Apply(ctx, tx, s.node, &tv1.Result{Result: &tv1.Result_Failed{Failed: &tv1.Failed{
				TransferId: transfer.Bytes(), Code: tv1.Failed_CODE_SOURCE_GONE, Detail: "the torrent is gone",
			}}})
		})
		var stage string
		if err := s.lib.QueryRowContext(t.Context(), `SELECT stage FROM grabs_grab`).Scan(&stage); err != nil || stage != string(grabs.Transferring) {
			t.Fatalf("after its source went the Grab is %s (%v), want still transferring", stage, err)
		}

		now := storage.Now()
		run(t, s.lib, sql(`UPDATE downloadclients_torrent SET removed_at = ?, removed_reason = 'client', parted_at = ?`, now, now))
		s.running()
		time.Sleep(time.Second)
		v := s.view(s.grab)
		if v.Stage != grabs.Failed || v.Failed.Stage != grabs.Transferring || v.Failed.Code != grabs.Removed {
			t.Errorf("once the Vanished report applied the Grab is %s with %+v, want failed at transferring as removed", v.Stage, v.Failed)
		}
	})
}

package grabs_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/storage"
)

var goneAtAnyStage = []struct {
	reason downloadclients.RemoveReason
	code   grabs.Code
	msg    string
}{
	{downloadclients.NodeWrittenOff, grabs.NodeWrittenOff, "The Admin wrote off what this Grab's Node downloads"},
	{downloadclients.ByClient, grabs.ClientRemoved, "Transmission removed the torrent before SlipStream imported it."},
	{downloadclients.ClientDeleted, grabs.ClientDeleted, "The Admin deleted Transmission before SlipStream imported this torrent."},
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
		sql(`UPDATE grabs_grab SET stage = ?`, string(stage)),
	)
	return s
}

// TestAGrabFailsAtAnyStageOnceItsTorrentIsGoneWithoutAnAdminsRemove parts
// a Grab's Torrent for each reason that no import can refuse, downloading
// and importing with nothing placed: the Grab fails at that stage with
// that reason's code, the two client reasons naming Transmission.
func TestAGrabFailsAtAnyStageOnceItsTorrentIsGoneWithoutAnAdminsRemove(t *testing.T) {
	for _, r := range goneAtAnyStage {
		for _, stage := range []grabs.Stage{grabs.Downloading, grabs.Importing} {
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

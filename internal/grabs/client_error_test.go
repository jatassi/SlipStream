package grabs_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

const erroringReport = `{"at":"2026-01-01T00:00:00Z","engine":"downloading","hold":"client_error","error":"No data found! Ensure your drives are connected.","done":1024}`

// TestAClientErrorWaitsInNeedsYouThenFailsTheGrabWithNoBlocklistEntry has
// Transmission report an error on a downloading Grab's Torrent: the Grab
// waits client_error naming the client, its Node, the client's words and
// when it fails, then fails client_error once the clock passes the
// setting, and the Release stays off the Blocklist.
func TestAClientErrorWaitsInNeedsYouThenFailsTheGrabWithNoBlocklistEntry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSending(t, erroringReport)
		last := time.Now().Add(-time.Minute)
		target := storage.NewUUID()
		run(t, s.lib,
			sql(`UPDATE downloadclients_torrent SET client_id = ?, client_error_ms = ?, client_error_at = ?`, s.transmission, (59*time.Minute).Milliseconds(), storage.FromTime(last)),
			sql(`INSERT INTO targets_target (id, name, created_at) VALUES (?, 'HD', 0)`, target),
			sql(`INSERT INTO grabs_target (grab_id, position, item_id, target_id) SELECT id, 0, item_id, ? FROM grabs_grab`, target),
		)

		cell := s.cellOf(t)
		want := files.ClientError{Client: "Transmission", Node: "home", Message: "No data found! Ensure your drives are connected.", FailsAt: last.Add(time.Minute).UTC()}
		if cell.Waiting != grabs.ClientErrorWait || cell.ClientError == nil || *cell.ClientError != want {
			t.Fatalf("the Grab's cell reads %+v with %+v, want it waiting client_error on %+v", cell, cell.ClientError, want)
		}
		s.running()
		time.Sleep(time.Minute)
		if v := s.view(s.grab); v.Stage != grabs.Downloading {
			t.Fatalf("with 59 minutes of the error counted the Grab is %s with %+v, want still downloading", v.Stage, v.Failed)
		}

		run(t, s.lib, sql(`UPDATE downloadclients_torrent SET client_error_ms = ?`, time.Hour.Milliseconds()))
		time.Sleep(time.Minute)
		v := s.view(s.grab)
		if v.Stage != grabs.Failed || v.Failed.Code != grabs.ClientError || v.Failed.Message != "Transmission on home reported an error for more than 1 h. Search can try another Release." {
			t.Fatalf("with an hour of the error counted the Grab is %s with %+v, want failed client_error", v.Stage, v.Failed)
		}
		var blocked int
		if err := s.lib.QueryRowContext(t.Context(), `SELECT count(*) FROM grabs_blocklist`).Scan(&blocked); err != nil || blocked != 0 {
			t.Errorf("the Blocklist holds %d entries (%v), want none: a client error is no fault of the Release", blocked, err)
		}
	})
}

func (s *sending) cellOf(t *testing.T) files.GrabCell {
	t.Helper()
	var got map[files.CellKey]files.GrabCell
	err := s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = grabs.GrabCells{InStep: func(role.NodeID) bool { return true }}.GrabCells(ctx, r, works.Only([]works.WorkID{s.work}))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		if len(got) == 1 {
			return c
		}
	}
	t.Fatalf("the Work's cells read %+v, want the Grab's one", got)
	return files.GrabCell{}
}

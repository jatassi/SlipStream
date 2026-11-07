package downloadclients_test

import (
	"testing"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
)

// held is what Media server's intent holds of id: live, discarded and
// with its files deleted, or neither once parted.
func (s *mediaServer) held(t *testing.T, id dc.ID) (live, discarded, deleteFiles bool) {
	t.Helper()
	in := s.intent(t)
	for _, tr := range in.GetTorrents() {
		if idFrom(tr.GetId()) == id {
			live = true
		}
	}
	for _, d := range in.GetDiscards() {
		if idFrom(d.GetId()) == id {
			discarded, deleteFiles = true, d.GetDeleteFiles()
		}
	}
	return live, discarded, deleteFiles
}

func (s *mediaServer) originIs(t *testing.T, id dc.ID, o dlv1.Origin) {
	t.Helper()
	s.report(t, s.node, s.torrent(t, &dlv1.TorrentReport{Id: id.Bytes(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Origin: o}))
}

// TestRemovingAnAttachedTorrentOnlyStopsTrackingIt removes, with its
// files, a torrent that was in Transmission before SlipStream attached to
// it: it parts at once with no discard for the Node, so Transmission keeps
// it and its files, and its row never offered Delete files.
func TestRemovingAnAttachedTorrentOnlyStopsTrackingIt(t *testing.T) {
	for _, kind := range []dc.ActionKind{dc.Remove, dc.MarkFailed} {
		t.Run(string(kind), func(t *testing.T) {
			s := newMediaServer(t)
			id := s.send(t, "Orchard")
			s.report(t, s.node, s.torrent(t, &dlv1.TorrentReport{
				Id: id.Bytes(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Origin: dlv1.Origin_ORIGIN_ATTACHED, Stalled: true,
			}))
			if row := s.row(t, id); !row.Attached || row.DeleteFiles {
				t.Fatalf("the attached torrent's row is attached %v, delete files %v, want attached with no Delete files", row.Attached, row.DeleteFiles)
			}

			results, _ := act(t, s.c, dc.Action{Kind: kind, IDs: []dc.ID{id}, DeleteFiles: true})
			if got := refusals(results); got[0] != nil {
				t.Fatalf("%s answered %+v, want done", kind, got[0])
			}
			if live, discarded, _ := s.held(t, id); live || discarded {
				t.Errorf("after %s Media server's intent holds it live %v, discarded %v, want neither", kind, live, discarded)
			}
			if got := outcomeOf(t, s.lib, id).Outcome; got != (dc.Gone{Reason: map[dc.ActionKind]dc.RemoveReason{dc.Remove: dc.ByAdmin, dc.MarkFailed: dc.ByMarkFailed}[kind]}) {
				t.Errorf("the attached torrent reads %#v, want Gone by the Admin's %s", got, kind)
			}
		})
	}
}

// TestARemoveGoesByTheClientsReach removes a torrent SlipStream added to
// Transmission while it responds, while it isn't responding and while it
// is off: the first two wait in the Node's discards with their files to
// delete, the second until Transmission responds again, and the third
// parts at once, since its Node doesn't check an off client.
func TestARemoveGoesByTheClientsReach(t *testing.T) {
	for _, c := range []struct {
		name    string
		reach   func(*testing.T, *mediaServer)
		discard bool
	}{
		{"responding", func(t *testing.T, s *mediaServer) {
			s.report(t, s.node, s.poll(1, dlv1.PollState_POLL_STATE_RESPONDING, nil, nil))
		}, true},
		{"not responding", func(t *testing.T, s *mediaServer) {
			s.report(t, s.node, s.poll(1, dlv1.PollState_POLL_STATE_NOT_RESPONDING, &dlv1.Problem{Cause: dlv1.Cause_CAUSE_NO_RESPONSE}, nil))
		}, true},
		{"off", func(t *testing.T, s *mediaServer) {
			exec(t, s.lib, "UPDATE downloadclients_client SET is_on = 0 WHERE id = ?", s.client)
		}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := newMediaServer(t)
			id := s.send(t, "Orchard")
			s.originIs(t, id, dlv1.Origin_ORIGIN_ADDED)
			c.reach(t, s)
			if row := s.row(t, id); row.DeleteFiles != c.discard {
				t.Errorf("the row offers Delete files %v, want %v", row.DeleteFiles, c.discard)
			}

			act(t, s.c, dc.Action{Kind: dc.Remove, IDs: []dc.ID{id}, DeleteFiles: true})
			live, discarded, deleteFiles := s.held(t, id)
			if live || discarded != c.discard || deleteFiles != c.discard {
				t.Errorf("after the remove Media server's intent holds it live %v, discarded %v with files %v, want discarded %v",
					live, discarded, deleteFiles, c.discard)
			}
			if got := outcomeOf(t, s.lib, id).Outcome; got != (dc.Gone{Reason: dc.ByAdmin}) {
				t.Errorf("the removed torrent reads %#v, want Gone by the Admin", got)
			}
		})
	}
}

// TestAnnounceAndRecheckAreRefusedForAClientThatCantTakeThem asks for an
// announce while Transmission is off, while it isn't responding, and
// while its Node says it isn't: each refusal names Transmission and Media
// server, and the Node is asked only the last time.
func TestAnnounceAndRecheckAreRefusedForAClientThatCantTakeThem(t *testing.T) {
	s := newMediaServer(t)
	id := s.send(t, "Orchard")
	node := &client{refuse: map[dc.ID]*dlv1.ActResult{id: {Id: id.Bytes(), Refused: dlv1.Refusal_REFUSAL_CLIENT_AWAY}}}
	c, _ := newClients(s.lib, &roster{serving: []role.NodeID{s.node}}, nil, node)
	ask := func(kind dc.ActionKind) *dc.Refusal {
		t.Helper()
		results, _ := act(t, c, dc.Action{Kind: kind, IDs: []dc.ID{id}})
		return results[0].Refusal
	}

	exec(t, s.lib, "UPDATE downloadclients_client SET is_on = 0 WHERE id = ?", s.client)
	if got := ask(dc.Announce); got == nil || *got != (dc.Refusal{Code: dc.RefusedClientOff, Client: "Transmission", Node: "Media server"}) {
		t.Errorf("an announce for an off Transmission answered %+v, want client_off naming it and Media server", got)
	}
	exec(t, s.lib, "UPDATE downloadclients_client SET is_on = 1 WHERE id = ?", s.client)
	s.report(t, s.node, s.poll(1, dlv1.PollState_POLL_STATE_NOT_RESPONDING, &dlv1.Problem{Cause: dlv1.Cause_CAUSE_NO_RESPONSE}, nil))
	if got := ask(dc.Recheck); got == nil || *got != (dc.Refusal{Code: dc.RefusedClientAway, Client: "Transmission", Node: "Media server"}) {
		t.Errorf("a recheck while Transmission isn't responding answered %+v, want client_away naming it and Media server", got)
	}
	if len(node.asked) != 0 {
		t.Fatalf("the Node was asked %v, want nothing while its client can't take it", node.asked)
	}

	s.report(t, s.node, s.poll(1, dlv1.PollState_POLL_STATE_RESPONDING, nil, nil))
	if got := ask(dc.Announce); got == nil || *got != (dc.Refusal{Code: dc.RefusedClientAway, Client: "Transmission", Node: "Media server"}) {
		t.Errorf("an announce the Node refused as its client away answered %+v, want client_away naming it and Media server", got)
	}
}

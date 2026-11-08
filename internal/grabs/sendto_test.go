package grabs_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

const downloadingReport = `{"at":"2026-01-01T00:00:00Z","engine":"downloading","done":1024}`

// sending is a downloading Grab on home's built-in client, Transmission on
// home and Media server, another downloading Node.
type sending struct {
	*pipeline
	grab         grabs.GrabID
	hash         bencode.InfoHash
	torrent      downloadclients.ID
	transmission downloadclients.ClientID
	media        role.NodeID
}

// newSending starts the Grab's download, reported by home when report
// is set, else not yet added.
func newSending(t *testing.T, report string) *sending {
	t.Helper()
	p := newPipeline(t)
	s := &sending{pipeline: p, media: role.NodeID{UUID: storage.NewUUID()}}
	rel := newRelease(t)
	s.grab, s.hash = p.grab(rel.link.String(), &rel.link.InfoHash), rel.link.InfoHash
	raw, err := rel.link.Torrent(rel.info)
	if err != nil {
		t.Fatal(err)
	}
	attachTorrent(t, p, raw)
	s.torrent = torrentOf(t, p)
	address, err := clients.ParseAddress("http://192.168.1.20:9091", clients.Transmission)
	if err != nil {
		t.Fatal(err)
	}
	label, err := clients.ParseLabel(clients.DefaultLabel, clients.Transmission)
	if err != nil {
		t.Fatal(err)
	}
	sealer := secrettest.Sealer(t)
	run(t, p.lib,
		func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return nodestest.Insert(ctx, tx, s.media, "Media server")
		},
		func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			s.transmission, err = downloadclients.Import(ctx, tx, sealer, downloadclients.Imported{
				Node: p.node, Kind: clients.Transmission, Name: "Transmission", Address: address, Label: label,
			}, storage.Now())
			return err
		},
		sql(`UPDATE grabs_grab SET stage = 'downloading', torrent_id = ?`, s.torrent),
	)
	if report != "" {
		run(t, p.lib, sql(`UPDATE downloadclients_torrent SET report = ?, reported_at = ?`, report, storage.Now()))
	}
	return s
}

func (s *sending) sendTo(c downloadclients.Choice) error {
	return s.g.SendTo(s.t.Context(), identity.Admin{}, s.grab, c)
}

// intent is node's Downloading intent: what it holds and what it
// discards.
func (s *sending) intent(node role.NodeID) *dlv1.Intent {
	s.t.Helper()
	var in *dlv1.Intent
	err := s.lib.Read(s.t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		in, err = downloadclients.Intent(ctx, r, node)
		return err
	})
	if err != nil {
		s.t.Fatal(err)
	}
	return in
}

func idOf(t *testing.T, raw []byte) storage.UUID {
	t.Helper()
	id, err := storage.UUIDFromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// TestSendToBeforeTheAddRewritesTheTorrentsClientInPlace sends a Grab
// whose Torrent home hasn't reported to Transmission on home, then to
// Media server's built-in client: each time the same Torrent moves with
// no removal, and the Node it is on now holds it in the new client.
func TestSendToBeforeTheAddRewritesTheTorrentsClientInPlace(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSending(t, "")

		if err := s.sendTo(downloadclients.Choice{Client: &s.transmission}); err != nil {
			t.Fatal(err)
		}
		in := s.intent(s.node)
		if len(in.GetTorrents()) != 1 || len(in.GetDiscards()) != 0 || idOf(t, in.GetTorrents()[0].GetId()) != s.torrent.UUID ||
			idOf(t, in.GetTorrents()[0].GetClientId()) != s.transmission.UUID {
			t.Fatalf("after the send home's intent is %v, want the same Torrent in Transmission and nothing discarded", in)
		}
		v := s.view(s.grab)
		if v.Stage != grabs.Downloading || v.Torrent.ID != s.torrent || v.Torrent.Client == nil || v.Torrent.Client.Name != "Transmission" {
			t.Fatalf("after the send the Grab is %s on %+v, want downloading its Torrent in Transmission", v.Stage, v.Torrent)
		}

		if err := s.sendTo(downloadclients.Choice{BuiltInOn: s.media}); err != nil {
			t.Fatal(err)
		}
		home, media := s.intent(s.node), s.intent(s.media)
		if len(home.GetTorrents())+len(home.GetDiscards()) != 0 || len(media.GetTorrents()) != 1 ||
			idOf(t, media.GetTorrents()[0].GetId()) != s.torrent.UUID || media.GetTorrents()[0].GetClientId() != nil {
			t.Fatalf("after the send to Media server home's intent is %v and Media server's %v, want the same Torrent moved to its built-in client", home, media)
		}
		if v := s.view(s.grab); v.Stage != grabs.Downloading || v.Node != s.media || v.Torrent.ID != s.torrent {
			t.Errorf("the Grab is %s on %v with %v, want downloading the same Torrent on Media server", v.Stage, v.Node, v.Torrent.ID)
		}
	})
}

// TestSendToAfterTheAddOnTheSameNodeIsRefused sends a Grab whose Torrent
// home already reported to Transmission on home: home holds one live copy
// of a torrent, so it is refused naming home's built-in client, and
// nothing changes.
func TestSendToAfterTheAddOnTheSameNodeIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSending(t, downloadingReport)

		err := s.sendTo(downloadclients.Choice{Client: &s.transmission})
		if e, ok := errors.AsType[*downloadclients.InClientError](err); !ok || *e != (downloadclients.InClientError{Node: "home"}) {
			t.Fatalf("the send answered %v, want home's built-in client already has the torrent", err)
		}
		in := s.intent(s.node)
		if len(in.GetTorrents()) != 1 || in.GetTorrents()[0].GetClientId() != nil {
			t.Errorf("after the refusal home's intent is %v, want the Torrent still in its built-in client", in)
		}
	})
}

// TestSendToAnotherNodeAfterTheAddRemovesTheOldTorrent sends a Grab whose
// Torrent home reported to Media server: home's Torrent is removed as an
// Admin's remove does, its files kept, and the Grab downloads a new
// Torrent on Media server, without failing once the Grabs run.
func TestSendToAnotherNodeAfterTheAddRemovesTheOldTorrent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSending(t, downloadingReport)

		if err := s.sendTo(downloadclients.Choice{BuiltInOn: s.media}); err != nil {
			t.Fatal(err)
		}
		home, media := s.intent(s.node), s.intent(s.media)
		if len(home.GetTorrents()) != 0 || len(home.GetDiscards()) != 1 || idOf(t, home.GetDiscards()[0].GetId()) != s.torrent.UUID || home.GetDiscards()[0].GetDeleteFiles() {
			t.Fatalf("after the send home's intent is %v, want the old Torrent discarded with its files kept", home)
		}
		if len(media.GetTorrents()) != 1 {
			t.Fatalf("after the send Media server's intent is %v, want one new Torrent", media)
		}
		moved := idOf(t, media.GetTorrents()[0].GetId())

		s.running()
		time.Sleep(time.Second)
		v := s.view(s.grab)
		if v.Stage != grabs.Downloading || v.Node != s.media || v.Torrent.ID.UUID != moved {
			t.Errorf("the Grab is %s on %v with %v (%+v), want downloading Media server's new Torrent", v.Stage, v.Node, v.Torrent.ID, v.Failed)
		}
	})
}

// TestSendToIsRefusedForAFinishedDownloadOrAnOffClient refuses a Grab
// whose download is complete, and a send to Transmission once it is off.
func TestSendToIsRefusedForAFinishedDownloadOrAnOffClient(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSending(t, "")
		run(t, s.lib, sql(`UPDATE downloadclients_client SET is_on = 0`))
		err := s.sendTo(downloadclients.Choice{Client: &s.transmission})
		if e, ok := errors.AsType[*downloadclients.ClientOffError](err); !ok || e.Name != "Transmission" {
			t.Errorf("a send to an off Transmission answered %v, want it refused as off", err)
		}

		run(t, s.lib, sql(`UPDATE downloadclients_torrent SET report = ?, reported_at = ?, completed_at = ?, completion = '{"files":[]}'`,
			seedingReport, storage.Now(), storage.Now()))
		if err := s.sendTo(downloadclients.Choice{BuiltInOn: s.media}); !errors.Is(err, grabs.ErrNotDownloading) {
			t.Errorf("a send of a finished download answered %v, want ErrNotDownloading", err)
		}
	})
}

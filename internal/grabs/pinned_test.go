package grabs_test

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// offTransmission adds Transmission on home, turned off.
func (p *pipeline) offTransmission() downloadclients.ClientID {
	p.t.Helper()
	address, err := clients.ParseAddress("http://192.168.1.20:9091", clients.Transmission)
	if err != nil {
		p.t.Fatal(err)
	}
	label, err := clients.ParseLabel(clients.DefaultLabel, clients.Transmission)
	if err != nil {
		p.t.Fatal(err)
	}
	var id downloadclients.ClientID
	run(p.t, p.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		id, err = downloadclients.Import(ctx, tx, secrettest.Sealer(p.t), downloadclients.Imported{
			Node: p.node, Kind: clients.Transmission, Name: "Transmission", Address: address, Label: label,
		}, storage.Now())
		return err
	})
	run(p.t, p.lib, sql(`UPDATE downloadclients_client SET is_on = 0 WHERE id = ?`, id))
	return id
}

// sentTo is a fetching Grab of rel from source, sent to client, with a
// Target of its own, which it returns.
func (p *pipeline) sentTo(rel release, source grabs.Source, client downloadclients.ClientID) (grabs.GrabID, storage.UUID) {
	p.t.Helper()
	id := p.grab(rel.link.String(), &rel.link.InfoHash)
	target := storage.NewUUID()
	run(p.t, p.lib,
		sql(`UPDATE grabs_grab SET source = ?, sent_client_id = ? WHERE id = ?`, string(source), client, id),
		sql(`INSERT INTO targets_target (id, name, created_at) VALUES (?, ?, 0)`, target, target.String()),
		sql(`INSERT INTO grabs_target (grab_id, position, item_id, target_id) SELECT id, 0, item_id, ? FROM grabs_grab WHERE id = ?`, target, id),
	)
	return id, target
}

func (p *pipeline) cells() grabs.GrabCells {
	return grabs.GrabCells{InStep: p.roster.InStep, Router: p.cfg.Router}
}

// TestAManualGrabPinnedToAnOffClientWaitsForItUntilItIsTurnedOn sends a
// Manual Grab and a search's Grab to Transmission while it is off, with
// home's built-in client free to take either. The search's Grab falls
// through to the built-in client as S29 (Trackers and seeding goals)
// routes it. The Manual one waits on Transmission, client_off, and no Node
// is asked for it, until the Admin turns Transmission on and it downloads
// there.
func TestAManualGrabPinnedToAnOffClientWaitsForItUntilItIsTurnedOn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		automatic, manual := releaseNamed(t, "Automatic.Release"), releaseNamed(t, "Manual.Release")
		p := newPipeline(t, answer{res: downloadclients.Resolved{Info: automatic.info}}, answer{res: downloadclients.Resolved{Info: manual.info}})
		transmission := p.offTransmission()
		auto, _ := p.sentTo(automatic, grabs.FromSearch, transmission)
		pinned, target := p.sentTo(manual, grabs.FromManual, transmission)
		p.running()

		time.Sleep(10 * hold)
		if v := p.view(auto); v.Stage != grabs.Downloading || v.Node != p.node || v.Torrent == nil || v.Torrent.Client != nil {
			t.Fatalf("the search's Grab sent to Transmission while off is %s with %+v (%+v), want downloading on home's built-in client", v.Stage, v.Torrent, v.Failed)
		}
		v := p.view(pinned)
		if v.Stage != grabs.Fetching || v.Torrent != nil || v.WaitsFor == nil || !v.WaitsFor.Off || v.WaitsFor.Node != p.node ||
			v.WaitsFor.Client == nil || v.WaitsFor.Client.ID != transmission || v.WaitsFor.Client.Name != "Transmission" || !v.Actions.SendTo {
			t.Fatalf("the Manual Grab pinned to Transmission while off is %s with %+v waiting for %+v (%+v), want fetching, waiting for Transmission off and sendable", v.Stage, v.Torrent, v.WaitsFor, v.Actions)
		}
		if w := p.waitingByTarget(p.cells())[target]; w != grabs.ClientOff {
			t.Fatalf("the Manual Grab's cell waits on %q, want %q", w, grabs.ClientOff)
		}
		if calls := p.swarm.asked(); len(calls) != 1 || calls[0].link.InfoHash != automatic.link.InfoHash {
			t.Fatalf("the Nodes were asked for %+v, want only the search's release", calls)
		}

		if _, err := p.swarm.clients.EditClient(t.Context(), transmission, downloadclients.Edit{On: new(true)}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * hold)
		v = p.view(pinned)
		if v.Stage != grabs.Downloading || v.Torrent == nil || v.Torrent.InfoHash != manual.link.InfoHash || v.Torrent.Client == nil || v.Torrent.Client.ID != transmission || v.WaitsFor != nil {
			t.Fatalf("once Transmission is on the Manual Grab is %s with %+v waiting for %+v, want downloading its release in Transmission", v.Stage, v.Torrent, v.WaitsFor)
		}
		if w := p.waitingByTarget(p.cells())[target]; w != "" {
			t.Fatalf("once Transmission is on the Manual Grab's cell waits on %q, want nothing", w)
		}
	})
}

// TestSendToFreesAManualGrabPinnedToAnOffClient sends a Manual Grab
// waiting on Transmission while it is off to home's built-in client, and
// it fetches and downloads there at once.
func TestSendToFreesAManualGrabPinnedToAnOffClient(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rel := releaseNamed(t, "Manual.Release")
		p := newPipeline(t, answer{res: downloadclients.Resolved{Info: rel.info}})
		pinned, _ := p.sentTo(rel, grabs.FromManual, p.offTransmission())
		p.running()

		time.Sleep(10 * hold)
		if v := p.view(pinned); v.Stage != grabs.Fetching || v.WaitsFor == nil || !v.WaitsFor.Off {
			t.Fatalf("the Manual Grab pinned to Transmission while off is %s waiting for %+v, want fetching and waiting for it", v.Stage, v.WaitsFor)
		}
		if err := p.g.SendTo(t.Context(), identity.Admin{}, pinned, downloadclients.Choice{BuiltInOn: p.node}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * hold)
		if v := p.view(pinned); v.Stage != grabs.Downloading || v.Node != p.node || v.Torrent == nil || v.Torrent.InfoHash != rel.link.InfoHash || v.Torrent.Client != nil || v.WaitsFor != nil {
			t.Fatalf("once sent the Manual Grab is %s on %v with %+v waiting for %+v, want downloading its release on home's built-in client", v.Stage, v.Node, v.Torrent, v.WaitsFor)
		}
	})
}

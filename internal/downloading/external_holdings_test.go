package downloading_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	faketransmission "github.com/jatassi/slipstream/fakes/transmission"
	"github.com/jatassi/slipstream/internal/bencode"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// addOutside adds the release to the fake Transmission under label, as
// someone other than SlipStream would.
func (c *composite) addOutside(t *testing.T, name, label string) {
	t.Helper()
	l, err := clients.ParseLabel(label, clients.Transmission)
	if err != nil {
		t.Fatal(err)
	}
	rel := c.rels[name]
	if err := c.session(t).Add(t.Context(), clients.Add{Metainfo: rel.metainfo, Hash: rel.hash, Label: l}); err != nil {
		t.Fatal(err)
	}
}

// TestAnAttachedTorrentOutsideTheLabelIsReportedByLookup has the Node add
// orchard under SlipStream's label, attach lantern, which someone added
// under their own label, and see harbor, added the same way but recorded
// nowhere. The Node's Holdings list orchard by the label and lantern by
// the lookup of its hash, both on the fake Transmission, and never harbor.
func TestAnAttachedTorrentOutsideTheLabelIsReportedByLookup(t *testing.T) {
	inEachTransmission(t, func(t *testing.T, version string) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		c := n.startComposite(version, 1000)
		c.addOutside(t, "lantern", "theirs")
		c.addOutside(t, "harbor", "theirs")
		lantern := c.rels["lantern"]
		n.ctl.answer(lantern.id, &dlv1.TorrentResponse{Metainfo: lantern.metainfo, ContentType: "Movies", Origin: dlv1.Origin_ORIGIN_ATTACHED})
		attached := &dlv1.Torrent{Id: lantern.id.Bytes(), InfoHash: lantern.hash[:], Wanted: dlv1.Wanted_WANTED_STARTED, Order: 2, ClientId: c.client.GetId()}
		c.send(true, c.orchard(1), attached)
		time.Sleep(30 * time.Second)
		synctest.Wait()

		s := live(c.orchard(1), attached)
		s.Intent.Clients = []*dlv1.Client{c.client}
		s.Intent.Holdings = &dlv1.HoldingsRequest{Round: 1, PageSize: 1000}
		c.h.intents <- s
		synctest.Wait()

		pages := c.h.holdings()
		if len(pages) != 1 {
			t.Fatalf("the Node sent %d Holdings pages, want 1", len(pages))
		}
		got := map[bencode.InfoHash]*dlv1.Holding{}
		for _, e := range pages[0].GetEntries() {
			got[bencode.InfoHash(e.GetInfoHash())] = e
		}
		if _, ok := got[c.rels["harbor"].hash]; ok {
			t.Error("the Holdings list harbor, which the Node never recorded and which lies outside the label")
		}
		for _, name := range []string{"orchard", "lantern"} {
			e, ok := got[c.rels[name].hash]
			switch {
			case !ok:
				t.Errorf("the Holdings don't list %s", name)
			case !slices.Equal(e.GetClientId(), c.client.GetId()):
				t.Errorf("%s is listed on client %x, want the fake Transmission %x", name, e.GetClientId(), c.client.GetId())
			case e.GetContent().GetName() != name+".mkv" || e.GetContent().GetSize() != 1000 || !slices.Equal(e.GetHosts(), []string{"127.0.0.1"}):
				t.Errorf("%s is listed as %v, want %s.mkv of 1000 bytes at 127.0.0.1", name, e, name)
			}
		}
		if len(got) != 2 {
			t.Errorf("the Holdings list %d torrents, want orchard and lantern alone", len(got))
		}
	})
}

// TestOtherKindsAnswerMetainfoUnimplemented asks the Node for the metainfo
// of a torrent in the fake Transmission: Unimplemented, since only
// qBittorrent exports.
func TestOtherKindsAnswerMetainfoUnimplemented(t *testing.T) {
	inEachTransmission(t, func(t *testing.T, version string) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		c := n.startComposite(version, 1000)
		c.send(true, c.orchard(1))
		time.Sleep(30 * time.Second)
		synctest.Wait()

		orchard := c.rels["orchard"].hash
		_, err := c.Metainfo(t.Context(), &dlv1.MetainfoRequest{InfoHash: orchard[:], ClientId: c.client.GetId()})
		if rpc.CodeOf(err) != rpc.CodeUnimplemented {
			t.Errorf("Metainfo from the fake Transmission = %v, want Unimplemented", err)
		}
	})
}

// TestARoundAClientCantAnswerGoesOnceItDoes asks for a round while the
// fake Transmission resets every connection: the Node sends nothing, then
// sends the whole round once the client answers again.
func TestARoundAClientCantAnswerGoesOnceItDoes(t *testing.T) {
	inEachTransmission(t, func(t *testing.T, version string) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		c := n.startComposite(version, 1000)
		c.send(true, c.orchard(1))
		time.Sleep(30 * time.Second)
		synctest.Wait()
		if _, err := c.served.Faults.Set(fault.Rule{ID: "down", Target: fault.Target(faketransmission.Fake.Name), Fault: fault.Reset{}}); err != nil {
			t.Fatal(err)
		}

		s := live(c.orchard(1))
		s.Intent.Clients = []*dlv1.Client{c.client}
		s.Intent.Holdings = &dlv1.HoldingsRequest{Round: 1, PageSize: 1000}
		c.h.intents <- s
		synctest.Wait()
		if pages := c.h.holdings(); len(pages) != 0 {
			t.Fatalf("the Node sent %v while the client was down, want nothing", pages)
		}

		c.served.Faults.Clear("down")
		time.Sleep(timings.ClientRetryMax)
		synctest.Wait()
		if pages := c.h.holdings(); len(pages) != 1 || len(pages[0].GetEntries()) != 1 {
			t.Errorf("once the client answered the Node sent %v, want round 1 listing orchard", pages)
		}
	})
}

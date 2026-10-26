package downloading_test

import (
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

// withDHT is live(ts...) with the Node's DHT switch set to on.
func withDHT(on bool, ts ...*dlv1.Torrent) role.Snapshot[*dlv1.Intent] {
	s := live(ts...)
	s.Intent.Settings = proto.CloneOf(settings)
	s.Intent.Settings.Dht = on
	return s
}

func TestOnlyAPublicTorrentDiscoversAndTheDHTRunsOnlyWhileOneIsHeld(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := newNode(t, enginetest.TrackerRules{Interval: 30 * time.Minute})
		private, public := newPrivateFile(t, "Hollow.mkv"), newFile(t, "Meadow.mkv")
		n.world.Payload(private.hash, private.payload)
		n.world.Payload(public.hash, public.payload)
		privateID, publicID := n.hold(private), n.hold(public)
		both := []*dlv1.Torrent{private.started(privateID, 1), public.started(publicID, 2)}

		c := n.start()
		c.send(withDHT(true, both[0]))
		if c.engine.running().DHT {
			t.Error("the DHT runs while the client holds only a private torrent")
		}
		c.send(withDHT(true, both...))
		if !c.engine.running().DHT || n.world.Lookups(public.hash) == 0 {
			t.Errorf("with a public torrent held the DHT runs = %v and looked it up %d times, want it running and looking", c.engine.running().DHT, n.world.Lookups(public.hash))
		}
		c.send(withDHT(false, both...))
		if c.engine.running().DHT {
			t.Error("the DHT runs with the Node's switch off")
		}
		looked := n.world.Lookups(public.hash)
		c.send(withDHT(true, both...))
		if got := n.world.Lookups(public.hash); got <= looked {
			t.Errorf("turning the switch on looked the public torrent up %d times more, want again", got-looked)
		}
		if got, want := c.engine.discovers(), []bool{false, true}; !slices.Equal(got, want) {
			t.Errorf("the private and the public torrent were added with Discover %v, want %v", got, want)
		}

		c.stop()
		time.Sleep(time.Hour)
		looked = n.world.Lookups(public.hash)
		c = n.start()
		c.send(withDHT(true, both...))
		time.Sleep(timings.StartSpread)
		synctest.Wait()
		if got, want := c.engine.discovers(), []bool{false, true}; !slices.Equal(got, want) {
			t.Errorf("after a restart the torrents were added back from resume data with Discover %v, want %v", got, want)
		}
		if got := n.world.Lookups(public.hash); got <= looked {
			t.Error("the public torrent added back from resume data didn't discover again")
		}
	})
}

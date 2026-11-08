package rtorrent_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	fakertorrent "github.com/jatassi/slipstream/fakes/rtorrent"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

func TestTheFloorIsTheOldestVersionTheFakePlays(t *testing.T) {
	floor := clients.RTorrent.Facts().Floor
	for i, v := range fakertorrent.Versions {
		got, err := clients.ParseVersion(v)
		if err != nil {
			t.Fatal(err)
		}
		if (i == 0) != (got == floor) || got.Less(floor) {
			t.Errorf("fake-rtorrent plays %s, the floor is %s: want the floor first and nothing older", v, floor)
		}
	}
}

func TestTheFactsPathAndActionsAreRTorrents(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version})
		a, err := clients.ParseAddress("http://"+r.served.Address.AddrPort.String(), clients.RTorrent)
		if err != nil {
			t.Fatal(err)
		}
		if a.String() != r.served.Address.URL("")+"/RPC2" {
			t.Fatalf("the facts' path makes %s", a)
		}
		c := r.session()
		if _, err := c.Test(t.Context(), slipstream); err != nil {
			t.Fatalf("Test at the facts' path: %v", err)
		}
		r.add(c, r.releases[0], "")
		for _, act := range clients.RTorrent.Facts().Acts {
			if err := c.Act(t.Context(), act, []bencode.InfoHash{r.releases[0].hash}); err != nil {
				t.Errorf("Act(%d), which the facts list: %v", act, err)
			}
		}
	})
}

func TestTheFactsSayTheNodeDeletesRTorrentsData(t *testing.T) {
	if clients.RTorrent.Facts().DeletesData || clients.RTorrent.Facts().Lockout || !clients.RTorrent.Facts().NoMovingState {
		t.Fatalf("rTorrent's facts = %+v, want no data deleted, no lockout and no moving state", clients.RTorrent.Facts())
	}
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, fakertorrent.Client{Version: version})
		c := r.session()
		orchard := r.releases[0]
		r.add(c, orchard, "")
		time.Sleep(time.Minute)
		r.status(c, orchard)
		if err := c.Remove(t.Context(), orchard.hash, true); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(r.client.Dir, "orchard.mkv")); err != nil {
			t.Fatalf("a remove with data on a kind that deletes none, its file: %v", err)
		}
	})
}

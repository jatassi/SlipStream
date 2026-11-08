package deluge_test

import (
	"testing"
	"testing/synctest"

	fakedeluge "github.com/jatassi/slipstream/fakes/deluge"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

func TestTheFloorIsTheOldestVersionTheFakePlays(t *testing.T) {
	floor := clients.Deluge.Facts().Floor
	for i, v := range fakedeluge.Versions {
		got, err := clients.ParseVersion(v)
		if err != nil {
			t.Fatal(err)
		}
		if (i == 0) != (got == floor) || got.Less(floor) {
			t.Errorf("fake-deluge plays %s, the floor is %s: want the floor first and nothing older", v, floor)
		}
	}
}

func TestTheFactsPathAndActionsAreDeluges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, fakedeluge.Client{})
		a, err := clients.ParseAddress("http://"+r.served.Address.AddrPort.String(), clients.Deluge)
		if err != nil {
			t.Fatal(err)
		}
		if a.String() != r.served.Address.URL("")+"/json" {
			t.Fatalf("the facts' path makes %s", a)
		}
		c := r.session()
		if _, err := c.Test(t.Context(), slipstream); err != nil {
			t.Fatalf("Test at the facts' path: %v", err)
		}
		r.add(c, r.releases[0], "")
		for _, act := range clients.Deluge.Facts().Acts {
			if err := c.Act(t.Context(), act, []bencode.InfoHash{r.releases[0].hash}); err != nil {
				t.Errorf("Act(%d), which the facts list: %v", act, err)
			}
		}
	})
}

package transmission_test

import (
	"testing"

	faketransmission "github.com/jatassi/slipstream/fakes/transmission"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

func TestTheFloorIsTheOldestVersionTheFakePlays(t *testing.T) {
	floor := clients.Transmission.Facts().Floor
	for i, v := range faketransmission.Versions {
		got, err := clients.ParseVersion(v)
		if err != nil {
			t.Fatal(err)
		}
		if (i == 0) != (got == floor) || got.Less(floor) {
			t.Errorf("fake-transmission plays %s, the floor is %s: want the floor first and nothing older", v, floor)
		}
	}
}

func TestTheFactsPathAndActionsAreTransmissions(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, faketransmission.Daemon{Version: version})
		c := r.open("http://"+r.served.Address.AddrPort.String(), "", "")
		if _, err := c.Test(t.Context(), slipstream); err != nil {
			t.Fatalf("Test at the facts' path: %v", err)
		}
		r.add(c, r.releases[0], "")
		for _, a := range clients.Transmission.Facts().Acts {
			if err := c.Act(t.Context(), a, []bencode.InfoHash{r.releases[0].hash}); err != nil {
				t.Errorf("Act(%d), which the facts list: %v", a, err)
			}
		}
	})
}

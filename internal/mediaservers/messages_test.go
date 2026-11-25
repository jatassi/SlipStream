package mediaservers

import (
	"flag"
	"testing"

	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mail/mailtest"
)

var update = flag.Bool("update", false, "rewrite the golden emails under testdata/mail")

func TestEveryEmailMatchesItsGolden(t *testing.T) {
	ada := mail.Address{Name: "Ada Okafor", Email: "ada@example.com"}
	grace := mail.Address{Name: "Grace Hopper", Email: "grace@example.com"}
	render := func(m mail.Message, err error) mail.Message {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	canWatch := func(accepted, plexPassLine bool) mail.Message {
		return render(canWatchOnPlex.Render(grace, canWatchData{
			Server: "Den", PlexUsername: "gracehopper", Accepted: accepted, PlexPassLine: plexPassLine,
		}))
	}
	cases := []struct {
		golden, subject string
		to              mail.Address
		m               mail.Message
	}{
		{"plex-account-to-add", "Grace Hopper wants to watch on Den", ada, render(plexAccountToAdd.Render(ada, toAddData{
			Member: "Grace Hopper", PlexUsername: "gracehopper", PlexEmail: "grace@plex.example", Server: "Den",
			AccountsURL: "https://slipstream.example.com/accounts",
		}))},
		{"can-watch-accepted", "You can watch Den on Plex", grace, canWatch(true, false)},
		{"can-watch-shared", "You can watch Den on Plex", grace, canWatch(false, false)},
		{"can-watch-accepted-plex-pass", "You can watch Den on Plex", grace, canWatch(true, true)},
		{"can-watch-shared-plex-pass", "You can watch Den on Plex", grace, canWatch(false, true)},
	}
	for _, c := range cases {
		t.Run(c.golden, func(t *testing.T) {
			if c.m.Subject != c.subject || c.m.To != c.to {
				t.Fatalf("to %+v subject %q, want %+v %q", c.m.To, c.m.Subject, c.to, c.subject)
			}
			mailtest.Golden(t, c.m, "testdata/mail", c.golden, *update)
		})
	}
}

package setup

import (
	"net/netip"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node"
)

func TestConfigWritesOnlyWhatTheWorldSets(t *testing.T) {
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.Nodes = append(s.Nodes, scenario.Node{Name: "work"})
	got, err := Config(s, 1, []string{"torrent.listen", "http.listen", "link.listen"}, netip.MustParseAddrPort("127.0.0.1:40123"), node.Defaults(contenttype.Set{}, install.Dev).Endpoints)
	if err != nil {
		t.Fatal(err)
	}
	want := `seed: single-machine/work
name: work
torrent:
  listen: 127.0.0.3:0
  outgoing_utp: false
http:
  listen: 127.0.0.3:0
link:
  listen: 127.0.0.3:0
  advertise:
    - 127.0.0.1:40123
timings:
  node:
    stop_deadline: 5s
`
	if string(got) != want {
		t.Fatalf("config.yaml =\n%s\nwant\n%s", got, want)
	}
}

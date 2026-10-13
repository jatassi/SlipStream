package setup

import (
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

func TestConfigWritesOnlyWhatTheWorldSets(t *testing.T) {
	s, err := scenario.Load("single-machine")
	if err != nil {
		t.Fatal(err)
	}
	s.Nodes = append(s.Nodes, scenario.Node{Name: "work"})
	got, err := Config(s, 1, []string{"torrent.listen", "http.listen", "link.listen"}, node.Defaults().Endpoints)
	if err != nil {
		t.Fatal(err)
	}
	want := `seed: single-machine/work
name: work
torrent:
  listen: 127.0.0.3:0
http:
  listen: 127.0.0.3:0
link:
  listen: 127.0.0.3:0
timings:
  node:
    stop_deadline: 5s
`
	if string(got) != want {
		t.Fatalf("config.yaml =\n%s\nwant\n%s", got, want)
	}
}

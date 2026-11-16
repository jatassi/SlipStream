package nodes_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
)

func TestADrainSaysWhichTorrentsItsSeedingGoalsFinish(t *testing.T) {
	torrent := nodes.Noun{One: "torrent", Many: "torrents"}
	root := nodes.Noun{One: "root", Many: "roots"}
	for _, c := range []struct {
		role role.Name
		held nodes.Held
		want string
	}{
		{role.Downloading, nodes.Held{Units: 12, Noun: torrent, Bytes: 41 << 30},
			"seedbox takes no new Downloading work and finishes the 12 torrents (41 GB) it has."},
		{role.Downloading, nodes.Held{Units: 12, Noun: torrent, Bytes: 41 << 30, EndlessUnits: 12},
			"seedbox takes no new Downloading work. 12 torrents have no seeding goal and stay until you remove, move or give them up."},
		{role.Downloading, nodes.Held{Units: 12, Noun: torrent, Bytes: 41 << 30, EndlessUnits: 4},
			"seedbox takes no new Downloading work. It finishes 8 torrents once they reach their seeding goals. 4 torrents have no seeding goal and stay until you remove, move or give them up."},
		{role.Downloading, nodes.Held{Units: 1, Noun: torrent, EndlessUnits: 1},
			"seedbox takes no new Downloading work. 1 torrent has no seeding goal and stays until you remove, move or give it up."},
		{role.Downloading, nodes.Held{Units: 2, Noun: torrent, EndlessUnits: 1},
			"seedbox takes no new Downloading work. It finishes 1 torrent once it reaches its seeding goal. 1 torrent has no seeding goal and stays until you remove, move or give it up."},
		{role.Storage, nodes.Held{Units: 2, Noun: root, EndlessUnits: 2},
			"seedbox takes no new Storage work. It keeps its 2 roots until you move them or give them up."},
	} {
		got := nodes.Drains{Node: seedbox, Role: c.role, Held: c.held}.Words()
		if got != c.want {
			t.Errorf("%s holding %+v:\n got %q\nwant %q", c.role, c.held, got, c.want)
		}
	}
}

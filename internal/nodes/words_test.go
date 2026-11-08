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
			"seedbox takes no new Downloading work and finishes the 12 torrents (41 GB) it holds."},
		{role.Downloading, nodes.Held{Units: 12, Noun: torrent, Bytes: 41 << 30, EndlessUnits: 12},
			"seedbox takes no new Downloading work. It keeps 12 torrents that have no seeding end until you remove, move or write them off."},
		{role.Downloading, nodes.Held{Units: 12, Noun: torrent, Bytes: 41 << 30, EndlessUnits: 4},
			"seedbox takes no new Downloading work. It finishes 8 torrents as their seeding goals are met, and keeps 4 that have no seeding end until you remove, move or write them off."},
		{role.Downloading, nodes.Held{Units: 1, Noun: torrent, EndlessUnits: 1},
			"seedbox takes no new Downloading work. It keeps 1 torrent that has no seeding end until you remove, move or write it off."},
		{role.Downloading, nodes.Held{Units: 2, Noun: torrent, EndlessUnits: 1},
			"seedbox takes no new Downloading work. It finishes 1 torrent as its seeding goal is met, and keeps 1 that has no seeding end until you remove, move or write it off."},
		{role.Storage, nodes.Held{Units: 2, Noun: root, EndlessUnits: 2},
			"seedbox takes no new Storage work. It keeps its 2 roots until you move or write them off."},
	} {
		got := nodes.Drains{Node: seedbox, Role: c.role, Held: c.held}.Words()
		if got != c.want {
			t.Errorf("%s holding %+v:\n got %q\nwant %q", c.role, c.held, got, c.want)
		}
	}
}

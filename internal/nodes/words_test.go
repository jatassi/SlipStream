package nodes_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
)

// TestEveryEffectNamesItsFacts renders each kind of Effect, and a drain for
// each mix of torrents with and without a seeding goal: its words name the
// Nodes, the Role and the counts it carries.
func TestEveryEffectNamesItsFacts(t *testing.T) {
	torrent := nodes.Noun{One: "torrent", Many: "torrents"}
	root := nodes.Noun{One: "root", Many: "roots"}
	drains := func(r role.Name, h nodes.Held) nodes.Effect { return nodes.Drains{Node: seedbox, Role: r, Held: h} }
	for _, c := range []struct {
		effect nodes.Effect
		facts  []string
	}{
		{nodes.Gains{Node: seedbox, Role: role.Relaying}, []string{"seedbox", "Relaying"}},
		{nodes.LetsGo{Node: seedbox, Role: role.Subtitles}, []string{"seedbox", "Subtitles"}},
		{drains(role.Downloading, nodes.Held{Units: 12, Noun: torrent, Bytes: 41 << 30}), []string{"seedbox", "Downloading", "12 torrents (41 GB)"}},
		{drains(role.Downloading, nodes.Held{Units: 12, Noun: torrent, Bytes: 41 << 30, EndlessUnits: 12}), []string{"12 torrents"}},
		{drains(role.Downloading, nodes.Held{Units: 12, Noun: torrent, Bytes: 41 << 30, EndlessUnits: 4}), []string{"8 torrents", "4 torrents"}},
		{drains(role.Storage, nodes.Held{Units: 2, Noun: root, EndlessUnits: 2}), []string{"Storage", "2 roots"}},
		{nodes.HandedBack{Role: role.Downloading, From: []nodes.Named{seedbox}, To: home}, []string{"home", "Downloading"}},
		{nodes.Pinned{Node: home, Role: role.Serving}, []string{"home", "Serving"}},
		{nodes.Moves{Role: role.Searching, From: seedbox, To: home, Held: indexers}, []string{"3 Indexers", "home"}},
		{nodes.WritesOff{Node: seedbox, Role: role.Downloading, Held: torrents}, []string{"seedbox", "Downloading", "3 torrents (41 GB)"}},
		{nodes.Leaves{Node: seedbox}, []string{"seedbox"}},
	} {
		wantFacts(t, c.effect.Words(), c.facts...)
	}
}

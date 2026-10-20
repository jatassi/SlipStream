package world

import (
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/fakes/world/internal/setup"
	"github.com/jatassi/slipstream/internal/node"
)

type divergence struct {
	id          string
	keyPatterns []string
	env         []string
	// settings are database setting keys the world sets through the API.
	settings []string
	reason   string
	spec     string
}

var divergences = []divergence{
	{
		id:          "loopback-listen",
		keyPatterns: []string{"*.listen"},
		reason:      "Every listener binds its Node's own 127.0.0.N on port 0, the torrent client's on a port the world picked, since port 0 makes it draw from the seed concurrent runs share.",
		spec:        "#130",
	},
	{
		id:          "seed",
		keyPatterns: []string{"seed"},
		reason:      "Each Node's seed is <scenario seed>/<node>, so runs replay. Production draws a random one.",
		spec:        "#127",
	},
	{
		id:          "node-name",
		keyPatterns: []string{"name"},
		reason:      "Each Node is named as in its Scenario, since every Node of a run shares the host name.",
		spec:        "#132",
	},
	{
		id:          "timings",
		keyPatterns: []string{"timings.*"},
		reason:      "Protocol durations are shortened for the wall-clock tier (boundary 42).",
		spec:        "#130",
	},
	{
		id:     "test-ca",
		env:    []string{"SSL_CERT_FILE"},
		reason: "Nodes trust the run's CA, so fakes serve real TLS with no product trust setting.",
		spec:   "#130",
	},
	{
		id:          "link-front",
		keyPatterns: []string{"link.advertise"},
		reason:      "Each Node advertises its link front, so the Node link crosses a front the fault rules play on.",
		spec:        "#135",
	},
	{
		id:       "tcp-peers",
		settings: setup.TCPPeers,
		reason:   "Nodes dial and accept peers over TCP only, since seeder fronts carry no uTP and a uTP attempt at one costs a leecher about 4 s before it falls back.",
		spec:     "#135",
	},
	{
		id:          "fake-endpoints",
		keyPatterns: []string{"endpoints.*"},
		reason:      "Each boundary's endpoints point at its fake.",
		spec:        "#133",
	},
}

type setting struct {
	name     string
	env      bool
	database bool
}

func settings(overrides []node.Key, worldEnv []string) []setting {
	var out []setting
	for _, k := range overrides {
		out = append(out, setting{name: k.Key})
	}
	for _, kv := range worldEnv {
		name, _, _ := strings.Cut(kv, "=")
		out = append(out, setting{name: name, env: true})
	}
	return out
}

func databaseSettings(keys []string) []setting {
	out := make([]setting, len(keys))
	for i, k := range keys {
		out[i] = setting{name: k, database: true}
	}
	return out
}

func (d divergence) covers(s setting) bool {
	switch {
	case s.env:
		return slices.Contains(d.env, s.name)
	case s.database:
		return slices.Contains(d.settings, s.name)
	}
	return slices.ContainsFunc(d.keyPatterns, func(pattern string) bool {
		ok, _ := path.Match(pattern, s.name)
		return ok
	})
}

func diverge(settings []setting) (used map[string]bool, uncovered []string) {
	used = map[string]bool{}
	for _, s := range settings {
		i := slices.IndexFunc(divergences, func(d divergence) bool { return d.covers(s) })
		if i < 0 {
			uncovered = append(uncovered, s.name)
			continue
		}
		used[divergences[i].id] = true
	}
	return used, uncovered
}

func usedIDs(used map[string]bool) []string {
	ids := []string{}
	for _, d := range divergences {
		if used[d.id] {
			ids = append(ids, d.id)
		}
	}
	return ids
}

package world

import (
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/node"
)

type divergence struct {
	id          string
	keyPatterns []string
	env         []string
	reason      string
	spec        string
}

var divergences = []divergence{
	{
		id:          "loopback-listen",
		keyPatterns: []string{"*.listen"},
		reason:      "Every listener binds its Node's own 127.0.0.N on port 0.",
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
		id:          "fake-endpoints",
		keyPatterns: []string{"endpoints.*"},
		reason:      "Each boundary's endpoints point at its fake.",
		spec:        "#133",
	},
}

type setting struct {
	name string
	env  bool
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

func (d divergence) covers(s setting) bool {
	if s.env {
		return slices.Contains(d.env, s.name)
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

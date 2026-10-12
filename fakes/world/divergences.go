package world

import (
	"path"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/node"
)

// divergence is one way a Fake world Node differs from production. Every
// launch refuses a Node whose config or environment differs in a way no
// entry covers.
type divergence struct {
	id string
	// keys are config key patterns, matched with path.Match.
	keys   []string
	env    []string
	reason string
	spec   string
}

var divergences = []divergence{
	{
		id:     "loopback-listen",
		keys:   []string{"*.listen"},
		reason: "Every listener binds its Node's own 127.0.0.N on port 0.",
		spec:   "#130",
	},
	{
		id:     "seed",
		keys:   []string{"seed"},
		reason: "Each Node's seed is <scenario seed>/<node>, so runs replay. Production draws a random one.",
		spec:   "#127",
	},
	{
		id:     "timings",
		keys:   []string{"timings.*"},
		reason: "Protocol durations are shortened for the wall-clock tier (boundary 42).",
		spec:   "#130",
	},
	{
		id:     "test-ca",
		env:    []string{"SSL_CERT_FILE"},
		reason: "Nodes trust the run's CA, so fakes serve real TLS with no product trust setting.",
		spec:   "#130",
	},
}

// setting is a config key or an environment variable a Node runs with
// that a production Node does not.
type setting struct {
	name string
	env  bool
}

// settings are a Node's overrides and the variables the world adds to its
// environment.
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
	return slices.ContainsFunc(d.keys, func(pattern string) bool {
		ok, _ := path.Match(pattern, s.name)
		return ok
	})
}

// diverge sorts settings into the divergence ids that cover them and the
// names no entry covers.
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

// usedIDs is the divergence ids in used, in the list's order.
func usedIDs(used map[string]bool) []string {
	ids := []string{}
	for _, d := range divergences {
		if used[d.id] {
			ids = append(ids, d.id)
		}
	}
	return ids
}

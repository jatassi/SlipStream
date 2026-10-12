package node

import (
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/node/internal/config"
)

// Census lists every endpoints key whose value still contains a host from
// its default, the hosts a fake has not replaced.
func Census(cfg Config) []string {
	defaults := Defaults()
	return census(&cfg, &defaults)
}

func census(tree, defaults any) []string {
	def := config.Walk(defaults)
	out := []string{}
	for i, leaf := range config.Walk(tree) {
		if !strings.HasPrefix(leaf.Path, "endpoints.") {
			continue
		}
		if slices.ContainsFunc(leaf.Hosts(), func(h string) bool { return slices.Contains(def[i].Hosts(), h) }) {
			out = append(out, leaf.Path)
		}
	}
	return out
}

// Resolved is what slipstream config prints: every key with its value,
// default and source, and the census.
type Resolved struct {
	Keys   []Key    `json:"keys"`
	Census []string `json:"census"`
}

// Resolve loads the configuration of the Node on dir without starting it.
func Resolve(dir DataDir, environ []string) (Resolved, error) {
	cfg, keys, err := LoadConfig(dir, environ)
	if err != nil {
		return Resolved{}, err
	}
	return Resolved{Keys: keys, Census: Census(cfg)}, nil
}

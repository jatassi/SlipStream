package node

import (
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node/internal/config"
)

// Census lists every endpoints key whose value still contains a host from
// its default, the hosts a fake has not replaced.
func Census(cfg Config, types contenttype.Set, kind install.Kind) []string {
	defaults := Defaults(types, kind)
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

// Resolved is what slipstream config prints: every key, the census, and
// the ids of the Content Types the binary registers.
type Resolved struct {
	Keys         []Key    `json:"keys"`
	Census       []string `json:"census"`
	ContentTypes []string `json:"contentTypes"`
}

func Resolve(env Env) (Resolved, error) {
	cfg, keys, err := LoadConfig(env.DataDir, env.Environ, env.Types, env.Install)
	if err != nil {
		return Resolved{}, err
	}
	ids := []string{}
	for _, id := range env.Types.IDs() {
		ids = append(ids, string(id))
	}
	return Resolved{Keys: keys, Census: Census(cfg, env.Types, env.Install), ContentTypes: ids}, nil
}

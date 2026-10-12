package node

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/jatassi/slipstream/internal/node/internal/config"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// Config is a Node's whole configuration. Every package's Settings sits at
// the top level, its Endpoints under endpoints and its Timings under
// timings. Adding a section is a field here and its line in Defaults.
type Config struct {
	Seed      string          `yaml:"seed"`
	Torrent   engine.Settings `yaml:"torrent"`
	Endpoints EndpointsBlock  `yaml:"endpoints"`
	Timings   TimingsBlock    `yaml:"timings"`
}

type EndpointsBlock struct{}

type TimingsBlock struct {
	Node Timings `yaml:"node"`
}

type Timings struct {
	StopDeadline time.Duration `yaml:"stop_deadline"`
}

func DefaultTimings() Timings {
	return Timings{StopDeadline: 30 * time.Second}
}

func Defaults() Config {
	return Config{
		Torrent: engine.DefaultSettings(),
		Timings: TimingsBlock{Node: DefaultTimings()},
	}
}

type Key = config.Key

// LoadConfig resolves the configuration of the Node on dir. An empty seed
// becomes 16 random bytes in hex, reported with source random.
func LoadConfig(dir DataDir, environ []string) (Config, []Key, error) {
	cfg := Defaults()
	keys, err := config.Load(&cfg, dir.Path(), environ)
	if err != nil {
		return Config{}, nil, &ConfigError{Err: err}
	}
	if cfg.Seed == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		cfg.Seed = hex.EncodeToString(b)
		for i := range keys {
			if keys[i].Key == "seed" {
				keys[i].Value, keys[i].Source = cfg.Seed, config.SourceRandom
			}
		}
	}
	return cfg, keys, nil
}

// overrides are the keys a layer set to something other than the default.
func overrides(keys []Key) []Key {
	var out []Key
	for _, k := range keys {
		if k.Value != k.Default && k.Source != config.SourceRandom {
			out = append(out, k)
		}
	}
	return out
}

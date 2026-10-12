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

// EndpointsBlock holds every fixed external host. No boundary exists yet.
type EndpointsBlock struct{}

// TimingsBlock holds every protocol duration.
type TimingsBlock struct {
	Node Timings `yaml:"node"`
}

// Timings is the Node's own protocol durations.
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

// Key is one resolved configuration key with its value, default and source.
type Key = config.Key

// LoadConfig resolves the configuration of the Node on dir from the
// environment, then dir/.env, then dir/config.yaml, then the defaults. An
// empty seed becomes 16 random bytes in hex, reported with source random.
func LoadConfig(dir DataDir, environ []string) (Config, []Key, error) {
	cfg := Defaults()
	keys, err := config.Load(&cfg, dir.Path(), environ)
	if err != nil {
		return Config{}, nil, &ConfigError{Err: err}
	}
	if cfg.Seed == "" {
		cfg.Seed = hex.EncodeToString(randomBytes(16))
		for i := range keys {
			if keys[i].Key == "seed" {
				keys[i].Value, keys[i].Source = cfg.Seed, config.SourceRandom
			}
		}
	}
	return cfg, keys, nil
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
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

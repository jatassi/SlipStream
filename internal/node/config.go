package node

import (
	"crypto/rand"
	"encoding/hex"
	"net/netip"
	"os"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/node/internal/config"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// Config is a Node's whole configuration. Every package's Settings sits at
// the top level, its Endpoints under endpoints and its Timings under
// timings. Adding a section is a field here and its line in Defaults.
type Config struct {
	Seed string `yaml:"seed"`
	// Name is sent once, at Pair. The Controller's row owns it after.
	Name      string           `yaml:"name"`
	Torrent   engine.Settings  `yaml:"torrent"`
	HTTP      api.Settings     `yaml:"http"`
	Link      link.Settings    `yaml:"link"`
	Join      pairing.Settings `yaml:"join"`
	Endpoints EndpointsBlock   `yaml:"endpoints"`
	Timings   TimingsBlock     `yaml:"timings"`
}

type EndpointsBlock struct{}

type TimingsBlock struct {
	Node    Timings         `yaml:"node"`
	Storage storage.Timings `yaml:"storage"`
	HTTP    api.Timings     `yaml:"http"`
	Live    live.Timings    `yaml:"live"`
	Link    link.Timings    `yaml:"link"`
	Role    role.Timings    `yaml:"role"`
}

type Timings struct {
	StopDeadline time.Duration `yaml:"stop_deadline"`
}

func DefaultTimings() Timings {
	return Timings{StopDeadline: 30 * time.Second}
}

// Defaults is every key's default. The link and role sections have theirs
// here, since only the composition root binds link.listen.
func Defaults() Config {
	name, _ := os.Hostname()
	return Config{
		Name:    name,
		Torrent: engine.DefaultSettings(),
		HTTP:    api.DefaultSettings(),
		Link: link.Settings{
			Listen:       netip.AddrPortFrom(netip.IPv6Unspecified(), 8485),
			StreamWindow: 16 << 20,
		},
		Timings: TimingsBlock{
			Node:    DefaultTimings(),
			Storage: storage.DefaultTimings(),
			HTTP:    api.DefaultTimings(),
			Live:    live.DefaultTimings(),
			Link: link.Timings{
				Handshake:     10 * time.Second,
				Keepalive:     25 * time.Second,
				WriteTimeout:  10 * time.Second,
				RedialMin:     time.Second,
				RedialMax:     time.Minute,
				FallbackDelay: 300 * time.Millisecond,
			},
			Role: role.Timings{RestartMin: time.Second, RestartMax: time.Minute},
		},
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

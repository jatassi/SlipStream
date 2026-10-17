package node

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"net/netip"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/node/internal/config"
	"github.com/jatassi/slipstream/internal/node/internal/logs"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
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
	Log       logs.Settings    `yaml:"log"`
	Endpoints EndpointsBlock   `yaml:"endpoints"`
	Timings   TimingsBlock     `yaml:"timings"`
}

type EndpointsBlock struct {
	TMDB         tmdb.Endpoints       `yaml:"tmdb"`
	ContentTypes ContentTypeEndpoints `yaml:"content_types"`
}

type ContentTypeEndpoints struct { //nolint:recvcheck // the walker and yaml hold the value; only Set changes it
	sections []config.Section
	set      map[string]map[string]string
}

func contentTypeEndpoints(types contenttype.Set) ContentTypeEndpoints {
	var c ContentTypeEndpoints
	for _, t := range types.All() {
		if e := t.Endpoints(); e != nil {
			c.sections = append(c.sections, config.Section{Name: string(t.ID()), Tree: e})
		}
	}
	return c
}

func (c ContentTypeEndpoints) ConfigSections() []config.Section { return c.sections }

func (c ContentTypeEndpoints) section(id works.TypeID) any {
	for _, s := range c.sections {
		if s.Name == string(id) {
			return s.Tree
		}
	}
	return nil
}

// Set points endpoints.content_types.<id>.<key> at url, on a block that
// holds no section for id.
func (c *ContentTypeEndpoints) Set(id, key, url string) {
	for _, s := range c.sections {
		if s.Name == id {
			panic("node: endpoints.content_types." + id + " has its typed section; set its field")
		}
	}
	if c.set == nil {
		c.set = map[string]map[string]string{}
	}
	if c.set[id] == nil {
		c.set[id] = map[string]string{}
	}
	c.set[id][key] = url
}

func (c ContentTypeEndpoints) Paths() []string {
	var out []string
	for id, keys := range c.set {
		for key := range keys {
			out = append(out, "endpoints.content_types."+id+"."+key)
		}
	}
	slices.Sort(out)
	return out
}

func (c ContentTypeEndpoints) MarshalYAML() (any, error) {
	out := map[string]any{}
	for _, s := range c.sections {
		out[s.Name] = s.Tree
	}
	for id, keys := range c.set {
		out[id] = keys
	}
	return out, nil
}

type TimingsBlock struct {
	Node        Timings             `yaml:"node"`
	Storage     storage.Timings     `yaml:"storage"`
	HTTP        api.Timings         `yaml:"http"`
	Live        live.Timings        `yaml:"live"`
	Link        link.Timings        `yaml:"link"`
	Role        role.Timings        `yaml:"role"`
	Works       works.Timings       `yaml:"works"`
	Indexers    indexers.Timings    `yaml:"indexers"`
	Transfers   transfers.Timings   `yaml:"transfers"`
	Grabs       grabs.Timings       `yaml:"grabs"`
	Files       files.Timings       `yaml:"files"`
	Downloading downloading.Timings `yaml:"downloading"`
	Media       media.Timings       `yaml:"media"`
	Mail        mail.Timings        `yaml:"mail"`
}

type Timings struct {
	StopDeadline time.Duration `yaml:"stop_deadline"`
	// RestartLockWait is how long a restarted process waits for the one it
	// follows to release the data dir lock.
	RestartLockWait time.Duration `yaml:"restart_lock_wait"`
}

func DefaultTimings() Timings {
	return Timings{StopDeadline: 30 * time.Second, RestartLockWait: 30 * time.Second}
}

// Defaults is every key's default for an install of kind, with a fresh
// section for each of types' endpoints. The link and role sections have
// theirs here, since only the composition root binds link.listen.
func Defaults(types contenttype.Set, kind install.Kind) Config {
	name, _ := os.Hostname()
	http := api.DefaultSettings()
	http.PortsToTry = kind.PortsToTry()
	return Config{
		Name:    name,
		Torrent: engine.DefaultSettings(),
		HTTP:    http,
		Log:     logs.DefaultSettings(),
		Endpoints: EndpointsBlock{
			TMDB:         tmdb.DefaultEndpoints(),
			ContentTypes: contentTypeEndpoints(types),
		},
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
				LogPoll:       20 * time.Second,
				LogArchiveTTL: 10 * time.Minute,
			},
			Role:        role.Timings{RestartMin: time.Second, RestartMax: time.Minute},
			Works:       works.DefaultTimings(),
			Indexers:    indexers.DefaultTimings(),
			Transfers:   transfers.DefaultTimings(),
			Grabs:       grabs.DefaultTimings(),
			Files:       files.DefaultTimings(),
			Downloading: downloading.DefaultTimings(),
			Media:       media.DefaultTimings(),
			Mail:        mail.DefaultTimings(),
		},
	}
}

type Key = config.Key

// LoadConfig resolves the configuration of the Node on dir. An empty seed
// becomes 16 random bytes in hex, reported with source random.
func LoadConfig(dir DataDir, environ []string, types contenttype.Set, kind install.Kind) (Config, []Key, error) {
	return loadConfig(os.DirFS(dir.Path()), environ, types, kind)
}

// loadConfig is LoadConfig with config.yaml and .env read from fsys.
func loadConfig(fsys fs.FS, environ []string, types contenttype.Set, kind install.Kind) (Config, []Key, error) {
	cfg := Defaults(types, kind)
	keys, err := config.LoadFS(&cfg, fsys, environ)
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

// CheckConfig holds the rules no leaf type carries: the Node settings
// table's ranges, and a web port the Node binds for something else.
func (c *Config) CheckConfig(fail func(key string, err error)) {
	if len(c.Link.Advertise) > 0 {
		if _, err := link.ParseAddresses(c.Link.Advertise); err != nil {
			fail("link.advertise", err)
		}
	}
	for _, r := range nodeSettings {
		if err := r.check(c); err != nil {
			fail(r.key(), err)
		}
	}
	if err := c.webPortUsed(); err != nil {
		fail("http.listen", err)
	}
}

// PortUsedError is http.listen's port configured for another of the
// Node's listeners, which then can never bind it.
type PortUsedError struct {
	Port uint16
	// Other is what uses the port: "the Node link" or "the torrent client".
	Other string
}

func (e *PortUsedError) Error() string {
	return fmt.Sprintf("%s already uses port %d", e.Other, e.Port)
}

// Sentence is the error as the port's field shows it.
func (e *PortUsedError) Sentence() string {
	return strings.ToUpper(e.Other[:1]) + e.Error()[1:] + "."
}

func (c *Config) webPortUsed() error {
	port := c.HTTP.Listen.Port()
	for _, other := range []struct {
		addr netip.AddrPort
		name string
	}{{c.Link.Listen, "the Node link"}, {c.Torrent.Listen, "the torrent client"}} {
		if port != 0 && other.addr.Port() == port {
			return &PortUsedError{Port: port, Other: other.name}
		}
	}
	return nil
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

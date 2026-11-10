// Package searching is the Searching Role's holder half: the indexer
// contract every kind implements on the Node an Indexer routes to. The
// Controller sends a Job (one search, recent-items call, fetch, Test or
// captcha) with the Indexer's address, opened settings, definition and
// this Node's session. The Node signs in, spaces requests per host, runs
// the kind's protocol and answers normalized Releases with one outcome. It keeps only memory: each
// Indexer's newest session, the per-host spacing, and the jobs running or
// just answered.
package searching

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/cardigann"
)

type Config struct {
	// HTTP reaches the network. The Service sends every request through a
	// copy that follows its own redirect policy.
	HTTP    *http.Client
	Timings Timings
	Log     *slog.Logger
	// Native runs kind Native. The Node injects it, so this package
	// imports no ported site.
	Native Driver
}

type Timings struct {
	// Request bounds one request, its body included.
	Request time.Duration `yaml:"request"`
	// Floor is the least spacing between two requests to one host.
	Floor time.Duration `yaml:"floor"`
	// Hold is the longest a Run call waits before it answers Running.
	Hold time.Duration `yaml:"hold"`
	// JobCap is the longest a job runs on the Node. It is longer than the
	// longest search wait, so a search past its wait is late, never failing.
	JobCap time.Duration `yaml:"job_cap"`
	// Keep is how long a finished answer stays for a call whose answer was
	// lost.
	Keep time.Duration `yaml:"keep"`
	// Results is the most Releases a Torznab search pages to.
	Results int `yaml:"results"`
	// Relogins is how many times a job signs in again after a response
	// reads as signed out.
	Relogins int `yaml:"relogins"`
}

func DefaultTimings() Timings {
	return Timings{
		Request: 30 * time.Second, Floor: 2 * time.Second, Hold: 20 * time.Second,
		JobCap: 11 * time.Minute, Keep: 2 * time.Minute, Results: 1000, Relogins: 1,
	}
}

// Service is the SearchingService every Node mounts.
type Service struct {
	cfg      Config
	client   *http.Client
	jobs     *jobs
	sessions *sessions
	cleared  *clearances
	spacer   *spacer
	defs     *definitions
	drivers  map[Kind]Driver
}

func New(cfg Config) *Service {
	client := *cfg.HTTP
	client.CheckRedirect = redirects
	client.Jar = nil
	drivers := map[Kind]Driver{
		Catalog: catalogDriver{relogins: cfg.Timings.Relogins},
		Torznab: torznabDriver{results: cfg.Timings.Results},
		RSS:     rssDriver{},
	}
	if cfg.Native != nil {
		drivers[Native] = cfg.Native
	}
	return &Service{
		cfg:      cfg,
		client:   &client,
		jobs:     &jobs{keep: cfg.Timings.Keep, byID: map[string]*run{}},
		sessions: &sessions{by: map[string]*slot{}},
		cleared:  &clearances{byHost: map[string]clearance{}},
		spacer:   &spacer{floor: cfg.Timings.Floor, next: map[string]time.Time{}},
		defs:     &definitions{bySHA: map[string]*cardigann.Definition{}},
		drivers:  drivers,
	}
}

package conversions

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1/convertingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// DefaultAtOnce is a Node's Conversions at once until its Admin sets one:
// one, two and three copies of the Quick Sync chain ran at about the same
// total frame rate.
const DefaultAtOnce = 1

type Timings struct {
	// DrainBatch is the nudges one Write converges.
	DrainBatch int `yaml:"drain_batch"`
	// Pass is the longest the reconciler sleeps with nothing due.
	Pass time.Duration `yaml:"pass"`
	// Horizon is how far ahead the forecast runs.
	Horizon time.Duration `yaml:"horizon"`
	// AssumedRate is bytes a second for a Node pair with no measured rate.
	AssumedRate int64 `yaml:"assumed_rate"`
	// TestsAtOnce is the test runs in flight per Node.
	TestsAtOnce int `yaml:"tests_at_once"`
	// Landing is how long before a root-side check whose probe failed is
	// tried again.
	Landing time.Duration `yaml:"landing"`
}

func DefaultTimings() Timings {
	return Timings{
		DrainBatch: 200, Pass: 30 * time.Second, Horizon: 7 * 24 * time.Hour,
		AssumedRate: 50 << 20, TestsAtOnce: 1, Landing: 30 * time.Second,
	}
}

// Files is *files.Files, as conversions reads it.
type Files interface {
	Sourced(ctx context.Context, r storage.Reader[storage.Library], ws works.Among[works.WorkID]) (map[works.WorkID][]files.Sourced, error)
	Located(ctx context.Context, r storage.Reader[storage.Library], id files.ID) (files.Located, error)
}

// Transfers is *transfers.Transfers, as conversions copies sources and
// returns outputs with it.
type Transfers interface {
	Want(ctx context.Context, tx *storage.Tx[storage.Library], r transfers.Request) (transfers.ID, error)
	Touch()
}

type Publisher[T any] interface {
	Publish(ctx context.Context, v T)
}

// Poke wakes the reconciler. Waking it while it is awake is nothing.
type Poke struct{ c chan struct{} }

func NewPoke() *Poke { return &Poke{c: make(chan struct{}, 1)} }

func (p *Poke) Wake() {
	select {
	case p.c <- struct{}{}:
	default:
	}
}

func (p *Poke) C() <-chan struct{} { return p.c }

type Config struct {
	Library   *storage.DB[storage.Library]
	Files     Files
	Targets   *targets.Targets
	Transfers Transfers
	Board     func() *nodes.Board
	// Control is the Converting Control, served after New, for its facts
	// and Touch.
	Control    func() *role.Control[*cv1.Intent, *cv1.Facts, *cv1.Result]
	Converting func(node role.NodeID) convertingv1connect.ConvertingServiceClient
	// Probes probe an output on its root's Node.
	Probes storing.Probes
	// RootRoom is the free bytes on root's volume, and whether they are
	// known: imports.CopyRoom.
	RootRoom func(ctx context.Context, r storage.Reader[storage.Library], root roots.ID) (int64, bool, error)
	// TranscodingOn is each Media Server on node that pauses its
	// Conversions, and whether it transcodes video now.
	TranscodingOn func(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) ([]mediaservers.Transcoding, error)
	History       entry.Recorder
	Changed       Publisher[Changed]
	// Live is sampled while Listening.
	Live live.AdminPublisher[LiveSample]
	Hub  *live.Hub
	Poke *Poke
	// Reconciling is whether a restored Controller is Reconciling.
	Reconciling func(ctx context.Context, r storage.Reader[storage.Library]) (bool, error)
	// Zone is the Maintenance window's zone, which a Node's hours are in.
	Zone    func() *time.Location
	Timings Timings
	Run     converting.Timings
	Clock   func() time.Time
	Log     *slog.Logger
}

type Conversions struct {
	cfg      Config
	importer Importer
}

func New(cfg Config) *Conversions { return &Conversions{cfg: cfg} }

var errNotBuilt = errors.New("conversions isn't built yet")

var (
	ErrNoConversion = errors.New("no Conversion has that id")
	ErrFinished     = errors.New("the Conversion has finished")
	ErrNotWaiting   = errors.New("the Conversion isn't waiting")
	ErrNotFailed    = errors.New("the Conversion hasn't failed")
	ErrPlacing      = errors.New("the Conversion's file is being placed")
)

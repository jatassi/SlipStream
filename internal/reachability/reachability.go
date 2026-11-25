package reachability

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/nodes"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/network/v1/networkv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Config struct {
	Library *storage.DB[storage.Library]
	Hub     *live.Hub
	Log     *slog.Logger
	Roster  Roster
	Tasks   Starter
	// Network reaches one Node's NodeNetworkService.
	Network      func(role.NodeID) networkv1connect.NodeNetworkServiceHandler
	MediaServers MediaServers
	Rules        netmodel.Rules
	// LinkSource is where the named Node's link appears to come from: its
	// remote address in production, and under scripted-network the
	// address its site shows outside.
	LinkSource func(name string, remote netip.AddrPort) netip.Addr
	// WebPort is the port SlipStream's own HTTP listener took, the UI and
	// portal's local port.
	WebPort func() uint16
	Timings Timings
	Zone    *time.Location
	Changed live.AdminPublisher[Changed]
	Now     func() time.Time
}

// Roster is what reachability reads of the Nodes: *nodes.Service.
type Roster interface {
	Self() role.NodeID
	InStep(role.NodeID) bool
	// Serving is every Node that serves r.
	Serving(r role.Name) []role.NodeID
	// Remote is the Node link's remote address and whether the Controller
	// dialed it, false with no live link.
	Remote(role.NodeID) (addr netip.AddrPort, controllerDialed, ok bool)
	// Names is every Node, removed ones marked.
	Names(context.Context) (map[role.NodeID]nodes.Named, error)
}

// MediaServers is what a unit asks of the Media Servers.
type MediaServers interface {
	Exposure(ctx context.Context, id link.ServerID) (mediaservers.Exposure, error)
}

// Starter starts the task's units: *tasks.Scheduler.
type Starter interface {
	Start(ctx context.Context, st tasks.Start) (tasks.Started, error)
}

var (
	ErrNotFound       = errors.New("no route has this id")
	ErrExists         = errors.New("this is already set up for use away from home")
	ErrPublicRequired = errors.New("a Media Server's route needs its public address")
	ErrPublicNotURL   = errors.New("the public address isn't an http or https URL with a host")
	ErrServerNotFound = errors.New("no Media Server has this id")
	ErrNothingWanted  = errors.New("nothing is set up for use away from home")
	ErrNodeNotFound   = errors.New("no Node has this id")
)

// Reachability is Remote access on the Controller: the Admin's Exposures,
// the task that judges them, the echo, the network channel's intent, Health
// and the two settings panels.
type Reachability struct {
	cfg     Config
	network *role.Control[*netv1.Intent, *netv1.Facts, *emptypb.Empty]
	echoes  echoes
	public  memo[string, bool]
	exposed memo[link.ServerID, mediaSuggestion]
	lookups lookups

	mu sync.Mutex
	// freshAfter is when each Node's last "Run again" or confirmed change
	// was: no STUN pass from before it counts.
	freshAfter map[role.NodeID]time.Time
	// reruns is each Node's last network-change rerun, and churn the Nodes
	// whose change came again inside one watch interval.
	reruns map[role.NodeID]time.Time
	churn  map[role.NodeID]bool
	armed  map[role.NodeID]bool
	// known is the wanted Media Server Exposures Run last saw, so it can
	// tell when a server's removal took one with it.
	known map[storage.UUID]bool
}

func New(cfg Config) *Reachability {
	return &Reachability{
		cfg:        cfg,
		echoes:     echoes{live: map[string]*echo{}},
		lookups:    lookups{wake: make(chan struct{}, 1)},
		freshAfter: map[role.NodeID]time.Time{},
		reruns:     map[role.NodeID]time.Time{},
		churn:      map[role.NodeID]bool{},
		armed:      map[role.NodeID]bool{},
	}
}

func (r *Reachability) now() time.Time { return r.cfg.Now() }

// Serve keeps the network channel's control, which Intent derives for.
func (r *Reachability) Serve(c *role.Control[*netv1.Intent, *netv1.Facts, *emptypb.Empty]) {
	r.network = c
}

// touch re-derives every live Node's intent.
func (r *Reachability) touch(ctx context.Context) {
	named, err := r.cfg.Roster.Names(ctx)
	if err != nil {
		r.cfg.Log.WarnContext(ctx, "list the Nodes to tell them what to watch", "err", err)
		return
	}
	var ids []role.NodeID
	for id, n := range named {
		if !n.Removed {
			ids = append(ids, id)
		}
	}
	r.network.Touch(ids...)
}

func (r *Reachability) changed(ctx context.Context, ids ...storage.UUID) {
	c := Changed{ExposureIDs: []string{}}
	for _, id := range ids {
		c.ExposureIDs = append(c.ExposureIDs, id.String())
	}
	r.cfg.Changed.Publish(ctx, c)
}

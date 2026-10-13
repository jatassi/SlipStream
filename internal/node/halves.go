package node

import (
	"context"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// peers is how the link treats a peer under the Node's claim: whom it
// admits, and how it answers a Pair.
type peers interface {
	link.Trust
	Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error)
}

// half is the claimed part of a running Node. Each Node starts at most one,
// from Unclaimed or at start, and never restarts it in process.
type half struct {
	peers    peers
	dispatch *rpc.Dispatch
	// nodes, controls, cluster and works are the Controller's, nil on a
	// Worker.
	nodes    *nodes.Service
	controls *role.Controls
	cluster  *rpc.Cluster
	works    *works.Works
}

// startController starts the Controller half: the control half, its own
// holder half in process, and call dispatch over the roster.
func (n *Node) startController(ctx context.Context) error {
	lib, err := n.openLibrary(ctx)
	if err != nil {
		return err
	}
	tree, err := works.Open(ctx, works.Config{
		Library:      lib.db,
		Catalogs:     contenttype.Project(n.openTypes(), func(t contenttype.Type) works.Catalog { return t }),
		Timings:      n.cfg.Timings.Works,
		Changed:      works.ChangedEvent.Bind(n.hub),
		TypesChanged: works.ContentTypesChangedEvent.Bind(n.hub),
		Log:          n.rootLog.With("subsystem", "works"),
	})
	if err != nil {
		return err
	}
	side, err := nodes.Open(ctx, nodes.Config{
		Library: lib.db,
		Key:     n.key.Public(),
		Name:    n.cfg.Name,
		Node:    n.info,
		Link:    n.link,
		Changed: nodes.ChangedEvent.Bind(n.hub),
		Served:  n.env.Test.Served,
		Always:  n.env.Test.Always,
		Log:     n.rootLog.With("subsystem", "nodes"),
	})
	if err != nil {
		return err
	}
	controls := side.Controls()
	holding, local, err := n.startHolding(ctx, role.InProcess(controls))
	if err != nil {
		return err
	}
	self := n.key.Public()
	dispatch := rpc.NewDispatch(n.link, self, self, holding.Holds)
	cluster := rpc.NewCluster(dispatch, side)
	n.lockstep.Handle(controls.Handler())
	n.lockstep.Handle("/", dispatch)
	n.setHalf(&half{peers: side, dispatch: dispatch, nodes: side, controls: controls, cluster: cluster, works: tree})
	n.claimed.Go("nodes", side.Run)
	n.env.Test.started(Half{Link: n.link, Dispatch: dispatch, Local: local, Library: lib.db, Controls: controls, Cluster: cluster})
	return nil
}

// openTypes builds each registered Content Type once, from its loaded
// endpoints section, with one HTTP client that dials through the Node's
// network and trusts the OS roots.
func (n *Node) openTypes() map[works.TypeID]contenttype.Type {
	transport := &http.Transport{
		DialContext:         n.env.Net.DialContext,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{Transport: transport}
	types := map[works.TypeID]contenttype.Type{}
	for _, t := range n.env.Types.All() {
		types[t.ID()] = t.Open(contenttype.Deps{
			HTTP:      client,
			UserAgent: "SlipStream/" + n.env.Version + " ( github.com/jatassi/SlipStream )",
			Log:       n.rootLog.With("subsystem", "contenttype."+string(t.ID())),
		}, n.cfg.Endpoints.ContentTypes.section(t.ID()))
	}
	return types
}

// startWorker starts the Worker half: the holder half over the link to its
// Controller, and call dispatch toward it.
func (n *Node) startWorker(ctx context.Context) error {
	side := n.pairing
	controller := side.Controller()
	holding, local, err := n.startHolding(ctx, role.OverLink(n.link, controller.Key))
	if err != nil {
		return err
	}
	dispatch := rpc.NewDispatch(n.link, n.key.Public(), controller.Key, holding.Holds)
	n.lockstep.Handle("/", dispatch)
	n.setHalf(&half{peers: side, dispatch: dispatch})
	n.link.Keep(controller)
	n.claimed.Go("pairing", side.Run)
	n.env.Test.started(Half{Link: n.link, Dispatch: dispatch, Local: local})
	return nil
}

// startHolding opens local.db and runs the holder half toward upstream.
func (n *Node) startHolding(ctx context.Context, upstream role.Upstream) (*role.Holding, *storage.DB[storage.Local], error) {
	local, err := storage.OpenLocal(ctx, n.env.DataDir.LocalDB(), localSchemas(), n.cfg.Timings.Storage, n.rootLog.With("subsystem", "storage"))
	if err != nil {
		return nil, nil, err
	}
	n.claimed.OnStop("local.db", func(context.Context) error { return local.Close() })
	holding, err := role.NewHolding(ctx, role.HoldingConfig{
		Local:    local,
		Upstream: upstream,
		Held:     append(heldRoles(), n.env.Test.held(local)...),
		Timings:  n.cfg.Timings.Role,
		Rand:     n.roleRand,
		Log:      n.rootLog.With("subsystem", "role"),
	})
	if err != nil {
		return nil, nil, err
	}
	n.claimed.Go("role", holding.Run)
	return holding, local, nil
}

func (n *Node) setHalf(h *half) {
	if !n.half.CompareAndSwap(nil, h) {
		panic("node: a second half started; a Node starts one, once")
	}
}

// heldRoles is every Role this binary runs, one held line each.
func heldRoles() []role.Held { return nil }

// claimPeers is how the link treats peers: as the half that started does,
// and refusing every one while the Node is Unclaimed.
type claimPeers struct{ n *Node }

func (p claimPeers) Admit(key link.Key) link.Admission {
	if h := p.n.half.Load(); h != nil {
		return h.peers.Admit(key)
	}
	return link.Refused{}
}

func (p claimPeers) Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	if h := p.n.half.Load(); h != nil {
		return h.peers.Pair(ctx, peer, req)
	}
	return notAController(), nil
}

func notAController() *linkv1.PairResponse {
	return &linkv1.PairResponse{Outcome: &linkv1.PairResponse_Refused{Refused: &linkv1.Refused{Reason: linkv1.RefusedReason_REFUSED_REASON_NOT_A_CONTROLLER}}}
}

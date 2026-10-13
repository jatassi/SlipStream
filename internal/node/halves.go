package node

import (
	"context"

	"github.com/jatassi/slipstream/internal/link"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// peers is how the link treats a peer under the Node's claim: whom it
// admits, and how it answers a Pair.
type peers interface {
	link.Trust
	Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error)
}

// controllerSide is the nodes owner's part of the Controller half: the
// roster calls and the control half read, and the peers it admits.
type controllerSide interface {
	peers
	rpc.Roster
	role.Nodes
	// Assigned derives the node scope: the Roles assigned to node.
	Assigned(ctx context.Context, r storage.Reader[storage.Library], node role.NodeID) ([]role.Name, error)
	// Changed hears that node's session or facts changed. It must not
	// block.
	Changed(node role.NodeID)
}

// workerSide is the pairing owner's part of the Worker half.
type workerSide interface {
	peers
	// Controller is the pinned Controller's key and recorded addresses.
	Controller() link.Target
}

// half is the claimed part of a running Node. Each Node starts at most one,
// from Unclaimed or at start, and never restarts it in process.
type half struct {
	peers    peers
	dispatch *rpc.Dispatch
	// controls and cluster are the Controller's, nil on a Worker.
	controls *role.Controls
	cluster  *rpc.Cluster
}

// startController starts the Controller half: the control half, its own
// holder half in process, and call dispatch over the roster.
func (n *Node) startController(ctx context.Context) error {
	lib, err := n.openLibrary(ctx)
	if err != nil {
		return err
	}
	side, err := n.controllerSide(ctx, lib.db)
	if err != nil {
		return err
	}
	controls := role.NewControls(role.ControlsConfig{
		Library:  lib.db,
		Nodes:    side,
		Assigned: side.Assigned,
		Changed:  side.Changed,
		Log:      n.rootLog.With("subsystem", "role"),
	})
	holding, err := n.startHolding(ctx, role.InProcess(controls))
	if err != nil {
		return err
	}
	self := n.key.Public()
	dispatch := rpc.NewDispatch(n.link, self, self, holding.Holds)
	n.lockstep.Handle(controls.Handler())
	n.lockstep.Handle("/", dispatch)
	n.setHalf(&half{peers: side, dispatch: dispatch, controls: controls, cluster: rpc.NewCluster(dispatch, side)})
	return nil
}

// startWorker starts the Worker half: the holder half over the link to its
// Controller, and call dispatch toward it.
func (n *Node) startWorker(ctx context.Context, side workerSide) error {
	controller := side.Controller()
	holding, err := n.startHolding(ctx, role.OverLink(n.link, controller.Key))
	if err != nil {
		return err
	}
	dispatch := rpc.NewDispatch(n.link, n.key.Public(), controller.Key, holding.Holds)
	n.lockstep.Handle("/", dispatch)
	n.setHalf(&half{peers: side, dispatch: dispatch})
	n.link.Keep(controller)
	return nil
}

// startHolding opens local.db and runs the holder half toward upstream.
func (n *Node) startHolding(ctx context.Context, upstream role.Upstream) (*role.Holding, error) {
	local, err := storage.OpenLocal(ctx, n.env.DataDir.LocalDB(), localSchemas(), n.cfg.Timings.Storage, n.rootLog.With("subsystem", "storage"))
	if err != nil {
		return nil, err
	}
	n.claimed.OnStop("local.db", func(context.Context) error { return local.Close() })
	holding, err := role.NewHolding(ctx, role.HoldingConfig{
		Local:    local,
		Upstream: upstream,
		Held:     heldRoles(),
		Timings:  n.cfg.Timings.Role,
		Rand:     n.roleRand,
		Log:      n.rootLog.With("subsystem", "role"),
	})
	if err != nil {
		return nil, err
	}
	n.claimed.Go("role", holding.Run)
	return holding, nil
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

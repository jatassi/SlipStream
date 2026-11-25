package netfacts

import (
	"context"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/netmodel"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/network/v1/networkv1connect"
	"github.com/jatassi/slipstream/internal/rpc"
)

// NewService serves Read, Resolve and Dial on this Node. Dispatch refuses
// every caller but the Controller before a call runs.
func NewService(n *Netfacts) networkv1connect.NodeNetworkServiceHandler { return service{n} }

type service struct{ n *Netfacts }

func (s service) Read(ctx context.Context, req *netv1.ReadRequest) (*netv1.ReadResponse, error) {
	ports := make([]uint16, 0, len(req.GetPorts()))
	for _, p := range req.GetPorts() {
		if p == 0 || p > 0xffff {
			return nil, rpc.Errorf(rpc.CodeInvalidArgument, "port %d", p)
		}
		ports = append(ports, uint16(p))
	}
	f, err := s.n.Facts(ctx, ports, time.Duration(req.GetMaxAgeMs())*time.Millisecond, req.GetLocalOnly())
	if err != nil {
		return nil, fmt.Errorf("read this Node's network: %w", err)
	}
	return &netv1.ReadResponse{Facts: netmodel.FactsOnWire(f)}, nil
}

func (s service) Resolve(ctx context.Context, req *netv1.ResolveRequest) (*netv1.ResolveResponse, error) {
	return s.n.resolve(ctx, req.GetHost()), nil
}

func (s service) Dial(ctx context.Context, req *netv1.DialRequest) (*netv1.DialResponse, error) {
	res, err := s.n.dial(ctx, req)
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	return res, nil
}

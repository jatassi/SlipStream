// Package mediaserverlink is the holder half of the Media Server link Role
// and the registry of kinds. Its wire conversion is the one boundary
// between the proto and link's types, and its Ports are how the
// Controller's sender reaches a server.
package mediaserverlink

import (
	"context"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
)

// Kinds is the one list of kinds. Adding a kind is a line here.
var Kinds = link.Registry{}

// Port is how the Controller's sender runs one step on a server. A
// link.Conn the Controller dials itself at the fallback address is one.
type Port interface {
	Advance(ctx context.Context, j link.Job) (link.Advanced, error)
}

// Linked is the part of MediaServerLinkService a Port calls.
type Linked interface {
	Advance(ctx context.Context, req *msv1.AdvanceRequest) (*msv1.AdvanceResponse, error)
}

// Remote is the reaching Node's Advance, the Controller's own through
// Local. An rpc error comes back as it is, the link's. A server's failure
// comes back as a *link.ServerError.
func Remote(c Linked) Port { return remote{c} }

type remote struct{ c Linked }

func (r remote) Advance(ctx context.Context, j link.Job) (link.Advanced, error) {
	resp, err := r.c.Advance(ctx, AdvanceRequestOf(j))
	if err != nil {
		return link.Advanced{}, err
	}
	return AdvancedOf(j.Server, resp)
}

package mediaserverlink

import (
	"context"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1/mediaserverv1connect"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// Service is the MediaServerLinkService the Node mounts: a draft's tests
// with the credential in the call, and one step of a Job on a server this
// Node reaches.
func (l *Link) Service() mediaserverv1connect.MediaServerLinkServiceHandler { return linkService{l} }

type linkService struct{ l *Link }

// draft dials a server the request names by kind and address.
func (s linkService) draft(kind, address string, d link.Dialed) (link.Conn, error) {
	k, ok := s.l.cfg.Kinds.Of(link.KindName(kind))
	if !ok {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "this Node has no Media Server kind %q", kind)
	}
	d.Address, d.HTTP, d.Timings = address, s.l.cfg.HTTP, s.l.cfg.Timings
	return k.Dial(d), nil
}

func (s linkService) Reach(ctx context.Context, req *msv1.ReachRequest) (*msv1.ReachResponse, error) {
	conn, err := s.draft(req.GetKind(), req.GetAddress(), link.Dialed{})
	if err != nil {
		return nil, err
	}
	id, err := conn.Reach(ctx)
	if err != nil {
		p, err := ProblemOnWire(err)
		return &msv1.ReachResponse{Problem: p}, err
	}
	return &msv1.ReachResponse{Identity: IdentityOnWire(id)}, nil
}

func (s linkService) Check(ctx context.Context, req *msv1.CheckRequest) (*msv1.CheckResponse, error) {
	conn, err := s.draft(req.GetKind(), req.GetAddress(), link.Dialed{Credential: CredentialOf(req.GetCredential()), DeviceID: req.GetDeviceId()})
	if err != nil {
		return nil, err
	}
	return CheckedOnWire(conn.Check(ctx))
}

func (s linkService) SignIn(ctx context.Context, req *msv1.SignInRequest) (*msv1.SignInResponse, error) {
	conn, err := s.draft(req.GetKind(), req.GetAddress(), link.Dialed{DeviceID: req.GetDeviceId()})
	if err != nil {
		return nil, err
	}
	cred, err := conn.SignIn(ctx, req.GetUser(), req.GetPassword())
	if err != nil {
		p, err := ProblemOnWire(err)
		return &msv1.SignInResponse{Problem: p}, err
	}
	return &msv1.SignInResponse{Credential: CredentialOnWire(cred)}, nil
}

// Advance runs one step on a server of this Node's intent, with its
// credential at the request's settings version or newer.
func (s linkService) Advance(ctx context.Context, req *msv1.AdvanceRequest) (*msv1.AdvanceResponse, error) {
	u, err := storage.UUIDFromBytes(req.GetServerId())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "read the Media Server id: %v", err)
	}
	id := link.ServerID{UUID: u}
	t, ok := s.l.target(id)
	if !ok {
		return nil, rpc.Errorf(rpc.CodeFailedPrecondition, "this Node doesn't reach Media Server %s", id)
	}
	t.ref.SettingsVersion = max(t.ref.SettingsVersion, req.GetSettingsVersion())
	h, err := s.l.credential(ctx, t.ref)
	if err != nil {
		return nil, err
	}
	job, err := JobOf(t.ref, req.GetJob())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "read the job: %v", err)
	}
	a, err := s.l.dial(t, h).Advance(ctx, job)
	if err != nil {
		p, err := ProblemOnWire(err)
		return &msv1.AdvanceResponse{Problem: p}, err
	}
	return AdvancedOnWire(a), nil
}

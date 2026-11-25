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

// saved dials a server of this Node's intent with its credential at
// version or newer.
func (s linkService) saved(ctx context.Context, server []byte, version int64) (link.ServerRef, link.Conn, error) {
	u, err := storage.UUIDFromBytes(server)
	if err != nil {
		return link.ServerRef{}, nil, rpc.Errorf(rpc.CodeInvalidArgument, "read the Media Server id: %v", err)
	}
	id := link.ServerID{UUID: u}
	t, ok := s.l.target(id)
	if !ok {
		return link.ServerRef{}, nil, rpc.Errorf(rpc.CodeFailedPrecondition, "Media Server %s doesn't connect from this Node", id)
	}
	t.ref.SettingsVersion = max(t.ref.SettingsVersion, version)
	h, err := s.l.credential(ctx, t.ref)
	if err != nil {
		return link.ServerRef{}, nil, err
	}
	return t.ref, s.l.dial(t, h), nil
}

// Advance runs one step on a server of this Node's intent.
func (s linkService) Advance(ctx context.Context, req *msv1.AdvanceRequest) (*msv1.AdvanceResponse, error) {
	ref, conn, err := s.saved(ctx, req.GetServerId(), req.GetSettingsVersion())
	if err != nil {
		return nil, err
	}
	job, err := JobOf(ref, req.GetJob())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "read the job: %v", err)
	}
	a, err := conn.Advance(ctx, job)
	if err != nil {
		p, err := ProblemOnWire(err)
		return &msv1.AdvanceResponse{Problem: p}, err
	}
	return AdvancedOnWire(a), nil
}

// Exposed reads what a server of this Node's intent reports about its use
// away from home.
func (s linkService) Exposed(ctx context.Context, req *msv1.ExposedRequest) (*msv1.ExposedResponse, error) {
	_, conn, err := s.saved(ctx, req.GetServerId(), req.GetSettingsVersion())
	if err != nil {
		return nil, err
	}
	return ExposedOnWire(conn.Exposed(ctx))
}

// Members runs one users call on a server of this Node's intent.
func (s linkService) Members(ctx context.Context, req *msv1.MembersRequest) (*msv1.MembersResponse, error) {
	_, conn, err := s.saved(ctx, req.GetServerId(), req.GetSettingsVersion())
	if err != nil {
		return nil, err
	}
	users, ok := UsersOf(conn)
	if !ok {
		return nil, rpc.Errorf(rpc.CodeFailedPrecondition, "this kind of Media Server has no users SlipStream can make")
	}
	return MembersOnWire(ctx, users, req)
}

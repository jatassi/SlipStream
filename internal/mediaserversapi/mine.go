package mediaserversapi

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
)

type myAccessIDInput struct {
	AccessID string `path:"accessId" format:"uuid"`
}

type makeAccountInput struct {
	AccessID string `path:"accessId" format:"uuid"`
	Body     struct {
		Username string `json:"username" minLength:"1" maxLength:"100" doc:"The name the Member signs in to the server's app with"`
		Password string `json:"password" minLength:"1" maxLength:"1024" doc:"Write-only. The Member's password for the server's app, set on the server and never stored"`
	}
}

type (
	myAccessOutput       struct{ Body mediaservers.MyAccess }
	myAccessRowOutput    struct{ Body mediaservers.MyAccessRow }
	memberPinOutput      struct{ Body mediaservers.MemberPin }
	memberPinStateOutput struct{ Body mediaservers.MemberPinState }
)

func myRow(r mediaservers.MyAccessRow, err error) (*myAccessRowOutput, error) {
	if err != nil {
		return nil, answerMine(err)
	}
	return &myAccessRowOutput{Body: r}, nil
}

func (h handlers) mine(ctx context.Context, me identity.Account, _ *struct{}) (*myAccessOutput, error) {
	a, err := h.svc().Mine(ctx, me)
	if err != nil {
		return nil, answerMine(err)
	}
	return &myAccessOutput{Body: a}, nil
}

func (h handlers) startMemberPin(ctx context.Context, me identity.Account, _ *struct{}) (*memberPinOutput, error) {
	p, err := h.svc().StartMemberPin(ctx, me)
	if err != nil {
		return nil, answerMine(err)
	}
	return &memberPinOutput{Body: p}, nil
}

func (h handlers) memberPin(ctx context.Context, me identity.Account, in *pinIDInput) (*memberPinStateOutput, error) {
	s, err := h.svc().MemberPin(ctx, me, in.ID)
	if err != nil {
		return nil, answerMine(err)
	}
	return &memberPinStateOutput{Body: s}, nil
}

func (h handlers) anotherPlex(ctx context.Context, me identity.Account, _ *struct{}) (*struct{}, error) {
	if err := h.svc().AnotherPlex(ctx, me); err != nil {
		return nil, answerMine(err)
	}
	return &struct{}{}, nil
}

func (h handlers) makeAccount(ctx context.Context, me identity.Account, in *makeAccountInput) (*myAccessRowOutput, error) {
	id, err := accessID(in.AccessID)
	if err != nil {
		return nil, err
	}
	pw, err := link.NewPassword(in.Body.Password)
	if err != nil {
		return nil, err
	}
	return myRow(h.svc().MakeAccount(ctx, me, id, in.Body.Username, pw))
}

func (h handlers) checkMine(ctx context.Context, me identity.Account, in *myAccessIDInput) (*myAccessRowOutput, error) {
	id, err := accessID(in.AccessID)
	if err != nil {
		return nil, err
	}
	return myRow(h.svc().CheckMine(ctx, me, id))
}

package mediaserversapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
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

// mineRoutes registers a Member's operations on their own access.
func mineRoutes(o *api.Owner, h handlers) {
	const base = "/account/media-servers"
	plex := []*api.Code{plexNotSignedIn, plexTVUnreached}
	api.Register(o, api.Account, api.Op{
		ID: "listMyMediaServerAccess", Method: http.MethodGet, Path: base, Status: http.StatusOK,
		Summary: "List the Media Servers the Member was offered, with what each needs next and how to watch on it",
		Errors:  building(),
	}, h.mine)
	api.Register(o, api.Confirmed, api.Op{
		ID: "startMemberPlexSignIn", Method: http.MethodPost, Path: base + "/plex/pin", Status: http.StatusCreated,
		Summary: "Start the Member's Plex sign-in and get the plex.tv page they sign in on",
		Errors:  building(plex...),
	}, h.startMemberPin)
	api.Register(o, api.Account, api.Op{
		ID: "getMemberPlexSignIn", Method: http.MethodGet, Path: base + "/plex/pin/{id}", Status: http.StatusOK,
		Summary: "Get whether the Member's Plex sign-in is still waiting, has expired, signed in or was refused",
		Errors:  building(plexPinNotFound),
	}, h.memberPin)
	api.Register(o, api.Account, api.Op{
		ID: "useAnotherPlexAccount", Method: http.MethodDelete, Path: base + "/plex", Status: http.StatusNoContent,
		Summary: "Forget the Plex account the Member signed in to Plex as, while no server is shared with it yet",
		Errors:  building(accessState),
	}, h.anotherPlex)
	api.Register(o, api.Confirmed, api.Op{
		ID: "makeMediaServerAccount", Method: http.MethodPost, Path: base + "/{accessId}/account", Status: http.StatusOK,
		Summary: "Make the Member's own user on a Jellyfin or Emby server they were offered",
		Errors:  building(accessNotFound, accessState, usernameTaken, serverRefused, unreached),
	}, h.makeAccount)
	api.Register(o, api.Account, api.Op{
		ID: "checkMyMediaServerAccess", Method: http.MethodPost, Path: base + "/{accessId}/check", Status: http.StatusOK,
		Summary: "Check whether the Member's Plex share is accepted, accepting it when SlipStream can",
		Errors:  building(accessNotFound, accessState, plexTVUnreached),
	}, h.checkMine)
}

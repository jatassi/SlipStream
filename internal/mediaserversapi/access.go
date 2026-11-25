package mediaserversapi

import (
	"context"
	"net/http"
	"slices"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mailapi"
	"github.com/jatassi/slipstream/internal/mediaservers"
)

type accessListInput struct {
	Account string `query:"account" format:"uuid" doc:"An Account id: only that Member's rows, and the servers they may be offered"`
}

type accessIDInput struct {
	ID string `path:"id" format:"uuid"`
}

type offerInput struct {
	ID       string `path:"id" format:"uuid" doc:"The Member's Account"`
	ServerID string `path:"serverId" format:"uuid"`
	Body     struct {
		PlexHome bool `json:"plexHome,omitempty" doc:"Plex only: add the Member to the Plex Home rather than as a friend"`
	}
}

type shareAccessInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		As string `json:"as" enum:"friend,home" doc:"friend shares the Libraries for Members; home also adds the Member to the Plex Home"`
	}
}

type linkUserInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		UserID string `json:"userId" minLength:"1" maxLength:"256" doc:"A user id from listMediaServerUsers"`
	}
}

type acted struct {
	Row  *mediaservers.AccessView `json:"row,omitempty" doc:"The row after the operation; absent when it deleted the row"`
	Mail *mailapi.Outcome         `json:"mail,omitempty" doc:"What became of the email telling the Member they can watch, when the operation sent one"`
}

type (
	accessListOutput  struct{ Body mediaservers.AccessList }
	actedOutput       struct{ Body acted }
	serverUsersOutput struct {
		Body struct {
			Users []mediaservers.ServerUser `json:"users" nullable:"false"`
		}
	}
)

func accessID(raw string) (mediaservers.AccessID, error) {
	u, err := parseUUID(raw, mediaservers.ErrAccessNotFound)
	return mediaservers.AccessID{UUID: u}, err
}

func actedOf(a mediaservers.Acted, err error) (*actedOutput, error) {
	if err != nil {
		return nil, answer(err)
	}
	out := &actedOutput{Body: acted{Row: a.Row}}
	if a.Mail != nil {
		m := mailapi.OutcomeOf(*a.Mail)
		out.Body.Mail = &m
	}
	return out, nil
}

func (h handlers) listAccess(ctx context.Context, _ identity.Admin, in *accessListInput) (*accessListOutput, error) {
	var f mediaservers.AccessFilter
	if in.Account != "" {
		u, err := parseUUID(in.Account, identity.ErrNoAccount)
		if err != nil {
			return nil, err
		}
		f.Account = identity.AccountID{UUID: u}
	}
	l, err := h.svc().Access(ctx, f)
	if err != nil {
		return nil, answer(err)
	}
	return &accessListOutput{Body: l}, nil
}

func (h handlers) listUsers(ctx context.Context, _ identity.Admin, in *idInput) (*serverUsersOutput, error) {
	id, err := serverID(in.ID)
	if err != nil {
		return nil, err
	}
	us, err := h.svc().LinkableUsers(ctx, id)
	if err != nil {
		return nil, answer(err)
	}
	out := &serverUsersOutput{}
	out.Body.Users = us
	return out, nil
}

func (h handlers) offer(ctx context.Context, by identity.Admin, in *offerInput) (*actedOutput, error) {
	u, err := parseUUID(in.ID, identity.ErrNoAccount)
	if err != nil {
		return nil, err
	}
	server, err := serverID(in.ServerID)
	if err != nil {
		return nil, err
	}
	return actedOf(h.svc().Offer(ctx, by, identity.AccountID{UUID: u}, server, in.Body.PlexHome))
}

func (h handlers) share(ctx context.Context, by identity.Admin, in *shareAccessInput) (*actedOutput, error) {
	id, err := accessID(in.ID)
	if err != nil {
		return nil, err
	}
	return actedOf(h.svc().Share(ctx, by, id, mediaservers.Invite(in.Body.As)))
}

func (h handlers) markAdded(ctx context.Context, by identity.Admin, in *accessIDInput) (*actedOutput, error) {
	id, err := accessID(in.ID)
	if err != nil {
		return nil, err
	}
	return actedOf(h.svc().MarkAdded(ctx, by, id))
}

func (h handlers) linkUser(ctx context.Context, by identity.Admin, in *linkUserInput) (*actedOutput, error) {
	id, err := accessID(in.ID)
	if err != nil {
		return nil, err
	}
	return actedOf(h.svc().LinkUser(ctx, by, id, in.Body.UserID))
}

func (h handlers) checkShare(ctx context.Context, by identity.Admin, in *accessIDInput) (*actedOutput, error) {
	id, err := accessID(in.ID)
	if err != nil {
		return nil, err
	}
	return actedOf(h.svc().CheckShare(ctx, by, id))
}

func (h handlers) removeAccess(ctx context.Context, by identity.Admin, in *accessIDInput) (*actedOutput, error) {
	id, err := accessID(in.ID)
	if err != nil {
		return nil, err
	}
	return actedOf(h.svc().RemoveAccess(ctx, by, id))
}

func (h handlers) replacePlexAccount(ctx context.Context, _ identity.Admin, in *pinIDInput) (*accountOutput, error) {
	a, err := h.svc().ReplacePlexAccount(ctx, in.ID)
	if err != nil {
		return nil, answer(err)
	}
	return &accountOutput{Body: a}, nil
}

// accessRoutes registers the Admin's operations on Members' access.
func accessRoutes(o *api.Owner, h handlers) {
	const base = "/media-servers"
	row := []*api.Code{accessNotFound, accessState}
	plex := []*api.Code{plexNotSignedIn, plexTVUnreached}
	api.Register(o, api.Admin, api.Op{
		ID: "listMediaServerAccess", Method: http.MethodGet, Path: base + "/access", Status: http.StatusOK,
		Summary: "List Members' access to Media Servers, the rows waiting on an Admin first",
		Errors:  building(accountNotFound),
	}, h.listAccess)
	api.Register(o, api.Admin, api.Op{
		ID: "listMediaServerUsers", Method: http.MethodGet, Path: base + "/{id}/users", Status: http.StatusOK,
		Summary: "List the users of a Jellyfin or Emby server that an Admin may link to a Member",
		Errors:  building(serverNotFound, unreached),
	}, h.listUsers)
	api.Register(o, api.Reconfirmed, api.Op{
		ID: "offerMediaServer", Method: http.MethodPut, Path: "/accounts/{id}/media-servers/{serverId}", Status: http.StatusOK,
		Summary: "Offer a Member access to a Media Server, or offer it again after the access was removed",
		Errors:  building(accountNotFound, serverNotFound, notShownToMembers, plexHomeNotPlex, accessState, unreached),
	}, h.offer)
	api.Register(o, api.ReconfirmedSession, api.Op{
		ID: "shareMediaServerAccess", Method: http.MethodPost, Path: base + "/access/{id}/share", Status: http.StatusOK,
		Summary: "Share a Plex server with a Member who signed in to Plex, as a friend or in the Plex Home",
		Errors:  building(slices.Concat(row, plex, []*api.Code{plexRefused, plexLibsShort})...),
	}, h.share)
	api.Register(o, api.Reconfirmed, api.Op{
		ID: "markMediaServerAccessAdded", Method: http.MethodPost, Path: base + "/access/{id}/mark-added", Status: http.StatusOK,
		Summary: "Record that an Admin shared a Plex server with a Member in Plex itself",
		Errors:  building(row...),
	}, h.markAdded)
	api.Register(o, api.ReconfirmedSession, api.Op{
		ID: "linkMediaServerUser", Method: http.MethodPost, Path: base + "/access/{id}/link", Status: http.StatusOK,
		Summary: "Link a Member to an existing user on a Jellyfin or Emby server",
		Errors:  building(slices.Concat(row, []*api.Code{unreached})...),
	}, h.linkUser)
	api.Register(o, api.Admin, api.Op{
		ID: "checkMediaServerAccess", Method: http.MethodPost, Path: base + "/access/{id}/check", Status: http.StatusOK,
		Summary: "Check whether a Member accepted a Plex share",
		Errors:  building(slices.Concat(row, plex)...),
	}, h.checkShare)
	api.Register(o, api.Reconfirmed, api.Op{
		ID: "removeMediaServerAccess", Method: http.MethodDelete, Path: base + "/access/{id}", Status: http.StatusOK,
		Summary: "Remove a Member's access to a Media Server, on the server or plex.tv too",
		Errors:  building(row...),
	}, h.removeAccess)
	api.Register(o, api.Reconfirmed, api.Op{
		ID: "replacePlexAccount", Method: http.MethodPost, Path: base + "/plex/pin/{id}/replace", Status: http.StatusOK,
		Summary: "Sign SlipStream in to the different Plex account a Plex sign-in names, which removes every Plex server and Members' access to them",
		Errors:  building(plexPinNotFound),
	}, h.replacePlexAccount)
}

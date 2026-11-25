package mediaserversapi

import (
	"context"

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

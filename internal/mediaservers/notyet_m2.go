package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

func (service) LinkUser(context.Context, identity.Admin, AccessID, string) (Acted, error) {
	return Acted{}, ErrNotYet
}

func (service) LinkableUsers(context.Context, link.ServerID) ([]ServerUser, error) {
	return nil, ErrNotYet
}

func (service) MakeAccount(context.Context, identity.Account, AccessID, string, link.Password) (MyAccessRow, error) {
	return MyAccessRow{}, ErrNotYet
}

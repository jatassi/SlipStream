package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
)

func (service) Share(context.Context, identity.Admin, AccessID, Invite) (Acted, error) {
	return Acted{}, ErrNotYet
}

func (service) MarkAdded(context.Context, identity.Admin, AccessID) (Acted, error) {
	return Acted{}, ErrNotYet
}

func (service) CheckShare(context.Context, identity.Admin, AccessID) (Acted, error) {
	return Acted{}, ErrNotYet
}

func (service) CheckMine(context.Context, identity.Account, AccessID) (MyAccessRow, error) {
	return MyAccessRow{}, ErrNotYet
}

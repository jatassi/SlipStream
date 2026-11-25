package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

func (service) Access(context.Context, AccessFilter) (AccessList, error) {
	return AccessList{}, ErrNotYet
}

func (service) Offer(context.Context, identity.Admin, identity.AccountID, link.ServerID, bool) (Acted, error) {
	return Acted{}, ErrNotYet
}

func (service) RemoveAccess(context.Context, identity.Admin, AccessID) (Acted, error) {
	return Acted{}, ErrNotYet
}

func (service) Mine(context.Context, identity.Account) (MyAccess, error) {
	return MyAccess{}, ErrNotYet
}

// patchMembers saves the Members fields of a Patch.
func patchMembers(p Patch) error {
	if p.MembersSee == nil && p.MemberLibraries == nil && p.MemberAddress == nil {
		return nil
	}
	return ErrNotYet
}

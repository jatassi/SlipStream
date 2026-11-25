package mediaservers

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
)

func (service) ReplacePlexAccount(context.Context, int64) (AccountView, error) {
	return AccountView{}, ErrNotYet
}

func (service) StartMemberPin(context.Context, identity.Account) (MemberPin, error) {
	return MemberPin{}, ErrNotYet
}

func (service) MemberPin(context.Context, identity.Account, int64) (MemberPinState, error) {
	return MemberPinState{}, ErrNotYet
}

func (service) AnotherPlex(context.Context, identity.Account) error { return ErrNotYet }

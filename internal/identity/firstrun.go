package identity

import (
	"context"

	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
)

// AddressChanged reaches Admins when the address is set.
type AddressChanged struct{}

var AddressChangedEvent = live.AdminEvent[AddressChanged]("identity.address-changed")

// AddressStep is first setup's address step, done once the address is set.
func (s *Service) AddressStep() firstrun.Step {
	return firstrun.Step{ID: firstrun.AddressID, Done: addressSet, Events: []live.Event{AddressChangedEvent}}
}

// PasskeyStep is first setup's passkey step, done while any enabled Admin
// has a passkey at any address, so every viewer reads the same state.
func (s *Service) PasskeyStep() firstrun.Step {
	return firstrun.Step{ID: firstrun.PasskeyID, Done: anyAdminHasPasskey, Events: []live.Event{PasskeysChangedEvent, AccountsChangedEvent}}
}

func addressSet(ctx context.Context, rd storage.Reader[file]) (bool, error) {
	set, err := readSettings(ctx, db.New(rd))
	return !set.Address.IsZero(), err
}

func anyAdminHasPasskey(ctx context.Context, rd storage.Reader[file]) (bool, error) {
	return db.New(rd).AnyAdminHasPasskey(ctx)
}

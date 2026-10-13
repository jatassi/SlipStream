package identitytest

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/system"
)

// Node is a Node whose library.db is open from the start: Unclaimed until
// its library holds an Admin, a Controller after.
type Node struct{ Service *identity.Service }

func (n Node) Claim(ctx context.Context) (system.Claim, error) {
	admin, err := n.Service.HasAdmin(ctx)
	if err != nil || !admin {
		return system.Unclaimed, err
	}
	return system.Controller, nil
}

func (n Node) Identity() (*identity.Service, bool) { return n.Service, true }

func (n Node) ClaimController(_ context.Context, claim func(*identity.Service) error) error {
	return claim(n.Service)
}

// Controller is a Node that is a Controller whatever its library holds, for
// tests that add Members without an Admin.
type Controller struct{ Node }

func (Controller) Claim(context.Context) (system.Claim, error) { return system.Controller, nil }

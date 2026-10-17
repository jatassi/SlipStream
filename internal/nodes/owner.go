package nodes

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Owner is the feature that keeps a Role's work. Every Role in the table
// has exactly one, so the roster and the plan can count what a Node holds
// for it.
type Owner struct {
	Role role.Name
	// Held is the work node holds for Role. The Role's Work says what that
	// work does when the Node lets the Role go: tied work drains there, and
	// portable work moves to the heir. It reads the library and nothing else.
	Held func(ctx context.Context, r storage.Reader[storage.Library], node ID) (Held, error)
}

// HoldsNothing is the Held of a Role whose work no feature keeps yet.
func HoldsNothing(context.Context, storage.Reader[storage.Library], ID) (Held, error) {
	return Held{}, nil
}

// Unowned is the owner of a Role whose work a later spec adds.
func Unowned(r role.Name) Owner { return Owner{Role: r, Held: HoldsNothing} }

// ownersByRole is owners keyed by Role. It panics unless each Role of the
// table has exactly one owner.
func ownersByRole(owners []Owner) map[role.Name]Owner {
	out := map[role.Name]Owner{}
	for _, o := range owners {
		if _, err := role.ParseName(string(o.Role)); err != nil {
			panic("nodes: an owner for " + err.Error())
		}
		if _, dup := out[o.Role]; dup {
			panic(fmt.Sprintf("nodes: the %s Role has two owners", o.Role))
		}
		out[o.Role] = o
	}
	for _, r := range role.Names() {
		if _, ok := out[r]; !ok {
			panic(fmt.Sprintf("nodes: the %s Role has no owner; give it a nodes.Owner in node.owners", r))
		}
	}
	return out
}

// countWork reads which Roles each Node holds tied work for, among the
// Roles not assigned to it or written off there.
func countWork(ctx context.Context, r storage.Reader[storage.Library], owners map[role.Name]Owner, nodes []nodeIn) (map[ID]role.Set, error) {
	out := map[ID]role.Set{}
	for _, n := range nodes {
		for _, name := range role.Names() {
			if name.Rule().Work != role.Tied || n.assigned.Has(name) && !n.forfeit.Has(name) {
				continue
			}
			held, err := owners[name].Held(ctx, r, n.id)
			if err != nil {
				return nil, fmt.Errorf("count %s work on node %s: %w", name, n.id, err)
			}
			if held.Units > 0 {
				out[n.id] = out[n.id].With(name)
			}
		}
	}
	return out, nil
}

// Package nodestest gives tests that open a roster what nodes.Open asks
// for.
package nodestest

import (
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
)

// Unowned is an owner for every Role, each holding nothing.
func Unowned() []nodes.Owner {
	out := make([]nodes.Owner, 0, len(role.Names()))
	for _, r := range role.Names() {
		out = append(out, nodes.Unowned(r))
	}
	return out
}

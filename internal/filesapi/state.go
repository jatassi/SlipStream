package filesapi

import (
	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
)

// State is a cell's state, or the status an Item or a Work rolls up to.
// Every wire state names this one schema, so the web's records over it
// are exhaustive.
type State struct{}

func (State) Values() []string { return files.States() }

// StatusOf is s on the wire, null for zero: no cell to roll up.
func StatusOf(s files.State) api.Nullable[api.Enum[State]] {
	if s == 0 {
		return api.Nullable[api.Enum[State]]{}
	}
	return api.Some(api.NewEnum[State](s.String()))
}

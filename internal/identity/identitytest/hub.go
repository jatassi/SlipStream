package identitytest

import (
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
)

// Hub is a hub carrying identity's events, closed when t ends.
func Hub(t testing.TB) *live.Hub {
	h := live.NewHub([]live.Event{identity.SessionsChangedEvent})
	t.Cleanup(h.Close)
	return h
}

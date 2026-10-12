// Package surface is the whole API: every owner's operations and every
// declared event. It never imports internal/node, which links libtorrent,
// so the document builds without cgo.
package surface

import (
	"context"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identityapi"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/liveapi"
	"github.com/jatassi/slipstream/internal/system"
	"github.com/jatassi/slipstream/internal/systemapi"
)

// Events lists every declared event, one line each. The hub, the document
// and every Bind read it.
var Events = []live.Event{
	identity.SessionsChangedEvent,
}

// Services are what the operations run against. Document passes none.
type Services struct {
	Identity *identity.Service
	Hub      *live.Hub
	Live     live.Timings
	System   system.Facts
	// Claimed is the composition root's one definition of a claimed Node.
	Claimed func(context.Context) (bool, error)
}

// Mount registers every owner's operations, one line each.
func Mount(r *api.Router, s Services) {
	identityapi.Routes(r, s.Identity)
	liveapi.Routes(r, s.Hub, s.Identity, s.Live)
	systemapi.Routes(r, s.System, s.Claimed)
}

// Document is the OpenAPI document. Registering only captures each
// handler, so none runs and no service is needed.
func Document() []byte {
	r := api.NewRouter(api.Services{})
	Mount(r, Services{Hub: live.NewHub(Events)})
	return r.Document()
}

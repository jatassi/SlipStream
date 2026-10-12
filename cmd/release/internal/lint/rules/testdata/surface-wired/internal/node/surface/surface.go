package surface

import (
	"example.com/app/internal/identity"
	ident "example.com/app/internal/identity/identityapi"
	"example.com/app/internal/live"
	"example.com/app/internal/live/liveapi"
	"example.com/app/internal/nodes"
	"example.com/app/internal/nodes/nodesapi"
)

func Mount() {
	liveapi.Routes()
	ident.Routes()
}

func mountLater() {
	nodesapi.Routes()
}

var Events = []live.Event{
	identity.SessionsChangedEvent,
	nodes.ListedEvent,
}

var others = []live.Event{
	nodes.ChangedEvent,
}

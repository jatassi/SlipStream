package nodes

import (
	"example.com/app/internal/live"
	events "example.com/app/internal/live"
)

type (
	Changed struct{}
	Listed  struct{}
	Removed struct{}
)

var (
	ListedEvent  = live.EveryoneEvent[Listed]("nodes.listed")
	ChangedEvent = live.AdminEvent[Changed]("nodes.changed")
)

var RemovedEvent, count = events.AccountEvent[Removed]("nodes.removed"), 0

func Publish() {
	local := live.AdminEvent[Changed]("nodes.local")
	_ = local
}

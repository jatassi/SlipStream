package nodes

import "example.com/app/internal/live"

type planted struct{}

var plantedEvent = live.AdminEvent[planted]("nodes.planted")

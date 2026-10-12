package media

import "example.com/vendor/live"

type Changed struct{}

var ChangedEvent = live.AdminEvent[Changed]("media.changed")

package mediaserversapi

import (
	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/live"
)

// Changed names the servers whose record, mappings or changes changed.
type Changed struct {
	ServerIDs []string `json:"serverIds" nullable:"false" format:"uuid"`
}

// SessionsChanged names the servers whose sessions started, changed or
// ended.
type SessionsChanged struct {
	ServerIDs []string `json:"serverIds" nullable:"false" format:"uuid"`
}

// WentUnhealthy is a server becoming unhealthy, for its reason.
type WentUnhealthy struct {
	ServerID string         `json:"serverId" format:"uuid"`
	Kind     api.Enum[Kind] `json:"kind"`
	Reason   string         `json:"reason" enum:"not_responding,sends_failing,sign_in_refused,other_server"`
}

// Recovered is a server no longer unhealthy for the reason it was.
type Recovered struct {
	ServerID string         `json:"serverId" format:"uuid"`
	Kind     api.Enum[Kind] `json:"kind"`
	Reason   string         `json:"reason" enum:"not_responding,sends_failing,sign_in_refused,other_server"`
}

// Found names the Works whose Open in links a lookup filled in.
type Found struct {
	WorkIDs []string `json:"workIds" nullable:"false" format:"uuid"`
}

var (
	ChangedEvent         = live.AdminEvent[Changed]("media-servers.changed")
	SessionsChangedEvent = live.AdminEvent[SessionsChanged]("media-servers.sessions-changed")
	UnhealthyEvent       = live.AdminEvent[WentUnhealthy]("media-servers.unhealthy")
	RecoveredEvent       = live.AdminEvent[Recovered]("media-servers.recovered")
	ItemsFoundEvent      = live.AdminEvent[Found]("media-servers.items-found")
)

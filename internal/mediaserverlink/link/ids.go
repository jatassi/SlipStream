package link

import "github.com/jatassi/slipstream/internal/storage"

// ServerID is mediaservers_server.id.
type ServerID struct{ storage.UUID }

// ChangeID is mediaservers_change.id, a UUIDv7, so ids sort in created
// order.
type ChangeID struct{ storage.UUID }

// PlayID is mediaservers_play.id. The holder mints it when a session
// starts, so every report about that play names one row.
type PlayID struct{ storage.UUID }

// ServerRef names a server as a Job carries it.
type ServerRef struct {
	ID   ServerID
	Kind KindName
	// Identity is the machineIdentifier or server Id SlipStream saved. A
	// different one at the address is ReasonOtherServer.
	Identity        string
	SettingsVersion int64
}

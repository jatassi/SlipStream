package identity

import "example.com/app/internal/live"

type SessionsChanged struct{}

var SessionsChangedEvent = live.AccountEvent[SessionsChanged]("identity.sessions-changed")

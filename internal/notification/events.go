package notification

import (
	"github.com/slipstream/slipstream/internal/module"
)

// EventType identifies the type of notification event.
type EventType = string

// Framework event IDs (re-exported from module package for convenience).
const (
	EventGrab           EventType = module.EventGrab
	EventImport         EventType = module.EventImport
	EventUpgrade        EventType = module.EventUpgrade
	EventHealthIssue    EventType = module.EventHealthIssue
	EventHealthRestored EventType = module.EventHealthRestored
	EventAppUpdate      EventType = module.EventAppUpdate
)

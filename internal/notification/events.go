package notification

import (
	"github.com/slipstream/slipstream/internal/module"
	moviemod "github.com/slipstream/slipstream/internal/modules/movie"
	tvmod "github.com/slipstream/slipstream/internal/modules/tv"
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

// Module event IDs (re-exported from module packages for convenience).
const (
	EventMovieAdded    EventType = moviemod.EventMovieAdded
	EventMovieDeleted  EventType = moviemod.EventMovieDeleted
	EventSeriesAdded   EventType = tvmod.EventTVAdded
	EventSeriesDeleted EventType = tvmod.EventTVDeleted
)

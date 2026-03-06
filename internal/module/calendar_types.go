package module

import "time"

// CalendarItem represents a single item with a date for the unified calendar view.
type CalendarItem struct {
	ID         int64
	Title      string
	ModuleType Type
	EntityType EntityType
	EventType  string
	Date       time.Time
	Status     string
	Monitored  bool

	ExternalIDs map[string]string
	Year        int
	ParentID    int64
	ParentTitle string
	Extra       map[string]any
}

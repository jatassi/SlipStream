package module

import "time"

// CalendarItem represents an item with a date for the calendar view.
type CalendarItem struct {
	EntityType EntityType
	EntityID   int64
	Title      string
	Date       time.Time
	ModuleType Type
}

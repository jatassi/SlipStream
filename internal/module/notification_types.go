package module

// NotificationEvent declares a notification event.
type NotificationEvent struct {
	ID          string
	Label       string
	Description string
}

// NotificationEventGroup is a labeled group of events for UI rendering.
type NotificationEventGroup struct {
	ID     string // "framework" or module type ID
	Label  string // "General" or module display name
	Events []NotificationEvent
}

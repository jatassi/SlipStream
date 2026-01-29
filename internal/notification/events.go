package notification

// EventType identifies the type of notification event
type EventType string

const (
	EventGrab           EventType = "grab"
	EventImport       EventType = "download"
	EventUpgrade        EventType = "upgrade"
	EventMovieAdded     EventType = "movie_added"
	EventMovieDeleted   EventType = "movie_deleted"
	EventSeriesAdded    EventType = "series_added"
	EventSeriesDeleted  EventType = "series_deleted"
	EventHealthIssue    EventType = "health_issue"
	EventHealthRestored EventType = "health_restored"
	EventAppUpdate      EventType = "app_update"
)

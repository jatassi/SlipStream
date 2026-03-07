package module

// Framework notification event IDs.
const (
	EventGrab           = "grab"
	EventImport         = "import"
	EventUpgrade        = "upgrade"
	EventHealthIssue    = "health_issue"
	EventHealthRestored = "health_restored"
	EventAppUpdate      = "app_update"
)

// FrameworkNotificationEvents returns the events declared by the framework itself
// (not tied to any specific module).
func FrameworkNotificationEvents() []NotificationEvent {
	return []NotificationEvent{
		{ID: EventGrab, Label: "On Grab", Description: "When a release is grabbed from an indexer"},
		{ID: EventImport, Label: "On Import", Description: "When a file is imported to the library"},
		{ID: EventUpgrade, Label: "On Upgrade", Description: "When a quality upgrade is imported"},
		{ID: EventHealthIssue, Label: "On Health Issue", Description: "When a health check fails"},
		{ID: EventHealthRestored, Label: "On Health Restored", Description: "When a health issue is resolved"},
		{ID: EventAppUpdate, Label: "On App Update", Description: "When the application is updated"},
	}
}

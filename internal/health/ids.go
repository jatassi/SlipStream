package health

// One const per category. A later spec adds one here and its binding in
// internal/node/health.go; Open panics unless the two match.
const (
	StorageID         ID = "storage"
	LibraryID         ID = "library"
	NodesID           ID = "nodes"
	DownloadClientsID ID = "download-clients"
	MediaServersID    ID = "media-servers"
	SeedingID         ID = "seeding"
	IndexersID        ID = "indexers"
	ScheduledTasksID  ID = "scheduled-tasks"
)

// IDs is every category, in page order.
func IDs() []ID {
	return []ID{StorageID, LibraryID, NodesID, DownloadClientsID, MediaServersID, SeedingID, IndexersID, ScheduledTasksID}
}

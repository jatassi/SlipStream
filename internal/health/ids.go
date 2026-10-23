package health

// One const per category. A later spec adds one here and its binding in
// internal/node/health.go; Open panics unless the two match.
const (
	StorageID        ID = "storage"
	NodesID          ID = "nodes"
	ScheduledTasksID ID = "scheduled-tasks"
)

// IDs is every category, in page order.
func IDs() []ID { return []ID{StorageID, NodesID, ScheduledTasksID} }

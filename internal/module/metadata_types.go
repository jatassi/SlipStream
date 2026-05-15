package module

// RefreshResult contains the structured diff from a metadata refresh operation.
type RefreshResult struct {
	EntityID        int64
	Updated         bool
	FieldsChanged   []string
	ChildrenAdded   []RefreshChildEntry
	ChildrenUpdated []RefreshChildEntry
	ChildrenRemoved []RefreshChildEntry
}

// RefreshChildEntry represents a child entity in a refresh diff.
type RefreshChildEntry struct {
	EntityType EntityType
	Identifier string
	EntityID   int64
	Title      string
}

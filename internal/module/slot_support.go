package module

// SlotSupport is an optional module interface for multi-version slot support.
// Modules that implement this interface can have files assigned to version slots.
// (spec S2.5)
type SlotSupport interface {
	// SlotEntityType returns the leaf entity type name used in slot assignment tables.
	// E.g., "movie" for movies, "episode" for TV episodes.
	SlotEntityType() string
}

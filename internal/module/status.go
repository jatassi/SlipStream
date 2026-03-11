package module

// Status constants for media entities.
const (
	StatusUnreleased  = "unreleased"
	StatusMissing     = "missing"
	StatusDownloading = "downloading"
	StatusFailed      = "failed"
	StatusUpgradable  = "upgradable"
	StatusAvailable   = "available"
)

// statusPriority defines the priority order for status aggregation.
// Higher value = higher priority (wins in aggregation).
var statusPriority = map[string]int{
	StatusUnreleased:  0,
	StatusAvailable:   1,
	StatusUpgradable:  2,
	StatusMissing:     3,
	StatusFailed:      4,
	StatusDownloading: 5,
}

// AggregateStatus computes a parent node's status from its children's statuses.
// Uses the priority rule: downloading > failed > missing > upgradable > available > unreleased.
// Returns StatusUnreleased if no children are provided.
func AggregateStatus(childStatuses []string) string {
	if len(childStatuses) == 0 {
		return StatusUnreleased
	}

	highest := StatusUnreleased
	highestPrio := statusPriority[highest]

	for _, s := range childStatuses {
		p, ok := statusPriority[s]
		if ok && p > highestPrio {
			highest = s
			highestPrio = p
		}
	}

	return highest
}

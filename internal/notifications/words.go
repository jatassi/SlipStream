package notifications

// healthWords is every label and sentence that names a Health item, in one
// table while the copy is decided. The settings label, the catalog labels
// and the renders read it, and no web string names a Health item.
var healthWords = struct {
	NewLabel     string
	ClearedLabel string
	WaitLabel    string
	// NewMany takes the count.
	NewMany string
	// ClearedMany takes the count.
	ClearedMany string
	// ClearedOne takes the item's subject.
	ClearedOne string
	// OnHealthSince takes the clock time it appeared.
	OnHealthSince string
	// OnHealthFrom takes the clock times it appeared and cleared.
	OnHealthFrom string
}{
	NewLabel:      "New Health item",
	ClearedLabel:  "Health item cleared",
	WaitLabel:     "Wait before sending a new Health item",
	NewMany:       "%d new Health items",
	ClearedMany:   "%d Health items cleared",
	ClearedOne:    "Cleared: %s",
	OnHealthSince: "It has been on Health since %s.",
	OnHealthFrom:  "It was on Health from %s to %s.",
}

package link

import "time"

// Job is one unit of work on one server. The Controller plans it and
// persists its step on the change rows. Advance runs one step of it,
// wherever the route is, and answers it one step on.
type Job struct {
	Server ServerRef
	Work   Work
	Step   Step
	// At is when Step was reached, zero at StepNone.
	At time.Time
}

// Work is what a Job does: Refresh, Carry, Fresh, Rescan, Lookup, Index or
// Libraries.
type Work interface{ work() }

// Refresh tells the server about changes at paths it can refresh in place:
// added, replaced, removed and the Sidecar kinds.
type Refresh struct{ Changes []JobChange }

// Carry is a group of moved changes, and replaced changes under a new name,
// that must keep their items' watch data and collections.
type Carry struct{ Changes []JobChange }

// Fresh is the fresh folder rule for one mapping, from its columns. The
// Job is done once the mapping counts as scanned.
type Fresh struct {
	Library string
	Folder  ServerPath
	// Requested is when the rule decided on a full scan, written before
	// the scan is sent. Zero: not decided.
	Requested time.Time
	// Started is the server's last scan end before that scan, on its own
	// clock, zero when it never scanned. A later end is the scan's end, so
	// a retry never starts a second scan and no skew between clocks fools
	// it (graft 3).
	Started time.Time
	// Changes are the rows that wait on the rule.
	Changes []ChangeID
}

// Rescan refreshes whole libraries: library changes, and Plex's section
// refresh past Timings.SectionFolders.
type Rescan struct {
	Libraries []string
	Changes   []ChangeID
}

// Lookup reads the items at paths, for Open in, Sidecars and stale index
// entries.
type Lookup struct{ Paths []ServerPath }

// Index reads one page of a library's items for the path index. Cursor ""
// is the first page. The Job answered carries the next cursor, "" after the
// last page.
type Index struct {
	Library string
	Cursor  string
}

// Libraries lists the server's libraries and their folders.
type Libraries struct{}

func (Refresh) work()   {}
func (Carry) work()     {}
func (Fresh) work()     {}
func (Rescan) work()    {}
func (Lookup) work()    {}
func (Index) work()     {}
func (Libraries) work() {}

// JobChange is one change row as a step sees it. Its server paths were
// joined on the Controller.
type JobChange struct {
	ID      ChangeID
	Kind    ChangeKind
	Library string
	Path    ServerPath
	// From is zero unless the change is a carry.
	From ServerPath
	// Snapshot is set once, at StepSnapshotted, and nil while the server
	// held no item at From.
	Snapshot *Snapshot
}

// Library is one Plex section, or one Jellyfin or Emby library.
type Library struct {
	// Key is Plex's section key, or the Jellyfin or Emby library ItemId.
	Key     string
	Name    string
	Folders []ServerPath
}

// Snapshot is what a carry keeps: the old item and every user's data and
// collections on it. On Plex Users holds the account owner's entry only.
type Snapshot struct {
	Item        ItemKey    `json:"item"`
	Users       []UserData `json:"users,omitempty"`
	Collections []string   `json:"collections,omitempty"`
}

// ItemKey is an item on a server: a ratingKey or an item Id. Top is the
// show item (grandparentRatingKey, SeriesId) when the item has one, else
// Item.
type ItemKey struct {
	Item string `json:"item"`
	Top  string `json:"top"`
}

type UserData struct {
	User       string        `json:"user"`
	Played     bool          `json:"played,omitempty"`
	PlayCount  int           `json:"playCount,omitempty"`
	Position   time.Duration `json:"position,omitempty"`
	Favorite   bool          `json:"favorite,omitempty"`
	LastPlayed time.Time     `json:"lastPlayed,omitzero"`
	// Rating is the user's rating out of 10, 0 for none.
	Rating float64 `json:"rating,omitempty"`
}

// Advanced is one step's answer.
type Advanced struct {
	// Job is the Job at its new step, with any snapshot it took and the
	// changes that haven't ended.
	Job  Job
	Done bool
	// Wait is when to call again, zero for now.
	Wait time.Time
	// Outcomes are the changes that ended at this step.
	Outcomes  []Outcome
	Learned   []IndexEntry
	Libraries []Library
}

type Outcome struct {
	Change ChangeID
	Result Result
	Lost   []Loss
}

// IndexEntry is what a step learned about one path: the item there, or
// that none is.
type IndexEntry struct {
	Path    ServerPath
	Library string
	Item    ItemKey
	Gone    bool
}

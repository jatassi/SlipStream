package imports

import (
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type ReasonCode string

// A root Import's file waits for S22 (Library scan and file probe)'s
// scanning.Reason codes, carried as they are. The codes below are the ones
// a download or a manual import reads.
const (
	Sample           ReasonCode = "sample"
	Bonus            ReasonCode = "bonus"
	HiddenFolder     ReasonCode = "hidden_folder"
	TooSmall         ReasonCode = "too_small"
	NotAVideoType    ReasonCode = "extension"
	RuntimeSample    ReasonCode = "runtime_sample"
	SiblingBetter    ReasonCode = "sibling_better"
	KeptBetter       ReasonCode = "kept_better"
	MeetsNothing     ReasonCode = "meets_nothing"
	NotAnUpgrade     ReasonCode = "not_an_upgrade"
	EditionNotWanted ReasonCode = "edition_not_wanted"
	OtherItems       ReasonCode = "other_items"
	NoNumber         ReasonCode = "no_number"
	Parts            ReasonCode = "parts"
	NameTaken        ReasonCode = "name_taken"
	NameTooLong      ReasonCode = "name_too_long"
	Settings         ReasonCode = "settings"
	Busy             ReasonCode = "busy"
	Confirm          ReasonCode = "confirm"
	Unreadable       ReasonCode = "unreadable"
	ImportFail       ReasonCode = "import_failed"
	RootChanged      ReasonCode = "root_changed"
	ByAdmin          ReasonCode = "by_admin"
	Removed          ReasonCode = "removed"
	Canceled         ReasonCode = "canceled"
	Blocklisted      ReasonCode = "blocklisted"
	WorkDeleted      ReasonCode = "work_deleted"
)

// Reason is why a file is left out, waits or failed. It is stored as JSON
// with the fields its code reads, and worded on read. Scan is a root
// Import's S22 reason, and Code is then its code.
type Reason struct {
	Code ReasonCode       `json:"code"`
	Scan *scanning.Reason `json:"scan,omitempty"`
	// Targets are not_an_upgrade's lost cells' Targets and meets_nothing's
	// Targets, and confirm's filled ones.
	Targets []targets.TargetID `json:"targets,omitempty"`
	// Reading is what the file's name reads as, for other_items and busy,
	// and Of the Grab's Items, for other_items.
	Reading []works.ItemID `json:"reading,omitempty"`
	Of      []works.ItemID `json:"of,omitempty"`
	// Level is the level no_number's folder reads as.
	Level works.LevelID `json:"level,omitzero"`
	// Name is name_taken's name and name_too_long's segment, Length its
	// length and Limit the Node's.
	Name   string `json:"name,omitempty"`
	Length int    `json:"length,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	// Setting is the setting settings names, and Value its value.
	Setting string `json:"setting,omitempty"`
	Value   int64  `json:"value,omitempty"`
	// Edition is edition_not_wanted's cut.
	Edition release.CutKey `json:"edition,omitempty"`
	// Parts is how many files parts splits the Work's one file into.
	Parts int `json:"parts,omitempty"`
	// Line is import_failed's and unreadable's first line.
	Line string `json:"line,omitempty"`
}

type HoldCode string

const (
	NoSpace      HoldCode = "no_space"
	RootNotReady HoldCode = "root_not_ready"
	NodeAway     HoldCode = "node_away"
	ProbeTimeout HoldCode = "probe_timeout"
	ProbeError   HoldCode = "probe_error"
)

// Hold is an import wait: retried automatically, never failed by time.
// Free and Needs are a no_space hold's bytes.
type Hold struct {
	Code     HoldCode     `json:"code"`
	Attempts int          `json:"attempts"`
	Free     int64        `json:"free,omitempty"`
	Needs    int64        `json:"needs,omitempty"`
	Since    storage.Time `json:"since"`
}

// offered reports whether h shows Retry and Don't import: after three
// probe timeouts in a row.
func (h Hold) offered() bool { return h.Code == ProbeTimeout && h.Attempts >= 3 }

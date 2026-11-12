package imports

import (
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
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
// and worded on read. Scan is a root Import's S22 reason, and Code is then
// its code.
type Reason struct {
	Code ReasonCode       `json:"code"`
	Scan *scanning.Reason `json:"scan,omitempty"`
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

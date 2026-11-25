package reachability

import "github.com/jatassi/slipstream/internal/live"

// Changed names the Exposures whose rows one Write changed; empty means
// any may have.
type Changed struct {
	ExposureIDs []string `json:"exposureIds" nullable:"false" format:"uuid"`
}

var ChangedEvent = live.AdminEvent[Changed]("reachability.changed")

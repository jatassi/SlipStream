package conversions

import "github.com/jatassi/slipstream/internal/live"

var (
	ChangedEvent = live.AdminEvent[Changed]("conversions.changed")
	LiveEvent    = live.AdminEvent[LiveSample]("conversions.live")
)

// Changed is conversions.changed's payload: the Conversions, Works and
// groups whose rows moved.
type Changed struct {
	ConversionIDs []string `json:"conversionIds" nullable:"false"`
	WorkIDs       []string `json:"workIds" nullable:"false"`
	GroupIDs      []string `json:"groupIds" nullable:"false"`
}

// LiveSample is conversions.live's payload: the runs that moved, and the
// Conversions running per Node.
type LiveSample struct {
	Rows  []LiveRow  `json:"rows" nullable:"false"`
	Nodes []LiveNode `json:"nodes" nullable:"false"`
}

type LiveRow struct {
	ID          string  `json:"id"`
	Percent     float64 `json:"percent"`
	Speed       float64 `json:"speed"`
	SecondsLeft int64   `json:"secondsLeft"`
	Paused      bool    `json:"paused"`
}

type LiveNode struct {
	ID      string `json:"id"`
	Running int    `json:"running"`
}

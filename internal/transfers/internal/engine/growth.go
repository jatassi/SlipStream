package engine

import (
	"time"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
)

// growth is how many lanes a route keeps open, as the receiver judges it
// from the bytes that arrive. It starts at LanesStart and, each LanesGrow, adds a lane while the last one
// added raised the rate by at least LanesGain, up to LanesMax. A lane that
// didn't is closed again and the count settles, because every lane is
// another share of a line the Transfer may share. It doesn't add while
// the send limit holds the rate.
type growth struct {
	lanes int
	// before is the rate over the window before the last add, 0 until the
	// first.
	before  float64
	settled bool
}

// startGrowth is a run's growth on route. Route 3's streams share one TCP
// connection and the no-route wait only probes, so both keep one lane.
func startGrowth(route tv1.Route, t Timings) growth {
	if route != tv1.Route_ROUTE_RECEIVER_DIALS && route != tv1.Route_ROUTE_SENDER_DIALS {
		return growth{lanes: 1, settled: true}
	}
	return growth{lanes: t.LanesStart}
}

// next judges one window of LanesGrow at g.lanes lanes, in which they
// moved rate bytes a second, and binding says whether the send limit held
// a chunk back. A window that moved nothing says nothing about lanes.
func (g growth) next(rate float64, binding bool, t Timings) growth {
	switch {
	case g.settled || binding || rate <= 0:
		return g
	case g.before > 0 && rate*100 < g.before*float64(100+t.LanesGain):
		return growth{lanes: g.lanes - 1, settled: true}
	case g.lanes >= t.LanesMax:
		return growth{lanes: g.lanes, settled: true}
	}
	return growth{lanes: g.lanes + 1, before: rate}
}

// window is the bytes an end's lanes moved since the lane count last
// changed, and whether the send limit bound since.
type window struct {
	since   time.Time
	bytes   int64
	binding bool
}

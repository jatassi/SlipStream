package engine

import (
	"testing"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
)

func TestGrowthAddsLanesWhileEachRaisesTheRate(t *testing.T) {
	timings := Timings{LanesStart: 4, LanesGain: 10, LanesMax: 6}
	type window struct {
		rate    float64
		binding bool
		lanes   int
	}
	for _, c := range []struct {
		name    string
		route   tv1.Route
		windows []window
	}{
		{"each lane adds the gain, up to the most", tv1.Route_ROUTE_RECEIVER_DIALS, []window{
			{rate: 100, lanes: 5}, {rate: 110, lanes: 6}, {rate: 130, lanes: 6}, {rate: 500, lanes: 6},
		}},
		{"a lane that adds less than the gain closes", tv1.Route_ROUTE_SENDER_DIALS, []window{
			{rate: 100, lanes: 5}, {rate: 109, lanes: 4}, {rate: 300, lanes: 4},
		}},
		{"a lane that slows the rate closes", tv1.Route_ROUTE_RECEIVER_DIALS, []window{
			{rate: 100, lanes: 5}, {rate: 80, lanes: 4}, {rate: 300, lanes: 4},
		}},
		{"no lane while the send limit binds", tv1.Route_ROUTE_RECEIVER_DIALS, []window{
			{rate: 100, binding: true, lanes: 4}, {rate: 100, lanes: 5}, {rate: 50, binding: true, lanes: 5}, {rate: 120, lanes: 6},
		}},
		{"a window that moved nothing", tv1.Route_ROUTE_RECEIVER_DIALS, []window{
			{rate: 0, lanes: 4}, {rate: 100, lanes: 5}, {rate: 0, lanes: 5}, {rate: 120, lanes: 6},
		}},
		{"route 3", tv1.Route_ROUTE_NODE_LINK, []window{{rate: 100, lanes: 1}, {rate: 200, lanes: 1}}},
		{"the no-route wait", tv1.Route_ROUTE_BOTH_DIAL, []window{{rate: 100, lanes: 1}, {rate: 200, lanes: 1}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := startGrowth(c.route, timings)
			for i, w := range c.windows {
				g = g.next(w.rate, w.binding, timings)
				if g.lanes != w.lanes {
					t.Fatalf("after window %d at %v bytes a second, %d lanes, want %d", i+1, w.rate, g.lanes, w.lanes)
				}
			}
		})
	}
}

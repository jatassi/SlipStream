package transfers_test

import (
	"context"
	"net"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/time/rate"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link/linktest"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/transfers"
)

// perLane is each lane's rate on the path between the Workers, as a line
// that shapes each flow does.
const perLane = 256 << 10

// growing sets a sender and a receiver to grow from two lanes every
// 4 s up to most, on a path that carries perLane on each lane, and
// returns a Transfer of a twelve-chunk file and a small one.
func growing(t *testing.T, most int) (tr *transfer, sender, receiver *end) {
	t.Helper()
	_, w, nw := trio(t)
	if _, err := nw.Path(linktest.WorkerAddr, linktest.OtherWorkerAddr).Set(fault.Rule{ID: "per-lane", Fault: fault.Throttle(perLane)}); err != nil {
		t.Fatal(err)
	}
	for _, e := range w {
		e.timings.LanesStart, e.timings.LanesGrow, e.timings.LanesMax = 2, 4*time.Second, most
		e.timings.Chunk = 2 * time.Minute
	}
	return newTransferOf(t, w[0], w[1], []int{12 * chunk, chunk + 7}), w[0], w[1]
}

func TestLanesGrowWhileEachAddsSpeedUpToTheMost(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tr, sender, receiver := growing(t, 4)
		lanes := newLaneLog(0)
		sender.hooks.Lane = lanes.wrap
		sender.start()
		receiver.start()
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
		tr.expectArrived()
		if counts := lanes.counts(); slices.Max(counts) != 4 {
			t.Fatalf("lanes open went %v on a path where each lane adds its own speed, want up to lanes_max 4", counts)
		}
	})
}

func TestLaneGrowthStopsAtTheLaneThatAddsNoSpeed(t *testing.T) {
	for _, c := range []struct {
		route tv1.Route
		// accepts is the end that accepts the lanes, where the path plays
		// its faults and so where the line shares them.
		accepts func(sender, receiver *end) *end
	}{
		{tv1.Route_ROUTE_RECEIVER_DIALS, func(sender, _ *end) *end { return sender }},
		{tv1.Route_ROUTE_SENDER_DIALS, func(_, receiver *end) *end { return receiver }},
	} {
		t.Run(c.route.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				tr, sender, receiver := growing(t, 16)
				line := newLaneLog(5 * perLane / 2)
				c.accepts(sender, receiver).hooks.Lane = line.wrap
				sender.start()
				receiver.start()
				tr.route(c.route, 0)
				tr.expectArrived()
				counts := line.counts()
				fourth := slices.Index(counts, 4)
				if fourth < 0 || slices.Max(counts) != 4 || counts[fourth+1] != 3 || slices.Max(counts[fourth+1:]) != 3 {
					t.Fatalf("lanes open went %v on a line two and a half lanes fill, want the fourth lane closed and no other opened", counts)
				}
			})
		})
	}
}

func TestALaneCutAmongOthersLosesNoChunk(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		_, w, nw := trio(t)
		tr := newTransfer(t, w[0], w[1])
		w[1].timings.SyncBytes = chunk
		lanes := newLaneLog(0)
		w[0].hooks.Lane = lanes.wrap
		if _, err := nw.Path(linktest.WorkerAddr, linktest.OtherWorkerAddr).Set(fault.Rule{ID: "cut", Fault: fault.Cut(chunk + chunk/2), Times: 1}); err != nil {
			t.Fatal(err)
		}
		w[0].start()
		w[1].start()
		tr.route(tv1.Route_ROUTE_RECEIVER_DIALS, 0)
		tr.expectArrived()
		if most := slices.Max(lanes.counts()); most < 2 {
			t.Fatalf("at most %d lane was open, want the cut among others", most)
		}
		again := 0
		for k, n := range tr.sent.counts() {
			if n > 2 {
				t.Fatalf("chunk %d of file %d was sent %d times across one cut", k[1], k[0], n)
			}
			if n == 2 {
				again++
			}
		}
		if again == 0 {
			t.Fatal("no chunk was sent again, so the cut never landed mid-chunk")
		}
		if tr.sent.count(0, 0) != 1 {
			t.Fatal("the chunk verified before the cut was sent again")
		}
	})
}

// laneLog is the lanes of one end as the network sees them: how many are
// open on each route, and each change in the count. With a line, every
// lane's bytes share it either way, as flows through one bottleneck do.
type laneLog struct {
	line *rate.Limiter

	mu      sync.Mutex
	open    map[tv1.Route]int
	history []int
}

func newLaneLog(line rate.Limit) *laneLog {
	g := &laneLog{open: map[tv1.Route]int{}}
	if line > 0 {
		g.line = rate.NewLimiter(line, 16<<10)
	}
	return g
}

func (g *laneLog) wrap(_ transfers.ID, route tv1.Route, c net.Conn) net.Conn {
	g.change(route, 1)
	return &loggedLane{Conn: c, log: g, route: route}
}

func (g *laneLog) change(route tv1.Route, by int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.open[route] += by
	n := 0
	for _, k := range g.open {
		n += k
	}
	g.history = append(g.history, n)
}

func (g *laneLog) openOn(route tv1.Route) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.open[route]
}

func (g *laneLog) counts() []int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]int(nil), g.history...)
}

type loggedLane struct {
	net.Conn
	log    *laneLog
	route  tv1.Route
	closed sync.Once
}

func (c *loggedLane) Read(b []byte) (int, error) {
	if c.log.line != nil {
		b = b[:min(len(b), c.log.line.Burst())]
	}
	n, err := c.Conn.Read(b)
	c.share(n)
	return n, err //nolint:wrapcheck // a test's conn
}

func (c *loggedLane) Write(b []byte) (int, error) {
	if c.log.line == nil {
		return c.Conn.Write(b) //nolint:wrapcheck // a test's conn
	}
	sent := 0
	for sent < len(b) {
		n := min(len(b)-sent, c.log.line.Burst())
		c.share(n)
		m, err := c.Conn.Write(b[sent : sent+n])
		sent += m
		if err != nil {
			return sent, err //nolint:wrapcheck // a test's conn
		}
	}
	return sent, nil
}

// share waits out n bytes on the line.
func (c *loggedLane) share(n int) {
	if c.log.line != nil && n > 0 {
		_ = c.log.line.WaitN(context.Background(), n)
	}
}

func (c *loggedLane) Close() error {
	c.closed.Do(func() { c.log.change(c.route, -1) })
	return c.Conn.Close() //nolint:wrapcheck // a test's conn
}

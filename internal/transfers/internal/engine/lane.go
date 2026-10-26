package engine

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
)

// maxFrame is the largest frame either side reads: a whole chunk and its
// envelope.
const maxFrame = ChunkSize + 64<<10

// lane is one connection carrying one Transfer's frames, each a uvarint
// length and then a Frame: a TLS lane on route 1 or 2, a kind-2 stream on
// the Node link on route 3.
type lane struct {
	c     net.Conn
	r     *bufio.Reader
	route tv1.Route
	// write bounds each write. A write that times out breaks the lane.
	write time.Duration
	// dialed: this Node opened the lane, so its first frame from the peer
	// is what proves the route works.
	dialed bool
	// done closes once the run lets the lane go.
	done    chan struct{}
	unwatch func() bool
	// open is the position of the file the lane last opened, -1 for none,
	// and gen that file's gen when it did.
	open int
	gen  int
	// asked is the chunks of the open file wanted and not yet answered.
	asked []int
	// retiring: growth let the lane go, so it asks for nothing more.
	retiring atomic.Bool
	// carried counts every byte read and written once a run holds the
	// lane.
	carried *atomic.Int64
}

func newLane(c net.Conn, route tv1.Route, write time.Duration) *lane {
	l := &lane{c: c, route: route, write: write, done: make(chan struct{}), open: -1}
	l.r = bufio.NewReader(l)
	return l
}

func (l *lane) Read(b []byte) (int, error) {
	n, err := l.c.Read(b)
	l.count(n)
	return n, err //nolint:wrapcheck // a Reader's errors pass through, io.EOF included
}

func (l *lane) count(n int) {
	if l.carried != nil {
		l.carried.Add(int64(n))
	}
}

// read returns the next frame, waiting at most wait, or for as long as it
// takes when wait is 0.
func (l *lane) read(wait time.Duration) (*tv1.Frame, error) {
	var deadline time.Time
	if wait > 0 {
		deadline = time.Now().Add(wait)
	}
	if err := l.c.SetReadDeadline(deadline); err != nil {
		return nil, fmt.Errorf("read a frame: %w", err)
	}
	n, err := binary.ReadUvarint(l.r)
	if err != nil {
		return nil, fmt.Errorf("read a frame: %w", err)
	}
	if n > maxFrame {
		return nil, fmt.Errorf("the peer sent a frame of %d bytes, over %d", n, maxFrame)
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(l.r, b); err != nil {
		return nil, fmt.Errorf("read a frame: %w", err)
	}
	var fr tv1.Frame
	if err := proto.Unmarshal(b, &fr); err != nil {
		return nil, fmt.Errorf("the peer sent a frame that doesn't decode: %w", err)
	}
	return &fr, nil
}

func (l *lane) send(fr *tv1.Frame) error { return l.sendPaced(context.Background(), fr, nil) }

// sendPaced sends fr, its bytes paced by lim when lim isn't nil.
func (l *lane) sendPaced(ctx context.Context, fr *tv1.Frame, lim *limiter) error {
	b, err := proto.Marshal(fr)
	if err != nil {
		return fmt.Errorf("encode a frame: %w", err)
	}
	head := binary.AppendUvarint(make([]byte, 0, binary.MaxVarintLen64), uint64(len(b)))
	if lim == nil {
		return l.put(append(head, b...))
	}
	if err := l.put(head); err != nil {
		return err
	}
	for len(b) > 0 {
		n := min(len(b), lim.piece)
		if err := lim.wait(ctx, n); err != nil {
			return err
		}
		if err := l.put(b[:n]); err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

func (l *lane) put(b []byte) error {
	if err := l.c.SetWriteDeadline(time.Now().Add(l.write)); err != nil {
		return fmt.Errorf("send a frame: %w", err)
	}
	n, err := l.c.Write(b)
	l.count(n)
	if err != nil {
		return fmt.Errorf("send a frame: %w", err)
	}
	return nil
}

func laneFrame(id ID, route tv1.Route) *tv1.Frame {
	return &tv1.Frame{Frame: &tv1.Frame_Lane{Lane: &tv1.Lane{TransferId: id.Bytes(), Route: route}}}
}

// holds reports whether a lane opened on lane belongs to a Transfer whose
// intent names route. Both ends hold the same lanes: the no-route wait
// holds a lane either end opened.
func holds(route, lane tv1.Route) bool {
	return route == lane || route == tv1.Route_ROUTE_BOTH_DIAL && (lane == tv1.Route_ROUTE_RECEIVER_DIALS || lane == tv1.Route_ROUTE_SENDER_DIALS)
}

// dials is the route on which side opens lanes under route, unspecified
// when the other end opens them.
func dials(side tv1.Side, route tv1.Route) tv1.Route {
	receiver := side == tv1.Side_SIDE_RECEIVE
	switch route {
	case tv1.Route_ROUTE_RECEIVER_DIALS, tv1.Route_ROUTE_NODE_LINK:
		if receiver {
			return route
		}
	case tv1.Route_ROUTE_SENDER_DIALS:
		if !receiver {
			return route
		}
	case tv1.Route_ROUTE_BOTH_DIAL:
		if receiver {
			return tv1.Route_ROUTE_RECEIVER_DIALS
		}
		return tv1.Route_ROUTE_SENDER_DIALS
	case tv1.Route_ROUTE_UNSPECIFIED:
	}
	return tv1.Route_ROUTE_UNSPECIFIED
}

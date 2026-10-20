package downloading

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading/internal/plan"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// Watch answers after the client's next live sample with what moved since
// the sample req names. The client samples only while a Watch waits.
func (c *Client) Watch(ctx context.Context, req *dlv1.WatchRequest) (*dlv1.WatchResponse, error) {
	reply := make(chan *dlv1.WatchResponse, 1)
	if !c.send(func(l *loop) { l.watch(watcher{ctx: ctx, after: req.GetAfter(), reply: reply}) }) {
		return nil, rpc.Errorf(rpc.CodeUnavailable, "the torrent client has stopped")
	}
	select {
	case res := <-reply:
		return res, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for the next live sample: %w", ctx.Err())
	}
}

// watching is Loop's live sampling. A sample spans the window from the
// first call waiting on it to its answer.
type watching struct {
	waiting []watcher
	timer   *time.Timer
	// armed is a sample due, from the timer's arming to its answer, and
	// sampling the sample's Stats posted.
	armed, sampling bool
	// from is the Stats the window started at.
	from engine.Stats
	// sample is the last answer's number, and told what it said of each
	// torrent.
	sample uint64
	told   map[bencode.InfoHash]told
}

type watcher struct {
	ctx   context.Context //nolint:containedctx // a call waiting in Loop for the next sample
	after uint64
	reply chan<- *dlv1.WatchResponse
}

type told struct {
	id   storage.UUID
	live *dlv1.Live
}

func newWatching() watching {
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	return watching{timer: timer, sample: uint64(time.Now().UnixMilli()), told: map[bencode.InfoHash]told{}}
}

func (l *loop) watch(w watcher) {
	l.watching.waiting = append(l.watching.waiting, w)
	if l.watching.armed {
		return
	}
	l.watching.armed = true
	l.postStats()
	l.watching.timer.Reset(l.c.cfg.Timings.LiveSample)
}

// takeSample has the engine send the torrents' status and then its Stats,
// whose arrival answers the waiting calls.
func (l *loop) takeSample() {
	l.watching.sampling = true
	if err := l.eng.PostUpdates(); err != nil && !errors.Is(err, engine.ErrClosed) {
		l.c.cfg.Log.Warn("read the torrents' status", "err", err.Error())
	}
	l.postStats()
}

func (l *loop) postStats() {
	if err := l.eng.PostStats(); err != nil && !errors.Is(err, engine.ErrClosed) {
		l.c.cfg.Log.Warn("read the engine's payload counters", "err", err.Error())
	}
}

func (l *loop) stats(s engine.Stats) {
	w := &l.watching
	if !w.sampling {
		w.from = s
		return
	}
	w.armed, w.sampling = false, false
	waiting := slices.DeleteFunc(w.waiting, func(c watcher) bool { return c.ctx.Err() != nil })
	w.waiting = nil
	if len(waiting) == 0 {
		return
	}
	res := l.answer(waiting[len(waiting)-1].after, w.from, s)
	for _, c := range waiting {
		c.reply <- res
	}
}

// answer is the sample after the window from..to: the Node's payload rates
// over it, and each torrent that moved since sample after, or each one
// moving now when after is not the last answer.
func (l *loop) answer(after uint64, from, to engine.Stats) *dlv1.WatchResponse {
	w := &l.watching
	res := &dlv1.WatchResponse{AtUnixMs: to.At.UnixMilli()}
	if d := to.At.Sub(from.At); !from.At.IsZero() && d > 0 {
		res.DownBps, res.UpBps = perSecond(to.RecvPayload-from.RecvPayload, d), perSecond(to.SentPayload-from.SentPayload, d)
	}
	since := after != 0 && after == w.sample
	now := map[bencode.InfoHash]told{}
	for _, h := range slices.SortedFunc(maps.Keys(l.torrents), compareHashes) {
		t := l.torrents[h]
		if t.handle == 0 || t.id.IsZero() || t.Phase == plan.Parting {
			continue
		}
		live := t.live()
		now[h] = told{id: t.id, live: live}
		was, ok := w.told[h]
		if since && ok && proto.Equal(was.live, live) || !since && live.GetDownBps() == 0 && live.GetUpBps() == 0 {
			continue
		}
		res.Rows = append(res.Rows, live)
	}
	if since {
		for _, h := range slices.SortedFunc(maps.Keys(w.told), compareHashes) {
			if was := w.told[h]; now[h].id != was.id {
				res.Gone = append(res.Gone, was.id.Bytes())
			}
		}
	}
	w.told = now
	w.sample++
	res.Sample = w.sample
	return res
}

func (t *torrent) live() *dlv1.Live {
	s := t.status
	return &dlv1.Live{
		Id: t.id.Bytes(), DownBps: s.DownBps, UpBps: s.UpBps, Done: s.Done, Wanted: s.Wanted,
		Peers: int32(s.Peers), Seeds: int32(s.Seeds), //nolint:gosec // a torrent's peer count fits
	}
}

func perSecond(bytes int64, d time.Duration) int64 { return int64(float64(bytes) / d.Seconds()) }

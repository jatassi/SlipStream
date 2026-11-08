package downloading

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/downloading/internal/external"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
)

// nodeWatching is the composite's numbering of its Watch answers.
type nodeWatching struct {
	mu sync.Mutex
	// sample is the last answer's number, builtin the built-in client's
	// number that answer carried, and told the external rows it stood on.
	sample, builtin uint64
	told            map[string]*dlv1.Live
	last            *dlv1.WatchResponse
}

func newNodeWatching() *nodeWatching {
	return &nodeWatching{sample: uint64(time.Now().UnixMilli()), told: map[string]*dlv1.Live{}}
}

// Watch answers with the built-in client's next sample and every external
// torrent whose live row moved since the composite's last answer. The
// composite numbers its own samples and keeps the built-in client's
// number beside each, so an after it doesn't know starts afresh on both
// sides. While a Watch waits, each actor polls at the live cadence.
func (n *Node) Watch(ctx context.Context, req *dlv1.WatchRequest) (*dlv1.WatchResponse, error) {
	w := n.watching
	w.mu.Lock()
	since := req.GetAfter() != 0 && req.GetAfter() == w.sample
	var after uint64
	if since {
		after = w.builtin
	}
	w.mu.Unlock()
	waiting, done := context.WithCancel(ctx)
	defer done()
	actors := n.actorList()
	for _, a := range actors {
		a.Watching(waiting)
	}
	res, err := n.builtin.Watch(ctx, &dlv1.WatchRequest{After: after})
	if err != nil {
		return nil, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.last != nil && res.GetSample() == w.builtin {
		return w.last, nil
	}
	out := proto.CloneOf(res)
	now := w.merge(out, actors, since)
	w.told, w.builtin = now, res.GetSample()
	w.sample++
	out.Sample = w.sample
	w.last = out
	return out, nil
}

// merge adds to out each actor's rates and its rows that moved since the
// last answer, or each one moving now when not since, and the rows told
// then that are gone. It answers every row as it is now.
func (w *nodeWatching) merge(out *dlv1.WatchResponse, actors []*external.Actor, since bool) map[string]*dlv1.Live {
	now := map[string]*dlv1.Live{}
	for _, a := range actors {
		rows, down, up := a.Live()
		out.DownBps += down
		out.UpBps += up
		for _, live := range rows {
			id := string(live.GetId())
			now[id] = live
			was, ok := w.told[id]
			if since && ok && proto.Equal(was, live) || !since && live.GetDownBps() == 0 && live.GetUpBps() == 0 {
				continue
			}
			out.Rows = append(out.Rows, live)
		}
	}
	if since {
		for _, id := range slices.Sorted(maps.Keys(w.told)) {
			if _, ok := now[id]; !ok {
				out.Gone = append(out.Gone, []byte(id))
			}
		}
	}
	return now
}

package downloading

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/magnet"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// Resolve answers once a peer sent the info dictionary of the magnet link
// req names, once Timings.ResolveHold passed with how the search stands,
// or once the search ended without it. A second call for the same info
// hash joins the first.
func (c *Client) Resolve(ctx context.Context, req *dlv1.ResolveRequest) (*dlv1.ResolveResponse, error) {
	if len(req.GetInfoHash()) != len(bencode.InfoHash{}) {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "an info hash is 20 bytes, not %d", len(req.GetInfoHash()))
	}
	a := ask{
		link:   magnet.Link{InfoHash: bencode.InfoHash(req.GetInfoHash()), Name: req.GetName(), Trackers: req.GetTrackers()},
		endsAt: time.UnixMilli(req.GetEndsAtUnixMs()),
		reply:  make(chan answer, 1),
	}
	if !c.send(func(l *loop) { l.resolve(a, time.Now()) }) {
		return nil, errStopped
	}
	select {
	case got := <-a.reply:
		return got.res, got.err
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for a magnet link's metadata: %w", ctx.Err())
	case <-c.stopped:
		return nil, errStopped
	}
}

// ask is one Resolve call waiting in Loop.
type ask struct {
	link   magnet.Link
	endsAt time.Time
	reply  chan answer
}

type answer struct {
	res *dlv1.ResolveResponse
	err error
}

// search is one magnet link's metadata search. It holds an engine
// torrent from its start until Removed, and the engine holds no other
// torrent of its hash meanwhile.
type search struct {
	link     magnet.Link
	phase    phase
	handle   engine.Handle
	discover bool
	since    time.Time
	// endsAt is the latest deadline a call asked for, and asked when the
	// last call came.
	endsAt, asked time.Time
	peers, most   int
	// trackerSaid is the last failure reason a tracker responded with.
	trackerSaid string
	waiting     []chan answer
	// end is what every waiting call gets once the engine torrent is
	// Removed, nil for a search nobody asked about.
	end *dlv1.ResolveResponse
	// found is when the found info dictionary started to wait for a call
	// whose answer was lost.
	found time.Time
}

type phase uint8

const (
	// looking: the engine torrent looks for peers that send the metadata.
	looking phase = iota + 1
	// ending: the engine torrent is being removed, and end is set.
	ending
	// kept: the engine torrent is gone, and end is the info dictionary,
	// kept Timings.ResolveKeep.
	kept
)

func (l *loop) taking() bool {
	return l.run != nil && l.intent.Intent != nil && l.intent.MayStart()
}

func (l *loop) resolve(a ask, now time.Time) {
	h := a.link.InfoHash
	if !l.taking() {
		a.reply <- answer{res: notTaking()}
		return
	}
	if l.torrents[h] != nil {
		a.reply <- answer{res: held()}
		return
	}
	s := l.searches[h]
	switch {
	case s == nil && !l.peerSource(a.link):
		a.reply <- answer{res: unresolved(dlv1.UnresolvedReason_UNRESOLVED_REASON_NO_PEER_SOURCE, 0, "", "")}
		return
	case s == nil:
		s = &search{link: a.link}
		if err := l.startSearch(s, now); err != nil {
			a.reply <- answer{res: unresolved(dlv1.UnresolvedReason_UNRESOLVED_REASON_ENGINE_ERROR, 0, "", err.Error())}
			return
		}
		l.searches[h] = s
	case s.phase == kept:
		a.reply <- answer{res: s.end}
		return
	}
	s.asked, s.endsAt = now, later(s.endsAt, a.endsAt)
	s.waiting = append(s.waiting, a.reply)
	c := l.c
	time.AfterFunc(c.cfg.Timings.ResolveHold, func() { c.send(func(l *loop) { l.hold(h, a.reply) }) })
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// peerSource reports whether a search for link could reach any peer: the
// link names a tracker, or the Node's DHT may run.
func (l *loop) peerSource(link magnet.Link) bool {
	return len(link.Trackers) > 0 || l.intent.Intent.GetSettings().GetDht()
}

func (l *loop) startSearch(s *search, now time.Time) error {
	discover := mayDiscover(fromLink{link: s.link})
	handle, err := l.eng.FetchMetadata(engine.MetadataParams{Link: s.link, SavePath: l.c.cfg.Dir, Discover: discover})
	if err != nil {
		l.c.cfg.Log.Warn("can't start looking for a magnet link's metadata", "info_hash", s.link.InfoHash.String(), "err", err.Error())
		return err
	}
	*s = search{
		link: s.link, phase: looking, handle: handle, discover: discover, since: now,
		endsAt: s.endsAt, asked: s.asked, waiting: s.waiting,
	}
	l.searchOf[handle] = s
	return nil
}

// hold answers a call still waiting with how its search stands, unless
// the search's deadline passed, when its end answers it.
func (l *loop) hold(h bencode.InfoHash, reply chan answer) {
	s := l.searches[h]
	if s == nil {
		return
	}
	i := slices.Index(s.waiting, reply)
	if i < 0 || s.phase == looking && !time.Now().Before(s.endsAt) {
		return
	}
	s.waiting = slices.Delete(s.waiting, i, i+1)
	reply <- answer{res: &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_Searching{Searching: &dlv1.Searching{
		SinceUnixMs: s.since.UnixMilli(), Peers: int32(s.peers), //nolint:gosec // a peer count fits
	}}}}
}

// end removes a looking search's engine torrent, and its waiting calls
// get end once it's Removed.
func (l *loop) end(s *search, end *dlv1.ResolveResponse) {
	if s.phase != looking {
		return
	}
	s.phase, s.end = ending, end
	if err := l.eng.Remove(s.handle, false); err != nil {
		l.c.cfg.Log.Warn("stop looking for a magnet link's metadata", "info_hash", s.link.InfoHash.String(), "err", err.Error())
	}
}

func (l *loop) searchEvent(s *search, ev engine.Event) {
	switch ev := ev.(type) {
	case engine.MetadataReceived:
		l.c.cfg.Log.Info("A magnet link's metadata arrived", "info_hash", s.link.InfoHash.String(), "peers", s.peers)
		l.end(s, &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_Info{Info: ev.Info}})
	case engine.TrackerFailed:
		if ev.Reason != "" {
			s.trackerSaid = ev.Reason
		}
	case engine.TorrentFailed:
		l.end(s, unresolved(dlv1.UnresolvedReason_UNRESOLVED_REASON_ENGINE_ERROR, s.most, s.trackerSaid, ev.Err.Text))
	case engine.Removed:
		l.searchRemoved(s, time.Now())
	}
}

// searchRemoved answers the waiting calls with the search's end, and
// keeps a found info dictionary. A search that ended unasked starts again
// for a call that came since.
func (l *loop) searchRemoved(s *search, now time.Time) {
	delete(l.searchOf, s.handle)
	s.handle = 0
	if s.end == nil && len(s.waiting) > 0 {
		err := l.startSearch(s, now)
		if err == nil {
			return
		}
		s.end = unresolved(dlv1.UnresolvedReason_UNRESOLVED_REASON_ENGINE_ERROR, 0, "", err.Error())
	}
	for _, w := range s.waiting {
		w <- answer{res: s.end}
	}
	s.waiting = nil
	if _, found := s.end.GetAnswer().(*dlv1.ResolveResponse_Info); found {
		s.phase, s.found = kept, now
		return
	}
	delete(l.searches, s.link.InfoHash)
}

// resyncSearches takes each ending search the engine no longer holds as
// Removed, as the engine dropped that event.
func (l *loop) resyncSearches(seen map[engine.Handle]bool, now time.Time) {
	for _, s := range l.searches {
		if s.phase == ending && !seen[s.handle] {
			l.searchRemoved(s, now)
		}
	}
}

// settleSearches ends each search whose deadline passed, that nobody
// asked about for Timings.ResolveLease, or that has lost its only peer
// source, and forgets each kept info dictionary after
// Timings.ResolveKeep.
func (l *loop) settleSearches(now time.Time) {
	t := l.c.cfg.Timings
	for h, s := range l.searches {
		switch {
		case s.phase == looking && !now.Before(s.endsAt):
			l.end(s, unresolved(dlv1.UnresolvedReason_UNRESOLVED_REASON_TIMED_OUT, s.most, s.trackerSaid, ""))
		case s.phase == looking && !l.peerSource(s.link):
			l.end(s, unresolved(dlv1.UnresolvedReason_UNRESOLVED_REASON_NO_PEER_SOURCE, s.most, s.trackerSaid, ""))
		case s.phase == looking && now.Sub(s.asked) >= t.ResolveLease:
			l.end(s, nil)
		case s.phase == kept && now.Sub(s.found) >= t.ResolveKeep:
			delete(l.searches, h)
		}
	}
}

// yield ends the search of a hash the intent lists, so the engine holds
// no metadata-only torrent of the hash when the download is added. It
// reports whether the add must wait for the search's removal.
func (l *loop) yield(h bencode.InfoHash) bool {
	s := l.searches[h]
	if s == nil || s.handle == 0 {
		return false
	}
	l.end(s, held())
	return true
}

// releaseSearches removes every search's engine torrent and answers its
// waiting calls: the Node let Downloading go.
func (l *loop) releaseSearches() {
	for _, s := range l.searches {
		if s.handle != 0 && s.phase == looking {
			if err := l.eng.Remove(s.handle, false); err != nil {
				l.c.cfg.Log.Warn("stop looking for a magnet link's metadata", "info_hash", s.link.InfoHash.String(), "err", err.Error())
			}
		}
		for _, w := range s.waiting {
			w <- answer{res: notTaking()}
		}
	}
	clear(l.searches)
	clear(l.searchOf)
}

func notTaking() *dlv1.ResolveResponse {
	return &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_NotTaking{NotTaking: &dlv1.NotTaking{}}}
}

func held() *dlv1.ResolveResponse {
	return &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_Held{Held: &dlv1.Held{}}}
}

func unresolved(reason dlv1.UnresolvedReason, most int, trackerSaid, engineErr string) *dlv1.ResolveResponse {
	return &dlv1.ResolveResponse{Answer: &dlv1.ResolveResponse_Unresolved{Unresolved: &dlv1.Unresolved{
		Reason: reason, MostPeers: int32(most), TrackerMessage: trackerSaid, Error: engineErr, //nolint:gosec // a peer count fits
	}}}
}

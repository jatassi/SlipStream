package downloading

import (
	"context"
	"path/filepath"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/downloading/internal/announce"
	"github.com/jatassi/slipstream/internal/downloading/internal/plan"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

var errStopped = rpc.Errorf(rpc.CodeUnavailable, "the torrent client has stopped")

// Inspect answers one torrent's files with their progress and its
// trackers as the announce policy sees them.
func (c *Client) Inspect(_ context.Context, req *dlv1.InspectRequest) (*dlv1.InspectResponse, error) {
	id, err := storage.UUIDFromBytes(req.GetId())
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "torrent id: %v", err)
	}
	var res *dlv1.InspectResponse
	if !c.ask(func(l *loop) { res = l.inspect(id, time.Now()) }) {
		return nil, errStopped
	}
	if res == nil {
		return nil, rpc.Errorf(rpc.CodeNotFound, "this Node holds no torrent %s", id)
	}
	return res, nil
}

// Act announces or rechecks each torrent req names, answering for each.
func (c *Client) Act(_ context.Context, req *dlv1.ActRequest) (*dlv1.ActResponse, error) {
	act := req.GetAct()
	if act != dlv1.Act_ACT_ANNOUNCE && act != dlv1.Act_ACT_RECHECK {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "no act %v", act)
	}
	res := &dlv1.ActResponse{}
	if !c.ask(func(l *loop) {
		now := time.Now()
		for _, id := range req.GetIds() {
			res.Results = append(res.Results, l.act(act, id, now))
		}
	}) {
		return nil, errStopped
	}
	return res, nil
}

// Frees answers what deleting each torrent's download copy would free. It
// reads the files off the Loop, since a folder may hold many.
func (c *Client) Frees(_ context.Context, req *dlv1.FreesRequest) (*dlv1.FreesResponse, error) {
	var copies []downloadCopy
	if !c.ask(func(l *loop) {
		for _, id := range req.GetIds() {
			copies = append(copies, l.copyOf(l.byID(id), id))
		}
	}) {
		return nil, errStopped
	}
	res := &dlv1.FreesResponse{}
	for _, d := range copies {
		res.Freed = append(res.Freed, d.frees())
	}
	return res, nil
}

// byID is the torrent the Controller knows as id, nil when the client
// holds none.
func (l *loop) byID(id []byte) *torrent {
	u, err := storage.UUIDFromBytes(id)
	if err != nil {
		return nil
	}
	for _, t := range l.torrents {
		if t.id == u {
			return t
		}
	}
	return nil
}

func (l *loop) inspect(id storage.UUID, now time.Time) *dlv1.InspectResponse {
	t := l.byID(id.Bytes())
	if t == nil {
		return nil
	}
	res := &dlv1.InspectResponse{AnnounceAtUnixMs: l.ledger.AnnounceAt(t.Hash, t.Private, now).UnixMilli()}
	if d := l.copyOf(t, t.id.Bytes()); len(d.done) > 0 {
		for i, f := range d.files {
			res.Files = append(res.Files, &dlv1.FileProgress{Index: int32(f), Done: d.done[i]}) //nolint:gosec // a torrent's file index fits
		}
	}
	lines := l.ledger.Detail(t.Hash, t.Private, now)
	for _, tr := range t.trackers {
		line := trackerLine(tr)
		d := &dlv1.TrackerDetail{Index: line.GetIndex(), Status: line.GetStatus(), Message: line.GetMessage(), AcceptsAtUnixMs: now.UnixMilli()}
		if !tr.NextAnnounce.IsZero() {
			d.NextAnnounceUnixMs = tr.NextAnnounce.UnixMilli()
		}
		if i := slices.IndexFunc(lines, func(p announce.Line) bool { return p.Index == tr.Index }); i >= 0 {
			p := lines[i]
			d.MinIntervalSeconds, d.AcceptsAtUnixMs = int64(p.MinInterval/time.Second), p.AcceptsAt.UnixMilli()
			if !p.Answered.IsZero() {
				d.AnsweredAtUnixMs = p.Answered.UnixMilli()
			}
		}
		res.Trackers = append(res.Trackers, d)
	}
	return res
}

func (l *loop) act(act dlv1.Act, id []byte, now time.Time) *dlv1.ActResult {
	r := &dlv1.ActResult{Id: id}
	t := l.byID(id)
	switch {
	case t == nil || t.Discard != plan.Kept:
		r.Refused = dlv1.Refusal_REFUSAL_NOT_HELD
	case t.failed != "":
		r.Refused = dlv1.Refusal_REFUSAL_FAILED
	case act == dlv1.Act_ACT_ANNOUNCE:
		l.announce(t, r, now)
	case t.Phase != plan.Checking:
		t.NeedsCheck = true
	}
	return r
}

// announce has every tracker of a running t announce now, unless some
// tracker would reject it, as the announce policy knows the trackers'
// rules better than libtorrent's own min announce, which this ignores.
func (l *loop) announce(t *torrent, r *dlv1.ActResult, now time.Time) {
	if t.Phase != plan.Running {
		r.Refused = dlv1.Refusal_REFUSAL_NOT_RUNNING
		return
	}
	if at := l.ledger.AnnounceAt(t.Hash, t.Private, now); at.After(now) {
		r.Refused, r.AcceptsAtUnixMs = dlv1.Refusal_REFUSAL_WINDOW, at.UnixMilli()
		return
	}
	for i := range t.trackers {
		l.call(t, "announce a torrent now", l.eng.Reannounce(t.handle, t.trackers[i].Index, 0, true))
		t.trackers[i].NextAnnounce = now
	}
}

// downloadCopy is a torrent's files as its engine holds them, with the
// bytes done of each.
type downloadCopy struct {
	id    []byte
	files []int
	paths []string
	done  []int64
}

// copyOf is t's download copy, empty while the engine doesn't hold it.
func (l *loop) copyOf(t *torrent, id []byte) downloadCopy {
	d := downloadCopy{id: id}
	if t == nil || t.handle == 0 {
		return d
	}
	files, err := l.eng.Files(t.handle)
	if err != nil {
		l.call(t, "list a torrent's files", err)
		return d
	}
	done, err := l.eng.FileProgress(t.handle)
	if err != nil {
		l.call(t, "read a torrent's file progress", err)
		return d
	}
	for i, f := range files {
		d.files = append(d.files, f.Index)
		d.paths = append(d.paths, filepath.Join(t.folder, filepath.FromSlash(f.Path)))
		d.done = append(d.done, done[i])
	}
	return d
}

// frees is what deleting d frees: the bytes done of each file with no
// other link. A file a library holds by hard link stays, and so do its
// bytes.
func (d downloadCopy) frees() *dlv1.Freed {
	f := &dlv1.Freed{Id: d.id}
	for i, path := range d.paths {
		n, err := links(path)
		switch {
		case err != nil:
		case n > 1:
			f.Linked = true
		default:
			f.Bytes += d.done[i]
		}
	}
	return f
}

// Sample samples the downloads folder's volume now and answers once the
// facts carry it.
func (c *Client) Sample(ctx context.Context, _ *dlv1.SampleRequest) (*dlv1.SampleResponse, error) {
	v, err := c.cfg.Volume(ctx, c.cfg.Dir)
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeUnavailable, "sample the downloads folder's volume: %v", err)
	}
	if !c.ask(func(l *loop) { l.sampled(v) }) {
		return nil, errStopped
	}
	return &dlv1.SampleResponse{SampledAtUnixMs: v.SampledAt.UnixMilli()}, nil
}

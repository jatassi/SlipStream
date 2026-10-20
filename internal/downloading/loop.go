package downloading

import (
	"context"
	"errors"
	"maps"
	"net/netip"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading/internal/announce"
	"github.com/jatassi/slipstream/internal/downloading/internal/db"
	"github.com/jatassi/slipstream/internal/downloading/internal/plan"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// attached is one run of the Role: Loop reports through it and fetches
// with it until it detaches.
type attached struct {
	ctx context.Context
	h   Holder
	ctl Control
}

// loop is Loop's state. Only Loop's goroutine touches it.
type loop struct {
	c        *Client
	eng      engine.Engine
	ledger   *announce.Ledger
	torrents map[bencode.InfoHash]*torrent
	byHandle map[engine.Handle]*torrent

	run   *attached
	local *storage.DB[storage.Local]
	// rows is the local rows, read when the first run attaches.
	rows   map[bencode.InfoHash]db.DownloadingTorrent
	intent role.Snapshot[*dlv1.Intent]

	// parted are the confirmations to report, in order.
	parted []*dlv1.Parted
	// saving is the torrents whose resume data a shutdown waits for.
	saving map[engine.Handle]bool
	saved  chan struct{}
	// resync is set once the engine dropped events, until the next
	// Updates carries every torrent.
	resync  bool
	version engine.Version
	listen  listener
	// sentFacts is nil until facts go to a run.
	sentFacts *dlv1.Facts
	wake      *time.Timer
	// limits is what the engine was last configured with, nil before the
	// first intent.
	limits *engine.Limits
}

// listener is what the client knows of its listen port.
type listener struct {
	// bound is where the engine listens, zero while it doesn't, and err
	// why it doesn't.
	bound netip.AddrPort
	err   string
	// last is the last address the engine listened on: incomingAt, the
	// first incoming peer that proved the port reachable, holds for it.
	last       netip.AddrPort
	incomingAt time.Time
}

// torrent is one torrent of the intent or the discards, or one the
// client holds that neither lists.
type torrent struct {
	plan.Torrent
	id     storage.UUID
	handle engine.Handle
	folder string
	// status is the engine's latest, from Updates, and statusAt when it
	// came.
	status   engine.Status
	statusAt time.Time
	trackers []engine.Tracker
	// replied is each tracker that answered since the last TrackerList,
	// with when.
	replied map[int]time.Time
	state   plan.State
	// failed is the engine's error text, "" while it holds the torrent.
	failed string
	// fault is the release fault recorded on the torrent, nil for none.
	fault *dlv1.Fault
	// freed is the bytes its removal deletes.
	freed          int64
	fetching       bool
	fetches        int
	retryAt        time.Time
	sent           *dlv1.TorrentReport
	sentFolder     bool
	sentCompletion bool
	sentTotals     *dlv1.TotalsRow
	listed         bool
}

// Loop is the client's actor: it reads the engine's events and every
// request until the engine closes or ctx ends.
func (c *Client) Loop(ctx context.Context) error {
	defer close(c.stopped)
	t := c.cfg.Timings
	l := &loop{
		c: c, eng: c.cfg.Engine, ledger: announce.New(c.cfg.Rand, t.AnnounceJitter, t.AnnounceFloor),
		torrents: map[bencode.InfoHash]*torrent{}, byHandle: map[engine.Handle]*torrent{},
		wake: time.NewTimer(time.Hour), version: c.cfg.Engine.Version(),
	}
	l.listened()
	defer l.wake.Stop()
	facts, resume, totals := time.NewTicker(t.Facts), time.NewTicker(t.ResumeSave), time.NewTicker(t.TotalsReport)
	defer facts.Stop()
	defer resume.Stop()
	defer totals.Stop()
	events := l.eng.Events()
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			l.event(ev)
			for drained := false; !drained; {
				select {
				case ev, ok := <-events:
					if !ok {
						return nil
					}
					l.event(ev)
				default:
					drained = true
				}
			}
		case f := <-c.in:
			f(l)
		case <-facts.C:
			l.tick()
		case <-resume.C:
			l.saveResume(func(t *torrent) bool { return t.status.NeedSave })
		case <-totals.C:
			l.sendTotals(time.Now())
		case <-l.wake.C:
		case <-ctx.Done():
			return nil
		}
		l.settle(time.Now())
	}
}

func (l *loop) facts() *dlv1.Facts {
	f := &dlv1.Facts{
		Libtorrent: l.version.Libtorrent, PeerIdPrefix: l.version.PeerIDPrefix, UserAgent: l.version.UserAgent,
		Listen: l.c.cfg.Listen.String(), ListenError: l.listen.err,
	}
	if l.listen.bound.IsValid() {
		f.Bound = l.listen.bound.String()
	}
	if !l.listen.incomingAt.IsZero() {
		f.IncomingAtUnixMs = l.listen.incomingAt.UnixMilli()
	}
	return f
}

// listened reads where the engine listens. A listener that moved, as one
// on an unspecified address does with the default route, proves nothing
// until a peer connects in to it.
func (l *loop) listened() {
	addr, err := l.eng.ListenAddr()
	switch {
	case err == nil:
		moved := addr != l.listen.last
		l.listen.bound, l.listen.last, l.listen.err = addr, addr, ""
		if moved {
			l.listen.incomingAt = time.Time{}
			l.configure()
		}
	case errors.Is(err, engine.ErrNotListening):
		l.listen.bound = netip.AddrPort{}
	case !errors.Is(err, engine.ErrClosed):
		l.c.cfg.Log.Warn("read where the torrent engine listens", "err", err.Error())
	}
}

// incoming notes a peer that connected in. The first that proves the port
// reachable ends the watch for incoming peers while the engine listens
// where it does.
func (l *loop) incoming(from netip.Addr) {
	if !l.listen.incomingAt.IsZero() || !l.c.cfg.ProvesPort(from) {
		return
	}
	l.listen.incomingAt = time.Now()
	l.c.cfg.Log.Info("A peer connected in to the torrent listen port, which proves it reachable", "listen", l.listen.bound.String())
	l.configure()
}

func (l *loop) attach(run *attached, local *storage.DB[storage.Local]) {
	l.run, l.local = run, local
	if l.rows == nil {
		rows, err := db.New(local).ListTorrents(run.ctx)
		if err != nil {
			l.c.cfg.Log.Error("read the torrents this Node holds", "err", err.Error())
		}
		l.rows = map[bencode.InfoHash]db.DownloadingTorrent{}
		for _, r := range rows {
			l.rows[bencode.InfoHash(r.InfoHash)] = r
		}
	}
	l.sentFacts = l.facts()
	run.h.SetFacts(l.sentFacts)
}

func (l *loop) detach(run *attached) {
	if l.run == run {
		l.run = nil
	}
}

// snapshot brings the torrents in line with s's lists: each listed one is
// held, added back from its resume data when the Node has any; each held
// one neither lists is discarded with its files kept.
func (l *loop) snapshot(s role.Snapshot[*dlv1.Intent]) {
	l.intent = s
	l.configure()
	for _, t := range l.torrents {
		t.listed = false
	}
	for _, it := range s.Intent.GetTorrents() {
		t := l.list(it.GetId(), it.GetInfoHash())
		t.Order, t.Paused, t.Discard = it.GetOrder(), it.GetWanted() == dlv1.Wanted_WANTED_PAUSED, plan.Kept
	}
	for _, d := range s.Intent.GetDiscards() {
		t := l.list(d.GetId(), d.GetInfoHash())
		t.Discard = plan.KeepFiles
		if d.GetDeleteFiles() {
			t.Discard = plan.DeleteFiles
		}
	}
	for _, t := range l.torrents {
		if !t.listed && t.Discard == plan.Kept {
			t.Discard = plan.KeepFiles
		}
	}
}

// configure applies the intent's settings to the engine when they changed,
// and watches incoming peers until one proves the listen port.
func (l *loop) configure() {
	if l.intent.Intent == nil {
		return
	}
	s := l.intent.Intent.GetSettings()
	limits := engine.Limits{
		DownloadLimit: s.GetDownloadLimit(), UploadLimit: s.GetUploadLimit(),
		Connections: int(s.GetConnections()), UTP: s.GetUtp(), WatchIncoming: l.listen.incomingAt.IsZero(),
	}
	if l.limits != nil && *l.limits == limits {
		return
	}
	if err := l.eng.Configure(limits); err != nil {
		l.c.cfg.Log.Warn("apply the client's settings", "err", err.Error())
		return
	}
	l.limits = &limits
}

func (l *loop) list(id, hash []byte) *torrent {
	h := bencode.InfoHash(hash)
	t := l.torrents[h]
	if t == nil {
		t = &torrent{Torrent: plan.Torrent{Hash: h, Phase: plan.Fetching}, replied: map[int]time.Time{}}
		l.torrents[h] = t
		l.addFromResume(t)
	}
	t.listed = true
	if u, err := storage.UUIDFromBytes(id); err == nil && u != t.id {
		t.id, t.sent, t.sentFolder, t.sentCompletion, t.sentTotals, t.fault = u, nil, false, false, nil, nil
	}
	return t
}

// addFromResume adds t back from the resume data the Node holds, its
// start spread over Timings.StartSpread.
func (l *loop) addFromResume(t *torrent) {
	row, ok := l.rows[t.Hash]
	if !ok || row.Resume == nil {
		return
	}
	handle, err := l.eng.Add(engine.AddParams{Resume: row.Resume, SavePath: row.SaveFolder})
	if err != nil {
		l.c.cfg.Log.Warn("add a torrent from its resume data; fetching it again", "info_hash", t.Hash.String(), "err", err.Error())
		return
	}
	spread := l.c.cfg.Timings.StartSpread
	t.NotBefore = time.Now().Add(time.Duration(l.c.cfg.Rand.Int64N(int64(spread) + 1)))
	t.Private, t.folder = row.Private, row.SaveFolder
	l.added(t, handle)
}

func (l *loop) added(t *torrent, handle engine.Handle) {
	t.handle, t.Phase = handle, plan.Added
	l.byHandle[handle] = t
	l.call(t, "list a torrent's trackers", l.eng.PostTrackers(handle))
}

func (l *loop) fetched(t *torrent, res *dlv1.TorrentResponse, err error) {
	t.fetching = false
	if l.torrents[t.Hash] != t || t.Phase != plan.Fetching {
		return
	}
	if err != nil {
		t.retryAt = time.Now().Add(backoff.Delay(l.c.cfg.RetryMin, l.c.cfg.RetryMax, t.fetches, l.c.cfg.Rand))
		t.fetches++
		return
	}
	t.fetches = 0
	m, err := bencode.DecodeMetainfo(res.GetMetainfo())
	if err != nil {
		t.fail(err, time.Now())
		return
	}
	t.Private, t.folder = m.Info.Private, res.GetSaveFolder()
	if t.folder == "" {
		t.folder = filepath.Join(l.c.cfg.Dir, folderName(res.GetContentType()), t.Hash.String())
	}
	err = l.local.Write(context.Background(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).PutTorrent(ctx, db.PutTorrentParams{InfoHash: t.Hash[:], SaveFolder: t.folder, Private: t.Private})
	})
	if err != nil {
		l.c.cfg.Log.Error("record a torrent the Node holds", "info_hash", t.Hash.String(), "err", err.Error())
		return
	}
	b := res.GetBaseline()
	handle, err := l.eng.Add(engine.AddParams{
		Metainfo: res.GetMetainfo(), SavePath: t.folder,
		Baseline: engine.Totals{
			Uploaded: b.GetUploaded(), Downloaded: b.GetDownloaded(),
			Active: time.Duration(b.GetActiveSeconds()) * time.Second, Seeding: time.Duration(b.GetSeedingSeconds()) * time.Second,
		},
	})
	if err != nil {
		t.fail(err, time.Now())
		return
	}
	t.NeedsCheck = true
	t.sentTotals = &dlv1.TotalsRow{
		Id: t.id.Bytes(), Uploaded: b.GetUploaded(), Downloaded: b.GetDownloaded(),
		ActiveSeconds: b.GetActiveSeconds(), SeedingSeconds: b.GetSeedingSeconds(),
	}
	l.added(t, handle)
	l.call(t, "save a torrent's resume data", l.eng.SaveResume(handle))
}

var unsafeInName = strings.NewReplacer(`\`, "_", "/", "_", ":", "_", "*", "_", "?", "_", `"`, "_", "<", "_", ">", "_", "|", "_")

// folderName is a Content Type's name as a folder name on every OS.
func folderName(name string) string { return unsafeInName.Replace(name) }

func (l *loop) settle(now time.Time) {
	if l.intent.Intent != nil {
		l.decide(now)
		l.noPayload(now)
	}
	l.report(now)
	l.publish()
}

func (l *loop) decide(now time.Time) {
	settings := l.intent.Intent.GetSettings()
	node := plan.Node{
		ActiveDownloads: int(settings.GetActiveDownloads()),
		ActiveChecks:    int(settings.GetActiveChecks()),
		StalledAfter:    time.Duration(settings.GetStalledAfterSeconds()) * time.Second,
		Live:            l.intent.Standing == role.Live,
		Draining:        l.intent.Draining,
		HoldDownloads:   l.intent.Intent.GetHoldDownloads(),
	}
	ts := make([]plan.Torrent, 0, len(l.torrents))
	for _, t := range l.torrents {
		if t.failed == "" || t.Discard != plan.Kept {
			ts = append(ts, t.Torrent)
		}
	}
	p := plan.Decide(ts, node, l.ledger, now)
	for _, t := range l.torrents {
		t.state = p.States[t.Hash]
	}
	wake := p.Wake
	for _, cmd := range p.Commands {
		if at := l.do(cmd, now); !at.IsZero() && (wake.IsZero() || at.Before(wake)) {
			wake = at
		}
	}
	if !wake.IsZero() {
		l.wake.Reset(max(wake.Sub(now), 0))
	}
}

// do runs one command, and answers when to decide again for a fetch that
// backs off.
func (l *loop) do(cmd plan.Command, now time.Time) time.Time {
	switch cmd := cmd.(type) {
	case plan.Fetch:
		return l.fetch(l.torrents[cmd.Hash], now)
	case plan.Resume:
		t := l.torrents[cmd.Hash]
		t.Phase, t.Resumed, t.DiskFull = plan.Running, now, false
		l.call(t, "resume a torrent", l.eng.Resume(t.handle))
	case plan.Pause:
		t := l.torrents[cmd.Hash]
		t.Phase = plan.Pausing
		l.call(t, "pause a torrent", l.eng.Pause(t.handle))
	case plan.Recheck:
		t := l.torrents[cmd.Hash]
		t.Phase = plan.Checking
		l.call(t, "check a torrent's files", l.eng.Recheck(t.handle))
	case plan.Remove:
		t := l.torrents[cmd.Hash]
		t.Phase = plan.Parting
		if cmd.DeleteFiles {
			t.freed = t.status.Done
		}
		l.publish()
		l.call(t, "remove a torrent", l.eng.Remove(t.handle, cmd.DeleteFiles))
	case plan.Parted:
		l.part(l.torrents[cmd.Hash])
	}
	return time.Time{}
}

func (l *loop) fetch(t *torrent, now time.Time) time.Time {
	if t.fetching || l.run == nil {
		return time.Time{}
	}
	if now.Before(t.retryAt) {
		return t.retryAt
	}
	t.fetching = true
	run, id := l.run, t.id
	go func() {
		res, err := run.ctl.Torrent(run.ctx, &dlv1.TorrentRequest{Id: id.Bytes()})
		l.c.send(func(l *loop) { l.fetched(t, res, err) })
	}()
	return time.Time{}
}

func (l *loop) part(t *torrent) {
	if t.listed {
		l.parted = append(l.parted, &dlv1.Parted{Id: t.id.Bytes(), Freed: t.freed})
	}
	delete(l.torrents, t.Hash)
	delete(l.byHandle, t.handle)
	delete(l.rows, t.Hash)
	l.ledger.Forget(t.Hash)
	err := l.local.Write(context.Background(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).DeleteTorrent(ctx, t.Hash[:])
	})
	if err != nil {
		l.c.cfg.Log.Error("forget a torrent the Node let go", "info_hash", t.Hash.String(), "err", err.Error())
	}
}

func (l *loop) call(t *torrent, what string, err error) {
	if err != nil {
		l.c.cfg.Log.Warn(what, "info_hash", t.Hash.String(), "err", err.Error())
	}
}

func (l *loop) release() error {
	held := slices.Collect(maps.Values(l.torrents))
	clear(l.torrents)
	clear(l.byHandle)
	clear(l.rows)
	l.parted, l.intent = nil, role.Snapshot[*dlv1.Intent]{}
	l.publish()
	for _, t := range held {
		if t.handle != 0 && t.Phase != plan.Parting {
			l.call(t, "remove a torrent", l.eng.Remove(t.handle, false))
		}
		l.ledger.Forget(t.Hash)
	}
	if l.local == nil {
		return nil
	}
	return l.local.Write(context.Background(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).DeleteTorrents(ctx)
	})
}

func (l *loop) shutdown(local *storage.DB[storage.Local], saved chan struct{}) {
	l.local = local
	l.saving, l.saved = map[engine.Handle]bool{}, saved
	l.saveResume(func(t *torrent) bool { return t.status.NeedSave || t.Phase == plan.Running })
	l.doneSaving()
}

func (l *loop) saveResume(due func(*torrent) bool) {
	for _, t := range l.torrents {
		if t.handle == 0 || t.Phase == plan.Parting || !due(t) {
			continue
		}
		if err := l.eng.SaveResume(t.handle); err != nil {
			l.call(t, "save a torrent's resume data", err)
			continue
		}
		if l.saving != nil {
			l.saving[t.handle] = true
		}
	}
}

func (l *loop) doneSaving() {
	if l.saving != nil && len(l.saving) == 0 {
		close(l.saved)
		l.saving = nil
	}
}

func (l *loop) tick() {
	if err := l.eng.PostUpdates(); err != nil && !errors.Is(err, engine.ErrClosed) {
		l.c.cfg.Log.Warn("read the torrents' status", "err", err.Error())
	}
	l.listened()
	if f := l.facts(); l.run != nil && !proto.Equal(f, l.sentFacts) {
		l.run.h.SetFacts(f)
		l.sentFacts = f
	}
}

// publish stores the finished torrents and the save folders for Path,
// Seeding and Folders. Loop publishes before it asks the engine to remove
// a torrent, as the engine forgets the handle Path would use at once.
func (l *loop) publish() {
	p := &published{finished: map[bencode.InfoHash]finished{}, folders: []string{l.c.cfg.Dir}}
	for _, t := range l.torrents {
		if t.handle != 0 && t.Complete && t.Phase != plan.Parting {
			p.finished[t.Hash] = finished{handle: t.handle, folder: t.folder}
		}
		if t.folder != "" && !inside(l.c.cfg.Dir, t.folder) && !slices.Contains(p.folders, t.folder) {
			p.folders = append(p.folders, t.folder)
		}
	}
	slices.Sort(p.folders[1:])
	l.c.held.Store(p)
}

func inside(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (l *loop) event(ev engine.Event) {
	switch ev := ev.(type) {
	case engine.ListenFailed:
		l.c.cfg.Log.Warn("The torrent engine is not listening", "error", ev.Err)
		l.listen.err = ev.Err.Text
	case engine.IncomingPeer:
		l.incoming(ev.Addr)
	case engine.Dropped:
		l.resync = true
		if err := l.eng.PostUpdates(); err != nil {
			l.c.cfg.Log.Warn("read the torrents' status", "err", err.Error())
		}
	case engine.Updates:
		l.updates(ev.Statuses)
	default:
		if t := l.byHandle[handleOf(ev)]; t != nil {
			l.torrentEvent(t, ev)
		}
	}
}

func (l *loop) torrentEvent(t *torrent, ev engine.Event) {
	switch ev := ev.(type) {
	case engine.StateChanged:
		t.status.State = ev.State
		t.Complete = t.Complete || complete(ev.State)
	case engine.Finished:
		t.Complete = true
	case engine.Checked:
		t.NeedsCheck = false
	case engine.Paused:
		switch {
		case t.Phase == plan.Checking && !t.NeedsCheck, t.Phase == plan.Pausing, t.Phase == plan.Running:
			t.Phase = plan.Added
		}
	case engine.Removed:
		l.part(t)
	case engine.TorrentFailed:
		if !errors.Is(ev.Err, engine.ErrDiskFull) {
			t.fail(ev.Err, time.Now())
		}
	case engine.FileError:
		if errors.Is(ev.Err, engine.ErrDiskFull) {
			t.DiskFull = true
		} else {
			t.fail(ev.Err, time.Now())
		}
	case engine.ResumeSaved:
		l.resumeSaved(t, ev.Data)
	case engine.ResumeFailed:
		l.call(t, "save a torrent's resume data", ev.Err)
		l.resumeDone(t)
	case engine.TrackerAnnounced:
		l.ledger.Sent(t.Hash, l.tracker(t, ev.Tracker), announce.Event(ev.Event), ev.At)
	case engine.TrackerReplied:
		l.ledger.Answered(t.Hash, l.tracker(t, ev.Tracker), ev.At)
		t.replied[ev.Tracker] = ev.At
		l.call(t, "list a torrent's trackers", l.eng.PostTrackers(t.handle))
	case engine.TrackerFailed:
		if l.ledger.Failed(t.Hash, l.tracker(t, ev.Tracker), ev.Reason, l.unregisteredGap(), ev.At) {
			t.recordFault(dlv1.FaultKind_FAULT_KIND_UNREGISTERED, ev.At)
		}
		l.call(t, "list a torrent's trackers", l.eng.PostTrackers(t.handle))
	case engine.TrackerList:
		l.trackerList(t, ev.Trackers)
	}
}

// trackerList records t's trackers, then the intervals of each that
// answered and schedules its next announce, in that order, since a
// reannounce moves libtorrent's min announce.
func (l *loop) trackerList(t *torrent, trackers []engine.Tracker) {
	t.trackers = trackers
	now := time.Now()
	for _, tr := range trackers {
		at, ok := t.replied[tr.Index]
		if !ok {
			continue
		}
		delete(t.replied, tr.Index)
		delay := l.ledger.Replied(t.Hash, l.tracker(t, tr.Index), announce.Reply{At: at, NextAnnounce: tr.NextAnnounce, MinAnnounce: tr.MinAnnounce})
		l.call(t, "schedule a tracker's next announce", l.eng.Reannounce(t.handle, tr.Index, max(at.Add(delay).Sub(now), 0), false))
	}
}

func (l *loop) tracker(t *torrent, index int) announce.Tracker {
	tr := announce.Tracker{Index: index}
	if i := slices.IndexFunc(t.trackers, func(tr engine.Tracker) bool { return tr.Index == index }); i >= 0 {
		tr.Host = hostOf(t.trackers[i].URL)
	}
	return tr
}

func (l *loop) unregisteredGap() time.Duration {
	return time.Duration(l.intent.Intent.GetThresholds().GetUnregisteredGapSeconds()) * time.Second
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

func (l *loop) resumeSaved(t *torrent, data []byte) {
	defer l.resumeDone(t)
	err := l.local.Write(context.Background(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).SetResume(ctx, db.SetResumeParams{Resume: data, InfoHash: t.Hash[:]})
	})
	if err != nil {
		l.c.cfg.Log.Error("save a torrent's resume data", "info_hash", t.Hash.String(), "err", err.Error())
	}
}

func (l *loop) resumeDone(t *torrent) {
	if l.saving != nil {
		delete(l.saving, t.handle)
		l.doneSaving()
	}
}

// updates folds a status batch into the cache. After the engine dropped
// events the batch carries every torrent, so the phases the lost events
// would have moved are read from it.
func (l *loop) updates(statuses []engine.Status) {
	now := time.Now()
	seen := map[engine.Handle]bool{}
	for _, s := range statuses {
		t := l.byHandle[s.Torrent]
		if t == nil {
			continue
		}
		seen[s.Torrent] = true
		t.status, t.statusAt = s, now
		t.Complete = t.Complete || complete(s.State)
		if s.LastPayload.After(t.LastPayload) {
			t.LastPayload = s.LastPayload
		}
		if l.resync && s.Paused && (t.Phase == plan.Running || t.Phase == plan.Pausing || t.Phase == plan.Checking && !checking(s.State)) {
			t.Phase, t.NeedsCheck = plan.Added, t.NeedsCheck && t.Phase != plan.Checking
		}
	}
	if l.resync {
		for _, t := range l.torrents {
			if t.Phase == plan.Parting && !seen[t.handle] {
				l.part(t)
			}
		}
		l.resync = false
	}
}

func complete(s engine.State) bool { return s == engine.StateSeeding || s == engine.StateFinished }

func checking(s engine.State) bool {
	return s == engine.StateCheckingFiles || s == engine.StateCheckingResumeData
}

func handleOf(ev engine.Event) engine.Handle {
	switch ev := ev.(type) {
	case engine.Added:
		return ev.Torrent
	case engine.StateChanged:
		return ev.Torrent
	case engine.Finished:
		return ev.Torrent
	case engine.Removed:
		return ev.Torrent
	case engine.TorrentFailed:
		return ev.Torrent
	case engine.TrackerAnnounced:
		return ev.Torrent
	case engine.TrackerReplied:
		return ev.Torrent
	case engine.TrackerWarned:
		return ev.Torrent
	case engine.TrackerFailed:
		return ev.Torrent
	case engine.TrackerList:
		return ev.Torrent
	case engine.Paused:
		return ev.Torrent
	case engine.Resumed:
		return ev.Torrent
	case engine.ResumeSaved:
		return ev.Torrent
	case engine.ResumeFailed:
		return ev.Torrent
	case engine.FilesDeleted:
		return ev.Torrent
	case engine.Checked:
		return ev.Torrent
	case engine.FileError:
		return ev.Torrent
	}
	return 0
}
